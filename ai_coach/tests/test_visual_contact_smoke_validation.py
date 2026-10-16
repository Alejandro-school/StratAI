from __future__ import annotations

import copy
import hashlib
import json
import sys
from pathlib import Path

import pytest

from ai_coach.contracts.validate_visual_contact_smoke import (
    VisualContactSmokeValidationError,
    main,
    validate_visual_contact_smoke_report,
)

SOURCE_SHA256 = "a" * 64
SOURCE_ALIAS = "demo-" + "a" * 20
ROUND_ID = SOURCE_ALIAS + ":round:001"
INPUT_FIELDS = [
    "active_smoke_effects",
    "map_geometry",
    "observer_eye_position",
    "observer_scope_and_zoom",
    "observer_view_angles",
    "subject_eye_position",
]


def _boolean_fact(value: bool | None) -> dict:
    return {
        "status": "derived" if value is not None else "unavailable",
        "value": value,
    }


def _fov_fact(value: bool | None) -> dict:
    return {
        "status": "derived" if value is not None else "unavailable",
        "inside": value,
        "reference_horizontal_fov_deg": 90 if value is not None else None,
    }


def _smoke_fact(value: bool | None, count: int | None) -> dict:
    return {
        "status": "derived" if value is not None else "unavailable",
        "occluded": value,
        "active_smoke_count": count,
    }


def _contact_id(index: int = 1) -> str:
    return f"contact-{index:03d}"


def _evaluation(
    *,
    tick: int,
    observer: str,
    subject: str,
    line_of_sight: bool | None,
    inside_fov: bool | None,
    smoke_occluded: bool | None,
    active_smoke_count: int | None,
    boundary: str,
    contact_id: str | None,
    reasons: list[str] | None = None,
) -> dict:
    available = (
        line_of_sight is not None
        and inside_fov is not None
        and smoke_occluded is not None
    )
    visual_contact = (
        line_of_sight and inside_fov and not smoke_occluded if available else None
    )
    return {
        "schema_id": "stratai.observed_visual_contact@1",
        "contract_version": "1.0.0",
        "identity_semantics": "join_only",
        "join_keys": {
            "match_id": SOURCE_ALIAS,
            "round_id": ROUND_ID,
            "observer_id": observer,
            "subject_id": subject,
            "contact_id": contact_id,
        },
        "round_number": 1,
        "tick": tick,
        "t0_tick": 108,
        "availability_tick": tick,
        "status": "observed" if available else "unavailable",
        "causal_role": "model_input_observation",
        "visibility_scope": "enemy_visual_contact_evaluation",
        "source": "replay_visual_contact_derivation",
        "line_of_sight": _boolean_fact(line_of_sight),
        "field_of_view": _fov_fact(inside_fov),
        "smoke_occlusion": _smoke_fact(smoke_occluded, active_smoke_count),
        "visual_contact": _boolean_fact(bool(visual_contact) if available else None),
        "boundary": boundary,
        "abstention_reasons": reasons or [],
        "provenance": {
            "source_artifact": "replay_data_in_memory",
            "source_schema_version": 16,
            "source_round": 1,
            "source_frame_tick": tick,
            "geometry_status": (
                "loaded" if line_of_sight is not None else "position_unavailable"
            ),
            "eye_position_source": "observer=native_eye_position;subject=native_eye_position",
            "raycast_method": "map_mesh_segment_raycast@1",
            "fov_method": "rectangular_frustum_16_9_from_source_4_3@1",
            "smoke_method": "segment_to_smoke_cylinder@1",
            "input_fields": list(INPUT_FIELDS),
        },
    }


