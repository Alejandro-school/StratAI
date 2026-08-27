from collections import defaultdict
from typing import Any

from ..utils.maps import CALLOUT_FIXED_POSITIONS, game_to_radar_percent
from .movement_contract_values import (
    clamp_value,
    count_value,
    first_value,
    number_value,
    percentage_value,
)

GRID_SIZE = 20


def normalize_heatmap_grid(
    raw_grid: list[dict[str, Any]],
    map_name: str,
    _callout_stats: dict[str, Any] | None = None,
) -> list[dict[str, Any]]:
    max_count = max((_sample_count(cell) for cell in raw_grid), default=0)
    normalized = []

    for cell in raw_grid:
        sample_count, ct_count, t_count = _side_counts(cell)
        intensity = _cell_intensity(cell, sample_count, max_count)
        position = _cell_position(cell, map_name)

        if position is None or (sample_count <= 0 and intensity <= 0):
            continue

        normalized.append(
            {
                "x": position["x"],
                "y": position["y"],
                "intensity": intensity,
                "sample_count": sample_count,
                "ct_count": ct_count,
                "t_count": t_count,
                "ct_ratio": _ct_ratio(cell, sample_count, ct_count),
                "avg_z": _average_z(cell, sample_count),
            }
        )

    normalized.sort(key=lambda cell: cell["intensity"])
    return normalized


def derive_top_positions(
    heatmap_grid: list[dict[str, Any]],
    map_name: str,
) -> list[dict[str, Any]]:
    fixed_positions = CALLOUT_FIXED_POSITIONS.get(map_name, {})
    if not fixed_positions:
        return []

    totals = defaultdict(lambda: {"sample_count": 0, "ct_count": 0})
    for cell in heatmap_grid:
        sample_count = count_value(cell.get("sample_count"))
        if sample_count <= 0:
            continue

        area = min(
            fixed_positions,
            key=lambda name: _distance_squared(cell, fixed_positions[name]),
        )
        totals[area]["sample_count"] += sample_count
        totals[area]["ct_count"] += count_value(cell.get("ct_count"))

    total_samples = sum(item["sample_count"] for item in totals.values())
    ranked = sorted(
        totals.items(),
        key=lambda item: item[1]["sample_count"],
        reverse=True,
    )

    return [
        {
            "area": area,
            "time_percent": round(data["sample_count"] / total_samples * 100, 1),
            "ct_percent": round(
                data["ct_count"] / data["sample_count"] * 100,
                1,
            ),
            "sample_count": data["sample_count"],
            "position": {
                "x": fixed_positions[area]["x"],
                "y": fixed_positions[area]["y"],
            },
        }
        for area, data in ranked[:10]
    ]


def add_top_position_coordinates(
    positions: list[dict[str, Any]],
    map_name: str,
) -> list[dict[str, Any]]:
    fixed_positions = CALLOUT_FIXED_POSITIONS.get(map_name, {})
    enriched = []

    for position in positions:
        item = dict(position)
        fixed_position = fixed_positions.get(str(item.get("area") or ""))
        if not item.get("position") and fixed_position:
            item["position"] = {
                "x": fixed_position["x"],
                "y": fixed_position["y"],
            }
        enriched.append(item)

    return enriched


def _side_counts(cell: dict[str, Any]) -> tuple[int, int, int]:
    sample_count = _sample_count(cell)
    raw_ct = first_value(cell, "ct_count", "ct")
    raw_t = first_value(cell, "t_count", "t")
    ct_count = count_value(raw_ct)
    t_count = count_value(raw_t)

    if sample_count <= 0:
        sample_count = ct_count + t_count

    if raw_ct is None and raw_t is not None:
        ct_count = max(0, sample_count - t_count)
    elif raw_t is None and raw_ct is not None:
        t_count = max(0, sample_count - ct_count)
    elif raw_ct is None and raw_t is None and sample_count > 0:
        ratio = percentage_value(cell.get("ct_ratio"), default=50.0)
        ct_count = round(sample_count * ratio / 100)
        t_count = sample_count - ct_count

    return sample_count, ct_count, t_count


def _cell_position(
    cell: dict[str, Any],
    map_name: str,
) -> dict[str, float] | None:
    if cell.get("game_x") is not None and cell.get("game_y") is not None:
        return game_to_radar_percent(
            number_value(cell["game_x"]),
            number_value(cell["game_y"]),
            map_name,
        )

    if cell.get("x") is not None and cell.get("y") is not None:
        return {
            "x": clamp_value(number_value(cell["x"]), 0, 100),
            "y": clamp_value(number_value(cell["y"]), 0, 100),
        }

    if cell.get("grid_x") is None or cell.get("grid_y") is None:
        return None

    cell_size = 100 / GRID_SIZE
    return {
        "x": (number_value(cell["grid_x"]) + 0.5) * cell_size,
        "y": (number_value(cell["grid_y"]) + 0.5) * cell_size,
    }


def _sample_count(cell: dict[str, Any]) -> int:
    return count_value(
        first_value(cell, "sample_count", "total", "count", "samples")
    )


def _cell_intensity(
    cell: dict[str, Any],
    sample_count: int,
    max_count: int,
) -> float:
    if cell.get("intensity") is not None:
        return max(0.0, number_value(cell["intensity"]))
    if max_count <= 0:
        return 0.0
    return round(sample_count / max_count * 100, 1)


def _ct_ratio(
    cell: dict[str, Any],
    sample_count: int,
    ct_count: int,
) -> float:
    if sample_count > 0 and first_value(cell, "ct_count", "ct") is not None:
        return round(ct_count / sample_count * 100, 1)
    return round(percentage_value(cell.get("ct_ratio"), default=50.0), 1)


def _average_z(
    cell: dict[str, Any],
    sample_count: int,
) -> float | None:
    if cell.get("avg_z") is not None:
        return number_value(cell["avg_z"])
    if cell.get("z_sum") is not None and sample_count > 0:
        return round(number_value(cell["z_sum"]) / sample_count, 1)
    return None


def _distance_squared(
    first: dict[str, Any],
    second: dict[str, Any],
) -> float:
    return (
        (number_value(first.get("x")) - number_value(second.get("x"))) ** 2
        + (number_value(first.get("y")) - number_value(second.get("y"))) ** 2
    )
