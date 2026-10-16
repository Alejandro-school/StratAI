"""Membership in recovered Source 2 place hulls, not nearest landmark labels.

Geometry is a local map reference; matching the demo's exact build is separate.
The public result preserves that qualification even for unique membership.
"""
from collections import Counter
from functools import lru_cache
import hashlib
import json
import math
from pathlib import Path

SCHEMA_ID = "stratai.replay_place_volumes@1"
EPSILON = 0.0001  # Numerical tolerance in world units, not a tactical radius.
ASSET = Path(__file__).with_name("maps") / "de_mirage_place_volumes.json"


def encoded(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()


def digest(value):
    return hashlib.sha256(encoded(value)).hexdigest()


def _require(condition, message):
    if not condition:
        raise ValueError(message)


def _point(value):
    return len(value) == 3 and all(type(v) in (int, float) and math.isfinite(v) for v in value)


def build_hull(vertices, faces):
    """Validate a closed convex polyhedron and derive outward unit planes."""
    _require(len(vertices) >= 4 and all(_point(v) for v in vertices), "invalid hull vertices")
    _require(len(faces) >= 4, "open hull")
    center = [sum(v[i] for v in vertices) / len(vertices) for i in range(3)]
    planes, edges, used = [], Counter(), set()
    for face in faces:
        _require(len(face) >= 3 and len(set(face)) == len(face), "degenerate hull face")
        _require(all(type(i) is int and 0 <= i < len(vertices) for i in face), "invalid face index")
        used.update(face)
        normal = [0.0, 0.0, 0.0]
        for a, b in zip(face, face[1:] + face[:1]):
            edges[tuple(sorted((a, b)))] += 1
            p, q = vertices[a], vertices[b]
            normal[0] += (p[1]-q[1])*(p[2]+q[2])
            normal[1] += (p[2]-q[2])*(p[0]+q[0])
            normal[2] += (p[0]-q[0])*(p[1]+q[1])
        length = math.sqrt(sum(v*v for v in normal))
        _require(length > EPSILON, "zero area face")
        normal = [v/length for v in normal]
        offset = sum(a*b for a,b in zip(normal,vertices[face[0]]))
        if sum(a*b for a,b in zip(normal,center)) > offset:
            normal = [-v for v in normal]; offset = -offset
        distance = lambda point: sum(a*b for a,b in zip(normal,point))-offset
        _require(all(abs(distance(vertices[i])) <= EPSILON for i in face), "nonplanar face")
        _require(all(distance(v) <= EPSILON for v in vertices), "nonconvex hull")
        _require(distance(center) < -EPSILON, "flat hull")
        planes.append([*normal, offset])
    _require(len(used) == len(vertices) and all(n == 2 for n in edges.values()), "open or nonmanifold hull")
    _require(len(vertices)-len(edges)+len(faces) == 2, "disconnected hull")
    return {"vertices":vertices,"faces":faces,"planes":planes,
            "bounds":{"min":[min(v[i] for v in vertices) for i in range(3)],
                      "max":[max(v[i] for v in vertices) for i in range(3)]}}


def resolve(position, package, *, map_name, xy_precision=0):
    """Classify the supplied point; uncertainty never falls back to a seed."""
    result = {"schema_id":"stratai.replay_place@1", "status":"unavailable", "zone":None,
              "candidates":[], "reference_id":package.get("reference_id") if package else None,
              "build_compatibility":"unverified", "reason":"outside_place_volumes"}
    if not package or package.get("map_name") != map_name:
        return {**result,"reference_id":None,"reason":"map_reference_unavailable"}
    point = [position.get(k) for k in ("x","y","z")] if isinstance(position, dict) else []
    if not _point(point) or type(xy_precision) not in (int, float) or not math.isfinite(xy_precision) or xy_precision < 0:
        return {**result,"reason":"position_unavailable"}
    for region in package["regions"]:
        if region["label"].startswith("position."):
            continue
        relation = None
        for hull in region["hulls"]:
            distances = [sum(n*p for n,p in zip(plane[:3],point))-plane[3] for plane in hull["planes"]]
            tolerances = [EPSILON+xy_precision*(abs(p[0])+abs(p[1])) for p in hull["planes"]]
            if all(d <= t for d,t in zip(distances,tolerances)):
                interior = all(d < -t for d,t in zip(distances,tolerances))
                relation = "inside" if interior or relation == "inside" else "boundary"
        if relation:
            result["candidates"].append({"id":region["id"],"label":region["label"],"relation":relation})
    result["candidates"].sort(key=lambda row:row["id"])
    labels = {c["label"] for c in result["candidates"]}
    if len(labels) > 1:
        result.update(status="ambiguous",reason="overlapping_place_volumes")
    elif labels:
        interior = any(c["relation"] == "inside" for c in result["candidates"])
        result.update(status="mapped" if interior else "boundary",zone=next(iter(labels)) if interior else None,
                      reason="unique_local_volume" if interior else "place_volume_boundary")
    return result


def validate(package):
    _require(package.get("schema_id") == SCHEMA_ID, "invalid place package schema")
    _require(package.get("build_compatibility") == "unverified", "unaccredited map build")
    _require(package["reference_id"] == digest({k:v for k,v in package.items() if k != "reference_id"}), "place package hash mismatch")
    _require(package["coordinate_space"] == "source2_world_baked_mesh", "unsupported coordinate space")
    _require(bool(package["regions"]), "no place volumes")
    ids = []
    for region in package["regions"]:
        _require(bool(region["label"]) and not region["label"].startswith("position."), "internal place label")
        _require(bool(region["hulls"]), "missing place hull")
        ids.append(region["id"])
        for hull in region["hulls"]:
            expected = build_hull(hull["vertices"], hull["faces"])
            _require(hull == expected, "place planes differ from mesh")
    _require(len(ids) == len(set(ids)), "duplicate place id")
    return package


@lru_cache(maxsize=1)
def load_reference():
    return validate(json.loads(ASSET.read_bytes()))
