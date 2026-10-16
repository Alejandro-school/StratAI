"""Read-only session evidence. Reference-name digests never identify asset content."""

from __future__ import annotations

import json
from pathlib import Path

from jsonschema import Draft202012Validator

from ai_coach.economy import rule_context
from ai_coach.economy.candidate_io import CONTRACTS, sha

DATA = "economy_session_payload.json"
SCHEMA = "economy_session_payload.schema.json"
PROBE = rule_context.PROBE.with_name("economy_session_payload_probe.json")
PATHS = (
    "CSVCMsg_ServerInfo.game_session_manifest",
    "CSVCMsg_ServerInfo.game_session_config.data",
)
require = rule_context.require


def probe() -> tuple[dict, str]:
    raw = PROBE.read_bytes()
    value = json.loads(raw)
    require(
        value["version"] == "1" and len(value["fields"]) == 9,
        "unknown session payload probe",
    )
    return value, sha(raw)


def schema_validator() -> Draft202012Validator:
    value = json.loads((CONTRACTS / SCHEMA).read_text(encoding="utf-8"))
    Draft202012Validator.check_schema(value)
    return Draft202012Validator(value)


def validate_data(data: dict, context: dict, *, require_observed: bool = False) -> dict:
    error = next(schema_validator().iter_errors(data), None)
    require(error is None, "invalid session payload schema")
    spec, fingerprint = probe()
    require(data["probe_sha256"] == fingerprint, "session probe differs")
    require(
        data["economic_probe_sha256"] == rule_context.probe()[1],
        "economic name probe differs",
    )
    require(
        all(data[k] == context[k] for k in rule_context.IDENTITY),
        "session source identity differs",
    )
    sessions = data["observations"]["sessions"]
    native = context["observations"]["sessions"]
    require(len(sessions) == len(native), "session coverage differs")
    decoded = integers = references = 0
    for row, source in zip(sessions, native, strict=True):
        require(
            all(row[k] == source[k] for k in ("sequence", "tick")),
            "session causal provenance differs",
        )
        require(
            [b["source_path"] for b in row["blocks"]] == list(PATHS),
            "session block paths differ",
        )
        for index, block in enumerate(row["blocks"]):
            origin = source[("manifest", "configuration_data")[index]]
            require(block["source"] == origin, "native block fingerprint differs")
            status = block["status"]
            if not origin["present"]:
                require(status == "absent", "absent block decoded")
            else:
                require(status != "absent", "present block marked absent")
                if origin["bytes"] > 65536:
                    require(status == "read_limit_exceeded", "read limit ignored")
                else:
                    require(status != "read_limit_exceeded", "spurious read limit")
            require(
                (block["format"], block["compression"])
                in {("unverified", "unverified"), ("vbkv_alt_end_11", "none")},
                "format/compression mismatch",
            )
            if index == 0:
                require(
                    status in {"absent", "read_limit_exceeded", "unsupported_format"}
                    and block["format"] == "unverified",
                    "unverified manifest interpreted",
                )
            if status != "decoded":
                require(
                    not block["fields"]
                    and not block["economic_names_seen"]
                    and block["nodes"] == block["scalars"] == 0
                    and not block["checksum_verified"],
                    "partial decode leaked",
                )
                continue
            decoded += 1
            require(
                index == 1
                and block["format"] == "vbkv_alt_end_11"
                and block["checksum_verified"]
                and origin["bytes"] >= 9,
                "invalid decoded block",
            )
            require(block["scalars"] <= block["nodes"], "scalar count exceeds nodes")
            names = block["economic_names_seen"]
            require(
                names == sorted(set(names))
                and set(names) <= set(rule_context.probe()[0]["convars"]),
                "unprobed economic names retained",
            )
            fields = block["fields"]
            require(
                sorted({f["path"] for f in fields}) == sorted(spec["fields"])
                and [f["path"] for f in fields] == sorted(f["path"] for f in fields),
                "selected field coverage differs",
            )
            node_indices = [f["node_index"] for f in fields if f["present"]]
            require(
                len(set(node_indices)) == len(node_indices)
                and all(
                    n is not None and 1 <= n <= block["nodes"] for n in node_indices
                ),
                "selected node provenance differs",
            )
            for path in spec["fields"]:
                group = [f for f in fields if f["path"] == path]
                require(
                    [f["occurrence"] for f in group] == list(range(len(group)))
                    and (all(f["present"] for f in group) or len(group) == 1),
                    "field occurrence coverage differs",
                )
                positions = [f["node_index"] for f in group if f["present"]]
                require(
                    positions == sorted(positions), "native occurrence order differs"
                )
            present = sum(f["present"] for f in fields)
            require(present <= block["nodes"], "field count exceeds nodes")
            for field in fields:
                expected = spec["fields"][field["path"]]
                typ = field["native_type"]
                state = field["status"]
                if not field["present"]:
                    require(
                        state == "not_seen_in_decoded_tree"
                        and typ is None
                        and field["node_index"] is None,
                        "absent field has type or value",
                    )
                elif expected == "int32" and typ == 2:
                    require(
                        state == "observed_native_integer"
                        and field["value"] is not None,
                        "integer type/status mismatch",
                    )
                    integers += 1
                elif expected == "reference_name_sha256" and typ == 1:
                    require(
                        state == "observed_reference_name_hash"
                        and field["reference_name_sha256"] is not None,
                        "reference type/status mismatch",
                    )
                    references += 1
                else:
                    require(
                        state == "native_type_unsupported", "wrong type interpreted"
                    )
                require(
                    (field["value"] is not None) == (state == "observed_native_integer")
                    and (field["reference_name_sha256"] is not None)
                    == (state == "observed_reference_name_hash"),
                    "field payload/status differs",
                )
    if require_observed:
        require(
            decoded > 0 and integers > 0 and references > 0,
            "session smoke lacks affected records",
        )
    return {
        "sessions": len(sessions),
        "decoded_blocks": decoded,
        "native_integers": integers,
        "reference_name_hashes": references,
        "exact_build_asset_binding": "not_accredited",
    }


def load_candidate(
    path: Path, *, require_observed: bool = False
) -> tuple[dict, dict, dict]:
    path = rule_context.base._safe_file(Path(path).parent.absolute(), Path(path).name)
    require(
        path.name == DATA and path.stat().st_size <= 8 * 1024 * 1024,
        "invalid session artifact path or size",
    )
    data = json.loads(path.read_text(encoding="utf-8"))
    # Fixed same-directory source; never follow arbitrary paths from the artifact.
    source = rule_context.base._safe_file(path.parent, rule_context.MANIFEST)
    source_raw = source.read_bytes()
    ref = data.get("source_context", {})
    require(
        ref.get("path") == rule_context.MANIFEST
        and ref.get("sha256") == sha(source_raw)
        and ref.get("bytes") == len(source_raw),
        "session context binding differs",
    )
    context, _, _, _ = rule_context.load_candidate(
        source, require_observed=require_observed
    )
    require(source.read_bytes() == source_raw, "session context changed during load")
    return (
        data,
        context,
        validate_data(data, context, require_observed=require_observed),
    )


def available_sessions(data: dict, tick: int, sequence: int) -> list[dict]:
    """Development inspection only; callback order does not prove player knowledge."""
    return [
        row
        for row in data["observations"]["sessions"]
        if row["tick"] <= tick and row["sequence"] <= sequence
    ]
