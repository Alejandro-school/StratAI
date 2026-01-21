/**
 * TransitionWarp - Minimal transition controller
 * 
 * Simply handles the timing for transitioning between scenes.
 * No visual effects - just smooth camera movement and scene approach.
 */
import { useEffect, useRef } from 'react';

const TransitionWarp = ({ isActive, onComplete, duration = 3500 }) => {
  const timerRef = useRef(null);
  const onCompleteRef = useRef(onComplete);

  // Keep callback ref fresh to avoid re-triggering effect
  useEffect(() => {
    onCompleteRef.current = onComplete;
  }, [onComplete]);

  // Handle timer lifecycle
  useEffect(() => {
    // If not active, ensure timer is cleared
    if (!isActive) {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
        timerRef.current = null;
      }
      return;
    }

    // Active: Start timer
    console.log(`🚀 TransitionWarp: Starting timer for ${duration}ms`);
    
    // Clear any existing timer just in case (e.g. strict mode remount)
    if (timerRef.current) {
      clearTimeout(timerRef.current);
    }

    timerRef.current = setTimeout(() => {
      console.log('✅ TransitionWarp: Timer complete, calling onComplete');
      if (onCompleteRef.current) {
        onCompleteRef.current();
      }
    }, duration);

    // Cleanup: Clear timer on unmount or if deps change
    return () => {
      if (timerRef.current) {
        console.log('🛑 TransitionWarp: Timer cleared via cleanup');
        clearTimeout(timerRef.current);
        timerRef.current = null;
      }
    };
  }, [isActive, duration]); // Minimal dependencies

  return null;
};

export default TransitionWarp;
