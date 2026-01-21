// LandingPage - Main orchestrator component with Shared Canvas
import React, { useState, useEffect, useCallback } from 'react';
import { LandingProvider, useLanding, LANDING_STEPS } from './LandingContext';
import Hero from './Hero/Hero';
import NicknameStep from './Hero/NicknameStep';
import SharedCanvas from './shared/SharedCanvas';
import Crosshair from './Challenges/AimChallenge/Crosshair';
import AimTrainingHUD from './Challenges/AimChallenge/AimTrainingHUD';
import EconomyChallenge from './Challenges/EconomyChallenge/EconomyChallenge';
import GrenadeChallenge from './Challenges/GrenadeChallenge/GrenadeChallenge';
import GameSenseChallenge from './Challenges/GameSenseChallenge/GameSenseChallenge';
import Verdict from './Verdict/Verdict';
import TransitionWarp from './shared/TransitionWarp';
import useAimTrainingStore, { TEST_STATES } from '../../hooks/useAimTrainingStore';

// Import base styles
import '../../styles/Landing/landing.css';

/**
 * LandingPageContent
 * 
 * Uses a SINGLE SharedCanvas that persists throughout the experience.
 * Scene content is swapped inside the Canvas, not the Canvas itself.
 * This prevents WebGL context loss during transitions.
 */
const LandingPageContent = () => {
  const { currentStep, showWarpTransition, setShowWarpTransition, goToStep, isTransitioning } = useLanding();
  const { testState, resetTest } = useAimTrainingStore();
  
  // State to control which scene is active inside the SharedCanvas
  const [activeScene, setActiveScene] = useState('hero'); // 'hero' | 'aim' | 'none'

  // Reset AIM test when entering AIM step
  useEffect(() => {
    if (currentStep === LANDING_STEPS.AIM) {
      resetTest();
    }
  }, [currentStep, resetTest]);

  // Handle transition between scenes
  useEffect(() => {
    if (showWarpTransition) {
      // During warp, keep hero visible for camera animation
      setActiveScene('hero');
    } else if (currentStep === LANDING_STEPS.AIM) {
      // After warp completes, switch to AIM scene
      setActiveScene('aim');
    } else {
      // Default to hero scene
      if (!isTransitioning) {
        setActiveScene('hero');
      }
    }
  }, [showWarpTransition, currentStep, isTransitioning]);

  const handleWarpComplete = useCallback(() => {
    console.log('🌀 Warp complete, transitioning to AIM');
    setShowWarpTransition(false);
    goToStep(LANDING_STEPS.AIM, 0);
  }, [setShowWarpTransition, goToStep]);

  // Render current step UI Overlay
  const renderStepUI = () => {
    switch (currentStep) {
      case LANDING_STEPS.HERO:
        return <Hero />;
        
      case LANDING_STEPS.NICKNAME:
        return <NicknameStep />;
      
      case LANDING_STEPS.AIM:
        return null; // AIM has its own HUD
      
      case LANDING_STEPS.ECONOMY:
        return <EconomyChallenge />;
      
      case LANDING_STEPS.GRENADE:
        return <GrenadeChallenge />;
      
      case LANDING_STEPS.GAMESENSE:
        return <GameSenseChallenge />;
      
      case LANDING_STEPS.VERDICT:
        return <Verdict />;
      
      default:
        return null;
    }
  };

  const isAimPlaying = currentStep === LANDING_STEPS.AIM && testState === TEST_STATES.PLAYING;

  console.log('🎯 Landing State:', { currentStep, showWarpTransition, activeScene });

  return (
    <div className="landing-page">
      {/* SINGLE Persistent 3D Canvas - Never unmounts */}
      <SharedCanvas 
        activeScene={activeScene} 
        isTransitioning={showWarpTransition}
      />
      
      {/* AIM Challenge UI Overlays (during transition or when active) */}
      {(currentStep === LANDING_STEPS.AIM || showWarpTransition) && (
        <>
          {isAimPlaying && <Crosshair isOverTarget={false} />}
          <AimTrainingHUD />
        </>
      )}

      {/* UI Overlay - Hide during warp transition */}
      <div className={`ui-orchestrator ${showWarpTransition ? 'hidden' : ''}`} style={{ opacity: showWarpTransition ? 0 : 1, transition: 'opacity 0.5s ease' }}>
        {renderStepUI()}
      </div>

      {/* Transition Timer (no visual effects) */}
      <TransitionWarp 
        isActive={showWarpTransition} 
        onComplete={handleWarpComplete}
        duration={3500}
      />
    </div>
  );
};

/**
 * LandingPage
 * 
 * Main entry component for the landing page.
 * Wraps content with the LandingProvider for state management.
 */
export const LandingPage = () => {
  return (
    <LandingProvider>
      <LandingPageContent />
    </LandingProvider>
  );
};

export default LandingPage;

