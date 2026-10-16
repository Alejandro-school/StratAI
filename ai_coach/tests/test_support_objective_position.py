import copy
import json

import pytest
from jsonschema import ValidationError

from ai_coach.support import anchors as a
from ai_coach.support import objective as o
from ai_coach.support import objective_position as s
from ai_coach.tests.test_support_anchors import ACTOR, ALLY, REF
from ai_coach.tests.test_support_attempt_origin import build as attempt_report
from ai_coach.tests.test_support_attempt_origin import inputs as attempt_inputs
from ai_coach.tests.test_support_c4 import ref
from ai_coach.tests.test_support_objective import build as objective_report
from ai_coach.tests.test_support_objective_clock import planted, scalar


def inputs(defusing=False, extra_bomb=False):
    args = attempt_inputs()
    if defusing:
        planted(args[:4])
    if extra_bomb:
        for row in args[3]["observations"]["frames"]:
            hidden = copy.deepcopy(row["bombs"][0])
            hidden["entity_life"], hidden["handle"] = "support-clock-object-000003", 42
            enemy = row["players"][2]
            hidden["defuser"] = ref(enemy["pawn_handle"], enemy["pawn_life"])
            row["bombs"].append(hidden)
    return [*args, capture_source(args)]


def capture_source(args, birth_frames=None, owners=None):
    birth_frames, owners = birth_frames or {}, owners or {}
    support, _objective, ownership, clock, attempt = args
    native = {}
    for row in clock["observations"]["frames"]:
        for old in [*row["items"], *row["bombs"]]:
            native.setdefault(old["entity_life"], old)
    objects = {}
    for old in native.values():
        fields = {k: scalar() for k in s.PROBES}
        for axis in s.AXES:
            fields[s.BODY + "m_cell" + axis] = scalar(32, "uint64", "uint16")
            fields[s.BODY + "m_vec" + axis] = scalar(
                0, "float32", "CNetworkedQuantizedFloat"
            )
        fields[s.BODY + "m_hParent"] = scalar(
            16777215, "uint64", "CGameSceneNodeHandle"
        )
        owner, defuser = (
            copy.deepcopy(
                owners.get(old["entity_life"], ref(16777215, status="invalid_sentinel"))
            ),
            ref(),
        )
        if old["class_name"] == "CC4":
            original = next(
                x
                for row in ownership["observations"]["states"]
                for x in row["entities"]
                if x["entity_life"] == old["entity_life"]
            )
            owner = copy.deepcopy(original["owner"])
            fields["m_iItemDefinitionIndex"] = scalar(49, "uint64", "uint16")
        else:
            defuser = copy.deepcopy(old["defuser"])
        fields["m_hOwnerEntity"] = scalar(
            owner["handle"], "uint64", "CHandle< C_BaseEntity >"
        )
        if defuser["handle"] is not None:
            fields["m_hBombDefuser"] = scalar(
                defuser["handle"], "uint64", "CHandle< CCSPlayerPawn >"
            )
        fields.update(
            {k: copy.deepcopy(v) for k, v in old["fields"].items() if k in fields}
        )
        objects[old["entity_life"]] = {
            "entity_life": old["entity_life"],
            "handle": old["handle"],
            "class_name": old["class_name"],
            "created_frame": birth_frames.get(old["entity_life"], 99),
            "fields": fields,
            "field_sequences": {},
            "owner": owner,
            "defuser": defuser,
            "parent": ref(16777215, status="invalid_sentinel"),
        }
    events = []
    for obj in objects.values():
        for name, value in obj["fields"].items():
            events.append(
                (
                    obj["created_frame"],
                    0,
                    obj["entity_life"],
                    "created",
                    name,
                    copy.deepcopy(value),
                )
            )
    for d in attempt["observations"]["deliveries"]:
        if d["kind"] != "created" and (
            d["property"] in s.PROBES or d["kind"] == "destroyed"
        ):
            events.append(
                (
                    d["frame"],
                    1,
                    d["entity_life"],
                    d["kind"],
                    d["property"],
                    copy.deepcopy(d["value"]),
                )
            )
    events.sort(key=lambda x: (x[0], x[1]))
    clock_by_frame = {x["frame"]: x for x in clock["observations"]["frames"]}
    deliveries, births, states, cuts = [], [], [], []
    cursor = 0
    for frame in sorted(
        set(clock_by_frame) | {x["created_frame"] for x in objects.values()}
    ):
        while cursor < len(events) and events[cursor][0] <= frame:
            at, _, life, kind, name, value = events[cursor]
            obj = objects[life]
            previous = (
                scalar(declared=value["declared_type"])
                if kind in ("created", "destroyed")
                else copy.deepcopy(obj["fields"][name])
            )
            d = {
                "sequence": len(deliveries) + 1,
                "frame": at,
                "tick": at + 100,
                "entity_life": life,
                "kind": kind,
                "property": name,
                "previous": previous,
                "value": value,
            }
            deliveries.append(d)
            if kind != "destroyed":
                obj["fields"][name] = copy.deepcopy(value)
                obj["field_sequences"][name] = d["sequence"]
            cursor += 1
        for obj in objects.values():
            if obj["created_frame"] == frame:
                births.append(
                    {
                        "sequence": len(births) + 1,
                        "frame": frame,
                        "tick": frame + 100,
                        "instance": 1,
                        "epoch": 0,
                        "delivery_sequence": len(deliveries),
                        "object": copy.deepcopy(obj),
                    }
                )
        if frame in clock_by_frame:
            row = clock_by_frame[frame]
            seq = row["source_sequence"]
            current = [
                objects[x["entity_life"]] for x in [*row["items"], *row["bombs"]]
            ]
            states.append(
                {"sequence": seq, "first_cut": seq, "objects": copy.deepcopy(current)}
            )
            cuts.append(
                {
                    "source_sequence": seq,
                    **{k: row[k] for k in ("frame", "tick", "instance", "epoch")},
                    "clock_order": row["order"],
                    "state_sequence": seq,
                    "delivery_sequence": len(deliveries),
                    "birth_sequence": len(births),
                }
            )
    return {
        "schema_id": "stratai.support_objective_position_observations@1",
        **{
            k: support[k]
            for k in (
                "contract_version",
                "source_alias",
                "source_sha256",
                "build_identifier",
                "comparison_allowed",
                "training_allowed",
                "product_supported",
            )
        },
        "references": {**attempt["references"], "attempt_source": REF},
        "capture_contract": "native_object_geometry_deliveries_births_and_support_cuts@1",
        "observations": {
            "deliveries": deliveries,
            "births": births,
            "states": states,
            "cuts": cuts,
        },
    }


