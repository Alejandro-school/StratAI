// Archivo: HowItWorks.jsx
import React, { useRef } from 'react';
import { useGsapFadeIn } from '../../hooks/useGsapFadeIn';
import styles from '../../styles/Landing/HowItWorks.module.css';

const steps = [
  {
    step: '1',
    title: 'Conecta tu cuenta de Steam',
    text: 'Accede con tu cuenta de Steam de forma segura y empieza a sincronizar tus partidas.'
  },
  {
    step: '2',
    title: 'Sube tus demos o códigos',
    text: 'Nuestro sistema detectará automáticamente tus partidas recientes para analizarlas.'
  },
  {
    step: '3',
    title: 'Recibe tu análisis',
    text: 'Visualiza las estadísticas detalladas, errores detectados y sugerencias de mejora.'
  }
];

const HowItWorks = () => {
  const ref = useRef(null);
  useGsapFadeIn(ref, { y: 80 });

  return (
    <section id="how" ref={ref} className={styles.howSection}>
      <div className={styles.wrapper}>
        <h2 className={styles.title}>¿Cómo funciona?</h2>
        <div className={styles.stepsGrid}>
          {steps.map((s, i) => (
            <div className={styles.stepCard} key={i}>
              <span className={styles.stepNumber}>{s.step}</span>
              <h3>{s.title}</h3>
              <p>{s.text}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default HowItWorks;
