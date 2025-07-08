import React from 'react';
import useFadeInOnScroll from '../../hooks/useFadeInOnScroll';
import useParallax from '../../hooks/useParallax';
const CTA = () => {
  const fadeRef = useFadeInOnScroll();
  const parallaxRef = useParallax(0.05);
  return (
    <section ref={fadeRef} className="fade-section">
      <div ref={parallaxRef} className="container parallax" style={{ textAlign: 'center' }}>

        <h2>Empieza ahora, mejora desde tu pr\u00f3xima partida.</h2>
        <a href="/auth/steam" className="btn-primary">Iniciar ahora \u2013 Gratis con Steam</a>
      </div>
    </section>
  );
};

export default CTA;
