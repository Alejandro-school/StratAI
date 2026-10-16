import copy
import json

import pytest
from jsonschema import ValidationError

from ai_coach.support import anchors as a
from ai_coach.support import native_motion as s
from ai_coach.support import objective as o
from ai_coach.tests.test_support_anchors import ACTOR, ALLY, REF
from ai_coach.tests.test_support_objective_clock import scalar
from ai_coach.tests.test_support_objective_position import inputs as position_inputs


def reference(handle=None, life=None, status=None, class_name=None):
    return {
        "status": status or ("resolved" if life else "absent"),
        "handle": handle,
        "target_life": life,
        "class_name": class_name,
    }


def inputs():
    old = position_inputs()
    support, position = old[0], old[-1]
    states, deliveries, cuts, by_actor = [], [], [], {}
    for base, previous in zip(
        support["observations"]["frames"], position["observations"]["cuts"], strict=True
    ):
        players = []
        for actor in base["players"]:
            alias = actor["actor_alias"]
            if alias not in by_actor:
                life = f"support-motion-pawn-{len(by_actor) + 1:06d}"
                fields = {name: scalar() for name in s.PROBES}
                fields["m_vecAbsVelocity"] = scalar(
                    [3, 4, 12], "[3]float32", "VectorWS"
                )
                fields["m_fFlags"] = scalar(1, "uint64", "uint32")
                fields["m_hGroundEntity"] = scalar(
                    0, "uint64", "CHandle< CBaseEntity >"
                )
                fields["m_MoveType"] = scalar(2, "uint64", "MoveType_t")
                fields["CBodyComponent.m_hParent"] = scalar(
                    16777215, "uint64", "CGameSceneNodeHandle"
                )
                seqs = {}
                for name, value in fields.items():
                    d = {
                        "sequence": len(deliveries) + 1,
                        "frame": 99,
                        "tick": 199,
                        "entity_life": life,
                        "kind": "created",
                        "property": name,
                        "previous": scalar(declared=value["declared_type"]),
                        "value": copy.deepcopy(value),
                    }
                    deliveries.append(d)
                    seqs[name] = d["sequence"]
                state = {
                    "sequence": len(states) + 1,
                    "first_cut": base["sequence"],
                    "instance": base["instance"],
                    "epoch": base["epoch"],
                    "entity_life": life,
                    "handle": actor["pawn_handle"],
                    "class_name": "CCSPlayerPawn",
                    "created_frame": 99,
                    "fields": fields,
                    "field_sequences": seqs,
                    "ground": reference(
                        0, "support-motion-reference-000001", class_name="CWorld"
                    ),
                    "parent": reference(16777215, status="invalid_sentinel"),
                }
                states.append(state)
                by_actor[alias] = state
            state = by_actor[alias]
            players.append(
                {
                    "actor_alias": alias,
                    "pawn_handle": actor["pawn_handle"],
                    "pawn_life": actor["pawn_life"],
                    "state_sequence": state["sequence"],
                }
            )
        cuts.append(
            {
                **{
                    k: previous[k]
                    for k in (
                        "source_sequence",
                        "frame",
                        "tick",
                        "instance",
                        "epoch",
                        "clock_order",
                    )
                },
                "delivery_sequence": len(deliveries),
                "state_sequence": len(states),
                "players": players,
            }
        )
    data = {
        "schema_id": "stratai.support_native_motion_observations@1",
        "contract_version": "1.0.0-candidate.1",
        **{
            k: support[k] for k in ("source_alias", "source_sha256", "build_identifier")
        },
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
        "references": {**position["references"], "position_source": REF},
        "capture_contract": "native_pawn_motion_deliveries_and_exact_support_cuts@1",
        "observations": {"deliveries": deliveries, "states": states, "cuts": cuts},
    }
    return [support, position, data]


def actor_state(args, alias=ACTOR):
    player = next(
        x
        for x in args[2]["observations"]["cuts"][0]["players"]
        if x["actor_alias"] == alias
    )
    return args[2]["observations"]["states"][player["state_sequence"] - 1]


def replace_field(args, name, value, alias=ACTOR):
    state = actor_state(args, alias)
    state["fields"][name] = copy.deepcopy(value)
    for d in args[2]["observations"]["deliveries"]:
        if d["entity_life"] == state["entity_life"] and d["property"] == name:
            d["value"] = copy.deepcopy(value)
            d["previous"] = scalar(declared=value["declared_type"])


