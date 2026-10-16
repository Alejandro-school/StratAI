"""A linked, translated place entity with world-baked tetrahedron geometry."""
from copy import deepcopy
import json
import pytest
from ai_coach.interpretation.vmap_place_extract import extract_vmap


def nodes():
    faces = [[0,2,1],[0,1,3],[0,3,2],[1,2,3]]
    edges = [(a,b) for face in faces for a,b in zip(face, face[1:]+face[:1])]
    identity = {'origin':'0 0 0','angles':'0 0 0','scales':'1 1 1'}
    mesh_data = {'_type':'CDmePolygonMesh', **identity,
        'vertexData':{'_type':'DmeVertexData','streams':[{'_type':'DmeVertexStream','semanticName':'position',
                    'data':['0 0 0','4 0 0','0 4 0','0 0 4']}]},
        'vertexDataIndices':list(range(4)), 'edgeVertexIndices':[a for a,b in edges],
        'edgeNextIndices':[i//3*3+(i+1)%3 for i in range(12)],
        'edgeOppositeIndices':[edges.index((b,a)) for a,b in edges],
        'edgeFaceIndices':[i//3 for i in range(12)],'faceEdgeIndices':[0,3,6,9]}
    return [
        {'_type':'CMapRootElement','world':{'_type':'CMapWorld',**identity,'children':['entity']}},
        {'_type':'CMapEntity','id':'entity','nodeID':'71',**identity,'origin':'999 888 777',
         'entity_properties':{'_type':'EditGameClassProps','classname':'env_cs_place','place_name':'Apartments'},'children':['mesh']},
        {'_type':'CMapMesh','id':'mesh',**identity,'meshData':mesh_data}]


def dmx(node, indent=0):
    rows=['{']
    for key,value in node.items():
        if key == '_type':continue
        prefix='\t'*(indent+1)+json.dumps(key)+' '
        if isinstance(value,dict):
            rows.append(prefix+json.dumps(value['_type'])+'\n'+'\t'*(indent+1)+dmx(value,indent+1))
        elif isinstance(value,list):
            elements=key in ('children','streams')
            kind='element_array' if elements else 'string_array'
            values=[]
            for item in value:
                if isinstance(item,dict):values.append(json.dumps(item['_type'])+' '+dmx(item,indent+2))
                else:values.append(('"element" ' if elements else '')+json.dumps(str(item)))
            rows.append(prefix+json.dumps(kind)+' [ '+', '.join(values)+' ]')
        else:
            rows.append(prefix+('"elementid" ' if key=='id' else '"string" ')+json.dumps(str(value)))
    return '\n'.join(rows)+'\n'+'\t'*indent+'}'


def extract(tmp_path, data):
    path=tmp_path/'map.vmap'
    path.write_text('<!-- dmx encoding keyvalues2 4 format vmap 40 -->\n'+
                    '\n'.join(json.dumps(n['_type'])+'\n'+dmx(n) for n in data),encoding='utf-8')
    return extract_vmap(path,map_name='de_test')


def test_linked_half_edges_preserve_world_coordinates_without_second_entity_translation(tmp_path):
    result=extract(tmp_path,nodes())
    region=result['regions'][0]
    assert region['label']=='Apartments' and region['entity_origin']==[999,888,777]
    assert region['hulls'][0]['bounds']=={'min':[0,0,0],'max':[4,4,4]}
    assert region['hulls'][0]['faces']==[[0,2,1],[0,1,3],[0,3,2],[1,2,3]]


@pytest.mark.parametrize('variant',['world_transform','entity_rotation','mesh_translation','missing_link','indirect_parent','broken_cycle','broken_opposite'])
def test_unsupported_transforms_hierarchy_or_topology_fail_closed(tmp_path,variant):
    data=deepcopy(nodes())
    if variant=='world_transform':data[0]['world']['scales']='2 1 1'
    elif variant=='entity_rotation':data[1]['angles']='0 90 0'
    elif variant=='mesh_translation':data[2]['origin']='1 2 3'
    elif variant=='missing_link':data[1]['children']=['absent']
    elif variant=='indirect_parent':data[0]['world']['children']=[]
    elif variant=='broken_cycle':data[2]['meshData']['edgeNextIndices'][0]=1;data[2]['meshData']['edgeNextIndices'][2]=1
    elif variant=='broken_opposite':data[2]['meshData']['edgeOppositeIndices'][0]=0
    with pytest.raises(ValueError):extract(tmp_path,data)
