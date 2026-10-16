"""Prepare a read-only Source 2 Viewer export for the landing asset pipeline."""
import argparse
import json
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument('source', type=Path)
parser.add_argument('destination', type=Path)
args = parser.parse_args()
if args.destination.exists():
    raise SystemExit('Refusing to overwrite a prepared export.')
document = json.loads(args.source.read_text(encoding='utf-8'))
removed_primitives = 0
for mesh in document['meshes']:
    visible = []
    for primitive in mesh['primitives']:
        material = document['materials'][primitive['material']]
        material_path = material.get('extras', {}).get('vmat', {}).get('Name', '')
        if '/tools/' in material_path.replace('\\', '/'):
            removed_primitives += 1
        else:
            visible.append(primitive)
    mesh['primitives'] = visible

nodes = document['nodes']
removed_nodes = set()
for index, node in enumerate(nodes):
    if 'mesh' in node and not document['meshes'][node['mesh']]['primitives']:
        node.pop('mesh')
        removed_nodes.add(index)
    # Use one landing lighting rig; do not import the game's hundreds of lights.
    if 'KHR_lights_punctual' in node.get('extensions', {}):
        node['extensions'].pop('KHR_lights_punctual')
        if 'mesh' not in node:
            removed_nodes.add(index)
for node in nodes:
    if 'children' in node:
        node['children'] = [i for i in node['children'] if i not in removed_nodes]

# VRF 20.1 exports Source inches as metres with axes y,z,x.
# Its verified Conversion.cs uses yaw=0, pitch=-pi/2, roll=-pi/2.
# A +pi/2 Y rotation restores the landing's documented axes x,z,-y.
scale = .02 / .0254
for scene in document['scenes']:
    root = len(nodes)
    nodes.append({'name': 'StratAI_Mirage_source_alignment',
                  'translation': [13.4, 3.2, -16.94],
                  'rotation': [0, 2 ** -.5, 0, 2 ** -.5],
                  'scale': [scale, scale, scale],
                  'children': [i for i in scene['nodes'] if i not in removed_nodes]})
    scene['nodes'] = [root]
document.get('extensions', {}).pop('KHR_lights_punctual', None)
for key in ['extensionsUsed', 'extensionsRequired']:
    if key in document:
        document[key] = [name for name in document[key] if name != 'KHR_lights_punctual']
for material in document['materials']:
    material.pop('extras', None)
args.destination.write_text(json.dumps(document), encoding='utf-8')
print(json.dumps({'helper_primitives_removed': removed_primitives,
                  'helper_or_light_nodes_removed': len(removed_nodes),
                  'source': str(args.source), 'prepared': str(args.destination)}))
