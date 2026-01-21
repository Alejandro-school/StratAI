/**
 * AIIndicator - Right panel with neural network status and metrics
 * 
 * Features:
 * - Pulse rings around brain icon
 * - Animated metrics cards with scan lines
 * - Processing status bar
 */
import React from 'react';
import { motion } from 'framer-motion';

// Scan Line Animation for cards
const ScanLine = () => (
  <motion.div
    className="ai-scan-line"
    initial={{ top: '0%', opacity: 0 }}
    animate={{ top: '100%', opacity: [0, 1, 1, 0] }}
    transition={{ duration: 2, repeat: Infinity, ease: 'linear' }}
  />
);

// Pulse Ring Animation
const PulseRing = ({ delay = 0 }) => (
  <motion.div
    className="ai-pulse-ring"
    initial={{ scale: 0.8, opacity: 0.8 }}
    animate={{ scale: 1.5, opacity: 0 }}
    transition={{ duration: 2, repeat: Infinity, delay, ease: 'easeOut' }}
  />
);

const AIIndicator = () => {
  const metrics = [
    { label: 'Precisión', value: '94.7%', color: 'cyan', icon: '◎' },
    { label: 'Reacción', value: '187ms', color: 'purple', icon: '⚡' },
    { label: 'Análisis', value: 'Activo', color: 'green', icon: '🔍' },
  ];

  return (
    <motion.div
      initial={{ opacity: 0, x: 50 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ delay: 1, duration: 0.8 }}
      className="ai-indicator"
    >
      {/* AI Core Header */}
      <div className="ai-core-header">
        <div className="ai-brain-wrapper">
          <PulseRing delay={0} />
          <PulseRing delay={0.5} />
          <PulseRing delay={1} />
          <motion.div
            className="ai-brain-icon"
            animate={{ rotate: 360 }}
            transition={{ duration: 20, repeat: Infinity, ease: 'linear' }}
          >
            🧠
          </motion.div>
        </div>
        <div className="ai-core-info">
          <motion.p
            className="ai-status-text"
            animate={{ opacity: [0.5, 1, 0.5] }}
            transition={{ duration: 2, repeat: Infinity }}
          >
            NEURAL NETWORK ACTIVE
          </motion.p>
          <p className="ai-engine-name">CS2 Analysis Engine</p>
        </div>
      </div>

      {/* Metrics Cards */}
      <div className="ai-metrics">
        {metrics.map((metric, index) => (
          <motion.div
            key={metric.label}
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: 1.2 + index * 0.15 }}
            className="ai-metric-card"
          >
            <ScanLine />
            <div className="ai-metric-content">
              <div className={`ai-metric-icon ai-metric-icon--${metric.color}`}>
                {metric.icon}
              </div>
              <span className="ai-metric-label">{metric.label}</span>
              <motion.span
                className={`ai-metric-value ai-metric-value--${metric.color}`}
                animate={{ opacity: [1, 0.7, 1] }}
                transition={{ duration: 1.5, repeat: Infinity, delay: index * 0.3 }}
              >
                {metric.value}
              </motion.span>
            </div>
          </motion.div>
        ))}
      </div>

      {/* Status Bar */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 1.8 }}
        className="ai-status-bar"
      >
        <div className="ai-status-indicator">
          <motion.div
            className="ai-status-dot"
            animate={{ scale: [1, 1.3, 1], opacity: [1, 0.7, 1] }}
            transition={{ duration: 1, repeat: Infinity }}
          />
          <span className="ai-status-message">
            Procesando datos de entrenamiento...
          </span>
        </div>
        <div className="ai-progress-bar">
          <motion.div
            className="ai-progress-fill"
            initial={{ width: '0%' }}
            animate={{ width: '100%' }}
            transition={{ duration: 3, repeat: Infinity, ease: 'easeInOut' }}
          />
        </div>
      </motion.div>
    </motion.div>
  );
};

export default AIIndicator;
