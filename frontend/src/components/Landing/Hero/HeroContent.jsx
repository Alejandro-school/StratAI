/**
 * HeroContent - Left side content with title, subtitle, buttons and stats
 * 
 * Features:
 * - Animated badge
 * - AIMLAB (cyan gradient) + ANALYZER (purple gradient) titles
 * - Subtitle with highlighted terms
 * - CTA buttons with hover effects
 * - Stats row
 * - Smoke/evaporation effect when starting test
 */
import React from 'react';
import { motion } from 'framer-motion';
import { useLanding, LANDING_STEPS } from '../LandingContext';

const HeroContent = () => {
  const { goToStep, setTriggerMaterialize } = useLanding();

  const handleStartAnalysis = () => {
    // Trigger C4 materialization and navigate
    setTriggerMaterialize(true);
    goToStep(LANDING_STEPS.NICKNAME);
  };

  const stats = [
    { value: '50K+', label: 'Demos Analizados' },
    { value: '99.2%', label: 'Precisión del Modelo' },
    { value: '0.3s', label: 'Tiempo de Inferencia' },
  ];

  return (
    <div className="hero-content">
      {/* Badge */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.6 }}
        className="hero-badge"
      >
        <motion.span
          className="hero-badge-icon"
          animate={{ rotate: 360 }}
          transition={{ duration: 4, repeat: Infinity, ease: 'linear' }}
        >
          ⊕
        </motion.span>
        <span className="hero-badge-text">
          ANÁLISIS TÁCTICO CON IA AVANZADA
        </span>
      </motion.div>

      {/* Main Title */}
      <div className="hero-title-wrapper">
        {/* AIMLAB */}
        <div className="hero-title-overflow">
          <motion.h1
            initial={{ y: 100, opacity: 0 }}
            animate={{ y: 0, opacity: 1 }}
            transition={{ delay: 0.2, duration: 0.8, ease: [0.25, 1, 0.5, 1] }}
            className="hero-title hero-title-aimlab"
          >
            STRAT<span style={{color: '#00d4ff'}}>AI</span>
          </motion.h1>
        </div>
        
        {/* ANALYZER */}
        <div className="hero-title-overflow">
          <motion.h1
            initial={{ y: 100, opacity: 0 }}
            animate={{ y: 0, opacity: 1 }}
            transition={{ delay: 0.3, duration: 0.8, ease: [0.25, 1, 0.5, 1] }}
            className="hero-title hero-title-analyzer"
          >
            ANALYZER
          </motion.h1>
        </div>
      </div>

      {/* Subtitle */}
      <motion.p
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.6, duration: 0.6 }}
        className="hero-subtitle"
      >
        Descubre los patrones que te hacen perder duelos con 
        <span className="hero-highlight-cyan"> machine learning</span>
        {' '}entrenado en{' '}
        <span className="hero-highlight-purple">+50K demos de CS2</span>.
      </motion.p>

      {/* CTA Buttons */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.8, duration: 0.6 }}
        className="hero-buttons"
      >
        <motion.button
          onClick={handleStartAnalysis}
          className="hero-btn-primary"
          whileHover={{ scale: 1.02 }}
          whileTap={{ scale: 0.98 }}
        >
          <span className="hero-btn-text">Hacer Test</span>
          <span className="hero-btn-arrow">›</span>
          <motion.span
            className="hero-btn-shine"
            initial={{ x: '-100%' }}
            whileHover={{ x: '100%' }}
            transition={{ duration: 0.5 }}
          />
        </motion.button>
      </motion.div>

      {/* Stats Row */}
      <motion.div
        initial={{ opacity: 0, y: 30 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 1, duration: 0.6 }}
        className="hero-stats"
      >
        {stats.map((stat, index) => (
          <motion.div
            key={stat.label}
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ delay: 1.2 + index * 0.1 }}
            className="hero-stat"
          >
            <p className="hero-stat-value">{stat.value}</p>
            <p className="hero-stat-label">{stat.label}</p>
          </motion.div>
        ))}
      </motion.div>
    </div>
  );
};

export default HeroContent;
