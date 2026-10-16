from __future__ import annotations

import copy
import gzip
import hashlib
import json
from types import SimpleNamespace

import pytest

from ai_coach.contracts.spatial_candidate import (
    BRIDGE_ID,
    FIELDS,
    KEY_CONTRACT,
    SpatialCandidateError,
    compact,
    decode,
    scoped_key,
    sha256,
    validate_candidate_example,
)
from ai_coach.datasets.build_examples import process_manifest
from ai_coach.datasets.spatial_candidate import (
    JOIN_FILENAME,
    ROOT,
    audit_spatial_candidate,
    build_joins,
    fact_status,
    index_spatial_rows,
    load_spatial_candidate,
    materialize_spatial_candidate,
    read_rows,
    temporary_root,
)
from ai_coach.datasets.streaming import load_direct_bundle_manifest
from ai_coach.tests import test_cover_full_demo_validation as cover_fixture
from ai_coach.tests import test_exposure_angle_full_demo_validation as exposure_fixture
from ai_coach.tests.test_decision_examples import (
    DIRECT_ALIAS,
    DIRECT_CHECKSUM,
    _direct_bundle,
)

MAPS = ROOT / "backend/data/maps"


def exit_row() -> dict:
    path = (
        ROOT
        / "ai_coach/spatial/exit_route/packages/de_mirage/1.0.0-candidate.1/package.json"
    )
    package = decode(path.read_bytes())
    sample = package["route_samples"][0]
    target = sample["candidates"][0]
    physical = target["objective_connectivity"]
    unavailable = {
        "status": "unavailable",
        "value_milliseconds": None,
        "speed_world_units_per_second": None,
        "speed_source": "unavailable",
        "assumption": "unavailable",
        "unavailability_reason": "motion_unavailable",
    }
    scenarios = {
        "optimistic_uninterrupted_250ups_baseline": {
            "status": "estimated",
            "value_milliseconds": physical["path_distance_world_units"] / 250 * 1000,
            "speed_world_units_per_second": 250,
            "speed_source": "reference",
            "assumption": "continuous",
            "unavailability_reason": None,
        },
        "active_weapon_uninterrupted_baseline": copy.deepcopy(unavailable),
        "observed_pace_projection": copy.deepcopy(unavailable),
    }
    return {
        "schema_id": "stratai.observed_exit_route_diagnostic@2",
        "contract_version": "2.0.0",
        "source_alias": DIRECT_ALIAS,
        "map_id": "de_mirage",
        "round_number": 1,
        "t0_tick": 100,
        "availability_tick": 100,
        "participant_alias": "participant-001",
        "training_allowed": False,
        "product_supported": False,
        "position_world": sample["origin_world"],
        "posture_context": {
            "availability_tick": None,
            "is_ducking": None,
            "is_walking": None,
            "is_scoped": None,
            "active_weapon_status": "unavailable",
            "active_weapon": None,
            "horizontal_speed_status": "unavailable",
            "horizontal_speed_world_units_per_second": None,
            "horizontal_speed_source": None,
        },
        "sample": {
            "status": "observed",
            "reason_codes": [],
            "sample_id": sample["id"],
            "sample_alias": sample["alias"],
        },
        "candidates": [
            {
                "candidate_id": target["id"],
                "candidate_alias": target["alias"],
                "destination_world": target["destination_world"],
                "objective_viability": {
                    "field_id": "FLD-NAV-EXIT-PATH",
                    "direction": "origin_to_destination",
                    "reason_codes": [],
                    "reachability": {"status": "derived", "value": True},
                    "origin_association_world_units": physical[
                        "origin_association_world_units"
                    ],
                    "destination_association_world_units": physical[
                        "destination_association_world_units"
                    ],
                },
                "physical_cost": {
                    "navigation_schema_id": "stratai.observed_navigation_path@2",
                    "navigation_contract_version": "2.0.0",
                    "direct_distance": {
                        "status": "derived",
                        "value_world_units": physical["direct_distance_world_units"],
                    },
                    "path_distance": {
                        "status": "derived",
                        "value_world_units": physical["path_distance_world_units"],
                    },
                    "internal_timing": {
                        "intended_use": "internal_decision_support_only",
                        "user_facing_mode": "qualitative_conditional_with_uncertainty_only",
                        "decision_semantics": "constraint_signal_not_recommendation",
                        "estimate_semantics": "scenario_reference_not_arrival_prediction",
                        "exact_user_facing_time_allowed": False,
                        "arrival_prediction": False,
                        "scenarios": scenarios,
                    },
                },
                "tactical_assessment": {
                    "status": "unavailable",
                    "reason_codes": ["tactical_decision_context_required"],
                },
            }
        ],
        "provenance": {
            "position_source": "replay_player_state_position_16hz",
            "posture_source": "replay_player_state_causal_context_16hz",
            "motion_source": "observed_navigation_path_at_t0",
            "package_schema_id": "stratai.exit_route_map_package@1",
            "package_contract_version": "1.0.0",
            "package_asset_version": package["asset_version"],
            "package_sha256": sha256(path),
            "geometry_revision": package["geometry_revision"],
            "nav_asset_sha256": next(
                item["sha256"]
                for item in package["source_assets"]
                if item["role"] == "nav_mesh"
            ),
            "input_fields": ["position"],
        },
    }


