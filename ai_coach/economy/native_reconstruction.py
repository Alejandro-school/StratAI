"""Partial item-level reconstruction from native counters and sellback entries."""

from __future__ import annotations

import argparse
from collections import Counter
from pathlib import Path

from ai_coach.economy import native_sources as io
from ai_coach.economy.candidate_io import _write_new, validate_output_directory

FIELDS = ("m_unDefIdx", "m_nCost", "m_nPrevArmor", "m_bPrevHelmet", "m_hItem")
CHARGE_FIELDS = (
    "item_definition_index",
    "recorded_cost",
    "previous_armor",
    "previous_helmet",
    "native_item_handle",
)
BLOCKERS = [
    "versioned_price_asset_binding_missing",
    "reward_schedule_and_prior_bonus_unverified",
    "purchasable_loadout_and_item_limits_unverified",
    "refund_rules_unverified",
    "teammate_acceptance_unobserved",
]


def rules(data: dict) -> dict:
    identity = {}
    for name in ("build_num", "patch_version", "network_protocol"):
        values = sorted(
            {
                r["fields"][name]["value"]
                for r in data["observations"]["version_observations"]
                if name in r["fields"] and r["fields"][name]["status"] == "observed"
            }
        )
        identity[name] = {
            "status": "observed"
            if len(values) == 1
            else "conflicting"
            if values
            else "unavailable",
            "value": values[0] if len(values) == 1 else None,
            "observed_values": values,
        }
    return {
        "game_identity": identity,
        "status": "compatibility_unverified",
        "price_scope": "observed_sellback_entry_cost_not_general_shop_price",
        "blockers": list(BLOCKERS),
        "comparison_allowed": False,
    }


def interval_reason(previous: dict | None, current: dict, family: str) -> str:
    if previous is None:
        return "previous_observation_unavailable"
    if not previous["connected"] or not current["connected"]:
        return "participant_disconnected"
    if current["side"] not in {"CT", "T"} or previous["side"] != current["side"]:
        return "participant_side_changed_or_unknown"
    if previous["tick"] >= current["tick"]:
        return "same_tick_order_ambiguous"
    if (
        previous[family]["status"] != "observed"
        or current[family]["status"] != "observed"
    ):
        return "native_array_or_members_unavailable"
    if family == "purchases_round":
        for row in (previous, current):
            ids = [e["m_nItemDefIndex"]["value"] for e in row[family]["entries"]]
            if 0 in ids or len(ids) != len(set(ids)):
                return "invalid_or_duplicate_item_definition"
    return "native_endpoint_difference_only"


def scope(previous: dict | None, current: dict, source: dict) -> str:
    round_ = next(r for r in source["rounds"] if r["instance"] == current["instance"])
    frontier = next(
        r for r in source["initial_frontiers"] if r["instance"] == current["instance"]
    )
    if (
        previous is not None
        and frontier["round_link"]["status"] == "verified"
        and round_["freeze_end_tick"] is not None
        and round_["start_tick"]
        <= previous["tick"]
        < current["tick"]
        < round_["freeze_end_tick"]
    ):
        return "competitive_execution"
    return "boundary_or_unlinked"


def counter_values(row: dict) -> dict:
    return {
        e["m_nItemDefIndex"]["value"]: e["m_nCount"]["value"]
        for e in row["purchases_round"]["entries"]
    }


def charges(row: dict) -> list[tuple]:
    return [tuple(e[k]["value"] for k in FIELDS) for e in row["sellback"]["entries"]]


