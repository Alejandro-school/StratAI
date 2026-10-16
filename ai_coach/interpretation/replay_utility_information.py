"""Validate estimated utility signals and independently recompute launcher groups."""
from copy import deepcopy

from ai_coach.support.anchors import require
from ai_coach.interpretation.replay_factual_validation import finite
from ai_coach.interpretation.replay_information import natural

SCHEMA = 'stratai.replay_utility_information@1'
TYPES = ('he', 'smoke', 'molotov', 'incendiary', 'flashbang', 'decoy', 'inferno')
SOURCES = {'launch': ('demoinfocs_event:GrenadeProjectileThrow',),
           'projectile': ('demoinfocs_state:GrenadeProjectiles',),
           'effect': ('demoinfocs_event:SmokeStart', 'demoinfocs_event:InfernoStart',
                      'demoinfocs_event:HeExplode', 'demoinfocs_event:FlashExplode')}


def launcher_estimates(rows, rate):
    unique = {}
    for row in rows:
        if row['kind'] != 'launch' or row['affiliation'] != 'unknown' or row['utility_type'] not in TYPES[:4]:
            continue
        key = row['object_id']
        require(key not in unique or (unique[key]['observation_tick'], unique[key]['utility_type']) ==
                (row['observation_tick'], row['utility_type']), 'conflicting observable launch')
        unique.setdefault(key, row)
    result = []
    for kind in TYPES[:4]:
        launches = sorted((r for r in unique.values() if r['utility_type'] == kind),
                          key=lambda r: (r['observation_tick'], natural(r['object_id'])))
        for index in range(2, len(launches)):
            group = launches[index-2:index+1]
            first, last = group[0]['observation_tick'], group[-1]['observation_tick']
            if last - first > 5 * rate:
                continue
            objects = [r['object_id'] for r in group]
            evidence = [r for r in rows if r['kind'] == 'launch' and r['object_id'] in objects]
            recipients = [{p['player_id'] for r in evidence if r['object_id'] == obj for p in r['recipients']} for obj in objects]
            result.append(dict(id='launchers:' + ':'.join(objects), utility_type=kind, minimum_launchers=3,
                status='estimated_expert_deduction', rule='three_same_type_observable_launches_within_five_seconds',
                first_launch_tick=first, last_launch_tick=last, maximum_interval_seconds=5,
                object_ids=objects, observation_ids=sorted((r['id'] for r in evidence), key=natural),
                possible_recipient_ids=sorted(set.intersection(*recipients), key=natural),
                side_status='unknown', origin_status='unknown', current_location_status='unknown',
                exceptions=['pickups_or_extra_supply_not_excluded', 'type_is_visual_estimate', 'overlapping_groups_do_not_add_launchers']))
    return result


def latest_groups(rows):
    groups = {}
    for row in rows:
        old = groups.get(row['object_id'], [])
        if not old or row['observation_tick'] > old[0]['observation_tick']:
            groups[row['object_id']] = [row]
        elif row['observation_tick'] == old[0]['observation_tick']:
            old.append(row)
    return [groups[key] for key in sorted(groups, key=natural)]


