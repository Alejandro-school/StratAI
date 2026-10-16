import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const load = name => {
  const buffer = readFileSync(resolve(process.cwd(), 'public/landing-world', name));
  const length = buffer.readUInt32LE(12);
  return { buffer, json: JSON.parse(buffer.subarray(20, 20 + length)), binary: buffer.subarray(28 + length) };
};
const triangles = json => json.meshes.flatMap(mesh => mesh.primitives).reduce((sum, primitive) => sum + json.accessors[primitive.indices].count / 3, 0);

describe('low-PC map asset budget', () => {
  it('retains native textured geometry with substantially fewer triangles and bounded scene complexity', () => {
    const original = load('mirage-textured.glb').json;
    const optimized = load('mirage-lowpc.glb').json;
    expect(triangles(optimized)).toBeGreaterThan(500000);
    expect(triangles(optimized)).toBeLessThan(triangles(original) * .6);
    expect(optimized.meshes.flatMap(mesh => mesh.primitives).length).toBeLessThanOrEqual(500);
    expect(optimized.extensionsRequired).toContain('KHR_texture_basisu');
    expect(optimized.extensionsRequired).not.toContain('EXT_texture_webp');
    for (const primitive of optimized.meshes.flatMap(mesh => mesh.primitives)) {
      expect(primitive.material).toBeDefined();
      const material = optimized.materials[primitive.material];
      const textured = material.pbrMetallicRoughness?.baseColorTexture || material.pbrMetallicRoughness?.metallicRoughnessTexture || material.normalTexture || material.occlusionTexture || material.emissiveTexture;
      // Constant backing/glass materials legitimately discard unused UVs.
      if (textured) expect(primitive.attributes.TEXCOORD_0).toBeDefined();
    }
  });
  it('ships GPU-compressed images with mipmaps and caps network size', () => {
    const { buffer, json, binary } = load('mirage-lowpc.glb');
    expect(buffer.byteLength).toBeLessThan(33e6);
    expect(json.images.length).toBeGreaterThan(400);
    for (const image of json.images) {
      expect(image.mimeType).toBe('image/ktx2');
      const offset = json.bufferViews[image.bufferView].byteOffset || 0;
      expect(binary.readUInt32LE(offset + 20)).toBeLessThanOrEqual(512);
      expect(binary.readUInt32LE(offset + 24)).toBeLessThanOrEqual(512);
      expect(binary.readUInt32LE(offset + 40)).toBeGreaterThan(1);
    }
  });
});