def reconstruct(data: dict, source: dict, bindings: dict) -> dict:
    rows, previous = [], {}
    for current in data["observations"]["snapshots"]:
        key = (current["instance"], current["actor_alias"])
        old = previous.get(key)
        counter_reason = interval_reason(old, current, "purchases_round")
        sellback_reason = interval_reason(old, current, "sellback")
        counter_ok = counter_reason == "native_endpoint_difference_only"
        sellback_ok = sellback_reason == "native_endpoint_difference_only"
        deltas, added, removed = [], [], []
        if counter_ok:
            before, after = counter_values(old), counter_values(current)
            deltas = [
                {
                    "item_definition_index": i,
                    "before": before.get(i, 0),
                    "after": after.get(i, 0),
                    "delta": after.get(i, 0) - before.get(i, 0),
                }
                for i in sorted(before.keys() | after.keys())
                if before.get(i, 0) != after.get(i, 0)
            ]
        if sellback_ok:
            before, after = Counter(charges(old)), Counter(charges(current))
            added = [
                dict(zip(CHARGE_FIELDS, item, strict=True))
                for item in sorted((after - before).elements())
            ]
            removed = [
                dict(zip(CHARGE_FIELDS, item, strict=True))
                for item in sorted((before - after).elements())
            ]
        rows.append(
            {
                "observation_id": current["observation_id"],
                "previous_observation_id": old["observation_id"] if old else None,
                "instance": current["instance"],
                "actor_alias": current["actor_alias"],
                "side": current["side"],
                "interval_start_tick": old["tick"] if old else None,
                "interval_end_tick": current["tick"],
                "scope": scope(old, current, source),
                "purchase_counter_status": "available" if counter_ok else "unavailable",
                "purchase_counter_reason": counter_reason,
                "purchase_counter_differences": deltas,
                "sellback_status": "available" if sellback_ok else "unavailable",
                "sellback_reason": sellback_reason,
                "sellback_entries_added": added,
                "sellback_entries_removed": removed,
                "transaction_timing": "interval_only_no_subtick_order",
                "refund_amount": None,
                "basket": None,
                "decision_eligible": False,
            }
        )
        previous[key] = current
    result = {
        "schema_id": "stratai.economy_native_reconstruction@1",
        **{
            k: data[k]
            for k in (
                "contract_version",
                "source_alias",
                "source_sha256",
                "build_identifier",
                "comparison_allowed",
                "training_allowed",
                "product_supported",
            )
        },
        **bindings,
        "rule_assessment": rules(data),
        "records": rows,
    }
    result["summary"] = summary(result)
    return result


def summary(report: dict) -> dict:
    rows = report["records"]
    return {
        "records": len(rows),
        "counter_intervals_available": sum(
            r["purchase_counter_status"] == "available" for r in rows
        ),
        "positive_counter_differences": sum(
            d["delta"] > 0 for r in rows for d in r["purchase_counter_differences"]
        ),
        "negative_counter_differences": sum(
            d["delta"] < 0 for r in rows for d in r["purchase_counter_differences"]
        ),
        "sellback_intervals_available": sum(
            r["sellback_status"] == "available" for r in rows
        ),
        "sellback_entries_added": sum(len(r["sellback_entries_added"]) for r in rows),
        "sellback_entries_removed": sum(
            len(r["sellback_entries_removed"]) for r in rows
        ),
        "competitive_execution_records": sum(
            r["scope"] == "competitive_execution" for r in rows
        ),
        "classified_actions": 0,
        "verified_rule_packages": 0,
    }


