/**
 * LandingPage - Stage-driven immersive experience
 * 
 * Flow: HERO → CHAT_DEMO → ECONOMY → GRENADE → GAMESENSE → VERDICT
 * 
 * Main orchestrator with:
 * - Finite State Machine for stage progression
 * - Framer Motion for seamless transitions
 * - Progressive agent materialization
 */
import React, { Suspense, useMemo } from 'react';
import { Canvas } from '@react-three/fiber';
import { EffectComposer, Bloom, Vignette } from '@react-three/postprocessing';
import { AnimatePresence, motion } from 'framer-motion';

// Context
import { LandingProvider, useLanding, STAGES } from './LandingContext';

// Sections
import HeroSection from './sections/00-Hero/HeroSection';
import ChallengeSection from './sections/02-ChallengeWrapper/ChallengeSection';
import ChatDemoSection from './sections/03-ChatDemo/ChatDemoSection';
import VerdictSection from './sections/99-Verdict/VerdictSection';

// Challenge components (no AIM)
import EconomyChallenge from './challenges/economy-decision/EconomyChallenge';
import GrenadeChallenge from './challenges/grenade-lineup/GrenadeChallenge';
import GameSenseChallenge from './challenges/game-sense/GameSenseChallenge';

// Agent model
import AgentModel from './core/agent/AgentModel';

// Background Effects
import BackgroundEffects from './core/effects/BackgroundEffects';

// Styles
import '../../styles/Landing/landing.css';
import '../../styles/Landing/sections/layout.css';

// Stage transition variants for Framer Motion
const stageVariants = {
  initial: (direction) => ({
    opacity: 0,
    x: direction > 0 ? 100 : -100,
    scale: 0.95,
  }),
  animate: {
    opacity: 1,
    x: 0,
    scale: 1,
    transition: {
      duration: 0.6,
      ease: [0.22, 1, 0.36, 1],
    },
  },
  exit: (direction) => ({
    opacity: 0,
    x: direction > 0 ? -100 : 100,
    scale: 0.95,
    transition: {
      duration: 0.4,
      ease: [0.22, 1, 0.36, 1],
    },
  }),
};

/**
 * StageRenderer - Renders the current stage content
 */
const StageRenderer = ({ stage }) => {
  switch (stage) {
    case STAGES.HERO:
      return <HeroSection />;
    
    case STAGES.CHAT_DEMO:
      return <ChatDemoSection />;
    
    case STAGES.ECONOMY:
      return (
        <ChallengeSection
          id="economy"
          title="Gestión Económica"
          description="Toma decisiones tácticas de compra"
          stepNumber="01"
        >
          <EconomyChallenge />
        </ChallengeSection>
      );
    
    case STAGES.GRENADE:
      return (
        <ChallengeSection
          id="grenade"
          title="Lineups de Granadas"
          description="Demuestra tu conocimiento de utilidades"
          stepNumber="02"
        >
          <GrenadeChallenge />
        </ChallengeSection>
      );
    
    case STAGES.GAMESENSE:
      return (
        <ChallengeSection
          id="gamesense"
          title="Inteligencia Táctica"
          description="Toma decisiones bajo presión"
          stepNumber="03"
        >
          <GameSenseChallenge />
        </ChallengeSection>
      );
    
    case STAGES.VERDICT:
      return <VerdictSection />;
    
    default:
      return <HeroSection />;
  }
};

/**
 * LandingPageContent - Main content with stage-based rendering
 */
const LandingPageContent = () => {
  const { 
    currentStage, 
    transitionDirection, 
    completedChallenges, 
    agentRevealProgress 
  } = useLanding();

  const stageContent = useMemo(() => (
    <StageRenderer stage={currentStage} />
  ), [currentStage]);

  return (
    <div className="landing-page landing-page--stage">
      {/* Ambient Background Effects */}
      <BackgroundEffects />

      {/* Fixed 3D Agent (right side) - Always visible */}
      <div className="agent-canvas-container">
        <Canvas
          dpr={[1, 2]}
          camera={{ position: [0, 0.5, 5], fov: 60 }}
          gl={{ antialias: true, alpha: true }}
        >
          <ambientLight intensity={1.2} />
          <directionalLight position={[5, 5, 5]} intensity={1.5} />
          <directionalLight position={[-5, 3, -2]} intensity={0.8} color="#ffffff" />
          <pointLight position={[0, 2, 3]} intensity={0.8} color="#00ffff" />
          
          <Suspense fallback={null}>
            <AgentModel 
              completedChallenges={completedChallenges}
              revealProgress={agentRevealProgress}
              position={[0, -2, 0]}
              scale={2}
              rotation={[0, -Math.PI * 0.15, 0]}
            />
          </Suspense>

          <EffectComposer>
            <Bloom 
              luminanceThreshold={0.6} 
              luminanceSmoothing={0.5} 
              intensity={0.5} 
            />
            <Vignette offset={0.3} darkness={0.5} />
          </EffectComposer>
        </Canvas>
      </div>

      {/* Stage Content with AnimatePresence */}
      <div className="landing-stage-container">
        <AnimatePresence mode="wait" custom={transitionDirection}>
          <motion.div
            key={currentStage}
            className="landing-stage"
            custom={transitionDirection}
            variants={stageVariants}
            initial="initial"
            animate="animate"
            exit="exit"
          >
            {stageContent}
          </motion.div>
        </AnimatePresence>
      </div>
    </div>
  );
};

/**
 * LandingPage - Entry component with provider
 */
export const LandingPage = () => {
  return (
    <LandingProvider>
      <LandingPageContent />
    </LandingProvider>
  );
};

export default LandingPage;
