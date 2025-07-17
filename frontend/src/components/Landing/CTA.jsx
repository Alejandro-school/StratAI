// Archivo: CTA.jsx
import React from 'react';
import styles from '../../styles/Landing/Cta.module.css';
import SteamLoginButton from '../../auth/SteamLoginButton.jsx';

const CTA = () => {
  return (
    <section id="cta" className={styles.ctaSection}>
      <div className={styles.wrapper}>
        <h2 className={styles.heading}>Empieza ahora y mejora desde tu próxima partida</h2>
        <SteamLoginButton />
      </div>
    </section>
  );
};

export default CTA;