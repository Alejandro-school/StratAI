import { Vector3 } from 'three';
import { AGENTS } from './sceneLayout';

export const STEAM_ENTRY_MS = 1100;
export const SOFT_ENTRY_MS = 180;
export const TOOL_EXIT_MS = 120;
export const TOOL_CAMERA_MS = 560;
const entryPosition = new Vector3(-1, 17, 31);
const entryTarget = new Vector3(...AGENTS.ct.position).add(new Vector3(0, .75, 0));
export const transitionEase = value => 1 - Math.pow(1 - Math.max(0, Math.min(1, value)), 3);

export function writeEntryPose(progress, fromPosition, fromTarget, position, target) {
  const t = transitionEase(progress), arc = Math.sin(Math.PI * t);
  position.lerpVectors(fromPosition, entryPosition, t);
  position.x -= arc * 1.2;
  position.y += arc * 1.4;
  target.lerpVectors(fromTarget, entryTarget, t);
}

export function writeToolPose(progress, fromPosition, fromTarget, toPosition, toTarget, position, target, right, direction) {
  const t = transitionEase(progress), arc = Math.sin(Math.PI * t);
  position.lerpVectors(fromPosition, toPosition, t).addScaledVector(right, arc * .85 * direction);
  position.y += arc * .35;
  target.lerpVectors(fromTarget, toTarget, t);
}
