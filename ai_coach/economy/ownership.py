"""Reconcile freshly observed ownership at delivery cuts, without inferring cause."""

from __future__ import annotations

import argparse
import json
from collections import Counter, defaultdict
from pathlib import Path

from jsonschema import Draft202012Validator

from ai_coach.economy import candidate_io, native_identity, transaction_trace

DATA = "economy_ownership_observations.json"
REPORT = "economy_ownership_reconciliation.json"
STATUSES = (
    "consistent",
    "pair_with_incomplete_inventory",
    "actor_unavailable",
    "actor_binding_nonunique",
    "owner_inventory_conflict",
    "duplicate_inventory_claim",
    "owner_only",
    "inventory_only",
    "no_resolved_claim",
    "unassigned_in_observed_scope",
)
require = native_identity.require


def schema_validator(name: str = DATA):
    schema = json.loads(
        (candidate_io.CONTRACTS / name.replace(".json", ".schema.json")).read_text(
            encoding="utf-8"
        )
    )
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


def check_reference(ref: dict, kind: str, live: dict) -> None:
    if ref["handle"] is None:
        require(
            ref["status"] in {"absent", "unsupported_type"}
            and ref["target_alias"] is None,
            "missing ownership reference acquired identity",
        )
    else:
        require(
            (ref["status"], ref["target_alias"])
            == native_identity.reference_status(ref["handle"], kind, live),
            "ownership reference serial or lifetime differs",
        )


def validate_state(state: dict, live: dict, known_actors: set) -> None:
    for key, kind in (("pawns", "pawn"), ("equipment", "equipment")):
        aliases = [row["entity_alias"] for row in state[key]]
        expected = sorted(
            row["entity_alias"]
            for row in live.values()
            if row["value"]["entity_kind"] == kind
        )
        require(aliases == expected, "ownership live entity coverage differs")
    for equipment in state["equipment"]:
        check_reference(equipment["owner"], "pawn", live)
    for pawn in state["pawns"]:
        require(
            pawn["actor_alias"] is None or pawn["actor_alias"] in known_actors,
            "ownership actor unknown",
        )
        status, length, slots = (
            pawn["inventory_status"],
            pawn["inventory_length"],
            pawn["slots"],
        )
        if status == "observed":
            require(
                length is not None and 0 <= length <= 64 and len(slots) == length,
                "active inventory length differs",
            )
        elif status == "outside_probe":
            require(
                length is not None and length > 64 and not slots,
                "out-of-probe inventory became known",
            )
        else:
            require(length is None and not slots, "missing inventory became known")
        for ref in slots:
            check_reference(ref, "equipment", live)


