"""Traceable map landmarks with explicit absence of region containment.

Local VMAP place origins are development landmarks, not zone boundaries or a
certificate that the asset matches the replay's build.
"""

import hashlib
import json
import math
import re
from pathlib import Path

from ai_coach.interpretation.temporal import digest
from ai_coach.support.anchors import require

SCHEMA_ID = "stratai.contextual_map_reference@1"
LABELS = {"BombsiteA": "A", "BombsiteB": "B", "Apartments": "apartamentos"}


def extract_vmap(path, *, map_name):
    """Read named env_cs_place origins. Reject transforms requiring interpretation."""
    path = Path(path)
    raw = path.read_bytes()
    text = raw.decode("utf-8").replace("\r\n", "\n")
    landmarks = []
    for block in text.split('"CMapEntity"\n')[1:]:
        # DMX top-level CMapEntity ends at its first unindented closing brace.
        block = block.split("\n}\n", 1)[0]
        if not re.search(r'"classname"\s+"string"\s+"env_cs_place"', block):
            continue

        def field(name, kind, block=block):
            matches = re.findall(
                r'"' + name + r'"\s+"' + kind + r'"\s+"([^"\n]+)"', block
            )
            require(len(matches) == 1, "ambiguous VMAP " + name)
            return matches[0]

        label = field("place_name", "string")
        # Tokens requiring localization or workshop context are not semantic labels.
        if label.startswith("position."):
            continue
        origin = [float(x) for x in field("origin", "vector3").split()]
        require(
            len(origin) == 3 and all(math.isfinite(x) for x in origin),
            "invalid landmark",
        )
        require(
            field("angles", "qangle") == "0 0 0"
            and field("scales", "vector3") == "1 1 1",
            "unsupported map landmark transform",
        )
        landmarks.append(
            {
                "id": field("nodeID", "int"),
                "source_label": label,
                "label": LABELS.get(label, label),
                "position": dict(zip(("x", "y", "z"), origin)),
                "kind": "site_landmark"
                if label in ("BombsiteA", "BombsiteB")
                else "landmark",
            }
        )
    require(bool(landmarks), "no named VMAP landmarks")
    package = {
        "schema_id": SCHEMA_ID,
        "map_name": map_name,
        "source": {
            "artifact": path.name,
            "sha256": hashlib.sha256(raw).hexdigest(),
            "bytes": len(raw),
            "kind": "local_vmap_env_cs_place_origin",
        },
        "scope": "development_reference",
        "build_compatibility": "unverified",
        "landmarks": sorted(landmarks, key=lambda x: x["id"]),
        "regions": [],
        "physical_route_accredited": False,
        "training_allowed": False,
    }
    package["reference_id"] = digest(package)
    validate(package)
    return package


def validate(package):
    required = {
        "schema_id",
        "map_name",
        "source",
        "scope",
        "build_compatibility",
        "landmarks",
        "regions",
        "physical_route_accredited",
        "training_allowed",
        "reference_id",
    }
    require(
        set(package) == required and package["schema_id"] == SCHEMA_ID,
        "invalid map reference",
    )
    require(
        package["reference_id"]
        == digest({k: v for k, v in package.items() if k != "reference_id"}),
        "map reference hash mismatch",
    )
    require(
        package["scope"] == "development_reference"
        and package["build_compatibility"] == "unverified"
        and package["physical_route_accredited"] is False
        and package["training_allowed"] is False,
        "unsupported map accreditation",
    )
    require(
        isinstance(package["map_name"], str) and package["map_name"].startswith("de_"),
        "invalid map",
    )
    require(
        package["source"]["kind"] == "local_vmap_env_cs_place_origin"
        and re.fullmatch(r"[0-9a-f]{64}", package["source"]["sha256"]) is not None,
        "invalid map source",
    )
    require(
        not package["regions"],
        "region boundaries require a separately verified producer",
    )
    ids = []
    for item in package["landmarks"]:
        require(
            set(item) == {"id", "source_label", "label", "position", "kind"},
            "invalid landmark fields",
        )
        require(
            item["label"] == LABELS.get(item["source_label"], item["source_label"]),
            "unbound map label",
        )
        require(
            item["kind"]
            == (
                "site_landmark"
                if item["source_label"] in ("BombsiteA", "BombsiteB")
                else "landmark"
            ),
            "unbound landmark kind",
        )
        require(
            set(item["position"]) == {"x", "y", "z"}
            and all(
                type(v) in (int, float) and math.isfinite(v)
                for v in item["position"].values()
            ),
            "invalid map coordinates",
        )
        ids.append(item["id"])
    require(len(set(ids)) == len(ids), "duplicate map landmark")
    return package


def resolve(position, package, *, map_name):
    """Distances are diagnostics. A seed never authorizes 'inside apartments'."""
    if package is None:
        return {
            "status": "unavailable",
            "zone": None,
            "reason": "missing_map_reference",
            "landmarks": [],
        }
    validate(package)
    require(package["map_name"] == map_name, "map reference mismatch")
    require(
        set(position) == {"x", "y", "z"}
        and all(math.isfinite(x) for x in position.values()),
        "invalid position",
    )
    return {
        "status": "development_reference",
        "zone": None,
        "reason": "landmarks_do_not_define_zone_boundaries_or_build_compatibility",
        "landmarks": [
            {
                "id": x["id"],
                "label": x["label"],
                "horizontal_distance": math.hypot(
                    position["x"] - x["position"]["x"],
                    position["y"] - x["position"]["y"],
                ),
                "reference_id": package["reference_id"],
            }
            for x in package["landmarks"]
        ],
    }


def load(path):
    return validate(json.loads(Path(path).read_bytes()))