def _valid_report(maps_root: Path) -> dict:
    first_observer, first_subject = "participant-001", "participant-002"
    contact_id = _contact_id()
    evaluations = [
        _evaluation(
            tick=100,
            observer=first_observer,
            subject=first_subject,
            line_of_sight=True,
            inside_fov=True,
            smoke_occluded=False,
            active_smoke_count=0,
            boundary="started",
            contact_id=contact_id,
        ),
        _evaluation(
            tick=100,
            observer="participant-003",
            subject="participant-004",
            line_of_sight=False,
            inside_fov=True,
            smoke_occluded=False,
            active_smoke_count=0,
            boundary="inactive",
            contact_id=None,
        ),
        _evaluation(
            tick=100,
            observer="participant-005",
            subject="participant-006",
            line_of_sight=True,
            inside_fov=False,
            smoke_occluded=False,
            active_smoke_count=0,
            boundary="inactive",
            contact_id=None,
        ),
        _evaluation(
            tick=100,
            observer="participant-007",
            subject="participant-008",
            line_of_sight=None,
            inside_fov=None,
            smoke_occluded=None,
            active_smoke_count=None,
            boundary="unavailable",
            contact_id=None,
            reasons=[
                "geometry_unavailable",
                "observer_orientation_unavailable",
                "smoke_snapshot_unavailable",
            ],
        ),
        _evaluation(
            tick=104,
            observer=first_observer,
            subject=first_subject,
            line_of_sight=True,
            inside_fov=True,
            smoke_occluded=False,
            active_smoke_count=0,
            boundary="continued",
            contact_id=contact_id,
        ),
        _evaluation(
            tick=108,
            observer=first_observer,
            subject=first_subject,
            line_of_sight=True,
            inside_fov=True,
            smoke_occluded=True,
            active_smoke_count=1,
            boundary="ended",
            contact_id=contact_id,
        ),
    ]
    episode = {
        "schema_id": "stratai.observed_visual_contact@1",
        "contract_version": "1.0.0",
        "identity_semantics": "join_only",
        "join_keys": {
            "match_id": SOURCE_ALIAS,
            "round_id": ROUND_ID,
            "observer_id": first_observer,
            "subject_id": first_subject,
            "contact_id": contact_id,
        },
        "round_number": 1,
        "t0_tick": 108,
        "start_tick": 100,
        "end_tick": 108,
        "last_confirmed_tick": 104,
        "start_availability_tick": 100,
        "end_availability_tick": 108,
        "status": "closed",
        "causal_role": "model_input_observation",
    }
    summary = {
        "evaluations": 6,
        "observed_evaluations": 5,
        "unavailable_evaluations": 1,
        "line_of_sight_clear": 4,
        "line_of_sight_blocked": 1,
        "line_of_sight_unavailable": 1,
        "fov_inside": 4,
        "fov_outside": 1,
        "fov_unavailable": 1,
        "smoke_occluded": 1,
        "smoke_clear": 4,
        "smoke_unavailable": 1,
        "visual_contact_true": 2,
        "visual_contact_false": 3,
        "visual_contact_unavailable": 1,
        "active_smoke_evaluations": 1,
        "boundaries": {
            "continued": 1,
            "ended": 1,
            "inactive": 2,
            "started": 1,
            "unavailable": 1,
        },
        "episodes": 1,
        "closed_episodes": 1,
        "open_episodes": 0,
        "participant_aliases": 8,
        "abstention_reasons": {
            "geometry_unavailable": 1,
            "observer_orientation_unavailable": 1,
            "smoke_snapshot_unavailable": 1,
        },
        "raw_steam_id_violations": 0,
    }
    report = {
        "schema_id": "stratai.observed_visual_contact_single_demo_smoke@1",
        "status": "passed",
        "source_alias": SOURCE_ALIAS,
        "source_sha256": SOURCE_SHA256,
        "map_name": "de_mirage",
        "candidate_schema_id": "stratai.observed_visual_contact@1",
        "candidate_contract_version": "1.0.0",
        "build_identifier": "test-build",
        "processing_duration_ms": 25,
        "selection": {
            "contract": "causal_round_prefix_with_smoke@1",
            "round_number": 1,
            "start_tick": 100,
            "end_tick": 108,
            "t0_tick": 108,
            "input_round_frames": 10,
            "selected_frames": 3,
            "frame_limit": 64,
        },
        "geometry_assets": [
            {
                "reference": "de_mirage/de_mirage_physics.bin",
                "sha256": "c" * 64,
                "role": "physics_buffer",
            },
            {
                "reference": "de_mirage/de_mirage_physics.gltf",
                "sha256": "b" * 64,
                "role": "physics_gltf",
            },
        ],
        "summary": summary,
        "ledger": {
            "schema_id": "stratai.observed_visual_contact@1",
            "contract_version": "1.0.0",
            "identity_semantics": "join_only",
            "causal_role": "model_input_observation",
            "join_keys": {"match_id": SOURCE_ALIAS},
            "t0_tick": 108,
            "sampling": {
                "target_hz": 16,
                "tick_rate": 64,
                "period_ticks": 4,
                "strategy": "round_anchored_right_closed_no_carry_forward",
            },
            "evaluations": evaluations,
            "contacts": [episode],
        },
    }
    _write_geometry_assets(report, maps_root)
    return report