def rows_for_fields() -> dict:
    exposure = exposure_fixture.load_package(
        exposure_fixture.REGISTRY_ROOT, MAPS, "de_mirage"
    )
    rows = {
        "FLD-GEOMETRIC-CLEARANCE-SECTORS": exposure_fixture._base_rows(
            exposure.package, exposure.package_sha256
        )[0],
        "FLD-GEOMETRIC-OCCLUSION": cover_fixture._observed_row(
            cover_fixture._package()
        ),
        "FLD-NAV-EXIT-PATH": exit_row(),
    }
    for row in rows.values():
        row.update(source_alias=DIRECT_ALIAS, t0_tick=100, availability_tick=100)
    return rows


@pytest.mark.parametrize("field_id", FIELDS)
def test_exact_join_before_equal_after_and_future_mutation(field_id):
    row = rows_for_fields()[field_id]
    package = {
        "package_sha256": row["provenance"]["package_sha256"],
        "geometry_revision": row["provenance"]["geometry_revision"],
        "contract_version": "1.0.0",
        "asset_version": row.get(
            "asset_version", row["provenance"].get("package_asset_version")
        ),
    }
    index = index_spatial_rows(
        [row],
        field_id=field_id,
        source_alias=DIRECT_ALIAS,
        map_id="de_mirage",
        package=package,
        aliases={"participant-001"},
    )
    indexes = {field: index if field == field_id else {} for field in FIELDS}
    bridge = {scoped_key(DIRECT_CHECKSUM, "steam:1"): "participant-001"}

    def decision(tick):
        return SimpleNamespace(
            value={
                "actor_player_id": "steam:1",
                "decision_id": "decision",
                "match_id": DIRECT_ALIAS,
                "round_number": 1,
                "t0_tick": tick,
                "decision_type": "peek_hold_or_reposition",
            }
        )

    for tick, expected in [(99, "missing"), (100, "present"), (101, "missing")]:
        joins = build_joins(
            {"decision": decision(tick)}, indexes, bridge, DIRECT_CHECKSUM
        )
        assert (
            next(item for item in joins if item["field_id"] == field_id)["status"]
            == expected
        )
    baseline = build_joins(
        {"decision": decision(100)}, indexes, bridge, DIRECT_CHECKSUM
    )
    future = copy.deepcopy(row)
    future.update(
        t0_tick=101, availability_tick=101, position_world={"x": 999999, "y": 0, "z": 0}
    )
    indexes[field_id] = index_spatial_rows(
        [row, future],
        field_id=field_id,
        source_alias=DIRECT_ALIAS,
        map_id="de_mirage",
        package=package,
        aliases={"participant-001"},
    )
    assert (
        build_joins({"decision": decision(100)}, indexes, bridge, DIRECT_CHECKSUM)
        == baseline
    )


