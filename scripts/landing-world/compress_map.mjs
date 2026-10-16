import { createRequire } from 'node:module';
import { resolve } from 'node:path';
import { access } from 'node:fs/promises';

// Isolated offline tools; no dependency is added to the application.
const require = createRequire(resolve('output/landing-world-assets-20261015/asset-tools/package.json'));
const { NodeIO } = require('@gltf-transform/core');
const { ALL_EXTENSIONS, KHRDracoMeshCompression } = require('@gltf-transform/extensions');
const { prune } = require('@gltf-transform/functions');
const draco = require('draco3dgltf');
const sharp = require('sharp');
const [mode, input, output] = process.argv.slice(2);
if (!['png', 'draco', 'draco-high'].includes(mode) || !input || !output || resolve(input) === resolve(output)) throw new Error('Usage: node compress_map.mjs png|draco|draco-high input.glb NEW-output.glb');
try { await access(output); throw new Error('Output already exists; retain intermediate attempts.'); } catch (error) { if (error.code !== 'ENOENT') throw error; }
const io = new NodeIO().registerExtensions(ALL_EXTENSIONS).registerDependencies({
  'draco3d.decoder': await draco.createDecoderModule(), 'draco3d.encoder': await draco.createEncoderModule(),
});
const document = await io.read(input);
if (mode === 'png') {
  // toktx accepts PNG/JPEG, not WebP. This changes container only, without
  // another lossy image encoding or additional texture resizing.
  for (const texture of document.getRoot().listTextures()) {
    if (texture.getMimeType() === 'image/ktx2') continue;
    texture.setImage(await sharp(texture.getImage()).png().toBuffer()).setMimeType('image/png');
  }
} else {
  document.createExtension(KHRDracoMeshCompression).setRequired(true).setEncoderOptions({ method: 'edgebreaker', encodeSpeed: 5, decodeSpeed: 7,
    ...(mode === 'draco-high' ? { quantizePosition: 20, quantizeNormal: 12, quantizeTexcoord: 14, quantizeColor: 10 } : {}),
  });
}
await document.transform(prune());
await io.write(output, document);
console.log(`${mode}: ${input} -> ${output}`);
