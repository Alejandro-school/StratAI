import React, { useRef } from 'react';
import { useGsapFadeIn } from '../../hooks/useGsapFadeIn';
import styles from '../../styles/Landing/SecurityPrivacy.module.css';

const SecurityPrivacy = () => {
  const ref = useRef(null);
  useGsapFadeIn(ref);

  return (
    <section id="security" ref={ref} className={styles.section}>
      <h2 className={styles.title}>Seguridad y Privacidad</h2>
      <p className={styles.text}>
        Todos tus datos se almacenan en servidores europeos con cifrado
        <strong> AES-256</strong>. Eliminamos replays brutas tras 30 días en el plan Free
        y tras 180 días en planes de pago. Cumplimos con el RGPD y nunca
        compartiremos tu información con terceros sin tu consentimiento.
      </p>
    </section>
  );
};

export default SecurityPrivacy;
