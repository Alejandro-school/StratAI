import { describe, expect, it } from 'vitest';
import { Vector3 } from 'three';
import { OVERVIEW } from './sceneLayout';
import { CAMERA_LIMITS, createNavigation, constrainNavigation, panNavigation, rotateNavigation, zoomNavigation, writeExplorationPose, flightEase } from './cameraNavigation';

describe('bounded A exploration', () => {
  it('starts at the approved overview and preserves the original camera height', () => {
    const position = new Vector3(), target = new Vector3();
    writeExplorationPose(createNavigation(), position, target);
    position.toArray().forEach((n,i) => expect(n).toBeCloseTo(OVERVIEW.camera[i], 8));
    expect(target.toArray()).toEqual(OVERVIEW.target);
  });
  it('constrains extreme, repeated inputs while preserving finite camera poses', () => {
    const view = createNavigation(), position = new Vector3(), target = new Vector3();
    for (let i=0;i<300;i++) {
      panNavigation(view, Math.sin(i)*1e5, Math.cos(i)*1e5); rotateNavigation(view, i%2 ? 1e5 : -1e5, i%3 ? 1e5 : -1e5); zoomNavigation(view, i%2 ? 1e5 : -1e5);
      for (const [key,[min,max]] of Object.entries(CAMERA_LIMITS)) { expect(view[key]).toBeGreaterThanOrEqual(min); expect(view[key]).toBeLessThanOrEqual(max); }
      writeExplorationPose(view,position,target); expect(position.y).toBeGreaterThanOrEqual(17-1e-8); expect(position.toArray().every(Number.isFinite)).toBe(true);
    }
    view.x=NaN; constrainNavigation(view); expect(Number.isFinite(view.x)).toBe(true);
  });
  it('allows a closer view and independent reset snapshots', () => {
    const view=createNavigation(), snapshot={...view}; zoomNavigation(view,-300); panNavigation(view,50,20);
    expect(view.zoom).toBeGreaterThan(snapshot.zoom); expect(createNavigation()).toEqual(snapshot);
    expect(flightEase(0)).toBe(0); expect(flightEase(1)).toBe(1); expect(flightEase(.5)).toBe(.5);
  });
  it('couples pan and orbit limits and does not drift when constrained twice', () => {
    const view = constrainNavigation({ ...createNavigation(), x: 6.38, z: 28.5, yaw: -.774, zoom: 2.1 });
    expect(view.x).toBeLessThan(4.7); expect(view.z).toBeLessThan(26.7);
    const snapshot = { ...view }; constrainNavigation(view);
    for (const key of Object.keys(snapshot)) expect(view[key]).toBeCloseTo(snapshot[key], 10);
    const edge = constrainNavigation({ ...createNavigation(), x: 100, z: 100, elevation: .8, distance: 29 });
    const saved = { ...edge }; constrainNavigation(edge);
    for (const key of Object.keys(saved)) expect(edge[key]).toBeCloseTo(saved[key], 10);
  });
});