def reconcile_state(state: dict, cut: dict, frontier: dict, source_alias: str) -> dict:
    pawns = {p["entity_alias"]: p for p in state["pawns"]}
    claims = defaultdict(list)
    actors = Counter(
        p["actor_alias"] for p in state["pawns"] if p["actor_alias"] is not None
    )
    complete = bool(pawns) and all(
        p["inventory_status"] == "observed"
        and all(ref["status"] in {"resolved", "invalid_sentinel"} for ref in p["slots"])
        for p in pawns.values()
    )
    for pawn in pawns.values():
        for ref in pawn["slots"]:
            if ref["status"] == "resolved":
                claims[ref["target_alias"]].append(pawn["entity_alias"])
    roster = {
        (p["actor_alias"], p["side"]) for p in frontier["round_start_probe"]["actors"]
    }
    records = []
    for equipment in state["equipment"]:
        alias, owner = equipment["entity_alias"], equipment["owner"]
        owner_pawn = owner["target_alias"] if owner["status"] == "resolved" else None
        holders = sorted(claims[alias])
        actor = side = opportunity = None
        status = "no_resolved_claim"
        if len(holders) > 1:
            status = "duplicate_inventory_claim"
        elif owner_pawn is not None and holders:
            if holders[0] != owner_pawn:
                status = "owner_inventory_conflict"
            else:
                pawn = pawns[owner_pawn]
                if not complete:
                    status = "pair_with_incomplete_inventory"
                elif pawn["actor_alias"] is None or pawn["side"] is None:
                    status = "actor_unavailable"
                elif actors[pawn["actor_alias"]] != 1:
                    status = "actor_binding_nonunique"
                else:
                    status = "consistent"
                    actor, side = pawn["actor_alias"], pawn["side"]
                    if (
                        frontier["round_link"]["status"] == "verified"
                        and (actor, side) in roster
                    ):
                        opportunity = (
                            f"{source_alias}:economy:{cut['instance']:05d}:{side}"
                        )
        elif owner_pawn is not None:
            status = "owner_only"
        elif holders:
            status = "inventory_only"
        elif owner["status"] == "invalid_sentinel" and complete:
            status = "unassigned_in_observed_scope"
        records.append(
            {
                "entity_alias": alias,
                "status": status,
                "owner_pawn": owner_pawn,
                "claim_pawns": holders,
                "actor_alias": actor,
                "side": side,
                "opportunity_id": opportunity,
            }
        )
    return {
        "state_sequence": state["sequence"],
        "first_cut": cut["sequence"],
        "instance": cut["instance"],
        "frame": cut["frame"],
        "demo_tick": cut["demo_tick"],
        "inventory_complete": complete,
        "records": records,
    }


