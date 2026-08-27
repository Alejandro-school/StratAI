from backend.app.tactical.movement_contract import (
    derive_top_positions,
    normalize_flow_lines,
    normalize_heatmap_grid,
    normalize_movement_contract,
)


def test_heatmap_contract_preserves_intensity_and_side_counts() -> None:
    normalized = normalize_heatmap_grid(
        [
            {
                "x": 20,
                "y": 30,
                "sample_count": 8,
                "ct_count": 5,
                "t_count": 3,
                "intensity": 42.25,
                "avg_z": -350.5,
            },
            {
                "x": 40,
                "y": 50,
                "total": 4,
                "ct": 1,
                "t": 3,
                "z_sum": 400,
            },
        ],
        "de_dust2",
    )

    assert normalized == [
        {
            "x": 20.0,
            "y": 30.0,
            "intensity": 42.25,
            "sample_count": 8,
            "ct_count": 5,
            "t_count": 3,
            "ct_ratio": 62.5,
            "avg_z": -350.5,
        },
        {
            "x": 40.0,
            "y": 50.0,
            "intensity": 50.0,
            "sample_count": 4,
            "ct_count": 1,
            "t_count": 3,
            "ct_ratio": 25.0,
            "avg_z": 100.0,
        },
    ]


def test_flow_contract_accepts_aggregate_and_fallback_formats() -> None:
    normalized = normalize_flow_lines(
        [
            {
                "from": "LongA",
                "to": "BombsiteA",
                "count": 8,
                "ct": 3,
                "t": 5,
            },
            {
                "from_area": "Unknown start",
                "to_area": "Unknown end",
                "from_x": 11,
                "from_y": 22,
                "to_x": 33,
                "to_y": 44,
                "count": 4,
                "ct_count": 1,
                "t_count": 3,
                "intensity": 17.5,
            },
        ],
        "de_dust2",
    )

    assert normalized[0] == {
        "from_area": "Long A",
        "to_area": "A Site",
        "from_x": 38,
        "from_y": 30,
        "to_x": 68,
        "to_y": 27,
        "count": 8,
        "ct_count": 3,
        "t_count": 5,
        "intensity": 100.0,
    }
    assert normalized[1] == {
        "from_area": "Unknown start",
        "to_area": "Unknown end",
        "from_x": 11.0,
        "from_y": 22.0,
        "to_x": 33.0,
        "to_y": 44.0,
        "count": 4,
        "ct_count": 1,
        "t_count": 3,
        "intensity": 17.5,
    }


def test_flow_contract_drops_routes_without_real_endpoint_coordinates() -> None:
    normalized = normalize_flow_lines(
        [
            {
                "from_area": "Unknown start",
                "to_area": "A Site",
                "count": 12,
            },
            {
                "from_area": "Unknown start",
                "to_area": "Unknown end",
                "count": 8,
            },
        ],
        "de_dust2",
    )

    assert normalized == []


def test_top_positions_are_derived_from_nearest_fixed_callout() -> None:
    top_positions = derive_top_positions(
        [
            {
                "x": 68,
                "y": 27,
                "sample_count": 8,
                "ct_count": 6,
                "t_count": 2,
            },
            {
                "x": 48,
                "y": 82,
                "sample_count": 2,
                "ct_count": 0,
                "t_count": 2,
            },
        ],
        "de_dust2",
    )

    assert top_positions[:2] == [
        {
            "area": "A Site",
            "time_percent": 80.0,
            "ct_percent": 75.0,
            "sample_count": 8,
            "position": {"x": 68, "y": 27},
        },
        {
            "area": "B Site",
            "time_percent": 20.0,
            "ct_percent": 0.0,
            "sample_count": 2,
            "position": {"x": 48, "y": 82},
        },
    ]


def test_movement_contract_preserves_existing_metrics() -> None:
    movement = {
        "heatmap_grid": [
            {
                "x": 68,
                "y": 27,
                "total": 5,
                "ct": 2,
                "t": 3,
            }
        ],
        "flow_lines": [],
        "metrics": {
            "avg_time_to_a": {"ct": 12.4, "t": 18.1},
            "total_rounds": 27,
            "total_samples": 999,
        },
    }

    normalized = normalize_movement_contract(movement, "de_dust2")

    assert normalized["metrics"]["avg_time_to_a"] == {"ct": 12.4, "t": 18.1}
    assert normalized["metrics"]["total_rounds"] == 27
    assert normalized["metrics"]["total_samples"] == 999
    assert normalized["metrics"]["top_positions"][0]["area"] == "A Site"


def test_missing_vertical_data_remains_unknown() -> None:
    normalized = normalize_movement_contract(
        {
            "heatmap_grid": [
                {"x": 20, "y": 20, "total": 4, "ct": 2, "t": 2},
            ],
            "flow_lines": [],
            "metrics": {},
        },
        "de_nuke",
        {
            "Upper area": {
                "position": {"x": 21, "y": 19},
                "avg_z": -320,
            },
            "Lower area": {
                "position": {"x": 80, "y": 80},
                "avg_z": -710,
            },
        },
    )

    assert normalized["heatmap_grid"][0]["avg_z"] is None
