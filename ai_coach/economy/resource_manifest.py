"""Bounded resource-reference evidence; no asset content or effective rules."""

from __future__ import annotations

import json
from pathlib import Path

from jsonschema import Draft202012Validator

from ai_coach.economy import rule_context
from ai_coach.economy.candidate_io import CONTRACTS, sha

DATA = "economy_resource_manifest.json"
SCHEMA = "economy_resource_manifest.schema.json"
CODEC = "clarity-7fb3f1d07564a12efa99194d45cfbf5762ba5910-bounded-v1"
require = rule_context.require


def schema_validator() -> Draft202012Validator:
    schema = json.loads((CONTRACTS / SCHEMA).read_text(encoding="utf-8"))
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


def validate_data(data: dict, context: dict, *, require_observed: bool = False) -> dict:
    require(
        not list(schema_validator().iter_errors(data)),
        "invalid resource manifest schema",
    )
    require(
        all(data[k] == context[k] for k in rule_context.IDENTITY),
        "manifest identity differs",
    )
    native = context["observations"]["sessions"]
    require(len(data["sessions"]) == len(native), "manifest coverage differs")
    decoded_count = entries = 0
    for row, origin in zip(data["sessions"], native, strict=True):
        require(
            all(row[k] == origin[k] for k in ("tick", "sequence")),
            "manifest causal provenance differs",
        )
        require(row["source"] == origin["manifest"], "manifest fingerprint differs")
        source, status, value = row["source"], row["status"], row["decoded"]
        require(
            (status == "absent") == (not source["present"]), "manifest presence differs"
        )
        if source["present"]:
            require(
                (status == "read_limit_exceeded") == (source["bytes"] > 1048576),
                "manifest read limit differs",
            )
        require(
            (value is not None) == (status == "decoded"),
            "partial manifest decode leaked",
        )
        if value is None:
            continue
        decoded_count += 1
        require(
            value["outer_consumed_bits"] == 25 + 8 * value["encoded_bytes"],
            "encoded size differs",
        )
        require(
            value["outer_consumed_bits"] + value["outer_padding_bits"]
            == source["bytes"] * 8,
            "outer consumption differs",
        )
        require(
            value["inner_consumed_bits"] + value["inner_padding_bits"]
            == value["decoded_bytes"] * 8,
            "inner consumption differs",
        )
        if value["compression"] == "none":
            require(
                value["encoded_bytes"] == value["decoded_bytes"],
                "uncompressed size differs",
            )
        else:
            require(value["encoded_bytes"] >= 11, "compressed frame too short")
        nt, nd = len(value["type_name_hashes"]), len(value["directory_name_hashes"])
        require(
            value["inner_consumed_bits"] >= 48 + 8 * (nt + nd),
            "table sizes exceed decoded bytes",
        )
        require(
            [e["ordinal"] for e in value["entries"]]
            == list(range(len(value["entries"]))),
            "manifest ordinal coverage differs",
        )
        for entry in value["entries"]:
            require(
                entry["type_index"] < nt and entry["directory_index"] < nd,
                "manifest table index invalid",
            )
        entries += len(value["entries"])
    if require_observed:
        require(
            decoded_count > 0 and entries > 0, "manifest smoke lacks affected records"
        )
    return {
        "sessions": len(native),
        "decoded_manifests": decoded_count,
        "reference_entries": entries,
        "exact_build_asset_binding": "not_accredited",
    }


def load_candidate(
    path: Path, *, require_observed: bool = False
) -> tuple[dict, dict, dict]:
    path = rule_context.base._safe_file(Path(path).parent.absolute(), Path(path).name)
    require(
        path.name == DATA and path.stat().st_size <= 8 * 1024 * 1024,
        "invalid resource manifest path or size",
    )
    data = json.loads(path.read_text(encoding="utf-8"))
    source = rule_context.base._safe_file(path.parent, rule_context.MANIFEST)
    raw = source.read_bytes()
    ref = data.get("source_context", {})
    require(
        ref == {"path": rule_context.MANIFEST, "sha256": sha(raw), "bytes": len(raw)},
        "manifest context binding differs",
    )
    context, _, _, _ = rule_context.load_candidate(
        source, require_observed=require_observed
    )
    require(source.read_bytes() == raw, "manifest context changed during load")
    return (
        data,
        context,
        validate_data(data, context, require_observed=require_observed),
    )


def available_sessions(data: dict, tick: int, sequence: int) -> list[dict]:
    """Callback availability is not evidence of a player's knowledge."""
    return [
        row
        for row in data["sessions"]
        if row["tick"] <= tick and row["sequence"] <= sequence
    ]
