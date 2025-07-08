import React from 'react';
import useFadeInOnScroll from '../../hooks/useFadeInOnScroll';
import styles from './Hero.module.css';

const Hero = () => {
  const ref = useFadeInOnScroll();
  return (
    <section ref={ref} className={`hero fade-section ${styles.heroSection}`}>
      <div className="hero-content">
        <h1>Mejora tu rendimiento. Entiende tus errores.</h1>
        <p>StratAI analiza tus partidas de CS2 y te ofrece consejos personalizados con IA.</p>
        <a href="/auth/steam" className="btn-primary">Inicia sesi\u00f3n con Steam</a>
      </div>
    </section>
  );
};

export default Hero;
