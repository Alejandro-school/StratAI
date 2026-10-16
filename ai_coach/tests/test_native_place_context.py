"""Native enrichment binds original context bytes, exact cuts and causal values."""

import hashlib
import json
from copy import deepcopy
from functools import cache

import pytest
from jsonschema import ValidationError

from ai_coach.interpretation import candidate_context
from ai_coach.interpretation import native_place_context as native
from ai_coach.tests.test_destination_orientation import inputs


def encode(value):
    return (
        json.dumps(value, sort_keys=True, indent=2, allow_nan=False) + "\n"
    ).encode()


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


@cache
def _candidate():
    packet, kwargs = inputs()
    return candidate_context.build(packet, **kwargs)


def candidate_fixture():
    return deepcopy(_candidate())


def snapshot_fixture(candidate, candidate_sha256):
    cut = {key: candidate["snapshot"]["cut"][key] for key in native.CUT_FIELDS}
    actors, deliveries = [], []
    for index, actor in enumerate(candidate["snapshot"]["actors"]):
        handle = actor["identity"]["pawn_handle"]
        live = actor["alive"]["value"] is True
        place = "BombsiteA" if index == 0 else "Middle"
        scalar = {"status": "observed", "native_type": "string", "value": place}
        delivery = (
            {
                "sequence": index + 1,
                "cut": deepcopy(cut),
                "pawn_handle": handle,
                "value": scalar,
            }
            if live
            else None
        )
        if delivery:
            deliveries.append(delivery)
        actors.append(
            {
                "alias": actor["label"],
                "pawn_handle": handle,
                "entity_id": handle & 16383,
                "serial": handle >> 14,
                "availability": "observed" if live else "actor_absent",
                "connected": actor["connected"],
                "alive": actor["alive"]["value"],
                "side": actor["identity"]["side"],
                "raw_position": deepcopy(actor["position"]["value"]),
                "rounded_xy_position": deepcopy(actor["position"]["value"]),
                "place": scalar
                if live
                else {"status": "entity_unavailable", "value": None},
                "sdk_last_place_name": place if live else None,
                "raw_api_equal": True if live else None,
                "last_native_delivery": delivery,
                "first_observed_in_instance": deepcopy(cut) if live else None,
                "last_observed_state_change": deepcopy(cut) if live else None,
            }
        )
    return {
        "schema_id": "stratai.native_place_snapshot@1",
        "target_id": "control",
        "candidate_context_sha256": candidate_sha256,
        "cut": cut,
        "sdk_frame": cut["frame"],
        "actors": actors,
    }, deliveries


def export_fixture(root, *, candidate=None):
    root.mkdir()
    candidate = candidate_fixture() if candidate is None else candidate
    raw = encode(candidate)
    candidate_path = root / "candidate.json"
    candidate_path.write_bytes(raw)
    snapshot, deliveries = snapshot_fixture(candidate, sha(raw))
    export = root / "native"
    export.mkdir()
    contents = {
        "snapshots.jsonl": json.dumps(snapshot).encode() + b"\n",
        "deliveries.jsonl": b"".join(
            json.dumps(row).encode() + b"\n" for row in deliveries
        ),
        "state_history.jsonl": b"",
    }
    for name, data in contents.items():
        (export / name).write_bytes(data)
    binding = {
        "target_id": "control",
        "candidate_context_sha256": sha(raw),
        "cut": deepcopy(snapshot["cut"]),
        "actors": [
            {
                "alias": actor["label"],
                "handle": actor["identity"]["pawn_handle"],
                "side": actor["identity"]["side"],
                "connected": actor["connected"],
                "alive": actor["alive"]["value"],
                "position": deepcopy(actor["position"]["value"]),
            }
            for actor in candidate["snapshot"]["actors"]
        ],
    }
    manifest = {
        "schema_id": "stratai.native_place_manifest@1",
        "producer_version": "stratai.native_place_candidate@1",
        "source_alias": candidate["packet"]["context"]["source_alias"],
        "source_sha256": candidate["packet"]["context"]["source_sha256"],
        "source_sha256_after": candidate["packet"]["context"]["source_sha256"],
        "targets_sha256": "c" * 64,
        "targets_sha256_after": "c" * 64,
        "target_bindings": [binding],
        "header": {"map_name": "de_mirage", "build": 10847, "patch": 14176},
        "property_name": "m_szLastPlaceName",
        "capture": {
            "snapshots": 1,
            "property_deliveries": len(deliveries),
            "last_processed_cut": deepcopy(snapshot["cut"]),
        },
        "artifacts": {
            name: {"sha256": sha(data), "bytes": len(data)}
            for name, data in contents.items()
        },
        "map_assets_loaded": False,
        "semantic_zone_accredited": False,
        "future_frames_processed": False,
        "training_allowed": False,
        "tactical_prediction_created": False,
    }
    (export / "manifest.json").write_bytes(encode(manifest))
    args = {
        "candidate_path": candidate_path,
        "candidate_sha256": sha(raw),
        "export_dir": export,
        "manifest_sha256": sha(encode(manifest)),
        "target_id": "control",
        "targets_sha256": "c" * 64,
    }
    return candidate, args, manifest, snapshot


