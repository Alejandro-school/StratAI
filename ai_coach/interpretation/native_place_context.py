"""Bind native place strings to an existing factual cut, without inventing zones.

The candidate context remains unchanged. Native strings are a separately hashed
enrichment, unavailable when no export was supplied. In-memory validation checks
consistency; ``load`` additionally binds retained export bytes to expected hashes.
Neither operation certifies a geometric boundary or the player's intention.
"""

import hashlib
import json
import math
from copy import deepcopy
from functools import cache
from pathlib import Path

from ai_coach.interpretation.candidate_context import content_hash
from ai_coach.support.anchors import require, schema_validator

SCHEMA_ID = "stratai.native_place_context@1"
SEMANTICS = "current_native_property_not_geometric_zone"
CUT_FIELDS = ("frame", "tick", "instance", "epoch", "round_number", "phase")


@cache
def _validator():
    return schema_validator("native_place_context.json")


def _seal(value):
    value["native_place_context_id"] = content_hash(value)
    return value


def _base(candidate):
    return {
        "schema_id": SCHEMA_ID,
        "candidate_context_id": candidate["candidate_context_id"],
        "source_demo_sha256": candidate["packet"]["context"]["source_sha256"],
        "cut": deepcopy(candidate["snapshot"]["cut"]),
    }


def _identity(actor):
    return actor["identity"] or {
        "identity_status": "unavailable",
        "pawn_life": None,
        "pawn_handle": None,
        "side": None,
    }


def unavailable(candidate):
    """Lack of an export is not evidence that a place or tactical pattern is absent."""
    value = _seal(
        {
            **_base(candidate),
            "status": "unavailable",
            "reason": "native_source_not_materialized",
            "observations": [],
            "provenance": None,
        }
    )
    return validate(value, candidate)


def _string(status, value, native_type=None):
    if status == "observed":
        require(isinstance(value, str) and bool(value), "native string unavailable")
    elif status == "empty_string":
        require(value == "", "empty native string differs")
    else:
        require(value is None, "unavailable native value asserted")
    if native_type is not None and status in {"observed", "empty_string"}:
        require(native_type == "string", "native string type differs")


def validate(value, candidate):
    """Check bindings against the candidate; its full validator is run by infer."""
    _validator().validate(value)
    require(
        value["native_place_context_id"]
        == content_hash(
            {
                key: item
                for key, item in value.items()
                if key != "native_place_context_id"
            }
        ),
        "native place context hash differs",
    )
    for key, expected in _base(candidate).items():
        require(value[key] == expected, f"native place {key} binding differs")
    if value["status"] == "unavailable":
        require(
            value["observations"] == []
            and value["provenance"] is None
            and value["reason"] == "native_source_not_materialized",
            "unavailable native context asserts evidence",
        )
        return value
    require(
        value["reason"] is None and value["provenance"] is not None,
        "observed native context provenance missing",
    )
    require(
        value["provenance"]["map_name"] == candidate["packet"]["map_name"],
        "native map differs",
    )
    actors = {actor["actor_alias"]: actor for actor in candidate["snapshot"]["actors"]}
    observations = {row["actor_alias"]: row for row in value["observations"]}
    require(
        len(observations) == len(value["observations"])
        and set(observations) == set(actors),
        "native actor roster differs",
    )
    for alias, actor in actors.items():
        row = observations[alias]
        identity = _identity(actor)
        for key, expected in {
            "label": actor["label"],
            "pawn_life": identity["pawn_life"],
            "pawn_handle": identity["pawn_handle"],
            "side": identity["side"],
            "alive": actor["alive"]["value"],
            "connected": actor["connected"],
            "position": actor["position"]["value"],
        }.items():
            require(row[key] == expected, f"native actor {key} binding differs")
        if row["position"] is not None:
            require(
                all(math.isfinite(v) for v in row["position"].values()),
                "native position nonfinite",
            )
        _string(row["value_status"], row["value"], row["native_type"])
        if row["value_status"] in {"observed", "empty_string"}:
            require(
                actor["alive"]["value"] is True
                and actor["connected"] is True
                and identity["identity_status"] == "observed"
                and identity["pawn_handle"] is not None
                and actor["position"]["status"] == "observed"
                and row["native_type"] == "string",
                "native value belongs to unavailable actor",
            )
        delivery = row["last_delivery"]
        if delivery is not None:
            cut = value["cut"]
            require(
                all(
                    delivery[key] == cut[key]
                    for key in ("instance", "epoch", "round_number")
                ),
                "native delivery crosses round or epoch",
            )
            require(
                delivery["frame"] <= cut["frame"] and delivery["tick"] <= cut["tick"],
                "native delivery comes from future",
            )
            _string(delivery["value_status"], delivery["value"])
            if row["value_status"] in {"observed", "empty_string"}:
                require(
                    delivery["value_status"] == row["value_status"]
                    and delivery["value"] == row["value"],
                    "native delivery value differs",
                )
    return value