def checked_utility_information(view, rate, previous):
    value = view.get('utility_information')
    require(isinstance(value, dict) and set(value) == {'schema_id','side','round_start_tick','communication_model',
        'reception_status','source_status','observations','objects','segments','by_player','launcher_estimates'}, 'utility information fields differ')
    require(value['schema_id'] == SCHEMA and value['side'] == view['side']
        and value['round_start_tick'] == view['information']['round_start_tick']
        and value['communication_model'] == 'immediate_simulated_possible_reception'
        and value['reception_status'] == 'possible_not_confirmed'
        and value['source_status'] in ('estimated_source_supplied', 'unavailable'), 'utility information policy differs')
    known = {p['player_id']: p for p in view['information']['known_team_players']}
    own = {str(p['steam_id']): p for p in view['frame']['players'] if p['team'] == view['side']}
    rows, refs = value['observations'], {}
    require(isinstance(rows, list) and (value['source_status'] != 'unavailable' or not rows), 'unavailable utility source contains observations')
    require(rows == sorted(rows, key=lambda r: (r['observation_tick'], natural(r['id']))), 'utility observations unordered')
    for row in rows:
        require(isinstance(row, dict) and set(row) == {'id','observer_id','object_id','kind','utility_type','type_status',
            'position','observation_tick','availability_tick','status','affiliation','source','recipient_basis','recipients'}, 'utility observation fields differ')
        when, who = row['observation_tick'], row['observer_id']
        require(type(when) is int and value['round_start_tick'] <= when <= view['tick'] and row['availability_tick'] == when
            and who in known and known[who]['observation_tick'] <= when, 'future or foreign utility observer')
        require(not any(str(d['victim_id']) == who and d['tick'] <= when for d in view['deaths']), 'utility observation after death')
        require(isinstance(row['id'], str) and row['id'] and row['id'] not in refs
            and isinstance(row['object_id'], str) and row['object_id'] and row['kind'] in SOURCES
            and row['utility_type'] in TYPES and row['status'] == 'estimated'
            and row['type_status'] == 'estimated_native_type_proxy', 'invalid utility identity or certainty')
        source = row['source']
        require(isinstance(source, dict) and set(source) == {'kind','reference','native_source','eye_source','position_basis'}
            and source['kind'] == 'go_calculated_utility' and source['reference'] == row['id']
            and source['native_source'] in SOURCES[row['kind']]
            and source['eye_source'] in ('native_eye_position', 'stance_eye_proxy')
            and source['position_basis'] in ('native_projectile_position','native_explosion_point','effect_representative_point_not_full_geometry'), 'utility source differs')
        point = row['position']
        if row['kind'] == 'effect':
            expected_type = {'demoinfocs_event:SmokeStart':'smoke','demoinfocs_event:InfernoStart':'inferno',
                'demoinfocs_event:HeExplode':'he','demoinfocs_event:FlashExplode':'flashbang'}[source['native_source']]
            require(row['utility_type'] == expected_type and source['position_basis'] ==
                ('native_explosion_point' if expected_type in ('he','flashbang') else 'effect_representative_point_not_full_geometry'), 'utility effect type or point basis differs')
        else:
            require(row['utility_type'] != 'inferno' and source['position_basis'] == 'native_projectile_position', 'utility projectile type or point basis differs')
        require(isinstance(point, dict) and set(point) == {'x','y','z'} and all(finite(n) for n in point.values()), 'invalid utility point')
        linked = any(e['type'] == 'grenade_throw' and e.get('actor_team') == view['side']
            and e['tick'] <= when and row['object_id'] == f"projectile:{e.get('projectile_id')}" for e in view['events'])
        require(row['affiliation'] == ('own_team' if linked else 'unknown'), 'hidden utility affiliation')
        basis = row['recipient_basis']
        require(isinstance(basis, dict) and set(basis) == {'frame_tick','present_player_ids'}
            and type(basis['frame_tick']) is int and value['round_start_tick'] <= basis['frame_tick'] <= when, 'utility recipient basis differs')
        ids = basis['present_player_ids']
        require(isinstance(ids, list) and ids == sorted(set(ids), key=natural) and who in ids
            and all(i in known and known[i]['observation_tick'] <= basis['frame_tick'] for i in ids), 'invalid utility recipients')
        if basis['frame_tick'] == view['availability_tick']:
            require(ids == sorted(own, key=natural), 'utility recipient snapshot differs')
            require(own[who]['alive'] or own[who].get('life_context', {}).get('observation_tick', -1) > when, 'dead utility observer snapshot')
        expected = [dict(player_id=i, kind='individual_observation' if i == who else 'simulated_team_communication',
                         reception='origin' if i == who else 'possible', availability_tick=when) for i in ids]
        require(row['recipients'] == expected, 'utility reception differs')
        require(row['id'] not in previous or previous[row['id']] == row, 'utility observation changed after availability')
        previous[row['id']] = deepcopy(row)
        refs[row['id']] = row
    require(all(r['id'] in refs for r in previous.values() if r['kind'] == 'launch'), 'observable launch disappeared')
    objects = []
    for group in latest_groups(rows):
        first = group[0]
        points = []
        for row in group:
            point = next((p for p in points if p['position'] == row['position']), None)
            if point is None:
                point = {'position':deepcopy(row['position']), 'observation_ids':[]}
                points.append(point)
            point['observation_ids'].append(row['id'])
        for point in points:
            point['observation_ids'].sort(key=natural)
        objects.append(dict(object_id=first['object_id'], utility_type=first['utility_type'],kind=first['kind'],affiliation=first['affiliation'],
            state='current_estimate' if view['tick'] == first['observation_tick'] else 'last_observed',
            position_status='estimated_point' if len(points)==1 else 'ambiguous_same_tick',position=first['position'] if len(points)==1 else None,points=points,
            observation_tick=first['observation_tick'],age_ticks=view['tick']-first['observation_tick'],
            observation_ids=sorted((r['id'] for r in group), key=natural)))
    require(value['objects'] == objects and view['frame'].get('utility_markers') == objects, 'utility memory or markers differ')
    points, segments, projectile_groups = {}, {}, {}
    for row in rows:
        if row['kind'] != 'projectile':
            continue
        key = row['observer_id'], row['object_id'], row['observation_tick']
        projectile_groups.setdefault(key, []).append(row)
    for group in projectile_groups.values():
        row = group[0]
        key = row['observer_id'], row['object_id']
        if any(r['position'] != row['position'] for r in group):
            points.pop(key, None)
            continue
        before = points.get(key)
        if before and row['observation_tick'] - before['observation_tick'] == 4:
            segments[key] = dict(object_id=row['object_id'], observer_id=row['observer_id'],
                from_point=before['position'], to=row['position'], start_tick=before['observation_tick'], end_tick=row['observation_tick'],
                status='estimated_sampled_segment', observation_ids=[before['id'],row['id']], interpolation_status='between_points_not_observed')
            segments[key]['from'] = segments[key].pop('from_point')
        points[key] = row
    expected_segments = [segments[key] for key in sorted(segments, key=lambda key: (natural(key[0]), natural(key[1])))]
    require(value['segments'] == expected_segments and view['frame'].get('utility_segments') == value['segments'], 'utility sampled segments differ')
    players = []
    for who in sorted(known, key=natural):
        received = [r for r in rows if any(p['player_id'] == who for p in r['recipients'])]
        direct = latest_groups([r for r in received if r['observer_id'] == who])
        shared = latest_groups([r for r in received if r['observer_id'] != who])
        retained = [r for groups in (direct, shared) for group in groups for r in group] + [r for r in received if r['kind'] == 'launch']
        players.append(dict(player_id=who, observation_ids=sorted({r['id'] for r in retained}, key=natural)))
    require(value['by_player'] == players, 'individual utility memory differs')
    require(value['launcher_estimates'] == launcher_estimates(rows, rate), 'launcher deduction differs')
    return deepcopy(value)
