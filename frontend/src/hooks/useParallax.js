import { useEffect, useRef } from 'react';

const useParallax = (speed = 0.3) => {
  const ref = useRef(null);

  useEffect(() => {
    const node = ref.current;
    if (!node) return;

    const handleScroll = () => {
      const offset = window.pageYOffset;
      node.style.transform = `translateY(${offset * speed}px)`;
    };

    handleScroll();
    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, [speed]);

  return ref;
};

export default useParallax;