def build(candidate, snapshot, provenance):
    """Normalize a hash-bound export snapshot; full raw evidence remains retained."""
    require(
        snapshot["schema_id"] == "stratai.native_place_snapshot@1",
        "native snapshot schema differs",
    )
    require(snapshot["target_id"] == provenance["target_id"], "native target differs")
    require(
        snapshot["candidate_context_sha256"] == provenance["candidate_context_sha256"],
        "native candidate file hash differs",
    )
    cut = candidate["snapshot"]["cut"]
    require(
        snapshot["cut"] == {key: cut[key] for key in CUT_FIELDS}, "native cut differs"
    )
    require(snapshot["sdk_frame"] == cut["frame"], "native SDK frame differs")
    raw_actors = {actor["alias"]: actor for actor in snapshot["actors"]}
    require(
        len(raw_actors) == len(snapshot["actors"])
        and set(raw_actors)
        == {actor["label"] for actor in candidate["snapshot"]["actors"]},
        "native snapshot roster differs",
    )
    observations = []
    for actor in candidate["snapshot"]["actors"]:
        identity = _identity(actor)
        raw = raw_actors[actor["label"]]
        row = {
            "actor_alias": actor["actor_alias"],
            "label": actor["label"],
            "pawn_life": identity["pawn_life"],
            "pawn_handle": identity["pawn_handle"],
            "side": identity["side"],
            "alive": actor["alive"]["value"],
            "connected": actor["connected"],
            "position": deepcopy(actor["position"]["value"]),
            "property": "m_szLastPlaceName",
            "value_status": "unavailable",
            "value": None,
            "native_type": None,
            "last_delivery": None,
        }
        if raw["availability"] == "observed":
            require(
                raw["pawn_handle"] == identity["pawn_handle"],
                "native pawn handle differs",
            )
            handle = raw["pawn_handle"]
            require(
                type(handle) is int
                and 0 <= handle <= 4294967295
                and raw["entity_id"] == (handle & 16383)
                and raw["serial"] == (handle >> 14),
                "native pawn serial differs",
            )
            for key, expected in {
                "connected": actor["connected"],
                "side": identity["side"],
                "alive": actor["alive"]["value"],
            }.items():
                if expected is not None:
                    require(raw[key] == expected, f"native observed {key} differs")
            if actor["position"]["status"] == "observed":
                require(
                    raw["rounded_xy_position"] == actor["position"]["value"],
                    "native observed position differs",
                )
            if actor["alive"]["value"] is True and actor["connected"] is True:
                place = raw["place"]
                row.update(
                    value_status=place["status"],
                    value=place["value"],
                    native_type=place.get("native_type"),
                )
                if place["status"] in {"observed", "empty_string"}:
                    require(
                        raw["raw_api_equal"] is True
                        and raw["sdk_last_place_name"] == place["value"],
                        "native SDK string differs",
                    )
                delivery = raw["last_native_delivery"]
                if delivery is not None:
                    require(
                        delivery["pawn_handle"] == handle,
                        "native delivery pawn differs",
                    )
                    row["last_delivery"] = {
                        **{
                            key: delivery["cut"][key]
                            for key in (
                                "frame",
                                "tick",
                                "epoch",
                                "instance",
                                "round_number",
                            )
                        },
                        "value_status": delivery["value"]["status"],
                        "value": delivery["value"]["value"],
                    }
        else:
            require(
                raw["availability"]
                in {
                    "unavailable",
                    "identity_unavailable",
                    "actor_unavailable",
                    "actor_absent",
                    "pawn_reference_unresolved",
                    "sample_error",
                },
                "unsupported native availability",
            )
        observations.append(row)
    return validate(
        _seal(
            {
                **_base(candidate),
                "status": "observed",
                "reason": None,
                "observations": observations,
                "provenance": deepcopy(provenance),
            }
        ),
        candidate,
    )


def _read(path, expected):
    path = Path(path)
    require(
        path.is_file() and not path.is_symlink(), "native input is not a regular file"
    )
    raw = path.read_bytes()
    require(
        hashlib.sha256(raw).hexdigest() == expected, "native input checksum differs"
    )
    return raw


