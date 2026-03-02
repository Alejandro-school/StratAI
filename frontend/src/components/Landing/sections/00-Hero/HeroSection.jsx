/**
 * HeroSection — High-conversion SaaS hero
 *
 * Three visual zones stacked vertically, all within the left 60% stage:
 *   1. Hero: Headline + subtitle + badges + CTA
 *   2. Analysis: 3 gradient-bordered cards
 *   3. Process: Horizontal timeline with connectors
 *
 * The 3D agent wireframe occupies the right 40% (rendered by LandingPage).
 */
import React, { useCallback } from 'react';
import { motion } from 'framer-motion';
import {
  ArrowRight,
  Crosshair,
  DollarSign,
  Bomb,
  Activity,
  Cpu,
  Zap,
  LogIn,
  ScanSearch,
  TrendingUp,
} from 'lucide-react';
import { useLanding, STAGES } from '../../LandingContext';
import '../../../../styles/Landing/sections/hero.css';

/* ── Framer variants ─────────────────────────────────────────────── */
const stagger = {
  hidden: {},
  show: { transition: { staggerChildren: 0.08, delayChildren: 0.15 } },
};

const fadeUp = {
  hidden: { opacity: 0, y: 18 },
  show: {
    opacity: 1,
    y: 0,
    transition: { duration: 0.6, ease: [0.22, 1, 0.36, 1] },
  },
};

const scaleIn = {
  hidden: { opacity: 0, scale: 0.92 },
  show: {
    opacity: 1,
    scale: 1,
    transition: { duration: 0.5, ease: [0.22, 1, 0.36, 1] },
  },
};

/* ── Static data ─────────────────────────────────────────────────── */
const BADGES = [
  { icon: Activity, text: '128-Tick Accuracy' },
  { icon: Cpu, text: 'Triple AI Agent' },
  { icon: Zap, text: 'Zero-Latency Analysis' },
];

const CARDS = [
  {
    icon: DollarSign,
    accent: 'var(--color-success)',
    title: 'Economía',
    desc: 'Detecta ecos mal gestionados, compras subóptimas y desincronización económica de equipo.',
  },
  {
    icon: Crosshair,
    accent: 'var(--color-primary-400)',
    title: 'Aim',
    desc: 'Mide precisión, spray control, counter-strafe y crosshair placement en cada duel.',
  },
  {
    icon: Bomb,
    accent: 'var(--color-secondary-500)',
    title: 'Utilidad',
    desc: 'Evalúa cada granada lanzada: smokes tardíos, flashes a aliados y molotovs sin impacto.',
  },
  {
    icon: Activity,
    accent: 'var(--color-warning)',
    title: 'Movimiento',
    desc: 'Analiza rotaciones, posicionamiento, timings de peek y decisiones en clutch.',
  },
];

const STEPS = [
  { icon: LogIn, label: 'Conectar Steam', num: '01' },
  { icon: ScanSearch, label: 'Análisis IA', num: '02' },
  { icon: TrendingUp, label: 'Mejora Continua', num: '03' },
];

/* ── Component ───────────────────────────────────────────────────── */
const HeroSection = () => {
  const { transitionTo } = useLanding();
  const handleStart = useCallback(
    () => transitionTo(STAGES.CHAT_DEMO),
    [transitionTo],
  );

  return (
    <section id="hero" className="hero-section">
      <div className="hero-section__content">
        <motion.div
          className="hero"
          variants={stagger}
          initial="hidden"
          animate="show"
        >
          {/* ═══ ZONE 1 — HERO ═══ */}
          <div className="hero__top">
            {/* Status */}
            <motion.div className="hero__status" variants={fadeUp}>
              <span className="hero__status-dot" />
              <span>SISTEMA ACTIVO</span>
            </motion.div>

            {/* Headline */}
            <motion.h1 className="hero__h1" variants={fadeUp}>
              Domina tu Elo.
            </motion.h1>
            <motion.h1 className="hero__h1 hero__h1--gradient" variants={fadeUp}>
              Deja de adivinar por qué pierdes.
            </motion.h1>

            {/* Subtitle */}
            <motion.p className="hero__sub" variants={fadeUp}>
              Conecta Steam. Descargamos tus demos automáticamente y las
              analizamos con 3 agentes de IA especializados. Sin esfuerzo.
            </motion.p>

            {/* Badges */}
            <motion.div className="hero__badges" variants={fadeUp}>
              {BADGES.map((b) => (
                <span key={b.text} className="hero__badge">
                  <b.icon size={13} strokeWidth={2.2} />
                  {b.text}
                </span>
              ))}
            </motion.div>

            {/* CTA */}
            <motion.div className="hero__cta-wrap" variants={fadeUp}>
              <button className="hero__cta" onClick={handleStart}>
                <span>Ver Demo</span>
                <ArrowRight size={17} strokeWidth={2.4} />
              </button>
            </motion.div>
          </div>

          {/* ═══ ZONE 2 — ANALYSIS CARDS ═══ */}
          <motion.div className="hero__cards-section" variants={fadeUp}>
            <span className="hero__label">Áreas de análisis</span>
            <div className="hero__cards">
              {CARDS.map((c) => (
                <motion.div
                  key={c.title}
                  className="hero__card"
                  style={{ '--card-accent': c.accent }}
                  variants={scaleIn}
                  whileHover={{ y: -3, transition: { duration: 0.25 } }}
                >
                  <div className="hero__card-icon">
                    <c.icon size={18} strokeWidth={2} />
                  </div>
                  <span className="hero__card-title">{c.title}</span>
                  <span className="hero__card-desc">{c.desc}</span>
                </motion.div>
              ))}
            </div>
          </motion.div>

          {/* ═══ ZONE 3 — PROCESS TIMELINE ═══ */}
          <motion.div className="hero__process" variants={fadeUp}>
            <span className="hero__label">Cómo funciona</span>
            <div className="hero__timeline">
              {STEPS.map((s, i) => (
                <React.Fragment key={s.num}>
                  <div className="hero__tl-step">
                    <div className="hero__tl-icon">
                      <s.icon size={16} strokeWidth={2} />
                    </div>
                    <span className="hero__tl-num">{s.num}</span>
                    <span className="hero__tl-label">{s.label}</span>
                  </div>
                  {i < STEPS.length - 1 && <div className="hero__tl-connector" />}
                </React.Fragment>
              ))}
            </div>
          </motion.div>
        </motion.div>
      </div>
    </section>
  );
};

export default HeroSection;
