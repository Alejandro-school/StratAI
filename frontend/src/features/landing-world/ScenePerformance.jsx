import { useEffect, useRef } from 'react';
import { useFrame, useThree } from '@react-three/fiber';

const query = new URLSearchParams(window.location.search);
export const MEASURE_SCENE = query.get('perf') === '1';
export const BASELINE_SCENE = MEASURE_SCENE && query.get('benchmark') === 'baseline';

function textureBytes(texture) {
  if (texture.isCompressedTexture) return texture.mipmaps.reduce((sum, mip) => sum + mip.data.byteLength, 0);
  const image = texture.image;
  return (image?.width || 0) * (image?.height || 0) * 4 * (texture.generateMipmaps ? 4 / 3 : 1);
}

export function sceneResources(scene) {
  const textures = new Map();
  const geometries = new Map();
  scene.traverse(object => {
    if (!object.isMesh) return;
    geometries.set(object.geometry.uuid, object.geometry);
    for (const material of Array.isArray(object.material) ? object.material : [object.material]) {
      for (const texture of Object.values(material)) if (texture?.isTexture) {
        const key = `${texture.source.uuid}/${texture.format}/${texture.colorSpace}/${texture.wrapS}/${texture.wrapT}`;
        textures.set(key, texture);
      }
    }
  });
  let geometryBytes = 0;
  for (const geometry of geometries.values()) {
    for (const attribute of Object.values(geometry.attributes)) geometryBytes += attribute.array.byteLength;
    geometryBytes += geometry.index?.array.byteLength || 0;
  }
  return { geometryMB: +(geometryBytes / 1e6).toFixed(2), textureMB: +([...textures.values()].reduce((sum, texture) => sum + textureBytes(texture), 0) / 1e6).toFixed(2), textures: textures.size };
}

// Explicit local diagnostic mode, kept out of the product UI. No telemetry.
export function ScenePerformance({ stage }) {
  const { gl, scene } = useThree();
  const samples = useRef({ frames: 0, times: [] });
  useFrame((_, delta) => {
    samples.current.frames++;
    if (delta < .25) samples.current.times.push(delta * 1000);
  });
  useEffect(() => {
    const context = gl.getContext();
    const debug = context.getExtension('WEBGL_debug_renderer_info');
    const renderer = debug ? context.getParameter(debug.UNMASKED_RENDERER_WEBGL) : 'WebGL';
    const timer = window.setInterval(() => {
      const data = samples.current;
      const times = data.times.sort((a, b) => a - b);
      if (stage.current) stage.current.dataset.performance = JSON.stringify({
        mode: BASELINE_SCENE ? 'baseline' : 'exploration', renderer, dpr: gl.getPixelRatio(),
        frames: data.frames, p50ms: +(times[Math.floor(times.length * .5)] || 0).toFixed(2),
        p95ms: +(times[Math.floor(times.length * .95)] || 0).toFixed(2),
        calls: gl.info.render.calls, triangles: gl.info.render.triangles, geometries: gl.info.memory.geometries,
        gpuTextures: gl.info.memory.textures, ...sceneResources(scene),
      });
      data.frames = 0; data.times = [];
    }, 1500);
    return () => window.clearInterval(timer);
  }, [gl, scene, stage]);
  return null;
}
