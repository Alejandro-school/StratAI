"""Causal kill-feed context for review cards, separate from visual perception.

No spectator trajectory is exported. Positions are callback observations at a
kill; the weapon is the killing weapon, not an inferred weapon still held later.
"""
import bisect
import hashlib
import json
import math
import re
from collections import Counter
from pathlib import Path

from jsonschema import Draft202012Validator

SCHEMA = Path(__file__).parents[1] / 'contracts/review_kill_context.schema.json'


def require(condition, message):
    if not condition:
        raise ValueError(message)


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def actor_alias(checksum, native_id):
    if not isinstance(native_id, str) or not re.fullmatch(r'steam:[1-9][0-9]*', native_id):
        return None
    return 'support-actor-' + digest((checksum + '\n' + native_id).encode())[:20]


def blocks(html):
    return {key: json.loads(re.search(
        '<script id="' + key + '" type="application/json">([\\s\\S]*?)</script>', html
    )[1]) for key in ('data', 'round-data')}


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
    refs = [r for r in manifest['artifacts'] if r['artifact_type'] == 'combat_events']
    require(len(refs) == 1 and refs[0]['schema_id'] == 'stratai.combat_event@2', 'combat contract differs')
    ref = refs[0]
    combat_path = (path.parent / ref['path']).resolve()
    require(combat_path.is_relative_to(path.parent), 'combat path escapes bundle')
    combat_raw = combat_path.read_bytes()
    require(len(combat_raw) == ref['bytes'] and digest(combat_raw) == ref['sha256'], 'combat binding differs')
    rows = [json.loads(line) for line in combat_raw.splitlines()]
    require(len(rows) == ref['record_count'], 'combat count differs')
    require(all(r['schema_id'] == ref['schema_id'] and r['match_id'] == manifest['match_id'] for r in rows), 'combat identity differs')
    return support, [r for r in rows if r['event_type'] == 'kill'], {
        'support_sha256': digest(raw), 'canonical_manifest_sha256': digest(manifest_raw),
        'combat_sha256': digest(combat_raw), 'combat_record_count': len(rows),
    }


def position(event, role):
    value = event.get(role + '_position')
    return value if (
        event.get(role + '_position_status') == 'observed'
        and event.get(role + '_position_source') == 'callback_player_position'
        and isinstance(value, dict)
        and all(type(value.get(axis)) in (int, float) and math.isfinite(value[axis]) for axis in ('x', 'y', 'z'))
    ) else None


