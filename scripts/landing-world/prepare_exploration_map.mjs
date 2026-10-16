import { createRequire } from 'node:module';
import { resolve } from 'node:path';
import { access, writeFile } from 'node:fs/promises';

const require = createRequire(resolve('output/landing-world-assets-20261015/asset-tools/package.json'));
const { NodeIO } = require('@gltf-transform/core');
const { ALL_EXTENSIONS } = require('@gltf-transform/extensions');
const { compactPrimitive, transformPrimitive, simplifyPrimitive, weldPrimitive, joinPrimitives, prune, dedup } = require('@gltf-transform/functions');
const { MeshoptSimplifier } = require('meshoptimizer');
const sharp = require('sharp');
const [input, output] = process.argv.slice(2);
if (!input || !output || resolve(input) === resolve(output)) throw new Error('Use distinct source and NEW output paths.');
try { await access(output); throw new Error('Output exists.'); } catch (error) { if (error.code !== 'ENOENT') throw error; }
await MeshoptSimplifier.ready;
const io = new NodeIO().registerExtensions(ALL_EXTENSIONS);
const document = await io.read(input), root = document.getRoot(), buffer = root.listBuffers()[0];
const sourceScenes = root.listScenes(), instances = [];
const sourceNodes = root.listNodes(), sourceMeshes = root.listMeshes();
for (const scene of sourceScenes) scene.traverse(node => { if (node.getMesh()) instances.push({ mesh: node.getMesh(), matrix: node.getWorldMatrix() }); });
const scene = document.createScene('Mirage_exploration'); root.setDefaultScene(scene);
const groups = { site_high: [], site_distant: [], background: [] };
const nearTextures = new Set();
let originalTriangles = 0, protectedTriangles = 0, distantTriangles = 0, backgroundTriangles = 0, windColorPrimitives = 0;
// Includes stairs, Tetris, CT boxes, palms and a margin around every allowed
// camera target. Triangle bounds preserve shapes crossing the boundary.
const inSite = (xmin, xmax, zmin, zmax) => xmax >= -16 && xmin <= 24 && zmax >= 4 && zmin <= 43;
for (const instance of instances) for (const original of instance.mesh.listPrimitives()) {
  if (original.getMode() !== 4 || !original.getIndices()) throw new Error('Expected indexed static triangles.');
  const p = original.clone(); compactPrimitive(p); transformPrimitive(p, instance.matrix);
  const material = p.getMaterial(), foliage = /palm|trees_branches|claypot_plants/.test(material?.getName() || '');
  // Source foliage COLOR_0 stores wind/variation weights, not PBR albedo.
  // Applying it as vertex tint makes the exported palm fronds red/black.
  if (foliage && p.getAttribute('COLOR_0')) { p.setAttribute('COLOR_0', null); windColorPrimitives++; }
  const pos = p.getAttribute('POSITION').getArray(), indices = p.getIndices().getArray();
  const parts = [[], []]; originalTriangles += indices.length / 3;
  for (let i = 0; i < indices.length; i += 3) {
    const a = indices[i] * 3, b = indices[i + 1] * 3, c = indices[i + 2] * 3;
    const near = inSite(Math.min(pos[a],pos[b],pos[c]),Math.max(pos[a],pos[b],pos[c]),Math.min(pos[a+2],pos[b+2],pos[c+2]),Math.max(pos[a+2],pos[b+2],pos[c+2]));
    parts[near ? 0 : 1].push(indices[i],indices[i+1],indices[i+2]);
  }
  for (let region = 0; region < 2; region++) {
    const oldIndices = parts[region]; if (!oldIndices.length) continue;
    const part = document.createPrimitive().setMaterial(material), remap = new Map(), unique = [], local = new Uint32Array(oldIndices.length);
    for (let i = 0; i < oldIndices.length; i++) { const old = oldIndices[i]; if (!remap.has(old)) { remap.set(old, unique.length); unique.push(old); } local[i] = remap.get(old); }
    part.setIndices(document.createAccessor().setType('SCALAR').setArray(local).setBuffer(buffer));
    for (const semantic of p.listSemantics()) {
      const a = p.getAttribute(semantic), source = a.getArray(), size = a.getElementSize(), array = new source.constructor(unique.length * size);
      for (let i = 0; i < unique.length; i++) for (let j = 0; j < size; j++) array[i*size+j] = source[unique[i]*size+j];
      part.setAttribute(semantic, document.createAccessor().setType(a.getType()).setNormalized(a.getNormalized()).setArray(array).setBuffer(buffer));
    }
    if (region === 0) {
      groups.site_high.push(part); protectedTriangles += local.length / 3;
      for (const slot of ['BaseColor','Normal','MetallicRoughness','Occlusion','Emissive']) { const t = material?.[`get${slot}Texture`](); if (t) nearTextures.add(t); }
      const low = part.clone(); compactPrimitive(low); weldPrimitive(low);
      simplifyPrimitive(low, { simplifier: MeshoptSimplifier, ratio: .08, error: .04, lockBorder: false });
      distantTriangles += low.getIndices().getCount()/3; groups.site_distant.push(low);
    } else {
      weldPrimitive(part); simplifyPrimitive(part, { simplifier: MeshoptSimplifier, ratio: .07, error: .05, lockBorder: false });
      backgroundTriangles += part.getIndices().getCount()/3; groups.background.push(part);
    }
  }
  p.dispose();
}
for (const [name, parts] of Object.entries(groups)) {
  const node = document.createNode(name), compatible = new Map();
  for (const p of parts) { const key = root.listMaterials().indexOf(p.getMaterial())+'/'+p.listSemantics().join(); if (!compatible.has(key)) compatible.set(key, []); compatible.get(key).push(p); }
  const mesh = document.createMesh(name);
  for (const parts of compatible.values()) mesh.addPrimitive(parts.length === 1 ? parts[0] : joinPrimitives(parts));
  scene.addChild(node.setMesh(mesh));
}
for (const source of sourceScenes) source.dispose();
for (const node of sourceNodes) node.dispose();
for (const mesh of sourceMeshes) mesh.dispose();
// Keep the high-detail texture identities before dedup can dispose equivalent
// references. All variants share images inside one GLB: no duplicate VRAM.
for (const texture of root.listTextures()) {
  const image = sharp(texture.getImage()), cap = nearTextures.has(texture) ? 1024 : 256;
  texture.setImage(await image.resize({width:cap,height:cap,fit:'inside',withoutEnlargement:true}).png().toBuffer()).setMimeType('image/png');
}
await document.transform(prune(), dedup());
await io.write(output, document);
const report = { originalTriangles, protectedTriangles, highDetailSimplification: false, distantTriangles, backgroundTriangles, windColorPrimitives,
  images: root.listTextures().length, primitives: root.listMeshes().reduce((n,m)=>n+m.listPrimitives().length,0), nearTextureCap:1024, backgroundTextureCap:256 };
await writeFile(`${output}.json`,JSON.stringify(report,null,2)); console.log(JSON.stringify(report));
