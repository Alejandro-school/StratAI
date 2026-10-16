"""Geometry regressions independent of any map's nearest-label heuristic."""
from copy import deepcopy
import pytest
from ai_coach.interpretation import place_volumes as places


def tetra():
    return places.build_hull([[0,0,0],[4,0,0],[0,4,0],[0,0,4]], [[0,2,1],[0,1,3],[0,3,2],[1,2,3]])


def package():
    return {'schema_id':places.SCHEMA_ID,'map_name':'de_test','reference_id':'fixture',
            'build_compatibility':'unverified','regions':[{'id':'r1','label':'room','hulls':[tetra()]}]}


def test_actual_volume_rejects_point_inside_bounding_box_but_outside_sloped_face():
    assert places.resolve({'x':.5,'y':.5,'z':.5},package(),map_name='de_test')['zone']=='room'
    assert places.resolve({'x':3,'y':3,'z':3},package(),map_name='de_test')['zone'] is None


def test_height_is_not_collapsed_to_xy():
    assert places.resolve({'x':.5,'y':.5,'z':4},package(),map_name='de_test')['status']=='unavailable'


def test_boundary_and_serialized_xy_uncertainty_are_not_unique_containment():
    assert places.resolve({'x':0,'y':1,'z':1},package(),map_name='de_test')['status']=='boundary'
    result=places.resolve({'x':.25,'y':1,'z':1},package(),map_name='de_test',xy_precision=.5)
    assert result['status']=='boundary' and result['zone'] is None


def test_overlap_retains_candidates_and_same_label_brushes_are_one_zone():
    data=package();other=deepcopy(data['regions'][0]);other.update(id='r2',label='other');data['regions'].append(other)
    value=places.resolve({'x':1,'y':1,'z':1},data,map_name='de_test')
    assert value['status']=='ambiguous' and value['zone'] is None and len(value['candidates'])==2
    other['label']='room'
    assert places.resolve({'x':1,'y':1,'z':1},data,map_name='de_test')['zone']=='room'


@pytest.mark.parametrize('faces',[[[0,2,1],[0,1,3],[0,3,2]],[[0,0,1],[0,1,3],[0,3,2],[1,2,3]]])
def test_open_or_degenerate_hulls_are_rejected(faces):
    with pytest.raises(ValueError):places.build_hull([[0,0,0],[4,0,0],[0,4,0],[0,0,4]],faces)


def test_unknown_and_nonfinite_do_not_fall_back_to_a_nearby_label():
    for position in ({'x':1,'y':1},{'x':float('nan'),'y':1,'z':1}):
        assert places.resolve(position,package(),map_name='de_test')['status']=='unavailable'
    assert places.resolve({'x':1,'y':1,'z':1},package(),map_name='de_other')['status']=='unavailable'
