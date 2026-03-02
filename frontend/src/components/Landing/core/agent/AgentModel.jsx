/**
 * AgentModel - CT Agent 3D Model with Subnautica-style Reveal Animation
 * 
 * Logic:
 * - A horizontal "light ring" scans from bottom to top
 * - Below the ring: Real textures (revealed)
 * - Above the ring: Hologram effect (unrevealed)
 * - Animation: Smooth lerp when reveal progress changes
 */
import React, { useRef, useEffect } from 'react';
import { useGLTF, useAnimations } from '@react-three/drei';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';

// ═══════════════════════════════════════════════════════════════════════════
// CONFIGURATION
// ═══════════════════════════════════════════════════════════════════════════

const REVEAL_CONFIG = {
  animationDuration: 5.0,
  glowIntensity: 1.2,
  glowWidth: 0.05,
  glowColor: [0.0, 0.8, 0.9],
  modelMinY: -2.0,
  modelMaxY: 2.0,
};

// Animation names from the GLB
const ANIMATIONS = {
  IDLE: 'ct_loadout_pistol01_walkup',
  CELEBRATE: 'celebrate_scuba_female_idle03',
};

// ═══════════════════════════════════════════════════════════════════════════
// REVEAL SHADER MATERIAL
// ═══════════════════════════════════════════════════════════════════════════

const createRevealMaterial = (originalMaterial) => {
  const material = originalMaterial.clone();
  material.transparent = true;
  material.side = THREE.DoubleSide;
  
  material.onBeforeCompile = (shader) => {
    shader.uniforms.time = { value: 0 };
    shader.uniforms.revealProgress = { value: 0 };
    shader.uniforms.modelMinY = { value: REVEAL_CONFIG.modelMinY };
    shader.uniforms.modelMaxY = { value: REVEAL_CONFIG.modelMaxY };
    shader.uniforms.glowColor = { value: new THREE.Color(...REVEAL_CONFIG.glowColor) };
    shader.uniforms.glowIntensity = { value: REVEAL_CONFIG.glowIntensity };
    shader.uniforms.glowWidth = { value: REVEAL_CONFIG.glowWidth };
    
    shader.vertexShader = shader.vertexShader.replace(
      '#include <common>',
      `#include <common>
      varying vec3 vWorldPos;
      varying vec3 vLocalPos;`
    );
    shader.vertexShader = shader.vertexShader.replace(
      '#include <worldpos_vertex>',
      `#include <worldpos_vertex>
      vWorldPos = (modelMatrix * vec4(transformed, 1.0)).xyz;
      vLocalPos = transformed;`
    );
    
    shader.fragmentShader = shader.fragmentShader.replace(
      '#include <common>',
      `#include <common>
      uniform float time;
      uniform float revealProgress;
      uniform float modelMinY;
      uniform float modelMaxY;
      uniform vec3 glowColor;
      uniform float glowIntensity;
      uniform float glowWidth;
      varying vec3 vWorldPos;
      varying vec3 vLocalPos;`
    );
    
    shader.fragmentShader = shader.fragmentShader.replace(
      '#include <dithering_fragment>',
      `#include <dithering_fragment>
      
      float normalizedY = clamp((vWorldPos.y - modelMinY) / (modelMaxY - modelMinY), 0.0, 1.0);
      
      float fresnel2 = pow(1.0 - saturate(dot(normal, normalize(vViewPosition))), 2.5);
      vec3 hologramColor = vec3(0.0, 0.5, 0.8) * 0.2 + fresnel2 * vec3(0.0, 0.8, 1.0) * 0.5;
      hologramColor += step(0.94, sin(vWorldPos.y * 300.0 + time * 2.0)) * vec3(0.0, 0.4, 0.5) * 0.15;
      float hologramOpacity = 0.25 + fresnel2 * 0.4;
      
      if (revealProgress > 0.001) {
        float scanlineY = revealProgress;
        float isRevealed = step(normalizedY, scanlineY - 0.01);
        
        float distToScanline = abs(normalizedY - scanlineY);
        float revealComplete = step(0.99, scanlineY);
        float glowRing = smoothstep(glowWidth, 0.0, distToScanline) * glowIntensity * (1.0 - revealComplete);
        float innerGlow = smoothstep(glowWidth * 0.5, 0.0, distToScanline) * glowIntensity * 0.3 * (1.0 - revealComplete);
        
        vec3 realColor = gl_FragColor.rgb;
        gl_FragColor.rgb = mix(hologramColor, realColor, isRevealed);
        
        float glowMask = 1.0 - isRevealed * 0.5;
        gl_FragColor.rgb += glowColor * (glowRing + innerGlow) * glowMask;
        
        gl_FragColor.a = mix(hologramOpacity, 1.0, isRevealed);
        
        float pulse = sin(time * 8.0) * 0.3 + 0.7;
        gl_FragColor.rgb += glowColor * glowRing * pulse * 0.3;
      } else {
        gl_FragColor.rgb = hologramColor;
        gl_FragColor.a = hologramOpacity;
      }
      `
    );
    
    material.userData.shader = shader;
  };
  
  return material;
};