def load(
    candidate,
    *,
    export_dir,
    manifest_sha256,
    target_id,
    targets_sha256,
    candidate_context_sha256,
):
    """Load a separate enrichment; expected hashes must come from a retained receipt."""
    export_dir = Path(export_dir)
    manifest = json.loads(_read(export_dir / "manifest.json", manifest_sha256))
    # Manifest field bindings are kept explicit, not inferred from a place string.
    require(
        manifest["schema_id"] == "stratai.native_place_manifest@1",
        "native manifest schema differs",
    )
    require(
        manifest["source_sha256"] == candidate["packet"]["context"]["source_sha256"],
        "native demo source differs",
    )
    require(
        manifest["source_sha256_after"] == manifest["source_sha256"],
        "native source changed during capture",
    )
    require(
        manifest["targets_sha256"] == targets_sha256
        and manifest["targets_sha256_after"] == targets_sha256,
        "native target source differs",
    )
    require(
        manifest["property_name"] == "m_szLastPlaceName"
        and manifest["producer_version"] == "stratai.native_place_candidate@1",
        "native producer property or version differs",
    )
    require(
        all(
            manifest[key] is False
            for key in (
                "map_assets_loaded",
                "semantic_zone_accredited",
                "future_frames_processed",
                "training_allowed",
                "tactical_prediction_created",
            )
        ),
        "native source semantics differ",
    )
    artifacts = manifest["artifacts"]
    require(
        set(artifacts)
        == {"snapshots.jsonl", "deliveries.jsonl", "state_history.jsonl"},
        "native artifact inventory incomplete",
    )
    for name in artifacts:
        require(
            name in {"snapshots.jsonl", "deliveries.jsonl", "state_history.jsonl"},
            "unexpected native artifact",
        )
        raw = _read(export_dir / name, artifacts[name]["sha256"])
        require(len(raw) == artifacts[name]["bytes"], "native artifact size differs")
    snapshots = _read(
        export_dir / "snapshots.jsonl", artifacts["snapshots.jsonl"]["sha256"]
    )
    rows = [json.loads(line) for line in snapshots.splitlines() if line]
    require(
        len({row["target_id"] for row in rows}) == len(rows),
        "duplicate native snapshot",
    )
    bindings = manifest["target_bindings"]
    require(
        len({row["target_id"] for row in bindings}) == len(bindings)
        and {row["target_id"] for row in bindings}
        == {row["target_id"] for row in rows},
        "native target inventory differs",
    )
    require(
        manifest["capture"]["snapshots"] == len(rows)
        and manifest["capture"]["last_processed_cut"]
        == max(bindings, key=lambda row: row["cut"]["frame"])["cut"],
        "native capture extends beyond bound cuts",
    )
    selected = [row for row in rows if row["target_id"] == target_id]
    require(len(selected) == 1, "native target unavailable")
    deliveries = [
        json.loads(line)
        for line in _read(
            export_dir / "deliveries.jsonl", artifacts["deliveries.jsonl"]["sha256"]
        ).splitlines()
        if line
    ]
    require(
        manifest["capture"]["property_deliveries"] == len(deliveries),
        "native delivery count differs",
    )
    delivered = {content_hash(row) for row in deliveries}
    require(
        all(
            actor["last_native_delivery"] is None
            or content_hash(actor["last_native_delivery"]) in delivered
            for actor in selected[0]["actors"]
        ),
        "native delivery absent from retained source",
    )
    binding = next(row for row in bindings if row["target_id"] == target_id)
    require(
        binding["candidate_context_sha256"] == candidate_context_sha256
        and binding["cut"] == selected[0]["cut"],
        "native selected target binding differs",
    )
    expected_actors = [
        {
            "alias": actor["label"],
            "handle": _identity(actor)["pawn_handle"],
            "side": _identity(actor)["side"]
            or candidate["packet"]["context"]["perspective"]["side"],
            "connected": actor["connected"],
            "alive": actor["alive"]["value"],
            "position": actor["position"]["value"],
        }
        for actor in candidate["snapshot"]["actors"]
    ]
    require(binding["actors"] == expected_actors, "native target actor binding differs")
    header = manifest["header"]
    provenance = {
        "manifest_sha256": manifest_sha256,
        "snapshot_sha256": artifacts["snapshots.jsonl"]["sha256"],
        "targets_sha256": targets_sha256,
        "candidate_context_sha256": candidate_context_sha256,
        "target_id": target_id,
        "map_name": header["map_name"],
        "build": header.get("build"),
        "patch": header.get("patch"),
        "source_semantics": SEMANTICS,
    }
    return build(candidate, selected[0], provenance)