def _reject(report: dict, message: str, *, maps_root: Path) -> None:
    with pytest.raises(VisualContactSmokeValidationError, match=message):
        validate_visual_contact_smoke_report(report, maps_root=maps_root)


def _write_geometry_assets(report: dict, maps_root: Path) -> None:
    contents = {
        "physics_buffer": b"temporary physics buffer\x00\x01",
        "physics_gltf": b'{"asset":{"version":"2.0"}}',
    }
    for asset in report["geometry_assets"]:
        asset_path = maps_root.joinpath(*asset["reference"].split("/"))
        asset_path.parent.mkdir(parents=True, exist_ok=True)
        content = contents[asset["role"]]
        asset_path.write_bytes(content)
        asset["sha256"] = hashlib.sha256(content).hexdigest()


def test_valid_visual_contact_smoke_recomputes_full_summary(tmp_path: Path) -> None:
    report = _valid_report(tmp_path)
    assert (
        validate_visual_contact_smoke_report(report, maps_root=tmp_path)
        == report["summary"]
    )


def test_public_validator_requires_a_real_maps_root(tmp_path: Path) -> None:
    report = _valid_report(tmp_path)
    with pytest.raises(TypeError, match="maps_root"):
        validate_visual_contact_smoke_report(report)  # type: ignore[call-arg]
    _reject(report, "expected pathlib.Path", maps_root=None)  # type: ignore[arg-type]


def test_report_selection_and_geometry_contracts_fail_closed(tmp_path: Path) -> None:
    unknown = _valid_report(tmp_path)
    unknown["unexpected"] = True
    _reject(unknown, "unknown", maps_root=tmp_path)

    checksum = _valid_report(tmp_path)
    checksum["source_alias"] = "demo-" + "b" * 20
    _reject(checksum, "alias/checksum", maps_root=tmp_path)

    future = _valid_report(tmp_path)
    future["selection"]["end_tick"] = 109
    _reject(future, "start_tick <= end_tick == t0_tick", maps_root=tmp_path)

    oversized = _valid_report(tmp_path)
    oversized["selection"]["selected_frames"] = 65
    _reject(oversized, "exceeds frame_limit", maps_root=tmp_path)

    too_few_frames = _valid_report(tmp_path)
    too_few_frames["selection"]["selected_frames"] = 2
    _reject(too_few_frames, "integer >= 3", maps_root=tmp_path)

    excessive_limit = _valid_report(tmp_path)
    excessive_limit["selection"]["frame_limit"] = 257
    _reject(excessive_limit, "between 16 and 256", maps_root=tmp_path)

    absolute_asset = _valid_report(tmp_path)
    absolute_asset["geometry_assets"][0]["reference"] = "C:/maps/de_mirage.gltf"
    _reject(absolute_asset, "relative reference", maps_root=tmp_path)

    missing_role = _valid_report(tmp_path)
    missing_role["geometry_assets"] = missing_role["geometry_assets"][:1]
    _reject(missing_role, "physics_gltf and physics_buffer", maps_root=tmp_path)

    duplicate_role = _valid_report(tmp_path)
    duplicate_role["geometry_assets"][1]["role"] = "physics_buffer"
    _reject(duplicate_role, "duplicate geometry role", maps_root=tmp_path)

    noncanonical_order = _valid_report(tmp_path)
    noncanonical_order["geometry_assets"].reverse()
    _reject(noncanonical_order, "canonical role/reference order", maps_root=tmp_path)


