import copy
import json

import pytest
from jsonschema import ValidationError

from ai_coach.support import anchors as a
from ai_coach.support import objective as o
from ai_coach.support import objective_clock as c
from ai_coach.tests.test_support_anchors import ACTOR, ALLY, ENEMY, REF
from ai_coach.tests.test_support_c4 import build as c4_report
from ai_coach.tests.test_support_c4 import inputs as c4_inputs
from ai_coach.tests.test_support_c4 import ref


def scalar(value=None, typ=None, declared=None):
    return {
        "status": "observed" if value is not None else "absent",
        "value": value,
        "native_type": typ
        or (
            "bool" if type(value) is bool else "float32" if value is not None else None
        ),
        "declared_type": declared,
    }


def clock_data(support, objective, ownership):
    rows = []
    for base, of, cut in zip(
        support["observations"]["frames"],
        objective["observations"]["frames"],
        ownership["observations"]["cuts"],
        strict=True,
    ):
        seq = base["sequence"]

        def message(value, typ, offset, base=base, seq=seq):
            return {
                "frame": base["frame"],
                "tick": base["tick"],
                "order": seq * 10 + offset,
                "value": scalar(value, typ),
            }

        fields = {
            k: scalar(False) if k.startswith("m_b") else scalar() for k in c.RULES
        }
        fields["m_fRoundStartTime"] = scalar(0, declared="GameTime_t")
        fields["m_iRoundTime"] = scalar(115, "int")
        rows.append(
            {
                "source_sequence": seq,
                **{k: base[k] for k in ("frame", "tick", "instance", "epoch")},
                "objective_order": of["order"],
                "c4_state_sequence": cut["state_sequence"],
                "order": seq * 10 + 9,
                "network": {
                    "tick": message(base["tick"] + 6400, "uint32", 1),
                    "packet_tick": message(base["tick"] + 6400, "uint32", 2),
                    "interval": message(1 / 64, "float32", 3),
                    "paused": message(False, "bool", 4),
                },
                "rules": {
                    "entity_life": "support-clock-object-000001",
                    "handle": 22,
                    "class_name": "CCSGameRulesProxy",
                    "fields": fields,
                    "defuser": None,
                },
                "players": [
                    {
                        "actor_alias": x["actor_alias"],
                        "pawn_life": x["pawn_life"],
                        "pawn_handle": x["pawn_handle"],
                        "fields": {
                            "m_flSimulationTime": scalar(
                                (base["tick"] + 6400) / 64, declared="float32"
                            ),
                            "m_nTickBase": scalar(),
                        },
                    }
                    for x in base["players"]
                ],
                "bombs": [],
                "items": [
                    {
                        "entity_life": "support-item-000001",
                        "handle": 32778,
                        "class_name": "CC4",
                        "fields": {
                            k: scalar(False) if k == "m_bStartedArming" else scalar()
                            for k in c.ITEM
                        },
                        "defuser": None,
                    }
                ],
            }
        )
    return {
        "schema_id": "stratai.support_objective_clock_observations@1",
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
        "capture_contract": "same_support_FrameDone_native_game_clocks@1",
        "references": {
            k: REF for k in ("support_source", "objective_source", "c4_source")
        },
        "observations": {"frames": rows},
    }


def inputs():
    support, objective, ownership = c4_inputs()
    return support, objective, ownership, clock_data(support, objective, ownership)


def features(*args):
    anchor = a.select_anchors(
        args[0]["observations"]["frames"], args[0]["source_alias"]
    )[0][0]
    return c.initial_features(
        *args, anchor["anchor_id"], ACTOR, anchor["t0_tick"], anchor["t0_frame"]
    )


def build(*args):
    refs = {
        k: REF
        for k in (
            "support_source",
            "objective_source",
            "c4_source",
            "clock_source",
            "anchors",
            "physical_context",
            "navigation_context",
            "objective_context",
            "c4_context",
        )
    }
    return c.build_report(
        *args, a.build_report(args[0], REF), c4_report(*args[:3]), refs
    )


def planted(args, active=True):
    _, objective, _, data = args
    objective["observations"]["events"].append(
        {
            **objective["observations"]["events"][0],
            "sequence": 2,
            "order": 2,
            "kind": "bomb_planted",
        }
    )
    for row, of in zip(
        data["observations"]["frames"], objective["observations"]["frames"], strict=True
    ):
        of["event_sequence"] = 2
        of["rules"]["m_bBombPlanted"]["value"] = True
        of["players"][0]["fields"]["m_bIsDefusing"]["value"] = active
        player = row["players"][0]
        fields = {k: scalar() for k in c.BOMB}
        fields.update(
            {
                "m_bBombTicking": scalar(True),
                "m_bBombDefused": scalar(False),
                "m_flC4Blow": scalar(130, declared="GameTime_t"),
                "m_flTimerLength": scalar(40),
                "m_flDefuseCountDown": scalar(108, declared="GameTime_t"),
                "m_flDefuseLength": scalar(5),
            }
        )
        row["bombs"] = [
            {
                "entity_life": "support-clock-object-000002",
                "handle": 23,
                "class_name": "CPlantedC4",
                "fields": fields,
                "defuser": ref(player["pawn_handle"], player["pawn_life"]),
            }
        ]


