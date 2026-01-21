// ProgressBar - Visual indicator for 4 challenges
import React from 'react';
import { useLanding, LANDING_STEPS } from '../LandingContext';

/**
 * ProgressBar Component
 * 
 * Shows the user's progress through the 4 challenges.
 * Each step can be: completed, active, or pending.
 */
const ProgressBar = () => {
  const { currentStep } = useLanding();
  
  const steps = [
    { id: LANDING_STEPS.AIM, label: 'Aim' },
    { id: LANDING_STEPS.ECONOMY, label: 'Economy' },
    { id: LANDING_STEPS.GRENADE, label: 'Grenades' },
    { id: LANDING_STEPS.GAMESENSE, label: 'Game Sense' },
  ];
  
  const stepOrder = [LANDING_STEPS.AIM, LANDING_STEPS.ECONOMY, LANDING_STEPS.GRENADE, LANDING_STEPS.GAMESENSE];
  const currentIndex = stepOrder.indexOf(currentStep);

  const getStepStatus = (stepId) => {
    const stepIndex = stepOrder.indexOf(stepId);
    
    if (stepIndex < currentIndex) return 'completed';
    if (stepIndex === currentIndex) return 'active';
    return 'pending';
  };

  return (
    <div className="progress-bar">
      {steps.map((step, index) => {
        const status = getStepStatus(step.id);
        
        return (
          <div
            key={step.id}
            className={`progress-bar__step progress-bar__step--${status}`}
            title={step.label}
          />
        );
      })}
      <span className="progress-bar__label">
        {currentIndex + 1}/{steps.length}
      </span>
    </div>
  );
};

export default ProgressBar;
