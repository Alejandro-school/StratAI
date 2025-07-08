import React from 'react';
import useFadeInOnScroll from '../../hooks/useFadeInOnScroll';
import useParallax from '../../hooks/useParallax';
import styles from './Hero.module.css';

const Hero = () => {
  const fadeRef = useFadeInOnScroll();
  const parallaxRef = useParallax(0.2);
  return (
    <section ref={fadeRef} className={`hero fade-section ${styles.heroSection}`}>
      <div ref={parallaxRef} className="hero-content parallax">
        <h1>Mejora tu rendimiento. Entiende tus errores. Gana m\u00e1s partidas.</h1>
        <p>StratAI analiza tus partidas de CS2 y te ofrece consejos personalizados con IA.</p>
        <a href="/auth/steam" className="btn-primary">Inicia sesi\u00f3n con Steam</a>
      </div>
    </section>
  );
};

export default Hero;