def features(args):
    anchor = next(
        x
        for x in a.select_anchors(
            args[0]["observations"]["frames"], args[0]["source_alias"]
        )[0]
        if x["actor_alias"] == ACTOR and x["ally_alias"] == ALLY
    )
    return s.initial_features(
        *args,
        anchor["anchor_id"],
        anchor["actor_alias"],
        anchor["t0_tick"],
        anchor["t0_frame"],
    )


def report_args(args):
    selected = a.select_anchors(
        args[0]["observations"]["frames"], args[0]["source_alias"]
    )[0]
    previous = {
        "rows": [
            {
                **{k: x[k] for k in o.KEYS},
                "competitive_round_status": "canonical_interval_matched",
            }
            for x in selected
        ]
    }
    schema = a.schema_validator(s.REPORT).schema
    refs = {k: REF for k in schema["properties"]["references"]["required"]}
    return [*args, previous, refs]


def test_native_world_vector_and_ground_are_independent_of_mean_motion():
    args = inputs()
    actual = features(args)["self"]
    assert actual["native_velocity"]["value"] == {"x": 3, "y": 4, "z": 12}
    assert actual["native_horizontal_speed"]["value"] == 5
    assert actual["native_speed"]["value"] == 13
    assert actual["ground_contact"]["value"] is True
    assert actual["ground_reference_state"]["value"] == "world"
    assert actual["movement_mode_code"]["value"] == 2


@pytest.mark.parametrize(
    "native", ["[3]float32", "[3]float64", "[]float32", "[]float64"]
)
def test_present_native_zero_vector_is_distinct_from_absence(native):
    args = inputs()
    replace_field(args, "m_vecAbsVelocity", scalar([0, 0, 0], native, "VectorWS"))
    assert features(args)["self"]["native_speed"]["value"] == 0
    replace_field(args, "m_vecAbsVelocity", scalar())
    assert (
        features(args)["self"]["native_speed"]["reason"]
        == "native_current_velocity_source_absent"
    )


@pytest.mark.parametrize(
    "substitute",
    [
        "m_vecBaseVelocity",
        "m_pMovementServices.m_flFallVelocity",
        "m_pMovementServices.m_flLastLandedVelocityX",
        "m_pMovementServices.m_flLastJumpVelocityZ",
    ],
)
def test_base_and_historical_values_never_replace_current_velocity(substitute):
    args = inputs()
    replace_field(args, "m_vecAbsVelocity", scalar())
    replace_field(
        args,
        substitute,
        scalar([300, 0, 0], "[3]float32", "Vector")
        if substitute == "m_vecBaseVelocity"
        else scalar(300, "float32", "float32"),
    )
    assert (
        features(args)["self"]["native_velocity"]["reason"]
        == "native_current_velocity_source_absent"
    )


@pytest.mark.parametrize(
    "value",
    [
        scalar([3, 4, 12], "[3]float32", "Vector"),
        scalar([3, 4, 12], "[]float32", None),
        scalar(0, "uint64", "VectorWS"),
    ],
)
def test_vector_name_or_values_without_reference_do_not_accredit_velocity(value):
    args = inputs()
    replace_field(args, "m_vecAbsVelocity", value)
    assert features(args)["self"]["native_speed"]["status"] == "unavailable"


@pytest.mark.parametrize(
    "value",
    [
        scalar([1, 2], "[]float32", "VectorWS"),
        scalar([1, 2, float("nan")], "[]float32", "VectorWS"),
        scalar([1, 2, True], "[]float32", "VectorWS"),
    ],
)
def test_partial_nonfinite_or_boolean_velocity_components_are_rejected(value):
    args = inputs()
    replace_field(args, "m_vecAbsVelocity", value)
    with pytest.raises((ValueError, ValidationError)):
        features(args)


def test_conflicting_world_velocity_sources_abstain():
    args = inputs()
    replace_field(args, "m_vecVelocity", scalar([0, 0, 1], "[3]float32", "VectorWS"))
    assert (
        features(args)["self"]["native_speed"]["reason"]
        == "native_velocity_sources_conflict"
    )


