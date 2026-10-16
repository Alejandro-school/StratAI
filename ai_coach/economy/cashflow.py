"""Reconcile replicated money and native entry changes without inferring cause."""

from __future__ import annotations

import argparse
import json
from collections import Counter, defaultdict
from pathlib import Path

from ai_coach.economy import (
    candidate_io,
    entry_identity,
    native_identity,
    ownership,
    transaction_trace,
)

DATA = "economy_cashflow_observations.json"
REPORT = "economy_cashflow_reconciliation.json"
PROPERTIES = {
    "account": "m_pInGameMoneyServices.m_iAccount",
    "spent": "m_pInGameMoneyServices.m_iCashSpentThisRound",
}
STATUSES = (
    "not_comparable",
    "no_change",
    "deliveries_without_net_change",
    "counter_decreased",
    "amounts_match_single_entry",
    "amounts_match_multiple_entries",
    "amounts_match_without_unique_object",
    "residuals_present",
)
ABSENT = {"status": "absent", "native_type": None, "value": None}
require = native_identity.require
schema_validator = ownership.schema_validator


def amount(field: dict) -> int | None:
    return field["value"] if entry_identity.member_valid("m_nCost", field) else None


def entry_changes(
    before: dict | None, after: dict | None, before_links: dict, after_links: dict
) -> dict:
    result = {
        "status": "unavailable",
        "added": [],
        "removed": [],
        "retained": None,
        "reordered_only": False,
        "duplicate_multiplicity": False,
        "added_cost": None,
        "removed_cost": None,
        "cost_net": None,
    }
    if (
        before is None
        or after is None
        or any(
            p["status"] != "observed"
            or any(
                not entry_identity.member_valid(k, f)
                for e in p["entries"]
                for k, f in e["fields"].items()
            )
            for p in (before, after)
        )
    ):
        return result

    def group(p):
        grouped = defaultdict(list)
        for e in p["entries"]:
            grouped[
                json.dumps(e["fields"], sort_keys=True, separators=(",", ":"))
            ].append(e)
        return grouped

    left, right = group(before), group(after)
    result.update(
        status="observed",
        retained=0,
        duplicate_multiplicity=any(
            len(v) > 1 for g in (left, right) for v in g.values()
        ),
    )
    for key in sorted(left.keys() | right.keys()):
        retained = min(len(left[key]), len(right[key]))
        result["retained"] += retained
        for target, values, links in (
            ("removed", left[key], before_links),
            ("added", right[key], after_links),
        ):
            for e in values[retained:]:
                linked = links[e["index"]]
                result[target].append(
                    {
                        "entry_index": e["index"],
                        "fields_sha256": candidate_io.sha(key.encode()),
                        "recorded_cost": e["fields"]["m_nCost"]["value"],
                        "entity_alias": linked["entity_alias"],
                        "link_status": linked["status"],
                        "multiplicity": len(values),
                    }
                )
    result["added_cost"] = sum(e["recorded_cost"] for e in result["added"])
    result["removed_cost"] = sum(e["recorded_cost"] for e in result["removed"])
    result["cost_net"] = result["added_cost"] - result["removed_cost"]
    result["reordered_only"] = (
        not result["added"]
        and not result["removed"]
        and [e["fields"] for e in before["entries"]]
        != [e["fields"] for e in after["entries"]]
    )
    return result