@pytest.mark.parametrize("field_id", FIELDS)
def test_false_facts_are_present_and_physical_absences_are_null(field_id):
    row = rows_for_fields()[field_id]
    if field_id == "FLD-GEOMETRIC-CLEARANCE-SECTORS":
        fact = row["geometry_clearance"]
        fact["view_inside_geometry_sector"] = False
    elif field_id == "FLD-GEOMETRIC-OCCLUSION":
        fact = row["geometry_occlusion"]
        for direction in fact["directions"]:
            direction["complete_volume_occlusion"] = False
    else:
        fact = row["candidates"][0]["objective_viability"]
        fact["reachability"]["value"] = False
    assert fact_status(field_id, row)[0] == "present"
    if field_id == "FLD-NAV-EXIT-PATH":
        fact["reachability"] = {"status": "unavailable", "value": None}
        fact["reason_codes"] = ["nav_graph_unavailable"]
    else:
        fact["status"] = "unavailable"
        fact["reason_codes"] = ["outside_declared_sample_coverage"]
    status, reasons = fact_status(field_id, row)
    assert status == "null" and reasons


@pytest.mark.parametrize("field_id", FIELDS)
@pytest.mark.parametrize(
    "mutation",
    ["map", "source", "hash", "asset", "provenance", "temporal", "duplicate", "order"],
)
def test_spatial_index_rejects_mismatch_duplicates_and_order(field_id, mutation):
    row = rows_for_fields()[field_id]
    package = {
        "package_sha256": row["provenance"]["package_sha256"],
        "geometry_revision": row["provenance"]["geometry_revision"],
        "contract_version": "1.0.0",
        "asset_version": row.get(
            "asset_version", row["provenance"].get("package_asset_version")
        ),
    }
    rows = [copy.deepcopy(row)]
    if mutation == "map":
        rows[0]["map_id"] = "de_other"
    if mutation == "source":
        rows[0]["source_alias"] = "demo-" + "f" * 20
    if mutation == "hash":
        rows[0]["provenance"]["package_sha256"] = "f" * 64
    if mutation == "asset":
        package["asset_version"] = "invalid"
    if mutation == "provenance":
        rows[0]["provenance"]["position_source"] = "oracle"
    if mutation == "temporal":
        rows[0]["availability_tick"] += 1
    if mutation == "duplicate":
        rows.append(copy.deepcopy(row))
    if mutation == "order":
        rows.append(copy.deepcopy(row))
        rows[0].update(t0_tick=101, availability_tick=101)
    with pytest.raises(SpatialCandidateError):
        index_spatial_rows(
            rows,
            field_id=field_id,
            source_alias=DIRECT_ALIAS,
            map_id="de_mirage",
            package=package,
            aliases={"participant-001"},
        )


def _rewrite_report(report_path, rows):
    report = decode(report_path.read_bytes())
    report.update(
        source_alias=DIRECT_ALIAS,
        source_sha256=DIRECT_CHECKSUM,
        build_identifier="fixture@1",
    )
    for row in rows:
        row["source_alias"] = DIRECT_ALIAS
    payload = b"".join(
        json.dumps(row, ensure_ascii=False, separators=(",", ":")).encode() + b"\n"
        for row in rows
    )
    path = report_path.parent / report["rows_artifact"]["path"]
    path.write_bytes(gzip.compress(payload, mtime=0))
    report["rows_artifact"].update(
        bytes=path.stat().st_size,
        sha256=sha256(path),
        logical_bytes=len(payload),
        logical_sha256=hashlib.sha256(payload).hexdigest(),
    )
    report_path.write_bytes(compact(report))
    return report


