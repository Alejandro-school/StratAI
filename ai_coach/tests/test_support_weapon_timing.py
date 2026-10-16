import copy
import json

import pytest
from jsonschema import ValidationError

from ai_coach.support import anchors as a
from ai_coach.support import objective as o
from ai_coach.support import weapon_timing as s
from ai_coach.tests.test_support_anchors import ACTOR, ALLY, ENEMY, REF
from ai_coach.tests.test_support_native_motion import inputs as motion_inputs
from ai_coach.tests.test_support_objective_clock import inputs as clock_inputs
from ai_coach.tests.test_support_objective_clock import scalar
from ai_coach.tests.test_support_physical import physical


def inputs():
    support, objective, ownership, clock = clock_inputs()
    motion = motion_inputs()[-1]
    phys = physical(support)
    deliveries, states, cuts, objects = [], [], [], {}
    for index, actor in enumerate(support["observations"]["frames"][0]["players"]):
        pawn_life = f"support-timing-entity-{index * 2 + 1:06d}"
        weapon_life = f"support-timing-entity-{index * 2 + 2:06d}"
        wh = 32768 + index + 10
        for kind, handle, life, target, target_handle, names in (
            ("pawn", actor["pawn_handle"], pawn_life, weapon_life, wh, s.PAWN),
            ("weapon", wh, weapon_life, pawn_life, actor["pawn_handle"], s.WEAPON),
        ):
            fields = {name: scalar() for name in names}
            fields[names[0]] = scalar(target_handle, "uint64", "CHandle< CBaseEntity >")
            if kind == "pawn":
                fields[s.PAWN[1]] = scalar(110, declared="GameTime_t")
            else:
                for name in s.WEAPON[1:]:
                    fields[name] = (
                        scalar(7040, "uint64", "GameTick_t")
                        if "Tick" in name and "Ratio" not in name
                        else scalar(0.25, declared="float32")
                    )
            seqs = {}
            for name, value in fields.items():
                deliveries.append(
                    {
                        "sequence": len(deliveries) + 1,
                        "frame": 99,
                        "tick": 199,
                        "entity_life": life,
                        "kind": "created",
                        "property": name,
                        "previous": scalar(declared=value["declared_type"]),
                        "value": copy.deepcopy(value),
                    }
                )
                seqs[name] = len(deliveries)
            state = {
                "sequence": len(states) + 1,
                "first_cut": 1,
                "instance": 1,
                "epoch": 0,
                "entity_life": life,
                "handle": handle,
                "class_name": "CCSPlayerPawn" if kind == "pawn" else "CAK47",
                "kind": kind,
                "created_frame": 99,
                "fields": fields,
                "field_sequences": seqs,
                "reference": {
                    "status": "resolved",
                    "handle": target_handle,
                    "target_life": target,
                },
            }
            states.append(state)
            objects[actor["actor_alias"], kind] = state
    # The fixture uses a positive round-origin clock witness, not pawn simulation.
    event = objective["observations"]["events"][0]
    event["frame"], event["tick"] = 100, 200
    for cf in clock["observations"]["frames"]:
        cf["rules"]["fields"]["m_fRoundStartTime"] = scalar(
            6600 / 64, declared="GameTime_t"
        )
        cf["network"]["interval"] = copy.deepcopy(
            clock["observations"]["frames"][0]["network"]["interval"]
        )
        for cp in cf["players"]:
            cp["fields"]["m_flSimulationTime"] = scalar()
    for seq, (base, mcut, pf, cf) in enumerate(
        zip(
            support["observations"]["frames"],
            motion["observations"]["cuts"],
            phys["observations"]["frames"],
            clock["observations"]["frames"],
            strict=True,
        ),
        1,
    ):
        players = []
        for actor, pp in zip(base["players"], pf["players"], strict=True):
            pawn, weapon = (
                objects[actor["actor_alias"], kind] for kind in ("pawn", "weapon")
            )
            pp["context"]["weapon_handle"] = weapon["handle"]
            players.append(
                {
                    "actor_alias": actor["actor_alias"],
                    "pawn_handle": actor["pawn_handle"],
                    "pawn_life": actor["pawn_life"],
                    "pawn_state_sequence": pawn["sequence"],
                    "weapon_state_sequence": weapon["sequence"],
                    "weapon_life": pp["context"]["weapon_life"],
                    "weapon_status": "observed",
                }
            )
        cuts.append(
            {
                **{
                    k: mcut[k]
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
        "schema_id": "stratai.support_weapon_timing_observations@1",
        "contract_version": "1.0.0-candidate.1",
        **{
            k: support[k] for k in ("source_alias", "source_sha256", "build_identifier")
        },
        "comparison_allowed": False,
        "training_allowed": False,
        "product_supported": False,
        "references": {
            **motion["references"],
            "motion_source": REF,
            "physical_source": REF,
        },
        "capture_contract": "native_weapon_timing_deliveries_and_exact_support_cuts@1",
        "observations": {"deliveries": deliveries, "states": states, "cuts": cuts},
    }
    return [support, motion, phys, objective, ownership, clock, data]


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
        *args, row["anchor_id"], ACTOR, row["t0_tick"], row["t0_frame"]
    )


def replace_field(args, name, value, index=0):
    state = args[-1]["observations"]["states"][index]
    state["fields"][name] = copy.deepcopy(value)
    for d in args[-1]["observations"]["deliveries"]:
        if d["entity_life"] == state["entity_life"] and d["property"] == name:
            d["value"] = copy.deepcopy(value)
            d["previous"] = scalar(declared=value["declared_type"])


def build(args):
    selected = a.select_anchors(
        args[0]["observations"]["frames"], args[0]["source_alias"]
    )[0]
    previous = {
        "rows": [
            {
                **{k: r[k] for k in o.KEYS},
                "competitive_round_status": "canonical_interval_matched",
            }
            for r in selected
        ]
    }
    refs = {
        k: REF
        for k in a.schema_validator(s.REPORT).schema["properties"]["references"][
            "required"
        ]
    }
    return s.build_report(*args, previous, refs)


def test_positive_native_game_time_clock_and_no_general_readiness():
    args = inputs()
    actual = features(args)
    assert actual["self"]["pawn_remaining_seconds"]["value"] == 110 - 6608 / 64
    assert actual["self"]["pawn_deadline_state"]["value"] == "pending"
    assert (
        actual["self"]["primary_remaining_seconds"]["reason"]
        == "native_weapon_tick_fraction_clock_unaccredited"
    )
    assert "support-" not in json.dumps(actual) and "can_fire" not in json.dumps(actual)
    report = build(args)
    assert len(report["clock_witnesses"]) == 1 and report["summary"]["anchors"] == 2
    assert all(
        sum(v.values()) == 2 for v in report["summary"]["fact_status_counts"].values()
    )
    assert report["summary"]["eligible_decisions"] == 0


@pytest.mark.parametrize(
    "value,typ,declared,expected",
    [
        (0, "float32", "GameTime_t", None),
        (None, None, None, None),
        (-1, "float32", "GameTime_t", None),
        (2147483647, "float32", "GameTime_t", None),
        (110, "uint64", "GameTime_t", None),
        (110, "float32", "float32", None),
        (100, "float32", "GameTime_t", 0),
        (6608 / 64, "float32", "GameTime_t", 0),
    ],
)
def test_zero_absence_sentinel_type_and_elapsed_deadline(
    value, typ, declared, expected
):
    args = inputs()
    replace_field(args, s.PAWN[1], scalar(value, typ, declared))
    actual = features(args)["self"]
    assert actual["pawn_remaining_seconds"]["value"] == expected
    assert actual["pawn_deadline_state"]["value"] == (
        "elapsed" if expected == 0 else None
    )


@pytest.mark.parametrize(
    "mutation",
    [
        "missing_tick",
        "disagree",
        "old_tick",
        "interval_change",
        "origin_change",
        "rules_life_change",
        "rewind",
        "pause",
        "unknown_pause",
    ],
)
def test_clock_proof_invalidations_and_pause(mutation):
    args = inputs()
    clock = args[5]["observations"]["frames"]
    row = clock[2]
    if mutation == "missing_tick":
        row["network"]["tick"] = None
    elif mutation == "disagree":
        row["network"]["tick"]["value"]["value"] += 1
    elif mutation == "old_tick":
        row["network"]["tick"]["frame"] -= 1
        row["network"]["tick"]["tick"] -= 1
    elif mutation == "interval_change":
        row["network"]["interval"]["order"] = 33
    elif mutation == "origin_change":
        row["rules"]["fields"]["m_fRoundStartTime"]["value"] += 1
    elif mutation == "rules_life_change":
        row["rules"]["entity_life"] = "support-clock-object-000002"
    elif mutation == "rewind":
        for k in ("tick", "packet_tick"):
            row["network"][k]["value"]["value"] = 6599
    elif mutation == "pause":
        row["rules"]["fields"]["m_bGamePaused"] = scalar(True)
    elif mutation == "unknown_pause":
        for name, value in row["rules"]["fields"].items():
            if name.startswith("m_b"):
                value.update(status="absent", value=None, native_type=None)
        row["network"]["paused"] = None
    actual = features(args)["self"]
    if mutation == "pause":
        assert actual["pawn_deadline_state"]["value"] == "paused"
    else:
        assert actual["pawn_remaining_seconds"]["status"] == "unavailable"


@pytest.mark.parametrize(
    "tick,ratio",
    [
        (0, 0),
        (2147483647, 0),
        (7040, None),
        (7040, 1),
        (7040, -0.1),
        (7040, 0.25),
        (7040, 0),
    ],
)
def test_tick_fraction_never_assumes_unproved_formula(tick, ratio):
    args = inputs()
    replace_field(args, s.WEAPON[1], scalar(tick, "uint64", "GameTick_t"), 1)
    replace_field(args, s.WEAPON[2], scalar(ratio, declared="float32"), 1)
    assert features(args)["self"]["primary_remaining_seconds"]["value"] is None


@pytest.mark.parametrize(
    "mutation",
    [
        "state_future",
        "delivery_frontier",
        "predecessor",
        "pawn_life",
        "handle",
        "owner",
        "weapon_life",
        "destroyed",
        "epoch",
        "instance",
        "clock_order",
        "future_clock_order",
        "future_objective_event",
    ],
)
def test_adversarial_provenance_rejected(mutation):
    args = inputs()
    obs = args[-1]["observations"]
    cut = obs["cuts"][2]
    if mutation == "state_future":
        cut["players"][0]["pawn_state_sequence"] = 999
    elif mutation == "delivery_frontier":
        cut["delivery_sequence"] -= 1
    elif mutation == "predecessor":
        obs["deliveries"][0]["previous"] = scalar(1, "uint64")
    elif mutation == "pawn_life":
        cut["players"][0]["pawn_life"] = "support-pawn-000099"
    elif mutation == "handle":
        obs["states"][1]["handle"] += 16384
    elif mutation == "owner":
        obs["states"][1]["reference"]["target_life"] = "support-timing-entity-000003"
    elif mutation == "weapon_life":
        cut["players"][0]["weapon_life"] = "support-weapon-000099"
    elif mutation == "destroyed":
        obs["deliveries"].append(
            {
                "sequence": len(obs["deliveries"]) + 1,
                "frame": 107,
                "tick": 207,
                "entity_life": obs["states"][1]["entity_life"],
                "kind": "destroyed",
                "property": "",
                "previous": scalar(),
                "value": scalar(),
            }
        )
        cut["delivery_sequence"] += 1
    elif mutation in ("epoch", "instance"):
        obs["states"][0][mutation] += 1
    elif mutation == "clock_order":
        cut["clock_order"] += 1
    elif mutation == "future_clock_order":
        args[5]["observations"]["frames"][2]["network"]["tick"]["order"] = 999
    elif mutation == "future_objective_event":
        args[3]["observations"]["events"][0]["frame"] = 999
    with pytest.raises((ValueError, ValidationError)):
        features(args)


def test_nonfinite_native_retained_as_unavailable():
    args = inputs()
    replace_field(
        args,
        s.PAWN[1],
        {
            "status": "nonfinite",
            "value": None,
            "native_type": "float32",
            "declared_type": "GameTime_t",
        },
    )
    assert (
        features(args)["self"]["pawn_remaining_seconds"]["reason"]
        == "native_clock_nonfinite"
    )


def test_enemy_and_malformed_future_cannot_change_initial_features():
    args = inputs()
    expected = features(args)
    replace_field(args, s.PAWN[1], scalar(999, declared="GameTime_t"), 4)
    assert args[0]["observations"]["frames"][0]["players"][2]["actor_alias"] == ENEMY
    for source, key in (
        (args[0], "frames"),
        (args[1], "cuts"),
        (args[2], "frames"),
        (args[3], "frames"),
        (args[5], "frames"),
        (args[6], "cuts"),
    ):
        for row in source["observations"][key][3:]:
            row["malformed_future"] = True
    args[-1]["observations"]["deliveries"].append(
        {"sequence": 99999, "malformed_future": True}
    )
    assert features(args) == expected


def test_non_ammunition_category_retains_native_without_firearm_deadline():
    args = inputs()
    for row in args[2]["observations"]["frames"]:
        ctx = row["players"][0]["context"]
        ctx["facts"]["active_weapon"]["value"] = "Knife"
        ctx.update(
            raw_clip=None,
            raw_reserve=None,
            reserve_source=None,
            reload_signal=None,
            reload_signal_tick=None,
        )
        for name in ("ammo_in_magazine", "ammo_reserve", "reloading"):
            ctx["facts"][name] = {
                "status": "not_applicable",
                "value": None,
                "reason": "non_ammunition_weapon",
            }
    assert (
        features(args)["self"]["primary_deadline_state"]["status"] == "not_applicable"
    )


def test_materialization_rejects_nonempty_before_reading_source(tmp_path):
    (tmp_path / "keep").write_text("retained")
    with pytest.raises(ValueError):
        s.materialize(tmp_path / s.DATA, tmp_path, tmp_path / "evaluator", tmp_path)


def test_reader_recomputes_new_report(monkeypatch, tmp_path):
    args = inputs()
    report = build(args)
    monkeypatch.setattr(s.m, "validate_artifact", lambda *a: None)
    monkeypatch.setattr(
        s,
        "report_inputs",
        lambda *a: [*args, {"rows": report["rows"]}, report["references"]],
    )
    path = tmp_path / s.REPORT
    path.write_text(json.dumps(report))
    assert (
        s.validate_artifact(tmp_path, tmp_path / s.DATA, tmp_path, tmp_path)
        == report["summary"]
    )
    report["rows"][0]["initial"]["self"]["pawn_remaining_seconds"]["value"] += 1
    path.write_text(json.dumps(report))
    with pytest.raises(ValueError, match="recomputation"):
        s.validate_artifact(tmp_path, tmp_path / s.DATA, tmp_path, tmp_path)