def completed_plant_inputs():
    args = attempt_inputs()
    _, objective, ownership, clock, attempt = args
    # A short fixture duration tests the measured native deadline, never a
    # conventional planting duration supplied by the implementation.
    deadline = 6604 / 64
    for d in attempt["observations"]["deliveries"]:
        if d["property"] == "m_fArmedTime" and d["kind"] == "update":
            d["value"] = scalar(deadline, declared="GameTime_t")
    attempt["observations"]["deliveries"].append(
        {
            "sequence": 6,
            "frame": 104,
            "tick": 204,
            "entity_life": "support-item-000001",
            "kind": "destroyed",
            "property": "",
            "previous": scalar(),
            "value": scalar(),
        }
    )
    attempt["observations"]["frames"] = [
        x for x in attempt["observations"]["frames"] if x["frame"] <= 104
    ]
    for frame in attempt["observations"]["frames"]:
        if frame["frame"] >= 102:
            frame["items"][0]["fields"]["m_fArmedTime"] = scalar(
                deadline, declared="GameTime_t"
            )
        if frame["frame"] == 104:
            frame["items"][0]["current"] = False
            frame["delivery_sequence"] = 6
    for cut in attempt["observations"]["cuts"]:
        available = [
            x for x in attempt["observations"]["frames"] if x["frame"] <= cut["frame"]
        ]
        cut["frame_sequence"], cut["delivery_sequence"] = (
            len(available),
            available[-1]["delivery_sequence"],
        )
    planted(args[:4], active=False)
    objective["observations"]["events"][-1].update(frame=104, tick=204, order=15)
    owner = copy.deepcopy(
        ownership["observations"]["states"][0]["entities"][0]["owner"]
    )
    for row, of, state in zip(
        clock["observations"]["frames"],
        objective["observations"]["frames"],
        ownership["observations"]["states"],
        strict=True,
    ):
        if row["frame"] < 104:
            of["event_sequence"] = 1
            of["rules"]["m_bBombPlanted"]["value"] = False
            row["bombs"] = []
        else:
            row["items"] = []
            row["bombs"][0]["defuser"] = ref(16777215, status="invalid_sentinel")
            state["entities"] = []
            state["players"][0]["slots"] = []
            state["players"][0]["inventory_length"] = 0
    data = capture_source(
        args,
        {"support-clock-object-000002": 104},
        {"support-clock-object-000002": owner},
    )
    return [*args, data]


