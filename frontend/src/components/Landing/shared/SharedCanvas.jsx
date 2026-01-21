/**
 * SharedCanvas - Persistent WebGL Canvas for all Landing scenes
 * 
 * Keeps a single WebGL context alive throughout the landing experience,
 * preventing context loss during scene transitions.
 */
import React, { Suspense, useState, useEffect, useRef, useCallback } from 'react';
import { Canvas, useFrame, useThree } from '@react-three/fiber';
import { EffectComposer, Bloom, Vignette, ToneMapping } from '@react-three/postprocessing';
import { ToneMappingMode } from 'postprocessing';

import UnifiedCameraController from './UnifiedCameraController';
import HeroSceneContent from '../Hero/Scene3D/HeroSceneContent';
import AimSceneContent from '../Challenges/AimChallenge/AimSceneContent';

import './SharedCanvas.css';

/**
 * SharedCanvas Component
 * 
 * @param {string} activeScene - 'hero' | 'aim' | 'none'
 * @param {boolean} isTransitioning - Whether a warp transition is in progress
 */
const SharedCanvas = ({ activeScene, isTransitioning }) => {
  const mousePosRef = useRef({ x: 0, y: 0 });
  const [mousePos, setMousePos] = useState({ x: 0, y: 0 });
  
  // Track mouse globally via window events (not canvas-dependent)
  // This ensures mouse tracking works even with overlays
  useEffect(() => {
    if (activeScene !== 'hero') return;
    
    const handleMouseMove = (e) => {
      // Normalize to -1 to 1
      const x = (e.clientX / window.innerWidth) * 2 - 1;
      const y = -((e.clientY / window.innerHeight) * 2 - 1); // Invert Y for 3D
      
      mousePosRef.current = { x, y };
      setMousePos({ x, y });
    };
    
    window.addEventListener('mousemove', handleMouseMove);
    
    // Debug log
    console.log('🖱️ Global mouse tracking enabled for Hero scene');
    
    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      console.log('🖱️ Global mouse tracking disabled');
    };
  }, [activeScene]);
  
  // Determine background color based on scene
  // Dark background for hologram effect - must be very dark
  const backgroundColor = activeScene === 'aim' ? '#87ceeb' : '#020408';
  
  console.log('🎨 SharedCanvas rendering:', { activeScene, isTransitioning, backgroundColor });
  
  return (
    <div className="shared-canvas-container">
      <Canvas
        dpr={[1.5, 2]}
        gl={{ 
          antialias: true, 
          stencil: false, 
          depth: true,
          powerPreference: "high-performance",
          precision: "highp",
          alpha: true, // Enable alpha for hologram blending
          preserveDrawingBuffer: true,
          localClippingEnabled: true // Enable clip planes for materialization effect
        }}
        camera={{ position: [0, 0.5, 8], fov: 50 }}
        shadows
        onCreated={({ gl }) => {
          gl.localClippingEnabled = true; // Ensure clipping is enabled
        }}
      >
        {/* Dynamic background color - very dark for hologram */}
        <color attach="background" args={[backgroundColor]} />
        
        {/* Unified Camera Controller - handles all scenes */}
        <UnifiedCameraController activeScene={activeScene} />

        <Suspense fallback={null}>
          {/* Hero Scene Content */}
          {activeScene === 'hero' && (
            <HeroSceneContent mousePos={mousePos} />
          )}
          
          {/* AIM Scene Content - Pre-mount during transition for seamless entry */}
          {(activeScene === 'aim' || isTransitioning) && (
            <AimSceneContent 
              position={isTransitioning ? [0, 0, -31.5] : [0, 0, 0]}
              enableLights={true}
              visible={!isTransitioning} // Hide during transition so user only sees warp -> menu
            />
          )}

          {/* Post-Processing - Adapted per scene */}
          <EffectComposer disableNormalPass multisampling={4}>
            <Bloom 
              luminanceThreshold={activeScene === 'aim' ? 0.7 : 0.7}
              luminanceSmoothing={0.3}
              intensity={activeScene === 'aim' ? 0.4 : 0.4}
              radius={activeScene === 'aim' ? 0.5 : 0.3}
              levels={4}
            />
            <Vignette 
              eskil={false} 
              offset={0.3} 
              darkness={0.5} 
            />
            {activeScene === 'hero' && (
              <ToneMapping mode={ToneMappingMode.ACES_FILMIC} />
            )}
          </EffectComposer>
        </Suspense>
      </Canvas>
    </div>
  );
};

export default SharedCanvas;
