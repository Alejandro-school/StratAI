import React from 'react';
import { FaSteam, FaCogs, FaChartBar } from 'react-icons/fa';
import useFadeInOnScroll from '../../hooks/useFadeInOnScroll';

const steps = [
  { icon: <FaSteam size={28} />, text: 'Inicia sesi\u00f3n con Steam' },
  { icon: <FaCogs size={28} />, text: 'Procesamos tus partidas autom\u00e1ticamente' },
  { icon: <FaChartBar size={28} />, text: 'Recibes an\u00e1lisis detallado' },
];

const HowItWorks = () => {
  const ref = useFadeInOnScroll();
  return (
    <section ref={ref} className="fade-section">
      <div className="container">
        <h2>\u00bfC\u00f3mo funciona?</h2>
        <div className="steps">
          {steps.map((s, i) => (
            <div key={i} className="step">
              {s.icon}
              <p>{s.text}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default HowItWorks;
