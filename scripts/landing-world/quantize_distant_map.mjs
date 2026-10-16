import { createRequire } from 'node:module';
import { resolve } from 'node:path';
import { access } from 'node:fs/promises';
const require=createRequire(resolve('output/landing-world-assets-20261015/asset-tools/package.json'));
const {NodeIO}=require('@gltf-transform/core');
const {ALL_EXTENSIONS}=require('@gltf-transform/extensions');
const [input,output]=process.argv.slice(2);
if(!input||!output||resolve(input)===resolve(output))throw new Error('Use new output.');
try{await access(output);throw new Error('Output exists')}catch(error){if(error.code!=='ENOENT')throw error}
const io=new NodeIO().registerExtensions(ALL_EXTENSIONS),document=await io.read(input);
for(const mesh of document.getRoot().listMeshes()){
 if(mesh.getName()==='site_high')continue;
 for(const primitive of mesh.listPrimitives())for(const semantic of primitive.listSemantics()){
  const step=semantic==='POSITION'?128:semantic==='NORMAL'?512:semantic.startsWith('TEXCOORD')?2048:0;
  if(!step)continue;
  const attribute=primitive.getAttribute(semantic),source=attribute.getArray();
  if(!(source instanceof Float32Array))continue;
  const array=new Float32Array(source.length);
  for(let i=0;i<array.length;i++)array[i]=Math.round(source[i]*step)/step;
  // Always detach: a distant quad may still share accessors with original A.
  primitive.setAttribute(semantic,attribute.clone().setArray(array));
 }
}
await io.write(output,document);console.log('Reduced precision only outside site_high.');
