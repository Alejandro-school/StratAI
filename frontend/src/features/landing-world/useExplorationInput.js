import { useEffect, useRef } from 'react';
import { constrainNavigation, panNavigation, rotateNavigation, zoomNavigation } from './cameraNavigation';

export function useExplorationInput({ stage, enabled, navigation, renderRequest, interaction }) {
  const contacts = useRef(new Map());
  useEffect(() => {
    const element = stage.current;
    if (!element || !enabled) return undefined;
    const points = contacts.current;
    const wake = () => renderRequest.current();
    const down = event => {
      if (![0, 2].includes(event.button)) return;
      if (event.target.closest('button,a,input,select,textarea')) return;
      points.set(event.pointerId, { x: event.clientX, y: event.clientY, button: event.button });
      interaction.current = true;
      element.setPointerCapture(event.pointerId); element.dataset.dragging = 'true';
      element.focus({ preventScroll: true });
      event.preventDefault(); wake();
    };
    const move = event => {
      const old = points.get(event.pointerId); if (!old) return;
      const next = { ...old, x: event.clientX, y: event.clientY };
      if (points.size === 2) {
        const other = [...points.entries()].find(([id]) => id !== event.pointerId)?.[1];
        if (other) {
          const before = Math.hypot(old.x-other.x, old.y-other.y), after = Math.hypot(next.x-other.x, next.y-other.y);
          if (before > 8 && after > 8) { navigation.current.zoom *= after/before; navigation.current.distance *= Math.pow(before/after,.25); constrainNavigation(navigation.current); }
          panNavigation(navigation.current, (next.x-old.x)/2, (next.y-old.y)/2);
        }
      } else if (old.button === 2 || event.shiftKey) rotateNavigation(navigation.current, next.x-old.x, next.y-old.y);
      else panNavigation(navigation.current, next.x-old.x, next.y-old.y);
      points.set(event.pointerId, next); event.preventDefault(); wake();
    };
    const up = event => {
      points.delete(event.pointerId);
      if (element.hasPointerCapture(event.pointerId)) element.releasePointerCapture(event.pointerId);
      interaction.current = points.size > 0;
      element.dataset.dragging = String(interaction.current); wake();
    };
    const cancel = () => {
      for (const id of points.keys()) if (element.hasPointerCapture(id)) element.releasePointerCapture(id);
      points.clear(); interaction.current = false; element.dataset.dragging = 'false'; wake();
    };
    const wheel = event => {
      event.preventDefault();
      zoomNavigation(navigation.current, event.deltaY * (event.deltaMode === 1 ? 16 : event.deltaMode === 2 ? element.clientHeight : 1)); wake();
    };
    const context = event => event.preventDefault();
    element.addEventListener('pointerdown', down);
    element.addEventListener('pointermove', move);
    for (const name of ['pointerup','pointercancel','lostpointercapture']) element.addEventListener(name, up);
    element.addEventListener('wheel', wheel, { passive: false }); element.addEventListener('contextmenu', context);
    window.addEventListener('blur', cancel);
    return () => {
      element.removeEventListener('pointerdown', down); element.removeEventListener('pointermove', move);
      for (const name of ['pointerup','pointercancel','lostpointercapture']) element.removeEventListener(name, up);
      element.removeEventListener('wheel', wheel); element.removeEventListener('contextmenu', context); window.removeEventListener('blur', cancel);
      cancel();
    };
  }, [stage, enabled, navigation, renderRequest, interaction]);
}
