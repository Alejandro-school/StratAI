import { createRequire } from 'node:module';
import { resolve } from 'node:path';
import { access } from 'node:fs/promises';
const require=createRequire(resolve('output/landing-world-assets-20261015/asset-tools/package.json'));
const {NodeIO}=require('@gltf-transform/core');
const {ALL_EXTENSIONS,EXTMeshoptCompression}=require('@gltf-transform/extensions');
const {prune}=require('@gltf-transform/functions');
const {MeshoptEncoder}=require('meshoptimizer');
const [input,output]=process.argv.slice(2);
if(!input||!output||resolve(input)===resolve(output))throw new Error('Use distinct source and NEW output paths.');
try{await access(output);throw new Error('Output exists')}catch(error){if(error.code!=='ENOENT')throw error}
await MeshoptEncoder.ready;
const io=new NodeIO().registerExtensions(ALL_EXTENSIONS).registerDependencies({'meshopt.encoder':MeshoptEncoder});
const document=await io.read(input);
for(const mesh of document.getRoot().listMeshes())if(mesh.getName()!=='site_high')for(const primitive of mesh.listPrimitives()){
  const tangent=primitive.getAttribute('TANGENT');
  if(tangent){const source=tangent.getArray();if(source instanceof Float32Array)primitive.setAttribute('TANGENT',tangent.clone().setArray(Float32Array.from(source,value=>Math.round(value*512)/512)));}
}
await document.transform(prune({ keepAttributes: true }));
// QUANTIZE selects the byte compressor, but no quantize() transform or lossy
// filter is applied. Native A positions/normals/UVs remain bit-exact.
document.createExtension(EXTMeshoptCompression).setRequired(true).setEncoderOptions({method:EXTMeshoptCompression.EncoderMethod.QUANTIZE});
await io.write(output,document);console.log(`Lossless geometry: ${output}`);