// ═══════════════════════════════════════════════════════════════════════════
// AGENT MODEL COMPONENT
// ═══════════════════════════════════════════════════════════════════════════

const AgentModel = ({ 
  revealProgress = 0,
  position = [0, -0.5, 0],
  scale = 1,
  rotation = [0, 0, 0],
  completedChallenges = {},
}) => {
  const groupRef = useRef();
  const { scene, animations } = useGLTF('/images/Landing/CT_model.glb');
  const { actions, names } = useAnimations(animations, groupRef);
  
  const currentRevealRef = useRef(0);
  const targetRevealRef = useRef(0);
  const timeRef = useRef(0);
  const materialsRef = useRef(new Map());
  
  useEffect(() => {
    targetRevealRef.current = revealProgress >= 0.99 ? 1.5 : revealProgress;
  }, [revealProgress]);
  
  useEffect(() => {
    if (!scene) return;
    
    scene.traverse((child) => {
      if (child.isMesh && child.material) {
        if (!materialsRef.current.has(child.uuid)) {
          const originalMat = child.material;
          const revealMat = createRevealMaterial(originalMat);
          materialsRef.current.set(child.uuid, {
            original: originalMat,
            reveal: revealMat,
          });
          child.material = revealMat;
        }
      }
    });
  }, [scene]);
  
  const gameSenseCompleted = completedChallenges?.gamesense?.success === true;
  
  useEffect(() => {
    if (!actions || names.length === 0) return;

    const targetAnimName = gameSenseCompleted ? ANIMATIONS.CELEBRATE : ANIMATIONS.IDLE;
    const action = actions[targetAnimName] || actions[names[0]];

    if (action) {
      Object.values(actions).forEach(act => {
        if (act !== action && act.isRunning()) {
          act.fadeOut(0.5);
        }
      });
      
      if (!action.isRunning()) {
        action.fadeIn(0.5).play();
        action.setLoop(THREE.LoopPingPong, Infinity);
      }
    }
  }, [actions, names, gameSenseCompleted]);
  
  useFrame((state, delta) => {
    timeRef.current += delta;
    
    const lerpSpeed = 1.0 / REVEAL_CONFIG.animationDuration;
    const newReveal = THREE.MathUtils.lerp(
      currentRevealRef.current,
      targetRevealRef.current,
      Math.min(delta * lerpSpeed * 3, 1)
    );
    
    if (Math.abs(newReveal - currentRevealRef.current) > 0.0001) {
      currentRevealRef.current = newReveal;
    }
    
    materialsRef.current.forEach(({ reveal }) => {
      if (reveal.userData.shader) {
        reveal.userData.shader.uniforms.time.value = timeRef.current;
        reveal.userData.shader.uniforms.revealProgress.value = currentRevealRef.current;
      }
    });
  });
  
  return (
    <group ref={groupRef} position={position} scale={scale} rotation={rotation}>
      <primitive object={scene} />
    </group>
  );
};

useGLTF.preload('/images/Landing/CT_model.glb');

export default AgentModel;
