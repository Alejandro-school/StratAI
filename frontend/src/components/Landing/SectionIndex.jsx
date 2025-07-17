import React, { useEffect, useState } from 'react';
import '../../styles/Landing/sectionIndex.css';

const sections = [
  { id: 'hero', label: 'Inicio' },
  { id: 'about', label: 'Sobre' },
  { id: 'features', label: 'Funciones' },
  { id: 'how-it-works', label: 'Proceso' },
  { id: 'testimonials', label: 'Opiniones' },
  { id: 'comparison', label: 'Comparativa' },
  { id: 'cta', label: 'Comenzar' },
];

const SectionIndex = () => {
  const [active, setActive] = useState('hero');

  useEffect(() => {
    const observers = sections.map(sec => {
      const el = document.getElementById(sec.id);
      if (!el) return null;
      const obs = new IntersectionObserver(
        ([entry]) => {
          if (entry.isIntersecting) {
            setActive(sec.id);
          }
        },
        { threshold: 0.5 }
      );
      obs.observe(el);
      return obs;
    });
    return () => {
      observers.forEach(o => o && o.disconnect());
    };
  }, []);

  return (
    <nav className="section-nav">
      {sections.map((sec, idx) => (
        <a key={sec.id} href={`#${sec.id}`} className={`nav-item${active === sec.id ? ' active' : ''}`}>
          <span className="progress" />
          <span className="label">{idx + 1}</span>
        </a>
      ))}
    </nav>
  );
};

export default SectionIndex;
