import pytest

from ai_coach.interpretation.replay_event_context import checked_utility
from ai_coach.interpretation.replay_reading_context import ReplayReadingSession
from ai_coach.tests.test_replay_events import packet


def test_firearm_event_cannot_smuggle_enemy_health_through_other_atomic_fields():
    data = packet()
    data['views'][0]['events'][-1].update(victim_id='8', victim_team='CT', damage=73, health=27)
    result = ReplayReadingSession(data).consultation().initial()
    event = next(e for e in result['events'] if e['key'] == 'weapon_fire')['value']
    for field in ('victim_id', 'victim_team', 'damage', 'health'):
        assert field not in event


def test_own_damage_with_unidentified_attacker_does_not_restore_hidden_weapon():
    data = packet()
    view = data['views'][0]
    own = next(str(p['steam_id']) for p in view['frame']['players'] if p['team'] == 'T')
    view['events'][-1] = {
        'id': 'hurt-own', 'type': 'player_hurt', 'tick': 10, 'availability_tick': 10,
        'victim_id': own, 'victim_team': 'T', 'damage': 15, 'health': 85,
        'knowledge': 'own_team_event', 'weapon': 'Hidden attacker weapon', 'actor_team': 'CT',
        'source': {'kind': 'demo_atomic_event', 'reference': 'hurt-own', 'native_source': 'PlayerHurt'},
    }
    result = ReplayReadingSession(data).consultation().initial()
    event = next(e for e in result['events'] if e['key'] == 'player_hurt')['value']
    assert event['damage'] == 15
    assert 'weapon' not in event
    assert 'actor_team' not in event


def test_own_event_does_not_preserve_an_unvalidated_enemy_observation():
    data = packet()
    data['views'][0]['events'][-1]['observation'] = {
        'observer_id': '8', 'tick': 9000,
        'source': {'kind': 'go_calculated_contact', 'reference': 'future-hidden-contact'},
    }
    result = ReplayReadingSession(data).consultation().initial()
    event = next(e for e in result['events'] if e['key'] == 'weapon_fire')['value']
    assert 'observation' not in event


@pytest.mark.parametrize('terminal', ['utility_start', 'utility_end'])
def test_utility_consumer_rejects_a_flight_already_terminated_by_available_event(terminal):
    view = {'tick': 11, 'availability_tick': 10, 'frame': {
        'players': [{'steam_id': '1', 'team': 'T'}],
        'projectiles': [{'id': '31', 'thrower_id': '1', 'type': 'smoke', 'x': 10, 'y': 20,
            'observation_tick': 10, 'availability_tick': 10,
            'knowledge': 'simulated_team_utility_tracking',
            'source': {'reference': 'frame:10:projectile:31'}}],
        'active_effects': [],
    }}
    events = [
        {'type': 'grenade_throw', 'tick': 10, 'availability_tick': 10, 'actor_id': '1', 'projectile_id': '31'},
        {'type': terminal, 'tick': 11, 'availability_tick': 11, 'actor_id': '1', 'projectile_id': '31'},
    ]
    with pytest.raises(ValueError):
        checked_utility(view, events)