def features(args, actor=ACTOR):
    anchor = next(
        x
        for x in a.select_anchors(
            args[0]["observations"]["frames"], args[0]["source_alias"]
        )[0]
        if x["actor_alias"] == actor
    )
    return s.initial_features(
        *args, anchor["anchor_id"], actor, anchor["t0_tick"], anchor["t0_frame"]
    )


def report_args(args):
    previous = attempt_report(args[:5])
    public = objective_report(args[0], args[1])
    return [
        *args,
        previous,
        public,
        {**previous["references"], "position_source": REF, "attempt_context": REF},
    ]


def item(args, life="support-item-000001"):
    return next(
        x
        for x in args[5]["observations"]["states"][2]["objects"]
        if x["entity_life"] == life
    )


def replace_field(data, life, name, value):
    for d in data["observations"]["deliveries"]:
        if d["entity_life"] == life and d["property"] == name:
            d["value"] = copy.deepcopy(value)
            if d["kind"] == "created":
                d["previous"] = scalar(declared=value["declared_type"])
            else:
                d["previous"] = copy.deepcopy(value)
    for row in data["observations"]["births"]:
        if row["object"]["entity_life"] == life:
            row["object"]["fields"][name] = copy.deepcopy(value)
    for row in data["observations"]["states"]:
        for obj in row["objects"]:
            if obj["entity_life"] == life:
                obj["fields"][name] = copy.deepcopy(value)


def test_root_zero_is_native_object_position_with_bilateral_own_possession():
    args = inputs()
    for row in args[0]["observations"]["frames"]:
        next(x for x in row["players"] if x["actor_alias"] == ACTOR)["position"][
            "x"
        ] = 13
    result = features(args)
    assert result["self"]["physical_objective_position"] == o.fact(
        {"x": 0, "y": 0, "z": 0}
    )
    assert result["self"]["position_availability"] == o.fact("current_owned_c4")
    assert result["ally"]["physical_objective_position"]["status"] == "unavailable"
    assert features(args, ALLY)["ally"] == result["self"]
    carrier = args[0]["observations"]["frames"][2]["players"][0]["position"]
    assert carrier != result["self"]["physical_objective_position"]["value"]


