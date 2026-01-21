/**
 * AimSceneContent - AIM Challenge scene elements (no Canvas wrapper)
 * 
 * Extracted from AimTraining3D to work with SharedCanvas.
 * Contains: PeekTrainingScene, TargetManager, lighting.
 */
import React, { useRef, useCallback, useState, useEffect } from 'react';

import PeekTrainingScene from './PeekTrainingScene';
import ImageTarget, { getRandomSpeed } from './ImageTarget';
import useAimTrainingStore, { TEST_STATES, AIM_CONFIG } from '../../../../hooks/useAimTrainingStore';

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
    
    const reactionTime = hitData.reactionTime || (Date.now() - target.spawnTime);
    
    registerHit({
      worldPosition: hitData.worldPosition,
      localX: 0,
      localY: hitData.isHeadshot ? 2 : 1,
      localZ: 0,
      reactionTime: reactionTime,
    });
    
    setActiveTargets(prev => prev.filter(t => t.id !== targetId));
    
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
    
    if (targetsSpawned === 0) {
      spawnPeekTarget();
    }
    
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

// Click handler for aim challenge
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

/**
 * AimSceneContent Component
 * 
 * Scene content for the AIM challenge (no Canvas wrapper)
 */
const AimSceneContent = ({ position = [0, 0, 0], visible = true, enableLights = true }) => {
  const { testState } = useAimTrainingStore();
  const isPlaying = testState === TEST_STATES.PLAYING;

  return (
    <group position={position} visible={visible}>
      {/* Scene Environment */}
      <PeekTrainingScene enableLights={enableLights} />
      
      {/* Target Manager (only when playing) */}
      {isPlaying && <TargetManager />}
      
      {/* Click Handler - only active when visible/playing */}
      {visible && <ClickHandler />}
    </group>
  );
};

export default AimSceneContent;
