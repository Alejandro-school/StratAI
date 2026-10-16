from __future__ import annotations

import hashlib
import json
from datetime import datetime, timezone
from pathlib import Path

import pytest
from jsonschema import Draft202012Validator

from ai_coach.spatial.review import build_contextual_review_forms as forms
from ai_coach.spatial.review.build_contextual_pilot import (
    ContextualPilotError,
    _attach_and_validate_traces,
)

ROOT = Path(__file__).resolve().parents[2]
PROTOCOL_ROOT = ROOT / "ai_coach" / "spatial" / "review" / "protocols" / "2.0.0"


def test_contextual_protocol_is_frozen_and_requires_visible_context() -> None:
    protocol = json.loads(
        (PROTOCOL_ROOT / "protocol.json").read_text(encoding="utf-8")
    )
    schema = json.loads(
        (
            ROOT
            / "ai_coach"
            / "contracts"
            / "spatial_context_review_protocol.schema.json"
        ).read_text(encoding="utf-8")
    )
    Draft202012Validator(schema).validate(protocol)
    handbook_hash = hashlib.sha256(
        (PROTOCOL_ROOT / "HANDBOOK.md").read_bytes()
    ).hexdigest()
    assert protocol["handbook"]["sha256"] == handbook_hash
    assert protocol["evidence_gate"] == {
        "observed_demo_context_required": True,
        "radar_registration_required": True,
        "route_polyline_required": True,
        "future_or_outcome_allowed": False,
        "unreviewable_units_delivered": False,
    }
    assert protocol["authorization"]["review_execution_authorized"] is True
    assert set(protocol["authorization"].values()) == {False, True}


def test_route_trace_attachment_requires_full_matching_polyline() -> None:
    selected = []
    routes = []
    for index in range(4):
        origin = {"x": float(index), "y": 2.0, "z": 3.0}
        destination = {"x": 10.0 + index, "y": 12.0, "z": 3.0}
        connectivity = {
            "direct_distance_world_units": 10.0,
            "path_distance_world_units": 14.0,
            "origin_association_world_units": 1.0,
            "destination_association_world_units": 2.0,
        }
        trace_id = f"trace-{index:024x}"
        selected.append(
            {
                "trace_id": trace_id,
                "reviewer_view": {
                    "objective_geometry": {
                        "origin_world": origin,
                        "destination_world": destination,
                        "objective_connectivity": connectivity,
                    }
                },
            }
        )
        routes.append(
            {
                "trace_id": trace_id,
                "measurement": connectivity,
                "path_points_world": [
                    origin,
                    {"x": 5.0 + index, "y": 7.0, "z": 3.0},
                    destination,
                ],
            }
        )
    payload = {
        "schema_id": "stratai.spatial_review_route_traces@1",
        "map_id": "de_mirage",
        "graph_contract_version": "nav_area_connections_with_polygons@2",
        "shortest_path_algorithm": "directed_dijkstra_stable_ties@1",
        "routes": routes,
    }
    _attach_and_validate_traces(selected, payload)
    assert all(
        item["reviewer_view"]["objective_geometry"]["trace_contract"]
        == "nav_area_centres_polyline@1"
        for item in selected
    )

    broken = json.loads(json.dumps(payload))
    broken["routes"][0]["path_points_world"] = []
    with pytest.raises(ContextualPilotError, match="polyline_missing"):
        _attach_and_validate_traces(selected, broken)


def test_contextual_forms_embed_radar_and_only_reviewer_partition(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    units = [
        {
            "review_unit_id": f"scu-{index:024x}",
            "display_order": index + 1,
            "reviewer_view": {
                "case_kind": "exposure",
                "observed_context": {
                    "status": "observed",
                    "participant_alias": "participant-001",
                },
                "objective_geometry": {"evidence_type": "situated"},
            },
            "visible_evidence_sha256": f"{index:064x}",
        }
        for index in range(12)
    ]
    radar = forms.RADAR_PATH.read_bytes()
    batch = {
        "batch_id": "spatial-context-pilot-" + "a" * 24,
        "protocol": {"protocol_version": "2.0.0", "handbook_version": "2.0.0"},
        "presentation": {
            "map_label": "Mirage",
            "radar_asset": {"sha256": hashlib.sha256(radar).hexdigest()},
            "coordinate_transform": {
                "pos_x": -3230,
                "pos_y": 1713,
                "scale": 5,
                "logical_pixels": 1024,
            },
            "render_contract": "mirage_radar_context_overlay@2",
        },
        "authorization": {
            "review_execution_authorized": True,
            "training_use_authorized": False,
            "product_supported": False,
            "canonical_materialization": False,
            "release_authorized": False,
            "corpus_reprocessing_authorized": False,
        },
        "review_units": [
            {
                **unit,
                "audit_context": {"source_join": "must-not-leak"},
                "review_responses": [],
            }
            for unit in units
        ],
    }
    batch_path = tmp_path / "batch.json"
    batch_path.write_text(json.dumps(batch), encoding="utf-8")
    destination = tmp_path / "forms"
    destination.mkdir()
    monkeypatch.setattr(
        forms,
        "validate_spatial_context_review_pilot",
        lambda *_args, **_kwargs: {"review_units": 12, "responses": 0},
    )

    result = forms.build_contextual_review_forms(
        batch_path,
        destination,
        created_at=datetime(2026, 9, 2, 18, 0, tzinfo=timezone.utc),
    )
    assert result["review_units"] == 12
    for path in result["reviewer_forms"].values():
        html = path.read_text(encoding="utf-8")
        assert "data:image/webp;base64," in html
        assert "participant-001" in html
        assert "must-not-leak" not in html
        assert '"audit_context"' not in html
        assert "fetch(" not in html
        assert "localStorage" not in html
        assert "sessionStorage" not in html
        assert "stratai.spatial_context_review_response@2" in html