@pytest.mark.parametrize(
    "name,value,reason",
    [
        (s.BODY + "m_cellX", scalar(), "native_coordinates_unavailable"),
        (
            s.BODY + "m_cellX",
            scalar(32, "float32", "uint16"),
            "native_coordinates_unavailable",
        ),
        (
            s.BODY + "m_cellX",
            scalar(65536, "uint64", "uint16"),
            "native_coordinates_unavailable",
        ),
        (
            s.BODY + "m_cellX",
            scalar(32, "uint64", "other"),
            "native_coordinates_unavailable",
        ),
        (s.BODY + "m_vecX", scalar(True), "native_coordinates_unavailable"),
        (s.BODY + "m_hParent", scalar(), "native_parent_unavailable"),
        (
            s.BODY + "m_hParent",
            scalar(0, "uint64", "CGameSceneNodeHandle"),
            "parent_transform_unavailable",
        ),
        (
            s.BODY + "m_hParent",
            scalar(49157, "uint64", "CGameSceneNodeHandle"),
            "parent_transform_unavailable",
        ),
        (
            s.BODY + "m_hParent",
            scalar(16777215, "uint64", "other"),
            "native_parent_unavailable",
        ),
        ("m_hMoveParent", scalar(49157, "uint64"), "native_parent_attachment_conflict"),
        (
            s.BODY + "m_hierarchyAttachName",
            scalar(973653956, "uint64", "CUtlStringToken"),
            "native_parent_attachment_conflict",
        ),
        (
            "m_iParentAttachment",
            scalar(1, "uint64"),
            "native_parent_attachment_conflict",
        ),
        (
            "m_vecAbsOrigin",
            scalar([1, 0, 0], "[]float32", "Vector"),
            "native_absolute_origin_conflict",
        ),
    ],
)
def test_geometry_requires_present_typed_coordinates_and_parent(name, value, reason):
    obj = item(inputs())
    obj["fields"][name] = value
    assert s.geometry(obj)["world_position"] == o.fact(reason=reason)


def test_cell_formula_and_matching_absolute_native_origin():
    obj = item(inputs())
    obj["fields"][s.BODY + "m_cellX"] = scalar(33, "uint64", "uint16")
    obj["fields"][s.BODY + "m_vecX"] = scalar(
        0.25, "float32", "CNetworkedQuantizedFloat"
    )
    obj["fields"]["m_vecAbsOrigin"] = scalar([512.25, 0, 0], "[]float32", "Vector")
    assert s.geometry(obj)["world_position"] == o.fact({"x": 512.25, "y": 0, "z": 0})


@pytest.mark.parametrize("native_type", ["[3]float32", "[3]float64"])
def test_fixed_native_rotation_is_retained_and_validated(native_type):
    args = inputs()
    replace_field(
        args[5],
        "support-item-000001",
        s.BODY + "m_angRotation",
        scalar([0, 45, 0], native_type, "QAngle"),
    )
    s.validate_source(args[5], args[0], args[2], args[3])
    assert features(args)["self"]["physical_objective_position"]["status"] == "observed"


@pytest.mark.parametrize("offset", [-1, 513, 864.03125, 1023.75])
def test_native_offset_is_not_artificially_bounded_by_cell_stride(offset):
    obj = item(inputs())
    obj["fields"][s.BODY + "m_vecX"] = scalar(
        offset, "float32", "CNetworkedQuantizedFloat"
    )
    assert s.geometry(obj)["world_position"] == o.fact({"x": offset, "y": 0, "z": 0})


def test_native_unowned_geometry_does_not_claim_legal_dropped_position():
    args = inputs()
    obj = item(args)
    obj["owner"] = ref(16777215, status="invalid_sentinel")
    obj["fields"]["m_hOwnerEntity"] = scalar(16777215, "uint64")
    assert s.geometry(obj)["native_object_state"] == o.fact("unowned")
    anchor = a.select_anchors(
        args[0]["observations"]["frames"], args[0]["source_alias"]
    )[0][0]
    public = o.initial_features(
        args[0],
        args[1],
        anchor["anchor_id"],
        ACTOR,
        anchor["t0_tick"],
        anchor["t0_frame"],
    )
    result, proof = s.initial_context(
        anchor,
        public,
        args[2]["observations"]["states"][2],
        {"objects": [obj]},
        args[5]["observations"]["cuts"][2],
        {},
        {},
    )
    assert (
        result["self"]["physical_objective_position"]["reason"]
        == "legal_object_position_not_observed"
        and proof["self"] is None
    )


