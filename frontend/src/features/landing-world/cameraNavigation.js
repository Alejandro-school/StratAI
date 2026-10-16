import { OVERVIEW } from './sceneLayout.js';

const clamp = (v, min, max) => Math.max(min, Math.min(max, Number.isFinite(v) ? v : min));
const dx = OVERVIEW.camera[0] - OVERVIEW.target[0], dz = OVERVIEW.camera[2] - OVERVIEW.target[2];
const originYaw = Math.atan2(dx, dz);
const originElevation = Math.atan2(OVERVIEW.camera[1], Math.hypot(dx, dz));
export const CAMERA_LIMITS = { x: [0, 4.8], z: [22.5, 26.8], distance: [19, 26], zoom: [.8, 2.1], elevation: [originElevation - .05, originElevation + .08], yaw: [originYaw - .2, originYaw + .06] };
export const CAMERA_MIN_HEIGHT = 17;
export function createNavigation() {
  return { x: OVERVIEW.target[0], z: OVERVIEW.target[2], distance: Math.hypot(dx, OVERVIEW.camera[1], dz), zoom: 1, yaw: originYaw, elevation: Math.atan2(OVERVIEW.camera[1], Math.hypot(dx, dz)) };
}
export function constrainNavigation(view) {
  for (const key of Object.keys(CAMERA_LIMITS)) view[key] = clamp(view[key], ...CAMERA_LIMITS[key]);
  // These degrees of freedom share one envelope. Combining all their individual
  // extremes would look past the Palace shell even though each focus is in A.
  const offsets = [view.x - OVERVIEW.target[0], view.z - OVERVIEW.target[2], view.yaw - originYaw, view.elevation - originElevation];
  const radii = [offsets[0] < 0 ? 3 : 1.8, offsets[1] < 0 ? 3.5 : .8, offsets[2] < 0 ? .2 : .06, offsets[3] < 0 ? .05 : .08];
  const spareDistance = Math.max(0, view.distance - CAMERA_MIN_HEIGHT / Math.sin(view.elevation));
  const scale = 1 / Math.max(1, Math.hypot(...offsets.map((value, index) => value / radii[index]), spareDistance / 2.5));
  view.x = OVERVIEW.target[0] + offsets[0] * scale;
  view.z = OVERVIEW.target[2] + offsets[1] * scale;
  view.yaw = originYaw + offsets[2] * scale;
  view.elevation = originElevation + offsets[3] * scale;
  view.distance = CAMERA_MIN_HEIGHT / Math.sin(view.elevation) + spareDistance * scale;
  return view;
}
export function panNavigation(view, dx, dy) {
  const scale = view.distance * .0016 / view.zoom;
  view.x += (-dx * Math.cos(view.yaw) - dy * Math.sin(view.yaw)) * scale;
  view.z += (dx * Math.sin(view.yaw) - dy * Math.cos(view.yaw)) * scale;
  return constrainNavigation(view);
}
export function rotateNavigation(view, dx, dy) {
  view.yaw -= dx * .002; view.elevation += dy * .0015;
  return constrainNavigation(view);
}
export function zoomNavigation(view, delta) {
  const amount = Math.max(-300, Math.min(300, delta));
  view.zoom *= Math.exp(-amount * .0015);
  view.distance *= Math.exp(amount * .0004);
  return constrainNavigation(view);
}
export function writeExplorationPose(view, position, target, pointer = { x: 0, y: 0 }) {
  const yaw = view.yaw + pointer.x * .025, elevation = view.elevation + pointer.y * .012;
  const horizontal = view.distance * Math.cos(elevation);
  target.set(view.x, 0, view.z);
  position.set(view.x + Math.sin(yaw) * horizontal, Math.sin(elevation) * view.distance, view.z + Math.cos(yaw) * horizontal);
}
export const flightEase = t => t < .5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2;
