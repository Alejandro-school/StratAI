import { useCallback, useEffect, useRef, useState } from 'react';
import { STEAM_ENTRY_MS, SOFT_ENTRY_MS } from './sceneTransitions';

const navigateNormally = href => window.location.assign(href);
export function useSteamEntry({ href, canAnimate, reducedMotion, renderRequest, onStart, navigate = navigateNormally }) {
  const [phase, setPhase] = useState('idle');
  const transition = useRef({ active:false }), timer = useRef(), trigger = useRef();
  const status = useRef();
  const cancel = useCallback(() => {
    window.clearTimeout(timer.current); transition.current.active = false;
    setPhase('idle'); renderRequest.current();
  }, [renderRequest]);
  useEffect(() => {
    const restore = event => { if (event.persisted) cancel(); };
    window.addEventListener('pageshow', restore);
    return () => { window.removeEventListener('pageshow', restore); window.clearTimeout(timer.current); transition.current.active = false; };
  }, [cancel]);
  useEffect(() => {
    if (phase === 'moving') status.current?.focus({ preventScroll:true });
    if (phase === 'idle' && trigger.current) { trigger.current.focus({ preventScroll:true }); trigger.current = undefined; }
  }, [phase]);
  const onLogin = useCallback(event => {
    if (event.defaultPrevented || event.button > 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || event.currentTarget.target === '_blank') return;
    event.preventDefault();
    if (transition.current.active) return;
    const animate = canAnimate && !reducedMotion;
    trigger.current = event.currentTarget;
    transition.current = { active:true, animate, fresh:true, startedAt:performance.now(), duration:animate ? STEAM_ENTRY_MS : SOFT_ENTRY_MS };
    onStart(); setPhase('moving'); renderRequest.current();
    timer.current = window.setTimeout(() => {
      setPhase('handoff');
      // Keep an ordinary link available if this browser prevents navigation.
      try { navigate(href); } catch { /* The handoff view retains direct access. */ }
    }, transition.current.duration);
  }, [canAnimate, reducedMotion, href, navigate, onStart, renderRequest]);
  return { entering:phase !== 'idle', handoff:phase === 'handoff', transition, status, onLogin, cancel };
}
