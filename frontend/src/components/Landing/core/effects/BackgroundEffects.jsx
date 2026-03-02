/**
 * BackgroundEffects - Ambient animated background for landing page
 * 
 * Creates subtle breathing glow effects with cyan, purple, and orange colors
 * positioned strategically across the viewport
 */
import React from 'react';
import '../../../../styles/Landing/core/backgroundEffects.css';

const BackgroundEffects = () => {
  return (
    <div className="landing-bg-effects" aria-hidden="true">
      {/* Primary Cyan Glow - Top Left */}
      <div className="landing-bg-glow landing-bg-glow--cyan landing-bg-glow--top-left" />
      
      {/* Purple/Violet Glow - Top Right */}
      <div className="landing-bg-glow landing-bg-glow--purple landing-bg-glow--top-right" />
      
      {/* Orange Glow - Bottom Left */}
      <div className="landing-bg-glow landing-bg-glow--orange landing-bg-glow--bottom-left" />
      
      {/* Cyan Secondary - Bottom Right */}
      <div className="landing-bg-glow landing-bg-glow--cyan-dim landing-bg-glow--bottom-right" />
      
      {/* Center Accent (subtle) */}
      <div className="landing-bg-glow landing-bg-glow--center" />
      
    </div>
  );
};

export default BackgroundEffects;
