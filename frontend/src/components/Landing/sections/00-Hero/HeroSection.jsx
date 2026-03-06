/**
 * HeroSection — Fullwidth centered hero with animated counters
 */
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { motion } from 'framer-motion';
import { ArrowUpRight, Activity, Cpu, Zap } from 'lucide-react';
import { useLang } from '../../i18n/useLang';
import { API_URL } from '../../../../utils/api';
import '../../../../styles/Landing/sections/hero.css';

const stagger = {
  hidden: {},
  show: { transition: { staggerChildren: 0.1, delayChildren: 0.2 } },
};

const fadeUp = {
  hidden: { opacity: 0, y: 24 },
  show: { opacity: 1, y: 0, transition: { duration: 0.7, ease: [0.22, 1, 0.36, 1] } },
};

const AnimatedCounter = ({ value, label }) => {
  const [display, setDisplay] = useState('0');
  const ref = useRef(null);
  const hasAnimated = useRef(false);

  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting && !hasAnimated.current) {
          hasAnimated.current = true;
          // Extract numeric part
          const numeric = parseInt(value.replace(/\D/g, ''), 10);
          if (isNaN(numeric)) {
            setDisplay(value);
            return;
          }
          const prefix = value.match(/^[^\d]*/)?.[0] || '';
          const suffix = value.match(/[^\d]*$/)?.[0] || '';
          const duration = 1500;
          const start = performance.now();

          const animate = (now) => {
            const elapsed = now - start;
            const progress = Math.min(elapsed / duration, 1);
            const eased = 1 - Math.pow(1 - progress, 3);
            const current = Math.round(eased * numeric);
            setDisplay(`${prefix}${current}${suffix}`);
            if (progress < 1) requestAnimationFrame(animate);
          };
          requestAnimationFrame(animate);
        }
      },
      { threshold: 0.5 }
    );
    if (ref.current) observer.observe(ref.current);
    return () => observer.disconnect();
  }, [value]);

  return (
    <div className="hero__stat" ref={ref}>
      <span className="hero__stat-value">{display}</span>
      <span className="hero__stat-label">{label}</span>
    </div>
  );
};

const BADGES = [
  { icon: Activity, key: 'badge1' },
  { icon: Cpu, key: 'badge2' },
  { icon: Zap, key: 'badge3' },
];

const HeroSection = ({ onOpenChallenge }) => {
  const { t } = useLang();

  const handleSteamLogin = useCallback(() => {
    window.location.href = `${API_URL}/auth/steam/login`;
  }, []);

  return (
    <section id="hero" className="hero-section">
      <div className="hero-section__content">
        <motion.div
          className="hero"
          variants={stagger}
          initial="hidden"
          animate="show"
        >
          {/* Status pill */}
          <motion.div className="hero__status" variants={fadeUp}>
            <span className="hero__status-dot" />
            <span>{t('hero.status')}</span>
          </motion.div>

          {/* Headlines */}
          <motion.h1 className="hero__h1" variants={fadeUp}>
            {t('hero.headline')}
          </motion.h1>
          <motion.h1 className="hero__h1 hero__h1--gradient" variants={fadeUp}>
            {t('hero.headlineGradient')}
          </motion.h1>

          {/* Subtitle */}
          <motion.p className="hero__sub" variants={fadeUp}>
            {t('hero.subtitle')}
          </motion.p>

          {/* Badges */}
          <motion.div className="hero__badges" variants={fadeUp}>
            {BADGES.map((b) => (
              <span key={b.key} className="hero__badge">
                <b.icon size={13} strokeWidth={2.2} />
                {t(`hero.${b.key}`)}
              </span>
            ))}
          </motion.div>

          {/* Dual CTA */}
          <motion.div className="hero__cta-wrap" variants={fadeUp}>
            <button className="hero__cta hero__cta--primary" onClick={handleSteamLogin}>
              <span>{t('hero.ctaPrimary')}</span>
              <ArrowUpRight size={17} strokeWidth={2.4} />
            </button>
            <button className="hero__cta hero__cta--secondary" onClick={onOpenChallenge}>
              <span>{t('hero.ctaSecondary')}</span>
            </button>
          </motion.div>

          {/* Animated stats */}
          <motion.div className="hero__stats" variants={fadeUp}>
            <AnimatedCounter value={t('hero.stat1Value')} label={t('hero.stat1Label')} />
            <div className="hero__stats-divider" />
            <AnimatedCounter value={t('hero.stat2Value')} label={t('hero.stat2Label')} />
            <div className="hero__stats-divider" />
            <AnimatedCounter value={t('hero.stat3Value')} label={t('hero.stat3Label')} />
            <div className="hero__stats-divider" />
            <AnimatedCounter value={t('hero.stat4Value')} label={t('hero.stat4Label')} />
            <div className="hero__stats-divider" />
            <AnimatedCounter value={t('hero.stat5Value')} label={t('hero.stat5Label')} />
          </motion.div>
        </motion.div>
      </div>
    </section>
  );
};

export default HeroSection;
