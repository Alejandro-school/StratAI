import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('A detail and shared distant LOD', () => {
  it('preserves original A triangles and provides distinct geometry sharing native GPU textures', () => {
    const buffer=readFileSync(resolve(process.cwd(),'public/landing-world/mirage-exploration.glb'));
    const size=buffer.readUInt32LE(12), json=JSON.parse(buffer.subarray(20,20+size));
    const count=name => json.meshes.filter(m=>m.name===name).flatMap(m=>m.primitives).reduce((n,p)=>n+json.accessors[p.indices].count/3,0);
    expect(count('site_high')).toBe(277445);
    expect(count('site_distant')).toBeLessThan(count('site_high')*.4);
    expect(count('background')).toBeLessThan(400000);
    expect(json.scenes[json.scene].nodes.length).toBe(3);
    expect(json.meshes.length).toBe(3);
    expect(json.images.every(image=>image.mimeType==='image/ktx2')).toBe(true);
    // Higher-resolution native A and lossless geometry have a separate budget
    // from the previous globally simplified map; the tradeoff is documented.
    expect(buffer.byteLength).toBeLessThan(75e6);
    const binary=buffer.subarray(28+size),dimensions=json.images.map(image=>binary.readUInt32LE((json.bufferViews[image.bufferView].byteOffset||0)+20));
    expect(Math.max(...dimensions)).toBe(1024);
  });
});
