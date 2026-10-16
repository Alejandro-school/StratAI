import copy
import json
import math

import pytest
from jsonschema import ValidationError

from ai_coach.support import anchors as a
from ai_coach.support import objective as o
from ai_coach.support import visual_contact as s
from ai_coach.tests.test_support_anchors import ACTOR, ALLY, ENEMY, REF
from ai_coach.tests.test_support_objective_clock import scalar
from ai_coach.tests.test_support_weapon_timing import inputs as timing_inputs


def inputs():
    support, _, physical, _, _, _, timing = timing_inputs()
    data = {
        "schema_id": "stratai.support_visual_contact_observations@1",
        "contract_version": "1.0.0-candidate.1",
        **{
            k: support[k] for k in ("source_alias", "source_sha256", "build_identifier")
        },
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
        "references": {**timing["references"], "timing_source": REF},
        "capture_contract": "native_visual_inputs_and_exact_support_cuts@1",
        "geometry": {
            "map_name": "de_mirage",
            "assets": [
                {
                    "reference": "de_mirage/physics.gltf",
                    "sha256": "a" * 64,
                    "role": "physics_gltf",
                },
                {
                    "reference": "de_mirage/physics.bin",
                    "sha256": "b" * 64,
                    "role": "physics_buffer",
                },
            ],
            "raycast_method": "map_mesh_segment_raycast@1",
            "semantic_scope": "static_physics_mesh_eye_segment_diagnostic_only",
            "dynamic_occluders_accredited": False,
            "smoke_volume_accredited": False,
            "rendered_flash_accredited": False,
        },
        "observations": {"deliveries": [], "states": [], "cuts": []},
    }
    obs = data["observations"]
    for i, actor in enumerate(support["observations"]["frames"][0]["players"]):
        fields = {k: scalar() for k in s.PROBES}
        for axis in "XYZ":
            fields["m_vec" + axis] = scalar(
                0 if axis != "Z" else 64, declared="CNetworkedQuantizedFloat"
            )
        for name in ("m_iFOV", "m_iFOVStart"):
            fields["m_pCameraServices." + name] = scalar(90, "uint64", "uint32")
        fields["m_bIsScoped"] = scalar(False, "bool", "bool")
        fields["m_flFlashDuration"] = scalar(0, declared="float32")
        fields["m_flFlashMaxAlpha"] = scalar(255, declared="float32")
        life = f"support-visual-pawn-{i + 1:06d}"
        sequences = {}
        for name, value in fields.items():
            obs["deliveries"].append(
                {
                    "sequence": len(obs["deliveries"]) + 1,
                    "frame": 99,
                    "tick": 199,
                    "entity_life": life,
                    "kind": "created",
                    "property": name,
                    "previous": scalar(declared=value["declared_type"]),
                    "value": copy.deepcopy(value),
                }
            )
            sequences[name] = len(obs["deliveries"])
        obs["states"].append(
            {
                "sequence": i + 1,
                "first_cut": 1,
                "instance": 1,
                "epoch": 0,
                "entity_life": life,
                "handle": actor["pawn_handle"],
                "class_name": "CCSPlayerPawn",
                "created_frame": 99,
                "fields": fields,
                "field_sequences": sequences,
            }
        )
    for base, prior in zip(
        support["observations"]["frames"], timing["observations"]["cuts"], strict=True
    ):
        cut = {
            k: prior[k]
            for k in (
                "source_sequence",
                "frame",
                "tick",
                "instance",
                "epoch",
                "clock_order",
            )
        }
        cut.update(
            delivery_sequence=len(obs["deliveries"]),
            state_sequence=len(obs["states"]),
            geometry_loaded=True,
            players=[],
            rays=[],
        )
        for i, actor in enumerate(base["players"]):
            coords = {}
            for axis in "XYZ":
                value = actor["position"][axis.lower()]
                cell = math.floor((value + 16384) / 512)
                coords["CBodyComponent.m_cell" + axis] = scalar(
                    cell, "uint64", "uint16"
                )
                coords["CBodyComponent.m_vec" + axis] = scalar(
                    value - cell * 512 + 16384, declared="CNetworkedQuantizedFloat"
                )
            cut["players"].append(
                {
                    "actor_alias": actor["actor_alias"],
                    "pawn_handle": actor["pawn_handle"],
                    "pawn_life": actor["pawn_life"],
                    "state_sequence": i + 1,
                    "coordinates": coords,
                }
            )
        cut["rays"] = [
            {"observer_index": i, "subject_index": j, "clear": True}
            for i, x in enumerate(base["players"])
            for j, y in enumerate(base["players"])
            if x["side"] != y["side"]
        ]
        obs["cuts"].append(cut)
    return support, physical, timing, data


