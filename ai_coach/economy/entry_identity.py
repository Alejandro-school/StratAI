"""Link native entries to live objects at the same delivery cut; cause stays unknown."""

from __future__ import annotations

import argparse
import json
from collections import Counter
from pathlib import Path

from ai_coach.economy import candidate_io, native_identity, ownership

DATA = "economy_entry_observations.json"
REPORT = "economy_entry_identity_links.json"
MEMBERS = ("m_unDefIdx", "m_nCost", "m_nPrevArmor", "m_bPrevHelmet", "m_hItem")
STATUSES = (
    "exact_link",
    "reference_unresolved",
    "identity_ambiguous",
    "special_entry_without_entity",
    "ownership_disagreement",
    "ownership_unaccredited",
    "partial_list",
    "invalid_member_value",
    "actor_unavailable",
    "definition_unavailable",
    "definition_disagreement",
)
require = native_identity.require
schema_validator = ownership.schema_validator


def validate_scalar(field: dict) -> None:
    status, typ, value = (field[k] for k in ("status", "native_type", "value"))
    if status == "absent":
        require(typ is None and value is None, "absent entry scalar became known")
    elif status == "unsupported_type":
        require(
            typ is not None and value is None, "unsupported entry scalar became known"
        )
    elif typ == "bool":
        require(type(value) is bool, "entry scalar native boolean differs")
    else:
        require(
            typ in {"int", "int32", "int64", "uint32", "uint64"} and type(value) is int,
            "entry scalar native integer differs",
        )
        minimum = 0 if typ.startswith("u") else -(2**31)
        maximum = 2**31 - 1 if typ == "int32" else 2**32 - 1
        require(
            minimum <= value <= maximum, "entry scalar outside captured native range"
        )


def member_valid(member: str, field: dict) -> bool:
    if field["status"] != "observed":
        return False
    v = field["value"]
    if member == "m_bPrevHelmet":
        return type(v) is bool or type(v) is int and v in (0, 1)
    maximum = {"m_unDefIdx": 65535, "m_nCost": 2**31 - 1}.get(member, 2**32 - 1)
    minimum = -(2**31) if member == "m_nPrevArmor" else 0
    return type(v) is int and minimum <= v <= maximum


def validate_entry(entry: dict, live: dict) -> None:
    for field in [*entry["fields"].values(), entry["entity_definition"]]:
        validate_scalar(field)
    field, ref = entry["fields"]["m_hItem"], entry["item"]
    if member_valid("m_hItem", field):
        expected = native_identity.reference_status(field["value"], "equipment", live)
        require(
            ref["handle"] == field["value"]
            and (ref["status"], ref["target_alias"]) == expected,
            "entry reference serial or lifetime differs",
        )
    else:
        expected_status = (
            "absent" if field["status"] == "absent" else "unsupported_type"
        )
        require(
            ref == {"handle": None, "status": expected_status, "target_alias": None},
            "invalid entry handle acquired identity",
        )
    if ref["status"] != "resolved":
        require(
            entry["entity_definition"]
            == {"status": "absent", "native_type": None, "value": None},
            "unresolved entry acquired entity definition",
        )


