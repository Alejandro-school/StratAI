"""Validate native clocks and entity lifetimes without inferring transactions."""

from __future__ import annotations

import argparse
import json
from collections import Counter
from pathlib import Path

from jsonschema import Draft202012Validator

from ai_coach.economy import candidate_io, transaction_trace

DATA = "economy_native_identity.json"
SCHEMA = "economy_native_identity.schema.json"
require = transaction_trace.require
INVENTORY = "m_pWeaponServices.m_hMyWeapons"
USER_MESSAGES = {
    "CUserMessageItemPickup",
    "CCSUsrMsg_ItemPickup",
    "CCSUsrMsg_ItemDrop",
    "CSVCMsg_UserMessage:economic",
}


def schema_validator():
    schema = json.loads((candidate_io.CONTRACTS / SCHEMA).read_text(encoding="utf-8"))
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


def reference_status(handle: int, kind: str, live: dict) -> tuple[str, str | None]:
    index, serial = handle & 16383, handle >> 14
    if index == 16383:
        return "invalid_sentinel", None
    entity = live.get(index)
    if entity is None:
        return "entity_unavailable", None
    value = entity["value"]
    if value["entity_serial"] != serial:
        return "serial_mismatch", None
    if value["entity_kind"] != kind:
        return "kind_mismatch", None
    return "resolved", entity["entity_alias"]