def test_native_domain_not_demo_clock_and_matrix_denominators():
    args = inputs()
    result = features(*args)
    assert result["self"]["round_remaining_seconds"] == o.fact(11.75)
    assert result["self"]["clock_state"] == o.fact("aligned_native_simulation")
    assert result["self"]["bomb_remaining_seconds"]["status"] == "not_applicable"
    report = build(*args)
    assert report["rows"][0]["initial"] == result
    assert report["summary"]["anchors"] == 2
    assert all(
        sum(x.values()) == 2 for x in report["summary"]["fact_status_counts"].values()
    )
    assert (
        not report["training_allowed"] and report["summary"]["eligible_decisions"] == 0
    )


@pytest.mark.parametrize(
    "mode,reason",
    [
        ("interval", "sdk_simtime_interval_unaccredited"),
        ("simtime", "current_actor_simulation_alignment_unaccredited"),
        ("tick_absent", "current_native_server_tick_absent"),
        ("ticks_disagree", "native_server_ticks_disagree"),
        ("stale_tick", "current_native_server_tick_absent"),
        ("game_domain", "native_game_time_domain_unaccredited"),
        ("pause_absent", "general_pause_state_unavailable"),
        ("duration_absent", "native_clock_absent"),
        ("duration_bool", "native_clock_type_mismatch"),
        ("duration_negative", "native_clock_outside_range"),
    ],
)
def test_clock_gaps_abstain_instead_of_defaults(mode, reason):
    args = inputs()
    row = args[3]["observations"]["frames"][2]
    if mode == "interval":
        row["network"]["interval"]["value"] = scalar(1 / 128)
    elif mode == "simtime":
        row["players"][0]["fields"]["m_flSimulationTime"] = scalar(3.25)
    elif mode == "tick_absent":
        row["network"]["tick"] = row["network"]["packet_tick"] = None
    elif mode == "ticks_disagree":
        row["network"]["packet_tick"]["value"]["value"] += 1
    elif mode == "stale_tick":
        for key in ("tick", "packet_tick"):
            row["network"][key]["frame"] -= 1
    elif mode == "game_domain":
        row["rules"]["fields"]["m_fRoundStartTime"]["declared_type"] = "float32"
    elif mode == "pause_absent":
        row["network"]["paused"] = None
        row["rules"]["fields"]["m_bGamePaused"] = scalar()
    elif mode == "duration_absent":
        row["rules"]["fields"]["m_iRoundTime"] = scalar()
    elif mode == "duration_bool":
        row["rules"]["fields"]["m_iRoundTime"] = scalar(False)
    elif mode == "duration_negative":
        row["rules"]["fields"]["m_iRoundTime"] = scalar(-1, "int")
    assert features(*args)["self"]["round_remaining_seconds"]["reason"] == reason


@pytest.mark.parametrize("paused", [False, True])
@pytest.mark.parametrize("delta", [-1, 0, 1])
def test_paused_expired_and_valid_zero_are_distinct(paused, delta):
    args = inputs()
    row = args[3]["observations"]["frames"][2]
    row["network"]["paused"]["value"] = scalar(paused)
    row["rules"]["fields"]["m_iRoundTime"] = scalar(0, "int")
    row["rules"]["fields"]["m_fRoundStartTime"] = scalar(
        103.25 + delta, declared="GameTime_t"
    )
    result = features(*args)["self"]
    assert result["round_remaining_seconds"] == o.fact(max(0, delta))
    assert result["round_clock_state"] == o.fact(
        "expired" if delta <= 0 else "paused" if paused else "active"
    )


def test_native_defuse_countdown_progress_and_signed_deadline_margin():
    args = inputs()
    planted(args)
    result = features(*args)
    assert result["self"]["bomb_remaining_seconds"] == o.fact(26.75)
    assert result["self"]["attempt_remaining_seconds"] == o.fact(4.75)
    assert result["self"]["attempt_progress"]["value"] == pytest.approx(0.05)
    assert result["self"]["defuse_deadline_margin_seconds"] == o.fact(22)
    assert result["ally"]["attempt_state"] == o.fact("none")
    assert result["self"]["round_remaining_seconds"]["status"] == "not_applicable"
    args[3]["observations"]["frames"][2]["bombs"][0]["fields"]["m_flC4Blow"][
        "value"
    ] = 104
    assert features(*args)["self"]["defuse_deadline_margin_seconds"] == o.fact(-4)


