"""Verify opportunity provenance and recompute checkpoint arithmetic from source.

This validator does not call the opportunity builder or its change calculator.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path

from ai_coach.contracts import economy_initial_frontiers as source_initial
from ai_coach.contracts import validate_economy_candidate as source_io
from ai_coach.contracts.runtime_contracts import RuntimeContractError
from ai_coach.economy import candidate_io as io
from ai_coach.economy import opportunities as domain

require = source_io._require


def _expected_initial(
    source: dict, round_: dict, frontier: dict, side: str
) -> tuple[dict | None, str]:
    if frontier["round_link"]["status"] != "verified":
        return None, "competitive_round_unavailable"
    own = [a for a in frontier["round_start_probe"]["actors"] if a["side"] == side]
    if not own:
        return None, "initial_roster_unavailable"
    try:
        return source_initial.initial_team_state(
            source, round_["instance"], own[0]["actor_alias"], round_["start_tick"]
        ), "complete_competitive_initial_team"
    except RuntimeContractError:
        return None, "complete_initial_team_unavailable"


def _verify_differences(
    records: list, initial: dict | None, checkpoint: dict | None, side: str
) -> tuple[int, int]:
    require(
        len(records) == (len(initial["actors"]) if initial else 0),
        "checkpoint actor difference count mismatch",
    )
    numeric, inventories = 0, 0
    if initial is None:
        return numeric, inventories
    end = checkpoint["tick"] if checkpoint else None
    final_by_alias = (
        {a["actor_alias"]: a for a in checkpoint["actors"]} if checkpoint else {}
    )
    for record, actor in zip(records, initial["actors"], strict=True):
        alias = actor["actor_alias"]
        require(record["actor_alias"] == alias, "checkpoint actor identity differs")
        last = final_by_alias.get(alias)
        if last is not None and (not last["connected"] or last["side"] != side):
            last = None
        for key, difference in record["changes"].items():
            before = actor["facts"][key]["value"]
            exists = last is not None and last["facts"][key]["status"] == "observed"
            after = last["facts"][key]["value"] if exists else None
            delta = after - before if exists and key in domain.NUMERIC_FIELDS else None
            require(
                difference
                == {
                    "before": before,
                    "after": after,
                    "delta": delta,
                    "status": "checkpoint_difference" if exists else "unavailable",
                    "availability_tick": end if exists else None,
                },
                "economic checkpoint difference differs from source arithmetic",
            )
            numeric += bool(exists and key in domain.NUMERIC_FIELDS)
        inv = last["facts"]["inventory"] if last else None
        old = {i["entity_alias"]: i for i in actor["facts"]["inventory"]["value"]}
        new = (
            {i["entity_alias"]: i for i in inv["value"]}
            if inv and inv["status"] == "observed"
            else None
        )
        conflicts = sorted(
            k for k in old if new is not None and k in new and old[k] != new[k]
        )
        valid = new is not None and not conflicts
        transition = record["inventory_transition"]
        retained = (
            sorted(
                (old[k] for k in old if new is not None and k in new),
                key=lambda x: x["entity_alias"],
            )
            if valid
            else None
        )
        added = (
            sorted(
                (new[k] for k in new if k not in old), key=lambda x: x["entity_alias"]
            )
            if valid
            else None
        )
        removed = (
            sorted(
                (old[k] for k in old if k not in new), key=lambda x: x["entity_alias"]
            )
            if valid
            else None
        )
        require(
            transition
            == {
                "status": "entity_membership_difference" if valid else "unavailable",
                "retained": retained,
                "added": added,
                "removed": removed,
                "identity_conflicts": conflicts,
                "availability_tick": end if valid else None,
            },
            "inventory difference differs from source membership",
        )
        inventories += valid
    return numeric, inventories


def validate_data(candidate: dict, source: dict, protocol: dict) -> dict:
    io.check_schema(candidate, "economy_opportunities.schema.json")
    for key in ("source_alias", "source_sha256", "map_id", "tick_rate"):
        require(candidate[key] == source[key], "opportunity source identity mismatch")
    require(
        source["contract_version"] == protocol["source_contract_version"],
        "unsupported opportunity source version",
    )
    rows = candidate["opportunities"]
    require(len(rows) == len(source["rounds"]) * 2, "opportunity denominator changed")
    numeric = inventories = 0
    for index, row in enumerate(rows):
        round_ = source["rounds"][index // 2]
        frontier = source["initial_frontiers"][index // 2]
        side = "CT" if index % 2 == 0 else "T"
        start, end = round_["start_tick"], round_["freeze_end_tick"]
        valid_window = end is not None and end > start
        if not valid_window:
            end = None
        require(
            row["opportunity_id"]
            == f"{source['source_alias']}:economy:{round_['instance']:05d}:{side}"
            and row["instance"] == round_["instance"]
            and row["round_number"] == round_["round_number"]
            and row["side"] == side
            and row["t0_tick"] == start,
            "opportunity identity/order/anchor mismatch",
        )
        state, reason = _expected_initial(source, round_, frontier, side)
        require(
            row["initial_state"] == state
            and row["initial_gate"]
            == {"status": "available" if state else "unavailable", "reason": reason},
            "initial opportunity gate or features differ from causal source",
        )
        window = row["execution_window"]
        require(
            window["status"] == ("observed" if valid_window else "unavailable")
            and window["start_exclusive_tick"] == start
            and window["end_exclusive_tick"] == end
            and window["duration_ticks"] == (end - start if end is not None else None),
            "freeze window must use exact observed boundaries",
        )
        rules = row["rules"]
        missing = (
            list(protocol["rules"]["server_convars"])
            if state is None
            else [
                k
                for k in protocol["rules"]["server_convars"]
                if state["context"][k]["status"] != "observed"
            ]
        )
        expected_config = (
            "unavailable" if state is None else "partial" if missing else "observed"
        )
        expected_zone = (
            "unavailable"
            if state is None
            else "all_in_buy_zone"
            if all(a["facts"]["buy_zone"]["value"] is True for a in state["actors"])
            else "not_all_in_buy_zone"
        )
        require(
            rules["missing_convars"] == missing
            and rules["server_configuration"] == expected_config
            and rules["initial_buy_zone"] == expected_zone
            and rules["legacy_references"] == source["configs"],
            "economic rule compatibility lacks native provenance",
        )
        aliases = {
            a["actor_alias"]
            for a in frontier["round_start_probe"]["actors"]
            if a["side"] == side
        }
        source_events = (
            round_["execution"]["events"] + round_["reconciliation"]["other_events"]
        )
        expected_events, expected_boundary = [], []
        for event in sorted(source_events, key=lambda e: e["callback_sequence"]):
            if event["actor_alias"] not in aliases:
                continue
            if end is not None and start < event["tick"] < end:
                expected_events.append(event)
            elif event["tick"] == start or (end is not None and event["tick"] == end):
                expected_boundary.append(event)
        execution = row["observed_action"]
        require(
            execution["callbacks"] == expected_events
            and execution["boundary_callbacks"] == expected_boundary,
            "economic callbacks crossed actor/window/boundary",
        )
        early = [
            e for e in frontier["pre_probe_callbacks"] if e["actor_alias"] in aliases
        ]
        require(
            execution["pre_probe_callbacks"] == early,
            "pre-probe activity differs from source",
        )
        checkpoint = None
        native_checkpoint = round_["execution"]["freeze_snapshot"]
        if (
            end is not None
            and native_checkpoint is not None
            and native_checkpoint["tick"] == end
        ):
            checkpoint = {
                "tick": end,
                "callback_sequence": native_checkpoint["callback_sequence"],
                "actors": [
                    a
                    for a in native_checkpoint["actors"]
                    if a["actor_alias"] in aliases
                ],
            }
        require(
            execution["checkpoint"] == checkpoint,
            "freeze checkpoint is late, changed or includes other actors",
        )
        changes, membership = _verify_differences(
            execution["actor_changes"], state, checkpoint, side
        )
        numeric += changes
        inventories += membership
        blockers = list(domain.ACTION_REASONS)
        if state is None:
            blockers.append("initial_team_unavailable")
        if not valid_window:
            blockers.append("freeze_window_unavailable")
        if checkpoint is None:
            blockers.append("exact_freeze_checkpoint_unavailable")
        if any(e["side"] != side for e in expected_events + expected_boundary):
            blockers.append("execution_actor_side_changed")
        if expected_boundary:
            blockers.append("boundary_callbacks_have_no_subtick_order")
        require(
            execution["blockers"] == blockers
            and execution["status"]
            == (
                "partial"
                if state is not None and checkpoint is not None and valid_window
                else "unavailable"
            ),
            "economic action uncertainty was dropped",
        )
    return {
        "independently_recomputed_numeric_differences": numeric,
        "independently_recomputed_inventory_transitions": inventories,
        "team_round_records_verified": len(rows),
    }


def validate_candidate(
    path: Path,
    source_path: Path,
    *,
    require_observed: bool = False,
    catalog_path: Path | None = None,
) -> dict:
    path = source_io._safe_file(path.parent.absolute(), path.name)
    require(
        path.name == domain.MANIFEST_FILE, "unexpected opportunity manifest filename"
    )
    manifest, _ = source_io._read(path)
    io.check_schema(manifest, "economy_opportunities_manifest.schema.json")
    ref = manifest["artifact"]
    candidate, payload = source_io._read(source_io._safe_file(path.parent, ref["path"]))
    require(
        len(payload) == ref["bytes"] and io.sha(payload) == ref["sha256"],
        "opportunity artifact hash/size mismatch",
    )
    source, source_manifest_hash, source_data_hash = io.load_source(source_path)
    protocol, protocol_hash, catalog, catalog_hash = io.load_bindings(catalog_path)
    for key, value in {
        "source_candidate_manifest_sha256": source_manifest_hash,
        "source_candidate_data_sha256": source_data_hash,
        "protocol_sha256": protocol_hash,
        "catalog_sha256": catalog_hash,
        "catalog_version": catalog["version"],
    }.items():
        require(
            candidate.get(key) == manifest[key] == value,
            "opportunity binding hash/version mismatch",
        )
    for key in (
        "contract_version",
        "source_alias",
        "source_sha256",
        "comparison_allowed",
        "training_allowed",
        "product_supported",
    ):
        require(
            candidate.get(key) == manifest[key],
            "opportunity manifest/data metadata differs",
        )
    recomputed = validate_data(candidate, source, protocol)
    counts = domain.summary(candidate)
    require(manifest["summary"] == counts, "opportunity summary differs")
    if require_observed:
        require(
            all(
                counts[k] > 0
                for k in (
                    "team_round_records",
                    "initial_teams_available",
                    "partial_execution_records",
                    "observed_numeric_differences",
                    "inventory_transitions",
                    "execution_callbacks",
                )
            ),
            "economic opportunity smoke has empty affected records",
        )
    return {
        "ok": True,
        "schema_id": "stratai.economy_opportunities_validation@1",
        "source_alias": candidate["source_alias"],
        "source_sha256": candidate["source_sha256"],
        "summary": counts,
        **recomputed,
        "training_allowed": False,
        "comparison_allowed": False,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("manifest", type=Path)
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--catalog", type=Path)
    parser.add_argument("--require-observed", action="store_true")
    args = parser.parse_args()
    try:
        report = validate_candidate(
            args.manifest,
            args.source,
            require_observed=args.require_observed,
            catalog_path=args.catalog,
        )
    except (OSError, ValueError, RuntimeContractError) as exc:
        parser.exit(1, f"economic opportunities rejected: {exc}\n")
    print(json.dumps(report, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
