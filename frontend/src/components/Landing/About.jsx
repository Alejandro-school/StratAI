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
        <h2>\u00bfQu\u00e9 es StratAI?</h2>
        <p className={styles.text}>StratAI es una plataforma de an\u00e1lisis de partidas para CS2 que utiliza inteligencia artificial para ofrecer recomendaciones reales.</p>
      </div>
    </section>
  );
};

export default About;
