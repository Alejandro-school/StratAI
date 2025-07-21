import React, { useRef } from 'react';
import { useGsapFadeIn } from '../../hooks/useGsapFadeIn';
import styles from '../../styles/Landing/Pricing.module.css';

const plans = [
  { name: 'Free', price: '0€', features: ['Análisis básico', 'Hasta 5 demos/mes', 'Historial 30 días'] },
  { name: 'Pro', price: '9€',  features: ['Sin límite de demos', 'Consejos tácticos avanzados', 'Integración Faceit', 'Historial 180 días'] },
  { name: 'Elite', price: '19€', features: ['IA en tiempo real', 'Informes del equipo', 'Soporte prioritario', 'Historial ilimitado'] },
];

const Pricing = () => {
  const ref = useRef(null);
  useGsapFadeIn(ref, { y: 50 });

  return (
    <section id="pricing" ref={ref} className={styles.section}>
      <h2 className={styles.title}>Elige tu plan</h2>
      <div className={styles.grid}>
        {plans.map(({ name, price, features }) => (
          <div key={name} className={styles.card}>
            <h3>{name}</h3>
            <span className={styles.price}>{price}/mes</span>
            <ul>
              {features.map(f => <li key={f}>{f}</li>)}
            </ul>
            <button className={styles.btn}>Empezar</button>
          </div>
        ))}
      </div>
    </section>
  );
};

export default Pricing;