def anchor(args):
    return next(
        x
        for x in a.select_anchors(
            args[0]["observations"]["frames"], args[0]["source_alias"]
        )[0]
        if x["actor_alias"] == ACTOR and x["ally_alias"] == ALLY
    )


def features(args):
    row = anchor(args)
    return s.initial_features(
        args[0],
        args[2],
        args[3],
        row["anchor_id"],
        ACTOR,
        row["t0_tick"],
        row["t0_frame"],
    )


def replace_field(args, name, value, index=0):
    data = args[-1]["observations"]
    state = data["states"][index]
    state["fields"][name] = copy.deepcopy(value)
    for d in data["deliveries"]:
        if d["entity_life"] == state["entity_life"] and d["property"] == name:
            d["value"] = copy.deepcopy(value)
            d["previous"] = scalar(declared=value["declared_type"])


def build(args):
    selected = a.select_anchors(
        args[0]["observations"]["frames"], args[0]["source_alias"]
    )[0]
    previous = {
        "rows": [
            {**{k: x[k] for k in o.KEYS}, "competitive_round_status": "not_accredited"}
            for x in selected
        ]
    }
    schema = a.schema_validator(s.REPORT).schema
    refs = {k: REF for k in schema["properties"]["references"]["required"]}
    return s.build_report(*args, previous, refs)


def test_native_zero_exact_cut_and_hidden_geometry_never_unlock_contact():
    args = inputs()
    result = features(args)
    assert result["self"]["native_eye_offset"] == o.fact({"x": 0, "y": 0, "z": 64})
    assert result["self"]["reference_fov_degrees"] == o.fact(90)
    for role in result:
        for name in ("flash_clear", "smoke_clear", "visual_contact"):
            assert result[role][name]["status"] == "unavailable"
            assert result[role][name]["value"] is None
    report = build(args)
    assert report["summary"]["eligible_decisions"] == 0
    assert len(report["matrix"]) > 0
    assert all(not x["decision_eligible"] for x in report["rows"])
    assert all(
        x["competitive_round_status"] == "not_accredited" for x in report["rows"]
    )
    assert "actor_alias" not in json.dumps(result)
    assert "subject_index" not in json.dumps(result)


@pytest.mark.parametrize(
    "name,value,reason",
    [
        ("m_pCameraServices.m_iFOV", scalar(), "native_camera_fov_unavailable"),
        (
            "m_pCameraServices.m_iFOV",
            scalar(90, "float32", "float32"),
            "native_camera_fov_unavailable",
        ),
        (
            "m_pCameraServices.m_iFOV",
            scalar(0, "uint64", "uint32"),
            "native_camera_fov_default_or_range_unaccredited",
        ),
        (
            "m_pCameraServices.m_iFOV",
            scalar(180, "uint64", "uint32"),
            "native_camera_fov_default_or_range_unaccredited",
        ),
        (
            "m_pCameraServices.m_iFOV",
            scalar(40, "uint64", "uint32"),
            "native_camera_fov_transition_unaccredited",
        ),
        (
            "m_bIsScoped",
            scalar(True, "bool", "bool"),
            "scoped_optics_or_scope_state_unaccredited",
        ),
        ("m_bIsScoped", scalar(), "scoped_optics_or_scope_state_unaccredited"),
    ],
)
def test_camera_presence_default_range_transition_and_scope(name, value, reason):
    args = inputs()
    replace_field(args, name, value)
    assert features(args)["self"]["reference_fov_degrees"]["reason"] == reason


@pytest.mark.parametrize("duration", [0, 0.001, 2, 20])
def test_flash_duration_and_max_alpha_never_stand_in_for_rendered_clearance(duration):
    args = inputs()
    replace_field(args, "m_flFlashDuration", scalar(duration, declared="float32"))
    assert features(args)["self"]["flash_clear"] == o.fact(
        reason="rendered_flash_opacity_unaccredited"
    )


def test_geometry_absent_and_native_eye_absent_are_distinct_from_obstruction():
    args = inputs()
    expected = features(args)
    args[-1]["geometry"]["assets"] = []
    for cut in args[-1]["observations"]["cuts"]:
        cut["geometry_loaded"] = False
        for ray in cut["rays"]:
            ray["clear"] = None
    assert features(args) == expected
    replace_field(args, "m_vecZ", scalar())
    assert features(args)["self"]["native_eye_offset"]["value"] is None


@pytest.mark.parametrize(
    "yaw,pitch,target,expected",
    [
        (0, 0, [10, 0, 0], True),
        (0, 0, [-10, 0, 0], False),
        (0, 0, [10, 50, 0], False),
        (90, 0, [0, 10, 0], True),
        (0, 0, [0, 0, 0], False),
        (720, 0, [10, 0, 0], None),
        (0, 100, [10, 0, 0], None),
        (-90, 0, [0, -10, 0], True),
        (0, 315, [10, 0, 10], True),
    ],
)
def test_fov_geometry_zero_and_angular_range(yaw, pitch, target, expected):
    assert (
        s.fov_contains([0, 0, 0], target, o.fact([yaw, pitch]), o.fact(90)) is expected
    )


