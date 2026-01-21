/**
 * HeroSceneContent - Hero scene elements (no Canvas wrapper)
 * 
 * Extracted from SceneOrchestrator to work with SharedCanvas.
 * Contains: C4 Hologram model, ambient particles, and lighting.
 * 
 * Features:
 * - Holographic C4 with custom shader (Fresnel, scanlines, glitch)
 * - Mouse-reactive parallax and dynamic lighting
 * - Morphing transition from hologram to PBR
 * - Dolly zoom to LCD screen
 */
import React, { useRef, useEffect, useMemo, useState } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';
import C4Explosive from './C4Explosive';
import C4Hologram from './C4Hologram';
import { useLanding, LANDING_STEPS } from '../../LandingContext';

// ═══════════════════════════════════════════════════════════════════════════
// AMBIENT PARTICLES - Floating holographic particles for depth
// ═══════════════════════════════════════════════════════════════════════════
const AmbientParticles = ({ hologramMode = true }) => {
  const particlesRef = useRef();
  const count = 150;
  
  const { positions, colors } = useMemo(() => {
    const pos = new Float32Array(count * 3);
    const col = new Float32Array(count * 3);
    
    const cyanColor = new THREE.Color(0x00D4DE);
    const blueColor = new THREE.Color(0x0D80FF);
    const purpleColor = new THREE.Color(0x8B5CF6);
    
    for (let i = 0; i < count; i++) {
      // Position - wider distribution with depth
      pos[i * 3] = (Math.random() - 0.5) * 20;
      pos[i * 3 + 1] = (Math.random() - 0.5) * 12;
      pos[i * 3 + 2] = (Math.random() - 0.5) * 15 - 5;
      
      // Color variation for hologram look
      const colorChoice = Math.random();
      let color;
      if (colorChoice < 0.5) {
        color = cyanColor;
      } else if (colorChoice < 0.8) {
        color = blueColor;
      } else {
        color = purpleColor;
      }
      
      col[i * 3] = color.r;
      col[i * 3 + 1] = color.g;
      col[i * 3 + 2] = color.b;
    }
    return { positions: pos, colors: col };
  }, []);
  
  useFrame((state) => {
    if (particlesRef.current) {
      particlesRef.current.rotation.y = state.clock.elapsedTime * 0.02;
      particlesRef.current.rotation.x = Math.sin(state.clock.elapsedTime * 0.01) * 0.1;
      
      // Pulse opacity for hologram effect
      const pulse = Math.sin(state.clock.elapsedTime * 2) * 0.2 + 0.6;
      particlesRef.current.material.opacity = pulse;
    }
  });
  
  return (
    <points ref={particlesRef}>
      <bufferGeometry>
        <bufferAttribute
          attach="attributes-position"
          count={count}
          array={positions}
          itemSize={3}
        />
        <bufferAttribute
          attach="attributes-color"
          count={count}
          array={colors}
          itemSize={3}
        />
      </bufferGeometry>
      <pointsMaterial
        size={0.04}
        transparent
        opacity={0.6}
        sizeAttenuation
        blending={THREE.AdditiveBlending}
        vertexColors
      />
    </points>
  );
};

/**
 * HeroSceneContent Component
 * 
 * @param {Object} mousePos - Mouse position for C4 interaction
 * @param {boolean} useHologram - Enable holographic shader mode (default: true)
 */
const HeroSceneContent = ({ mousePos = { x: 0, y: 0 }, useHologram = true }) => {
  const { currentStep, triggerMaterialize, setTriggerMaterialize } = useLanding();
  const c4Ref = useRef();
  const c4HologramRef = useRef();
  const [hologramActive] = useState(useHologram);

  // Listen for materialize trigger (from button click)
  useEffect(() => {
    if (triggerMaterialize && hologramActive && c4HologramRef.current) {
      console.log('🎬 Iniciando materialización Subnautica (triggered by button)...');
      c4HologramRef.current.materialize();
      // Reset the trigger
      setTriggerMaterialize(false);
    }
  }, [triggerMaterialize, hologramActive, setTriggerMaterialize]);

  // Listen for step changes to handle reset
  useEffect(() => {
    if (currentStep === LANDING_STEPS.HERO) {
      if (hologramActive && c4HologramRef.current) {
        c4HologramRef.current.reset();
      } else if (c4Ref.current) {
        c4Ref.current.resetCamera();
      }
    }
  }, [currentStep, hologramActive]);

  // Callback when hologram transition completes
  const handleTransitionComplete = () => {
    console.log('✨ Materialización completada');
  };

  return (
    <>
      {/* ═══════════════════════════════════════════════════════════════════════
          LIGHTING SETUP - Minimal for authentic hologram look
          Hologram should be self-illuminated, not lit by scene lights
          ═══════════════════════════════════════════════════════════════════════ */}
      
      {/* Very low ambient - hologram provides its own light */}
      <ambientLight intensity={0.1} color="#001122" />
      
      {/* Subtle environmental lights for depth */}
      <pointLight 
        position={[10, 5, 5]} 
        intensity={0.3} 
        color="#003366"
        distance={25}
      />
      
      <pointLight 
        position={[-8, 3, 3]} 
        intensity={0.2} 
        color="#001144"
        distance={20}
      />
      
      {/* ═══════════════════════════════════════════════════════════════════════
          AMBIENT PARTICLES - Holographic floating particles
          ═══════════════════════════════════════════════════════════════════════ */}
      <AmbientParticles hologramMode={hologramActive} />
      
      {/* ═══════════════════════════════════════════════════════════════════════
          C4 MODEL - Large hologram in background
          ═══════════════════════════════════════════════════════════════════════ */}
      {hologramActive ? (
        <C4Hologram 
          ref={c4HologramRef}
          mousePos={mousePos}
          onTransitionComplete={handleTransitionComplete}
        />
      ) : (
        <C4Explosive 
          ref={c4Ref}
          mousePos={mousePos} 
        />
      )}
    </>
  );
};

export default HeroSceneContent;
