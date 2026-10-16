"""Validate one isolated real-demo observed visual-contact smoke report.

The report is deliberately separate from canonical bundle 3.8.0.  This
validator does not trust the producer's summary: it validates every candidate
row, rebuilds contact episode state, recomputes all counts, and enforces the
coverage and pseudonymisation required by the one-demo smoke gate.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import math
import re
import stat
from collections import Counter
from collections.abc import Mapping
from pathlib import Path
from typing import Any

REPORT_SCHEMA_ID = "stratai.observed_visual_contact_single_demo_smoke@1"
CANDIDATE_SCHEMA_ID = "stratai.observed_visual_contact@1"
CANDIDATE_CONTRACT_VERSION = "1.0.0"
SELECTION_CONTRACT = "causal_round_prefix_with_smoke@1"

SOURCE_ALIAS_PATTERN = re.compile(r"^demo-[0-9a-f]{20}$")
SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")
PARTICIPANT_ALIAS_PATTERN = re.compile(r"^participant-[0-9]{3,}$")
CONTACT_ID_PATTERN = re.compile(r"^contact-[0-9]{3,}$")
RAW_STEAM_ID_PATTERN = re.compile(r"^(?:steam:)?[0-9]{15,20}$", re.IGNORECASE)

EXPECTED_INPUT_FIELDS = [
    "active_smoke_effects",
    "map_geometry",
    "observer_eye_position",
    "observer_scope_and_zoom",
    "observer_view_angles",
    "subject_eye_position",
]
EXPECTED_METHODS = {
    "source_artifact": "replay_data_in_memory",
    "raycast_method": "map_mesh_segment_raycast@1",
    "fov_method": "rectangular_frustum_16_9_from_source_4_3@1",
    "smoke_method": "segment_to_smoke_cylinder@1",
}
BOUNDARIES = {"started", "continued", "ended", "inactive", "unavailable"}
GEOMETRY_ROLES = {"physics_gltf", "physics_buffer"}

REPORT_KEYS = {
    "schema_id",
    "status",
    "source_alias",
    "source_sha256",
    "map_name",
    "candidate_schema_id",
    "candidate_contract_version",
    "build_identifier",
    "processing_duration_ms",
    "selection",
    "geometry_assets",
    "summary",
    "ledger",
}
SELECTION_KEYS = {
    "contract",
    "round_number",
    "start_tick",
    "end_tick",
    "t0_tick",
    "input_round_frames",
    "selected_frames",
    "frame_limit",
}
SUMMARY_KEYS = {
    "evaluations",
    "observed_evaluations",
    "unavailable_evaluations",
    "line_of_sight_clear",
    "line_of_sight_blocked",
    "line_of_sight_unavailable",
    "fov_inside",
    "fov_outside",
    "fov_unavailable",
    "smoke_occluded",
    "smoke_clear",
    "smoke_unavailable",
    "visual_contact_true",
    "visual_contact_false",
    "visual_contact_unavailable",
    "active_smoke_evaluations",
    "boundaries",
    "episodes",
    "closed_episodes",
    "open_episodes",
    "participant_aliases",
    "abstention_reasons",
    "raw_steam_id_violations",
}
LEDGER_KEYS = {
    "schema_id",
    "contract_version",
    "identity_semantics",
    "causal_role",
    "join_keys",
    "t0_tick",
    "sampling",
    "evaluations",
    "contacts",
}
EVALUATION_KEYS = {
    "schema_id",
    "contract_version",
    "identity_semantics",
    "join_keys",
    "round_number",
    "tick",
    "t0_tick",
    "availability_tick",
    "status",
    "causal_role",
    "visibility_scope",
    "source",
    "line_of_sight",
    "field_of_view",
    "smoke_occlusion",
    "visual_contact",
    "boundary",
    "abstention_reasons",
    "provenance",
}
EPISODE_KEYS = {
    "schema_id",
    "contract_version",
    "identity_semantics",
    "join_keys",
    "round_number",
    "t0_tick",
    "start_tick",
    "end_tick",
    "last_confirmed_tick",
    "start_availability_tick",
    "end_availability_tick",
    "status",
    "causal_role",
}
JOIN_KEYS = {"match_id", "round_id", "observer_id", "subject_id", "contact_id"}
PROVENANCE_KEYS = {
    "source_artifact",
    "source_schema_version",
    "source_round",
    "source_frame_tick",
    "geometry_status",
    "eye_position_source",
    "raycast_method",
    "fov_method",
    "smoke_method",
    "input_fields",
}


class VisualContactSmokeValidationError(ValueError):
    """Raised when the isolated smoke report is invalid or insufficient."""


def _fail(message: str) -> None:
    raise VisualContactSmokeValidationError(message)


def _object(value: Any, location: str) -> Mapping[str, Any]:
    if not isinstance(value, Mapping):
        _fail(f"{location}: expected object")
    return value


def _exact_keys(value: Mapping[str, Any], expected: set[str], location: str) -> None:
    missing = sorted(expected - set(value))
    unknown = sorted(set(value) - expected)
    if missing or unknown:
        _fail(f"{location}: missing={missing}, unknown={unknown}")


def _string(value: Any, location: str) -> str:
    if not isinstance(value, str) or not value.strip():
        _fail(f"{location}: expected non-empty string")
    return value


def _integer(value: Any, location: str, *, minimum: int = 0) -> int:
    if isinstance(value, bool) or not isinstance(value, int) or value < minimum:
        _fail(f"{location}: expected integer >= {minimum}")
    return value


def _number(value: Any, location: str, *, positive: bool = False) -> float:
    if isinstance(value, bool) or not isinstance(value, (int, float)):
        _fail(f"{location}: expected number")
    result = float(value)
    if not math.isfinite(result) or result < 0 or (positive and result <= 0):
        qualifier = "positive" if positive else "non-negative"
        _fail(f"{location}: expected finite {qualifier} number")
    return result


def _optional_integer(value: Any, location: str) -> int | None:
    if value is None:
        return None
    return _integer(value, location)


def _sorted_unique_strings(value: Any, location: str) -> list[str]:
    if not isinstance(value, list):
        _fail(f"{location}: expected list")
    result = [_string(item, f"{location}[{index}]") for index, item in enumerate(value)]
    if result != sorted(set(result)):
        _fail(f"{location}: must be sorted and unique")
    return result


def _relative_reference(value: Any, location: str) -> str:
    reference = _string(value, location)
    normalized = reference.replace("\\", "/")
    parts = normalized.split("/")
    if (
        "\\" in reference
        or ":" in reference
        or normalized.startswith("/")
        or any(part in {"", ".", ".."} for part in parts)
    ):
        _fail(f"{location}: expected normalized relative reference")
    return reference


def _validate_selection(value: Any) -> dict[str, int]:
    selection = _object(value, "report.selection")
    _exact_keys(selection, SELECTION_KEYS, "report.selection")
    if selection["contract"] != SELECTION_CONTRACT:
        _fail("report.selection.contract: unexpected value")
    result = {
        "round_number": _integer(
            selection["round_number"], "report.selection.round_number", minimum=1
        ),
        "start_tick": _integer(selection["start_tick"], "report.selection.start_tick"),
        "end_tick": _integer(selection["end_tick"], "report.selection.end_tick"),
        "t0_tick": _integer(selection["t0_tick"], "report.selection.t0_tick"),
        "input_round_frames": _integer(
            selection["input_round_frames"],
            "report.selection.input_round_frames",
            minimum=1,
        ),
        "selected_frames": _integer(
            selection["selected_frames"],
            "report.selection.selected_frames",
            minimum=3,
        ),
        "frame_limit": _integer(
            selection["frame_limit"], "report.selection.frame_limit", minimum=16
        ),
    }
    if not result["start_tick"] <= result["end_tick"] == result["t0_tick"]:
        _fail("report.selection: require start_tick <= end_tick == t0_tick")
    if result["frame_limit"] > 256:
        _fail("report.selection.frame_limit: must be between 16 and 256")
    if result["selected_frames"] > result["frame_limit"]:
        _fail("report.selection.selected_frames: exceeds frame_limit")
    if result["selected_frames"] > result["input_round_frames"]:
        _fail("report.selection.selected_frames: exceeds input_round_frames")
    return result


def _is_symlink_or_reparse_point(path: Path) -> bool:
    if path.is_symlink():
        return True
    attributes = getattr(path.lstat(), "st_file_attributes", 0)
    reparse_flag = getattr(stat, "FILE_ATTRIBUTE_REPARSE_POINT", 0)
    return bool(attributes & reparse_flag)


def _geometry_root(maps_root: Path) -> Path:
    if not isinstance(maps_root, Path):
        _fail("maps_root: expected pathlib.Path")
    try:
        if _is_symlink_or_reparse_point(maps_root):
            _fail("maps_root: symlinks and reparse points are forbidden")
        resolved = maps_root.resolve(strict=True)
    except OSError as error:
        _fail(f"maps_root: cannot resolve directory: {error}")
    if not resolved.is_dir():
        _fail("maps_root: expected directory")
    return resolved


def _verify_geometry_asset(
    maps_root: Path,
    reference: str,
    expected_sha256: str,
    location: str,
) -> None:
    candidate = maps_root
    try:
        for part in reference.split("/"):
            candidate /= part
            if _is_symlink_or_reparse_point(candidate):
                _fail(
                    f"{location}.reference: symlinks and reparse points are forbidden"
                )
        resolved = candidate.resolve(strict=True)
        resolved.relative_to(maps_root)
    except VisualContactSmokeValidationError:
        raise
    except (OSError, ValueError) as error:
        _fail(f"{location}.reference: cannot resolve inside maps_root: {error}")
    if not resolved.is_file():
        _fail(f"{location}.reference: expected regular file")
    digest = hashlib.sha256()
    try:
        with resolved.open("rb") as asset_file:
            for chunk in iter(lambda: asset_file.read(1024 * 1024), b""):
                digest.update(chunk)
    except OSError as error:
        _fail(f"{location}.reference: cannot read geometry asset: {error}")
    if digest.hexdigest() != expected_sha256:
        _fail(f"{location}.sha256: geometry asset hash mismatch")


def _validate_geometry_assets(value: Any, maps_root: Path) -> None:
    if not isinstance(value, list) or not value:
        _fail("report.geometry_assets: expected non-empty list")
    resolved_root = _geometry_root(maps_root)
    roles: set[str] = set()
    references: set[str] = set()
    for index, item in enumerate(value):
        location = f"report.geometry_assets[{index}]"
        asset = _object(item, location)
        _exact_keys(asset, {"reference", "sha256", "role"}, location)
        reference = _relative_reference(asset["reference"], f"{location}.reference")
        if reference in references:
            _fail(f"{location}.reference: duplicate geometry asset")
        references.add(reference)
        checksum = asset["sha256"]
        if not isinstance(checksum, str) or not SHA256_PATTERN.fullmatch(checksum):
            _fail(f"{location}.sha256: expected lowercase SHA-256")
        role = asset["role"]
        if role not in GEOMETRY_ROLES:
            _fail(f"{location}.role: unexpected value")
        if role in roles:
            _fail(f"{location}.role: duplicate geometry role")
        roles.add(role)
        _verify_geometry_asset(resolved_root, reference, checksum, location)
    if roles != GEOMETRY_ROLES:
        _fail("report.geometry_assets: physics_gltf and physics_buffer are required")
    canonical = sorted(value, key=lambda item: (item["role"], item["reference"]))
    if value != canonical:
        _fail("report.geometry_assets: assets must use canonical role/reference order")


def _validate_sampling(value: Any) -> None:
    sampling = _object(value, "report.ledger.sampling")
    _exact_keys(
        sampling,
        {"target_hz", "tick_rate", "period_ticks", "strategy"},
        "report.ledger.sampling",
    )
    if sampling["target_hz"] != 16:
        _fail("report.ledger.sampling.target_hz: expected 16")
    tick_rate = _number(sampling["tick_rate"], "report.ledger.sampling.tick_rate")
    if tick_rate < 16:
        _fail("report.ledger.sampling.tick_rate: expected at least 16")
    period = _number(
        sampling["period_ticks"], "report.ledger.sampling.period_ticks", positive=True
    )
    if not math.isclose(period, tick_rate / 16.0, rel_tol=1e-9, abs_tol=1e-9):
        _fail("report.ledger.sampling.period_ticks: must equal tick_rate / 16")
    if sampling["strategy"] != "round_anchored_right_closed_no_carry_forward":
        _fail("report.ledger.sampling.strategy: unexpected value")


def _participant_alias(value: Any, location: str) -> str:
    alias = _string(value, location)
    if RAW_STEAM_ID_PATTERN.fullmatch(alias) or alias.lower().startswith("steam:"):
        _fail(f"{location}: raw Steam ID is forbidden")
    if not PARTICIPANT_ALIAS_PATTERN.fullmatch(alias):
        _fail(f"{location}: expected local participant alias")
    number = int(alias.removeprefix("participant-"))
    if number <= 0 or alias != f"participant-{number:03d}":
        _fail(f"{location}: expected canonical local participant alias")
    return alias


def _contact_alias(value: Any, location: str) -> str:
    contact_id = _string(value, location)
    if RAW_STEAM_ID_PATTERN.fullmatch(contact_id) or "steam:" in contact_id.lower():
        _fail(f"{location}: raw Steam ID is forbidden")
    if CONTACT_ID_PATTERN.fullmatch(contact_id) is None:
        _fail(f"{location}: expected pseudonymised contact ID")
    number = int(contact_id.removeprefix("contact-"))
    if number <= 0 or contact_id != f"contact-{number:03d}":
        _fail(f"{location}: expected canonical pseudonymised contact ID")
    return contact_id


def _validate_join_keys(
    value: Any,
    location: str,
    source_alias: str,
    round_number: int,
) -> dict[str, Any]:
    joins = _object(value, location)
    _exact_keys(joins, JOIN_KEYS, location)
    if joins["match_id"] != source_alias:
        _fail(f"{location}.match_id: must equal report source_alias")
    expected_round_id = f"{source_alias}:round:{round_number:03d}"
    if joins["round_id"] != expected_round_id:
        _fail(f"{location}.round_id: unexpected pseudonymised round ID")
    observer = _participant_alias(joins["observer_id"], f"{location}.observer_id")
    subject = _participant_alias(joins["subject_id"], f"{location}.subject_id")
    if observer == subject:
        _fail(f"{location}: observer_id and subject_id must differ")
    contact = None
    if joins["contact_id"] is not None:
        contact = _contact_alias(joins["contact_id"], f"{location}.contact_id")
    return {
        "round_id": expected_round_id,
        "observer": observer,
        "subject": subject,
        "contact_id": contact,
    }


def _validate_boolean_fact(value: Any, location: str) -> bool | None:
    fact = _object(value, location)
    _exact_keys(fact, {"status", "value"}, location)
    if fact["status"] == "unavailable":
        if fact["value"] is not None:
            _fail(f"{location}: unavailable value must be null")
        return None
    if fact["status"] != "derived" or not isinstance(fact["value"], bool):
        _fail(f"{location}: derived fact requires a boolean")
    return fact["value"]


def _validate_fov_fact(value: Any, location: str) -> bool | None:
    fact = _object(value, location)
    _exact_keys(fact, {"status", "inside", "reference_horizontal_fov_deg"}, location)
    if fact["status"] == "unavailable":
        if (
            fact["inside"] is not None
            or fact["reference_horizontal_fov_deg"] is not None
        ):
            _fail(f"{location}: unavailable FOV values must be null")
        return None
    if fact["status"] != "derived" or not isinstance(fact["inside"], bool):
        _fail(f"{location}: derived FOV requires a boolean")
    _number(
        fact["reference_horizontal_fov_deg"],
        f"{location}.reference_horizontal_fov_deg",
        positive=True,
    )
    return fact["inside"]


def _validate_smoke_fact(value: Any, location: str) -> tuple[bool | None, int | None]:
    fact = _object(value, location)
    _exact_keys(fact, {"status", "occluded", "active_smoke_count"}, location)
    if fact["status"] == "unavailable":
        if fact["occluded"] is not None or fact["active_smoke_count"] is not None:
            _fail(f"{location}: unavailable smoke values must be null")
        return None, None
    if fact["status"] != "derived" or not isinstance(fact["occluded"], bool):
        _fail(f"{location}: derived smoke requires a boolean")
    count = _integer(fact["active_smoke_count"], f"{location}.active_smoke_count")
    if fact["occluded"] and count == 0:
        _fail(f"{location}: occluded=true requires active_smoke_count > 0")
    return fact["occluded"], count


def _validate_provenance(
    value: Any,
    location: str,
    round_number: int,
    availability_tick: int,
    selection: Mapping[str, int],
    line_of_sight: bool | None,
) -> None:
    provenance = _object(value, location)
    _exact_keys(provenance, PROVENANCE_KEYS, location)
    for key, expected in EXPECTED_METHODS.items():
        if provenance[key] != expected:
            _fail(f"{location}.{key}: unexpected value")
    source_schema = provenance["source_schema_version"]
    if source_schema is not None:
        _integer(source_schema, f"{location}.source_schema_version", minimum=1)
    if provenance["source_round"] != round_number:
        _fail(f"{location}.source_round: must match evaluation round")
    source_tick = _integer(
        provenance["source_frame_tick"], f"{location}.source_frame_tick"
    )
    if not selection["start_tick"] <= source_tick <= availability_tick:
        _fail(
            f"{location}.source_frame_tick: must be within the causal selected window"
        )
    geometry_status = provenance["geometry_status"]
    if geometry_status == "not_loaded":
        _fail(
            f"{location}.geometry_status: global smoke report requires loaded geometry"
        )
    if geometry_status not in {"loaded", "position_unavailable"}:
        _fail(f"{location}.geometry_status: unexpected value")
    if line_of_sight is not None and geometry_status != "loaded":
        _fail(f"{location}.geometry_status: derived LOS requires loaded geometry")
    if line_of_sight is None and geometry_status != "position_unavailable":
        _fail(
            f"{location}.geometry_status: unavailable LOS requires position_unavailable"
        )
    _string(provenance["eye_position_source"], f"{location}.eye_position_source")
    if provenance["input_fields"] != EXPECTED_INPUT_FIELDS:
        _fail(f"{location}.input_fields: unexpected causal input fields")


def _validate_evaluation(
    value: Any,
    index: int,
    source_alias: str,
    ledger_t0: int,
    selection: Mapping[str, int],
) -> dict[str, Any]:
    location = f"report.ledger.evaluations[{index}]"
    evaluation = _object(value, location)
    _exact_keys(evaluation, EVALUATION_KEYS, location)
    expected_constants = {
        "schema_id": CANDIDATE_SCHEMA_ID,
        "contract_version": CANDIDATE_CONTRACT_VERSION,
        "identity_semantics": "join_only",
        "causal_role": "model_input_observation",
        "visibility_scope": "enemy_visual_contact_evaluation",
        "source": "replay_visual_contact_derivation",
    }
    for key, expected in expected_constants.items():
        if evaluation[key] != expected:
            _fail(f"{location}.{key}: unexpected value")
    round_number = _integer(
        evaluation["round_number"], f"{location}.round_number", minimum=1
    )
    if round_number != selection["round_number"]:
        _fail(f"{location}.round_number: outside selected round")
    joins = _validate_join_keys(
        evaluation["join_keys"], f"{location}.join_keys", source_alias, round_number
    )
    tick = _integer(evaluation["tick"], f"{location}.tick")
    if not selection["start_tick"] <= tick <= selection["end_tick"]:
        _fail(f"{location}.tick: outside selected window")
    if evaluation["t0_tick"] != ledger_t0:
        _fail(f"{location}.t0_tick: must match ledger t0_tick")
    availability_tick = _integer(
        evaluation["availability_tick"], f"{location}.availability_tick"
    )
    if not selection["start_tick"] <= availability_tick <= tick <= ledger_t0:
        _fail(f"{location}: require selection start <= availability_tick <= tick <= T0")

    line_of_sight = _validate_boolean_fact(
        evaluation["line_of_sight"], f"{location}.line_of_sight"
    )
    inside_fov = _validate_fov_fact(
        evaluation["field_of_view"], f"{location}.field_of_view"
    )
    smoke_occluded, active_smoke_count = _validate_smoke_fact(
        evaluation["smoke_occlusion"], f"{location}.smoke_occlusion"
    )
    visual_contact = _validate_boolean_fact(
        evaluation["visual_contact"], f"{location}.visual_contact"
    )
    reasons = _sorted_unique_strings(
        evaluation["abstention_reasons"], f"{location}.abstention_reasons"
    )
    inputs_available = (
        line_of_sight is not None
        and inside_fov is not None
        and smoke_occluded is not None
    )
    if inputs_available:
        expected_contact = bool(line_of_sight and inside_fov and not smoke_occluded)
        if (
            evaluation["status"] != "observed"
            or visual_contact is None
            or visual_contact != expected_contact
            or reasons
        ):
            _fail(f"{location}: visual contact must equal LOS && FOV && !smoke")
    elif (
        evaluation["status"] != "unavailable"
        or visual_contact is not None
        or not reasons
    ):
        _fail(f"{location}: missing inputs require unavailable contact and abstention")

    boundary = evaluation["boundary"]
    if boundary not in BOUNDARIES:
        _fail(f"{location}.boundary: unexpected value")
    contact_id = joins["contact_id"]
    if visual_contact is None:
        if boundary != "unavailable":
            _fail(f"{location}: unavailable contact requires unavailable boundary")
    elif visual_contact:
        if boundary not in {"started", "continued"} or contact_id is None:
            _fail(f"{location}: true contact requires started/continued and contact_id")
    elif boundary == "inactive":
        if contact_id is not None:
            _fail(f"{location}: inactive boundary cannot carry contact_id")
    elif boundary != "ended" or contact_id is None:
        _fail(f"{location}: false contact requires inactive or ended boundary")

    _validate_provenance(
        evaluation["provenance"],
        f"{location}.provenance",
        round_number,
        availability_tick,
        selection,
        line_of_sight,
    )
    return {
        "round_number": round_number,
        "round_id": joins["round_id"],
        "observer": joins["observer"],
        "subject": joins["subject"],
        "contact_id": contact_id,
        "tick": tick,
        "availability_tick": availability_tick,
        "status": evaluation["status"],
        "line_of_sight": line_of_sight,
        "inside_fov": inside_fov,
        "smoke_occluded": smoke_occluded,
        "active_smoke_count": active_smoke_count,
        "visual_contact": visual_contact,
        "boundary": boundary,
        "reasons": reasons,
        "source_frame_tick": evaluation["provenance"]["source_frame_tick"],
    }


def _validate_episode(
    value: Any,
    index: int,
    source_alias: str,
    ledger_t0: int,
    selection: Mapping[str, int],
) -> dict[str, Any]:
    location = f"report.ledger.contacts[{index}]"
    episode = _object(value, location)
    _exact_keys(episode, EPISODE_KEYS, location)
    expected_constants = {
        "schema_id": CANDIDATE_SCHEMA_ID,
        "contract_version": CANDIDATE_CONTRACT_VERSION,
        "identity_semantics": "join_only",
        "causal_role": "model_input_observation",
    }
    for key, expected in expected_constants.items():
        if episode[key] != expected:
            _fail(f"{location}.{key}: unexpected value")
    round_number = _integer(
        episode["round_number"], f"{location}.round_number", minimum=1
    )
    if round_number != selection["round_number"]:
        _fail(f"{location}.round_number: outside selected round")
    joins = _validate_join_keys(
        episode["join_keys"], f"{location}.join_keys", source_alias, round_number
    )
    if joins["contact_id"] is None:
        _fail(f"{location}.join_keys.contact_id: episode requires contact ID")
    if episode["t0_tick"] != ledger_t0:
        _fail(f"{location}.t0_tick: must match ledger t0_tick")
    start_tick = _integer(episode["start_tick"], f"{location}.start_tick")
    last_tick = _integer(
        episode["last_confirmed_tick"], f"{location}.last_confirmed_tick"
    )
    start_availability = _integer(
        episode["start_availability_tick"], f"{location}.start_availability_tick"
    )
    if not (
        selection["start_tick"]
        <= start_availability
        <= start_tick
        <= last_tick
        <= selection["end_tick"]
        <= ledger_t0
    ):
        _fail(f"{location}: invalid causal start/confirmation ticks")
    end_tick = _optional_integer(episode["end_tick"], f"{location}.end_tick")
    end_availability = _optional_integer(
        episode["end_availability_tick"], f"{location}.end_availability_tick"
    )
    status = episode["status"]
    if status == "closed":
        if end_tick is None or end_availability is None:
            _fail(f"{location}: closed episode requires end ticks")
        if not last_tick <= end_availability <= end_tick <= selection["end_tick"]:
            _fail(f"{location}: invalid causal end ticks")
    elif status == "open_at_t0":
        if end_tick is not None or end_availability is not None:
            _fail(f"{location}: open episode cannot contain end ticks")
    else:
        _fail(f"{location}.status: unexpected value")
    return {
        "round_number": round_number,
        "observer": joins["observer"],
        "subject": joins["subject"],
        "contact_id": joins["contact_id"],
        "start_tick": start_tick,
        "start_availability_tick": start_availability,
        "last_confirmed_tick": last_tick,
        "end_tick": end_tick,
        "end_availability_tick": end_availability,
        "status": status,
    }


def _validate_episode_state(
    evaluations: list[dict[str, Any]], episodes: list[dict[str, Any]]
) -> None:
    episode_by_id: dict[str, dict[str, Any]] = {}
    for episode in episodes:
        contact_id = episode["contact_id"]
        if contact_id in episode_by_id:
            _fail("report.ledger.contacts: duplicate contact_id")
        episode_by_id[contact_id] = episode

    seen_evaluations: set[tuple[int, int, str, str]] = set()
    active: dict[tuple[int, str, str], str] = {}
    ordered = sorted(
        evaluations,
        key=lambda item: (
            item["tick"],
            item["round_number"],
            item["observer"],
            item["subject"],
        ),
    )
    for row in ordered:
        evaluation_key = (
            row["round_number"],
            row["tick"],
            row["observer"],
            row["subject"],
        )
        if evaluation_key in seen_evaluations:
            _fail("report.ledger.evaluations: duplicate observer/subject/tick key")
        seen_evaluations.add(evaluation_key)
        pair = (row["round_number"], row["observer"], row["subject"])
        active_id = active.get(pair)
        boundary, contact_id = row["boundary"], row["contact_id"]
        if contact_id is not None:
            episode = episode_by_id.get(contact_id)
            if episode is None:
                _fail("report.ledger.evaluations: contact_id has no episode")
            if (
                episode["round_number"],
                episode["observer"],
                episode["subject"],
            ) != pair:
                _fail("report.ledger.evaluations: contact_id points to another pair")
        if boundary == "started":
            if active_id is not None or contact_id is None:
                _fail(
                    "report.ledger.evaluations: started boundary while contact is active"
                )
            active[pair] = contact_id
        elif boundary == "continued":
            if active_id is None or contact_id != active_id:
                _fail(
                    "report.ledger.evaluations: continued boundary changed contact_id"
                )
        elif boundary == "ended":
            if active_id is None or contact_id != active_id:
                _fail(
                    "report.ledger.evaluations: ended boundary has no matching active contact"
                )
            del active[pair]
        elif boundary == "inactive":
            if active_id is not None or contact_id is not None:
                _fail(
                    "report.ledger.evaluations: inactive boundary while contact is active"
                )
        elif active_id is None:
            if contact_id is not None:
                _fail(
                    "report.ledger.evaluations: unavailable inactive pair carried contact_id"
                )
        elif contact_id != active_id:
            _fail(
                "report.ledger.evaluations: unavailable active pair changed contact_id"
            )

    for contact_id, episode in episode_by_id.items():
        rows = [row for row in evaluations if row["contact_id"] == contact_id]
        starts = [row for row in rows if row["boundary"] == "started"]
        if len(starts) != 1:
            _fail(
                "report.ledger.contacts: episode requires exactly one started boundary"
            )
        start = starts[0]
        if (
            start["tick"] != episode["start_tick"]
            or start["availability_tick"] != episode["start_availability_tick"]
        ):
            _fail(
                "report.ledger.contacts: episode start differs from started evaluation"
            )
        confirmed = [row["tick"] for row in rows if row["visual_contact"] is True]
        if not confirmed or max(confirmed) != episode["last_confirmed_tick"]:
            _fail(
                "report.ledger.contacts: last_confirmed_tick differs from evaluations"
            )
        ended = [row for row in rows if row["boundary"] == "ended"]
        pair = (episode["round_number"], episode["observer"], episode["subject"])
        if episode["status"] == "closed":
            if len(ended) != 1:
                _fail(
                    "report.ledger.contacts: closed episode requires exactly one ended boundary"
                )
            if (
                ended[0]["tick"] != episode["end_tick"]
                or ended[0]["availability_tick"] != episode["end_availability_tick"]
            ):
                _fail(
                    "report.ledger.contacts: episode end differs from ended evaluation"
                )
            if active.get(pair) == contact_id:
                _fail("report.ledger.contacts: closed episode remains active")
        else:
            if ended or active.get(pair) != contact_id:
                _fail("report.ledger.contacts: open episode is not active at T0")


def _validate_count_map(value: Any, location: str) -> dict[str, int]:
    mapping = _object(value, location)
    result: dict[str, int] = {}
    for key, count in mapping.items():
        name = _string(key, f"{location}.key")
        result[name] = _integer(count, f"{location}.{name}", minimum=1)
    return dict(sorted(result.items()))


def _recompute_summary(
    evaluations: list[dict[str, Any]], episodes: list[dict[str, Any]]
) -> dict[str, Any]:
    participants = {
        participant
        for row in evaluations
        for participant in (row["observer"], row["subject"])
    }
    return {
        "evaluations": len(evaluations),
        "observed_evaluations": sum(row["status"] == "observed" for row in evaluations),
        "unavailable_evaluations": sum(
            row["status"] == "unavailable" for row in evaluations
        ),
        "line_of_sight_clear": sum(row["line_of_sight"] is True for row in evaluations),
        "line_of_sight_blocked": sum(
            row["line_of_sight"] is False for row in evaluations
        ),
        "line_of_sight_unavailable": sum(
            row["line_of_sight"] is None for row in evaluations
        ),
        "fov_inside": sum(row["inside_fov"] is True for row in evaluations),
        "fov_outside": sum(row["inside_fov"] is False for row in evaluations),
        "fov_unavailable": sum(row["inside_fov"] is None for row in evaluations),
        "smoke_occluded": sum(row["smoke_occluded"] is True for row in evaluations),
        "smoke_clear": sum(row["smoke_occluded"] is False for row in evaluations),
        "smoke_unavailable": sum(row["smoke_occluded"] is None for row in evaluations),
        "visual_contact_true": sum(
            row["visual_contact"] is True for row in evaluations
        ),
        "visual_contact_false": sum(
            row["visual_contact"] is False for row in evaluations
        ),
        "visual_contact_unavailable": sum(
            row["visual_contact"] is None for row in evaluations
        ),
        "active_smoke_evaluations": sum(
            row["active_smoke_count"] is not None and row["active_smoke_count"] > 0
            for row in evaluations
        ),
        "boundaries": dict(
            sorted(Counter(row["boundary"] for row in evaluations).items())
        ),
        "episodes": len(episodes),
        "closed_episodes": sum(row["status"] == "closed" for row in episodes),
        "open_episodes": sum(row["status"] == "open_at_t0" for row in episodes),
        "participant_aliases": len(participants),
        "abstention_reasons": dict(
            sorted(
                Counter(
                    reason for row in evaluations for reason in row["reasons"]
                ).items()
            )
        ),
        "raw_steam_id_violations": 0,
    }


def _validate_summary(value: Any, expected: Mapping[str, Any]) -> None:
    summary = _object(value, "report.summary")
    _exact_keys(summary, SUMMARY_KEYS, "report.summary")
    normalized: dict[str, Any] = {}
    for key in SUMMARY_KEYS - {"boundaries", "abstention_reasons"}:
        normalized[key] = _integer(summary[key], f"report.summary.{key}")
    normalized["boundaries"] = _validate_count_map(
        summary["boundaries"], "report.summary.boundaries"
    )
    normalized["abstention_reasons"] = _validate_count_map(
        summary["abstention_reasons"], "report.summary.abstention_reasons"
    )
    if normalized != dict(expected):
        _fail(f"report.summary: expected recomputed summary {dict(expected)}")
    if normalized["raw_steam_id_violations"] != 0:
        _fail("report.summary.raw_steam_id_violations: must be zero")
    required_positive = {
        "observed_evaluations",
        "line_of_sight_clear",
        "line_of_sight_blocked",
        "fov_inside",
        "fov_outside",
        "smoke_occluded",
        "smoke_clear",
        "visual_contact_true",
        "visual_contact_false",
        "active_smoke_evaluations",
        "episodes",
    }
    missing = sorted(key for key in required_positive if normalized[key] == 0)
    for boundary in ("started", "ended", "inactive"):
        if normalized["boundaries"].get(boundary, 0) == 0:
            missing.append(f"boundaries.{boundary}")
    if missing:
        _fail(f"report.summary: required real-demo coverage is zero for {missing}")


def validate_visual_contact_smoke_report(
    value: Mapping[str, Any], maps_root: Path
) -> dict[str, Any]:
    """Validate and return the independently recomputed summary."""

    report = _object(value, "report")
    _exact_keys(report, REPORT_KEYS, "report")
    if report["schema_id"] != REPORT_SCHEMA_ID or report["status"] != "passed":
        _fail("report: schema or status mismatch")
    checksum = report["source_sha256"]
    alias = report["source_alias"]
    if (
        not isinstance(checksum, str)
        or not SHA256_PATTERN.fullmatch(checksum)
        or not isinstance(alias, str)
        or not SOURCE_ALIAS_PATTERN.fullmatch(alias)
        or alias != f"demo-{checksum[:20]}"
    ):
        _fail("report: source alias/checksum mismatch")
    _string(report["map_name"], "report.map_name")
    if (
        report["candidate_schema_id"] != CANDIDATE_SCHEMA_ID
        or report["candidate_contract_version"] != CANDIDATE_CONTRACT_VERSION
    ):
        _fail("report: candidate schema/version mismatch")
    _string(report["build_identifier"], "report.build_identifier")
    _integer(report["processing_duration_ms"], "report.processing_duration_ms")
    selection = _validate_selection(report["selection"])
    _validate_geometry_assets(report["geometry_assets"], maps_root)

    ledger = _object(report["ledger"], "report.ledger")
    _exact_keys(ledger, LEDGER_KEYS, "report.ledger")
    expected_ledger_constants = {
        "schema_id": CANDIDATE_SCHEMA_ID,
        "contract_version": CANDIDATE_CONTRACT_VERSION,
        "identity_semantics": "join_only",
        "causal_role": "model_input_observation",
    }
    for key, expected in expected_ledger_constants.items():
        if ledger[key] != expected:
            _fail(f"report.ledger.{key}: unexpected value")
    ledger_joins = _object(ledger["join_keys"], "report.ledger.join_keys")
    _exact_keys(ledger_joins, {"match_id"}, "report.ledger.join_keys")
    if ledger_joins["match_id"] != alias:
        _fail("report.ledger.join_keys.match_id: must equal source_alias")
    ledger_t0 = _integer(ledger["t0_tick"], "report.ledger.t0_tick")
    if ledger_t0 != selection["t0_tick"]:
        _fail("report.ledger.t0_tick: must equal selection t0_tick")
    _validate_sampling(ledger["sampling"])

    raw_evaluations = ledger["evaluations"]
    raw_episodes = ledger["contacts"]
    if not isinstance(raw_evaluations, list) or not raw_evaluations:
        _fail("report.ledger.evaluations: expected non-empty list")
    if not isinstance(raw_episodes, list):
        _fail("report.ledger.contacts: expected list")
    evaluations = [
        _validate_evaluation(item, index, alias, ledger_t0, selection)
        for index, item in enumerate(raw_evaluations)
    ]
    episodes = [
        _validate_episode(item, index, alias, ledger_t0, selection)
        for index, item in enumerate(raw_episodes)
    ]
    unique_frames = {row["source_frame_tick"] for row in evaluations}
    if len(unique_frames) > selection["selected_frames"]:
        _fail("report.selection.selected_frames: fewer than ledger source frames")
    _validate_episode_state(evaluations, episodes)
    recomputed = _recompute_summary(evaluations, episodes)
    _validate_summary(report["summary"], recomputed)
    return recomputed


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("report", type=Path)
    parser.add_argument(
        "--maps-root",
        required=True,
        type=Path,
        help="Read-only root containing the referenced map geometry assets",
    )
    arguments = parser.parse_args()
    try:
        payload = json.loads(arguments.report.read_text(encoding="utf-8"))
        summary = validate_visual_contact_smoke_report(
            payload, maps_root=arguments.maps_root
        )
    except (
        OSError,
        UnicodeError,
        json.JSONDecodeError,
        VisualContactSmokeValidationError,
    ) as error:
        print(
            json.dumps(
                {"ok": False, "error": str(error)}, ensure_ascii=False, sort_keys=True
            )
        )
        return 1
    print(
        json.dumps({"ok": True, "summary": summary}, ensure_ascii=False, sort_keys=True)
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
