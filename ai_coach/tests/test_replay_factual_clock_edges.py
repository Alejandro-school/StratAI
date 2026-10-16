"""Adversarial clock cases reproduced independently after the first GREEN."""
import pytest

from ai_coach.interpretation.replay_reading_context import ReplayReadingSession
from ai_coach.tests.test_replay_factual_round_guards import factual_packet, factual_view, round_fact


def checked_clock_fixture():
    value = factual_packet(factual_view(104))
    # Every negative starts from accepted, independently recomputed native evidence.
    assert round_fact(value)["round_remaining_seconds"] == 114.9375
    return value, value["views"][0]["frame"]["round_context"]


@pytest.mark.parametrize("engine", [None, False])
def test_factual_clock_allows_native_boolean_or_absent_engine_pause(engine):
    packet, context = checked_clock_fixture()
    context["clock_evidence"]["engine_paused"] = engine
    assert round_fact(packet)["round_remaining_seconds"] == 114.9375


@pytest.mark.parametrize("engine", [1, 0, "paused", "false", [], {}])
def test_factual_clock_cannot_treat_malformed_engine_pause_as_running(engine):
    packet, context = checked_clock_fixture()
    context["clock_evidence"]["engine_paused"] = engine
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


@pytest.mark.parametrize("tick", [999, 0, -1, True])
def test_factual_clock_freeze_witness_is_causal_even_when_elapsed_is_absent(tick):
    packet, context = checked_clock_fixture()
    context.update(freeze_end_tick=tick, elapsed_seconds=None)
    context["clock_evidence"]["origin_witness"].update(tick=tick, freeze_end_tick=tick)
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


@pytest.mark.parametrize("frame", [-1, True, "100", 100.0, 105])
def test_factual_clock_freeze_frame_requires_a_prior_integer_frame(frame):
    packet, context = checked_clock_fixture()
    context["clock_evidence"]["freeze_frame"] = frame
    context["clock_evidence"]["origin_witness"]["frame"] = frame
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)


@pytest.mark.parametrize("message", ["tick_message", "packet_message"])
def test_factual_clock_current_message_frame_cannot_use_numeric_equality_to_bypass_type(message):
    packet, context = checked_clock_fixture()
    context["clock_evidence"][message]["frame"] = 104.0
    with pytest.raises(ValueError):
        ReplayReadingSession(packet)
