import { useEffect, useRef } from 'react';
import { useFrame, useThree } from '@react-three/fiber';

// Long demand-loop pauses are idle time, not animation time.
export const animationDelta = delta => Math.min(Math.max(delta, 0), .05);
export const playbackDelta = (delta, wasPlaying) => wasPlaying ? Math.min(Math.max(delta, 0), .25) : 0;
export const nextPixelRatio = ratio => ratio > 1 ? 1 : Math.max(.8, ratio - .2);

export function SceneRendering({ renderRequest, activity, adaptive = true }) {
  const { invalidate, gl, setDpr } = useThree();
  const timing = useRef({ count: 0, seconds: 0, active: false });
  const resolution = useRef({ maximum: gl.getPixelRatio(), moving: gl.getPixelRatio() });
  useEffect(() => {
    renderRequest.current = invalidate;
    invalidate();
    return () => { renderRequest.current = () => {}; };
  }, [invalidate, renderRequest]);
  useFrame((_, delta) => {
    const sample = timing.current;
    const active = activity.current.camera || activity.current.playback;
    const continuous = active && sample.active;
    const changed = active !== sample.active;
    sample.active = active;
    if (adaptive && changed) {
      const ratio = active ? resolution.current.moving : resolution.current.maximum;
      if (gl.getPixelRatio() !== ratio) { setDpr(ratio); invalidate(); }
    }
    if (!adaptive || !continuous || delta > 1) { sample.count = 0; sample.seconds = 0; return; }
    sample.count++; sample.seconds += delta;
    // Only sustained, active rendering changes resolution. Loading and idle
    // time cannot accidentally downgrade quality; the map itself stays intact.
    if (sample.count >= 30 && sample.seconds >= 2) {
      const ratio = gl.getPixelRatio();
      if (sample.seconds / sample.count > 1 / 30 && ratio > .8) {
        resolution.current.moving = nextPixelRatio(ratio);
        setDpr(resolution.current.moving); invalidate();
      }
      sample.count = 0; sample.seconds = 0;
    }
  });
  return null;
}
