import React from 'react';
import useFadeInOnScroll from '../../hooks/useFadeInOnScroll';
import useParallax from '../../hooks/useParallax';



const reviews = [
  { name: 'Carlos', text: 'Sub\u00ed de 0.9 a 1.3 de K/D en 3 semanas con StratAI.' },
  { name: 'Luc\u00eda', text: 'El an\u00e1lisis ronda a ronda me ayud\u00f3 a entender mis errores.' },
];

const Testimonials = () => {
  const fadeRef = useFadeInOnScroll();
  const parallaxRef = useParallax(0.05);
  return (
    <section ref={fadeRef} className="fade-section">
      <div ref={parallaxRef} className="container parallax">
        <h2>Testimonios</h2>
        <div className="testimonials">
          {reviews.map((r, i) => (
            <div key={i} className="testimonial">
              <p>{r.text}</p>
              <strong>- {r.name}</strong>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default Testimonials;
