"""Native damage and C4 events for human review; no inferred visual perception."""
import bisect
import json
from collections import Counter
from pathlib import Path

from jsonschema import Draft202012Validator

from ai_coach.interpretation.kill_context import actor_alias, digest, position, require
from ai_coach.interpretation.kill_context import project as project_kills

SCHEMA = Path(__file__).parents[1] / 'contracts/review_extra_context.schema.json'


def load_source(source):
    source = Path(source).resolve()
    raw = (source / 'support_observations.json').read_bytes()
    support = json.loads(raw)
    require(support['schema_id'] == 'stratai.support_observations@1', 'support schema differs')
    ref = support['base_manifest']
    manifest_path = (source / ref['path']).resolve()
    require(manifest_path.is_relative_to(source), 'manifest escapes source')
    manifest_raw = manifest_path.read_bytes()
    require(len(manifest_raw) == ref['bytes'] and digest(manifest_raw) == ref['sha256'], 'manifest binding differs')
    manifest = json.loads(manifest_raw)
    require(manifest['schema_id'] == 'stratai.canonical_manifest@3', 'canonical schema differs')
    require(manifest['demo_checksum_sha256'] == support['source_sha256'], 'demo checksum differs')
    refs = {'support_sha256': digest(raw), 'canonical_manifest_sha256': digest(manifest_raw)}
    streams = {}
    for kind, schema in [('combat_events', 'stratai.combat_event@2'), ('objective_events', 'stratai.objective_event@2')]:
        matches = [r for r in manifest['artifacts'] if r['artifact_type'] == kind]
        require(len(matches) == 1 and matches[0]['schema_id'] == schema, 'event contract differs')
        ref = matches[0]
        path = (manifest_path.parent / ref['path']).resolve()
        require(path.is_relative_to(manifest_path.parent), 'event path escapes bundle')
        raw = path.read_bytes()
        require(len(raw) == ref['bytes'] and digest(raw) == ref['sha256'], 'event binding differs')
        rows = [json.loads(line) for line in raw.splitlines()]
        require(len(rows) == ref['record_count'], 'event count differs')
        require(all(r['schema_id'] == schema and r['match_id'] == manifest['match_id'] for r in rows), 'event identity differs')
        refs[kind + '_sha256'] = digest(raw)
        refs[kind + '_record_count'] = len(rows)
        streams[kind] = rows
    return support, streams['combat_events'], streams['objective_events'], refs