def load_args(args):
    return {
        **{
            k: args[k]
            for k in ("export_dir", "manifest_sha256", "target_id", "targets_sha256")
        },
        "candidate_context_sha256": args["candidate_sha256"],
    }


def replace_manifest(args, manifest):
    raw = encode(manifest)
    (args["export_dir"] / "manifest.json").write_bytes(raw)
    args["manifest_sha256"] = sha(raw)


def replace_snapshot(args, manifest, snapshot):
    raw = json.dumps(snapshot).encode() + b"\n"
    (args["export_dir"] / "snapshots.jsonl").write_bytes(raw)
    manifest["artifacts"]["snapshots.jsonl"] = {"sha256": sha(raw), "bytes": len(raw)}
    replace_manifest(args, manifest)


def test_export_roundtrip_preserves_values_delivery_and_original_candidate(tmp_path):
    candidate, args, _, _ = export_fixture(tmp_path / "source")
    before = deepcopy(candidate)
    value = native.load(candidate, **load_args(args))
    assert native.validate(value, candidate) == value
    assert value["observations"][0]["value"] == "BombsiteA"
    assert value["observations"][1]["value"] == "Middle"
    assert value["observations"][2]["value_status"] == "unavailable"
    assert value["provenance"]["source_semantics"] == native.SEMANTICS
    assert value["provenance"]["candidate_context_sha256"] == args["candidate_sha256"]
    assert candidate == before
    assert sha(args["candidate_path"].read_bytes()) == args["candidate_sha256"]


@pytest.mark.parametrize(
    "artifact",
    ["manifest.json", "snapshots.jsonl", "deliveries.jsonl", "state_history.jsonl"],
)
def test_expected_byte_hash_rejects_changed_source(tmp_path, artifact):
    candidate, args, _, _ = export_fixture(tmp_path / "source")
    (args["export_dir"] / artifact).write_bytes(b"changed")
    with pytest.raises(ValueError, match="checksum"):
        native.load(candidate, **load_args(args))


@pytest.mark.parametrize(
    "mutation",
    [
        "target",
        "candidate",
        "cut",
        "sdk",
        "serial",
        "position",
        "side",
        "raw_api",
        "delivery_future",
        "delivery_missing",
    ],
)
def test_semantic_misalignment_rejected_even_with_new_expected_file_hashes(
    tmp_path, mutation
):
    candidate, args, manifest, snapshot = export_fixture(tmp_path / "source")
    first = snapshot["actors"][0]
    if mutation == "target":
        snapshot["target_id"] = "other"
    elif mutation == "candidate":
        snapshot["candidate_context_sha256"] = "f" * 64
    elif mutation == "cut":
        snapshot["cut"]["tick"] += 1
    elif mutation == "sdk":
        snapshot["sdk_frame"] += 1
    elif mutation == "serial":
        first["serial"] += 1
    elif mutation == "position":
        first["rounded_xy_position"]["x"] += 1
    elif mutation == "side":
        first["side"] = "CT"
    elif mutation == "raw_api":
        first["raw_api_equal"] = False
    elif mutation == "delivery_future":
        first["last_native_delivery"]["cut"]["tick"] += 1
        # Rebind the source line too: exact current-cut validation must still fail.
        rows = [
            a["last_native_delivery"]
            for a in snapshot["actors"]
            if a["last_native_delivery"]
        ]
        raw = b"".join(json.dumps(row).encode() + b"\n" for row in rows)
        (args["export_dir"] / "deliveries.jsonl").write_bytes(raw)
        manifest["artifacts"]["deliveries.jsonl"] = {
            "sha256": sha(raw),
            "bytes": len(raw),
        }
    else:
        first["last_native_delivery"]["sequence"] = 100
    replace_snapshot(args, manifest, snapshot)
    with pytest.raises(ValueError):
        native.load(candidate, **load_args(args))


