import React, { useRef } from 'react';
import gsap from 'gsap';
import styles from '../../styles/Landing/FAQ.module.css';

const faqs = [
  { q: '¿Necesito subir mis demos?', a: 'No es necesario, nuestro sistema se encarga de extraer las demos de forma automatizada. Aunque también puedes subir alguna demo si así lo deseas' },
  { q: '¿Funciona con CS:GO antiguo?', a: 'Analizamos CS2; versiones previas no tienen soporte oficial.' },
  { q: '¿Puedo cancelar cuando quiera?', a: 'Claro, no hay permanencia. Tu plan se detendrá al final del ciclo de facturación.' },
  { q: '¿Mi información está segura?', a: 'Aplicamos cifrado AES-256 y eliminamos datos brutos tras 30 días en planes Free.' },
];

const FAQ = () => {
  const toggle = (i) => {
    const item = document.querySelector(`#faq-${i}`);
    const content = item.querySelector('div');
    const isOpen = item.classList.contains(styles.open);

    gsap.to(content, {
      height: isOpen ? 0 : content.scrollHeight,
      duration: 0.4,
      ease: 'power2.out',
    });
    item.classList.toggle(styles.open);
  };

  return (
    <section id="faq" className={styles.section}>
      <h2 className={styles.title}>Preguntas frecuentes</h2>
      <div className={styles.wrapper}>
        {faqs.map(({ q, a }, i) => (
          <div id={`faq-${i}`} key={q} className={styles.item}>
            <button className={styles.question} onClick={() => toggle(i)}>{q}</button>
            <div className={styles.answer}><p>{a}</p></div>
          </div>
        ))}
      </div>
    </section>
  );
};

export default FAQ;
