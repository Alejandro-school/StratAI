/**
 * Hook que fija (“pin”) cada hijo del contenedor y hace que el
 * siguiente panel deslice / empuje al anterior (estilo GSAP Field™).
 * Por defecto sólo se aplica en desktop (≥ 768 px).
 */
import { useLayoutEffect } from 'react';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';

export const useGsapSectionPin = (
  containerRef,
  { end = '+=100%', scrub = true } = {}
) => {
  useLayoutEffect(() => {
    if (!containerRef.current) return;
    gsap.registerPlugin(ScrollTrigger);

    const ctx = gsap.context(() => {
      ScrollTrigger.matchMedia({
        '(min-width: 768px)': () => {
          gsap.utils
            .toArray(containerRef.current.children)
            .forEach((panel) => {
              ScrollTrigger.create({
                trigger: panel,
                start: 'top top',
                end,               // cuánto dura el pin (viewport por defecto)
                pin: true,
                pinSpacing: false, // el siguiente panel “empuja”
                scrub,             // true = follow scroll; number = delay
              });
            });
        },
      });
    }, containerRef);

    return () => ctx.revert();
  }, [containerRef, end, scrub]);
};
