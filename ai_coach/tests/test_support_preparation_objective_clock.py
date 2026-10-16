import json
from pathlib import Path

import pytest
from jsonschema import ValidationError

from ai_coach.support import anchors as a
from ai_coach.support import objective as o
from ai_coach.support import preparation as c
from ai_coach.support import preparation_objective_clock as p
from ai_coach.tests import test_support_objective_clock as legacy
from ai_coach.tests.test_support_anchors import ACTOR, ALLY, ENEMY, REF
from ai_coach.tests.test_support_preparation_c4 import inputs as c4_inputs


def inputs(count=4):
    args = c4_inputs(count)
    return *args, legacy.clock_data(*args)


def features(*args, index=0, **overrides):
    cut = args[0]["observations"]["frames"][index]
    keys = {
        "identifier": c.candidate_id(args[0]["source_alias"], cut, ACTOR, ALLY),
        "actor": ACTOR,
        "ally": ALLY,
        "sequence": cut["sequence"],
        "tick": cut["tick"],
        "frame": cut["frame"],
    }
    keys.update(overrides)
    return p.initial_features(*args, **keys)


def test_all_pairs_first_samples_and_matrix():
    args = inputs()
    rows = list(p.iter_rows(*args))
    expected = [v["candidate_id"] for _, pairs in c.iter_census(args[0]) for v in pairs]
    assert [v["candidate_id"] for row in rows for v in row["candidates"]] == expected
    coverage = p.Coverage()
    for i, row in enumerate(rows):
        a.schema_validator(p.ROW_SCHEMA).validate(row)
        coverage.add(row)
        facts = features(*args, index=i)
        for actor in row["actors"]:
            role = "self" if actor["actor_alias"] == ACTOR else "ally"
            assert actor["initial"] == facts[role]["objective_clock"]
    assert len(rows) == 4
    for cell in coverage.result()["matrix"]:
        assert all(
            sum(v.values()) == cell["candidates"]
            for v in cell["fact_status_counts"].values()
        )


@pytest.mark.parametrize(
    "key,value",
    [
        ("identifier", "wrong"),
        ("actor", ENEMY),
        ("ally", ACTOR),
        ("sequence", 2),
        ("sequence", True),
        ("tick", 201),
        ("frame", 101),
    ],
)
def test_exact_candidate(key, value):
    with pytest.raises(ValueError):
        features(*inputs(), **{key: value})


@pytest.mark.parametrize("index", [0, 1, 2, 10])
def test_clip_before_validation_and_detached_legal_view(index):
    args = inputs(12)
    before = features(*args, index=index)
    for source, key in zip(args, ("frames", "frames", "cuts", "frames"), strict=True):
        source["observations"][key][index + 1 :] = [None, {"bad": True}]
    args[1]["observations"]["events"].append(None)
    args[2]["observations"]["states"][index + 1 :] = [None]
    for row in args[0]["observations"]["frames"][: index + 1]:
        row["players"][2]["position"] = {"x": 1e8, "y": -1e8, "z": 123}
    for row in args[3]["observations"]["frames"][: index + 1]:
        row["players"][2]["fields"]["m_flSimulationTime"] = legacy.scalar(99999)
    assert features(*args, index=index) == before
    for secret in (
        ACTOR,
        ALLY,
        ENEMY,
        "pawn_life",
        "handle",
        "provenance",
        "witness",
        "native_player",
        "matrix",
        "execution",
        "source_sha256",
    ):
        assert secret not in json.dumps(before)
    before["self"]["objective_clock"]["round_remaining_seconds"]["value"] = 999
    assert features(*args, index=index) != before


@pytest.mark.parametrize("empty", [True, False])
def test_missing_exact_clock_never_borrows_another_frame_same_tick(empty):
    args = inputs(2)
    args[0]["observations"]["frames"][1]["tick"] = 200
    # Recreate all other layers against the same-tick source.
    from ai_coach.tests.test_support_c4 import ownership
    from ai_coach.tests.test_support_objective import objective

    obj = objective(args[0])
    own = ownership(args[0], obj)
    args = args[0], obj, own, legacy.clock_data(args[0], obj, own)
    args[3]["observations"]["frames"] = args[3]["observations"]["frames"][
        : 0 if empty else 1
    ]
    result = features(*args, index=1)
    assert all(
        v["reason"] == "missing_exact_clock_sample"
        for v in result["self"]["objective_clock"].values()
    )
    assert len(list(p.iter_rows(*args))[1]["candidates"]) == 2


