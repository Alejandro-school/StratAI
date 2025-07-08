import React from 'react';
import { FaChartLine, FaRobot, FaPlayCircle, FaStar } from 'react-icons/fa';
import useFadeInOnScroll from '../../hooks/useFadeInOnScroll';
import useParallax from '../../hooks/useParallax';

const data = [
  { icon: <FaChartLine size={32} />, title: 'An\u00e1lisis de partidas', text: 'Revisa cada detalle de tus encuentros.' },
  { icon: <FaRobot size={32} />, title: 'Entrenador virtual', text: 'Recibe sugerencias autom\u00e1ticas para mejorar.' },
  { icon: <FaPlayCircle size={32} />, title: 'Repeticiones 2D', text: 'Visualiza cada ronda desde un punto de vista estrat\u00e9gico.' },
  { icon: <FaStar size={32} />, title: 'Seguimiento del progreso', text: 'Observa c\u00f3mo evoluciona tu rendimiento semana a semana.' },
];

const Features = () => {
  const fadeRef = useFadeInOnScroll();
  const parallaxRef = useParallax(0.1);
  return (
    <section ref={fadeRef} className="fade-section">
      <div ref={parallaxRef} className="container parallax">
        <h2>Funciones Principales</h2>
        <div className="features-grid">
          {data.map((f, i) => (
            <div key={i} className="feature-item">
              {f.icon}
              <h3>{f.title}</h3>
              <p>{f.text}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default Features;