@pytest.mark.parametrize("kind", ["missing", "parented", "alternate", "attachment"])
def test_world_velocity_requires_unambiguous_current_parent_context(kind):
    args = inputs()
    state = actor_state(args)
    if kind == "missing":
        replace_field(args, "CBodyComponent.m_hParent", scalar())
        state["parent"] = reference()
    elif kind == "parented":
        replace_field(
            args,
            "CBodyComponent.m_hParent",
            scalar(0, "uint64", "CGameSceneNodeHandle"),
        )
        state["parent"] = reference(
            0, "support-motion-reference-000001", class_name="CWorld"
        )
    elif kind == "alternate":
        replace_field(
            args, "m_hMoveParent", scalar(0, "uint64", "CHandle< CBaseEntity >")
        )
    else:
        replace_field(args, "m_iParentAttachment", scalar(1, "uint64", "uint32"))
    assert features(args)["self"]["native_speed"]["status"] == "unavailable"


def test_current_airborne_proof_uses_false_flag_and_explicit_invalid_ground():
    args = inputs()
    replace_field(args, "m_fFlags", scalar(0, "uint64", "uint32"))
    replace_field(
        args, "m_hGroundEntity", scalar(16777215, "uint64", "CHandle< CBaseEntity >")
    )
    actor_state(args)["ground"] = reference(16777215, status="invalid_sentinel")
    assert features(args)["self"]["ground_contact"]["value"] is False
    assert features(args)["self"]["ground_reference_state"]["value"] == "none"


@pytest.mark.parametrize(
    "kind",
    [
        "absent_flags",
        "absent_ground",
        "wrong_flags_type",
        "wrong_ground_declaration",
        "serial_mismatch",
        "flag_conflict",
    ],
)
def test_ground_contact_requires_presence_types_and_concordant_current_reference(kind):
    args = inputs()
    state = actor_state(args)
    if kind == "absent_flags":
        replace_field(args, "m_fFlags", scalar())
    elif kind == "absent_ground":
        replace_field(args, "m_hGroundEntity", scalar())
        state["ground"] = reference()
    elif kind == "wrong_flags_type":
        replace_field(args, "m_fFlags", scalar(True, "bool", "uint32"))
    elif kind == "wrong_ground_declaration":
        replace_field(args, "m_hGroundEntity", scalar(0, "uint64", "uint32"))
    elif kind == "serial_mismatch":
        state["ground"] = reference(0, status="serial_mismatch")
    else:
        replace_field(args, "m_fFlags", scalar(0, "uint64", "uint32"))
    assert features(args)["self"]["ground_contact"]["status"] == "unavailable"
    assert features(args)["self"]["native_speed"]["value"] == 13


def test_dynamic_ground_contact_does_not_claim_world_floor_or_route():
    args = inputs()
    actor_state(args)["ground"]["class_name"] = "CFuncMoveLinear"
    actual = features(args)["self"]
    assert actual["ground_contact"]["value"] is True
    assert actual["ground_reference_state"]["value"] == "entity"


@pytest.mark.parametrize(
    "key", ["clock_order", "frame", "tick", "epoch", "source_sequence"]
)
def test_cut_corruption_is_rejected(key):
    args = inputs()
    args[2]["observations"]["cuts"][0][key] += 1
    with pytest.raises((ValueError, ValidationError)):
        s.validate_source(args[2], args[0], args[1])


@pytest.mark.parametrize(
    "kind",
    [
        "future_field",
        "predecessor",
        "creation_replay",
        "wrong_handle",
        "wrong_pawn",
        "epoch",
        "first_cut",
        "destroyed",
        "orphan_state",
    ],
)
def test_state_delivery_and_pawn_identity_corruption_is_rejected(kind):
    args = inputs()
    obs = args[2]["observations"]
    state = actor_state(args)
    if kind == "future_field":
        state["field_sequences"]["m_fFlags"] += 10000
    elif kind == "predecessor":
        obs["deliveries"][0]["previous"] = scalar(0, "uint64", "uint32")
    elif kind == "creation_replay":
        obs["deliveries"][0]["kind"] = "creation_replay"
    elif kind == "wrong_handle":
        state["handle"] += 16384
    elif kind == "wrong_pawn":
        obs["cuts"][0]["players"][0]["pawn_life"] = "support-pawn-999999"
    elif kind == "epoch":
        state["epoch"] += 1
    elif kind == "first_cut":
        state["first_cut"] += 1
    elif kind == "destroyed":
        obs["deliveries"].append(
            {
                "sequence": len(obs["deliveries"]) + 1,
                "frame": 99,
                "tick": 199,
                "entity_life": state["entity_life"],
                "kind": "destroyed",
                "property": "",
                "previous": scalar(),
                "value": scalar(),
            }
        )
        for c in obs["cuts"]:
            c["delivery_sequence"] += 1
    else:
        obs["states"].append(copy.deepcopy(state))
    with pytest.raises((ValueError, ValidationError)):
        s.validate_source(args[2], args[0], args[1])


