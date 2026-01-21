import React, { useState, useEffect, useRef } from 'react';
import { useLanding } from '../LandingContext';
import { C4SoundManager } from './Scene3D/C4Explosive';
import gsap from 'gsap';
import '../../../styles/Landing/landing.css'; 
import '../../../styles/Landing/nickname.css';

// ═══════════════════════════════════════════════════════════════════════════
// NICKNAME STEP - Agent Identification via C4 Interface
// ═══════════════════════════════════════════════════════════════════════════

const NicknameStep = () => {
  const { nickname, setNickname, isArmed, setIsArmed, setShowWarpTransition } = useLanding();
  const [error, setError] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  // Refs
  const containerRef = useRef(null);
  const inputRef = useRef(null);

  // Initialize sound manager
  useEffect(() => {
    C4SoundManager.init();
  }, []);

  // Entry animation and window click focus management
  useEffect(() => {
    const handleWindowClick = () => {
      try {
        if (inputRef.current) {
          inputRef.current.focus();
          const val = inputRef.current.value || '';
          if (inputRef.current.setSelectionRange) {
            inputRef.current.setSelectionRange(val.length, val.length);
          }
        }
      } catch (e) {
        // Ignore focus errors
      }
    };

    window.addEventListener('click', handleWindowClick);
    
    return () => {
      window.removeEventListener('click', handleWindowClick);
    };
  }, []);

  const handleSubmit = (e) => {
    if (e) e.preventDefault();
    if (isSubmitting || isArmed) return;
    
    if (!nickname.trim()) {
      setError('⚠ AGENT ID REQUIRED');
      if (inputRef.current) {
        gsap.to(inputRef.current, {
          x: [-8, 8, -6, 6, -4, 4, 0],
          duration: 0.5,
          ease: 'power2.out'
        });
      }
      return;
    }
    
    if (nickname.length > 8) {
      setError('⚠ MAX 8 CHARS');
      return;
    }
    
    setError('');
    setIsSubmitting(true);
    setIsArmed(true);
    
    // Play sounds
    C4SoundManager.playRandomKeySound();

    // After short delay, play explosion and trigger warp transition
    setTimeout(() => {
      console.log('💣 Triggering explosion and warp transition');
      C4SoundManager.playExplosion();
      setShowWarpTransition(true);
    }, 800);
  };

  const handleInputChange = (e) => {
    const value = e.target.value.toUpperCase();
    const prevLength = nickname.length;
    const newLength = value.length;
    
    setNickname(value);
    setError('');
    
    // Play sound for both typing and deleting
    if (newLength !== prevLength) {
      C4SoundManager.playRandomKeySound();
    }
  };

  const handleKeyDown = (e) => {
    // Robust Enter detection for mobile
    if (e.key === 'Enter' || e.keyCode === 13) {
      e.preventDefault();
      handleSubmit(e);
      return;
    }
  };

  return (
    <div ref={containerRef} className="landing-step nickname-step">
      <input
        ref={inputRef}
        id="nickname-hidden-input"
        type="text"
        value={nickname}
        onChange={handleInputChange}
        onInput={handleInputChange} // Secondary for mobile
        onKeyDown={handleKeyDown}
        maxLength={8}
        autoFocus
        autoComplete="new-password"
        autoCorrect="off"
        autoCapitalize="none"
        spellCheck="false"
        enterKeyHint="done"
        style={{
          position: 'fixed',
          top: '-9999px', // Way off screen
          left: '-9999px',
          opacity: 0,
          width: '1px',
          height: '1px',
          fontSize: '16px',
          border: 'none',
          outline: 'none'
        }}
      />
    </div>
  );
};

export default NicknameStep;