def channel_flow(
    before: dict,
    after: dict,
    rows: list,
    metadata: dict,
    controller: dict,
    pawn_handle: int,
    actor: str,
    side: str,
    instance: int,
) -> dict:
    start, end = amount(before), amount(after)
    result = {
        "status": "observed",
        "from_value": start,
        "to_value": end,
        "snapshot_delta": end - start
        if start is not None and end is not None
        else None,
        "deliveries": [],
        "positive_total": None,
        "negative_total": None,
        "delivery_net": None,
        "snapshot_residual": None,
    }
    reasons = set()
    if start is None or end is None:
        reasons.add("snapshot_unavailable")
    current = start
    for row in rows:
        m = metadata[row["sequence"]]
        value = (
            row["value"]["value"]
            if row["value"]["status"] == "observed"
            and type(row["value"]["value"]) is int
            and 0 <= row["value"]["value"] <= 2**31 - 1
            else None
        )
        phase = (
            "creation_frame_or_replay"
            if row["frame"] == controller["created_frame"]
            else "later_frame_delivery"
        )
        binding = (
            entry_identity.member_valid("m_hItem", m["pawn_handle"])
            and m["pawn_handle"]["value"] == pawn_handle
            and (row["actor_alias"], row["side"], row["instance"])
            == (actor, side, instance)
        )
        baseline = (
            row["channel"] == "property_baseline" or phase == "creation_frame_or_replay"
        )
        delta = (
            value - current
            if current is not None and value is not None and binding and not baseline
            else None
        )
        if not binding:
            reasons.add("delivery_binding_unavailable")
        if baseline:
            reasons.add("baseline_or_creation_delivery")
        if value is None:
            reasons.add("delivery_value_unavailable")
        result["deliveries"].append(
            {
                "trace_sequence": row["sequence"],
                "frame": row["frame"],
                "channel": row["channel"],
                "phase": phase,
                "binding_matches": binding,
                "before_value": current,
                "after_value": value,
                "delta": delta,
            }
        )
        # A baseline may anchor later deliveries, but never repairs the interval.
        current = value if binding else None
    if current is not None and end is not None:
        result["snapshot_residual"] = end - current
    if not reasons and result["snapshot_residual"] != 0:
        reasons.add("untracked_snapshot_change")
    if not reasons:
        deltas = [d["delta"] for d in result["deliveries"]]
        require(all(d is not None for d in deltas), "cashflow lost a delivery delta")
        result.update(
            positive_total=sum(d for d in deltas if d > 0),
            negative_total=sum(-d for d in deltas if d < 0),
            delivery_net=sum(deltas),
        )
    result["status"] = next(
        (
            r
            for r in (
                "snapshot_unavailable",
                "delivery_binding_unavailable",
                "baseline_or_creation_delivery",
                "delivery_value_unavailable",
                "untracked_snapshot_change",
            )
            if r in reasons
        ),
        "observed",
    )
    result["reasons"] = sorted(reasons)
    return result


