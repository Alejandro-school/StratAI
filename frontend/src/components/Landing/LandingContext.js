/**
 * LandingContext - Stage-based state management
 * 
 * Flow: HERO (with nickname) → CHAT_DEMO → ECONOMY → GRENADE → GAMESENSE → VERDICT
 */
import React, { createContext, useContext, useState, useCallback, useMemo, useRef, useEffect } from 'react';

// Stage definitions (ordered progression)
export const STAGES = {
  HERO: 'hero',
  CHAT_DEMO: 'chat_demo',
  ECONOMY: 'economy',
  GRENADE: 'grenade',
  GAMESENSE: 'gamesense',
  VERDICT: 'verdict',
};

// Ordered array for navigation
const STAGE_ORDER = [
  STAGES.HERO,
  STAGES.CHAT_DEMO,
  STAGES.ECONOMY,
  STAGES.GRENADE,
  STAGES.GAMESENSE,
  STAGES.VERDICT,
];

// Challenges that contribute to agent reveal (no AIM)
const CHALLENGE_STAGES = ['economy', 'grenade', 'gamesense'];

const LandingContext = createContext();

export const LandingProvider = ({ children }) => {
  const [currentStage, setCurrentStage] = useState(STAGES.HERO);
  const [transitionDirection, setTransitionDirection] = useState(1);
  const [isTransitioning, setIsTransitioning] = useState(false);
  const timeoutIdsRef = useRef([]);
  
  // User state
  const [nickname, setNickname] = useState('');
  
  // Challenge completion tracking
  const [completedChallenges, setCompletedChallenges] = useState({
    economy: { completed: false, success: false },
    grenade: { completed: false, success: false },
    gamesense: { completed: false, success: false },
  });
  
  // Challenge results
  const [results, setResults] = useState({});
  const [skipped, setSkipped] = useState(false);

  // Agent reveal progress (0-1) - based on 3 challenges now
  const agentRevealProgress = useMemo(() => {
    const successes = CHALLENGE_STAGES.filter(
      stage => completedChallenges[stage]?.success
    ).length;
    return successes / CHALLENGE_STAGES.length;
  }, [completedChallenges]);

  const currentStageIndex = useMemo(() => {
    return STAGE_ORDER.indexOf(currentStage);
  }, [currentStage]);

  const registerTimeout = useCallback((callback, delay) => {
    const id = setTimeout(callback, delay);
    timeoutIdsRef.current.push(id);
    return id;
  }, []);

  useEffect(() => {
    return () => {
      timeoutIdsRef.current.forEach(clearTimeout);
      timeoutIdsRef.current = [];
    };
  }, []);

  // Transition to a specific stage
  const transitionTo = useCallback((targetStage, options = {}) => {
    const targetIndex = STAGE_ORDER.indexOf(targetStage);
    if (targetIndex === -1) {
      console.warn(`Invalid stage: ${targetStage}`);
      return;
    }

    const currentIndex = STAGE_ORDER.indexOf(currentStage);
    const direction = targetIndex > currentIndex ? 1 : -1;
    
    setTransitionDirection(direction);
    setIsTransitioning(true);
    
    registerTimeout(() => {
      setCurrentStage(targetStage);
      registerTimeout(() => {
        setIsTransitioning(false);
      }, options.enterDuration || 600);
    }, options.exitDuration || 400);
  }, [currentStage, registerTimeout]);

  const nextStage = useCallback(() => {
    const nextIndex = currentStageIndex + 1;
    if (nextIndex < STAGE_ORDER.length) {
      transitionTo(STAGE_ORDER[nextIndex]);
    }
  }, [currentStageIndex, transitionTo]);

  const prevStage = useCallback(() => {
    const prevIndex = currentStageIndex - 1;
    if (prevIndex >= 0) {
      transitionTo(STAGE_ORDER[prevIndex]);
    }
  }, [currentStageIndex, transitionTo]);

  // Complete a challenge
  const completeChallenge = useCallback((challengeId, resultData, success = true, autoAdvance = false) => {
    setCompletedChallenges(prev => ({
      ...prev,
      [challengeId]: { completed: true, success },
    }));
    
    if (resultData) {
      setResults(prev => ({
        ...prev,
        [challengeId]: resultData,
      }));
    }
    
    if (autoAdvance) {
      registerTimeout(() => nextStage(), 800);
    }
  }, [nextStage, registerTimeout]);

  const saveResult = useCallback((key, data) => {
    setResults(prev => ({
      ...prev,
      [key]: data
    }));
  }, []);

  // Skip a specific challenge
  const skipChallenge = useCallback((challengeId) => {
    setCompletedChallenges(prev => ({
      ...prev,
      [challengeId]: { completed: true, success: false, skipped: true },
    }));
    setResults(prev => ({
      ...prev,
      [challengeId]: { skipped: true }
    }));
    registerTimeout(() => nextStage(), 300);
  }, [nextStage, registerTimeout]);

  // Skip all challenges and go directly to verdict (from Chat Demo)
  const skipAllChallenges = useCallback(() => {
    setSkipped(true);
    
    setCompletedChallenges(prev => {
      const updated = { ...prev };
      Object.keys(updated).forEach(key => {
        if (!updated[key].completed) {
          updated[key] = { completed: true, success: false, skipped: true };
        }
      });
      return updated;
    });

    transitionTo(STAGES.VERDICT);
  }, [transitionTo]);

  // Start challenges flow (from Chat Demo — goes directly to economy)
  const startChallenges = useCallback(() => {
    transitionTo(STAGES.ECONOMY);
  }, [transitionTo]);

  const value = {
    // Stage state
    currentStage,
    currentStageIndex,
    transitionDirection,
    isTransitioning,
    transitionTo,
    nextStage,
    prevStage,
    
    // Challenge state
    completedChallenges,
    agentRevealProgress,
    completeChallenge,
    skipChallenge,
    skipAllChallenges,
    startChallenges,
    
    // User state
    nickname,
    setNickname,
    results,
    setResults,
    saveResult,
    skipped,
    setSkipped,
    
    // Constants
    STAGES,
    STAGE_ORDER,
  };

  return (
    <LandingContext.Provider value={value}>
      {children}
    </LandingContext.Provider>
  );
};

export const useLanding = () => {
  const context = useContext(LandingContext);
  if (!context) {
    throw new Error('useLanding must be used within a LandingProvider');
  }
  return context;
};
