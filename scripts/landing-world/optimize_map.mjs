// Offline-only tools live outside the application dependency tree.
import { createRequire } from 'node:module';
import { resolve } from 'node:path';
import { writeFile, mkdir, access } from 'node:fs/promises';
import { parseArgs } from 'node:util';

const { values } = parseArgs({ options: {
  input: { type: 'string' }, output: { type: 'string' },
  tools: { type: 'string', default: 'output/landing-world-assets-20261015/asset-tools' },
  tile: { type: 'string', default: '24' },
} });
if (!values.input || !values.output || resolve(values.input) === resolve(values.output)) throw new Error('Use distinct input and new output paths.');
try { await access(values.output); throw new Error('Refusing to overwrite an asset.'); } catch (error) { if (error.code !== 'ENOENT') throw error; }
const require = createRequire(resolve(values.tools, 'package.json'));
const { NodeIO } = require('@gltf-transform/core');
const { ALL_EXTENSIONS, KHRDracoMeshCompression } = require('@gltf-transform/extensions');
const { compactPrimitive, transformPrimitive, simplifyPrimitive, weldPrimitive, joinPrimitives, prune, dedup } = require('@gltf-transform/functions');
const { MeshoptSimplifier } = require('meshoptimizer');
const draco = require('draco3dgltf');
const sharp = require('sharp');
await MeshoptSimplifier.ready;
const io = new NodeIO().registerExtensions(ALL_EXTENSIONS).registerDependencies({
  'draco3d.decoder': await draco.createDecoderModule(), 'draco3d.encoder': await draco.createEncoderModule(),
});
const document = await io.read(values.input);
const root = document.getRoot();
const buffer = root.listBuffers()[0];
const sourceScenes = root.listScenes();
const instances = [];
for (const scene of sourceScenes) scene.traverse(node => { if (node.getMesh()) instances.push({ mesh: node.getMesh(), matrix: node.getWorldMatrix() }); });
const outputScene = document.createScene('Mirage_spatial_lowpc'); root.setDefaultScene(outputScene);
const tiles = new Map(), nearMaterials = new Set();
const tileSize = Number(values.tile);
if (!(tileSize >= 8 && tileSize <= 64)) throw new Error('Tile size must be between 8 and 64 world units.');
let before = 0, after = 0, beforeVertices = 0, afterVertices = 0;
const tileIntersectsSite = (x, z) => x * tileSize < 24 && (x + 1) * tileSize > -18 && z * tileSize < 43 && (z + 1) * tileSize > 5;

