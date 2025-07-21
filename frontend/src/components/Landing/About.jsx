import React, { useRef } from 'react';
import { useGsapFadeIn } from '../../hooks/useGsapFadeIn';
import styles from '../../styles/Landing/About.module.css';

const About = () => {
  const sectionRef = useRef(null);
  useGsapFadeIn(sectionRef);

  return (
    <section id="about" ref={sectionRef} className={styles.aboutSection}>
      <div className={styles.contentWrapper}>
        <h2 className={styles.heading}>¿Qué es StratAI?</h2>
        <p className={styles.paragraph}>
          StratAI es tu entrenador personal de Counter-Strike 2 — analiza cada
          ronda, detecta errores de posicionamiento y te da consejos
          personalizados para mejorar.
        </p>
        <p className={styles.paragraph}>
          Nuestra plataforma te ofrece una ventaja competitiva al desgranar tus
          estrategias, jugadas clave y gestión de economía. No es solo un
          _tracker_, es un **coach virtual** con visión real de juego.
        </p>
      </div>
    </section>
  );
};

export default About;
