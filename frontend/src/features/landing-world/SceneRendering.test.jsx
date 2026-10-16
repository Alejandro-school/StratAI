import React from 'react';
import { render, cleanup, fireEvent, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { animationDelta, playbackDelta, nextPixelRatio, SceneRendering } from './SceneRendering';
import { ProductPanel } from './ProductPanel';
import { FEATURES } from './journey';

afterEach(cleanup);
const fiber = vi.hoisted(() => ({ frame: null, invalidate: vi.fn(), setDpr: vi.fn(), gl: { getPixelRatio: () => 1.25 } }));
vi.mock('@react-three/fiber', () => ({ useFrame: callback => { fiber.frame = callback; }, useThree: () => fiber }));

describe('demand rendering', () => {
  it('excludes idle time from the first playback frame', () => {
    expect(animationDelta(60)).toBe(.05);
    expect(animationDelta(1 / 60)).toBe(1 / 60);
    expect(animationDelta(-1)).toBe(0);
    expect(playbackDelta(60, false)).toBe(0);
    expect(playbackDelta(1 / 10, true)).toBe(1 / 10);
    expect(playbackDelta(60, true)).toBe(.25);
  });
  it('reduces resolution in bounded steps while preserving a readable floor', () => {
    expect(nextPixelRatio(1.25)).toBe(1);
    expect(nextPixelRatio(1)).toBe(.8);
    expect(nextPixelRatio(.8)).toBe(.8);
  });
  it('adapts only during sustained animation and releases its render callback', () => {
    fiber.setDpr.mockClear();
    let ratio = 1.25;
    fiber.gl.getPixelRatio = () => ratio;
    fiber.setDpr.mockImplementation(value => { ratio = value; });
    const activity = { current: { camera: false, playback: false } };
    const renderRequest = { current: null };
    const view = render(<SceneRendering renderRequest={renderRequest} activity={activity} />);
    expect(renderRequest.current).toBe(fiber.invalidate);
    for (let i = 0; i < 60; i++) fiber.frame({}, .2);
    expect(fiber.setDpr).not.toHaveBeenCalled();
    activity.current.camera = true;
    fiber.frame({}, 60); // Wake after a long idle period.
    for (let i = 0; i < 45; i++) fiber.frame({}, .05);
    expect(fiber.setDpr).toHaveBeenCalledWith(1);
    expect(ratio).toBe(1);
    activity.current.camera = false;
    fiber.frame({}, .016);
    expect(ratio).toBe(1.25); // Restore a sharp still image, then sleep.
    activity.current.camera = true;
    fiber.frame({}, 60);
    expect(ratio).toBe(1); // Reuse the measured moving resolution.
    view.unmount();
    expect(renderRequest.current).not.toBe(fiber.invalidate);
  });
  it('wakes a sleeping canvas on play and pause, and restarts completed playback', () => {
    const playback = { current: { playing: false, progress: 1 } };
    const renderRequest = { current: vi.fn() };
    render(<ProductPanel panel={FEATURES.find(feature => feature.id === 'utility')} playback={playback} renderRequest={renderRequest} onSelect={() => {}} onClose={() => {}} />);
    fireEvent.click(screen.getByRole('button', { name: 'Ver trayectoria en Mirage' }));
    expect(playback.current).toEqual({ playing: true, progress: 0 });
    expect(renderRequest.current).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole('button', { name: 'Pausar trayectoria en Mirage' }));
    expect(playback.current.playing).toBe(false);
    expect(renderRequest.current).toHaveBeenCalledTimes(2);
  });
});