@pytest.mark.parametrize(
    "mode",
    [
        "serial",
        "life",
        "sentinel",
        "destroyed",
        "future_order",
        "same_tick_later",
        "type",
        "schema_extra",
    ],
)
def test_identity_order_and_strict_schema(mode):
    args = inputs()
    planted(args)
    row = args[3]["observations"]["frames"][2]
    if mode == "serial":
        row["bombs"][0]["defuser"]["handle"] += 16384
    elif mode == "life":
        row["bombs"][0]["defuser"]["target_life"] = "support-pawn-999999"
    elif mode == "sentinel":
        row["bombs"][0]["defuser"] = ref(4294967295, status="invalid_sentinel")
    elif mode == "destroyed":
        row["bombs"] = []
    elif mode in ("future_order", "same_tick_later"):
        row["network"]["tick"]["order"] = row["order"] + 1
        if mode == "future_order":
            row["network"]["tick"]["frame"] += 1
    elif mode == "type":
        row["network"]["tick"]["value"]["value"] = False
    elif mode == "schema_extra":
        row["oracle"] = True
    if mode in ("serial", "life", "sentinel", "destroyed"):
        assert (
            features(*args)["self"]["attempt_state"]["reason"]
            == "native_defuser_identity_uncorroborated"
        )
    else:
        with pytest.raises((ValueError, ValidationError)):
            features(*args)


def test_native_arming_has_no_default_duration_and_interrupt_is_causal():
    args = inputs()
    for row in args[3]["observations"]["frames"]:
        row["items"][0]["fields"]["m_bStartedArming"] = scalar(True)
        row["items"][0]["fields"]["m_fArmedTime"] = scalar(105, declared="GameTime_t")
    result = features(*args)["self"]
    assert result["attempt_state"] == o.fact("planting")
    assert result["attempt_remaining_seconds"] == o.fact(1.75)
    assert result["attempt_progress"]["reason"] == "native_attempt_duration_unavailable"
    args[3]["observations"]["frames"][2]["items"][0]["fields"]["m_bStartedArming"] = (
        scalar(False)
    )
    assert features(*args)["self"]["attempt_state"] == o.fact("interrupted_planting")


def test_defuse_interruption_does_not_reuse_old_timer():
    args = inputs()
    planted(args)
    args[1]["observations"]["frames"][2]["players"][0]["fields"]["m_bIsDefusing"][
        "value"
    ] = False
    row = args[3]["observations"]["frames"][2]
    row["bombs"][0]["defuser"] = ref(4294967295, status="invalid_sentinel")
    result = features(*args)["self"]
    assert result["attempt_state"] == o.fact("interrupted_defusing")
    assert result["attempt_remaining_seconds"]["status"] == "not_applicable"


def test_native_flag_contradicting_current_own_reference_abstains():
    args = inputs()
    planted(args, active=False)
    assert (
        features(*args)["self"]["attempt_state"]["reason"]
        == "native_attempt_flag_reference_conflict"
    )


def test_future_and_unrelated_enemy_fields_do_not_change_initial_features():
    args = inputs()
    planted(args)
    before = features(*args)
    for row in args[3]["observations"]["frames"][:3]:
        enemy = next(x for x in row["players"] if x["actor_alias"] == ENEMY)
        enemy["fields"]["m_flSimulationTime"] = scalar(99999)
        enemy["fields"]["m_nTickBase"] = scalar(88888, "int")
    for source, key in (
        (args[0], "frames"),
        (args[1], "frames"),
        (args[2], "cuts"),
        (args[3], "frames"),
    ):
        for row in source["observations"][key][3:]:
            row["oracle"] = "future garbage"
    assert features(*args) == before
    serialized = json.dumps(before)
    assert not any(
        x in serialized
        for x in (
            ACTOR,
            ALLY,
            ENEMY,
            "pawn_life",
            "handle",
            "anchor_id",
            "oracle",
            "later_observation",
        )
    )


def test_epoch_or_instance_change_cannot_propagate_interruption():
    args = inputs()
    row = args[3]["observations"]["frames"][2]
    prior = c.context_inputs(*args, o.public_prefixes(args[1]), 2)
    prior[4]["items"][0]["fields"]["m_bStartedArming"] = scalar(True)
    prior[4]["items"][0]["fields"]["m_fArmedTime"] = scalar(105, declared="GameTime_t")
    anchor = a.select_anchors(
        args[0]["observations"]["frames"], args[0]["source_alias"]
    )[0][0]
    current = c.context_inputs(*args, o.public_prefixes(args[1]), 3)
    for key in ("instance", "epoch"):
        saved = row[key]
        row[key] += 1
        assert c.initial_context(anchor, *current, prior=prior)["self"][
            "attempt_state"
        ] == o.fact("none")
        row[key] = saved


