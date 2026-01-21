// ChallengeWrapper - Common layout for all challenges
import React from 'react';
import ProgressBar from './ProgressBar';
import Timer from './Timer';
import '../../../styles/Landing/challenges.css';

/**
 * ChallengeWrapper Component
 * 
 * Provides consistent layout for all challenge screens:
 * - Header with brand, challenge type, progress bar, and timer
 * - Content area for the challenge
 * - Optional footer for stats
 * 
 * @param {string} type - Challenge type label (e.g., "MECÁNICA", "INTELECTO")
 * @param {string} title - Challenge title
 * @param {string} subtitle - Challenge description
 * @param {number} timerDuration - Timer duration in seconds
 * @param {boolean} timerRunning - Whether timer is active
 * @param {function} onTimerComplete - Callback when timer ends
 * @param {ReactNode} children - Challenge content
 * @param {ReactNode} footer - Optional footer content
 * @param {boolean} showTimer - Whether to show the timer
 */
const ChallengeWrapper = ({
  type,
  title,
  subtitle,
  timerDuration = 20,
  timerRunning = true,
  onTimerComplete,
  children,
  footer,
  showTimer = true,
  fullScreen = false,
}) => {
  return (
    <div className={`challenge ${fullScreen ? 'challenge--fullscreen' : ''}`}>
      {/* Header */}
      <header className="challenge__header">
        <div className="challenge__header-left">
          {/* Brand */}
          <div className="challenge__brand">
            <div className="challenge__brand-logo">S</div>
            <span>StratAI</span>
          </div>
          
          {/* Challenge Type */}
          <span className="challenge__type">{type}</span>
        </div>
        
        <div className="challenge__header-right">
          {/* Progress Bar */}
          <ProgressBar />
          
          {/* Timer */}
          {showTimer && (
            <Timer 
              duration={timerDuration}
              running={timerRunning}
              onComplete={onTimerComplete}
            />
          )}
        </div>
      </header>
      
      {/* Content */}
      <div className="challenge__content">
        {title && <h2 className="challenge__title">{title}</h2>}
        {subtitle && <p className="challenge__subtitle">{subtitle}</p>}
        
        {children}
      </div>
      
      {/* Footer */}
      {footer && (
        <footer className="challenge__footer">
          {footer}
        </footer>
      )}
    </div>
  );
};

// Stat component for footer
export const ChallengeStat = ({ value, label }) => (
  <div className="challenge__stat">
    <span className="challenge__stat-value">{value}</span>
    <span className="challenge__stat-label">{label}</span>
  </div>
);

// Completion overlay
export const ChallengeComplete = ({ title, message, children }) => (
  <div className="challenge__complete">
    <div className="challenge__complete-icon">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
        <polyline points="20 6 9 17 4 12" />
      </svg>
    </div>
    <h3 className="challenge__complete-title">{title}</h3>
    <p className="challenge__complete-message">{message}</p>
    {children}
  </div>
);

export default ChallengeWrapper;