def test_current_defuser_contact_accredits_only_its_object_and_roles():
    args = inputs(defusing=True)
    result = features(args)
    assert result["self"]["position_availability"] == o.fact("current_defuse_contact")
    assert result["self"]["objective_object_state"] == o.fact("planted")
    assert result["ally"]["physical_objective_position"]["status"] == "unavailable"
    assert features(args, ALLY)["ally"] == result["self"]


def test_global_plant_and_spotted_do_not_grant_exact_position():
    args = inputs(defusing=True)
    for row in args[1]["observations"]["frames"]:
        row["players"][0]["fields"]["m_bIsDefusing"]["value"] = False
    replace_field(
        args[5],
        "support-clock-object-000002",
        "m_entitySpottedState.m_bSpotted",
        scalar(True),
    )
    assert (
        features(args)["self"]["physical_objective_position"]["reason"]
        == "legal_object_position_not_observed"
    )


def test_undecodable_body_update_cannot_preserve_stationary_knowledge():
    unknown = {
        "status": "unsupported_type",
        "native_type": "[]interface {}",
        "declared_type": "CUtlVector<Animation>",
        "value": None,
    }
    data = {
        "observations": {
            "deliveries": [
                {
                    "kind": "update",
                    "sequence": 1,
                    "entity_life": "support-clock-object-000002",
                    "property": s.BODY + "m_animationCollection",
                    "previous": unknown,
                    "value": unknown,
                }
            ]
        }
    }
    assert s.position_changes(data) == {"support-clock-object-000002": [1]}


def test_unrelated_native_object_geometry_is_not_initial_information():
    args = inputs(defusing=True, extra_bomb=True)
    before = features(args)
    replace_field(
        args[5],
        "support-clock-object-000003",
        s.BODY + "m_vecX",
        scalar(511, "float32", "CNetworkedQuantizedFloat"),
    )
    assert features(args) == before
    assert not any(
        k in json.dumps(before)
        for k in (
            "support-actor-",
            "support-pawn-",
            "support-clock-object-",
            "entity_life",
            "handle",
            "source_sequence",
        )
    )


@pytest.mark.parametrize("variant", ["future", "same_frame_future"])
def test_prefix_excludes_even_malformed_future_deliveries_births_and_states(variant):
    args = inputs()
    before = features(args)
    obs = args[5]["observations"]
    obs["cuts"].append(
        {
            "source_sequence": 999,
            "frame": 108 if variant == "same_frame_future" else 999,
        }
    )
    for name in ("deliveries", "births", "states"):
        obs[name].append({"sequence": 99999, "oracle": "malformed future"})
    assert features(args) == before


@pytest.mark.parametrize(
    "variant",
    [
        "predecessor",
        "future_snapshot",
        "frontier",
        "birth_frame",
        "reused_life",
        "census",
        "owner_handle",
        "parent_handle",
        "state_epoch",
        "missing_cut",
    ],
)
def test_source_rejects_causality_identity_and_reference_corruption(variant):
    args = inputs()
    obs = args[5]["observations"]
    if variant == "predecessor":
        obs["deliveries"][-1]["previous"]["value"] = 1
    elif variant == "future_snapshot":
        obs["states"][0]["objects"][0]["field_sequences"]["m_bStartedArming"] = len(
            obs["deliveries"]
        )
    elif variant == "frontier":
        obs["cuts"][0]["delivery_sequence"] += 1
    elif variant == "birth_frame":
        obs["births"][0]["frame"] += 1
    elif variant == "reused_life":
        obs["states"][2]["objects"][0]["handle"] += 16384
    elif variant == "census":
        obs["states"][2]["objects"] = []
    elif variant == "owner_handle":
        obs["states"][2]["objects"][0]["owner"]["handle"] += 16384
    elif variant == "parent_handle":
        obs["states"][2]["objects"][0]["parent"]["handle"] = 0
    elif variant == "state_epoch":
        obs["cuts"][2]["epoch"] += 1
    else:
        obs["cuts"].pop(2)
    with pytest.raises((ValueError, ValidationError)):
        s.validate_source(args[5], args[0], args[2], args[3])