def validate_sources(data: dict, entries: dict, own: dict, identity: dict, trace: dict):
    require(
        not list(schema_validator(DATA).iter_errors(data)),
        "invalid cashflow observation schema",
    )
    require(
        all(
            data[k] == entries[k] == trace[k]
            for k in ("source_alias", "source_sha256", "build_identifier")
        ),
        "cashflow source identity differs",
    )
    obs = data["observations"]
    controllers = {c["alias"]: c for c in obs["controllers"]}
    require(
        len(controllers) == len(obs["controllers"]),
        "cashflow controller alias repeated",
    )
    require(
        [c["created_frame"] for c in obs["controllers"]]
        == sorted(c["created_frame"] for c in obs["controllers"]),
        "cashflow controller creation order differs",
    )
    property_rows = [
        r
        for r in trace["observations"]["rows"]
        if r["channel"] in {"property_baseline", "property_update"}
    ]
    require(
        [d["trace_sequence"] for d in obs["deliveries"]]
        == [r["sequence"] for r in property_rows],
        "cashflow delivery coverage differs",
    )
    metadata = {d["trace_sequence"]: d for d in obs["deliveries"]}
    for row in property_rows:
        c = controllers.get(row["entity_alias"])
        require(
            c is not None
            and c["created_frame"] <= row["frame"]
            and (c["destroyed_frame"] is None or row["frame"] <= c["destroyed_frame"]),
            "cashflow delivery uses unavailable controller",
        )
        require(
            row["source"] in PROPERTIES.values(), "cashflow unknown monetary property"
        )
        if row["channel"] == "property_baseline":
            require(
                row["frame"] == c["created_frame"], "cashflow baseline after creation"
            )
        entry_identity.validate_scalar(metadata[row["sequence"]]["pawn_handle"])
    for c in controllers.values():
        require(
            c["destroyed_frame"] is None or c["destroyed_frame"] >= c["created_frame"],
            "cashflow controller destruction precedes creation",
        )
        evidence = [
            r
            for r in trace["observations"]["rows"]
            if r["entity_alias"] == c["alias"]
            and r["channel"] in {"probe", "property_baseline"}
        ]
        require(
            bool(evidence) and min(r["frame"] for r in evidence) == c["created_frame"],
            "cashflow controller creation lacks trace source",
        )
    own_cuts = own["observations"]["cuts"]
    require(
        [c["ownership_cut"] for c in obs["cuts"]] == [c["sequence"] for c in own_cuts],
        "cashflow common cut coverage differs",
    )
    require(
        [s["sequence"] for s in obs["states"]]
        == list(range(1, len(obs["states"]) + 1)),
        "cashflow state sequence differs",
    )
    created = {
        r["entity_alias"]: r
        for r in identity["observations"]["rows"]
        if r["kind"] == "entity_created"
    }
    trace_rows = trace["observations"]["rows"]
    position = active = 0
    for marker, cut, entry_cut in zip(
        obs["cuts"], own_cuts, entries["observations"]["cuts"]
    ):
        while (
            position < len(trace_rows) and trace_rows[position]["frame"] <= cut["frame"]
        ):
            position += 1
        require(
            marker["trace_next_frame"] == cut["frame"] + 1
            and marker["trace_sequence"]
            == (trace_rows[position - 1]["sequence"] if position else 0),
            "cashflow trace frame or full prefix differs",
        )
        seq = marker["state_sequence"]
        require(
            seq in {active, active + 1} and 1 <= seq <= len(obs["states"]),
            "cashflow state future or retrospectively reused",
        )
        state = obs["states"][seq - 1]
        require(
            state["entry_state_sequence"] == entry_cut["state_sequence"],
            "cashflow and entry delivery states differ",
        )
        # Check live controller coverage even when the scalar payload is shared.
        live = {}
        for c in obs["controllers"]:
            if c["created_frame"] <= cut["frame"]:
                live[c["index"]] = c
        live = {
            ix: c
            for ix, c in live.items()
            if c["destroyed_frame"] is None or c["destroyed_frame"] > cut["frame"]
        }
        for p in state["pawns"]:
            if p["controller_alias"] is not None:
                c = controllers.get(p["controller_alias"])
                require(
                    c is not None and live.get(c["index"]) == c,
                    "cashflow controller lifetime differs at cut",
                )
        if seq == active:
            continue
        require(state["first_cut"] == cut["sequence"], "cashflow state backfilled")
        pawns = own["observations"]["states"][cut["state_sequence"] - 1]["pawns"]
        require(
            [p["pawn_alias"] for p in state["pawns"]]
            == [p["entity_alias"] for p in pawns],
            "cashflow pawn coverage differs",
        )
        for p in state["pawns"]:
            for key in ("pawn_handle", "account", "spent"):
                entry_identity.validate_scalar(p[key])
            if p["controller_alias"] is None:
                require(
                    all(p[k] == ABSENT for k in ("pawn_handle", "account", "spent")),
                    "missing controller acquired money",
                )
            else:
                entity = created[p["pawn_alias"]]
                expected = (entity["value"]["entity_serial"] << 14) | entity["value"][
                    "entity_index"
                ]
                require(
                    entry_identity.member_valid("m_hItem", p["pawn_handle"])
                    and p["pawn_handle"]["value"] == expected,
                    "cashflow pawn serial differs",
                )
        active = seq
    require(active == len(obs["states"]), "cashflow states unobserved")
    return controllers, metadata, property_rows


