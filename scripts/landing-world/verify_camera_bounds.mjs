import { createRequire } from 'node:module';
import { resolve } from 'node:path';
import { writeFile } from 'node:fs/promises';
import { CAMERA_LIMITS,constrainNavigation,writeExplorationPose } from '../../frontend/src/features/landing-world/cameraNavigation.js';
const require=createRequire(resolve('output/landing-world-assets-20261015/asset-tools/package.json'));
const frontend=createRequire(resolve('frontend/package.json'));
const THREE=frontend('three'),{MeshBVH,acceleratedRaycast}=createRequire(frontend.resolve('@react-three/drei'))('three-mesh-bvh');
const {NodeIO}=require('@gltf-transform/core');
const document=await new NodeIO().read('output/landing-world-exploration-20261015/mirage-regions-final.glb');
const meshes=[];
for(const mesh of document.getRoot().listMeshes().filter(m=>m.getName()!=='site_distant'))for(const p of mesh.listPrimitives()){
 const material=p.getMaterial();if(/palm|trees_|plants/.test(material?.getName()||'')||material?.getAlphaMode()!=='OPAQUE')continue;
 const geometry=new THREE.BufferGeometry();geometry.setAttribute('position',new THREE.BufferAttribute(p.getAttribute('POSITION').getArray(),3));geometry.setIndex(new THREE.BufferAttribute(p.getIndices().getArray(),1));
 geometry.boundsTree=new MeshBVH(geometry);
 const object=new THREE.Mesh(geometry,new THREE.MeshBasicMaterial({side:THREE.DoubleSide}));object.raycast=acceleratedRaycast;object.updateMatrixWorld();meshes.push(object);
}
const ray=new THREE.Raycaster(),direction=new THREE.Vector3(0,-1,0),origin=new THREE.Vector3();ray.firstHitOnly=true;
const belowRoof=[];let samples=0;
const range=key=>[CAMERA_LIMITS[key][0],(CAMERA_LIMITS[key][0]+CAMERA_LIMITS[key][1])/2,CAMERA_LIMITS[key][1]];
for(const x of range('x'))for(const z of range('z'))for(const radius of range('distance'))for(const azimuth of range('yaw'))for(const elevation of range('elevation'))for(const px of [-1,1])for(const py of [-1,1]){
 const view=constrainNavigation({x,z,distance:radius,yaw:azimuth,elevation}),position=new THREE.Vector3(),target=new THREE.Vector3();
 writeExplorationPose(view,position,target,{x:px,y:py});const camera=position.toArray();
 origin.set(camera[0],45,camera[2]);ray.set(origin,direction);const hits=ray.intersectObjects(meshes,false);samples++;
 if(hits[0]&&hits[0].point.y>camera[1]-.6)belowRoof.push({view,camera,roof: +hits[0].point.y.toFixed(3)});
}
const report={samples,belowRoofCount:belowRoof.length,belowRoof};
await writeFile('output/landing-world-exploration-20261015/camera-volume-validation.json',JSON.stringify(report,null,2));console.log(JSON.stringify(report));
for(const mesh of meshes){mesh.geometry.dispose();mesh.material.dispose();}
