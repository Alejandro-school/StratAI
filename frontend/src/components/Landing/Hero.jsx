// Archivo: Hero.jsx
import React, { useEffect, useRef } from 'react';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import styles from '../../styles/Landing/Hero.module.css';
import backgroundVideo from '../../media/Background.mp4';
import SteamLoginButton from '../../auth/SteamLoginButton.jsx';

const Hero = () => {
  const overlayRef = useRef(null);

  useEffect(() => {
    gsap.registerPlugin(ScrollTrigger);
    if (!overlayRef.current) return;
    gsap.to(overlayRef.current, {
      yPercent: -20,
      ease: 'none',
      scrollTrigger: {
        trigger: overlayRef.current,
        start: 'top top',
        scrub: true,
      },
    });
  }, []);

  return (
    <section id="hero" className={styles.heroSection}>
      <video autoPlay muted loop className={styles.backgroundVideo}>
        <source src={backgroundVideo} type="video/mp4" />
      </video>
      <div ref={overlayRef} className={styles.overlay}>
        <h1 className={styles.title}>Domina tus partidas con <span>StratAI</span></h1>
        <p className={styles.subtitle}>
          Analiza, aprende y mejora tu rendimiento en CS2 gracias a inteligencia artificial avanzada.
        </p>
        <SteamLoginButton />
      </div>
    </section>
  );
};

export default Hero;
