import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import useMatchProgress from './useMatchProgress';

class Stream {
  static instances = [];
  listeners = {};
  close = vi.fn();
  constructor() { Stream.instances.push(this); }
  addEventListener(name, listener) { this.listeners[name] = listener; }
  emit(name, data) { act(() => this.listeners[name]({ data: JSON.stringify(data) })); }
}

afterEach(() => { vi.unstubAllGlobals(); Stream.instances = []; });

function setup() {
  vi.stubGlobal('EventSource', Stream);
  const hook = renderHook(() => useMatchProgress('76561198000000000'));
  return { ...hook, stream: Stream.instances.at(-1) };
}

describe('Live demo progress', () => {
  it('keeps processing until every simultaneous match has finished', () => {
    const { result, stream } = setup();
    stream.emit('pipeline', { job_id: 'a', stage: 'analyzing' });
    stream.emit('pipeline', { job_id: 'b', stage: 'downloading' });
    stream.emit('pipeline', { job_id: 'a', stage: 'completed' });
    expect(result.current.isProcessing).toBe(true);
    stream.emit('pipeline', { job_id: 'b', stage: 'failed' });
    expect(result.current.isProcessing).toBe(false);
  });

  it('counts a completion once and refreshes completions received on reconnection', () => {
    const { result, stream } = setup();
    stream.emit('pipeline', { job_id: 'a', stage: 'completed' });
    stream.emit('pipeline', { job_id: 'a', stage: 'completed' });
    expect(result.current.completedCount).toBe(1);
    stream.emit('snapshot', { jobs: [{ job_id: 'a', stage: 'completed' }, { job_id: 'b', stage: 'completed' }] });
    expect(result.current.completedCount).toBe(2);
  });

  it('finishes discovery without leaving a permanent processing indicator', () => {
    const { result, stream } = setup();
    stream.emit('pipeline', { job_id: 'discovery-1', stage: 'discovery' });
    stream.emit('pipeline', { job_id: 'discovery-1', stage: 'discovery_completed' });
    expect(result.current.isProcessing).toBe(false);
  });

  it('closes the stream on unmount', () => {
    const { stream, unmount } = setup();
    unmount();
    expect(stream.close).toHaveBeenCalledOnce();
  });
});