def project(support, kills, card, case_id, source_refs):
    data, clocks = card['data'], card['round-data']
    require(len(data['frames']) == len(clocks['frames']) > 0, 'card cuts differ')
    index = {f['sequence']: f for f in support['observations']['frames']}
    require(len(index) == len(support['observations']['frames']), 'duplicate sample sequence')
    frames = [index[c['source_sequence']] for c in clocks['frames']]
    require(len({(f['round_number'], f['instance'], f['epoch']) for f in frames}) == 1, 'card crosses round or epoch')
    ticks = [f['tick'] for f in frames]
    require(all(a < b for a, b in zip(ticks, ticks[1:])), 'card ticks not increasing')
    own_sides = {p['identity']['side'] for f in data['frames'] for p in f['actors']}
    require(len(own_sides) == 1, 'card side ambiguous')
    own_side = own_sides.pop()
    labels = {}
    for side in ('T', 'CT'):
        aliases = sorted({p['actor_alias'] for f in frames for p in f['players'] if p['side'] == side})
        labels.update({a: f'{side}{n+1}' for n, a in enumerate(aliases)})
    for f, rendered in zip(frames, data['frames']):
        require(f['round_number'] == rendered['round'], 'card round differs')
        own = {labels[p['actor_alias']]: p for p in f['players'] if p['side'] == own_side}
        require(set(own) == {p['label'] for p in rendered['actors']}, 'card roster differs')
        for p in rendered['actors']:
            original = own[p['label']]
            require(all(original[k] == p['identity'][k] for k in ('side', 'pawn_handle', 'pawn_life', 'identity_status')), 'card identity differs')
            require(p['alive'] == (original['alive'] if original['alive_status'] == 'observed' else None), 'card alive state differs')
    events, omitted = [], Counter()
    event_ids = set()
    ordered = sorted(kills, key=lambda e: (e['round_number'], e['tick'], e['sequence_in_tick'], e['event_id']))
    for e in ordered:
        require(e['event_id'] not in event_ids, 'duplicate kill event')
        event_ids.add(e['event_id'])
        if e['round_number'] != frames[0]['round_number'] or not ticks[0] <= e['tick'] <= ticks[-1]:
            continue
        i = bisect.bisect_left(ticks, e['tick'])
        if clocks['frames'][i]['eligibility']['kind'] in ('knife', 'warmup'):
            omitted['excluded_round'] += 1
            continue
        if e.get('tick_status') != 'observed' or e.get('source') != 'demoinfocs.kill':
            omitted['event_source_unavailable'] += 1
            continue
        if i and (data['frames'][i]['continuity'] != 'adjacent_samples' or frames[i]['sequence'] != frames[i-1]['sequence'] + 1):
            omitted['sample_gap'] += 1
            continue
        participants = {}
        for role in ('actor', 'target'):
            alias = actor_alias(support['source_sha256'], e.get(role + '_player_id'))
            players = [p for p in frames[i]['players'] if p['actor_alias'] == alias]
            p = players[0] if len(players) == 1 else None
            if p and i:
                earlier = [old for old in frames[i-1]['players'] if old['actor_alias'] == alias]
                if len(earlier) != 1 or any(earlier[0][key] != p[key] for key in ('side', 'pawn_life', 'pawn_handle')):
                    p = None
            if (p and e.get(role + '_status') == 'observed' and p['identity_status'] == 'observed'
                    and p['side'].lower() == e.get(role + '_side')):
                participants[role] = {'label': labels[alias], 'side': p['side'], 'life': p['pawn_life'], 'position': position(e, role)}
            else:
                participants[role] = None
        victim = participants['target']
        if victim is None:
            omitted['victim_identity_unavailable'] += 1
            continue
        victim_sample = next(p for p in frames[i]['players'] if labels[p['actor_alias']] == victim['label'])
        if victim_sample['alive_status'] != 'observed' or victim_sample['alive'] is not False:
            omitted['victim_death_not_confirmed_at_cut'] += 1
            continue
        # Time is reconstructed only from the preceding clock observation.
        # Availability is the first sampled cut at/after the native event tick.
        j = bisect.bisect_right(ticks, e['tick']) - 1
        clock = clocks['frames'][j]['round_elapsed_seconds']
        native = clocks['frames'][j].get('native_clock', {})
        pause = native.get('pause_state', {}).get('value')
        seconds = None
        if clock['status'] == 'observed' and (ticks[j] == e['tick'] or pause == 'running'):
            seconds = clock['value'] + (e['tick'] - ticks[j]) / frames[j]['tick_rate']
        weapon = e.get('weapon') if e.get('weapon_status') == 'observed' and e.get('weapon_source') == 'callback_weapon' else None
        events.append({'id': 'kill-' + digest(e['event_id'].encode())[:20], 'available_index': i,
            'available_sequence': frames[i]['sequence'], 'tick': e['tick'], 'round': e['round_number'],
            'elapsed_seconds': seconds, 'attacker': participants['actor'], 'victim': victim,
            'relation': e['relation'], 'weapon': weapon, 'weapon_is_utility': e.get('weapon_is_utility'),
            'through_smoke': e.get('through_smoke'), 'penetrated_objects': e.get('penetrated_objects'),
            'source': 'kill_callback', 'position_meaning': 'event_position_not_current_or_confirmed_perception'})
    result = {'schema_id': 'stratai.review_kill_context@1', 'case_id': case_id,
        'source_alias': support['source_alias'], 'source_sha256': support['source_sha256'],
        'frame_count': len(frames), 'side': own_side, 'events': events,
        'source_refs': source_refs, 'omissions': dict(omitted), 'perception_proven': False}
    Draft202012Validator(json.loads(SCHEMA.read_bytes())).validate(result)
    return result
