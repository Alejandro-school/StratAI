import React, { useRef } from 'react';
import { useGsapFadeIn } from '../../hooks/useGsapFadeIn';
import styles from '../../styles/Landing/ProblemSolution.module.css';

const ProblemSolution = () => {
  const ref = useRef(null);
  useGsapFadeIn(ref, { y: 70 });

  return (
    <section id="problem" ref={ref} className={styles.section}>
      <h2 className={styles.title}>El problema</h2>
      <p className={styles.text}>
        Los <strong>stats genéricos</strong> no te dicen <em>por qué</em> pierdes las rondas.
        Tus demos se acumulan sin ningún análisis táctico...
      </p>

      <h2 className={styles.title}>La solución</h2>
      <p className={styles.text}>
        <strong>StratAI</strong> desglosa cada ronda, identifica errores de posicionamiento y
        propone <em>calls</em> óptimos basados en IA. ¡Mejora tu win-rate hasta un 18 %!
      </p>
    </section>
  );
};

export default ProblemSolution;
