"""Explicit, fail-closed contracts for the spatial 3.9 candidate (never 3.8)."""

from __future__ import annotations

import copy
import hashlib
import json
import re
from functools import cache
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator

from ai_coach.contracts.runtime_contracts import (
    _validate_observed_safety,
    validate_decision_example,
)

VERSION = "3.9.0-candidate.2"
MANIFEST_ID = "stratai.canonical_spatial_candidate_manifest@2"
LOADER_ID = "stratai.spatial_candidate_loader@2"
VALIDATOR_ID = "stratai.spatial_candidate_validator@2"
JOIN_ID = "stratai.spatial_fact_join@1"
EXAMPLE_ID = "stratai.ai_coach_decision_example@3"
NAV_PHYSICAL_REASON = "nav_physical_connection_not_accredited"
JOIN_POLICY = "exact_source_round_participant_tick_no_carry_forward@1"
BRIDGE_ID = "stratai.spatial_participant_bridge@1"
KEY_CONTRACT = "sha256(source_sha256 + LF + canonical_actor_id)@1"
FIELDS = {
    "FLD-GEOMETRIC-CLEARANCE-SECTORS": "exposure_angle",
    "FLD-GEOMETRIC-OCCLUSION": "cover",
    "FLD-NAV-EXIT-PATH": "exit_route",
}
SCHEMAS = Path(__file__).resolve().parent


class SpatialCandidateError(ValueError):
    """The candidate must not be consumed after this failure."""


def require(condition: bool, message: str) -> None:
    if not condition:
        raise SpatialCandidateError(message)


def _pairs(pairs: list[tuple[str, Any]]) -> dict:
    result = {}
    for key, value in pairs:
        require(key not in result, "duplicate JSON key")
        result[key] = value
    return result


def decode(payload: bytes | str) -> Any:
    def reject(value: str) -> None:
        raise SpatialCandidateError(f"non-finite JSON constant: {value}")

    return json.loads(payload, object_pairs_hook=_pairs, parse_constant=reject)


def compact(value: Any) -> bytes:
    return (
        json.dumps(
            value,
            sort_keys=True,
            ensure_ascii=False,
            separators=(",", ":"),
            allow_nan=False,
        ).encode()
        + b"\n"
    )


def sha256(path: Path) -> str:
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def scoped_key(checksum: str, identity: str) -> str:
    return hashlib.sha256((checksum + "\n" + identity).encode()).hexdigest()


@cache
def schema_validator(filename: str) -> Draft202012Validator:
    schema = decode((SCHEMAS / filename).read_bytes())
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


def validate_schema(value: dict, filename: str) -> None:
    errors = list(schema_validator(filename).iter_errors(value))
    require(
        not errors,
        f"{filename}: schema mismatch" + (f": {errors[0].message}" if errors else ""),
    )


def reject_private_or_tactical(value: Any) -> None:
    if isinstance(value, dict):
        for key, child in value.items():
            require(
                key.lower()
                not in {
                    "tactical_assessment",
                    "responses",
                    "adjudications",
                    "human_response",
                    "steam_id",
                    "steamid",
                    "display_name",
                    "raw_identity",
                },
                "private identity or human/tactical input in candidate example",
            )
            reject_private_or_tactical(child)
    elif isinstance(value, list):
        for child in value:
            reject_private_or_tactical(child)
    elif isinstance(value, str):
        require(
            not re.search(r"steam:|(?<![a-zA-Z0-9])[0-9]{17}(?![a-zA-Z0-9])", value),
            "raw identity in candidate example",
        )


def validate_candidate_example(example: dict) -> None:
    validate_schema(example, "decision_example_v3.schema.json")
    reject_private_or_tactical(example)
    legacy = copy.deepcopy(example)
    legacy["schema_id"] = "stratai.ai_coach_decision_example@1"
    legacy["contract_version"] = "1.0.0"
    for key in ("candidate_version", "training_allowed", "map_id", "spatial_evidence"):
        legacy.pop(key)
    facts = legacy["observed"].pop("spatial_facts")
    validate_decision_example(legacy)
    for field_id, fact in facts.items():
        evidence = example["spatial_evidence"][field_id]
        if field_id == "FLD-NAV-EXIT-PATH":
            diagnostic = evidence["diagnostic"]
            require(
                NAV_PHYSICAL_REASON in evidence["reason_codes"],
                "NAV physical boundary lacks its abstention cause",
            )
            require(
                evidence["status"]
                == ("missing" if diagnostic["status"] == "missing" else "null"),
                "NAV diagnostic cannot credit physical evidence",
            )
            require(
                not diagnostic["values"]
                if diagnostic["status"] == "missing"
                else set(diagnostic["values"])
                == {"sample", "candidates", "posture_context"},
                "NAV diagnostic values do not match the source join",
            )
            validate_spatial_ticks(diagnostic["values"], example["t0_tick"])
        require(
            fact["status"] in {"derived", "unavailable"},
            "invalid spatial feature status",
        )
        if fact["status"] == "unavailable":
            require(
                not fact["values"] and bool(evidence["reason_codes"]),
                "unavailable fact must abstain with a cause",
            )
        require(fact["availability_tick"] <= example["t0_tick"], "future spatial fact")
        require(
            evidence["status"] == "present"
            if fact["status"] == "derived"
            else evidence["status"] in {"missing", "null"},
            "fact/join status mismatch",
        )
        if evidence["status"] == "missing":
            require(
                evidence["row_reference"] is None
                and evidence["sample_tick"] is None
                and not fact["values"],
                "unjoined fact contains values",
            )
        else:
            require(
                evidence["row_reference"] is not None, "joined fact lacks reference"
            )
            require(
                evidence["sample_tick"]
                == example["t0_tick"]
                == fact["availability_tick"],
                "spatial join is not exact T0",
            )
        _validate_observed_safety(fact["values"])
        validate_spatial_ticks(fact["values"], example["t0_tick"])


def validate_spatial_ticks(value: Any, t0_tick: int) -> None:
    if isinstance(value, dict):
        for key, child in value.items():
            if key == "availability_tick" and child is not None:
                require(
                    type(child) is int and 0 <= child <= t0_tick,
                    "nested spatial temporal violation",
                )
            validate_spatial_ticks(child, t0_tick)
    elif isinstance(value, list):
        for child in value:
            validate_spatial_ticks(child, t0_tick)


def feature_values(value: Any) -> Any:
    """Project validated facts; all IDs/provenance stay in spatial_evidence."""
    if isinstance(value, dict):
        return {
            key: feature_values(child)
            for key, child in value.items()
            if key not in {"tactical_assessment", "provenance"}
            and not key.endswith(("_id", "_ids", "_ref", "_refs", "_alias"))
        }
    if isinstance(value, list):
        return [feature_values(child) for child in value]
    return copy.deepcopy(value)
