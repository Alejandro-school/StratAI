"""Exact-cut ground C4 coordinates for T review cards, without actor fallback."""
import gzip
import json
import math
from collections import Counter
from pathlib import Path

from jsonschema import Draft202012Validator

from ai_coach.interpretation.kill_context import digest, require

SCHEMA = Path(__file__).parents[1] / 'contracts/review_bomb_context.schema.json'


def load_source(source):
    source = Path(source).resolve()
    raw = (source / 'support_observations.json').read_bytes()
    support = json.loads(raw)
    require(support['schema_id'] == 'stratai.support_observations@1', 'support schema differs')
    ref = support['base_manifest']
    path = (source / ref['path']).resolve()
    require(path.is_relative_to(source), 'manifest escapes source')
    manifest_raw = path.read_bytes()
    require(len(manifest_raw) == ref['bytes'] and digest(manifest_raw) == ref['sha256'], 'manifest binding differs')
    manifest = json.loads(manifest_raw)
    require(manifest['schema_id'] == 'stratai.canonical_manifest@3', 'canonical schema differs')
    require(manifest['demo_checksum_sha256'] == support['source_sha256'], 'demo checksum differs')
    rounds, references = {}, {}
    for ref in manifest['artifacts']:
        if ref['artifact_type'] != 'replay_round':
            continue
        require(ref['schema_id'] == 'stratai.replay_round@5', 'replay contract differs')
        replay_path = (path.parent / ref['path']).resolve()
        require(replay_path.is_relative_to(path.parent), 'replay escapes bundle')
        raw = replay_path.read_bytes()
        require(len(raw) == ref['bytes'] and digest(raw) == ref['sha256'], 'replay binding differs')
        replay = json.loads(gzip.decompress(raw))
        require(replay['schema_id'] == ref['schema_id'] and replay['match_id'] == manifest['match_id'], 'replay identity differs')
        number = replay['round']['round']
        require(number not in rounds, 'duplicate replay round')
        frames = replay['round']['frames']
        ticks = [f['tick'] for f in frames]
        require(all(type(t) is int for t in ticks) and all(a < b for a, b in zip(ticks, ticks[1:])), 'replay ticks not increasing')
        # Do not pass names, Steam IDs, opponents or other replay data to the projection.
        rounds[number] = {f['tick']: f.get('bomb') for f in frames}
        references[number] = {'replay_round_sha256': digest(raw), 'replay_frame_count': len(frames)}
    return support, rounds, {
        'support_sha256': digest((source / 'support_observations.json').read_bytes()),
        'canonical_manifest_sha256': digest(manifest_raw), 'rounds': references,
    }


def project(support, replays, card, case_id, refs):
    data, clocks = card['data']['frames'], card['round-data']['frames']
    require(len(data) == len(clocks) > 0, 'card cuts differ')
    by_sequence = {f['sequence']: f for f in support['observations']['frames']}
    require(len(by_sequence) == len(support['observations']['frames']), 'duplicate support sample')
    frames = [by_sequence[c['source_sequence']] for c in clocks]
    require(len({(f['round_number'], f['instance'], f['epoch']) for f in frames}) == 1, 'card crosses round or epoch')
    require(all(a['tick'] < b['tick'] for a, b in zip(frames, frames[1:])), 'card ticks not increasing')
    sides = {p['identity']['side'] for f in data for p in f['actors']}
    require(len(sides) == 1, 'ambiguous review side')
    side = sides.pop()
    number = frames[0]['round_number']
    require(number in replays, 'replay round absent')
    labels = sorted({p['actor_alias'] for f in frames for p in f['players'] if p['side'] == side})
    cuts, counts = [], Counter()
    for index, (frame, rendered, clock) in enumerate(zip(frames, data, clocks)):
        require(frame['round_number'] == rendered['round'], 'card round differs')
        own = {side + str(labels.index(p['actor_alias']) + 1): p for p in frame['players'] if p['side'] == side}
        require(set(own) == {p['label'] for p in rendered['actors']}, 'card roster differs')
        for p in rendered['actors']:
            require(all(own[p['label']][k] == p['identity'][k] for k in ('side', 'pawn_handle', 'pawn_life', 'identity_status')), 'card identity differs')
        bomb = replays[number].get(frame['tick'])
        status, position = 'not_dropped', None
        if side != 'T':
            status = 'review_side_not_eligible'
        elif clock['eligibility']['kind'] != 'bomb_objective':
            status = 'excluded'
        elif bomb is None:
            status = 'frame_unavailable'
        elif bomb.get('state') == 'dropped':
            if (bomb.get('objective_phase') != 'preplant' or bomb.get('carrier_id') is not None
                    or bomb.get('is_planted_now') is not False or rendered.get('planted') is not False
                    or rendered.get('carriers') or any(p.get('c4') is True for p in rendered['actors'])):
                status = 'state_conflict'
            elif (bomb.get('position_status') != 'observed' or bomb.get('position_source') != 'demoinfocs_native_snapshot'
                    or not all(type(bomb.get(k)) in (int, float) and math.isfinite(bomb[k]) for k in ('x', 'y'))):
                status = 'position_unavailable'
            else:
                status, position = 'dropped_position', {k: bomb[k] for k in ('x', 'y')}
        cuts.append({'index': index, 'source_sequence': frame['sequence'], 'tick': frame['tick'],
                     'round': number, 'status': status, 'position': position})
        counts[status] += 1
    result = {'schema_id': 'stratai.review_bomb_context@1', 'case_id': case_id,
              'source_alias': support['source_alias'], 'source_sha256': support['source_sha256'],
              'side': side, 'frame_count': len(cuts), 'frames': cuts, 'counts': dict(counts),
              'source': 'replay_bomb_native_snapshot_exact_tick',
              'source_refs': {k: v for k, v in refs.items() if k != 'rounds'} | refs['rounds'][number]}
    Draft202012Validator(json.loads(SCHEMA.read_bytes())).validate(result)
    return result
