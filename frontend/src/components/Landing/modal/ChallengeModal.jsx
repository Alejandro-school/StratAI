/**
 * ChallengeModal — Fullscreen overlay wrapping the challenge flow
 *
 * Renders: ChatDemo → Economy → Grenade → GameSense → Verdict
 * Uses the existing LandingProvider FSM for stage management.
 * Includes the AgentModel in a right panel for desktop.
 */
import React, { Suspense, lazy, useCallback, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { X } from 'lucide-react';
import { LandingProvider, useLanding, STAGES } from '../LandingContext';
import '../../../styles/Landing/modal/challengeModal.css';

// Sections (kept synchronous since the modal is already lazy-loaded)
import ChatDemoSection from '../sections/03-ChatDemo/ChatDemoSection';
import ChallengeSection from '../sections/02-ChallengeWrapper/ChallengeSection';
import VerdictSection from '../sections/99-Verdict/VerdictSection';

// Challenges
import EconomyChallenge from '../challenges/economy-decision/EconomyChallenge';
import GrenadeChallenge from '../challenges/grenade-lineup/GrenadeChallenge';
import GameSenseChallenge from '../challenges/game-sense/GameSenseChallenge';

// Agent model — lazy inside the modal
const AgentModel = lazy(() => import('../core/agent/AgentModel'));

/* ── Stage renderer ──────────────────────────────────────────────────── */
const StageRenderer = () => {
  const { currentStage } = useLanding();

  switch (currentStage) {
    case STAGES.HERO:
    case STAGES.CHAT_DEMO:
      return <ChatDemoSection />;

    case STAGES.ECONOMY:
      return (
        <ChallengeSection
          id="economy"
          title="Gestión Económica"
          description="Evalúa tu toma de decisiones de compra en rondas clave."
          stepNumber="01"
        >
          <EconomyChallenge />
        </ChallengeSection>
      );

    case STAGES.GRENADE:
      return (
        <ChallengeSection
          id="grenade"
          title="Uso de Utilidad"
          description="Demuestra tu conocimiento de lineups y granadas."
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
          description="Toma decisiones bajo presión en situaciones de clutch."
          stepNumber="03"
        >
          <GameSenseChallenge />
        </ChallengeSection>
      );

    case STAGES.VERDICT:
      return <VerdictSection />;

    default:
      return null;
  }
};

/* ── Inner content (needs LandingContext) ─────────────────────────────── */
const ModalInner = ({ onClose }) => {
  const { currentStage, agentRevealProgress } = useLanding();
  const showAgent = currentStage !== STAGES.VERDICT;

  // ESC key handler
  useEffect(() => {
    const onKey = (e) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  return (
    <motion.div
      className="challenge-modal"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.35 }}
      role="dialog"
      aria-modal="true"
      aria-label="Challenge assessment"
    >
      {/* Backdrop */}
      <div className="challenge-modal__backdrop" onClick={onClose} />

      {/* Close button */}
      <button className="challenge-modal__close" onClick={onClose} aria-label="Close">
        <X size={20} />
      </button>

      {/* Content area */}
      <div className="challenge-modal__layout">
        {/* Left: Stage content */}
        <div className="challenge-modal__content">
          <AnimatePresence mode="wait">
            <motion.div
              key={currentStage}
              className="challenge-modal__stage"
              initial={{ opacity: 0, y: 16 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -16 }}
              transition={{ duration: 0.35 }}
            >
              <StageRenderer />
            </motion.div>
          </AnimatePresence>
        </div>

        {/* Right: Agent model (desktop only) */}
        {showAgent && (
          <div className="challenge-modal__agent">
            <Suspense fallback={null}>
              <AgentModel revealProgress={agentRevealProgress} />
            </Suspense>
          </div>
        )}
      </div>
    </motion.div>
  );
};

/* ── Exported wrapper (provides LandingContext) ───────────────────────── */
const ChallengeModal = ({ onClose }) => {
  const stableClose = useCallback(() => onClose(), [onClose]);

  return (
    <LandingProvider>
      <AnimatePresence>
        <ModalInner onClose={stableClose} />
      </AnimatePresence>
    </LandingProvider>
  );
};

export default ChallengeModal;