@pytest.mark.parametrize(
    "mutation",
    [
        "frontier",
        "same_tick_later_frame",
        "clock_order",
        "epoch",
        "instance",
        "serial",
        "life",
        "creation",
        "previous",
        "field_sequence",
        "destroyed",
        "coordinates",
        "ray_target",
        "ray_missing",
        "future_state",
        "false_geometry_proof",
    ],
)
def test_forged_causal_identity_or_physics_provenance_is_rejected(mutation):
    args = inputs()
    obs = args[-1]["observations"]
    cut, state = obs["cuts"][2], obs["states"][0]
    if mutation == "frontier":
        cut["delivery_sequence"] -= 1
    elif mutation == "same_tick_later_frame":
        cut["frame"] += 1
    elif mutation == "clock_order":
        cut["clock_order"] += 1
    elif mutation in ("epoch", "instance"):
        state[mutation] += 1
    elif mutation == "serial":
        state["handle"] += 16384
    elif mutation == "life":
        cut["players"][0]["pawn_life"] = "support-pawn-999999"
    elif mutation == "creation":
        state["created_frame"] += 1
    elif mutation == "previous":
        obs["deliveries"][0]["previous"] = scalar(3)
    elif mutation == "field_sequence":
        state["field_sequences"]["m_vecX"] += 1
    elif mutation == "destroyed":
        obs["deliveries"].append(
            {
                "sequence": len(obs["deliveries"]) + 1,
                "frame": 100,
                "tick": 200,
                "entity_life": state["entity_life"],
                "kind": "destroyed",
                "property": "",
                "value": scalar(),
                "previous": scalar(),
            }
        )
        for c in obs["cuts"]:
            c["delivery_sequence"] += 1
    elif mutation == "coordinates":
        cut["players"][0]["coordinates"]["CBodyComponent.m_vecX"]["value"] += 10
    elif mutation == "ray_target":
        cut["rays"][0]["subject_index"] = 0
    elif mutation == "ray_missing":
        cut["rays"].pop()
    elif mutation == "future_state":
        state["first_cut"] = 999
    elif mutation == "false_geometry_proof":
        args[-1]["geometry"]["smoke_volume_accredited"] = True
    with pytest.raises((ValueError, ValidationError)):
        features(args)


def test_hidden_enemy_and_malformed_future_do_not_change_initial_information():
    args = inputs()
    expected = features(args)
    enemy = next(
        i
        for i, x in enumerate(args[0]["observations"]["frames"][0]["players"])
        if x["actor_alias"] == ENEMY
    )
    replace_field(args, "m_flFlashDuration", scalar(8, declared="float32"), enemy)
    for cut in args[-1]["observations"]["cuts"]:
        for ray in cut["rays"]:
            ray["clear"] = False
    seq = anchor(args)["source_sequence"] + 1
    args[-1]["observations"]["cuts"].append(
        {"source_sequence": seq, "bad_future": True}
    )
    args[-1]["observations"]["states"].append({"sequence": 99999, "bad_future": True})
    args[-1]["observations"]["deliveries"].append(
        {"sequence": 99999, "bad_future": True}
    )
    assert features(args) == expected


def test_schema_rejects_fabricated_contact_and_reader_recomputes(tmp_path, monkeypatch):
    args = inputs()
    report = build(args)
    bad = copy.deepcopy(report)
    bad["rows"][0]["initial"]["self"]["visual_contact"] = o.fact(True)
    with pytest.raises(ValidationError):
        a.schema_validator(s.REPORT).validate(bad)
    monkeypatch.setattr(s.t, "validate_artifact", lambda *args: None)
    monkeypatch.setattr(s, "verify_physics", lambda *args: {})
    refs = report["references"]
    previous = {"rows": report["rows"]}
    monkeypatch.setattr(s, "report_inputs", lambda *unused: (*args, previous, refs))
    path = tmp_path / s.REPORT
    path.write_text(json.dumps(report))
    assert (
        s.validate_artifact(tmp_path, tmp_path / s.DATA, None, None, None)
        == report["summary"]
    )
    report["summary"]["anchors"] += 1
    path.write_text(json.dumps(report))
    with pytest.raises(ValueError, match="recomputation"):
        s.validate_artifact(tmp_path, tmp_path / s.DATA, None, None, None)


def test_materialization_rejects_nonempty_before_source_read(tmp_path):
    (tmp_path / "keep").write_text("retained")
    with pytest.raises(ValueError):
        s.materialize(tmp_path / s.DATA, tmp_path, None, None, None)
