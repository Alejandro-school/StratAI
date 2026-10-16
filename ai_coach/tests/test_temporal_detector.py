"""Policy controls are software fixtures, not independent expert replay labels."""

from copy import deepcopy

import pytest

from ai_coach.interpretation import (
    context,
    contextual_io,
    detector,
    temporal,
    temporal_detector as d,
)
from ai_coach.tests.test_contextual_interpretation import (
    inputs,
    map_reference as original_map,
    UTILITY_ACTOR,
)
from ai_coach.tests.test_support_anchors import ACTOR, ALLY
from ai_coach.tests.test_support_c4 import ownership
from ai_coach.tests.test_support_objective import objective
from ai_coach.tests.test_support_physical import physical


def data(xs, ys=None, *, utility=False, tick_rate=64):
    source, _ = inputs(len(xs), utility=utility)
    for frame, x, y in zip(source["observations"]["frames"], xs, ys or xs):
        frame["tick_rate"] = tick_rate
        frame["sample_step"] = int(tick_rate / 16)
        frame["tick"] = 200 + (frame["sequence"] - 1) * frame["sample_step"]
        frame["players"][0]["position"]["x"] = round(x * 10)
        frame["players"][1]["position"]["x"] = round((y + 5) * 10)
    obj = objective(source)
    phy = physical(source)
    if utility:
        for row in phy["observations"]["frames"]:
            row["players"][-1]["context"]["facts"]["active_weapon"]["value"] = (
                "Smoke Grenade"
            )
    return source, {
        "physical_source": phy,
        "objective_source": obj,
        "c4_source": ownership(source, obj),
    }


def map_reference():
    value = original_map()
    for landmark in value["landmarks"]:
        landmark["position"]["x"] *= 10
    value["reference_id"] = temporal.digest(
        {k: v for k, v in value.items() if k != "reference_id"}
    )
    return value


def episode(source, sources, start=1, end=None):
    return context.from_sources(
        source,
        side="T",
        start_sequence=start,
        end_sequence=end or len(source["observations"]["frames"]),
        **sources,
    )


def infer(value, previous=None, package=None):
    return d.interpret(
        value,
        map_name="de_mirage",
        map_reference=package or map_reference(),
        previous=previous,
    )


def rehash(value):
    value["context_id"] = temporal.digest(
        {k: v for k, v in value.items() if k != "context_id"}
    )
    return value


def destinations(value):
    return [h["destination"] for h in value["hypotheses"]]


def test_small_correction_then_entire_group_pause_preserves_possible_entry():
    source, sources = data([0, 10, 20, 19.9, 19.8, 19.7] + [19.7] * 90)
    value = episode(source, sources)
    reading = infer(value)
    assert destinations(reading) == ["B"]
    assert (
        reading["temporal_state"]["active"][0]["motion"][ACTOR]["away_seconds"]
        == 3 / 16
    )
    assert (
        detector.interpret(value, map_name="de_mirage", map_reference=map_reference())[
            "status"
        ]
        == "abstained"
    )
    assert reading["recommendations"] == [] and reading["comparison_allowed"] is False


def test_static_positions_cannot_create_a_plan_without_prior_approach():
    source, sources = data([20] * 25)
    reading = infer(episode(source, sources))
    assert reading["status"] == "abstained"
    assert "carrier_has_no_joint_approach_evidence" in reading["abstention_reasons"]


def test_prior_support_survives_window_rollover_without_expiry_or_recursive_history():
    source, sources = data([0, 10, 20] + [20] * 80)
    previous = infer(episode(source, sources, end=3))
    original = previous["temporal_state"]["active"][0]["seed_signals"]
    for end in (25, 50, 83):
        reading = infer(
            episode(source, sources, start=previous["cut"]["sequence"] + 1, end=end),
            previous,
        )
        assert (
            destinations(reading) == ["B"] and reading["update"]["kind"] == "maintained"
        )
        assert reading["temporal_state"]["active"][0]["seed_signals"] == original
        previous = reading
    assert infer(episode(source, sources, start=60))["status"] == "abstained"


@pytest.mark.parametrize("seconds", [0.5, 1.0, 2.0])
def test_sustained_retreat_with_net_approach_still_positive_retires_old_plan(
    monkeypatch, seconds
):
    monkeypatch.setattr(d, "CONTRADICTION_SECONDS", seconds)
    steps = int(seconds * 16)
    xs = [0, 10, 20] + [20 - i / 10 for i in range(1, steps + 1)]
    source, sources = data(xs, [i / 10 for i in range(len(xs))])
    before = infer(episode(source, sources, end=len(xs) - 1))
    assert destinations(before) == ["B"]
    reading = infer(episode(source, sources), before)
    assert "B" not in destinations(reading)
    event = reading["temporal_events"][-1]
    assert event["reason"] == "sustained_carrier_retreat"
    assert event["signal"]["value"]["away_seconds"] == seconds
    assert "sustained_carrier_retreat" in reading["abstention_reasons"]


