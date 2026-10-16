import { createRequire } from 'node:module';
import { resolve } from 'node:path';
import { access, writeFile } from 'node:fs/promises';
const require=createRequire(resolve('output/landing-world-assets-20261015/asset-tools/package.json'));
const {NodeIO}=require('@gltf-transform/core');
const {ALL_EXTENSIONS,KHRDracoMeshCompression}=require('@gltf-transform/extensions');
const {joinPrimitives,prune}=require('@gltf-transform/functions');
const draco=require('draco3dgltf');
const [input,output]=process.argv.slice(2);
if(!input||!output||resolve(input)===resolve(output))throw new Error('Use distinct input and new output paths');
try{await access(output);throw new Error('Output exists')}catch(error){if(error.code!=='ENOENT')throw error}
const io=new NodeIO().registerExtensions(ALL_EXTENSIONS).registerDependencies({'draco3d.decoder':await draco.createDecoderModule(),'draco3d.encoder':await draco.createEncoderModule()});
const document=await io.read(input),root=document.getRoot(),source=root.listScenes(),scene=document.createScene('Mirage_lowpc');root.setDefaultScene(scene);
const small=new Map();let retained=0;
// Large spatial pieces retain frustum culling. Tiny pieces sharing a material
// are batched globally: fewer draw calls and scene nodes on modest CPUs.
for(const mesh of root.listMeshes())for(const primitive of mesh.listPrimitives()){
  if(primitive.getIndices().getCount()/3>=2000){scene.addChild(document.createNode().setMesh(document.createMesh().addPrimitive(primitive)));retained++;}
  else{const key=root.listMaterials().indexOf(primitive.getMaterial())+'/'+primitive.listSemantics().join();if(!small.has(key))small.set(key,[]);small.get(key).push(primitive);}
}
for(const group of small.values())scene.addChild(document.createNode().setMesh(document.createMesh().addPrimitive(group.length===1?group[0]:joinPrimitives(group))));
for(const old of source)old.dispose();
await document.transform(prune());
for(const extension of root.listExtensionsUsed())if(extension.extensionName==='EXT_texture_webp')extension.dispose();
document.createExtension(KHRDracoMeshCompression).setRequired(true).setEncoderOptions({method:'edgebreaker',encodeSpeed:5,decodeSpeed:7});
await io.write(output,document);
const report={smallPrimitiveThreshold:2000,spatialLargePieces:retained,batchedMaterialGroups:small.size,primitives:root.listMeshes().reduce((n,m)=>n+m.listPrimitives().length,0)};
await writeFile(`${output}.json`,JSON.stringify(report,null,2));console.log(JSON.stringify(report));
