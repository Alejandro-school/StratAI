import React, { createContext, useContext, useState, useCallback } from 'react';

// Define steps constants
export const LANDING_STEPS = {
  HERO: 'hero',
  NICKNAME: 'nickname',
  AIM: 'aim',
  ECONOMY: 'economy',
  GRENADE: 'grenade',
  GAMESENSE: 'gamesense',
  VERDICT: 'verdict'
};

const LandingContext = createContext();

export const LandingProvider = ({ children }) => {
  const [currentStep, setCurrentStep] = useState(LANDING_STEPS.HERO);
  const [showWarpTransition, setShowWarpTransition] = useState(false);
  const [isTransitioning, setIsTransitioning] = useState(false);
  const [isEvaporating, setIsEvaporating] = useState(false); // For smoke/evaporate effect on hero text
  const [triggerMaterialize, setTriggerMaterialize] = useState(false); // Trigger C4 materialization
  
  // User state
  const [nickname, setNickname] = useState('');
  const [isArmed, setIsArmed] = useState(false);
  const [results, setResults] = useState({});
  const [skipped, setSkipped] = useState(false);

  const goToStep = useCallback((step, delay = 0) => {
    if (delay > 0) {
      setIsTransitioning(true);
      setTimeout(() => {
        setCurrentStep(step);
        setIsTransitioning(false);
      }, delay);
    } else {
      setCurrentStep(step);
      // If we are just switching instantly, we might not want to toggle isTransitioning to true and back 
      // unless needed. But for consistency:
      setIsTransitioning(false); 
    }
  }, []);

  // Save results for a specific challenge
  const saveResult = useCallback((key, data) => {
    setResults(prev => ({
      ...prev,
      [key]: data
    }));
  }, []);

  // Navigate to the next challenge automatically
  const nextChallenge = useCallback(() => {
    const steps = Object.values(LANDING_STEPS);
    const currentIndex = steps.indexOf(currentStep);
    
    if (currentIndex >= 0 && currentIndex < steps.length - 1) {
      const nextStep = steps[currentIndex + 1];
      goToStep(nextStep);
    }
  }, [currentStep, goToStep]);

  const value = {
    currentStep,
    setCurrentStep,
    showWarpTransition,
    setShowWarpTransition,
    goToStep,
    isTransitioning,
    isEvaporating,
    setIsEvaporating,
    triggerMaterialize,
    setTriggerMaterialize,
    // User state
    nickname,
    setNickname,
    isArmed,
    setIsArmed,
    results,
    setResults,
    skipped,
    setSkipped,
    // Actions
    saveResult,
    nextChallenge
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
