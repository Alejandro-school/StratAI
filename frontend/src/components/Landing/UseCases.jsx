import React, { useRef } from 'react';
import { useGsapFadeIn } from '../../hooks/useGsapFadeIn';
import styles from '../../styles/Landing/UseCases.module.css';

const cases = [
  { title: 'Jugador competitivo', text: 'Pulsa tu técnica individual y toma mejores decisiones en clutch.' },
  { title: 'IGL / Capitan', text: 'Genera estrategias basadas en patrones reales de tu equipo y rivales.' },
  { title: 'Coach / Analista', text: 'Ahorra horas revisando demos; obtén insights tácticos inmediatos.' },
];

const UseCases = () => {
  const ref = useRef(null);
  useGsapFadeIn(ref);

  return (
    <section id="usecases" ref={ref} className={styles.section}>
      <h2 className={styles.title}>Casos de uso</h2>
      <div className={styles.grid}>
        {cases.map(({ title, text }) => (
          <div key={title} className={styles.card}>
            <h3>{title}</h3>
            <p>{text}</p>
          </div>
        ))}
      </div>
    </section>
  );
};

export default UseCases;
