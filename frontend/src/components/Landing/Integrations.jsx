import React, { useRef } from 'react';
import { useGsapFadeIn } from '../../hooks/useGsapFadeIn';
import styles from '../../styles/Landing/Integrations.module.css';

const logos = [
  '/logos/steam.svg',
  '/logos/faceit.svg',
  '/logos/discord.svg',
  '/logos/twitch.svg',
  '/logos/openai.svg',
];

const Integrations = () => {
  const ref = useRef(null);
  useGsapFadeIn(ref, { y: 40 });

  return (
    <section id="integrations" ref={ref} className={styles.section}>
      <h2 className={styles.title}>Integraciones & Partners</h2>
      <div className={styles.marquee}>
        <div className={styles.track}>
          {logos.concat(logos).map((src, i) => (
            <img key={i} src={src} alt="logo partner" className={styles.logo} />
          ))}
        </div>
      </div>
    </section>
  );
};

export default Integrations;
