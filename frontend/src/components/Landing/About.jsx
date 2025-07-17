// Archivo: About.jsx
import React from 'react';
import styles from '../../styles/Landing/About.module.css';

const About = () => {
  return (
    <section id="about" className={styles.aboutSection}>
      <div className={styles.contentWrapper}>
        <h2 className={styles.heading}>¿Qué es StratAI?</h2>
        <p className={styles.paragraph}>
          StratAI es tu entrenador personal de Counter-Strike 2. Utilizando inteligencia artificial, analiza tus partidas, detecta errores estratégicos, técnicos y de posicionamiento, y te da consejos personalizados para mejorar.
        </p>
        <p className={styles.paragraph}>
          Nuestra plataforma te ofrece una ventaja competitiva analizando estadísticas avanzadas, jugadas clave, gestión de economía y más. No es solo un tracker, es un **coach virtual** con visión real de juego.
        </p>
      </div>
    </section>
  );
};

export default About;
