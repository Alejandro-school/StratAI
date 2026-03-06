/**
 * BackgroundEffects - Ambient background with CT (teal) and T (warm earth) themed glows
 */
import React from 'react';
import '../../../../styles/Landing/core/backgroundEffects.css';

const BackgroundEffects = () => {
  return (
    <div className="landing-bg-effects" aria-hidden="true">
      {/* T-side warm earth glow — Top Left */}
      <div className="landing-bg-glow landing-bg-glow--t-warm landing-bg-glow--top-left" />
      
      {/* CT-side teal glow — Top Right */}
      <div className="landing-bg-glow landing-bg-glow--ct-teal landing-bg-glow--top-right" />
      
      {/* Center Accent (subtle blend) */}
      <div className="landing-bg-glow landing-bg-glow--center" />
    </div>
  );
};

export default BackgroundEffects;
