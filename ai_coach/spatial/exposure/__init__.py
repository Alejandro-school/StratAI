"""Map-agnostic exposure-angle package loading and validation."""

from .loader import (
    CONTRACT_VERSION,
    PackageResolution,
    build_sectors,
    load_package,
    normalize_degrees,
    relative_sectors,
    validate_package,
)

__all__ = [
    "CONTRACT_VERSION",
    "PackageResolution",
    "build_sectors",
    "load_package",
    "normalize_degrees",
    "relative_sectors",
    "validate_package",
]
