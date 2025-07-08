import React from 'react';
import useFadeInOnScroll from '../../hooks/useFadeInOnScroll';

const CTA = () => {
  const ref = useFadeInOnScroll();
  return (
    <section ref={ref} className="fade-section">
      <div className="container" style={{ textAlign: 'center' }}>
        <h2>Empieza ahora, mejora desde tu pr\u00f3xima partida.</h2>
        <a href="/auth/steam" className="btn-primary">Iniciar ahora \u2013 Gratis con Steam</a>
      </div>
    </section>
  );
};

export default CTA;