def validate_data(
    data: dict, trace: dict, candidate: dict, *, require_observed: bool = False
) -> dict:
    require(
        not list(schema_validator().iter_errors(data)), "invalid native identity schema"
    )
    require(
        all(
            data[k] == trace[k]
            for k in ("source_alias", "source_sha256", "build_identifier")
        ),
        "identity source differs",
    )
    observations = data["observations"]
    clocks, rows = observations["clocks"], observations["rows"]
    require(
        [c["sequence"] for c in clocks] == sorted(c["sequence"] for c in clocks),
        "clock order differs",
    )
    require(
        [r["sequence"] for r in rows] == sorted(r["sequence"] for r in rows),
        "identity row order differs",
    )
    entries = sorted([*clocks, *rows], key=lambda r: r["sequence"])
    require(
        [r["sequence"] for r in entries] == list(range(1, len(entries) + 1)),
        "identity sequence coverage differs",
    )
    trace_rows = {r["sequence"]: r for r in trace["observations"]["rows"]}
    rounds = {r["instance"]: r for r in candidate["rounds"]}
    frontiers = {f["instance"]: f for f in candidate["initial_frontiers"]}
    aliases = {
        r["actor_alias"] for r in trace_rows.values() if r["actor_alias"] is not None
    }
    entities, live, seen_trace = {}, {}, set()
    counts, references, timing, user_messages = (
        Counter(),
        Counter(),
        Counter(),
        Counter(),
    )
    latest_clock, last_frame, epoch = None, -1, 0
    for row in entries:
        require(row["frame"] >= last_frame, "identity frame order differs")
        last_frame = row["frame"]
        if "network_tick" in row:
            reason = "first_sample" if latest_clock is None else "continuous"
            if latest_clock is not None:
                if latest_clock["network_tick"] is None or row["network_tick"] is None:
                    reason = "missing_tick_boundary"
                elif row["demo_tick"] < latest_clock["demo_tick"]:
                    reason = "demo_tick_regression"
                elif row["network_tick"] < latest_clock["network_tick"]:
                    reason = "network_tick_regression"
            if reason != "continuous":
                epoch += 1
            require(
                (row["epoch"], row["epoch_reason"]) == (epoch, reason),
                "clock epoch differs",
            )
            latest_clock = row
            counts[f"clock:{reason}"] += 1
            continue
        kind, value, source = row["kind"], row["value"], row["source"]
        counts[kind] += 1
        require(
            row["clock_sequence"]
            == (None if latest_clock is None else latest_clock["sequence"]),
            "clock provenance is not causal latest prefix",
        )
        require(
            row["actor_alias"] is None or row["actor_alias"] in aliases,
            "identity actor unknown",
        )
        alias = row["entity_alias"]
        require(
            row["instance"] is None or row["instance"] in rounds,
            "native instance unknown",
        )
        if kind == "entity_created":
            require(
                alias is not None and alias not in entities,
                "entity lifetime alias reused",
            )
            entities[alias] = row
            if value["entity_index"] in live:
                counts["live_index_replaced_without_destroy_callback"] += 1
            live[value["entity_index"]] = row
        elif alias is not None:
            require(alias in entities, "entity identity backfilled from future")
        if kind == "entity_destroyed":
            require(alias is not None, "destroy identity absent")
            require(source == entities[alias]["source"], "destroy class differs")
            index = entities[alias]["value"]["entity_index"]
            if live.get(index, {}).get("entity_alias") == alias:
                del live[index]
        if kind in {"reference_baseline", "reference_callback"}:
            require(alias is not None, "reference source identity absent")
            require(
                source == "m_hOwnerEntity"
                or (
                    source.startswith(INVENTORY + ".")
                    and source[len(INVENTORY) + 1 :] in {f"{i:04d}" for i in range(64)}
                ),
                "reference source outside probe",
            )
            target_kind = "pawn" if source == "m_hOwnerEntity" else "equipment"
            ref = value["reference"]
            if ref["handle"] is None:
                require(
                    ref["status"] in {"absent", "unsupported_type"}
                    and ref["target_alias"] is None,
                    "missing reference acquired identity",
                )
            else:
                expected = reference_status(ref["handle"], target_kind, live)
                require(
                    (ref["status"], ref["target_alias"]) == expected,
                    "reference serial/lifetime reconciliation differs",
                )
            references[f"{target_kind}:{ref['status']}"] += 1
        if kind.startswith(("reference_", "definition_", "inventory_length_")):
            require(alias is not None, "property source entity absent")
            phase = (
                "creation_frame_or_replay"
                if row["frame"] == entities[alias]["frame"]
                else "later_frame_callback"
            )
            require(value["phase"] == phase, "creation replay mislabeled")
            counts[f"property_phase:{phase}"] += 1
        if kind.startswith("definition_"):
            require(
                source == "m_iItemDefinitionIndex"
                and (value["status"] == "observed") == (value["value"] is not None),
                "definition presence differs",
            )
        if kind.startswith("inventory_length_"):
            require(
                source == INVENTORY
                and (value["status"] == "observed") == (value["length"] is not None),
                "inventory length presence differs",
            )
        if kind == "trace_event":
            seq = value["trace_sequence"]
            if seq is not None:
                origin = trace_rows.get(seq)
                require(
                    origin is not None and seq not in seen_trace,
                    "trace event missing or reused",
                )
                require(
                    origin["channel"] in {"native_event", "sdk_callback"},
                    "invalid trace channel",
                )
                require(
                    all(
                        row[k] == origin[k]
                        for k in ("frame", "source", "actor_alias", "side", "instance")
                    )
                    and row["demo_tick"] == origin["tick"],
                    "trace observation link differs",
                )
                expected_tick = (
                    origin["value"]["server_tick"]
                    if origin["channel"] == "native_event"
                    else None
                )
                require(
                    value["server_tick"] == expected_tick, "native server tick changed"
                )
                seen_trace.add(seq)
            if value["server_tick"] is not None:
                if latest_clock is None or latest_clock["network_tick"] is None:
                    status = "network_tick_absent"
                elif (
                    latest_clock["frame"] != row["frame"]
                    or latest_clock["demo_tick"] != row["demo_tick"]
                ):
                    status = "network_tick_from_other_frame"
                elif value["server_tick"] == latest_clock["network_tick"]:
                    status = "equal_in_same_frame"
                else:
                    status = "different_in_same_frame"
                timing[status] += 1
        if kind == "user_message":
            require(
                source in USER_MESSAGES
                and alias is None
                and row["actor_alias"] is None
                and row["side"] is None,
                "user message invented actor/entity",
            )
            user_messages[source] += 1
            if source in {"CUserMessageItemPickup", "CCSUsrMsg_ItemPickup"}:
                require(
                    "item" in value
                    and (value["status"] == "observed") == (value["item"] is not None),
                    "pickup message presence differs",
                )
            elif source == "CCSUsrMsg_ItemDrop":
                require("item_id_alias" in value, "drop message misinterpreted")
            else:
                require("payload_sha256" in value, "envelope misinterpreted")
        if kind == "round_boundary":
            require(
                source in {"RoundStart", "RoundFreezetimeEnd", "RoundEndOfficial"},
                "boundary source invalid",
            )
            if row["instance"] is not None:
                round_ = rounds[row["instance"]]
                expected = {
                    "RoundStart": round_["start_tick"],
                    "RoundFreezetimeEnd": round_["freeze_end_tick"],
                    "RoundEndOfficial": frontiers[row["instance"]]["official_end_tick"],
                }[source]
                require(
                    expected == row["demo_tick"],
                    "native boundary differs from verified instance",
                )
    require(
        observations["user_message_counts"]
        == {s: user_messages[s] for s in USER_MESSAGES},
        "user message coverage differs",
    )
    if require_observed:
        require(
            len(clocks) > 0
            and counts["entity_created"] > 0
            and counts["entity_destroyed"] > 0
            and references["equipment:resolved"] > 0
            and len(seen_trace) > 0,
            "native identity smoke lacks affected records",
        )
    return {
        "clocks": len(clocks),
        "clock_epochs": epoch,
        "rows": len(rows),
        "counts": dict(sorted(counts.items())),
        "reference_counts": dict(sorted(references.items())),
        "native_event_clock_comparison": dict(sorted(timing.items())),
        "trace_links": len(seen_trace),
        "user_message_counts": observations["user_message_counts"],
        "entity_lifetimes": len(entities),
        "live_at_end": len(live),
        "global_tick_offset_applied": False,
        "itemized_transactions_verified": 0,
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
    }