@pytest.mark.parametrize(
    "mutation",
    [
        "source_changed",
        "target_changed",
        "missing_artifact",
        "future",
        "capture",
        "actor_binding",
        "property",
    ],
)
def test_manifest_bindings_are_more_than_self_consistent_hashes(tmp_path, mutation):
    candidate, args, manifest, _ = export_fixture(tmp_path / "source")
    if mutation == "source_changed":
        manifest["source_sha256_after"] = "f" * 64
    elif mutation == "target_changed":
        manifest["targets_sha256_after"] = "f" * 64
    elif mutation == "missing_artifact":
        manifest["artifacts"].pop("state_history.jsonl")
    elif mutation == "future":
        manifest["future_frames_processed"] = True
    elif mutation == "capture":
        manifest["capture"]["last_processed_cut"]["frame"] += 1
    elif mutation == "actor_binding":
        manifest["target_bindings"][0]["actors"][0]["handle"] += 16384
    else:
        manifest["property_name"] = "geometric_zone"
    replace_manifest(args, manifest)
    with pytest.raises(ValueError):
        native.load(candidate, **load_args(args))


@pytest.mark.parametrize(
    "status", ["actor_absent", "pawn_reference_unresolved", "sample_error"]
)
def test_native_capture_absence_is_preserved_without_inventing_place(tmp_path, status):
    candidate, args, manifest, snapshot = export_fixture(tmp_path / "source")
    snapshot["actors"][0]["availability"] = status
    replace_snapshot(args, manifest, snapshot)
    value = native.load(candidate, **load_args(args))
    assert value["observations"][0]["value_status"] == "unavailable"
    assert value["observations"][0]["value"] is None
    assert value["observations"][0]["last_delivery"] is None


def test_failed_current_read_keeps_old_delivery_without_claiming_current_place(
    tmp_path,
):
    candidate, args, manifest, snapshot = export_fixture(tmp_path / "source")
    snapshot["actors"][0]["place"] = {"status": "read_error", "value": None}
    replace_snapshot(args, manifest, snapshot)
    value = native.load(candidate, **load_args(args))
    assert value["observations"][0]["value_status"] == "read_error"
    assert value["observations"][0]["value"] is None
    assert value["observations"][0]["last_delivery"]["value"] == "BombsiteA"


def test_unavailable_identity_is_handled_without_type_error_or_invented_place():
    candidate = candidate_fixture()
    value = native.unavailable(candidate)
    actor = candidate["snapshot"]["actors"][1]
    actor["identity"] = None
    actor["alive"]["value"] = None
    actor["position"]["value"] = None
    # This unit isolates adapter binding; the full factual validator is called by infer.
    assert native.validate(value, candidate)["status"] == "unavailable"
    snapshot, _ = snapshot_fixture(candidate_fixture(), "d" * 64)
    snapshot["actors"][1]["availability"] = "identity_unavailable"
    provenance = {
        "manifest_sha256": "a" * 64,
        "snapshot_sha256": "b" * 64,
        "targets_sha256": "c" * 64,
        "candidate_context_sha256": "d" * 64,
        "target_id": "control",
        "map_name": "de_mirage",
        "build": None,
        "patch": None,
        "source_semantics": native.SEMANTICS,
    }
    bound = native.build(candidate, snapshot, provenance)
    assert bound["observations"][1]["pawn_handle"] is None
    assert bound["observations"][1]["value"] is None
    snapshot["actors"][1]["availability"] = "observed"
    with pytest.raises(ValueError, match="pawn handle"):
        native.build(candidate, snapshot, provenance)


def test_unavailable_status_cannot_carry_tactical_or_native_assertion():
    candidate = candidate_fixture()
    value = native.unavailable(candidate)
    value["orientation"] = "A"
    with pytest.raises(ValidationError):
        native.validate(value, candidate)