def link_state(
    state: dict, own_state: dict, holding: dict, cut: dict, live: dict
) -> dict:
    pawns = {p["entity_alias"]: p for p in own_state["pawns"]}
    require(
        [p["entity_alias"] for p in state["pawns"]] == list(pawns),
        "entry pawn coverage differs from common cut",
    )
    held = {r["entity_alias"]: r for r in holding["records"]}
    actor_counts = Counter(
        p["actor_alias"] for p in pawns.values() if p["actor_alias"] is not None
    )
    claims = Counter(
        e["item"]["target_alias"]
        for p in state["pawns"]
        for e in p["entries"]
        if e["item"]["status"] == "resolved"
    )
    complete = bool(pawns) and all(p["status"] == "observed" for p in state["pawns"])
    handles_complete = all(
        member_valid("m_hItem", e["fields"]["m_hItem"])
        for p in state["pawns"]
        for e in p["entries"]
    )
    records, lists = [], []
    for p in state["pawns"]:
        status, length, entries = (p[k] for k in ("status", "length", "entries"))
        if status in {"observed", "partial"}:
            require(
                length is not None
                and 0 <= length <= 256
                and [e["index"] for e in entries] == list(range(length)),
                "entry indices or length differ",
            )
            partial = any(
                f["status"] != "observed" for e in entries for f in e["fields"].values()
            )
            require((status == "partial") == partial, "entry list presence differs")
        elif status == "outside_probe":
            require(
                length is not None and length > 256 and not entries,
                "entry list exceeds probe",
            )
        else:
            require(length is None and not entries, "missing entry list became empty")
        pawn = pawns[p["entity_alias"]]
        actor, side = pawn["actor_alias"], pawn["side"]
        duplicates = Counter(json.dumps(e["fields"], sort_keys=True) for e in entries)
        lists.append(
            {
                "pawn_alias": p["entity_alias"],
                "actor_alias": actor,
                "side": side,
                "status": status,
                "length": length,
            }
        )
        for entry in entries:
            validate_entry(entry, live)
            fields, ref = entry["fields"], entry["item"]
            target = ref["target_alias"]
            holder = held.get(target)
            duplicate_count = duplicates[json.dumps(fields, sort_keys=True)]
            valid = all(member_valid(k, fields[k]) for k in MEMBERS)
            definition = entry["entity_definition"]
            result = "exact_link"
            if not complete:
                result = "partial_list"
            elif not valid:
                result = "invalid_member_value"
            elif not handles_complete:
                result = "partial_list"
            elif duplicate_count > 1 or target is not None and claims[target] > 1:
                result = "identity_ambiguous"
            elif ref["status"] == "invalid_sentinel":
                result = "special_entry_without_entity"
            elif ref["status"] != "resolved":
                result = "reference_unresolved"
            elif actor is None or side is None:
                result = "actor_unavailable"
            elif actor_counts[actor] != 1:
                result = "identity_ambiguous"
            elif not member_valid("m_unDefIdx", definition):
                result = "definition_unavailable"
            elif definition["value"] != fields["m_unDefIdx"]["value"]:
                result = "definition_disagreement"
            elif (
                holder["owner_pawn"] is not None
                and holder["owner_pawn"] != p["entity_alias"]
            ) or any(alias != p["entity_alias"] for alias in holder["claim_pawns"]):
                result = "ownership_disagreement"
            elif (
                holder["status"] != "consistent"
                or holder["actor_alias"] != actor
                or holder["side"] != side
            ):
                result = "ownership_unaccredited"
            records.append(
                {
                    "pawn_alias": p["entity_alias"],
                    "entry_index": entry["index"],
                    "actor_alias": actor,
                    "side": side,
                    "entity_alias": target,
                    "item_status": ref["status"],
                    "status": result,
                    "duplicate_entry_count": duplicate_count,
                    "resolved_entry_claims": claims[target]
                    if target is not None
                    else 0,
                    "ownership_status": holder["status"] if holder else None,
                    "recorded_cost": fields["m_nCost"]["value"]
                    if member_valid("m_nCost", fields["m_nCost"])
                    else None,
                    "opportunity_id": holder["opportunity_id"]
                    if result == "exact_link"
                    else None,
                    "cause": "not_accredited",
                    "refund_amount": None,
                    "decision_eligible": False,
                }
            )
    return {
        "state_sequence": state["sequence"],
        "first_cut": state["first_cut"],
        "ownership_state_sequence": own_state["sequence"],
        "instance": cut["instance"],
        "frame": cut["frame"],
        "demo_tick": cut["demo_tick"],
        "lists": lists,
        "records": records,
    }