@pytest.mark.parametrize(
    "mode",
    [
        "frame",
        "tick",
        "instance",
        "epoch",
        "order",
        "objective_order",
        "c4_state_sequence",
        "life",
        "handle",
        "message_order",
        "type",
    ],
)
def test_source_identity_and_delivery_rejected(mode):
    args = inputs()
    row = args[3]["observations"]["frames"][0]
    if mode == "life":
        row["players"][0]["pawn_life"] = "support-pawn-999999"
    elif mode == "handle":
        row["players"][0]["pawn_handle"] += 16384
    elif mode == "message_order":
        row["network"]["tick"]["order"] = row["order"]
    elif mode == "type":
        row["network"]["tick"]["value"]["value"] = False
    elif mode == "order":
        row[mode] = 1
    else:
        row[mode] += 1
    with pytest.raises((ValueError, ValidationError)):
        features(*args)


@pytest.mark.parametrize(
    "legacy_test,kwargs",
    [
        (legacy.test_clock_gaps_abstain_instead_of_defaults, {"mode": m, "reason": r})
        for m, r in [
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
        ]
    ]
    + [
        (
            legacy.test_paused_expired_and_valid_zero_are_distinct,
            {"paused": paused, "delta": delta},
        )
        for paused in (False, True)
        for delta in (-1, 0, 1)
    ]
    + [
        (fn, {})
        for fn in (
            legacy.test_native_defuse_countdown_progress_and_signed_deadline_margin,
            legacy.test_native_arming_has_no_default_duration_and_interrupt_is_causal,
            legacy.test_defuse_interruption_does_not_reuse_old_timer,
            legacy.test_native_flag_contradicting_current_own_reference_abstains,
        )
    ],
)
def test_c0_semantics_against_existing_numeric_and_abstention_cases(
    monkeypatch, legacy_test, kwargs
):
    def c0_features(*args):
        result = features(*args, index=2)
        return {role: result[role]["objective_clock"] for role in c.ROLES}

    monkeypatch.setattr(legacy, "features", c0_features)
    legacy_test(**kwargs)


@pytest.mark.parametrize("interval", [1 / 64, 1 / 128])
def test_origin_available_from_first_delivery_without_simtime(interval):
    args = legacy.origin_fixture(interval)
    for index in (0, 2):
        result = features(*args, index=index)["self"]["objective_clock"]
        assert result["clock_state"] == o.fact("aligned_native_round_origin")
        assert result["round_remaining_seconds"] == o.fact(115 - 4 * index * interval)


def test_later_origin_cannot_fill_earlier_cut():
    args = legacy.origin_fixture()
    rows = args[3]["observations"]["frames"]
    freeze = args[1]["observations"]["events"][0]
    freeze.update(frame=rows[1]["frame"], tick=rows[1]["tick"], order=15)
    args[1]["observations"]["frames"][0]["event_sequence"] = 0
    for row in rows:
        row["rules"]["fields"]["m_fRoundStartTime"] = legacy.scalar(
            rows[1]["network"]["tick"]["value"]["value"] / 64, declared="GameTime_t"
        )
    assert (
        features(*args)["self"]["objective_clock"]["clock_state"]["status"]
        == "unavailable"
    )
    assert (
        features(*args, index=1)["self"]["objective_clock"]["clock_state"]["value"]
        == "aligned_native_round_origin"
    )


@pytest.mark.parametrize("key", ["side", "pawn_handle", "pawn_life"])
def test_interruption_requires_current_identity_and_side(key):
    args = inputs(2)
    public = o.public_prefixes(args[1])
    prior = p.k.context_inputs(*args, public, 1)
    current = p.k.context_inputs(*args, public, 2)
    prior[4]["items"][0]["fields"]["m_bStartedArming"] = legacy.scalar(True)
    before = p.actor_context(ACTOR, current, prior, None)
    assert before["attempt_state"]["value"] == "interrupted_planting"
    previous_player = prior[0]["players"][0]
    previous_player[key] = (
        ("T" if previous_player["side"] == "CT" else "CT")
        if key == "side"
        else 999
        if key == "pawn_handle"
        else "support-pawn-999999"
    )
    assert (
        p.actor_context(ACTOR, current, prior, None)["attempt_state"]["value"] == "none"
    )


