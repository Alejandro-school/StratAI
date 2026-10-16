from copy import deepcopy
import pytest
from ai_coach.interpretation.place_volumes import load_reference, resolve
from ai_coach.interpretation.replay_reading_context import ReplayReadingSession


def packet():
    position={'x':-1500,'y':750,'z':-40}
    place={**resolve(position,load_reference(),map_name='de_mirage',xy_precision=.5),
           'observation_tick':10,'availability_tick':10,'xy_precision':.5}
    own={'steam_id':'1','team':'T','alive':True,**position,'has_c4':False,
         'knowledge':'own_team_snapshot','observation_tick':10,'availability_tick':10,'place':place}
    return {'schema_id':'stratai.replay_context_input@3','side':'T','map_name':'de_mirage',
            'place_reference_id':load_reference()['reference_id'],'tick_rate':64,'start_tick':10,'cut_tick':10,
            'views':[{'schema_id':'stratai.team_replay_projection@5','side':'T','tick':10,'availability_tick':10,
                      'place_reference_id':load_reference()['reference_id'],
                      'vision_status':'unavailable','uncertainty':{},'frame':{'players':[own],'bomb':None},'events':[],'deaths':[]}]}


def test_places_are_dated_estimates_with_reference_not_native_observed_truth():
    context=ReplayReadingSession(packet()).consultation().initial()
    place=next(f for f in context['situation'] if f['key']=='place_context')
    assert place['value']['zone']=='Apartments' and place['status']=='estimated'
    assert place['value']['build_compatibility']=='unverified' and place['subject']=='P01'
    assert not any('named map zones are unknown' in s for s in context['reading_constraints'])


@pytest.mark.parametrize('field,value',[('zone','BombsiteA'),('observation_tick',9),('availability_tick',11),
                                      ('reference_id','wrong'),('xy_precision',0),('build_compatibility','verified')])
def test_fabricated_mismatched_or_future_place_is_rejected(field,value):
    data=packet();data['views'][0]['frame']['players'][0]['place'][field]=value
    with pytest.raises(ValueError):ReplayReadingSession(data)


def test_place_missing_on_living_player_or_added_to_dead_player_is_rejected():
    data=packet();del data['views'][0]['frame']['players'][0]['place']
    with pytest.raises(ValueError):ReplayReadingSession(data)
    data=packet();data['views'][0]['frame']['players'][0]['alive']=False
    with pytest.raises(ValueError):ReplayReadingSession(data)


def test_missing_height_never_manufactures_a_zone():
    data=packet();player=data['views'][0]['frame']['players'][0];del player['z']
    with pytest.raises(ValueError):ReplayReadingSession(data)