def test_formula_and_causal_availability_are_independently_validated(
    tmp_path: Path,
) -> None:
    formula = _valid_report(tmp_path)
    formula["ledger"]["evaluations"][-1]["visual_contact"]["value"] = True
    _reject(formula, "LOS && FOV && !smoke", maps_root=tmp_path)

    availability = _valid_report(tmp_path)
    availability["ledger"]["evaluations"][0]["availability_tick"] = 101
    _reject(availability, "availability_tick <= tick <= T0", maps_root=tmp_path)

    future_source = _valid_report(tmp_path)
    future_source["ledger"]["evaluations"][0]["provenance"]["source_frame_tick"] = 101
    _reject(future_source, "causal selected window", maps_root=tmp_path)

    sampling = _valid_report(tmp_path)
    sampling["ledger"]["sampling"]["period_ticks"] = 5
    _reject(sampling, "tick_rate / 16", maps_root=tmp_path)

    impossible_smoke = _valid_report(tmp_path)
    impossible_smoke["ledger"]["evaluations"][-1]["smoke_occlusion"][
        "active_smoke_count"
    ] = 0
    _reject(
        impossible_smoke,
        "occluded=true requires active_smoke_count > 0",
        maps_root=tmp_path,
    )


def test_provenance_join_keys_and_raw_steam_ids_are_rejected(tmp_path: Path) -> None:
    method = _valid_report(tmp_path)
    method["ledger"]["evaluations"][0]["provenance"]["raycast_method"] = "unknown"
    _reject(method, "raycast_method", maps_root=tmp_path)

    geometry_not_loaded = _valid_report(tmp_path)
    geometry_not_loaded["ledger"]["evaluations"][3]["provenance"]["geometry_status"] = (
        "not_loaded"
    )
    _reject(
        geometry_not_loaded,
        "global smoke report requires loaded geometry",
        maps_root=tmp_path,
    )

    input_fields = _valid_report(tmp_path)
    input_fields["ledger"]["evaluations"][0]["provenance"]["input_fields"].append(
        "oracle_future"
    )
    _reject(input_fields, "input_fields", maps_root=tmp_path)

    mixed_match = _valid_report(tmp_path)
    mixed_match["ledger"]["evaluations"][0]["join_keys"]["match_id"] = (
        "demo-" + "b" * 20
    )
    _reject(mixed_match, "source_alias", maps_root=tmp_path)

    raw_steam = _valid_report(tmp_path)
    raw_steam["ledger"]["evaluations"][1]["join_keys"]["observer_id"] = (
        "steam:76561198000000001"
    )
    _reject(raw_steam, "raw Steam ID", maps_root=tmp_path)

    zero_alias = _valid_report(tmp_path)
    zero_alias["ledger"]["evaluations"][1]["join_keys"]["observer_id"] = (
        "participant-000"
    )
    _reject(zero_alias, "canonical local participant alias", maps_root=tmp_path)


