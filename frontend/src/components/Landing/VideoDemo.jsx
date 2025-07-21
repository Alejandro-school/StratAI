import React, { useRef, useEffect } from 'react';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import styles from '../../styles/Landing/VideoDemo.module.css';

const VideoDemo = () => {
  const videoRef = useRef(null);

  useEffect(() => {
    gsap.registerPlugin(ScrollTrigger);
    const el = videoRef.current;
    if (!el) return;

    gsap.fromTo(
      el,
      { opacity: 0, scale: 0.9 },
      {
        opacity: 1,
        scale: 1,
        duration: 1,
        ease: 'power3.out',
        scrollTrigger: { trigger: el, start: 'top 80%' },
      }
    );
  }, []);

  return (
    <section id="video" className={styles.section}>
      <h2 className={styles.title}>Mira StratAI en acción</h2>
      <video
        ref={videoRef}
        className={styles.video}
        src="/videos/hero-bg.mp4"
        playsInline
        muted
        loop
        autoPlay
      />
    </section>
  );
};

export default VideoDemo;