def load_candidate(path: Path, *, require_observed: bool = False) -> tuple[dict, dict]:
    path = candidate_io.source_io._safe_file(path.parent.absolute(), path.name)
    require(
        path.name == DATA and path.stat().st_size <= 128 * 1024 * 1024,
        "invalid native identity path or size",
    )
    raw = path.read_bytes()
    data = json.loads(raw)
    require(
        not list(schema_validator().iter_errors(data)), "invalid native identity schema"
    )
    source = candidate_io.source_io._safe_file(path.parent, transaction_trace.DATA)
    source_raw = source.read_bytes()
    ref = data["source_trace"]
    require(
        len(source_raw) == ref["bytes"]
        and candidate_io.sha(source_raw) == ref["sha256"],
        "native identity trace hash differs",
    )
    trace, _ = transaction_trace.load_candidate(source)
    require(
        source.read_bytes() == source_raw and path.read_bytes() == raw,
        "native identity source changed during load",
    )
    candidate, manifest_hash, _ = candidate_io.load_source(
        path.parent / "economy_candidate_manifest.json"
    )
    require(
        manifest_hash == trace["source_candidate"]["sha256"],
        "identity candidate chain changed",
    )
    return data, validate_data(
        data, trace, candidate, require_observed=require_observed
    )


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--require-observed", action="store_true")
    args = parser.parse_args()
    output = candidate_io.validate_output_directory(args.output)
    _, report = load_candidate(args.source, require_observed=args.require_observed)
    report["source_identity_sha256"] = candidate_io.sha(args.source.read_bytes())
    candidate_io._write_new(output / "economy_native_identity_validation.json", report)
    print(json.dumps(report))


if __name__ == "__main__":
    main()
