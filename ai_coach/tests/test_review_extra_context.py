"""Review context preserves event identity, timing and the absence of knowledge."""
from copy import deepcopy

import pytest

from ai_coach.interpretation.review_extra_context import project
from ai_coach.tests.test_kill_context import fixture


def setup():
    support, kills, card, refs = fixture()
    damage = {**kills[0], 'event_id': 'damage', 'event_type': 'player_hurt',
              'source': 'demoinfocs.player_hurt', 'damage_status': 'observed',
              'health_damage_taken': 20, 'armor_damage_taken': 3}
    kills[0]['event_type'] = 'kill'
    drops = [{'event_id': 'drop', 'event_type': 'bomb_drop', 'tick': 102,
              'round_number': 1, 'sequence_in_tick': 1, 'actor_player_id': 'steam:10',
              'actor_side': 't', 'source': 'demoinfocs_event'}]
    refs = {**{k: v for k, v in refs.items() if not k.startswith('combat_')},
            'combat_events_sha256': 'a'*64, 'combat_events_record_count': 2,
            'objective_events_sha256': 'b'*64, 'objective_events_record_count': 1}
    return support, [damage, *kills], drops, card, 'E03', refs


def test_damage_positions_weapon_and_drop_are_native_and_causal():
    value = project(*setup())
    assert value['damage'][0]['attacker']['label'] == 'CT1'
    assert value['damage'][0]['health_damage'] == 20
    assert value['damage'][0]['available_index'] == 1
    assert value['damage'][0]['elapsed_seconds'] == 2/64
    assert value['c4'][0]['actor']['label'] == 'T1'
    assert value['c4'][0]['death_same_tick'] is True
    assert 'steam:' not in str(value) and not value['perception_proven']


def test_drop_is_not_invented_from_missing_carrier_and_pickup_remains_separate():
    args = list(setup()); args[2] = []
    assert project(*args)['c4'] == []
    args = list(setup()); args[1] = args[1][:1]
    args[2].append({**args[2][0], 'event_id': 'pickup', 'event_type': 'bomb_pickup', 'tick': 104})
    result = project(*args)
    assert len(result['c4']) == 2 and not result['c4'][0]['death_same_tick']
    assert result['c4'][1]['kind'] == 'bomb_pickup'


def test_no_positive_damage_and_unknown_actor_do_not_reveal_enemy_or_dropper():
    args = list(setup()); args[1][0].update(health_damage_taken=0, armor_damage_taken=0)
    args[2][0]['actor_player_id'] = None
    result = project(*args)
    assert not result['damage'] and not result['c4']
    assert result['omissions'] == {'positive_damage_unavailable': 1, 'objective_actor_unavailable': 1}


@pytest.mark.parametrize('mode', ['gap', 'life', 'knife'])
def test_unavailable_cuts_do_not_join_events(mode):
    args = list(setup()); support, _, _, card, _, _ = args
    if mode == 'gap': card['data']['frames'][1]['continuity'] = 'unavailable'
    if mode == 'knife': card['round-data']['frames'][1]['eligibility']['kind'] = 'knife'
    if mode == 'life':
        support['observations']['frames'][1]['players'][0]['pawn_life'] = 'other'
        card['data']['frames'][1]['actors'][0]['identity']['pawn_life'] = 'other'
    result = project(*args)
    assert not result['damage'] and not result['c4']


def test_unknown_clock_or_position_stays_unknown_and_future_is_excluded():
    args = list(setup()); args[3]['round-data']['frames'][0]['round_elapsed_seconds'] = {'status': 'unavailable', 'value': None}
    args[1][0]['actor_position_status'] = 'unavailable'
    result = project(*args)
    assert result['damage'][0]['attacker']['position'] is None
    assert result['c4'][0]['elapsed_seconds'] is None
    shorter = deepcopy(args)
    for block in shorter[3].values(): block['frames'] = block['frames'][:1]
    assert not project(*shorter)['damage'] and not project(*shorter)['c4']
    args[2].append({**args[2][0], 'event_id': 'future', 'tick': 110})
    assert project(*args) == result


def test_duplicate_events_and_card_mismatch_fail_closed():
    args = list(setup()); args[2] += args[2]
    with pytest.raises(ValueError, match='duplicate'): project(*args)
    args = list(setup()); args[3]['data']['frames'][0]['round'] = 2
    with pytest.raises(ValueError, match='round'): project(*args)
