/**
 * HowItWorksSection — Alternating vertical timeline
 * Scroll-driven progress line + deblur parallax cards + spring node reveals
 */
import React, { useRef } from 'react';
import { motion, useScroll, useTransform, useSpring } from 'framer-motion';
import { LogIn, Brain, TrendingUp } from 'lucide-react';
import { useLang } from '../../i18n/useLang';
import '../../../../styles/Landing/sections/howItWorks.css';

const STEPS = [
  { num: '01', icon: LogIn, titleKey: 'step1Title', descKey: 'step1Desc' },
  { num: '02', icon: Brain, titleKey: 'step2Title', descKey: 'step2Desc' },
  { num: '03', icon: TrendingUp, titleKey: 'step3Title', descKey: 'step3Desc' },
];

/* ── Single timeline node + card ────────────────────────────────────── */
const TimelineStep = ({ num, icon: Icon, titleKey, descKey, index, t }) => {
  const ref = useRef(null);
  const { scrollYProgress } = useScroll({
    target: ref,
    offset: ['start 0.85', 'start 0.35'],
  });

  const smoothProgress = useSpring(scrollYProgress, { stiffness: 100, damping: 30 });

  /* Card deblur + slide */
  const side = index % 2 === 0 ? -1 : 1;
  const x      = useTransform(smoothProgress, [0, 1], [side * 80, 0]);
  const opacity = useTransform(smoothProgress, [0, 0.4], [0, 1]);
  const scale   = useTransform(smoothProgress, [0, 1], [0.88, 1]);
  const blur    = useTransform(smoothProgress, [0, 0.6], [8, 0]);
  const filter  = useTransform(blur, (v) => `blur(${v}px)`);

  /* Node pop */
  const nodeScale = useTransform(smoothProgress, [0, 0.5], [0, 1]);
  const nodeSpring = useSpring(nodeScale, { stiffness: 260, damping: 18 });

  return (
    <div ref={ref} className={`hiw-row ${index % 2 === 0 ? 'hiw-row--left' : 'hiw-row--right'}`}>
      {/* Card */}
      <motion.div
        className="hiw-card"
        style={{ x, opacity, scale, filter }}
      >
        <span className="hiw-card__num">{num}</span>
        <div className="hiw-card__icon">
          <Icon size={22} strokeWidth={1.8} />
        </div>
        <h3 className="hiw-card__title">{t(`howItWorks.${titleKey}`)}</h3>
        <p className="hiw-card__desc">{t(`howItWorks.${descKey}`)}</p>
      </motion.div>

      {/* Central node */}
      <motion.div className="hiw-node" style={{ scale: nodeSpring }}>
        <span className="hiw-node__dot" />
      </motion.div>

      {/* Empty cell for grid alignment */}
      <div className="hiw-spacer" />
    </div>
  );
};

/* ── Main section ───────────────────────────────────────────────────── */
const HowItWorksSection = () => {
  const { t } = useLang();
  const sectionRef = useRef(null);

  /* Scroll-linked progress for the vertical line */
  const { scrollYProgress } = useScroll({
    target: sectionRef,
    offset: ['start 0.7', 'end 0.6'],
  });
  const lineProgress = useSpring(scrollYProgress, { stiffness: 80, damping: 28 });

  return (
    <section id="how-it-works" className="hiw-section" ref={sectionRef}>
      <div className="hiw-section__inner">
        {/* Header */}
        <motion.div
          className="section-header"
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true, margin: '-80px' }}
          transition={{ duration: 0.6 }}
        >
          <span className="section-label">{t('howItWorks.label')}</span>
          <h2 className="section-title">{t('howItWorks.title')}</h2>
        </motion.div>

        {/* Timeline container */}
        <div className="hiw-timeline">
          {/* Vertical scroll-driven line */}
          <div className="hiw-line-track" aria-hidden="true">
            <motion.div className="hiw-line-fill" style={{ scaleY: lineProgress }} />
          </div>

          {/* Steps */}
          {STEPS.map((step, i) => (
            <TimelineStep key={step.num} {...step} index={i} t={t} />
          ))}
        </div>
      </div>
    </section>
  );
};

export default HowItWorksSection;