def test_full_reversal_immediate_and_positive_new_destination_required():
    source, sources = data([0, 10, 20, -1, -2, -3])
    reading = infer(episode(source, sources, end=4))
    assert reading["status"] == "abstained"
    assert reading["temporal_events"][-1]["reason"] == "carrier_approach_fully_reversed"
    changed = infer(episode(source, sources), reading)
    assert destinations(changed) == ["A"]
    assert changed["hypotheses"][0]["support_origin_cut"]["sequence"] > 4


def test_ally_sustained_retreat_is_not_ignored_while_carrier_moves_forward():
    source, sources = data(
        [0, 10, 20] + [20 + i / 10 for i in range(1, 17)],
        [0, 10, 20] + [20 - i / 10 for i in range(1, 17)],
    )
    reading = infer(episode(source, sources))
    assert reading["status"] == "abstained"
    assert reading["temporal_events"][-1]["reason"] == "sustained_ally_retreat"


@pytest.mark.parametrize(
    "change", ["gap", "identity", "c4", "position", "missing", "round"]
)
def test_pause_does_not_bridge_unavailable_or_changed_facts(change):
    source, sources = data([0, 10, 20, 20])
    prior = infer(episode(source, sources, end=3))
    value = episode(source, sources)
    row = value["samples"][-1]
    actor = row["actors"][0]
    if change == "gap":
        row["continuity"] = "unavailable"
    elif change == "identity":
        actor["identity"]["pawn_life"] = "support-pawn-000099"
    elif change == "c4":
        actor["has_c4"]["value"] = False
    elif change == "position":
        actor["position"] = {
            "status": "unavailable",
            "value": None,
            "reason": "missing_position",
        }
    elif change == "missing":
        row["actors"].pop(0)
        row["missing_previous_actors"] = [ACTOR]
    else:
        row["cut"]["round_number"] += 1
    reading = infer(rehash(value), prior)
    assert "B" not in destinations(reading)


def test_explicit_death_recomputes_survivors_and_does_not_carry_dead_utility_role():
    source, sources = data([0, 10, 20, 20], utility=True)
    prior = infer(episode(source, sources, end=3))
    assert any(
        r["actor_alias"] == UTILITY_ACTOR
        for r in prior["hypotheses"][0]["possible_functions"]
    )
    value = episode(source, sources)
    actor = next(
        a for a in value["samples"][-1]["actors"] if a["actor_alias"] == UTILITY_ACTOR
    )
    actor["alive"]["value"] = False
    actor["exclusion"] = "not_alive"
    reading = infer(rehash(value), prior)
    assert destinations(reading) == ["B"]
    assert all(
        r["actor_alias"] != UTILITY_ACTOR
        for r in reading["hypotheses"][0]["possible_functions"]
    )
    other = next(a for a in value["samples"][-1]["actors"] if a["actor_alias"] == ALLY)
    other["alive"]["value"] = False
    other["exclusion"] = "not_alive"
    assert infer(rehash(value), prior)["status"] == "abstained"


def test_prior_reading_must_match_source_map_current_overlap_and_hash():
    source, sources = data([0, 10, 20, 20])
    prior = infer(episode(source, sources, end=3))
    corrupt = deepcopy(prior)
    corrupt["temporal_state"]["active"] = []
    with pytest.raises(ValueError, match="hash"):
        infer(episode(source, sources), corrupt)
    for field in ("source_sha256", "side"):
        changed = deepcopy(prior)
        changed[field] = "0" * 64 if field == "source_sha256" else "CT"
        changed["interpretation_id"] = temporal.digest(
            {k: v for k, v in changed.items() if k != "interpretation_id"}
        )
        reading = infer(episode(source, sources, start=4), changed)
        assert (
            reading["status"] == "abstained"
            and reading["update"]["kind"] == "prior_context_unavailable"
        )
    value = episode(source, sources, start=3)
    value["samples"][0]["actors"][0]["position"]["value"]["x"] += 1
    reading = infer(rehash(value), prior)
    assert reading["update"]["kind"] == "prior_context_unavailable"