for (const instance of instances) for (const original of instance.mesh.listPrimitives()) {
  if (original.getMode() !== 4 || !original.getIndices()) throw new Error('Expected an indexed triangle-only static map.');
  const primitive = original.clone();
  compactPrimitive(primitive); transformPrimitive(primitive, instance.matrix);
  const positions = primitive.getAttribute('POSITION').getArray();
  const indices = primitive.getIndices().getArray();
  const groups = new Map(); before += indices.length / 3; beforeVertices += positions.length / 3;
  for (let i = 0; i < indices.length; i += 3) {
    const x = Math.floor((positions[indices[i] * 3] + positions[indices[i + 1] * 3] + positions[indices[i + 2] * 3]) / (3 * tileSize));
    const z = Math.floor((positions[indices[i] * 3 + 2] + positions[indices[i + 1] * 3 + 2] + positions[indices[i + 2] * 3 + 2]) / (3 * tileSize));
    const key = `${x},${z}`;
    if (!groups.has(key)) groups.set(key, { x, z, indices: [] });
    groups.get(key).indices.push(indices[i], indices[i + 1], indices[i + 2]);
  }
  for (const [key, group] of groups) {
    const material = primitive.getMaterial();
    const near = tileIntersectsSite(group.x, group.z);
    if (near && material) nearMaterials.add(material);
    const part = document.createPrimitive().setMaterial(material);
    const remap = new Map(), unique = [], localIndices = new Uint32Array(group.indices.length);
    for (let i = 0; i < group.indices.length; i++) {
      const oldIndex = group.indices[i];
      if (!remap.has(oldIndex)) { remap.set(oldIndex, unique.length); unique.push(oldIndex); }
      localIndices[i] = remap.get(oldIndex);
    }
    part.setIndices(document.createAccessor().setType('SCALAR').setArray(localIndices).setBuffer(buffer));
    for (const semantic of primitive.listSemantics()) {
      const accessor = primitive.getAttribute(semantic), source = accessor.getArray(), size = accessor.getElementSize();
      const array = new source.constructor(unique.length * size);
      for (let i = 0; i < unique.length; i++) for (let c = 0; c < size; c++) array[i * size + c] = source[unique[i] * size + c];
      part.setAttribute(semantic, document.createAccessor().setType(accessor.getType()).setNormalized(accessor.getNormalized()).setArray(array).setBuffer(buffer));
    }
    weldPrimitive(part);
    simplifyPrimitive(part, { simplifier: MeshoptSimplifier, ratio: near ? .45 : .12, error: near ? .001 : .01, lockBorder: true });
    after += part.getIndices().getCount() / 3; afterVertices += part.getAttribute('POSITION').getCount();
    if (!tiles.has(key)) tiles.set(key, []);
    tiles.get(key).push(part);
  }
  primitive.dispose();
}

for (const [key, parts] of tiles) {
  const mesh = document.createMesh(`tile_${key}`);
  const compatible = new Map();
  for (const part of parts) {
    const key = `${part.getMaterial()?.getName()}/${root.listMaterials().indexOf(part.getMaterial())}/${part.listSemantics().join()}`;
    if (!compatible.has(key)) compatible.set(key, []);
    compatible.get(key).push(part);
  }
  for (const group of compatible.values()) mesh.addPrimitive(group.length === 1 ? group[0] : joinPrimitives(group));
  outputScene.addChild(document.createNode(`tile_${key}`).setMesh(mesh));
}
for (const scene of sourceScenes) scene.dispose();
await document.transform(prune(), dedup());

// Keep 512px native textures around A; only materials exclusive to distant
// scenery use 256px. Every original UV/material assignment remains in place.
const protectedTextures = new Set();
for (const material of nearMaterials) for (const slot of ['BaseColor', 'Normal', 'MetallicRoughness', 'Occlusion', 'Emissive']) {
  const texture = material[`get${slot}Texture`](); if (texture) protectedTextures.add(texture);
}
let resized = 0;
for (const texture of root.listTextures()) {
  if (protectedTextures.has(texture)) continue;
  const source = sharp(texture.getImage()), metadata = await source.metadata();
  if (metadata.width <= 256 && metadata.height <= 256) continue;
  texture.setImage(await source.resize({ width: 256, height: 256, fit: 'inside', withoutEnlargement: true }).png().toBuffer()).setMimeType('image/png');
  resized++;
}
document.createExtension(KHRDracoMeshCompression).setRequired(true).setEncoderOptions({ method: 'edgebreaker', encodeSpeed: 5, decodeSpeed: 7 });
await mkdir(resolve(values.output, '..'), { recursive: true });
await io.write(values.output, document);
const report = { input: values.input, output: values.output, tileSize, nearRatio: .45, farRatio: .12, nearError: .001, farError: .01, lockBorders: true,
  trianglesBefore: before, trianglesAfter: after, verticesBefore: beforeVertices, verticesAfter: afterVertices,
  tiles: tiles.size, primitives: root.listMeshes().reduce((sum, mesh) => sum + mesh.listPrimitives().length, 0), protectedMaterials: nearMaterials.size,
  textures: root.listTextures().length, texturesResizedOnlyInDistance: resized };
await writeFile(`${values.output}.json`, JSON.stringify(report, null, 2));
console.log(JSON.stringify(report));
