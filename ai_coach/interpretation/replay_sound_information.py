"""Validate the dated geometric sound estimate, independently of JS memory.

Native points remain in the producer audit. This boundary checks the catalog,
range and team contract; it cannot authenticate those hidden points itself.
"""
from copy import deepcopy
import hashlib
import json
from pathlib import Path
import re

from ai_coach.support.anchors import require
from ai_coach.interpretation.replay_factual_validation import finite
from ai_coach.interpretation.replay_information import natural
from ai_coach.interpretation.place_volumes import load_reference

_RAW = (Path(__file__).resolve().parents[2] / 'backend/go-service/pkg/sound/cs2_profiles.json').read_bytes()
CATALOG = json.loads(_RAW)
CATALOG_SHA256 = hashlib.sha256(_RAW).hexdigest()
PROFILES = {p['hash']: p for p in CATALOG['events']}
SOURCES = ('net.CMsgSosStartSoundEvent', 'net.CCSUsrMsg_WeaponSound', 'net.CMsgTEFireBullets')
PERCEPTION = {k:'unknown' for k in ('listening','identity','enemy_count','exact_position','current_enemy_position','masking')}


def cell(row):
    return row['kind'], row['source_side'], tuple(c['id'] for c in row['region']['candidates'])


def latest(rows):
    result = {}
    for row in rows:
        result[cell(row)] = row
    return sorted((r['id'] for r in result.values()), key=natural)


