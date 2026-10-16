import { createRequire } from 'node:module';
import { resolve } from 'node:path';
const require = createRequire(resolve('output/landing-world-assets-20261015/asset-tools/package.json'));
const { NodeIO } = require('@gltf-transform/core');
const { ALL_EXTENSIONS } = require('@gltf-transform/extensions');
const draco = require('draco3dgltf');
const THREE = createRequire(resolve('frontend/package.json'))('three');
const io = new NodeIO().registerExtensions(ALL_EXTENSIONS).registerDependencies({'draco3d.decoder':await draco.createDecoderModule()});
const doc = await io.read('frontend/public/landing-world/mirage-lowpc.glb');
const primitives=doc.getRoot().listMeshes().flatMap(mesh=>mesh.listPrimitives());
const camera=new THREE.PerspectiveCamera(55,2554/1221,.15,600);camera.position.set(-6,17,34);camera.lookAt(3,0,26);camera.updateMatrixWorld();
const frustum=new THREE.Frustum().setFromProjectionMatrix(new THREE.Matrix4().multiplyMatrices(camera.projectionMatrix,camera.matrixWorldInverse));
function estimate(groups){
 let calls=0,triangles=0;
 for(const group of groups){
  const box=new THREE.Box3(), point=new THREE.Vector3();
  for(const primitive of group){const array=primitive.getAttribute('POSITION').getArray();for(let i=0;i<array.length;i+=3)box.expandByPoint(point.fromArray(array,i));}
  const center=box.getCenter(new THREE.Vector3());let radius2=0;
  for(const primitive of group){const array=primitive.getAttribute('POSITION').getArray();for(let i=0;i<array.length;i+=3)radius2=Math.max(radius2,center.distanceToSquared(point.fromArray(array,i)));}
  if(frustum.intersectsSphere(new THREE.Sphere(center,Math.sqrt(radius2)))){calls++;triangles+=group.reduce((n,p)=>n+p.getIndices().getCount()/3,0);}
 }
 return{groups:groups.length,calls,triangles};
}
for(const limit of [0,200,500,1000,2000,5000,Infinity]){
 const small=new Map(),groups=[];
 for(const p of primitives){
  if(p.getIndices().getCount()/3>=limit)groups.push([p]);
  else{const key=doc.getRoot().listMaterials().indexOf(p.getMaterial())+'/'+p.listSemantics().join();if(!small.has(key))small.set(key,[]);small.get(key).push(p);}
 }
 console.log(JSON.stringify({limit:String(limit),...estimate([...groups,...small.values()])}));
}