def _source_fixture(root):
    base = _direct_bundle(root)
    all_rows = []
    exposure_path = exposure_fixture._write_fixture(root)
    cover_path = cover_fixture._write_fixture(root)
    for path in (exposure_path, cover_path):
        report = decode(path.read_bytes())
        rows = list(read_rows(root / report["rows_artifact"]["path"], compressed=True))
        _rewrite_report(path, rows)
        all_rows.extend(rows)
    row = exit_row()
    package = decode(
        (
            ROOT
            / "ai_coach/spatial/exit_route/packages/de_mirage/1.0.0-candidate.1/package.json"
        ).read_bytes()
    )
    report = {
        "schema_id": "stratai.observed_exit_route_full_demo_report@2",
        "contract_version": "2.0.0",
        "candidate_schema_id": row["schema_id"],
        "navigation_schema_id": "stratai.observed_navigation_path@2",
        "navigation_contract_version": "2.0.0",
        "source_alias": DIRECT_ALIAS,
        "source_sha256": DIRECT_CHECKSUM,
        "map_id": "de_mirage",
        "build_identifier": "fixture@1",
        "training_allowed": False,
        "product_supported": False,
        "canonical_bundle_version": "3.8.0_unmodified_sidecar_only",
        "package_resolution": {
            "status": "available",
            "reason_codes": [],
            "contract_version": "1.0.0",
            "asset_version": package["asset_version"],
            "package_sha256": row["provenance"]["package_sha256"],
            "geometry_revision": package["geometry_revision"],
            "product_supported": False,
        },
        "navigation_contract": package["navigation_contract"],
        "sampling": {
            "target_hz": 1,
            "tick_rate_hz": 64,
            "period_ticks": 64,
            "strategy": "round_anchored_right_closed_alive_players",
        },
        "rows_artifact": {
            "path": "observed_exit_route_diagnostic.jsonl.gz",
            "logical_records": 1,
        },
        "coverage": {
            "rounds": 1,
            "records": 1,
            "samples_observed": 1,
            "samples_unavailable": 0,
            "candidate_evaluations": 1,
            "reachable": 1,
            "unreachable": 0,
            "reachability_unavailable": 0,
            "physical_path_available": 1,
            "physical_path_unavailable": 0,
            "active_weapon_scenario_available": 0,
            "active_weapon_scenario_unavailable": 1,
            "observed_pace_scenario_available": 0,
            "observed_pace_scenario_unavailable": 1,
            "tactical_judgment_unavailable": 1,
            "samples_hit": {"exit-sample-001": 1},
            "sample_absence_reason_counts": {},
            "objective_unavailability_reason_counts": {},
            "tactical_judgment_reason_counts": {
                "tactical_decision_context_required": 1
            },
        },
    }
    path = root / "observed_exit_route_full_demo.json"
    path.write_bytes(compact(report))
    _rewrite_report(path, [row])
    all_rows.append(row)
    manifest_path = base / "canonical/manifest.json"
    manifest = decode(manifest_path.read_bytes())
    for entry in manifest["artifacts"]:
        path = base / "canonical" / entry["path"]
        payload = (
            gzip.decompress(path.read_bytes())
            if entry.get("compression") == "gzip"
            else path.read_bytes()
        )
        payload = payload.replace(b"actor-1", b"steam:1")
        path.write_bytes(
            gzip.compress(payload, mtime=0)
            if entry.get("compression") == "gzip"
            else payload
        )

    def add(kind, relative, schema, value, compressed=False):
        path = base / "canonical" / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        payload = compact(value)
        path.write_bytes(gzip.compress(payload, mtime=0) if compressed else payload)
        manifest["artifacts"].append(
            {
                "artifact_type": kind,
                "path": relative,
                "schema_id": schema,
                "format": "json",
                "compression": "gzip" if compressed else "",
                "record_count": 1,
                "sort_order": [],
                "bytes": 1,
                "sha256": "",
            }
        )

    add("match", "core/match.json", "stratai.match@1", {"map_name": "de_mirage"})
    add(
        "participants",
        "core/participants.json",
        "stratai.participants@1",
        {"players": [{"player_id": "steam:1"}, {"player_id": "steam:2"}]},
    )
    players = {}
    for row in all_rows:
        key = (row["t0_tick"], row["participant_alias"])
        if key in players:
            continue
        players[key] = {
            "steam_id": str(int(row["participant_alias"][-3:])),
            "alive": True,
            **row["position_world"],
            "yaw": row.get("view_yaw_degrees", 0),
            "is_ducking": row.get("posture_id") == "crouching",
        }
    frames = [
        {
            "tick": tick,
            "players": [player for (t, _), player in players.items() if t == tick],
        }
        for tick in sorted({key[0] for key in players})
    ]
    add(
        "replay_round",
        "presentation/replay/round_001.json.gz",
        "stratai.replay_round@5",
        {"match_id": DIRECT_ALIAS, "round": {"round": 1, "frames": frames}},
        True,
    )
    for entry in manifest["artifacts"]:
        path = base / "canonical" / entry["path"]
        entry.update(bytes=path.stat().st_size, sha256=sha256(path))
    manifest["artifacts"].sort(key=lambda entry: entry["path"])
    manifest_path.write_bytes(compact(manifest))
    (root / "spatial_participant_bridge.json").write_bytes(
        compact(
            {
                "schema_id": BRIDGE_ID,
                "key_contract": KEY_CONTRACT,
                "source_alias": DIRECT_ALIAS,
                "source_sha256": DIRECT_CHECKSUM,
                "map_id": "de_mirage",
                "training_allowed": False,
                "participants": [
                    {
                        "actor_key": scoped_key(DIRECT_CHECKSUM, "steam:" + str(i)),
                        "participant_alias": f"participant-{i:03d}",
                    }
                    for i in (1, 2)
                ],
            }
        )
    )
    return base


