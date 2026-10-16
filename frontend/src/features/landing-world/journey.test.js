import { describe, expect, it } from 'vitest';
import { sampleJourney, sourceToWorld, getFeature, FEATURES } from './journey';
import { PerspectiveCamera, Vector3 } from 'three';
import { FRAMING_LANDMARKS, UTILITY_ARC_POINTS, utilityPanelOcclusion } from './sceneLayout';

describe('landing world camera journey', () => {
  it('descends from an aerial position into A and keeps a valid look direction', () => {
    const start = sampleJourney(0);
    const end = sampleJourney(1);
    expect(start.position[1]).toBeGreaterThan(90);
    expect(end.position[1]).toBeLessThan(start.position[1] / 4);
    expect(end.position[1]).toBeGreaterThan(end.target[1]);
    for (let i = 0; i <= 200; i += 1) {
      const pose = sampleJourney(i / 200);
      expect([...pose.position, ...pose.target].every(Number.isFinite)).toBe(true);
      expect(Math.hypot(...pose.position.map((value, axis) => value - pose.target[axis]))).toBeGreaterThan(1);
      if (i) {
        const previous = sampleJourney((i - 1) / 200);
        expect(Math.hypot(...pose.position.map((value, axis) => value - previous.position[axis]))).toBeLessThan(4);
      }
    }
  });

  it('includes A site, stairs and CT boxes in the wider final frame', () => {
    const end = sampleJourney(1);
    for (const [width, height] of [[1440, 900], [390, 844], [320, 740]]) {
      const camera = new PerspectiveCamera(width < height ? 80 : 55, width / height, .15, 600);
      camera.position.fromArray(end.position);
      camera.lookAt(new Vector3(...end.target));
      camera.updateMatrixWorld();
      for (const landmark of Object.values(FRAMING_LANDMARKS)) {
        const point = new Vector3(...landmark).project(camera);
        expect(Math.abs(point.x)).toBeLessThan(.98);
        expect(Math.abs(point.y)).toBeLessThan(.78);
        expect(point.z).toBeGreaterThan(-1);
        expect(point.z).toBeLessThan(1);
      }
    }
  });

  it.each([[800, 600], [1024, 768], [1440, 900], [2560, 1221]])('keeps the utility arc and complete smoke volume beside the panel at %i × %i', (width, height) => {
    const feature = getFeature('utility');
    const camera = new PerspectiveCamera(55, width / height, .15, 600);
    camera.position.fromArray(feature.camera); camera.lookAt(new Vector3(...feature.target));
    const occlusion = utilityPanelOcclusion(width);
    camera.setViewOffset(width, height, occlusion / 2, 0, width, height); camera.updateMatrixWorld();
    const bounds = [...UTILITY_ARC_POINTS];
    for (const x of [5.2, 10.8]) for (const y of [-.6, 3.5]) for (const z of [22.2, 27.8]) bounds.push([x, y, z]);
    for (const point of bounds) {
      const projected = new Vector3(...point).project(camera);
      const x = (projected.x * .5 + .5) * width;
      const y = (-projected.y * .5 + .5) * height;
      expect(x).toBeGreaterThan(20);
      expect(x).toBeLessThan(width - occlusion - 20);
      expect(y).toBeGreaterThan(90);
      expect(y).toBeLessThan(height - 140);
    }
  });

  it('clamps overscroll and offers discrete poses for reduced motion', () => {
    expect(sampleJourney(-10)).toEqual(sampleJourney(0));
    expect(sampleJourney(10)).toEqual(sampleJourney(1));
    expect(sampleJourney(.11, true)).toEqual(sampleJourney(.12, true));
    expect(sampleJourney(1, true)).toEqual(sampleJourney(1));
  });

  it('has arrived when exploration begins and holds the interior pose through the remaining scroll', () => {
    expect(sampleJourney(.82)).toEqual(sampleJourney(1));
    expect(sampleJourney(.9, true)).toEqual(sampleJourney(1, true));
  });

  it('preserves the documented Source-to-model transform for spatial anchors', () => {
    expect(sourceToWorld([0, 0, 0])).toEqual([13.4, 3.2, -16.94]);
    const position = sourceToWorld([-462.5, -1575, 13.28125]);
    expect(position[0]).toBeCloseTo(4.15);
    expect(position[1]).toBeCloseTo(3.465625);
    expect(position[2]).toBeCloseTo(14.56);
  });

  it('keeps all product destinations reachable and labels prospective coaching', () => {
    expect(new Set(FEATURES.map(feature => feature.id)).size).toBe(5);
    for (const feature of FEATURES) {
      expect(getFeature(feature.id)).toBe(feature);
      expect([...feature.position, ...feature.camera, ...feature.target].every(Number.isFinite)).toBe(true);
    }
    expect(getFeature('coach').status).toBe('En desarrollo');
    expect(getFeature('missing')).toBeNull();
  });
});
