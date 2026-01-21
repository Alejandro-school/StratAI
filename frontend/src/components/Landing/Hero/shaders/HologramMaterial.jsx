/**
 * HologramMaterial - Custom shader material for holographic C4 effect
 * 
 * Features:
 * - Fresnel edge glow (neon cyan/blue)
 * - Animated horizontal scanlines
 * - Low opacity on front faces, high on edges
 * - Random glitch interference effect
 * - Mouse-reactive lighting
 * - Smooth transition to PBR material
 */
import React, { useRef, useMemo, useEffect } from 'react';
import { useFrame, extend } from '@react-three/fiber';
import { shaderMaterial } from '@react-three/drei';
import * as THREE from 'three';
import { 
  hologramVertexShader, 
  hologramFragmentShader, 
  hologramUniforms 
} from './hologramShader';

// Create the shader material class
const HologramShaderMaterial = shaderMaterial(
  {
    uTime: 0,
    uTransition: 0,
    uHologramColor: new THREE.Color(0x00D4DE),
    uSecondaryColor: new THREE.Color(0x0D80FF),
    uFresnelPower: 2.5,
    uScanlineSpeed: 2.0,
    uScanlineCount: 80.0,
    uGlitchIntensity: 0.3,
    uOpacityBase: 0.15,
    uOpacityFresnel: 0.85,
    uTexture: null,
    uHasTexture: false,
    uMouseLightPos: new THREE.Vector3(0, 0, 5),
    uMouseLightIntensity: 1.0
  },
  hologramVertexShader,
  hologramFragmentShader
);

// Extend R3F with the new material
extend({ HologramShaderMaterial });

/**
 * HologramMaterial Component
 * 
 * @param {Object} props
 * @param {number} props.transition - 0 = hologram, 1 = PBR (animated via GSAP)
 * @param {Object} props.mousePos - { x, y } normalized mouse position
 * @param {THREE.Texture} props.originalTexture - Original PBR texture for morphing
 * @param {string} props.hologramColor - Hex color for hologram (default cyan)
 * @param {string} props.secondaryColor - Hex color for secondary glow
 * @param {number} props.glitchIntensity - 0-1 glitch effect intensity
 * @param {boolean} props.enableGlitch - Enable/disable glitch effect
 */
const HologramMaterial = React.forwardRef(({
  transition = 0,
  mousePos = { x: 0, y: 0 },
  originalTexture = null,
  hologramColor = '#00D4DE',
  secondaryColor = '#0D80FF',
  glitchIntensity = 0.3,
  enableGlitch = true,
  opacityBase = 0.15,
  opacityFresnel = 0.85,
  fresnelPower = 2.5,
  scanlineSpeed = 2.0,
  scanlineCount = 80.0,
  ...props
}, ref) => {
  const materialRef = useRef();
  
  // Random glitch trigger
  const glitchState = useRef({
    active: false,
    intensity: 0,
    nextTrigger: Math.random() * 3 + 2
  });
  
  // Convert mouse position to 3D world position for light
  const mouseLightPos = useMemo(() => new THREE.Vector3(), []);
  
  // Initialize material
  useEffect(() => {
    if (materialRef.current) {
      materialRef.current.transparent = true;
      materialRef.current.side = THREE.DoubleSide;
      materialRef.current.depthWrite = false;
      materialRef.current.blending = THREE.AdditiveBlending;
    }
  }, []);
  
  // Update texture if provided
  useEffect(() => {
    if (materialRef.current && originalTexture) {
      materialRef.current.uTexture = originalTexture;
      materialRef.current.uHasTexture = true;
    }
  }, [originalTexture]);
  
  // Animation frame updates
  useFrame((state, delta) => {
    if (!materialRef.current) return;
    
    const mat = materialRef.current;
    const elapsed = state.clock.elapsedTime;
    
    // Update time
    mat.uTime = elapsed;
    
    // Update transition (controlled externally via GSAP)
    mat.uTransition = transition;
    
    // Update mouse light position
    mouseLightPos.set(
      mousePos.x * 3,
      mousePos.y * 2,
      4
    );
    mat.uMouseLightPos = mouseLightPos;
    
    // Random glitch trigger
    if (enableGlitch) {
      const glitch = glitchState.current;
      
      if (!glitch.active && elapsed > glitch.nextTrigger) {
        // Trigger glitch
        glitch.active = true;
        glitch.intensity = Math.random() * 0.5 + 0.3;
        
        // Short duration glitch
        setTimeout(() => {
          glitch.active = false;
          glitch.nextTrigger = elapsed + Math.random() * 5 + 3;
        }, Math.random() * 150 + 50);
      }
      
      mat.uGlitchIntensity = glitch.active ? glitch.intensity : glitchIntensity * 0.1;
    } else {
      mat.uGlitchIntensity = 0;
    }
  });
  
  return (
    <hologramShaderMaterial
      ref={(r) => {
        materialRef.current = r;
        if (ref) {
          if (typeof ref === 'function') ref(r);
          else ref.current = r;
        }
      }}
      uHologramColor={new THREE.Color(hologramColor)}
      uSecondaryColor={new THREE.Color(secondaryColor)}
      uFresnelPower={fresnelPower}
      uScanlineSpeed={scanlineSpeed}
      uScanlineCount={scanlineCount}
      uOpacityBase={opacityBase}
      uOpacityFresnel={opacityFresnel}
      {...props}
    />
  );
});

HologramMaterial.displayName = 'HologramMaterial';

export default HologramMaterial;
export { HologramShaderMaterial };
