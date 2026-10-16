import vm from 'node:vm';
import { readFile, writeFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';

const original = await readFile('frontend/node_modules/three/examples/jsm/libs/basis/basis_transcoder.js', 'utf8');
const patched = await readFile('frontend/public/basis/basis_transcoder.js', 'utf8');
const wasmBinary = await readFile('frontend/public/basis/basis_transcoder.wasm');
async function initialize(code, strings) {
  const context = vm.createContext({ console, WebAssembly, TextDecoder, TextEncoder, setTimeout, clearTimeout }, { codeGeneration: { strings, wasm: true } });
  vm.runInContext(code, context);
  const module = await context.BASIS({ wasmBinary });
  module.initializeBasis();
  return module;
}
let blocked = false;
try { await initialize(original, false); } catch (error) { blocked = /code generation disallowed|EvalError/i.test(String(error)); }
if (!blocked) throw new Error('Control did not demonstrate strict-CSP failure');
const control = await initialize(original, true);
const secure = await initialize(patched, false);
const glb = await readFile(process.argv[2] || 'frontend/public/landing-world/mirage-lowpc.glb');
const length = glb.readUInt32LE(12), json = JSON.parse(glb.subarray(20, 20 + length));
const binary = glb.subarray(28 + length);
let images = 0, levels = 0, bytes = 0;
const hash = createHash('sha256');
for (const image of json.images) {
  const view = json.bufferViews[image.bufferView];
  const data = new Uint8Array(binary.subarray(view.byteOffset || 0, (view.byteOffset || 0) + view.byteLength));
  const files = [control, secure].map(module => new module.KTX2File(data));
  if (files.some(file => !file.isValid() || !file.startTranscoding())) throw new Error(`Invalid KTX: ${image.name}`);
  for (const format of [2, 3, 6]) for (let level = 0; level < files[0].getLevels(); level++) {
    const results = files.map(file => {
      const buffer = new Uint8Array(file.getImageTranscodedSizeInBytes(level, 0, 0, format));
      if (!file.transcodeImage(buffer, level, 0, 0, format, 0, -1, -1)) throw new Error('Transcode failed');
      return Buffer.from(buffer);
    });
    if (!results[0].equals(results[1])) throw new Error(`CSP glue changed pixels: ${image.name}/${level}/${format}`);
    hash.update(results[1]); levels++; bytes += results[1].byteLength;
  }
  for (const file of files) { file.close(); file.delete(); }
  images++;
}
const report = { strictCspControlBlocked: blocked, patchedStringCodeGeneration: false, images, mipFormatComparisons: levels, bytesCompared: bytes, identical: true, decodedSha256: hash.digest('hex') };
await writeFile(process.argv[3] || 'output/landing-world-optimization-20261015/basis-validation.json', JSON.stringify(report, null, 2));
console.log(JSON.stringify(report));