@pytest.mark.parametrize("mutation", ["future", "enemy", "answers"])
def test_causal_prefix_does_not_read_future_opponents_or_expert_answer(mutation):
    source, sources = data([0, 10, 20, 19.9, 19.8, 19.7, 19.7])
    expected = infer(episode(source, sources, end=6))
    if mutation == "future":
        source["observations"]["frames"][6:] = [None]
        for key in ("physical_source", "objective_source"):
            sources[key]["observations"]["frames"][6:] = [None]
        for key in ("cuts", "states"):
            sources["c4_source"]["observations"][key][6:] = [None]
    elif mutation == "enemy":
        for row in source["observations"]["frames"]:
            row["players"][2]["position"] = {"x": float("nan")}
    else:
        source.update(expert_answer="execute A", case="R2", outcome="win")
    assert infer(episode(source, sources, end=6)) == expected


def test_both_configurations_are_explicit_and_offline_validation_preserves_baseline(
    tmp_path, monkeypatch
):
    source, sources = data([0, 10, 20, 19.9, 19.8, 19.7])
    monkeypatch.setattr(
        contextual_io,
        "load_inputs",
        lambda *_: (source, sources, {}, "de_mirage", map_reference()),
    )
    for configuration in (detector.CONFIGURATION, d.CONFIGURATION):
        output = tmp_path / configuration.replace("@", "-")
        output.mkdir()
        manifest = contextual_io.materialize(
            tmp_path,
            output,
            side="T",
            intervals=[(1, 3), (1, 6)],
            configuration=configuration,
        )
        assert contextual_io.validate(tmp_path, output) == manifest
        assert manifest["configuration"] == configuration
    assert "posibilidad conservada" in (output / "contextual_report.md").read_text(
        encoding="utf-8"
    )


def test_review_of_one_ally_preserves_other_paths_for_new_support_after_death():
    source, sources = data(
        [0, 10, 20] + [20 + i / 10 for i in range(1, 22)] + [22.0], utility=True
    )
    value = episode(source, sources)
    for i, row in enumerate(value["samples"]):
        actor = next(a for a in row["actors"] if a["actor_alias"] == UTILITY_ACTOR)
        actor["position"]["value"]["x"] = -10 - max(0, i - 2) * 10
        if i >= 21:
            actor["alive"]["value"] = False
            actor["exclusion"] = "not_alive"
    reading = infer(rehash(value))
    assert destinations(reading) == ["B"]
    assert reading["temporal_events"][0]["reason"] == "sustained_ally_retreat"
    assert (
        reading["hypotheses"][0]["support_origin_cut"]["sequence"]
        > reading["temporal_events"][0]["sequence"]
    )
    assert reading["hypotheses"][0]["supporting_survivors"] == [ACTOR, ALLY]


def test_retired_plan_does_not_resurrect_during_a_static_pause():
    source, sources = data([0, 10, 20, -1] + [-1] * 35)
    reading = infer(episode(source, sources))
    assert reading["status"] == "abstained"
    assert (
        "fresh_carrier_approach_required_after_review" in reading["abstention_reasons"]
    )


@pytest.mark.parametrize("tick_rate", [32, 128])
def test_motion_confirmation_uses_native_clock_across_tick_rates(tick_rate):
    source, sources = data(
        [0, 10, 20] + [20 - i / 10 for i in range(1, 17)], tick_rate=tick_rate
    )
    assert destinations(infer(episode(source, sources, end=18))) == ["B"]
    reading = infer(episode(source, sources))
    assert reading["status"] == "abstained"
    assert reading["temporal_events"][0]["signal"]["value"]["away_seconds"] == 1.0


def test_equipped_utility_role_is_refreshed_and_not_copied_from_prior_reading():
    source, sources = data([0, 10, 20, 20], utility=True)
    prior = infer(episode(source, sources, end=3))
    value = episode(source, sources)
    actor = next(
        a for a in value["samples"][-1]["actors"] if a["actor_alias"] == UTILITY_ACTOR
    )
    actor["physical"]["active_weapon"]["value"] = "AK-47"
    reading = infer(rehash(value), prior)
    assert destinations(reading) == ["B"]
    assert [r["function"] for r in reading["hypotheses"][0]["possible_functions"]] == [
        "c4_carrier"
    ]


def test_result_schema_and_causality_reject_tampering_even_with_recomputed_hash():
    source, sources = data([0, 10, 20])
    reading = infer(episode(source, sources))
    reading["hypotheses"][0]["signals"][0]["end_sequence"] = 99
    reading["interpretation_id"] = temporal.digest(
        {k: v for k, v in reading.items() if k != "interpretation_id"}
    )
    with pytest.raises(ValueError, match="future temporal"):
        d.validate(reading)
