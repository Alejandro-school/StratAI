// Archivo: Features.jsx
import React, { useRef } from 'react';
import { useGsapFadeIn } from '../../hooks/useGsapFadeIn';
import styles from '../../styles/Landing/Features.module.css';

const features = [
  {
    title: 'Estadísticas Avanzadas',
    description: 'Analiza cada detalle de tu rendimiento: K/D, ADR, HS%, economía, y más con visualizaciones claras y útiles.'
  },
  {
    title: 'Detección de Errores',
    description: 'Identificamos errores tácticos, mecánicos y de posicionamiento para que sepas exactamente qué mejorar.'
  },
  {
    title: 'Coach Inteligente',
    description: 'Recomendaciones automáticas y personalizadas basadas en IA que evolucionan con tu juego.'
  },
  {
    title: 'Historial Completo',
    description: 'Consulta el histórico de tus partidas, clasificaciones por mapa, y evalúa tu evolución a lo largo del tiempo.'
  }
];

const Features = () => {
  const ref = useRef(null);
  useGsapFadeIn(ref);

  return (
    <section id="features" ref={ref} className={styles.featuresSection}>
      <div className={styles.container}>
        <h2 className={styles.title}>Funciones Clave</h2>
        <div className={styles.grid}>
          {features.map(({ title, description }, i) => (
            <div key={i} className={styles.card}>
              <h3>{title}</h3>
              <p>{description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default Features;