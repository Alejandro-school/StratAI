"""Numbers in factual prose keep the recorded precision, including tick fractions."""
import pytest

from ai_coach.interpretation.replay_factual_reading import number, describe


@pytest.mark.parametrize('value,text', [(114.9375,'114.9375'), (14.984375,'14.984375'),
                                     (-.015625,'-0.015625'), (100.0,'100'), (70,'70')])
def test_factual_number_does_not_silently_round_the_recorded_value(value, text):
    assert number(value) == text


def test_factual_countdown_prose_preserves_the_native_fraction():
    assert describe({'key':'round_remaining_seconds','value':114.9375,'subject':None}) == 'Cuenta atrás de ronda: 114.9375 segundos.'
