"""Extract linked env_cs_place half-edge hulls from the retained DMX text.

VRF MapExtract bakes worldTransform into AddPhysHull vertices. Entity origins
are metadata, not an additional parent translation. Only the audited, identity
mesh-transform representation is accepted; unsupported transforms fail closed.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re

from ai_coach.interpretation.place_volumes import SCHEMA_ID, build_hull, digest, encoded, validate, _require

TOKEN = re.compile(r'"(?:[^"\\]|\\.)*"|[{}\[\],]')


def _object(text):
    tokens = [json.loads(t) if t.startswith('"') else t for t in TOKEN.findall(text)]
    index = 0
    def take():
        nonlocal index
        _require(index < len(tokens), "truncated DMX")
        token=tokens[index];index+=1;return token
    def value(kind):
        nonlocal index
        token=take()
        if token == '{':
            result={"_type":kind}
            while tokens[index] != '}':
                key, field_type=take(),take()
                _require(key not in result, "duplicate DMX field")
                result[key]=value(field_type)
            take();return result
        if token == '[':
            result=[]
            while tokens[index] != ']':
                if tokens[index] == ',':take();continue
                result.append(value(take()) if kind == 'element_array' else take())
            take();return result
        return token
    result=value(take())
    _require(index == len(tokens), "trailing DMX tokens")
    return result


def _vector(text):
    values=[float(x) for x in text.split()]
    _require(len(values)==3, "invalid transform")
    return values


def _identity(node, *, translated=False):
    _require(_vector(node['angles']) == [0,0,0] and _vector(node['scales']) == [1,1,1], "unsupported place transform")
    if not translated:
        _require(_vector(node['origin']) == [0,0,0], "nonidentity baked mesh transform")


def _mesh_hull(node):
    _identity(node);mesh=node['meshData'];_identity(mesh)
    streams=[s for s in mesh['vertexData']['streams'] if s.get('semanticName') == 'position']
    _require(len(streams)==1, "missing or ambiguous vertex positions")
    data=[_vector(v) for v in streams[0]['data']]
    vertex_data=[int(v) for v in mesh['vertexDataIndices']]
    _require(all(0<=v<len(data) for v in vertex_data), "invalid vertex data index")
    vertices=[data[v] for v in vertex_data]
    edge_vertex=[int(v) for v in mesh['edgeVertexIndices']]
    edge_next=[int(v) for v in mesh['edgeNextIndices']]
    edge_opposite=[int(v) for v in mesh['edgeOppositeIndices']]
    edge_face=[int(v) for v in mesh['edgeFaceIndices']]
    starts=[int(v) for v in mesh['faceEdgeIndices']]
    _require(len(edge_vertex)==len(edge_next)==len(edge_opposite)==len(edge_face), "incomplete half-edge topology")
    _require(all(0<=e<len(edge_next) for e in edge_next+edge_opposite+starts), "half-edge index out of range")
    _require(all(edge_opposite[edge_opposite[i]]==i and edge_opposite[i]!=i for i in range(len(edge_vertex))), "invalid opposite edges")
    faces=[];visited=set()
    for face_index,start in enumerate(starts):
        edge=start;face=[];local=set()
        while edge not in local:
            _require(edge not in visited and edge_face[edge]==face_index, "crossed half-edge face")
            local.add(edge);visited.add(edge);face.append(edge_vertex[edge]);edge=edge_next[edge]
        _require(edge == start, "open half-edge cycle")
        faces.append(face)
    _require(len(visited)==len(edge_vertex), "unvisited half-edge")
    return build_hull(vertices,faces)


def extract_vmap(path, *, map_name):
    path=Path(path);raw=path.read_bytes();text=raw.decode('utf-8').replace('\r\n','\n')
    _require(text.startswith('<!-- dmx encoding keyvalues2 4 format vmap 40 -->'), "unsupported VMAP encoding")
    root = re.search(r'^"CMapRootElement"\n\{\n.*?^\}', text, re.M | re.S)
    _require(root is not None, "missing VMAP root")
    world = _object(root.group())['world']
    _require(world['_type'] == 'CMapWorld', "unsupported VMAP world")
    _identity(world)
    blocks=list(re.finditer(r'^"(CMapEntity|CMapMesh)"\n\{\n.*?^\}',text,re.M|re.S))
    entities=[];mesh_blocks={}
    for match in blocks:
        block=match.group()
        if match.group(1)=='CMapEntity' and re.search(r'"classname"\s+"string"\s+"env_cs_place"',block):
            entities.append(_object(block))
        elif match.group(1)=='CMapMesh':
            identity=re.search(r'^\t"id" "elementid" "([^"]+)"',block,re.M)
            _require(identity is not None and identity[1] not in mesh_blocks, "ambiguous mesh ID")
            mesh_blocks[identity[1]]=block
    _require(len(entities)==len(re.findall(r'"classname"\s+"string"\s+"env_cs_place"',text)), "unsupported nested place entity")
    regions=[];excluded=[]
    for entity in entities:
        _require(world['children'].count(entity['id']) == 1, "place is not a direct world child")
        label=entity['entity_properties']['place_name']
        if label.startswith('position.'):
            excluded.append({'id':entity['nodeID'],'label':label});continue
        _identity(entity,translated=True)
        children=entity['children']
        _require(children and all(isinstance(c,str) and c in mesh_blocks for c in children), "missing linked place mesh")
        hulls=[_mesh_hull(_object(mesh_blocks[c])) for c in children]
        regions.append({'id':entity['nodeID'],'label':label,'entity_id':entity['id'],
                        'entity_origin':_vector(entity['origin']),'mesh_ids':children,'hulls':hulls})
    package={'schema_id':SCHEMA_ID,'map_name':map_name,'build_compatibility':'unverified',
             'coordinate_space':'source2_world_baked_mesh','source':{'artifact':path.name,'sha256':hashlib.sha256(raw).hexdigest(),'bytes':len(raw)},
             'regions':sorted(regions,key=lambda r:r['id']),'excluded_internal_labels':excluded,
             'transform_policy':'VRF physics hull world transform already baked; entity origin not reapplied; identity mesh transforms required'}
    package['reference_id']=digest(package)
    return validate(package)


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source',type=Path,required=True)
    parser.add_argument('--map',required=True)
    parser.add_argument('--output',type=Path,required=True)
    args=parser.parse_args()
    _require(not args.output.exists(), 'refusing to overwrite place package')
    package=extract_vmap(args.source,map_name=args.map)
    args.output.parent.mkdir(parents=True,exist_ok=True)
    with args.output.open('xb') as output:output.write(encoded(package)+b'\n')
    print(json.dumps({'regions':len(package['regions']),'hulls':sum(len(r['hulls']) for r in package['regions']),'reference_id':package['reference_id']}))


if __name__=='__main__':main()
