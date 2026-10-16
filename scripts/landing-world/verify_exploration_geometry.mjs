import { createRequire } from 'node:module';
import { resolve } from 'node:path';
import { writeFile } from 'node:fs/promises';
const require=createRequire(resolve('output/landing-world-assets-20261015/asset-tools/package.json'));
const {NodeIO}=require('@gltf-transform/core');
const {ALL_EXTENSIONS}=require('@gltf-transform/extensions');
const draco=require('draco3dgltf');
const {MeshoptDecoder}=require('meshoptimizer'); await MeshoptDecoder.ready;
const io=new NodeIO().registerExtensions(ALL_EXTENSIONS).registerDependencies({'draco3d.decoder':await draco.createDecoderModule(),'meshopt.decoder':MeshoptDecoder});
const paths=['output/landing-world-exploration-20261015/mirage-regions-final.glb',process.argv[2]||'frontend/public/landing-world/mirage-exploration.glb'];
const reports=[];
let reference;
for(const path of paths){
 const doc=await io.read(path);let triangles=0,degenerate=0,minArea=Infinity;
 const primitives=doc.getRoot().listMeshes().filter(m=>m.getName()==='site_high').flatMap(m=>m.listPrimitives());
 if(reference){
  if(primitives.length!==reference.length)throw new Error('A primitive count changed');
  for(let i=0;i<primitives.length;i++)for(const semantic of ['INDICES',...reference[i].listSemantics()]){
   const source=(semantic==='INDICES'?reference[i].getIndices():reference[i].getAttribute(semantic)).getArray();
   const target=(semantic==='INDICES'?primitives[i].getIndices():primitives[i].getAttribute(semantic)).getArray();
   if(source.length!==target.length)throw new Error(`Native A length changed: ${i}/${semantic}`);
   if(semantic==='INDICES'){
    // Meshopt may cyclically rotate a triangle's indices; this preserves the
    // exact face, winding, and interpolated vertex attributes.
    for(let j=0;j<source.length;j+=3){let same=false;for(let r=0;r<3;r++)if([0,1,2].every(k=>source[j+k]===target[j+(k+r)%3]))same=true;if(!same)throw new Error(`Native A topology changed: ${i}/${j}`);}
   }else if(!source.every((value,index)=>value===target[index]))throw new Error(`Native A attribute changed: ${i}/${semantic}`);
  }
 }else reference=primitives;
 for(const mesh of doc.getRoot().listMeshes().filter(m=>m.getName()==='site_high'))for(const p of mesh.listPrimitives()){
  const a=p.getAttribute('POSITION').getArray(),idx=p.getIndices().getArray();
  for(let i=0;i<idx.length;i+=3){
   const x=idx[i]*3,y=idx[i+1]*3,z=idx[i+2]*3;
   const u=[a[y]-a[x],a[y+1]-a[x+1],a[y+2]-a[x+2]],v=[a[z]-a[x],a[z+1]-a[x+1],a[z+2]-a[x+2]];
   const area=Math.hypot(u[1]*v[2]-u[2]*v[1],u[2]*v[0]-u[0]*v[2],u[0]*v[1]-u[1]*v[0])/2;
   if(area<1e-10)degenerate++;else minArea=Math.min(minArea,area);triangles++;
  }
 }
 reports.push({path,triangles,degenerate,minArea,originalAAttributesAndTopologyIdentical:path===paths[1]});
}
await writeFile('output/landing-world-exploration-20261015/geometry-validation.json',JSON.stringify(reports,null,2));console.log(JSON.stringify(reports));
