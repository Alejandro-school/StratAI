/**
 * Crosshair - Dynamic aiming reticle with animations
 * 
 * Features:
 * - Follows mouse position
 * - GSAP scale/color animation on target hover
 * - Recoil animation on click (GSAP)
 * - Hitmarker feedback
 */
import React, { useRef, useEffect, useState, useCallback } from 'react';
import gsap from 'gsap';
import useAimTrainingStore from '../../../../hooks/useAimTrainingStore';
import './Crosshair.css';

const Crosshair = ({ isOverTarget = false }) => {
  const crosshairRef = useRef(null);
  const hitmarkerRef = useRef(null);
  const [position, setPosition] = useState({ x: 0, y: 0 });
  
  const { showHitmarker, hitmarkerType } = useAimTrainingStore();
  
  // Track mouse position
  useEffect(() => {
    const handleMouseMove = (e) => {
      setPosition({ x: e.clientX, y: e.clientY });
    };
    
    window.addEventListener('mousemove', handleMouseMove);
    return () => window.removeEventListener('mousemove', handleMouseMove);
  }, []);
  
  // GSAP hover animation
  useEffect(() => {
    if (!crosshairRef.current) return;
    
    if (isOverTarget) {
      gsap.to(crosshairRef.current, {
        scale: 1.3,
        duration: 0.15,
        ease: 'power2.out',
      });
      
      // Animate lines to expand
      gsap.to('.crosshair__line', {
        '--line-gap': '8px',
        duration: 0.15,
        ease: 'power2.out',
      });
    } else {
      gsap.to(crosshairRef.current, {
        scale: 1,
        duration: 0.15,
        ease: 'power2.out',
      });
      
      gsap.to('.crosshair__line', {
        '--line-gap': '4px',
        duration: 0.15,
        ease: 'power2.out',
      });
    }
  }, [isOverTarget]);
  
  // GSAP recoil animation on click
  const triggerRecoil = useCallback(() => {
    if (!crosshairRef.current) return;
    
    gsap.to(crosshairRef.current, {
      y: -8,
      duration: 0.05,
      ease: 'power2.out',
      onComplete: () => {
        gsap.to(crosshairRef.current, {
          y: 0,
          duration: 0.15,
          ease: 'elastic.out(1, 0.5)',
        });
      },
    });
    
    // Expand lines briefly
    gsap.to('.crosshair__line', {
      '--line-gap': '12px',
      duration: 0.05,
      ease: 'power2.out',
      onComplete: () => {
        gsap.to('.crosshair__line', {
          '--line-gap': '4px',
          duration: 0.2,
          ease: 'power2.out',
        });
      },
    });
  }, []);
  
  // Listen for clicks
  useEffect(() => {
    const handleClick = () => triggerRecoil();
    window.addEventListener('click', handleClick);
    return () => window.removeEventListener('click', handleClick);
  }, [triggerRecoil]);
  
  // Hitmarker animation
  useEffect(() => {
    if (showHitmarker && hitmarkerRef.current) {
      gsap.fromTo(hitmarkerRef.current, 
        { scale: 0, opacity: 0 },
        { 
          scale: 1.5, 
          opacity: 1, 
          duration: 0.1,
          ease: 'power2.out',
          onComplete: () => {
            gsap.to(hitmarkerRef.current, {
              scale: 1,
              opacity: 0,
              duration: 0.1,
              ease: 'power2.in',
            });
          }
        }
      );
    }
  }, [showHitmarker]);
  
  return (
    <>
      {/* Main Crosshair */}
      <div 
        ref={crosshairRef}
        className={`crosshair ${isOverTarget ? 'crosshair--active' : ''}`}
        style={{ 
          left: position.x, 
          top: position.y,
        }}
      >
        <div className="crosshair__line crosshair__line--top" />
        <div className="crosshair__line crosshair__line--right" />
        <div className="crosshair__line crosshair__line--bottom" />
        <div className="crosshair__line crosshair__line--left" />
        <div className="crosshair__dot" />
      </div>
      
      {/* Hitmarker */}
      <div 
        ref={hitmarkerRef}
        className={`hitmarker ${hitmarkerType === 'headshot' ? 'hitmarker--headshot' : ''}`}
        style={{ 
          left: position.x, 
          top: position.y,
          opacity: 0,
        }}
      >
        <div className="hitmarker__line hitmarker__line--1" />
        <div className="hitmarker__line hitmarker__line--2" />
        <div className="hitmarker__line hitmarker__line--3" />
        <div className="hitmarker__line hitmarker__line--4" />
      </div>
    </>
  );
};

export default Crosshair;