def _materialize(root):
    return materialize_spatial_candidate(
        root,
        source_alias=DIRECT_ALIAS,
        source_sha256=DIRECT_CHECKSUM,
        map_id="de_mirage",
        maps_root=MAPS,
    )


def test_candidate_end_to_end_compatibility_and_eligibility(tmp_path):
    base = _source_fixture(tmp_path)
    before = {
        p.relative_to(base).as_posix(): sha256(p)
        for p in base.rglob("*")
        if p.is_file()
    }
    path = _materialize(tmp_path)
    candidate = load_spatial_candidate(path, maps_root=MAPS)
    examples = []
    audit = audit_spatial_candidate(candidate, example_sink=examples.append)
    assert audit["family_counts"] == {
        "peek_hold_or_reposition": 1,
        "spacing_or_trade_connection": 1,
    }
    for family in audit["spatial_by_family"].values():
        assert family["FLD-GEOMETRIC-CLEARANCE-SECTORS"]["present"] == 1
        assert family["FLD-NAV-EXIT-PATH"]["present"] == 1
        assert family["FLD-GEOMETRIC-OCCLUSION"]["missing"] == 1
    peek = next(
        item
        for item in audit["task_eligibility"]
        if item["capability_id"] == "CAP-PEEK-001"
    )
    assert "FLD-GEOMETRIC-CLEARANCE-SECTORS" not in peek["missing_required_fields"]
    assert "FLD-GEOMETRIC-OCCLUSION" in peek["missing_required_fields"]
    for example in examples:
        validate_candidate_example(example)
        assert "tactical_assessment" not in json.dumps(example)
        assert "steam:" not in json.dumps(example)
        for forbidden in ("outcome", "oracle", "responses"):
            assert forbidden not in json.dumps(example["observed"])
    legacy = load_direct_bundle_manifest(
        base, expected_match_id=DIRECT_ALIAS, expected_demo_checksum=DIRECT_CHECKSUM
    )
    result = process_manifest(legacy, catalog_required_fields=list(FIELDS))
    assert all(
        metric.missing == 1
        for families in result.field_coverage.values()
        for metric in families.values()
    )
    assert {
        p.relative_to(base).as_posix(): sha256(p)
        for p in base.rglob("*")
        if p.is_file()
    } == before
    with pytest.raises(SpatialCandidateError, match="overwrites"):
        _materialize(tmp_path)


