"""Bomb points come from the object at the exact cut, never the dropper."""
import gzip
import json
from copy import deepcopy

import pytest

from ai_coach.interpretation.kill_context import digest
from ai_coach.interpretation.review_bomb_context import load_source, project
from ai_coach.tests.test_kill_context import fixture


def setup():
    support, _, card, _ = fixture()
    support['source_alias'] = 'demo-' + 'a'*20
    for frame in card['data']['frames']:
        frame.update(planted=False, carriers=[])
        frame['actors'][0]['c4'] = False
    rounds = {1: {tick: {'state': 'dropped', 'objective_phase': 'preplant', 'is_planted_now': False,
                        'carrier_id': None, 'position_status': 'observed',
                        'position_source': 'demoinfocs_native_snapshot', 'x': tick, 'y': -tick}
                  for tick in (100, 104, 108)}}
    refs = {'support_sha256': 'a'*64, 'canonical_manifest_sha256': 'b'*64,
            'rounds': {1: {'replay_round_sha256': 'c'*64, 'replay_frame_count': 3}}}
    return support, rounds, card, 'E03', refs


def test_ground_position_updates_at_exact_cut_and_never_uses_future_or_actor_position():
    args = setup()
    full = project(*args)
    assert [f['position']['x'] for f in full['frames']] == [100, 104, 108]
    assert 'steam:' not in str(full)
    prefix = deepcopy(args)
    for block in prefix[2].values():
        block['frames'] = block['frames'][:1]
    assert project(*prefix)['frames'] == full['frames'][:1]
    args[1][1][112] = {**args[1][1][108], 'x': 900}
    assert project(*args) == full


@pytest.mark.parametrize('change,status', [
    ({'state': 'carried'}, 'not_dropped'), ({'state': 'planted'}, 'not_dropped'),
    ({'position_source': 'demoinfocs_event'}, 'position_unavailable'),
    ({'position_status': 'unavailable'}, 'position_unavailable'),
    ({'x': float('nan')}, 'position_unavailable'),
    ({'carrier_id': 'unexpected'}, 'state_conflict'),
    ({'is_planted_now': True}, 'state_conflict'),
    ({'objective_phase': 'resolved'}, 'state_conflict'),
])
def test_invalid_or_no_longer_dropped_bomb_removes_the_marker(change, status):
    args = setup(); args[1][1][104].update(change)
    frame = project(*args)['frames'][1]
    assert frame['position'] is None and frame['status'] == status


def test_missing_exact_frame_does_not_carry_forward_the_last_point():
    args = setup(); del args[1][1][104]
    result = project(*args)
    assert result['frames'][1]['status'] == 'frame_unavailable'
    assert result['frames'][1]['position'] is None
    assert result['frames'][2]['position'] == {'x': 108, 'y': -108}


@pytest.mark.parametrize('mode', ['knife', 'warmup', 'unknown', 'plant', 'carrier', 'identity'])
def test_ineligible_or_conflicting_card_does_not_show_ground_bomb(mode):
    args = setup()
    if mode in ('knife', 'warmup', 'unknown'):
        args[2]['round-data']['frames'][1]['eligibility']['kind'] = mode
    elif mode == 'plant':
        args[2]['data']['frames'][1]['planted'] = True
    elif mode == 'carrier':
        args[2]['data']['frames'][1]['carriers'] = ['T1']
    else:
        args[2]['data']['frames'][1]['actors'][0]['identity']['pawn_life'] = 'different'
        with pytest.raises(ValueError, match='identity'):
            project(*args)
        return
    assert project(*args)['frames'][1]['position'] is None


def test_ct_review_does_not_gain_global_bomb_information():
    args = setup()
    for frame in args[2]['data']['frames']:
        frame['actors'][0]['label'] = 'CT1'
        frame['actors'][0]['identity'] = {k: args[0]['observations']['frames'][0]['players'][1][k]
                                        for k in ('side', 'pawn_handle', 'pawn_life', 'identity_status')}
    assert all(f['status'] == 'review_side_not_eligible' for f in project(*args)['frames'])


def test_loader_rejects_changed_bound_replay(tmp_path):
    support, rounds, _, _, _ = setup()
    replay = {'schema_id': 'stratai.replay_round@5', 'match_id': 'opaque', 'round': {'round': 1,
              'frames': [{'tick': t, 'bomb': b} for t, b in rounds[1].items()]}}
    raw = gzip.compress(json.dumps(replay).encode())
    (tmp_path/'round.json.gz').write_bytes(raw)
    manifest = {'schema_id': 'stratai.canonical_manifest@3', 'demo_checksum_sha256': support['source_sha256'],
                'match_id': 'opaque', 'artifacts': [{'artifact_type': 'replay_round', 'schema_id': replay['schema_id'],
                'path': 'round.json.gz', 'sha256': digest(raw), 'bytes': len(raw)}]}
    raw = json.dumps(manifest).encode(); (tmp_path/'manifest.json').write_bytes(raw)
    support.update(schema_id='stratai.support_observations@1', base_manifest={'path': 'manifest.json', 'bytes': len(raw), 'sha256': digest(raw)})
    (tmp_path/'support_observations.json').write_text(json.dumps(support), encoding='utf-8')
    assert load_source(tmp_path)[1] == rounds
    (tmp_path/'round.json.gz').write_bytes(b'changed')
    with pytest.raises(ValueError, match='replay binding'):
        load_source(tmp_path)