def checked_sound_information(view, previous):
    value = view.get('sound_information')
    require(isinstance(value, dict) and set(value) == {'schema_id','side','round_start_tick','source_status',
        'catalog_id','catalog_sha256','communication_model','reception_status','geometry_model','build_compatibility',
        'perception','observations','by_player'}, 'sound information fields differ')
    require(value['schema_id']=='stratai.replay_sound_information@1' and value['side']==view['side']
        and value['round_start_tick']==view['information']['round_start_tick']
        and value['catalog_id']==CATALOG['catalog_id'] and value['catalog_sha256']==CATALOG_SHA256
        and value['source_status'] in ('estimated_source_supplied','unavailable')
        and value['communication_model']=='immediate_simulated_possible_reception'
        and value['reception_status']=='possible_not_confirmed'
        and value['geometry_model']=='native_points_euclidean_3d_no_visual_occlusion'
        and value['build_compatibility']=='unverified' and value['perception']==PERCEPTION, 'sound policy differs')
    known = {p['player_id']:p for p in view['information']['known_team_players']}
    own = {str(p['steam_id']):p for p in view['frame']['players'] if p['team']==view['side']}
    rows, refs = value['observations'], {}
    require(isinstance(rows,list) and (value['source_status']!='unavailable' or not rows), 'unavailable sound source has signals')
    require(rows==sorted(rows,key=lambda r:(r['observation_tick'],natural(r['id']))), 'sound observations unordered')
    regions = {r['id']:r['label'] for r in load_reference()['regions']}
    for row in rows:
        require(isinstance(row,dict) and set(row)=={'id','observer_id','kind','profile_hash','source_side','side_status',
            'region','observation_tick','availability_tick','status','distance_3d_units','radius_units','radius_basis',
            'source','recipient_basis','recipients'}, 'sound observation fields differ')
        when, who = row['observation_tick'], row['observer_id']
        require(type(when) is int and value['round_start_tick']<=when<=view['tick'] and row['availability_tick']==when
            and who in known and known[who]['observation_tick']<=when, 'future or foreign sound observer')
        require(not any(str(d['victim_id'])==who and d['tick']<=when for d in view['deaths']), 'sound reception after death')
        require(isinstance(row['id'],str) and re.fullmatch(r'sound-observation:\d+',row['id']) and row['id'] not in refs
            and row['status']=='estimated_possible_reception' and row['source_side'] in ('T','CT','unknown')
            and row['source_side']!=view['side'] and row['side_status']=='native_origin_estimate_not_recognition', 'sound identity or certainty differs')
        require(type(row['profile_hash']) is int, 'invalid sound profile hash')
        p = PROFILES.get(row['profile_hash'])
        require(p and p['usable'] and not p['local_only'] and not p['silent'] and p['broadcast'] and row['kind']==p['family']
            and row['radius_units']==p['radius_units'] and row['radius_basis']==p['radius_basis']
            and finite(row['distance_3d_units']) and 0<=row['distance_3d_units']<p['radius_units'], 'sound profile or range differs')
        source = row['source']
        require(isinstance(source,dict) and set(source)=={'kind','reference','emission_reference','native_source','catalog_id','catalog_sha256'}
            and source['kind']=='go_calculated_sound' and source['reference']==row['id']
            and isinstance(source['emission_reference'],str) and re.fullmatch(r'sound-emission:\d+',source['emission_reference'])
            and source['native_source'] in SOURCES and source['catalog_id']==CATALOG['catalog_id']
            and source['catalog_sha256']==CATALOG_SHA256, 'sound source differs')
        region = row['region']
        require(isinstance(region,dict) and set(region)=={'status','reference_id','candidates','build_compatibility'}
            and region['status']=='compatible_region_estimate' and region['reference_id']==view['place_reference_id']
            and region['build_compatibility']=='unverified' and isinstance(region['candidates'],list), 'sound region differs')
        seen = set()
        for candidate in region['candidates']:
            require(isinstance(candidate,dict) and set(candidate)=={'id','label'} and candidate['id'] not in seen
                and regions.get(candidate['id'])==candidate['label'], 'invalid compatible sound region')
            seen.add(candidate['id'])
        basis = row['recipient_basis']
        require(isinstance(basis,dict) and set(basis)=={'frame_tick','present_player_ids'}
            and type(basis['frame_tick']) is int and value['round_start_tick']<=basis['frame_tick']<=when, 'sound recipient basis differs')
        ids = basis['present_player_ids']
        require(isinstance(ids,list) and ids==sorted(set(ids),key=natural) and who in ids
            and all(i in known and known[i]['observation_tick']<=basis['frame_tick'] for i in ids), 'invalid sound recipients')
        if basis['frame_tick']==view['availability_tick']:
            require(ids==sorted(own,key=natural), 'sound recipient snapshot differs')
            require(own[who]['alive'] or own[who].get('life_context',{}).get('observation_tick',-1)>when, 'dead sound observer snapshot')
        expected = [dict(player_id=i,kind='individual_estimate' if i==who else 'simulated_team_communication',
            reception='possible',availability_tick=when) for i in ids]
        require(row['recipients']==expected, 'sound reception differs')
        refs[row['id']] = row
    players = []
    for who in sorted(known,key=natural):
        received = [r for r in rows if any(p['player_id']==who for p in r['recipients'])]
        players.append(dict(player_id=who,individual_signal_ids=latest([r for r in received if r['observer_id']==who]),
            possible_shared_signal_ids=latest([r for r in received if r['observer_id']!=who])))
    require(value['by_player']==players, 'individual sound memory differs')
    if previous:
        before = {r['id']:r for r in previous['observations']}
        for player in previous['by_player']:
            current = next((p for p in players if p['player_id']==player['player_id']),None)
            require(current is not None, 'sound player memory lost')
            for mode in ('individual_signal_ids','possible_shared_signal_ids'):
                for ref in player[mode]:
                    old = before[ref]
                    replacement = [refs[r] for r in current[mode] if cell(refs[r])==cell(old)]
                    require(replacement and replacement[0]['observation_tick']>=old['observation_tick'], 'sound memory lost or reverted')
        for ref in before.keys() & refs.keys():
            require(before[ref]==refs[ref], 'sound signal changed after availability')
    previous.clear()
    previous.update(deepcopy(value))
    return deepcopy(value)
