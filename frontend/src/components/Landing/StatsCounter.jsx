import React from 'react';
import { useCountUp } from '../../hooks/useCountUp';
import { useGsapFadeIn } from '../../hooks/useGsapFadeIn';
import styles from '../../styles/Landing/StatsCounter.module.css';

/* --- Datos estáticos --- */
const stats = [
  { label: 'Partidas analizadas', value: 12_857 },
  { label: 'Usuarios activos',    value: 3_526  },
  { label: 'Win-rate medio ↑',    value: 18, suffix: '%' },
  { label: 'Demos procesadas / h',value: 740   },
];

/* --- Sub-componente con el hook --- */
const CounterCard = ({ label, value, suffix = '' }) => {
  const numRef = useCountUp(value);

  return (
    <div className={styles.card}>
      <span ref={numRef} className={styles.number} />{suffix}
      <p className={styles.label}>{label}</p>
    </div>
  );
};

const StatsCounter = () => {
  const sectionRef = React.useRef(null);
  useGsapFadeIn(sectionRef, { y: 40 });

  return (
    <section id="stats" ref={sectionRef} className={styles.section}>
      <div className={styles.grid}>
        {stats.map((s) => (
          <CounterCard key={s.label} {...s} />
        ))}
      </div>
    </section>
  );
};

export default StatsCounter;