def test_artifact_reader_recomputes_and_rejects_mutation(monkeypatch, tmp_path):
    args = inputs()
    report = build(*args)
    (tmp_path / c.REPORT).write_text(json.dumps(report), encoding="utf8")
    monkeypatch.setattr(c.c4, "validate_artifact", lambda *x: None)
    monkeypatch.setattr(
        c,
        "report_inputs",
        lambda *x: (
            *args,
            a.build_report(args[0], REF),
            c4_report(*args[:3]),
            report["references"],
        ),
    )
    assert (
        c.validate_artifact(tmp_path, tmp_path / c.DATA, None, None)
        == report["summary"]
    )
    changed = copy.deepcopy(report)
    changed["rows"][0]["initial"]["self"]["round_remaining_seconds"]["value"] += 1
    (tmp_path / c.REPORT).write_text(json.dumps(changed), encoding="utf8")
    with pytest.raises(ValueError, match="recomputation"):
        c.validate_artifact(tmp_path, tmp_path / c.DATA, None, None)


def origin_fixture(interval=1 / 64):
    args = inputs()
    rows = args[3]["observations"]["frames"]
    first = rows[0]
    freeze = args[1]["observations"]["events"][0]
    freeze.update(frame=first["frame"], tick=first["tick"])
    origin = first["network"]["tick"]["value"]["value"] * interval
    message = copy.deepcopy(first["network"]["interval"])
    message["value"] = scalar(interval)
    for row in rows:
        row["network"]["interval"] = copy.deepcopy(message)
        row["rules"]["fields"]["m_fRoundStartTime"] = scalar(
            origin, declared="GameTime_t"
        )
        for player in row["players"]:
            player["fields"]["m_flSimulationTime"] = scalar()
    return args


@pytest.mark.parametrize("interval", [1 / 64, 1 / 128])
def test_native_round_origin_accredits_domain_without_pawn_simtime(interval):
    args = origin_fixture(interval)
    result = features(*args)
    assert result["self"]["clock_state"] == o.fact("aligned_native_round_origin")
    assert result["self"]["round_remaining_seconds"] == o.fact(115 - 8 * interval)
    report = build(*args)
    assert len(report["clock_witnesses"]) == 1
    assert report["clock_witnesses"][0]["source_sequence"] == 1
    assert "clock_witnesses" not in result and "source_sequence" not in json.dumps(
        result
    )


@pytest.mark.parametrize(
    "mode",
    [
        "future_origin",
        "different_domain",
        "interval_session",
        "rewind",
        "new_origin",
        "new_rules_life",
        "missing_tick",
        "paused_origin",
        "warmup_origin",
        "freeze_delivery",
    ],
)
def test_round_origin_proof_never_crosses_unproved_clocks(mode):
    args = origin_fixture()
    rows = args[3]["observations"]["frames"]
    if mode == "future_origin":
        event = args[1]["observations"]["events"][0]
        event.update(frame=rows[3]["frame"], tick=rows[3]["tick"])
        for of in args[1]["observations"]["frames"][:3]:
            of["event_sequence"] = 0
    elif mode == "different_domain":
        for row in rows:
            row["rules"]["fields"]["m_fRoundStartTime"]["value"] -= 100
    elif mode == "interval_session":
        rows[2]["network"]["interval"]["order"] = rows[2]["order"] - 1
    elif mode == "rewind":
        for k in ("tick", "packet_tick"):
            rows[2]["network"][k]["value"]["value"] = 0
    elif mode == "new_origin":
        rows[2]["rules"]["fields"]["m_fRoundStartTime"]["value"] += 1
    elif mode == "new_rules_life":
        rows[2]["rules"]["entity_life"] = "support-clock-object-999999"
    elif mode == "missing_tick":
        rows[2]["network"]["packet_tick"] = None
    elif mode == "paused_origin":
        rows[0]["rules"]["fields"]["m_bGamePaused"] = scalar(True)
    elif mode == "warmup_origin":
        rows[0]["rules"]["fields"]["m_bWarmupPeriod"] = scalar(True)
    elif mode == "freeze_delivery":
        args[1]["observations"]["events"][0]["frame"] -= 1
    result = features(*args)
    assert result["self"]["clock_state"]["status"] == "unavailable"
    assert result["self"]["round_remaining_seconds"]["status"] == "unavailable"
