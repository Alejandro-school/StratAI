// Archivo: Hero.jsx
import React from 'react';
import styles from '../../styles/Landing/Hero.module.css';
import backgroundVideo from '../../media/Background.mp4';
import SteamLoginButton from '../../auth/SteamLoginButton.jsx';

const Hero = () => {
  return (
    <section className={styles.heroSection}>
      <video autoPlay muted loop className={styles.backgroundVideo}>
        <source src={backgroundVideo} type="video/mp4" />
      </video>
      <div className={styles.overlay}>
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
