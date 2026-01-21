/**
 * Hero - Main Hero component for Landing Page
 * 
 * Replicates the reference design with:
 * - CSS-based 3D scene background
 * - Holographic C4 effect overlays
 * - Gradient overlays for depth
 * - Glow effects (cyan & purple)
 * - Scan lines overlay
 * - HeroContent (left) + AIIndicator (right)
 * - Corner decorations
 * - Version tag
 */
import React from 'react';
import { motion } from 'framer-motion';
import Scene3DCSS from './Scene3DCSS';
import HeroContent from './HeroContent';
import AIIndicator from './AIIndicator';
import '../../../styles/Landing/hero.css';

const Hero = () => {
  return (
    <div className="hero-wrapper">
      {/* 3D Background Scene */}
      <Scene3DCSS />

      {/* Hologram Effect Overlays */}
      <div className="hero-hologram-glow" />
      <div className="hero-hologram-grid" />
      <div className="hero-hologram-scanline" />
      <div className="hero-hologram-vignette" />
      <div className="hero-hologram-noise" />
      <div className="hero-hologram-flicker" />
      <div className="hero-hologram-interference" />

      {/* Gradient Overlays */}
      <div className="hero-gradient-bottom" />
      <div className="hero-gradient-sides" />
      
      {/* Glow Effects */}
      <div className="hero-glow hero-glow--cyan" />
      <div className="hero-glow hero-glow--purple" />

      {/* Scan Lines Overlay */}
      <div className="hero-scanlines" />

      {/* Main Content */}
      <div className="hero-main">
        <div className="hero-container">
          <div className="hero-grid-layout">
            {/* Left Content - Hero */}
            <div className="hero-left">
              <HeroContent />
            </div>

            {/* Right Content - AI Indicator */}
            <div className="hero-right">
              <AIIndicator />
            </div>
          </div>
        </div>
      </div>

      {/* Bottom Gradient Fade */}
      <div className="hero-fade-bottom" />

      {/* Corner Decorations */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 1.5 }}
        className="hero-corner hero-corner--top-left"
      />
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 1.5 }}
        className="hero-corner hero-corner--bottom-right"
      />

      {/* Version Tag */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 2 }}
        className="hero-version"
      >
        v2.4.1 // NEURAL_ENGINE
      </motion.div>
    </div>
  );
};

export default Hero;
