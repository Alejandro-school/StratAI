/**
 * Scene3DCSS - CSS-based 3D background effects
 * 
 * Provides visual depth with:
 * - Animated perspective grid
 * - Floating particles
 * - Crosshair targets
 * - Scan line effect
 * 
 * Note: The C4 model is rendered in SharedCanvas (Three.js) and visible through transparent background
 */
import React from 'react';
import { motion } from 'framer-motion';

// Floating Particles - positioned away from center to not obstruct C4
const FloatingParticles = ({ count = 40 }) => (
  <>
    {[...Array(count)].map((_, i) => {
      // Keep particles on the edges, away from center
      const side = i % 2 === 0;
      const left = side ? `${5 + Math.random() * 25}%` : `${70 + Math.random() * 25}%`;
      
      return (
        <motion.div
          key={i}
          className="hero-particle"
          style={{
            left,
            top: `${Math.random() * 100}%`,
          }}
          animate={{
            y: [0, -30, 0],
            opacity: [0.2, 0.8, 0.2],
            scale: [1, 1.5, 1],
          }}
          transition={{
            duration: 3 + Math.random() * 2,
            repeat: Infinity,
            delay: Math.random() * 2,
            ease: 'easeInOut',
          }}
        />
      );
    })}
  </>
);

// Crosshair Target - positioned on corners
const CrosshairTarget = ({ className, size = 96, delay = 0 }) => (
  <motion.div
    className={`hero-crosshair ${className}`}
    initial={{ opacity: 0, scale: 0.5 }}
    animate={{
      opacity: 1,
      scale: [1, 1.1, 1],
      rotate: [0, 180, 360],
    }}
    transition={{
      opacity: { delay, duration: 0.5 },
      scale: { duration: 8, repeat: Infinity, ease: 'linear' },
      rotate: { duration: 8, repeat: Infinity, ease: 'linear' },
    }}
    style={{ width: size, height: size }}
  >
    <div className="hero-crosshair-outer" />
    <motion.div 
      className="hero-crosshair-inner"
      animate={{ scale: [1, 1.2, 1] }}
      transition={{ duration: 2, repeat: Infinity }}
    />
    <div className="hero-crosshair-dot" />
    <div className="hero-crosshair-line-v" />
    <div className="hero-crosshair-line-h" />
  </motion.div>
);

// Floating Data Points - on edges only
const DataPoints = ({ count = 6 }) => (
  <>
    {[...Array(count)].map((_, i) => {
      // Position on left or right edges only
      const onLeft = i % 2 === 0;
      const left = onLeft ? `${5 + Math.random() * 20}%` : `${75 + Math.random() * 20}%`;
      
      return (
        <motion.div
          key={`data-${i}`}
          className="hero-data-point"
          style={{
            left,
            top: `${15 + Math.random() * 70}%`,
          }}
          initial={{ opacity: 0, scale: 0 }}
          animate={{ 
            opacity: [0, 1, 0],
            scale: [0.5, 1, 0.5],
          }}
          transition={{
            duration: 4,
            repeat: Infinity,
            delay: i * 0.5,
          }}
        />
      );
    })}
  </>
);

// Scan Line Effect
const ScanLine = () => (
  <motion.div
    className="hero-scan-line"
    animate={{
      top: ['0%', '100%'],
    }}
    transition={{
      duration: 4,
      repeat: Infinity,
      ease: 'linear',
    }}
  />
);

const Scene3DCSS = () => {
  return (
    <div className="hero-scene3d">
      {/* Animated Grid Floor */}
      <div className="hero-grid-perspective">
        <motion.div
          className="hero-grid-floor"
          animate={{
            backgroundPosition: ['0px 0px', '0px 60px'],
          }}
          transition={{
            duration: 2,
            repeat: Infinity,
            ease: 'linear',
          }}
        />
      </div>

      {/* Floating Particles - on edges only */}
      <FloatingParticles count={40} />

      {/* Crosshair Targets - positioned on corners to not obstruct C4 */}
      <CrosshairTarget className="hero-crosshair--primary" size={72} delay={0.5} />
      <CrosshairTarget className="hero-crosshair--secondary" size={48} delay={1} />

      {/* Floating Data Points - on edges */}
      <DataPoints count={6} />

      {/* Scan Line Effect */}
      <ScanLine />
    </div>
  );
};

export default Scene3DCSS;