def reconcile(
    data: dict, identity: dict, candidate: dict, *, require_observed: bool = False
) -> dict:
    require(
        not list(schema_validator().iter_errors(data)),
        "invalid ownership observation schema",
    )
    require(
        all(
            data[k] == identity[k] == candidate[k]
            for k in ("source_alias", "source_sha256", "build_identifier")
        ),
        "ownership source identity differs",
    )
    cuts, states = (data["observations"][key] for key in ("cuts", "states"))
    require(
        [c["sequence"] for c in cuts] == list(range(1, len(cuts) + 1)),
        "ownership cut sequence differs",
    )
    require(
        [s["sequence"] for s in states] == list(range(1, len(states) + 1)),
        "ownership state sequence differs",
    )
    entries = sorted(
        identity["observations"]["clocks"] + identity["observations"]["rows"],
        key=lambda r: r["sequence"],
    )
    rounds = {r["instance"]: r for r in candidate["rounds"]}
    frontiers = {f["instance"]: f for f in candidate["initial_frontiers"]}
    known_actors = {
        a["actor_alias"]
        for f in frontiers.values()
        for a in f["round_start_probe"]["actors"]
    }
    known_actors.update(
        r["actor_alias"]
        for r in identity["observations"]["rows"]
        if r["actor_alias"] is not None
    )
    position = 0
    live, entities, derived, transitions = {}, {}, [], []
    latest_clock = previous_cut = previous_state = previous_clock = None
    last_frame = -1
    active_state = 0
    unavailable_clock = temporal_gaps = 0
    status_counts, opportunity_counts = Counter(), Counter()
    for cut in cuts:
        require(cut["frame"] > last_frame, "ownership frame repeated or reordered")
        last_frame = cut["frame"]
        round_ = rounds.get(cut["instance"])
        require(
            round_ is not None
            and round_["freeze_end_tick"] is not None
            and round_["start_tick"] < cut["demo_tick"] < round_["freeze_end_tick"],
            "ownership cut outside strict freeze",
        )
        while position < len(entries) and entries[position]["frame"] <= cut["frame"]:
            row = entries[position]
            position += 1
            if "network_tick" in row:
                latest_clock = row
            elif row["kind"] == "entity_created":
                entities[row["entity_alias"]] = row
                live[row["value"]["entity_index"]] = row
            elif row["kind"] == "entity_destroyed":
                index = entities[row["entity_alias"]]["value"]["entity_index"]
                if live.get(index, {}).get("entity_alias") == row["entity_alias"]:
                    del live[index]
        require(
            cut["identity_sequence"]
            == (entries[position - 1]["sequence"] if position else 0),
            "ownership cut is not the full observed frame prefix",
        )
        require(
            cut["clock_sequence"]
            == (latest_clock["sequence"] if latest_clock else None),
            "ownership clock uses a future or stale sample",
        )
        clock = (
            latest_clock
            if latest_clock
            and latest_clock["network_tick"] is not None
            and (latest_clock["frame"], latest_clock["demo_tick"])
            == (cut["frame"], cut["demo_tick"])
            else None
        )
        unavailable_clock += clock is None
        seq = cut["state_sequence"]
        require(
            seq in {active_state, active_state + 1} and 1 <= seq <= len(states),
            "ownership state referenced from future or reused retrospectively",
        )
        state = states[seq - 1]
        if seq != active_state:
            require(state["first_cut"] == cut["sequence"], "ownership state backfilled")
            validate_state(state, live, known_actors)
            result = reconcile_state(
                state, cut, frontiers[cut["instance"]], data["source_alias"]
            )
            derived.append(result)
            active_state = seq
            status_counts.update(r["status"] for r in result["records"])
            opportunity_counts.update(
                r["opportunity_id"]
                for r in result["records"]
                if r["opportunity_id"] is not None
            )
        else:
            require(
                previous_cut["instance"] == cut["instance"],
                "ownership state crosses instances",
            )
            for key, kind in (("pawns", "pawn"), ("equipment", "equipment")):
                require(
                    [r["entity_alias"] for r in state[key]]
                    == sorted(
                        r["entity_alias"]
                        for r in live.values()
                        if r["value"]["entity_kind"] == kind
                    ),
                    "reused ownership state conceals a lifetime change",
                )
        current = {r["entity_alias"]: r for r in derived[-1]["records"]}
        if previous_cut is not None:
            adjacent = (
                previous_cut["instance"] == cut["instance"]
                and previous_cut["frame"] + 1 == cut["frame"]
                and previous_cut["demo_tick"] <= cut["demo_tick"]
                and previous_clock is not None
                and clock is not None
                and previous_clock["epoch"] == clock["epoch"]
            )
            temporal_gaps += not adjacent
            if adjacent and previous_cut["state_sequence"] != seq:
                for alias in sorted(current.keys() & previous_state.keys()):
                    before, after = previous_state[alias], current[alias]
                    if (
                        before["status"]
                        in {"consistent", "unassigned_in_observed_scope"}
                        and after["status"]
                        in {"consistent", "unassigned_in_observed_scope"}
                        and before["actor_alias"] != after["actor_alias"]
                    ):
                        classification = "holder_changed_between_adjacent_delivery_cuts"
                        if before["actor_alias"] is None:
                            classification = (
                                "holding_appeared_between_adjacent_delivery_cuts"
                            )
                        elif after["actor_alias"] is None:
                            classification = (
                                "holding_disappeared_between_adjacent_delivery_cuts"
                            )
                        transitions.append(
                            {
                                "entity_alias": alias,
                                "instance": cut["instance"],
                                "from_cut": previous_cut["sequence"],
                                "to_cut": cut["sequence"],
                                "from_actor": before["actor_alias"],
                                "to_actor": after["actor_alias"],
                                "from_side": before["side"],
                                "to_side": after["side"],
                                "from_status": before["status"],
                                "to_status": after["status"],
                                "classification": classification,
                                "cause": "not_accredited",
                            }
                        )
        previous_cut, previous_state, previous_clock = cut, current, clock
    require(active_state == len(states), "ownership states are unobserved")
    if require_observed:
        require(
            bool(cuts) and bool(states) and sum(status_counts.values()) > 0,
            "ownership smoke lacks affected records",
        )
    opportunities = [
        {
            "opportunity_id": f"{data['source_alias']}:economy:{instance:05d}:{side}",
            "instance": instance,
            "side": side,
            "consistent_state_records": opportunity_counts[
                f"{data['source_alias']}:economy:{instance:05d}:{side}"
            ],
            "comparison_allowed": False,
        }
        for instance in sorted(rounds)
        for side in ("CT", "T")
    ]
    return {
        "states": derived,
        "transitions": transitions,
        "opportunities": opportunities,
        "summary": {
            "cuts": len(cuts),
            "states": len(states),
            "equipment_state_records": sum(status_counts.values()),
            "status_counts": {s: status_counts[s] for s in STATUSES},
            "consistent_opportunities": sum(
                o["consistent_state_records"] > 0 for o in opportunities
            ),
            "observed_holding_transitions": len(transitions),
            "cut_clock_unavailable": unavailable_clock,
            "transition_gaps_or_epoch_boundaries": temporal_gaps,
            "verified_itemized_transactions": 0,
        },
    }


