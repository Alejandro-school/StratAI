// src/hooks/useGsapFadeIn.js
import { useLayoutEffect } from 'react';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';

/**
 * Aplica un fade-in + desplazamiento a los hijos directos
 * de la ref que se le pase.
 *
 * @param {React.MutableRefObject} parentRef
 * @param {Object} opts            — Configurable en cada llamada
 *   y:        desplazamiento inicial en Y   (px)   · default 50
 *   duration: duración de la animación      (s)    · default 1
 *   ease:     curva de aceleración GSAP      · default 'power3.out'
 */
export const useGsapFadeIn = (
  parentRef,
  { y = 50, duration = 1, ease = 'power3.out' } = {}
) => {
  useLayoutEffect(() => {
    if (!parentRef.current) return;

    gsap.registerPlugin(ScrollTrigger);

    // Un context para limpiar al desmontar
    const ctx = gsap.context(() => {
      gsap.fromTo(
        parentRef.current.children,
        { opacity: 0, y },
        {
          opacity: 1,
          y: 0,
          duration,
          ease,
          stagger: 0.25,
          scrollTrigger: {
            trigger: parentRef.current,
            start: 'top 80%',
          },
        }
      );
    }, parentRef);

    return () => ctx.revert(); // limpieza
  }, [parentRef, y, duration, ease]);
};