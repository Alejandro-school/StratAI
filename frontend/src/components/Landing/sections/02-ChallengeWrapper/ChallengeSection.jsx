/**
 * ChallengeSection - Clean assessment wrapper for gameplay challenges
 * Communicates "AI evaluating your knowledge" — professional, not gamey
 */
import React, { useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { useLanding, STAGES } from '../../LandingContext';
import { ChevronRight, Check } from 'lucide-react';
import '../../../../styles/Landing/challenges/tokens.css';
import '../../../../styles/Landing/challenges/shared.css';
import '../../../../styles/Landing/sections/challengeSection.css';

// Map challenge ID → next stage
const NEXT_STAGE_MAP = {
  economy: STAGES.GRENADE,
  grenade: STAGES.GAMESENSE,
  gamesense: STAGES.VERDICT,
};

const STEP_INDEX = { economy: 1, grenade: 2, gamesense: 3 };

const ChallengeSection = ({ 
  id,
  title,
  description,
  stepNumber = "00",
  children,
}) => {
  const { completedChallenges, transitionTo } = useLanding();
  const isCompleted = completedChallenges[id]?.completed;
  const step = STEP_INDEX[id] || 1;

  const handleNextSection = useCallback(() => {
    const nextStage = NEXT_STAGE_MAP[id];
    if (nextStage) transitionTo(nextStage);
  }, [id, transitionTo]);

  return (
    <section id={id} className="challenge-section">
      <div className="challenge-section__wrapper">
        {/* Assessment Header */}
        <motion.div 
          className="challenge-section__header"
          initial={{ opacity: 0, y: -8 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.4, ease: 'easeOut' }}
        >
          <div className="challenge-section__header-row">
            {/* Step indicator */}
            <div className="challenge-section__step-info">
              <span className="challenge-section__step-label">Evaluación</span>
              <span className="challenge-section__step-number">
                {step}<span className="challenge-section__step-sep">/</span>3
              </span>
            </div>

            {/* Progress bar */}
            <div className="challenge-section__progress-bar">
              {[1, 2, 3].map(i => (
                <div
                  key={i}
                  className={`challenge-section__progress-segment ${
                    i < step ? 'completed' : i === step ? 'active' : ''
                  }`}
                />
              ))}
            </div>

            {/* Completed badge */}
            {isCompleted && (
              <motion.div
                className="challenge-section__status"
                initial={{ opacity: 0, x: 12 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.3 }}
              >
                <Check size={14} />
                <span>Analizado</span>
              </motion.div>
            )}
          </div>

          <div className="challenge-section__header-text">
            <h2 className="challenge-section__title">{title}</h2>
            <p className="challenge-section__description">{description}</p>
          </div>
        </motion.div>

        {/* Challenge Content */}
        <div className="challenge-section__content">
          <div className={`challenge-section__game ${isCompleted ? 'challenge-section__game--completed' : ''}`}>
            {children}
          </div>

          {/* Next section CTA */}
          <AnimatePresence>
            {isCompleted && (
              <motion.div
                className="challenge-section__next-action"
                initial={{ opacity: 0, y: 16 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: 16 }}
                transition={{ duration: 0.4, delay: 0.2 }}
              >
                <motion.button
                  className="challenge-section__next-button"
                  onClick={handleNextSection}
                  whileHover={{ x: 4 }}
                  whileTap={{ scale: 0.98 }}
                >
                  <span>Siguiente evaluación</span>
                  <ChevronRight size={18} />
                </motion.button>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </div>
    </section>
  );
};

export default ChallengeSection;
