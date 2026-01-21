// Timer - Visual countdown timer for challenges
import React, { useState, useEffect, useCallback, useRef } from 'react';

/**
 * Timer Component
 * 
 * Countdown timer with visual states:
 * - Normal (white): > 10 seconds
 * - Warning (yellow): 5-10 seconds
 * - Critical (red): < 5 seconds with pulse animation
 * 
 * @param {number} duration - Total duration in seconds
 * @param {function} onComplete - Callback when timer reaches 0
 * @param {boolean} running - Whether the timer is active
 * @param {function} onTick - Optional callback on each tick with remaining time
 */
const Timer = ({ 
  duration = 20, 
  onComplete, 
  running = true,
  onTick 
}) => {
  const [timeLeft, setTimeLeft] = useState(duration);
  const intervalRef = useRef(null);

  // Get timer status for styling
  const getTimerStatus = () => {
    if (timeLeft <= 5) return 'critical';
    if (timeLeft <= 10) return 'warning';
    return 'normal';
  };

  // Format time display
  const formatTime = (seconds) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    
    if (mins > 0) {
      return `${mins}:${secs.toString().padStart(2, '0')}`;
    }
    return `${secs}s`;
  };

  // Timer logic
  useEffect(() => {
    if (!running) {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
      return;
    }

    intervalRef.current = setInterval(() => {
      setTimeLeft(prev => {
        const newTime = prev - 1;
        
        if (onTick) {
          onTick(newTime);
        }
        
        if (newTime <= 0) {
          clearInterval(intervalRef.current);
          if (onComplete) {
            onComplete();
          }
          return 0;
        }
        
        return newTime;
      });
    }, 1000);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [running, onComplete, onTick]);

  // Reset timer when duration changes
  useEffect(() => {
    setTimeLeft(duration);
  }, [duration]);

  const status = getTimerStatus();

  return (
    <div className={`timer timer--${status}`}>
      <TimerIcon className="timer__icon" />
      <span className="timer__value">{formatTime(timeLeft)}</span>
    </div>
  );
};

// Timer Icon
const TimerIcon = ({ className }) => (
  <svg 
    className={className} 
    viewBox="0 0 24 24" 
    fill="none" 
    stroke="currentColor" 
    strokeWidth="2" 
    strokeLinecap="round" 
    strokeLinejoin="round"
  >
    <circle cx="12" cy="12" r="10" />
    <polyline points="12 6 12 12 16 14" />
  </svg>
);

export default Timer;
