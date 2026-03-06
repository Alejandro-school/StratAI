/**
 * useGSAP — Register ScrollTrigger + Lenis integration helper
 */
import { useEffect, useRef } from 'react';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';

gsap.registerPlugin(ScrollTrigger);

/**
 * Syncs Lenis scroll with GSAP ScrollTrigger.
 * Call once at the top-level component where Lenis is initialized.
 */
export const syncLenisWithScrollTrigger = (lenis) => {
  if (!lenis) return;
  lenis.on('scroll', ScrollTrigger.update);
  gsap.ticker.add((time) => lenis.raf(time * 1000));
  gsap.ticker.lagSmoothing(0);
};

/**
 * useGSAPContext — scoped GSAP context with auto-cleanup
 * @param {Function} callback - receives (ctx, gsapRef) with the container element
 * @param {Array} deps - dependency array
 */
export const useGSAPContext = (callback, deps = []) => {
  const containerRef = useRef(null);

  useEffect(() => {
    if (!containerRef.current) return;

    const ctx = gsap.context(() => {
      callback(containerRef.current);
    }, containerRef);

    return () => ctx.revert();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  return containerRef;
};

export { gsap, ScrollTrigger };
export default useGSAPContext;