def test_retained_baseline_does_not_require_a_new_delivery_at_every_cut():
    args = inputs()
    s.validate_source(args[2], args[0], args[1])
    assert all(
        c["delivery_sequence"]
        == args[2]["observations"]["cuts"][0]["delivery_sequence"]
        for c in args[2]["observations"]["cuts"]
    )
    assert features(args)["self"]["ground_contact"]["value"] is True


def test_native_life_cannot_be_reassigned_to_a_later_canonical_pawn_life():
    args = inputs()
    for base, cut in zip(
        args[0]["observations"]["frames"][1:],
        args[2]["observations"]["cuts"][1:],
        strict=True,
    ):
        base["players"][0]["pawn_life"] = "support-pawn-999999"
        cut["players"][0]["pawn_life"] = "support-pawn-999999"
    with pytest.raises(ValueError, match="identity changed"):
        s.validate_source(args[2], args[0], args[1])


def test_unavailable_pawn_keeps_all_motion_facts_unavailable():
    args = inputs()
    obs = args[2]["observations"]
    for cut in obs["cuts"]:
        for player in cut["players"]:
            if player["actor_alias"] == ACTOR:
                player["state_sequence"] = player["pawn_handle"] = player[
                    "pawn_life"
                ] = None
    # Remove the unused first state and reindex the remaining state references.
    obs["states"] = obs["states"][1:]
    for state in obs["states"]:
        state["sequence"] -= 1
    for cut in obs["cuts"]:
        cut["state_sequence"] -= 1
        for player in cut["players"]:
            if player["state_sequence"] is not None:
                player["state_sequence"] -= 1
    assert all(
        f["reason"] == "current_motion_pawn_unavailable"
        for f in features(args)["self"].values()
    )


def test_malformed_future_and_enemy_motion_do_not_change_initial_features():
    args = inputs()
    before = features(args)
    obs = args[2]["observations"]
    enemy = next(
        x["actor_alias"]
        for x in obs["cuts"][0]["players"]
        if x["actor_alias"] not in (ACTOR, ALLY)
    )
    replace_field(
        args,
        "m_vecAbsVelocity",
        scalar([777, 555, 333], "[3]float32", "VectorWS"),
        enemy,
    )
    for key, index in (
        ("cuts", "source_sequence"),
        ("states", "sequence"),
        ("deliveries", "sequence"),
    ):
        obs[key].append(
            {index: max(x[index] for x in obs[key]) + 1, "oracle": "malformed future"}
        )
    assert features(args) == before
    assert not any(
        token in json.dumps(before)
        for token in [
            "support-actor-",
            "support-pawn-",
            "entity_life",
            "anchor_id",
            "oracle",
            "source_sequence",
        ]
    )


def test_report_preserves_anchor_denominators_matrix_and_closed_gates():
    args = inputs()
    report = s.build_report(*report_args(args))
    count = len(report["rows"])
    assert report["summary"]["anchors"] == count > 0
    assert sum(row["anchors"] for row in report["matrix"]) == count
    assert all(
        sum(v.values()) == count
        for v in report["summary"]["fact_status_counts"].values()
    )
    assert report["summary"]["eligible_decisions"] == 0
    assert not any(
        report[k]
        for k in ("comparison_allowed", "training_allowed", "product_supported")
    )


def test_reader_recomputes_motion_and_calls_previous_reader(tmp_path, monkeypatch):
    args = report_args(inputs())
    report = s.build_report(*args)
    target = tmp_path / s.REPORT
    target.write_text(json.dumps(report))
    calls = []
    monkeypatch.setattr(s.z, "validate_artifact", lambda *x: calls.append(x))
    monkeypatch.setattr(s, "report_inputs", lambda *x: args)
    source = tmp_path / s.DATA
    assert (
        s.validate_artifact(tmp_path, source, tmp_path / "evaluator", tmp_path / "maps")
        == report["summary"]
    )
    assert len(calls) == 1
    report["rows"][0]["initial"]["self"]["ground_contact"]["value"] = False
    target.write_text(json.dumps(report))
    with pytest.raises(ValueError, match="recomputation"):
        s.validate_artifact(tmp_path, source, tmp_path / "evaluator", tmp_path / "maps")