def test_report_preserves_denominators_and_separates_native_geometry():
    args = inputs(defusing=True)
    report = s.build_report(*report_args(args))
    assert report["rows"][0]["initial"] == features(args)
    assert (
        report["summary"]["anchors"] == 2
        and report["summary"]["eligible_decisions"] == 0
    )
    assert all(
        sum(v.values()) == 2 for v in report["summary"]["fact_status_counts"].values()
    )
    assert (
        report["native_geometry"]
        and report["native_geometry_is_initial_information"] is False
    )


def test_reader_independently_rebuilds_changed_position(tmp_path, monkeypatch):
    args = report_args(inputs())
    report = s.build_report(*args)
    monkeypatch.setattr(s.r, "validate_artifact", lambda *_: None)
    monkeypatch.setattr(s, "report_inputs", lambda *_: args)
    target = tmp_path / s.REPORT
    target.write_text(json.dumps(report))
    assert (
        s.validate_artifact(tmp_path, tmp_path / s.DATA, tmp_path, tmp_path)
        == report["summary"]
    )
    report["rows"][0]["initial"]["self"]["physical_objective_position"]["value"][
        "x"
    ] += 1
    target.write_text(json.dumps(report))
    with pytest.raises(ValueError, match="recomputation"):
        s.validate_artifact(tmp_path, tmp_path / s.DATA, tmp_path, tmp_path)


def test_exact_native_completion_accredits_stationary_plant_position():
    args = completed_plant_inputs()
    result = features(args)
    assert result["self"]["position_availability"] == o.fact(
        "witnessed_stationary_plant"
    )
    assert result["ally"]["physical_objective_position"]["status"] == "unavailable"
    report = s.build_report(*report_args(args))
    proof = report["rows"][0]["position_reference"]["self"]
    assert proof["availability_frame"] == 104 and proof["birth_sequence"] == 2
    assert (
        proof["frame"] == 108 and proof["entity_life"] == "support-clock-object-000002"
    )


@pytest.mark.parametrize(
    "variant",
    [
        "missing_public_event",
        "missing_destruction",
        "previous_frame_gap",
        "wrong_owner",
        "ambiguous_creation",
        "clock_paused",
        "clock_absent",
        "clock_domain_changed",
        "before_deadline",
    ],
)
def test_plant_position_witness_requires_positive_completion_and_native_clock(variant):
    args = completed_plant_inputs()
    if variant == "missing_public_event":
        args[1]["observations"]["events"].pop()
        for row in args[1]["observations"]["frames"]:
            row["event_sequence"] = min(
                row["event_sequence"], len(args[1]["observations"]["events"])
            )
    elif variant == "missing_destruction":
        args[4]["observations"]["deliveries"].pop()
    elif variant == "previous_frame_gap":
        args[4]["observations"]["frames"] = [
            x for x in args[4]["observations"]["frames"] if x["frame"] != 103
        ]
    elif variant == "wrong_owner":
        args[5]["observations"]["births"][-1]["object"]["owner"] = ref(
            16386, "support-pawn-000002"
        )
    elif variant == "ambiguous_creation":
        args[5]["observations"]["births"].append(
            copy.deepcopy(args[5]["observations"]["births"][-1])
        )
    else:
        row = args[4]["observations"]["frames"][-1]
        if variant == "clock_paused":
            row["rules"]["fields"]["m_bGamePaused"] = scalar(True)
        elif variant == "clock_absent":
            row["network"]["tick"] = []
        elif variant == "clock_domain_changed":
            row["interval"]["order"] += 1
        else:
            for kind in ("tick", "packet_tick"):
                row["network"][kind][0]["value"]["value"] -= 1
    assert not s.planting_witnesses(*args)


def test_observed_plant_completion_may_follow_the_native_deadline():
    args = completed_plant_inputs()
    current = args[4]["observations"]["frames"][-1]
    for kind in ("tick", "packet_tick"):
        current["network"][kind][0]["value"]["value"] += 8
    assert len(s.planting_witnesses(*args)) == 1