def validate_report(report: dict, data: dict, source: dict, bindings: dict) -> dict:
    """Recompute arithmetic/multiset membership without calling reconstruct."""
    io.check_schema(report, "economy_native_reconstruction.schema.json")
    for key in (
        "contract_version",
        "source_alias",
        "source_sha256",
        "build_identifier",
    ):
        io.require(report[key] == data[key], "native reconstruction identity differs")
    for key, value in bindings.items():
        io.require(report[key] == value, "native reconstruction binding differs")
    io.require(
        report["rule_assessment"] == rules(data), "unsupported rule compatibility"
    )
    io.require(
        len(report["records"]) == len(data["observations"]["snapshots"]),
        "native reconstruction denominator differs",
    )
    previous, calculations, memberships = {}, 0, 0
    for row, current in zip(
        report["records"], data["observations"]["snapshots"], strict=True
    ):
        key = (current["instance"], current["actor_alias"])
        old = previous.get(key)
        for field in ("observation_id", "instance", "actor_alias", "side"):
            io.require(
                row[field] == current[field],
                "native reconstruction row identity differs",
            )
        io.require(
            row["previous_observation_id"] == (old["observation_id"] if old else None)
            and row["interval_start_tick"] == (old["tick"] if old else None)
            and row["interval_end_tick"] == current["tick"],
            "native interval differs",
        )
        io.require(
            row["scope"] == scope(old, current, source), "native interval scope differs"
        )
        for family, status, reason in (
            ("purchases_round", "purchase_counter_status", "purchase_counter_reason"),
            ("sellback", "sellback_status", "sellback_reason"),
        ):
            expected_reason = interval_reason(old, current, family)
            io.require(
                row[reason] == expected_reason
                and row[status]
                == (
                    "available"
                    if expected_reason == "native_endpoint_difference_only"
                    else "unavailable"
                ),
                "native interval availability differs",
            )
        expected_deltas = []
        if row["purchase_counter_status"] == "available":
            ids = sorted(
                {
                    e["m_nItemDefIndex"]["value"]
                    for s in (old, current)
                    for e in s["purchases_round"]["entries"]
                }
            )
            for item in ids:
                values = [
                    sum(
                        e["m_nCount"]["value"]
                        for e in s["purchases_round"]["entries"]
                        if e["m_nItemDefIndex"]["value"] == item
                    )
                    for s in (old, current)
                ]
                if values[0] != values[1]:
                    expected_deltas.append(
                        {
                            "item_definition_index": item,
                            "before": values[0],
                            "after": values[1],
                            "delta": values[1] - values[0],
                        }
                    )
                    calculations += 1
        io.require(
            row["purchase_counter_differences"] == expected_deltas,
            "native purchase counter arithmetic differs",
        )
        added, removed = [], []
        if row["sellback_status"] == "available":
            before = [
                tuple(e[k]["value"] for k in FIELDS) for e in old["sellback"]["entries"]
            ]
            after = [
                tuple(e[k]["value"] for k in FIELDS)
                for e in current["sellback"]["entries"]
            ]
            remaining = list(before)
            for item in after:
                if item in remaining:
                    remaining.remove(item)
                else:
                    added.append(item)
            removed = remaining
            memberships += 1
        io.require(
            row["sellback_entries_added"]
            == [dict(zip(CHARGE_FIELDS, e, strict=True)) for e in sorted(added)]
            and row["sellback_entries_removed"]
            == [dict(zip(CHARGE_FIELDS, e, strict=True)) for e in sorted(removed)],
            "native sellback membership differs",
        )
        previous[key] = current
    io.require(
        report["summary"] == summary(report), "native reconstruction summary differs"
    )
    return {
        "ok": True,
        "summary": report["summary"],
        "independently_recomputed_counter_differences": calculations,
        "independently_recomputed_sellback_intervals": memberships,
    }


def materialize(
    source_path: Path, output: Path, *, require_observed: bool = False
) -> dict:
    output = validate_output_directory(output)
    data, source, bindings = io.load_candidate(
        source_path, require_observed=require_observed
    )
    report = reconstruct(data, source, bindings)
    validation = validate_report(report, data, source, bindings)
    if require_observed:
        io.require(
            validation["summary"]["records"] > 0, "native reconstruction has no records"
        )
    artifact = _write_new(output / "economy_native_reconstruction.json", report)
    return {
        **validation,
        "source_summary": io.summary(data),
        "artifact": artifact,
        "rule_assessment": report["rule_assessment"],
    }


def main() -> None:
    import json

    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--require-observed", action="store_true")
    args = parser.parse_args()
    print(
        json.dumps(
            materialize(
                args.source, args.output, require_observed=args.require_observed
            ),
            sort_keys=True,
        )
    )


if __name__ == "__main__":
    main()
