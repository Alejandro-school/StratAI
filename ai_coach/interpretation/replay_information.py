"""Check causal contact memory and explicitly possible simulated recipients."""
from copy import deepcopy
import re

from ai_coach.support.anchors import require
from ai_coach.interpretation.replay_factual_validation import finite

SCHEMA = 'stratai.replay_information@1'


def natural(value):
    return [int(x) if x.isdigit() else x for x in re.split(r'(\d+)', value)]


def latest_groups(rows):
    result = {}
    for row in rows:
        key = row['subject_id']
        existing = result.get(key, [])
        if not existing or row['observation_tick'] > existing[0]['observation_tick']:
            result[key] = [row]
        elif row['observation_tick'] == existing[0]['observation_tick']:
            existing.append(row)
    return result


def checked_information(view, start, previous):
    value = view.get('information')
    require(isinstance(value, dict) and set(value) == {'schema_id','side','round_start_tick',
        'communication_model','reception_status','own_team_state_scope','enemy_identity_scope','known_team_players','observations','contacts','by_player'},
        'information fields differ')
    require(value['schema_id'] == SCHEMA and value['side'] == view['side']
        and value['communication_model'] == 'immediate_simulated_possible_reception'
        and value['reception_status'] == 'possible_not_confirmed'
        and value['own_team_state_scope'] == 'collective_demo_overview_not_individual_knowledge'
        and value['enemy_identity_scope'] == 'opaque_demo_tracking_not_recognized_person', 'information policy differs')
    origin = value['round_start_tick']
    require(type(origin) is int and 0 <= origin <= start, 'information round instance differs')
    own = {str(p['steam_id']): p for p in view['frame']['players'] if p['team'] == view['side']}
    known = {}
    require(isinstance(value['known_team_players'], list), 'known team source absent')
    for player in value['known_team_players']:
        require(isinstance(player, dict) and set(player) == {'player_id','team','observation_tick','source'}, 'known team fields differ')
        who, when = player['player_id'], player['observation_tick']
        require(isinstance(who, str) and who.isdigit() and 1 <= int(who) <= 10 and who not in known
            and player['team'] == view['side'] and type(when) is int and origin <= when <= view['availability_tick']
            and player['source'] == {'kind':'demo_replay_frame','reference':f"round:{view['round']}:frame:{when}:player:{who}"}, 'known team provenance differs')
        require(not any(str(p['steam_id']) == who and p['team'] != view['side'] for p in view['frame']['players']), 'rival as known teammate')
        known[who] = player
    require(set(own) <= set(known), 'current own roster missing')
    deaths = view['deaths']
    dead = {str(d['victim_id']) for d in deaths if d['tick'] <= view['tick']}
    rows = value['observations']
    require(isinstance(rows, list) and isinstance(value['contacts'], list) and isinstance(value['by_player'], list), 'invalid information arrays')
    refs = {}
    for row in rows:
        require(isinstance(row, dict) and set(row) == {'id','observer_id','subject_id','subject_side','position',
            'observation_tick','availability_tick','status','source','recipient_basis','recipients'}, 'observation fields differ')
        observer, subject = row['observer_id'], row['subject_id']
        observed, available = row['observation_tick'], row['availability_tick']
        require(type(observed) is int and type(available) is int and origin <= observed <= available <= view['tick'], 'future information')
        require(observer in known and subject not in known and isinstance(subject, str) and subject.isdigit()
            and 1 <= int(subject) <= 10 and row['subject_side'] == ('CT' if view['side'] == 'T' else 'T'), 'hidden information participant')
        require(row['id'] == f"round:{view['round']}:start:{origin}:contact:{observed}:{observer}:{subject}"
            and row['id'] not in refs, 'information reference differs')
        require(not any(str(d['victim_id']) in (observer, subject) and d['tick'] <= observed for d in deaths), 'observation after death')
        position = row['position']
        require(isinstance(position, dict) and set(position) in ({'x','y'}, {'x','y','z'})
            and all(finite(n) for n in position.values()), 'invalid information position')
        require(row['status'] in ('estimated', 'observed'), 'information certainty differs')
        source = row['source']
        require(isinstance(source, dict) and set(source) == {'kind','reference'}
            and all(isinstance(x, str) and bool(x) for x in source.values()), 'information source absent')
        if row['status'] == 'estimated':
            require(source == {'kind': 'go_calculated_contact',
                'reference': f"round:{view['round']}:contact:{observed}:{observer}:{subject}"}
                and observed == available, 'calculated source differs')
        basis = row['recipient_basis']
        require(isinstance(basis, dict) and set(basis) == {'frame_tick','source','present_player_ids'}
            and type(basis['frame_tick']) is int and origin <= basis['frame_tick'] <= available
            and basis['source'] == {'kind':'demo_replay_frame',
                'reference': f"round:{view['round']}:frame:{basis['frame_tick']}:recipients:{view['side']}"}, 'recipient basis differs')
        ids = basis['present_player_ids']
        require(isinstance(ids, list) and all(isinstance(i, str) and i in known and known[i]['observation_tick'] <= basis['frame_tick'] for i in ids)
            and ids == sorted(set(ids), key=natural) and observer in ids, 'invalid recipient eligibility')
        if basis['frame_tick'] == view['availability_tick']:
            expected = sorted(own, key=natural)
            require(ids == expected, 'recipient snapshot differs')
            if observed >= basis['frame_tick']:
                require(own[observer]['alive'] or own[observer].get('life_context', {}).get('observation_tick', -1) > observed, 'dead observer snapshot')
        recipients = [{'player_id': i, 'kind': 'individual_observation' if i == observer else 'simulated_team_communication',
                       'reception': 'origin' if i == observer else 'possible', 'availability_tick': available} for i in ids]
        require(row['recipients'] == recipients, 'possible recipients differ')
        require(row['id'] not in previous or previous[row['id']] == row, 'information changed after availability')
        previous[row['id']] = deepcopy(row)
        refs[row['id']] = row
    expected_players = []
    for who in sorted(known, key=natural):
        received = [o for o in rows if any(r['player_id'] == who for r in o['recipients'])]
        direct = latest_groups([o for o in received if o['observer_id'] == who])
        shared = latest_groups([o for o in received if o['observer_id'] != who])
        expected_players.append({'player_id':who, 'observation_ids':sorted(
            (o['id'] for groups in (direct, shared) for group in groups.values() for o in group), key=natural)})
    require(value['by_player'] == expected_players, 'individual information differs')
    recent = {}
    for row in rows:
        key = row['observer_id'], row['subject_id']
        if key not in recent or row['observation_tick'] >= recent[key]['observation_tick']:
            recent[key] = row
    living = {str(p['steam_id']): p for p in view['frame']['players'] if p['team'] != view['side']}
    expected_contacts = []
    for subject, group in sorted(latest_groups(recent.values()).items(), key=lambda pair: natural(pair[0])):
        if subject in dead:
            continue
        first = group[0]
        require(all(o['position'] == first['position'] for o in group), 'simultaneous positions differ')
        current = living.get(subject)
        if current:
            require(current['observation_tick'] == first['observation_tick'] and all(current.get(k) == n for k, n in first['position'].items()), 'current contact differs')
        expected_contacts.append({'subject_id': subject, 'subject_side': first['subject_side'],
            'state': 'current_contact' if current else 'last_observed', 'position': first['position'],
            'observation_tick': first['observation_tick'], 'availability_tick': max(o['availability_tick'] for o in group),
            'age_ticks': view['tick'] - first['observation_tick'], 'observation_ids': sorted((o['id'] for o in group), key=natural)})
    require(value['contacts'] == expected_contacts, 'contact memory differs')
    require(set(living) == {c['subject_id'] for c in expected_contacts if c['state'] == 'current_contact'}, 'current contact provenance absent')
    markers = view['frame'].get('last_seen_markers')
    require(isinstance(markers, list) and len(markers) == sum(c['state'] == 'last_observed' for c in expected_contacts), 'historical markers differ')
    for marker, contact in zip(markers, [c for c in expected_contacts if c['state'] == 'last_observed']):
        require(set(marker) == set(contact) | {'name','team','knowledge','current_position_status'}
            and all(marker[k] == v for k, v in contact.items()) and marker['team'] == contact['subject_side']
            and marker['knowledge'] == 'historical_contact' and marker['current_position_status'] == 'unknown', 'historical marker as live or moved')
    return deepcopy(value)