def project(support, combat, objective, card, case_id, refs):
    # Reuse the established card/source identity and round binding, without changing kills.
    project_kills(support, [], card, case_id, {
        'support_sha256': refs['support_sha256'], 'canonical_manifest_sha256': refs['canonical_manifest_sha256'],
        'combat_sha256': refs['combat_events_sha256'], 'combat_record_count': refs['combat_events_record_count']})
    data, clocks = card['data']['frames'], card['round-data']['frames']
    by_sequence = {f['sequence']: f for f in support['observations']['frames']}
    frames = [by_sequence[c['source_sequence']] for c in clocks]
    ticks = [f['tick'] for f in frames]
    own_side = data[0]['actors'][0]['identity']['side']
    labels = {}
    for side in ('T', 'CT'):
        aliases = sorted({p['actor_alias'] for f in frames for p in f['players'] if p['side'] == side})
        labels.update({alias: side + str(i + 1) for i, alias in enumerate(aliases)})
    omissions, damage, c4 = Counter(), [], []

    def participant(event, role, index, objective_event=False):
        alias = actor_alias(support['source_sha256'], event.get(role + '_player_id'))
        matches = [p for p in frames[index]['players'] if p['actor_alias'] == alias]
        p = matches[0] if len(matches) == 1 else None
        if p and index:
            previous = [old for old in frames[index-1]['players'] if old['actor_alias'] == alias]
            if len(previous) != 1 or any(previous[0][k] != p[k] for k in ('side', 'pawn_life', 'pawn_handle')):
                p = None
        if not p or p['identity_status'] != 'observed' or p['side'].lower() != event.get(role + '_side'):
            return None
        if not objective_event and event.get(role + '_status') != 'observed':
            return None
        return {'label': labels[alias], 'side': p['side'], 'life': p['pawn_life'],
                'position': None if objective_event else position(event, role)}

    def available(event):
        if event['round_number'] != frames[0]['round_number'] or not ticks[0] <= event['tick'] <= ticks[-1]:
            return None
        i = bisect.bisect_left(ticks, event['tick'])
        if clocks[i]['eligibility']['kind'] in ('knife', 'warmup'):
            omissions['excluded_round'] += 1
            return None
        if i and (data[i]['continuity'] != 'adjacent_samples' or frames[i]['sequence'] != frames[i-1]['sequence'] + 1):
            omissions['sample_gap'] += 1
            return None
        return i

    def common(event, index, prefix):
        j = bisect.bisect_right(ticks, event['tick']) - 1
        clock = clocks[j]['round_elapsed_seconds']
        pause = clocks[j].get('native_clock', {}).get('pause_state', {}).get('value')
        seconds = None
        if clock['status'] == 'observed' and (ticks[j] == event['tick'] or pause == 'running'):
            seconds = clock['value'] + (event['tick'] - ticks[j]) / frames[j]['tick_rate']
        return {'id': prefix + digest(event['event_id'].encode())[:20], 'tick': event['tick'],
                'round': event['round_number'], 'available_index': index,
                'available_sequence': frames[index]['sequence'], 'elapsed_seconds': seconds}

    seen = set()
    for event in sorted(combat, key=lambda e: (e['tick'], e['sequence_in_tick'], e['event_id'])):
        require(event['event_id'] not in seen, 'duplicate combat event')
        seen.add(event['event_id'])
        if event['event_type'] != 'player_hurt':
            continue
        i = available(event)
        if i is None:
            continue
        if event.get('source') != 'demoinfocs.player_hurt' or event.get('tick_status') != 'observed' or event.get('damage_status') != 'observed':
            omissions['damage_source_unavailable'] += 1
            continue
        attacker, victim = participant(event, 'actor', i), participant(event, 'target', i)
        if attacker is None or victim is None:
            omissions['damage_identity_unavailable'] += 1
            continue
        if event.get('relation') != 'enemy' or {attacker['side'], victim['side']} != {'T', 'CT'}:
            continue
        health, armor = event.get('health_damage_taken'), event.get('armor_damage_taken')
        if not all(type(v) is int and v >= 0 for v in (health, armor)) or health + armor == 0:
            omissions['positive_damage_unavailable'] += 1
            continue
        weapon = event.get('weapon') if event.get('weapon_status') == 'observed' and event.get('weapon_source') == 'callback_weapon' else None
        damage.append({**common(event, i, 'damage-'), 'attacker': attacker, 'victim': victim,
                       'weapon': weapon, 'weapon_is_utility': event.get('weapon_is_utility'),
                       'health_damage': health, 'armor_damage': armor, 'source': 'player_hurt_callback'})

    # Same-tick death is stated only when it is backed by a kill callback.
    deaths = {(e['round_number'], e['tick'], e.get('target_player_id')) for e in combat
              if e['event_type'] == 'kill' and e.get('source') == 'demoinfocs.kill' and e.get('tick_status') == 'observed'}
    seen = set()
    for event in sorted(objective, key=lambda e: (e['tick'], e['sequence_in_tick'], e['event_id'])):
        require(event['event_id'] not in seen, 'duplicate objective event')
        seen.add(event['event_id'])
        if event['event_type'] not in ('bomb_drop', 'bomb_pickup'):
            continue
        i = available(event)
        if i is None:
            continue
        if event.get('source') != 'demoinfocs_event':
            omissions['objective_source_unavailable'] += 1
            continue
        actor = participant(event, 'actor', i, objective_event=True)
        if actor is None or actor['side'] != own_side:
            omissions['objective_actor_unavailable'] += 1
            continue
        c4.append({**common(event, i, 'c4-'), 'kind': event['event_type'], 'actor': actor,
                   'death_same_tick': (event['round_number'], event['tick'], event.get('actor_player_id')) in deaths,
                   'source': 'objective_callback'})
    result = {'schema_id': 'stratai.review_extra_context@1', 'case_id': case_id,
              'source_alias': support['source_alias'], 'source_sha256': support['source_sha256'],
              'frame_count': len(frames), 'side': own_side, 'damage': damage, 'c4': c4,
              'source_refs': refs, 'omissions': dict(omissions), 'perception_proven': False}
    Draft202012Validator(json.loads(SCHEMA.read_bytes())).validate(result)
    return result
