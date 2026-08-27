from typing import Any

from .movement_flows import normalize_flow_lines
from .movement_heatmap import (
    add_top_position_coordinates,
    derive_top_positions,
    normalize_heatmap_grid,
)


def normalize_movement_contract(
    movement: dict[str, Any],
    map_name: str,
    callout_stats: dict[str, Any] | None = None,
) -> dict[str, Any]:
    heatmap_grid = normalize_heatmap_grid(
        movement.get("heatmap_grid", []),
        map_name,
        callout_stats,
    )
    flow_lines = normalize_flow_lines(
        movement.get("flow_lines", []),
        map_name,
    )
    metrics = dict(movement.get("metrics") or {})

    if metrics.get("top_positions"):
        metrics["top_positions"] = add_top_position_coordinates(
            metrics["top_positions"],
            map_name,
        )
    else:
        metrics["top_positions"] = derive_top_positions(heatmap_grid, map_name)

    metrics.setdefault(
        "total_samples",
        sum(cell["sample_count"] for cell in heatmap_grid),
    )

    return {
        "heatmap_grid": heatmap_grid,
        "flow_lines": flow_lines,
        "metrics": metrics,
    }


__all__ = [
    "derive_top_positions",
    "add_top_position_coordinates",
    "normalize_flow_lines",
    "normalize_heatmap_grid",
    "normalize_movement_contract",
]
