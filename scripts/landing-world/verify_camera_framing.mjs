// Offline source-geometry coverage check, complementary to roof clearance.
// Opaque, double-sided rays are a sampled proxy, not a rendered-image verdict.
import { createRequire } from 'node:module';
import { resolve } from 'node:path';
import { writeFile } from 'node:fs/promises';
import { CAMERA_LIMITS, constrainNavigation, createNavigation, writeExplorationPose } from '../../frontend/src/features/landing-world/cameraNavigation.js';
const require = createRequire(resolve('output/landing-world-assets-20261015/asset-tools/package.json'));
const frontend = createRequire(resolve('frontend/package.json'));
const THREE = frontend('three');
const { MeshBVH, acceleratedRaycast } = createRequire(frontend.resolve('@react-three/drei'))('three-mesh-bvh');
const { NodeIO } = require('@gltf-transform/core');
const document = await new NodeIO().read('output/landing-world-exploration-20261015/mirage-regions-final.glb');
const positions = [], indices = [];
for (const mesh of document.getRoot().listMeshes().filter(mesh => mesh.getName() !== 'site_distant')) for (const primitive of mesh.listPrimitives()) {
  if (primitive.getMaterial()?.getAlphaMode() !== 'OPAQUE') continue;
  const offset = positions.length / 3;
  for (const value of primitive.getAttribute('POSITION').getArray()) positions.push(value);
  for (const value of primitive.getIndices().getArray()) indices.push(value + offset);
}
const geometry = new THREE.BufferGeometry();
geometry.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3)); geometry.setIndex(indices);
geometry.boundsTree = new MeshBVH(geometry);
const mesh = new THREE.Mesh(geometry, new THREE.MeshBasicMaterial({ side: THREE.DoubleSide })); mesh.raycast = acceleratedRaycast; mesh.updateMatrixWorld();
const camera = new THREE.PerspectiveCamera(55, 2560 / 1221, .15, 600), target = new THREE.Vector3();
const ray = new THREE.Raycaster(); ray.firstHitOnly = true;
const results = [], center = createNavigation();
const range = key => [CAMERA_LIMITS[key][0], center[key], CAMERA_LIMITS[key][1]];
for (const x of range('x')) for (const z of range('z')) for (const distance of range('distance')) for (const yaw of range('yaw')) for (const elevation of range('elevation')) for (const zoom of [.8, 1.4, 2.1]) for (const px of [-1, 1]) for (const py of [-1, 1]) {
  const view = constrainNavigation({ x, z, distance, yaw, elevation, zoom });
  writeExplorationPose(view, camera.position, target, { x: px, y: py }); camera.zoom = zoom; camera.lookAt(target); camera.updateProjectionMatrix(); camera.updateMatrixWorld();
  let misses = 0;
  for (let yy = 0; yy < 11; yy++) for (let xx = 0; xx < 19; xx++) {
    ray.setFromCamera(new THREE.Vector2(-.9 + xx * .1, .8 - yy * .16), camera);
    if (!ray.intersectObject(mesh, false).length) misses++;
  }
  results.push({ view, pointer: { x: px, y: py }, misses });
}
results.sort((a,b) => b.misses-a.misses);
const report = { samples: results.length, raysPerSample: 209, viewport: [2560,1221], coverage: '90% horizontal / 80% vertical; opaque source geometry, both faces', byZoom: [.8,1.4,2.1].map(zoom => ({zoom,samples:results.filter(r=>r.view.zoom===zoom).length,maxMisses:Math.max(...results.filter(r=>r.view.zoom===zoom).map(r=>r.misses))})), worst: results.slice(0,4) };
await writeFile('output/landing-world-exploration-20261015/camera-framing-validation.json', JSON.stringify(report,null,2));
console.log(JSON.stringify(report));
geometry.dispose(); mesh.material.dispose();