@pytest.mark.parametrize(
    "mutation",
    [
        "map",
        "base_hash",
        "row_hash",
        "package",
        "source",
        "join_duplicate",
        "join_missing",
        "join_order",
        "join_future",
        "example",
        "bridge",
    ],
)
def test_candidate_loader_fails_closed_for_tampering(tmp_path, mutation):
    _source_fixture(tmp_path)
    path = _materialize(tmp_path)
    manifest = decode(path.read_bytes())
    if mutation == "map":
        manifest["map_id"] = "de_other"
    if mutation == "base_hash":
        manifest["base_manifest"]["sha256"] = "0" * 64
    if mutation == "row_hash":
        manifest["facts"][next(iter(FIELDS))]["rows"]["sha256"] = "0" * 64
    if mutation == "package":
        manifest["facts"][next(iter(FIELDS))]["package"]["asset_version"] = "wrong"
    if mutation == "source":
        manifest["source_sha256"] = "b" * 64
    if mutation.startswith("join_"):
        joins = list(read_rows(tmp_path / JOIN_FILENAME))
        if mutation == "join_duplicate":
            joins.append(joins[-1])
        if mutation == "join_missing":
            joins.pop()
        if mutation == "join_order":
            joins.reverse()
        if mutation == "join_future":
            joins[0]["sample_tick"] = 101
        (tmp_path / JOIN_FILENAME).write_bytes(b"".join(compact(row) for row in joins))
        manifest["joins"].update(
            bytes=(tmp_path / JOIN_FILENAME).stat().st_size,
            sha256=sha256(tmp_path / JOIN_FILENAME),
            record_count=len(joins),
        )
    if mutation in {"example", "bridge"}:
        role = "decision_examples" if mutation == "example" else "participant_bridge"
        (tmp_path / manifest[role]["path"]).write_bytes(b"tampered")
    path.write_bytes(compact(manifest))
    with pytest.raises((SpatialCandidateError, ValueError)):
        load_spatial_candidate(path, maps_root=MAPS)


def test_schema_metadata_and_non_temporary_write_guard():
    for name in [
        "spatial_candidate_artifact",
        "spatial_participant_bridge",
        "spatial_candidate_manifest",
        "spatial_fact_join",
        "decision_example_v2",
    ]:
        from ai_coach.contracts.spatial_candidate import schema_validator

        schema_validator(name + ".schema.json")
    with pytest.raises(SpatialCandidateError, match="TEMP"):
        temporary_root(ROOT)
    with pytest.raises(SpatialCandidateError, match="duplicate JSON"):
        decode('{"status":1,"status":2}')


def test_oracle_outcome_mutation_invariance_and_example_safety(tmp_path):
    outputs = []
    for mutate in (False, True):
        root = tmp_path / str(mutate)
        root.mkdir()
        base = _source_fixture(root)
        if mutate:
            path = base / "canonical/manifest.json"
            manifest = decode(path.read_bytes())
            for entry in manifest["artifacts"]:
                if entry["artifact_type"] not in {
                    "oracle_context",
                    "decision_outcomes",
                }:
                    continue
                artifact_path = base / "canonical" / entry["path"]
                rows = list(read_rows(artifact_path))
                for row in rows:
                    if entry["artifact_type"] == "oracle_context":
                        row.update(
                            available=True,
                            status="available",
                            hidden_enemy_position=[999, 999],
                        )
                    else:
                        row["outcome"] = "opposite-future"
                artifact_path.write_bytes(b"".join(compact(row) for row in rows))
                entry.update(
                    bytes=artifact_path.stat().st_size, sha256=sha256(artifact_path)
                )
            path.write_bytes(compact(manifest))
        candidate = load_spatial_candidate(_materialize(root), maps_root=MAPS)
        examples = []
        audit_spatial_candidate(candidate, example_sink=examples.append)
        outputs.append(examples)
    assert [example["observed"] for example in outputs[0]] == [
        example["observed"] for example in outputs[1]
    ]
    for forbidden in (
        "tactical_assessment",
        "responses",
        "oracle",
        "outcome",
        "steam_id",
    ):
        example = copy.deepcopy(outputs[0][0])
        example["observed"]["context"]["values"][forbidden] = "injected"
        with pytest.raises(ValueError):
            validate_candidate_example(example)


def test_source_reconciliation_rejects_recomputed_facts_from_wrong_frame(tmp_path):
    base = _source_fixture(tmp_path)
    path = base / "canonical/manifest.json"
    manifest = decode(path.read_bytes())
    entry = next(
        item
        for item in manifest["artifacts"]
        if item["artifact_type"] == "replay_round"
    )
    replay_path = base / "canonical" / entry["path"]
    replay = decode(gzip.decompress(replay_path.read_bytes()))
    replay["round"]["frames"][0]["players"][0]["x"] += 1
    replay_path.write_bytes(gzip.compress(compact(replay), mtime=0))
    entry.update(bytes=replay_path.stat().st_size, sha256=sha256(replay_path))
    path.write_bytes(compact(manifest))
    with pytest.raises(SpatialCandidateError, match="position/source mismatch"):
        _materialize(tmp_path)