def move_away_and_back(args, frame):
    data = args[5]["observations"]
    life, name = "support-clock-object-000002", s.BODY + "m_vecX"
    before = copy.deepcopy(item(args, life)["fields"][name])
    for at, value in ((frame, 1), (frame + 1, 0)):
        after = scalar(value, "float32", "CNetworkedQuantizedFloat")
        data["deliveries"].append(
            {
                "sequence": len(data["deliveries"]) + 1,
                "frame": at,
                "tick": at + 100,
                "entity_life": life,
                "kind": "update",
                "property": name,
                "previous": before,
                "value": after,
            }
        )
        before = copy.deepcopy(after)
    for cut, state in zip(data["cuts"], data["states"], strict=True):
        if cut["frame"] >= frame + 1:
            cut["delivery_sequence"] = len(data["deliveries"])
            obj = next(x for x in state["objects"] if x["entity_life"] == life)
            obj["fields"][name], obj["field_sequences"][name] = (
                copy.deepcopy(before),
                len(data["deliveries"]),
            )


def test_returning_to_same_position_does_not_repair_lost_plant_witness():
    args = completed_plant_inputs()
    move_away_and_back(args, 105)
    assert s.geometry(item(args, "support-clock-object-000002"))[
        "world_position"
    ] == o.fact({"x": 0, "y": 0, "z": 0})
    assert (
        features(args)["self"]["physical_objective_position"]["reason"]
        == "legal_object_position_not_observed"
    )


def test_later_motion_does_not_retroactively_remove_legal_plant_position():
    args = completed_plant_inputs()
    before = features(args)
    move_away_and_back(args, 109)
    s.validate_source(args[5], args[0], args[2], args[3])
    assert features(args) == before


@pytest.mark.parametrize("variant", ["new_pawn", "new_epoch", "new_instance"])
def test_stationary_plant_witness_does_not_cross_pawn_or_domain(variant):
    args = completed_plant_inputs()
    anchor = a.select_anchors(
        args[0]["observations"]["frames"], args[0]["source_alias"]
    )[0][0]
    public = o.initial_features(
        args[0],
        args[1],
        anchor["anchor_id"],
        ACTOR,
        anchor["t0_tick"],
        anchor["t0_frame"],
    )
    owner, cut = (
        copy.deepcopy(args[2]["observations"]["states"][2]),
        copy.deepcopy(args[5]["observations"]["cuts"][2]),
    )
    if variant == "new_pawn":
        owner["players"][0]["pawn_life"] = "support-pawn-000099"
    elif variant == "new_epoch":
        cut["epoch"] += 1
    else:
        cut["instance"] += 1
    result, _ = s.initial_context(
        anchor,
        public,
        owner,
        args[5]["observations"]["states"][2],
        cut,
        s.planting_witnesses(*args),
        s.position_changes(args[5]),
    )
    assert result["self"]["physical_objective_position"]["status"] == "unavailable"


def test_direct_defuse_can_reacquire_position_after_stationary_witness_is_lost():
    args = completed_plant_inputs()
    move_away_and_back(args, 105)
    anchor = a.select_anchors(
        args[0]["observations"]["frames"], args[0]["source_alias"]
    )[0][0]
    public = o.initial_features(
        args[0],
        args[1],
        anchor["anchor_id"],
        ACTOR,
        anchor["t0_tick"],
        anchor["t0_frame"],
    )
    public["self"]["defusing"] = o.fact(True)
    owner = args[2]["observations"]["states"][2]
    state = copy.deepcopy(args[5]["observations"]["states"][2])
    state["objects"][0]["defuser"] = ref(
        owner["players"][0]["pawn_handle"], owner["players"][0]["pawn_life"]
    )
    result, _ = s.initial_context(
        anchor,
        public,
        owner,
        state,
        args[5]["observations"]["cuts"][2],
        s.planting_witnesses(*args),
        s.position_changes(args[5]),
    )
    assert result["self"]["position_availability"] == o.fact("current_defuse_contact")
