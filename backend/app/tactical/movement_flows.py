from typing import Any

from ..utils.maps import CALLOUT_FIXED_POSITIONS, normalize_callout
from .movement_contract_values import clamp_value, count_value, first_value, number_value


def normalize_flow_lines(
    raw_lines: list[dict[str, Any]],
    map_name: str,
) -> list[dict[str, Any]]:
    max_count = max(
        (count_value(line.get("count")) for line in raw_lines),
        default=0,
    )
    normalized = []

    for line in raw_lines:
        count = count_value(line.get("count"))
        if count <= 0:
            continue

        from_area = _area_name(line, "from", map_name)
        to_area = _area_name(line, "to", map_name)
        from_position, to_position = _flow_positions(
            line,
            from_area,
            to_area,
            map_name,
        )
        if from_position is None or to_position is None:
            continue

        route = {
            "from_area": from_area,
            "to_area": to_area,
            "from_x": from_position["x"],
            "from_y": from_position["y"],
            "to_x": to_position["x"],
            "to_y": to_position["y"],
            "count": count,
            "ct_count": count_value(
                first_value(line, "ct_count", "ct")
            ),
            "t_count": count_value(first_value(line, "t_count", "t")),
            "intensity": _flow_intensity(line, count, max_count),
        }
        for field in ("from_level", "to_level"):
            if line.get(field):
                route[field] = str(line[field])
        for field in ("from_avg_z", "to_avg_z"):
            if line.get(field) is not None:
                route[field] = number_value(line[field])
        normalized.append(route)

    return normalized


def _flow_positions(
    line: dict[str, Any],
    from_area: str,
    to_area: str,
    map_name: str,
) -> tuple[dict[str, float] | None, dict[str, float] | None]:
    fixed_positions = CALLOUT_FIXED_POSITIONS.get(map_name, {})
    from_position = fixed_positions.get(from_area) or _explicit_position(line, "from")
    to_position = fixed_positions.get(to_area) or _explicit_position(line, "to")
    return from_position, to_position


def _explicit_position(
    line: dict[str, Any],
    prefix: str,
) -> dict[str, float] | None:
    nested = line.get(f"{prefix}_pos")
    if (
        isinstance(nested, dict)
        and nested.get("x") is not None
        and nested.get("y") is not None
    ):
        return {
            "x": clamp_value(number_value(nested["x"]), 0, 100),
            "y": clamp_value(number_value(nested["y"]), 0, 100),
        }

    x_value = line.get(f"{prefix}_x")
    y_value = line.get(f"{prefix}_y")
    if x_value is None or y_value is None:
        return None

    return {
        "x": clamp_value(number_value(x_value), 0, 100),
        "y": clamp_value(number_value(y_value), 0, 100),
    }


def _area_name(line: dict[str, Any], prefix: str, map_name: str) -> str:
    raw_name = str(first_value(line, f"{prefix}_area", prefix) or "")
    return normalize_callout(raw_name, map_name) or raw_name


def _flow_intensity(
    line: dict[str, Any],
    count: int,
    max_count: int,
) -> float:
    if line.get("intensity") is not None:
        return max(0.0, number_value(line["intensity"]))
    if max_count <= 0:
        return 0.0
    return round(count / max_count * 100, 1)