def test_boundaries_and_episodes_must_form_one_consistent_state_machine(
    tmp_path: Path,
) -> None:
    legacy_contact_id = _valid_report(tmp_path)
    legacy_contact_id["ledger"]["evaluations"][4]["join_keys"]["contact_id"] = (
        "contact:001:participant-001:participant-002:100"
    )
    _reject(legacy_contact_id, "pseudonymised contact ID", maps_root=tmp_path)

    zero_contact_id = _valid_report(tmp_path)
    zero_contact_id["ledger"]["evaluations"][4]["join_keys"]["contact_id"] = (
        "contact-000"
    )
    _reject(zero_contact_id, "canonical pseudonymised contact ID", maps_root=tmp_path)

    changed_id = _valid_report(tmp_path)
    changed_id["ledger"]["evaluations"][4]["join_keys"]["contact_id"] = "contact-099"
    _reject(changed_id, "contact_id", maps_root=tmp_path)

    wrong_end = _valid_report(tmp_path)
    wrong_end["ledger"]["contacts"][0]["end_tick"] = 107
    wrong_end["ledger"]["contacts"][0]["end_availability_tick"] = 107
    _reject(wrong_end, "episode end differs", maps_root=tmp_path)

    inactive_with_id = _valid_report(tmp_path)
    inactive_with_id["ledger"]["evaluations"][1]["join_keys"]["contact_id"] = (
        _contact_id()
    )
    _reject(
        inactive_with_id,
        "inactive boundary cannot carry contact_id",
        maps_root=tmp_path,
    )

    missing_episode = _valid_report(tmp_path)
    missing_episode["ledger"]["contacts"] = []
    _reject(missing_episode, "contact_id has no episode", maps_root=tmp_path)


def test_summary_tampering_and_missing_real_coverage_are_rejected(
    tmp_path: Path,
) -> None:
    tampered = _valid_report(tmp_path)
    tampered["summary"]["observed_evaluations"] = 4
    _reject(tampered, "recomputed summary", maps_root=tmp_path)

    no_blocked_los = copy.deepcopy(_valid_report(tmp_path))
    row = no_blocked_los["ledger"]["evaluations"][1]
    row["line_of_sight"] = _boolean_fact(True)
    row["field_of_view"] = _fov_fact(False)
    no_blocked_los["summary"]["line_of_sight_clear"] = 5
    no_blocked_los["summary"]["line_of_sight_blocked"] = 0
    no_blocked_los["summary"]["fov_inside"] = 3
    no_blocked_los["summary"]["fov_outside"] = 2
    _reject(no_blocked_los, "line_of_sight_blocked", maps_root=tmp_path)


def test_geometry_assets_are_resolved_and_hashed_from_maps_root(tmp_path: Path) -> None:
    report = _valid_report(tmp_path)
    assert (
        validate_visual_contact_smoke_report(report, maps_root=tmp_path)
        == report["summary"]
    )

    buffer_path = tmp_path / report["geometry_assets"][0]["reference"]
    buffer_path.write_bytes(b"tampered physics buffer")
    _reject(report, "geometry asset hash mismatch", maps_root=tmp_path)


def test_cli_requires_maps_root_and_rejects_tampered_asset(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
    capsys: pytest.CaptureFixture[str],
) -> None:
    report = _valid_report(tmp_path)
    report_path = tmp_path / "observed_visual_contact_smoke.json"
    report_path.write_text(json.dumps(report), encoding="utf-8")

    monkeypatch.setattr(
        sys, "argv", ["validate_visual_contact_smoke", str(report_path)]
    )
    with pytest.raises(SystemExit, match="2"):
        main()
    capsys.readouterr()

    gltf_path = tmp_path / report["geometry_assets"][1]["reference"]
    gltf_path.write_bytes(b"tampered gltf")
    monkeypatch.setattr(
        sys,
        "argv",
        [
            "validate_visual_contact_smoke",
            str(report_path),
            "--maps-root",
            str(tmp_path),
        ],
    )
    assert main() == 1
    output = json.loads(capsys.readouterr().out)
    assert output["ok"] is False
    assert "geometry asset hash mismatch" in output["error"]
