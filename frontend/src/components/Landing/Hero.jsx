import React, { useEffect, useRef } from 'react';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { useGsapFadeIn } from '../../hooks/useGsapFadeIn';
import styles from '../../styles/Landing/Hero.module.css';
import SteamLoginButton from '../../auth/SteamLoginButton.jsx';

/**
 * Hero – sección principal con vídeo de fondo.
 *
 * ▸ Coloca tu archivo MP4 en `public/videos/hero-bg.mp4`.
 *   (Ajusta la ruta en el atributo `src` si usas otro nombre.)
 * ▸ Mantiene el parallax overlay & fade‑in existentes.
 */
const Hero = () => {
  const overlayRef = useRef(null);

  /* ① Parallax sobre el overlay */
  useEffect(() => {
    gsap.registerPlugin(ScrollTrigger);
    if (!overlayRef.current) return;

    gsap.to(overlayRef.current, {
      yPercent: -10,
      ease: 'none',
      scrollTrigger: {
        trigger: overlayRef.current,
        start: 'top top',
        scrub: true,
      },
    });
  }, []);

  /* ② Fade‑in para título, subtítulo y botón */
  useGsapFadeIn(overlayRef, { y: 60, duration: 1.2 });

  return (
    <section id="hero" className={styles.heroSection}>
      {/* ──────────────── Vídeo de fondo ────────────── */}
      <video
        className={styles.backgroundVideo}
        src="/videos/hero-bg.mp4"
        autoPlay
        loop
        muted
        playsInline
        preload="metadata"
      />

      {/* ──────────────── Overlay de contenido ────────────── */}
      <div ref={overlayRef} className={styles.overlay}>
        <h1 className={styles.title}>
          Domina tus partidas con <span>StratAI</span>
        </h1>
        <p className={styles.subtitle}>
          Analiza, aprende y mejora tu rendimiento en CS2 gracias a inteligencia
          artificial avanzada.
        </p>
        <SteamLoginButton className={styles.ctaButton} />
      </div>
    </section>
  );
};

export default Hero;
