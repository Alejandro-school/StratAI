// Archivo: HowItWorks.jsx
import React from 'react';
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
  return (
    <section id="how-it-works" className={styles.section}>
      <div className={styles.container}>
        <h2 className={styles.heading}>¿Cómo Funciona?</h2>
        <div className={styles.steps}>
          {steps.map((item, index) => (
            <div className={styles.stepCard} key={index}>
              <span className={styles.stepNumber}>{item.step}</span>
              <h3>{item.title}</h3>
              <p>{item.text}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default HowItWorks;
