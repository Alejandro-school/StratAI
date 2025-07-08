import React from 'react';
import useFadeInOnScroll from '../../hooks/useFadeInOnScroll';
import styles from './About.module.css';

const About = () => {
  const ref = useFadeInOnScroll();
  return (
    <section ref={ref} className="fade-section">
      <div className="container">
        <h2>\u00bfQu\u00e9 es StratAI?</h2>
        <p className={styles.text}>StratAI es una plataforma de an\u00e1lisis de partidas para CS2 que utiliza inteligencia artificial para ofrecer recomendaciones reales.</p>
      </div>
    </section>
  );
};

export default About;
