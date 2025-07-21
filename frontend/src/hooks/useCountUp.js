// src/hooks/useCountUp.js
import { useRef, useLayoutEffect } from 'react';
import gsap from 'gsap';

export const useCountUp = (endValue = 0, { duration = 2, ease = 'power2.out' } = {}) => {
  const elRef = useRef(null);

  useLayoutEffect(() => {
    const obj = { val: 0 };
    const ctx = gsap.context(() => {
      gsap.to(obj, {
        val: endValue,
        duration,
        ease,
        onUpdate: () => {
          if (elRef.current) elRef.current.textContent = Math.floor(obj.val).toLocaleString();
        },
        scrollTrigger: { trigger: elRef.current, start: 'top 80%' },
      });
    }, elRef);

    return () => ctx.revert();
  }, [endValue, duration, ease]);

  return elRef;
};