def reconcile(
    data: dict,
    own: dict,
    identity: dict,
    candidate: dict,
    *,
    require_observed: bool = False,
) -> dict:
    require(
        not list(schema_validator(DATA).iter_errors(data)),
        "invalid entry observation schema",
    )
    require(
        all(
            data[k] == own[k]
            for k in ("source_alias", "source_sha256", "build_identifier")
        ),
        "entry source identity differs",
    )
    holding = ownership.reconcile(
        own, identity, candidate, require_observed=require_observed
    )
    cuts, states = (data["observations"][k] for k in ("cuts", "states"))
    own_cuts, own_states = (own["observations"][k] for k in ("cuts", "states"))
    require(
        [c["ownership_cut"] for c in cuts] == [c["sequence"] for c in own_cuts],
        "entry delivery cuts incomplete or reordered",
    )
    require(
        [s["sequence"] for s in states] == list(range(1, len(states) + 1)),
        "entry state sequence differs",
    )
    rows = identity["observations"]["rows"]
    live, entities, derived = {}, {}, []
    position = active = 0
    for marker, cut in zip(cuts, own_cuts):
        while (
            position < len(rows)
            and rows[position]["sequence"] <= cut["identity_sequence"]
        ):
            row = rows[position]
            position += 1
            if row["kind"] == "entity_created":
                entities[row["entity_alias"]] = row
                live[row["value"]["entity_index"]] = row
            elif row["kind"] == "entity_destroyed":
                index = entities[row["entity_alias"]]["value"]["entity_index"]
                if live.get(index, {}).get("entity_alias") == row["entity_alias"]:
                    del live[index]
        seq = marker["state_sequence"]
        require(
            seq in {active, active + 1} and 1 <= seq <= len(states),
            "entry state referenced from future or reused retrospectively",
        )
        state = states[seq - 1]
        require(
            state["ownership_state_sequence"] == cut["state_sequence"],
            "entry and ownership delivery states differ",
        )
        if seq != active:
            require(state["first_cut"] == cut["sequence"], "entry state backfilled")
            derived.append(
                link_state(
                    state,
                    own_states[cut["state_sequence"] - 1],
                    holding["states"][cut["state_sequence"] - 1],
                    cut,
                    live,
                )
            )
            active = seq
    require(active == len(states), "entry states unobserved")
    counts = Counter(r["status"] for s in derived for r in s["records"])
    if require_observed:
        require(
            bool(cuts) and sum(counts.values()) > 0,
            "entry smoke lacks affected records",
        )
    opportunities = []
    for o in holding["opportunities"]:
        n = sum(
            r["opportunity_id"] == o["opportunity_id"]
            for s in derived
            for r in s["records"]
        )
        opportunities.append(
            {
                "opportunity_id": o["opportunity_id"],
                "instance": o["instance"],
                "side": o["side"],
                "exact_entry_state_records": n,
                "comparison_allowed": False,
            }
        )
    return {
        "states": derived,
        "opportunities": opportunities,
        "summary": {
            "cuts": len(cuts),
            "states": len(states),
            "entry_state_records": sum(counts.values()),
            "status_counts": {s: counts[s] for s in STATUSES},
            "exact_unique_objects": len(
                {
                    r["entity_alias"]
                    for s in derived
                    for r in s["records"]
                    if r["status"] == "exact_link"
                }
            ),
            "exact_opportunities": sum(
                o["exact_entry_state_records"] > 0 for o in opportunities
            ),
            "list_status_counts": dict(
                sorted(
                    Counter(p["status"] for s in derived for p in s["lists"]).items()
                )
            ),
            "verified_itemized_transactions": 0,
        },
    }


def load_source(path: Path):
    path = candidate_io.source_io._safe_file(path.parent.absolute(), path.name)
    require(
        path.name == DATA and path.stat().st_size <= 256 * 1024 * 1024,
        "invalid entry source path or size",
    )
    raw = path.read_bytes()
    data = json.loads(raw)
    require(
        not list(schema_validator(DATA).iter_errors(data)),
        "invalid entry observation schema",
    )
    source = candidate_io.source_io._safe_file(path.parent, ownership.DATA)
    own, identity, candidate, own_ref = ownership.load_source(source)
    require(data["source_ownership"] == own_ref, "entry ownership hash differs")
    require(
        path.read_bytes() == raw
        and candidate_io.sha(source.read_bytes()) == own_ref["sha256"],
        "entry source changed during load",
    )
    return (
        data,
        own,
        identity,
        candidate,
        {"path": DATA, "bytes": len(raw), "sha256": candidate_io.sha(raw)},
    )


def build_report(
    data: dict,
    own: dict,
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
        schema_id="stratai.economy_entry_identity_links@1",
        source_observations=ref,
        **reconcile(data, own, identity, candidate, require_observed=require_observed),
    )
    require(
        not list(schema_validator(REPORT).iter_errors(report)),
        "invalid entry link report schema",
    )
    return report


def validate_report(
    report: dict, data: dict, own: dict, identity: dict, candidate: dict, ref: dict
) -> None:
    require(
        report == build_report(data, own, identity, candidate, ref),
        "entry links differ from source",
    )


def materialize(source: Path, output: Path, *, require_observed: bool = False) -> dict:
    output = candidate_io.validate_output_directory(output)
    report = build_report(*load_source(source), require_observed=require_observed)
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
