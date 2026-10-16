import { act, cleanup, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useSteamEntry } from './useSteamEntry';

beforeEach(() => vi.useFakeTimers());
afterEach(() => { cleanup(); vi.useRealTimers(); });
const href = 'https://api.example.test/auth/steam/login';
function setup(overrides={}) {
  const navigate=vi.fn(), renderRequest={current:vi.fn()}, onStart=vi.fn();
  const hook=renderHook(()=>useSteamEntry({href,canAnimate:true,reducedMotion:false,renderRequest,onStart,navigate,...overrides}));
  const event={button:0,currentTarget:{focus:vi.fn()},preventDefault:vi.fn()};
  return {...hook,navigate,event,onStart};
}
describe('Steam entry intent and navigation',()=>{
  it('acknowledges immediately and hands off once to the unchanged URL after the authored motion',()=>{
    const {result,event,navigate,onStart}=setup();
    act(()=>{result.current.onLogin(event);result.current.onLogin(event);});
    expect(result.current.entering).toBe(true);expect(onStart).toHaveBeenCalledTimes(1);expect(result.current.transition.current.animate).toBe(true);
    act(()=>vi.advanceTimersByTime(1099));expect(navigate).not.toHaveBeenCalled();
    act(()=>vi.advanceTimersByTime(1));expect(navigate).toHaveBeenCalledExactlyOnceWith(href);expect(result.current.handoff).toBe(true);
  });
  it.each([{metaKey:true},{ctrlKey:true},{shiftKey:true},{button:1}])('preserves native modified-link activation %j',modifier=>{
    const {result,event,onStart}=setup();act(()=>result.current.onLogin({...event,...modifier}));
    expect(event.preventDefault).not.toHaveBeenCalled();expect(onStart).not.toHaveBeenCalled();expect(result.current.entering).toBe(false);
  });
  it.each([{reducedMotion:true},{canAnimate:false}])('uses a short nonspatial handoff for %j',options=>{
    const {result,event,navigate}=setup(options);act(()=>result.current.onLogin(event));
    expect(result.current.transition.current.animate).toBe(false);act(()=>vi.advanceTimersByTime(180));expect(navigate).toHaveBeenCalledExactlyOnceWith(href);
  });
  it('cancels pending navigation and restores focus, including returning with browser back',()=>{
    const {result,event,navigate}=setup();act(()=>result.current.onLogin(event));act(()=>result.current.cancel());
    act(()=>vi.advanceTimersByTime(2000));expect(navigate).not.toHaveBeenCalled();expect(event.currentTarget.focus).toHaveBeenCalled();
    act(()=>result.current.onLogin(event));act(()=>vi.advanceTimersByTime(1100));expect(navigate).toHaveBeenCalledTimes(1);
    const back = new Event('pageshow');Object.defineProperty(back,'persisted',{value:true});
    act(()=>window.dispatchEvent(back));expect(result.current.entering).toBe(false);expect(result.current.transition.current.active).toBe(false);
  });
  it('releases the pending redirect on unmount',()=>{
    const {result,event,navigate,unmount}=setup();act(()=>result.current.onLogin(event));unmount();act(()=>vi.advanceTimersByTime(2000));expect(navigate).not.toHaveBeenCalled();
  });
});
