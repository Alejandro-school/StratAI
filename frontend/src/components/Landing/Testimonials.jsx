// Archivo: Testimonials.jsx
import React from 'react';
import styles from '../../styles/Landing/Testimonials.module.css';

const testimonials = [
  {
    name: 'Carlos “D4rk” Fernández',
    text: 'StratAI me ayudó a subir de nivel. Antes cometía errores sin darme cuenta, ahora entiendo mis fallos y he mejorado mi winrate.'
  },
  {
    name: 'Lucía “Foxie” Martínez',
    text: 'Gracias a las recomendaciones automáticas descubrí que rotaba tarde o mal. Ahora mis timings son mucho más precisos.'
  },
  {
    name: 'Javier “SniperOne” Gómez',
    text: 'No es solo un analizador, es un coach real. La IA detecta patrones que ningún humano ve. Me encanta.'
  }
];

const Testimonials = () => {
  return (
    <section id="testimonials" className={styles.testimonialSection}>
      <div className={styles.container}>
        <h2 className={styles.heading}>Lo que dicen nuestros jugadores</h2>
        <div className={styles.testimonialGrid}>
          {testimonials.map((item, index) => (
            <div key={index} className={styles.card}>
              <p className={styles.text}>&ldquo;{item.text}&rdquo;</p>
              <span className={styles.name}>— {item.name}</span>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default Testimonials;