def reconcile(
    data: dict,
    entries: dict,
    own: dict,
    identity: dict,
    candidate: dict,
    trace: dict,
    *,
    require_observed: bool = False,
) -> dict:
    links = entry_identity.reconcile(
        entries, own, identity, candidate, require_observed=require_observed
    )
    controllers, metadata, property_rows = validate_sources(
        data, entries, own, identity, trace
    )
    clocks = {c["sequence"]: c for c in identity["observations"]["clocks"]}
    cuts = own["observations"]["cuts"]
    obs = data["observations"]
    rounds = {f["instance"]: f for f in candidate["initial_frontiers"]}
    intervals = []
    status_counts = Counter()
    reason_counts = Counter()
    opps = Counter()
    examined = omitted = 0
    position = 0
    for ci in range(1, len(cuts)):
        before_cut, after_cut = cuts[ci - 1 : ci + 1]
        bm, am = obs["cuts"][ci - 1 : ci + 1]
        before_state = obs["states"][bm["state_sequence"] - 1]
        after_state = obs["states"][am["state_sequence"] - 1]
        states = (before_state, after_state)
        cash = [{p["pawn_alias"]: p for p in s["pawns"]} for s in states]
        entry_states = [
            entries["observations"]["states"][s["entry_state_sequence"] - 1]
            for s in states
        ]
        entry_pawns = [{p["entity_alias"]: p for p in s["pawns"]} for s in entry_states]
        linked = [defaultdict(dict), defaultdict(dict)]
        for i, s in enumerate(states):
            for r in links["states"][s["entry_state_sequence"] - 1]["records"]:
                linked[i][r["pawn_alias"]][r["entry_index"]] = r
        own_pawns = [
            {
                p["entity_alias"]: p
                for p in own["observations"]["states"][c["state_sequence"] - 1]["pawns"]
            }
            for c in (before_cut, after_cut)
        ]
        batch = defaultdict(list)
        while (
            position < len(property_rows)
            and property_rows[position]["sequence"] <= am["trace_sequence"]
        ):
            row = property_rows[position]
            position += 1
            if row["sequence"] > bm["trace_sequence"]:
                batch[row["entity_alias"]].append(row)
        temporal = (
            before_cut["instance"] == after_cut["instance"]
            and before_cut["frame"] + 1 == after_cut["frame"]
            and before_cut["demo_tick"] <= after_cut["demo_tick"]
        )
        clock_pair = [clocks.get(c["clock_sequence"]) for c in (before_cut, after_cut)]
        temporal = (
            temporal
            and all(
                k is not None
                and k["network_tick"] is not None
                and (k["frame"], k["demo_tick"]) == (c["frame"], c["demo_tick"])
                for k, c in zip(clock_pair, (before_cut, after_cut))
            )
            and clock_pair[0]["epoch"] == clock_pair[1]["epoch"]
        )
        for pawn_alias in sorted(cash[0].keys() | cash[1].keys()):
            examined += 1
            a, b = (g.get(pawn_alias) for g in cash)
            actor_a = own_pawns[0].get(pawn_alias, {}).get("actor_alias")
            actor_b = own_pawns[1].get(pawn_alias, {}).get("actor_alias")
            side_a = own_pawns[0].get(pawn_alias, {}).get("side")
            side_b = own_pawns[1].get(pawn_alias, {}).get("side")
            aliases = {
                p["controller_alias"]
                for p in (a, b)
                if p and p["controller_alias"] is not None
            }
            deliveries = sorted(
                (r for alias in aliases for r in batch[alias]),
                key=lambda r: r["sequence"],
            )
            if (
                temporal
                and a == b
                and entry_pawns[0].get(pawn_alias) == entry_pawns[1].get(pawn_alias)
                and own_pawns[0].get(pawn_alias) == own_pawns[1].get(pawn_alias)
                and not deliveries
            ):
                omitted += 1
                continue
            reasons = []
            if not temporal:
                reasons.append("incompatible_delivery_cuts")
            if a is None or b is None:
                reasons.append("pawn_lifetime_changed")
            if (
                actor_a is None
                or actor_a != actor_b
                or side_a is None
                or side_a != side_b
            ):
                reasons.append("actor_binding_unavailable")
            if any(
                sum(p["actor_alias"] == actor_a for p in g.values()) != 1
                for g in own_pawns
            ) or any(p is None or p["controller_alias"] is None for p in (a, b)):
                reasons.append("controller_or_actor_nonunique_or_absent")
            if len(aliases) != 1:
                reasons.append("controller_lifetime_changed")
            if len(aliases) == 1 and any(
                sum(p["controller_alias"] == next(iter(aliases)) for p in g.values())
                != 1
                for g in cash
            ):
                reasons.append("controller_claim_nonunique")
            changes = (
                entry_changes(
                    entry_pawns[0].get(pawn_alias),
                    entry_pawns[1].get(pawn_alias),
                    linked[0][pawn_alias],
                    linked[1][pawn_alias],
                )
                if not reasons
                else entry_changes(None, None, {}, {})
            )
            if changes["status"] != "observed":
                reasons.append("entry_change_unavailable")
            channels = {"account": None, "spent": None}
            if not any(r != "entry_change_unavailable" for r in reasons):
                controller = controllers[a["controller_alias"]]
                for key, source in PROPERTIES.items():
                    channels[key] = channel_flow(
                        a[key],
                        b[key],
                        [r for r in deliveries if r["source"] == source],
                        metadata,
                        controller,
                        a["pawn_handle"]["value"],
                        actor_a,
                        side_a,
                        after_cut["instance"],
                    )
                    if channels[key]["status"] != "observed":
                        reasons.append(key + ":" + channels[key]["status"])
            status = "not_comparable"
            account_residual = spent_residual = None
            if not reasons:
                account_residual = (
                    channels["account"]["snapshot_delta"] + changes["cost_net"]
                )
                spent_residual = (
                    channels["spent"]["snapshot_delta"] - changes["cost_net"]
                )
                changed = changes["added"] + changes["removed"]
                if channels["spent"]["negative_total"] > 0:
                    status = "counter_decreased"
                elif account_residual != 0 or spent_residual != 0:
                    status = "residuals_present"
                elif not changed:
                    status = (
                        "deliveries_without_net_change" if deliveries else "no_change"
                    )
                elif changes["duplicate_multiplicity"] or any(
                    e["link_status"] != "exact_link" for e in changed
                ):
                    status = "amounts_match_without_unique_object"
                else:
                    status = (
                        "amounts_match_single_entry"
                        if len(changed) == 1
                        else "amounts_match_multiple_entries"
                    )
            frontier = rounds[after_cut["instance"]]
            opportunity = None
            if (
                actor_a == actor_b
                and side_a == side_b
                and frontier["round_link"]["status"] == "verified"
                and (actor_b, side_b)
                in {
                    (p["actor_alias"], p["side"])
                    for p in frontier["round_start_probe"]["actors"]
                }
            ):
                opportunity = f"{data['source_alias']}:economy:{after_cut['instance']:05d}:{side_b}"
                opps[opportunity] += 1
            intervals.append(
                {
                    "from_cut": before_cut["sequence"],
                    "to_cut": after_cut["sequence"],
                    "from_state": before_state["sequence"],
                    "to_state": after_state["sequence"],
                    "instance": after_cut["instance"],
                    "pawn_alias": pawn_alias,
                    "from_actor": actor_a,
                    "to_actor": actor_b,
                    "side": side_b,
                    "controller_alias": next(iter(aliases))
                    if len(aliases) == 1
                    else None,
                    "opportunity_id": opportunity,
                    "status": status,
                    "reasons": sorted(set(reasons)),
                    "trace_sequences": [r["sequence"] for r in deliveries],
                    "entries": changes,
                    "channels": channels,
                    "account_delta_plus_entry_cost_net": account_residual,
                    "spent_delta_minus_entry_cost_net": spent_residual,
                    "cause": "not_accredited",
                    "refund_amount": None,
                    "decision_eligible": False,
                }
            )
            status_counts[status] += 1
            reason_counts.update(set(reasons))
    if require_observed:
        require(
            bool(intervals) and bool(property_rows),
            "cashflow smoke lacks affected records",
        )
    return {
        "intervals": intervals,
        "opportunities": [
            {
                "opportunity_id": f"{data['source_alias']}:economy:{instance:05d}:{side}",
                "instance": instance,
                "side": side,
                "retained_intervals": opps[
                    f"{data['source_alias']}:economy:{instance:05d}:{side}"
                ],
                "comparison_allowed": False,
            }
            for instance in sorted(rounds)
            for side in ("CT", "T")
        ],
        "summary": {
            "cuts": len(cuts),
            "states": len(obs["states"]),
            "controllers": len(controllers),
            "monetary_deliveries": len(property_rows),
            "examined_pawn_pairs": examined,
            "omitted_unchanged_snapshot_pairs": omitted,
            "retained_intervals": len(intervals),
            "status_counts": {s: status_counts[s] for s in STATUSES},
            "reason_counts": dict(sorted(reason_counts.items())),
            "verified_itemized_transactions": 0,
        },
    }


