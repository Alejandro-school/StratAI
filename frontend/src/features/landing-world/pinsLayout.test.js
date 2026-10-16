import { describe, expect, it } from 'vitest';
import { cameraControlsExclusion, layoutWorldPins } from './pinsLayout';
import { PerspectiveCamera, Vector3 } from 'three';
import { OVERVIEW, AGENTS } from './sceneLayout';
import { FEATURES } from './journey';

const clustered = ['replay', 'utility', 'stats', 'coach', 'challenges'].map(id => ({ id, x: 430, y: 300, worldX: 430, worldY: 300 }));

describe('world callout readability', () => {
  it.each([[2560,1221],[1440,900],[800,600],[390,844],[320,740]])('keeps extreme callouts outside camera guidance at %i × %i', (width,height) => {
    const controls = cameraControlsExclusion(width,height);
    const result = layoutWorldPins(clustered.map(point => ({...point,x:-500,y:height+500})),width,height);
    const halfWidth = width <= 760 ? Math.min(166,width/2-16)/2 : 85;
    for (const point of result) expect(point.x-halfWidth >= controls.right || point.y+23 <= controls.top).toBe(true);
  });
  it.each([[1440, 900], [1024, 768], [800, 600], [390, 844], [320, 740]])('keeps five clustered destinations separate at %i × %i', (width, height) => {
    const result = layoutWorldPins(clustered, width, height);
    const halfWidth = width <= 760 ? Math.min(166, width / 2 - 16) / 2 : 85;
    for (const [index, point] of result.entries()) {
      expect(point.x - halfWidth).toBeGreaterThanOrEqual(0);
      expect(point.x + halfWidth).toBeLessThanOrEqual(width);
      expect(point.y - 23).toBeGreaterThan(76);
      expect(point.y + 23).toBeLessThan(height - 120);
      expect(point.worldX).toBe(430);
      for (const other of result.slice(index + 1)) {
        expect(Math.abs(point.x - other.x) >= halfWidth * 2 || Math.abs(point.y - other.y) >= 46).toBe(true);
      }
    }
  });

  it.each([[2560, 1221], [1440, 900], [390, 844], [320, 740]])('leaves both naturally placed agents visible at %i × %i', (width, height) => {
    const camera = new PerspectiveCamera(width < height ? 80 : 55, width / height, .15, 600);
    camera.position.fromArray(OVERVIEW.camera);
    camera.lookAt(new Vector3(...OVERVIEW.target)); camera.updateMatrixWorld();
    const points = FEATURES.map(feature => {
      const point = new Vector3(...feature.position).project(camera);
      return { id: feature.id, x: (point.x * .5 + .5) * width, y: (-point.y * .5 + .5) * height };
    });
    const pins = layoutWorldPins(points, width, height);
    const halfWidth = width <= 760 ? Math.min(166, width / 2 - 16) / 2 : 85;
    for (const agent of Object.values(AGENTS)) {
      const projected = [0, agent.height].map(y => new Vector3(...agent.position).add(new Vector3(0, y, 0)).project(camera));
      const agentX = (projected[0].x * .5 + .5) * width;
      const top = (-projected[1].y * .5 + .5) * height;
      const bottom = (-projected[0].y * .5 + .5) * height;
      for (const pin of pins) expect(pin.x + halfWidth < agentX - 10 || pin.x - halfWidth > agentX + 10 || pin.y + 23 < top || pin.y - 23 > bottom).toBe(true);
    }
  });
});