def load_source(path: Path):
    path = candidate_io.source_io._safe_file(path.parent.absolute(), path.name)
    require(
        path.name == DATA and path.stat().st_size <= 256 * 1024 * 1024,
        "invalid ownership source path or size",
    )
    raw = path.read_bytes()
    data = json.loads(raw)
    require(
        not list(schema_validator().iter_errors(data)),
        "invalid ownership observation schema",
    )
    source = candidate_io.source_io._safe_file(path.parent, native_identity.DATA)
    source_raw = source.read_bytes()
    ref = data["source_identity"]
    require(
        len(source_raw) == ref["bytes"]
        and candidate_io.sha(source_raw) == ref["sha256"],
        "ownership identity hash differs",
    )
    identity, _ = native_identity.load_candidate(source)
    candidate, manifest_hash, _ = candidate_io.load_source(
        path.parent / "economy_candidate_manifest.json"
    )
    trace_file = candidate_io.source_io._safe_file(path.parent, transaction_trace.DATA)
    trace_raw = trace_file.read_bytes()
    require(
        candidate_io.sha(trace_raw) == identity["source_trace"]["sha256"]
        and json.loads(trace_raw)["source_candidate"]["sha256"] == manifest_hash,
        "ownership candidate provenance chain changed",
    )
    require(
        path.read_bytes() == raw and source.read_bytes() == source_raw,
        "ownership source changed during load",
    )
    return (
        data,
        identity,
        candidate,
        {"path": DATA, "bytes": len(raw), "sha256": candidate_io.sha(raw)},
    )


def build_report(
    data: dict,
    identity: dict,
    candidate: dict,
    ref: dict,
    *,
    require_observed: bool = False,
) -> dict:
    report = {
        k: data[k]
        for k in (
            "contract_version",
            "source_alias",
            "source_sha256",
            "build_identifier",
            "partition",
            "comparison_allowed",
            "training_allowed",
            "product_supported",
        )
    }
    report.update(
        schema_id="stratai.economy_ownership_reconciliation@1",
        source_observations=ref,
        **reconcile(data, identity, candidate, require_observed=require_observed),
    )
    require(
        not list(schema_validator(REPORT).iter_errors(report)),
        "invalid ownership reconciliation schema",
    )
    return report


def validate_report(
    report: dict, data: dict, identity: dict, candidate: dict, ref: dict
) -> None:
    require(
        report == build_report(data, identity, candidate, ref),
        "ownership reconciliation differs from source",
    )


def materialize(source: Path, output: Path, *, require_observed: bool = False) -> dict:
    output = candidate_io.validate_output_directory(output)
    data, identity, candidate, ref = load_source(source)
    report = build_report(
        data, identity, candidate, ref, require_observed=require_observed
    )
    candidate_io._write_new(output / REPORT, report)
    return report["summary"]


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--require-observed", action="store_true")
    args = parser.parse_args()
    print(
        json.dumps(
            materialize(
                args.source, args.output, require_observed=args.require_observed
            )
        )
    )


if __name__ == "__main__":
    main()