def load_source(path: Path):
    path = candidate_io.source_io._safe_file(path.parent.absolute(), path.name)
    require(
        path.name == DATA and path.stat().st_size <= 256 * 1024 * 1024,
        "invalid cashflow source path or size",
    )
    raw = path.read_bytes()
    data = json.loads(raw)
    require(
        not list(schema_validator(DATA).iter_errors(data)),
        "invalid cashflow source schema",
    )
    entries, own, identity, candidate, ref = entry_identity.load_source(
        path.parent / entry_identity.DATA
    )
    require(ref == data["source_entries"], "cashflow entry hash differs")
    trace_path = candidate_io.source_io._safe_file(path.parent, transaction_trace.DATA)
    trace_raw = trace_path.read_bytes()
    require(
        data["source_trace"]
        == {
            "path": transaction_trace.DATA,
            "bytes": len(trace_raw),
            "sha256": candidate_io.sha(trace_raw),
        }
        and candidate_io.sha(trace_raw) == identity["source_trace"]["sha256"],
        "cashflow trace provenance differs",
    )
    require(
        path.read_bytes() == raw
        and candidate_io.sha((path.parent / entry_identity.DATA).read_bytes())
        == ref["sha256"],
        "cashflow source changed during load",
    )
    return (
        data,
        entries,
        own,
        identity,
        candidate,
        json.loads(trace_raw),
        {"path": DATA, "bytes": len(raw), "sha256": candidate_io.sha(raw)},
    )


def build_report(
    data: dict,
    entries: dict,
    own: dict,
    identity: dict,
    candidate: dict,
    trace: dict,
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
        schema_id="stratai.economy_cashflow_reconciliation@1",
        source_observations=ref,
        **reconcile(
            data,
            entries,
            own,
            identity,
            candidate,
            trace,
            require_observed=require_observed,
        ),
    )
    require(
        not list(schema_validator(REPORT).iter_errors(report)),
        "invalid cashflow report schema",
    )
    return report


def validate_report(report: dict, *args) -> None:
    require(report == build_report(*args), "cashflow report differs from source")


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
    a = parser.parse_args()
    print(
        json.dumps(materialize(a.source, a.output, require_observed=a.require_observed))
    )


if __name__ == "__main__":
    main()
