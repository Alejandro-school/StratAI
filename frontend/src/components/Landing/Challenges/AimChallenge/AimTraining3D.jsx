/**
 * AimTraining3D - CS2 Peek Training Experience
 * 
 * Simplified version with 2D sprite targets
 */
import React, { Suspense, useRef, useEffect, useCallback, useState } from 'react';
import { Canvas, useThree } from '@react-three/fiber';
import { EffectComposer, Bloom, Vignette } from '@react-three/postprocessing';

import PeekTrainingScene from './PeekTrainingScene';
import ImageTarget, { getRandomSpeed } from './ImageTarget';
import Crosshair from './Crosshair';
import AimTrainingHUD from './AimTrainingHUD';
import useAimTrainingStore, { TEST_STATES, AIM_CONFIG } from '../../../../hooks/useAimTrainingStore';

import './AimTraining3D.css';

// Fixed camera for FPS view - close to wall
const FPSCamera = () => {
  const { camera } = useThree();
  
  useEffect(() => {
    // Position: closer to wall for immersive feel
    camera.position.set(0, 1.6, 1.5);
    camera.lookAt(0, 1.4, -3);
    camera.fov = 70;
    camera.updateProjectionMatrix();
    console.log('📷 Camera positioned at:', camera.position);
  }, [camera]);
  
  return null;
};

// Viewmodel removed for cleaner experience



// Target Manager - spawns sprite targets
const TargetManager = () => {
  const { 
    testState, 
    targetsSpawned, 
    spawnTarget,
    registerHit,
    registerMiss,
    endTest,
  } = useAimTrainingStore();
  
  const [activeTargets, setActiveTargets] = useState([]);
  const spawnTimeoutRef = useRef(null);
  const targetIdRef = useRef(0);
  
  // Spawn a new target
  const spawnPeekTarget = useCallback(() => {
    if (targetsSpawned >= AIM_CONFIG.TOTAL_TARGETS) return;
    
    const id = targetIdRef.current++;
    const side = Math.random() > 0.5 ? 'left' : 'right';
    const speed = getRandomSpeed();
    const targetType = Math.random() > 0.5 ? 'terrorist' : 'agent';
    
    const newTarget = { id, side, speed, targetType, spawnTime: Date.now() };
    
    console.log(`🎯 Spawning target #${targetsSpawned + 1}:`, { side, speed: speed.toFixed(1) });
    
    setActiveTargets(prev => [...prev, newTarget]);
    spawnTarget();
  }, [targetsSpawned, spawnTarget]);
  
  // Handle hit
  const handleHit = useCallback((targetId, hitData) => {
    const target = activeTargets.find(t => t.id === targetId);
    if (!target) return;
    
    // Use reactionTime from hitData (time from visibility to hit) if available
    const reactionTime = hitData.reactionTime || (Date.now() - target.spawnTime);
    
    registerHit({
      worldPosition: hitData.worldPosition,
      localX: 0,
      localY: hitData.isHeadshot ? 2 : 1,
      localZ: 0,
      reactionTime: reactionTime, // Pass the accurate reaction time
    });
    
    setActiveTargets(prev => prev.filter(t => t.id !== targetId));
    
    // Check complete
    if (targetsSpawned >= AIM_CONFIG.TOTAL_TARGETS && activeTargets.length <= 1) {
      setTimeout(() => endTest(), 500);
    }
  }, [activeTargets, targetsSpawned, registerHit, endTest]);
  
  // Handle miss
  const handleMiss = useCallback((targetId) => {
    setActiveTargets(prev => prev.filter(t => t.id !== targetId));
    registerMiss({ x: 0, y: 0, z: 0 });
    
    if (targetsSpawned >= AIM_CONFIG.TOTAL_TARGETS && activeTargets.length <= 1) {
      setTimeout(() => endTest(), 500);
    }
  }, [activeTargets.length, targetsSpawned, registerMiss, endTest]);
  
  // Spawn loop
  useEffect(() => {
    if (testState !== TEST_STATES.PLAYING) return;
    
    // Initial spawn
    if (targetsSpawned === 0) {
      spawnPeekTarget();
    }
    
    // Schedule next
    const delay = 1500 + Math.random() * 1000;
    
    spawnTimeoutRef.current = setTimeout(() => {
      if (targetsSpawned < AIM_CONFIG.TOTAL_TARGETS) {
        spawnPeekTarget();
      }
    }, delay);
    
    return () => {
      if (spawnTimeoutRef.current) clearTimeout(spawnTimeoutRef.current);
    };
  }, [testState, targetsSpawned, spawnPeekTarget]);
  
  return (
    <>
      {activeTargets.map(target => (
        <ImageTarget
          key={target.id}
          id={target.id}
          side={target.side}
          speed={target.speed}
          targetType={target.targetType}
          wallPosition={[0, 0, -3]}
          onHit={(hitData) => handleHit(target.id, hitData)}
          onMiss={() => handleMiss(target.id)}
          isActive={true}
        />
      ))}
    </>
  );
};

// Click handler
const ClickHandler = () => {
  const { triggerRecoil, registerClick, testState } = useAimTrainingStore();
  
  useEffect(() => {
    const handleClick = () => {
      if (testState === TEST_STATES.PLAYING) {
        triggerRecoil();
        registerClick();
      }
    };
    
    window.addEventListener('click', handleClick);
    return () => window.removeEventListener('click', handleClick);
  }, [testState, triggerRecoil, registerClick]);
  
  return null;
};

// Main Component
const AimTraining3D = () => {
  const [isOverTarget] = useState(false);
  const { testState, resetTest } = useAimTrainingStore();
  const containerRef = useRef(null);
  
  useEffect(() => {
    resetTest();
  }, [resetTest]);
  
  const isPlaying = testState === TEST_STATES.PLAYING;
  
  return (
    <div 
      ref={containerRef}
      className={`aim-training-3d ${isPlaying ? 'aim-training-3d--playing' : ''}`}
    >
      <Canvas
        dpr={[1, 2]}
        gl={{
          antialias: true,
          alpha: false,
          powerPreference: 'high-performance',
        }}
        camera={{ fov: 75, near: 0.1, far: 100 }}
        shadows
        style={{ pointerEvents: isPlaying ? 'auto' : 'none' }}
      >
        <color attach="background" args={['#87ceeb']} />
        
        <Suspense fallback={null}>
          <FPSCamera />
          <PeekTrainingScene />
          
          {isPlaying && <TargetManager />}
          
          <ClickHandler />
          
          <EffectComposer disableNormalPass>
            <Bloom
              luminanceThreshold={0.7}
              intensity={0.4}
              radius={0.5}
            />
            <Vignette offset={0.3} darkness={0.5} />
          </EffectComposer>
        </Suspense>
      </Canvas>
      
      {isPlaying && (
        <Crosshair isOverTarget={isOverTarget} />
      )}
      
      <AimTrainingHUD />
    </div>
  );
};

export default AimTraining3D;
