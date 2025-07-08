import React from 'react';
import useFadeInOnScroll from '../../hooks/useFadeInOnScroll';
import useParallax from '../../hooks/useParallax';
import styles from './About.module.css';

const About = () => {
  const fadeRef = useFadeInOnScroll();
  const parallaxRef = useParallax(0.15);

  return (
    <section ref={fadeRef} className="fade-section">
      <div ref={parallaxRef} className="container parallax">
        <h2>¿Qué es StratAI?</h2>
        <p className={styles.text}>
          StratAI es una plataforma de análisis de partidas para CS2 que utiliza inteligencia artificial para ofrecer recomendaciones reales.
        </p>
      </div>
    </section>
  );
};

export default About;
