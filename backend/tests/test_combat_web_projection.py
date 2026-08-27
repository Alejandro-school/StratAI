from backend.app.matches.combat_web_projection import (
    _horizontal_velocity,
    _horizontal_velocity_available,
)


def test_horizontal_velocity_reads_player_state_v2_field() -> None:
    assert (
        _horizontal_velocity(
            {
                "horizontal_velocity_world_units_per_second": 127.5,
                "velocity_world_units_per_second": 42,
            }
        )
        == 127.5
    )


def test_horizontal_velocity_preserves_v2_unavailable_null() -> None:
    assert (
        _horizontal_velocity(
            {
                "horizontal_velocity_world_units_per_second": None,
                "velocity_world_units_per_second": 42,
            }
        )
        == 0
    )


def test_horizontal_velocity_falls_back_to_player_state_v1_field() -> None:
    assert _horizontal_velocity({"velocity_world_units_per_second": 84}) == 84


def test_horizontal_velocity_rejects_invalid_values() -> None:
    assert (
        _horizontal_velocity({"horizontal_velocity_world_units_per_second": "bad"}) == 0
    )
    assert (
        _horizontal_velocity(
            {"horizontal_velocity_world_units_per_second": float("inf")}
        )
        == 0
    )
    assert _horizontal_velocity({"horizontal_velocity_world_units_per_second": -1}) == 0


def test_horizontal_velocity_availability_distinguishes_unknown_and_stationary() -> (
    None
):
    assert _horizontal_velocity_available(
        {
            "horizontal_velocity_world_units_per_second": 0,
            "velocity_source": "position_delta",
        }
    )
    assert not _horizontal_velocity_available(
        {
            "horizontal_velocity_world_units_per_second": None,
            "velocity_source": "insufficient_history",
        }
    )
