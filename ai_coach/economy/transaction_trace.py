"""Read-only transaction-channel evidence. No inferred purchases or refunds."""

from __future__ import annotations

import argparse
import json
from collections import Counter, defaultdict
from pathlib import Path

from jsonschema import Draft202012Validator

from ai_coach.contracts.runtime_contracts import RuntimeContractError
from ai_coach.economy import candidate_io

SCHEMA = "economy_transaction_trace.schema.json"
DATA = "economy_transaction_trace.json"
EVENTS = {
    "item_purchase",
    "item_pickup",
    "item_pickup_slerp",
    "item_remove",
    "item_refund",
}
ACCOUNT = "m_pInGameMoneyServices.m_iAccount"


def require(condition: bool, message: str) -> None:
    if not condition:
        raise RuntimeContractError(message)


def schema_validator():
    schema = json.loads((candidate_io.CONTRACTS / SCHEMA).read_text(encoding="utf-8"))
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


def check_schema(data: dict) -> None:
    require(
        not list(schema_validator().iter_errors(data)),
        "invalid transaction trace schema",
    )


def validate_data(
    data: dict, candidate: dict, *, require_observed: bool = False
) -> dict:
    check_schema(data)
    require(
        all(
            data[k] == candidate[k]
            for k in ("source_alias", "source_sha256", "build_identifier")
        ),
        "trace identity differs",
    )
    rows = data["observations"]["rows"]
    require(
        [r["sequence"] for r in rows] == list(range(1, len(rows) + 1)),
        "trace order differs",
    )
    aliases = {p["participant_alias"] for p in candidate["participants"]}
    rounds = {r["instance"]: r for r in candidate["rounds"]}
    counts, links, missing = Counter(), Counter(), Counter()
    groups = defaultdict(list)
    balances, starts = {}, {}
    previous_frame = -1
    for row in rows:
        ch, src = row["channel"], row["source"]
        counts[f"{ch}:{src}"] += 1
        require(row["frame"] >= previous_frame, "trace frame order differs")
        previous_frame = row["frame"]
        require(
            row["actor_alias"] is None or row["actor_alias"] in aliases,
            "trace actor unknown",
        )
        require(
            row["instance"] is None or row["instance"] in rounds,
            "trace instance unknown",
        )
        if row["actor_alias"] is None:
            missing["actor_unavailable_at_observation"] += 1
        if row["entity_alias"] is None:
            missing["object_unavailable_or_not_applicable"] += 1
        if ch in {"native_event", "sdk_generic"}:
            require(src in EVENTS, "trace event outside probe")
        if ch == "native_event":
            fields = row["value"]["fields"]
            require(
                [f["ordinal"] for f in fields] == list(range(len(fields))),
                "trace field order differs",
            )
            for field in fields:
                observed = field["status"] == "observed"
                require(
                    observed == (field["value"] is not None),
                    "trace field presence differs",
                )
                if observed:
                    require(
                        field["name"]
                        in {
                            "item",
                            "defindex",
                            "silent",
                            "inventory",
                            "canbuy",
                            "team",
                        },
                        "trace payload outside probe",
                    )
                    require(
                        field["wire_type"] == field["descriptor_type"]
                        and field["wire_type"] in range(1, 8),
                        "trace native type differs",
                    )
        if ch in {"property_baseline", "property_update"}:
            value = row["value"]
            require(
                (value["status"] == "observed") == (value["value"] is not None),
                "trace property presence differs",
            )
            require(
                row["entity_alias"] is not None,
                "trace property has no lifetime identity",
            )
            if (
                ch == "property_update"
                and src == ACCOUNT
                and value["status"] == "observed"
            ):
                key = (row["entity_alias"], row["frame"], row["tick"])
                if key not in groups:
                    starts[key] = balances.get(row["entity_alias"])
                groups[key].append(row)
            if src == ACCOUNT:
                balances[row["entity_alias"]] = value["value"]
        instance = rounds.get(row["instance"])
        if instance is None or row["actor_alias"] is None or row["side"] is None:
            missing["opportunity_link_unavailable"] += 1
            continue
        start, end = instance["start_tick"], instance["freeze_end_tick"]
        if end is None or not start < row["tick"] < end:
            missing["outside_strict_freeze_or_boundary_ambiguous"] += 1
            continue
        # Roster is used only to link retrospective execution evidence. It never
        # qualifies T0, player knowledge, or a transaction cause.
        frontier = next(
            f
            for f in candidate["initial_frontiers"]
            if f["instance"] == row["instance"]
        )
        roster = frontier["round_start_probe"]["actors"]
        if not any(
            a["actor_alias"] == row["actor_alias"] and a["side"] == row["side"]
            for a in roster
        ):
            missing["opportunity_roster_disagreement"] += 1
            continue
        oid = f"{data['source_alias']}:economy:{row['instance']:05d}:{row['side']}"
        links[oid] += 1
    repeated = [g for g in groups.values() if len(g) > 1]
    changed = [g for g in repeated if len({r["value"]["value"] for r in g}) > 1]
    compensated = sum(
        starts[k] is not None
        and starts[k] == g[-1]["value"]["value"]
        and any(r["value"]["value"] != starts[k] for r in g)
        for k, g in groups.items()
    )
    if require_observed:
        require(
            counts[f"property_update:{ACCOUNT}"] > 0
            and counts["sdk_callback:ItemPickup"] > 0,
            "trace smoke lacks affected records",
        )
    return {
        "rows": len(rows),
        "channel_counts": dict(sorted(counts.items())),
        "linked_opportunities": dict(sorted(links.items())),
        "abstentions": dict(sorted(missing.items())),
        "same_frame_tick_account_groups": len(repeated),
        "same_frame_tick_distinct_account_groups": len(changed),
        "same_frame_tick_compensated_account_groups": compensated,
        "verified_itemized_transactions": 0,
        "transaction_classification": "abstain_channel_evidence_only",
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
    }


def load_candidate(path: Path, *, require_observed: bool = False) -> tuple[dict, dict]:
    path = candidate_io.source_io._safe_file(path.parent.absolute(), path.name)
    require(
        path.name == DATA and path.stat().st_size <= 64 * 1024 * 1024,
        "invalid trace path or size",
    )
    raw = path.read_bytes()
    data = json.loads(raw)
    check_schema(data)
    source = candidate_io.source_io._safe_file(
        path.parent, "economy_candidate_manifest.json"
    )
    source_raw = source.read_bytes()
    ref = data["source_candidate"]
    require(
        candidate_io.sha(source_raw) == ref["sha256"]
        and len(source_raw) == ref["bytes"],
        "trace source hash differs",
    )
    candidate, manifest_hash, _ = candidate_io.load_source(source)
    require(
        manifest_hash == ref["sha256"] and path.read_bytes() == raw,
        "trace source changed during load",
    )
    return data, validate_data(data, candidate, require_observed=require_observed)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--require-observed", action="store_true")
    args = parser.parse_args()
    output = candidate_io.validate_output_directory(args.output)
    _, report = load_candidate(args.source, require_observed=args.require_observed)
    report["source_trace_sha256"] = candidate_io.sha(args.source.read_bytes())
    candidate_io._write_new(
        output / "economy_transaction_trace_validation.json", report
    )
    print(json.dumps(report))


if __name__ == "__main__":
    main()
