from typing import Any


def first_value(data: dict[str, Any], *keys: str) -> Any:
    for key in keys:
        if data.get(key) is not None:
            return data[key]
    return None


def count_value(value: Any) -> int:
    return max(0, int(number_value(value)))


def number_value(value: Any, default: float = 0.0) -> float:
    try:
        return float(value)
    except (TypeError, ValueError):
        return default


def percentage_value(value: Any, default: float) -> float:
    return clamp_value(number_value(value, default), 0, 100)


def clamp_value(value: float, minimum: float, maximum: float) -> float:
    return max(minimum, min(maximum, value))