@pytest.mark.parametrize(
    "mutation",
    [
        "none",
        "fact",
        "cut",
        "witness",
        "candidate",
        "missing",
        "extra",
        "reorder",
        "hash",
        "coverage",
    ],
)
def test_full_reader_roundtrip_and_tampering(tmp_path, monkeypatch, mutation):
    args = inputs()
    refs = {
        name: REF
        for name in ("support_source", "objective_source", "c4_source", "clock_source")
    }
    refs["preparation_protocol"] = c.protocol()[1]
    monkeypatch.setattr(p, "load_inputs", lambda _: (args, refs))
    report = p.materialize(Path("unused"), tmp_path)
    if mutation == "none":
        assert p.validate_artifact(Path("unused"), tmp_path) == report
        return
    if mutation in ("hash", "coverage"):
        target = tmp_path / p.MANIFEST
        value = json.loads(target.read_bytes())
        if mutation == "hash":
            value["artifact"]["sha256"] = "0" * 64
        else:
            value["coverage"]["counts"]["candidates"] += 1
        target.write_text(json.dumps(value))
    else:
        target = tmp_path / p.ROWS
        rows = [json.loads(line) for line in target.read_bytes().splitlines()]
        if mutation == "fact":
            rows[0]["actors"][0]["initial"]["round_remaining_seconds"]["value"] += 1
        elif mutation == "cut":
            rows[0]["cut"]["frame"] += 1
        elif mutation == "witness":
            rows[0]["provenance"]["order"] += 1
        elif mutation == "candidate":
            rows[0]["candidates"].pop()
        elif mutation == "missing":
            rows.pop()
        elif mutation == "extra":
            rows.append(rows[0])
        elif mutation == "reorder":
            rows.reverse()
        target.write_bytes(b"".join(c.encoded_row(row) for row in rows))
    with pytest.raises((ValueError, ValidationError)):
        p.validate_artifact(Path("unused"), tmp_path)


def test_preflight_before_source(tmp_path, monkeypatch):
    monkeypatch.setattr(
        p, "load_inputs", lambda _: pytest.fail("read before preflight")
    )
    (tmp_path / "retained").write_text("keep")
    for path in (tmp_path, Path.cwd()):
        with pytest.raises(ValueError):
            p.materialize(Path("unused"), path)


@pytest.mark.parametrize(
    "mode",
    [
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
def test_origin_invalidations_at_c0(monkeypatch, mode):
    monkeypatch.setattr(
        legacy,
        "features",
        lambda *args: {
            role: value["objective_clock"]
            for role, value in features(*args, index=2).items()
        },
    )
    legacy.test_round_origin_proof_never_crosses_unproved_clocks(mode)


@pytest.mark.parametrize("mode", ["serial", "life", "sentinel", "destroyed"])
def test_defuser_requires_complete_current_reference(monkeypatch, mode):
    monkeypatch.setattr(
        legacy,
        "features",
        lambda *args: {
            role: value["objective_clock"]
            for role, value in features(*args, index=2).items()
        },
    )
    legacy.test_identity_order_and_strict_schema(mode)


def test_same_tick_later_pause_and_public_event_do_not_backfill():
    from ai_coach.tests.test_support_c4 import ownership
    from ai_coach.tests.test_support_objective import objective

    support = inputs(2)[0]
    support["observations"]["frames"][1]["tick"] = 200
    obj = objective(support)
    own = ownership(support, obj)
    data = legacy.clock_data(support, obj, own)
    before = features(support, obj, own, data)
    obj["observations"]["events"].append(
        {
            **obj["observations"]["events"][0],
            "sequence": 2,
            "order": 15,
            "frame": 101,
            "tick": 200,
            "kind": "bomb_planted",
        }
    )
    obj["observations"]["frames"][1]["event_sequence"] = 2
    obj["observations"]["frames"][1]["rules"]["m_bBombPlanted"]["value"] = True
    data["observations"]["frames"][1]["network"]["paused"]["value"] = legacy.scalar(
        True
    )
    assert features(support, obj, own, data) == before
    later = features(support, obj, own, data, index=1)["self"]["objective_clock"]
    assert later["pause_state"] == o.fact("paused")
    assert later["round_remaining_seconds"]["status"] == "not_applicable"


def test_excluded_cut_retained():
    args = inputs(1)
    args[0]["observations"]["frames"][0]["phase"] = "postround"
    row = next(p.iter_rows(*args))
    assert row["actors"] == row["candidates"] == [] and row["provenance"] is None
    a.schema_validator(p.ROW_SCHEMA).validate(row)
