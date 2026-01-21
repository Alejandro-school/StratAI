import React, { Suspense, useEffect, useRef, useState } from 'react';
import { Canvas, useThree, useFrame } from '@react-three/fiber';
import { EffectComposer, Bloom, Vignette, ToneMapping } from '@react-three/postprocessing';
import { ToneMappingMode } from 'postprocessing';
import * as THREE from 'three';
import C4Explosive from './C4Explosive';
import { useLanding, LANDING_STEPS } from '../../LandingContext';
import gsap from 'gsap';

// Mouse tracking for 3D interaction
const MouseTracker = ({ onMouseMove }) => {
  const { viewport } = useThree();
  
  useFrame(({ mouse }) => {
    onMouseMove({
      x: (mouse.x * viewport.width) / 2,
      y: (mouse.y * viewport.height) / 2
    });
  });
  
  return null;
};

const CameraController = () => {
  const { camera } = useThree();
  const { currentStep, showWarpTransition } = useLanding();

  // Handle Warp Transition - Smooth camera movement forward
  useEffect(() => {
    if (showWarpTransition) {
      // Simple, smooth camera advance forward
      // The camera moves through the C4 towards the AIM scene
      gsap.to(camera.position, {
        x: 0,
        y: 0,
        z: -30, // Move forward past the C4
        duration: 3.5,
        ease: "power1.inOut" // Very smooth, natural acceleration/deceleration
      });
      
      // Subtle FOV increase for slight speed sensation (not extreme)
      gsap.to(camera, {
        fov: 65, // Just slightly wider than normal
        duration: 1.5,
        ease: "power1.out",
        onUpdate: () => camera.updateProjectionMatrix()
      });
      
      // Normalize FOV smoothly towards the end
      gsap.to(camera, {
        fov: 50,
        duration: 1.5,
        delay: 2.0,
        ease: "power1.inOut",
        onUpdate: () => camera.updateProjectionMatrix()
      });
    }
  }, [showWarpTransition, camera]);

  useEffect(() => {
    switch (currentStep) {
      case LANDING_STEPS.HERO:
        // Hero shot - Camera positioned to see C4 on the right side
        gsap.to(camera.position, {
          x: 0,
          y: 0.5,
          z: 8,
          duration: 2.5,
          ease: "power3.inOut"
        });
        gsap.to(camera.rotation, {
          x: 0,
          y: 0.2,
          z: 0,
          duration: 2.5,
          ease: "power3.inOut"
        });
        gsap.to(camera, {
          fov: 50,
          duration: 2.5,
          ease: "power3.inOut",
          onUpdate: () => camera.updateProjectionMatrix()
        });
        break;

      case LANDING_STEPS.NICKNAME:
        // El zoom lo maneja C4Explosive.zoomToLCD()
        // No hacer nada aquí para evitar doble animación
        break;

      case LANDING_STEPS.AIM:
        // Pull back for aim challenge arena
        gsap.to(camera.position, {
          x: 0,
          y: 1,
          z: 8,
          duration: 1.8,
          ease: "power2.inOut"
        });
        gsap.to(camera, {
          fov: 50,
          duration: 1.8,
          ease: "power2.inOut",
          onUpdate: () => camera.updateProjectionMatrix()
        });
        break;

      default:
        break;
    }
    
    // Cleanup GSAP animations on unmount or dependency change
    return () => {
        gsap.killTweensOf(camera.position);
        gsap.killTweensOf(camera.rotation);
        gsap.killTweensOf(camera);
    };
  }, [currentStep, camera]);

  return null;
};

// Ambient floating particles for depth
const AmbientParticles = () => {
  const particlesRef = useRef();
  const count = 100;
  
  const positions = React.useMemo(() => {
    const pos = new Float32Array(count * 3);
    for (let i = 0; i < count; i++) {
      pos[i * 3] = (Math.random() - 0.5) * 15;
      pos[i * 3 + 1] = (Math.random() - 0.5) * 10;
      pos[i * 3 + 2] = (Math.random() - 0.5) * 15 - 5;
    }
    return pos;
  }, []);
  
  useFrame((state) => {
    if (particlesRef.current) {
      particlesRef.current.rotation.y = state.clock.elapsedTime * 0.02;
      particlesRef.current.rotation.x = Math.sin(state.clock.elapsedTime * 0.01) * 0.1;
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
      </bufferGeometry>
      <pointsMaterial
        size={0.03}
        color="#14b8a6"
        transparent
        opacity={0.6}
        sizeAttenuation
        blending={THREE.AdditiveBlending}
      />
    </points>
  );
};

const SceneOrchestrator = () => {
  const [mousePos, setMousePos] = useState({ x: 0, y: 0 });
  const { currentStep } = useLanding();
  
  // Ref para acceder a los métodos del C4
  const c4Ref = useRef();

  // Listen for step changes to trigger zoom
  useEffect(() => {
    if (currentStep === LANDING_STEPS.NICKNAME && c4Ref.current) {
      // Trigger zoom cuando se avanza al paso NICKNAME
      console.log('🎬 Iniciando zoom a LCD...');
      c4Ref.current.zoomToLCD();
    } else if (currentStep === LANDING_STEPS.HERO && c4Ref.current) {
      // Reset camera cuando vuelve a HERO
      c4Ref.current.resetCamera();
    }
  }, [currentStep]);

  return (
    <div className="scene-orchestrator">
      <Canvas
        dpr={[2, 3]}
        gl={{ 
          antialias: true, 
          stencil: false, 
          depth: true,
          powerPreference: "high-performance",
          precision: "highp"
        }}
        camera={{ position: [0, 0.5, 8], fov: 50 }}
      >
        {/* Dynamic gradient background instead of plain black */}
        <color attach="background" args={['#050508']} />
        
        <CameraController />
        <MouseTracker onMouseMove={setMousePos} />

        <Suspense fallback={null}>
          {/* Modern Lighting Setup - BRIGHT Cinematic 3-point */}
          <ambientLight intensity={1.2} color="#ffffff" />
          
          {/* Key Light - Cool Turquoise from above-right */}
          <spotLight 
            position={[5, 8, 5]} 
            intensity={4.0} 
            color="#14b8a6" 
            penumbra={0.8}
            angle={0.5}
            castShadow
          />
          
          {/* Fill Light - Warm from left */}
          <spotLight 
            position={[-6, 4, 4]} 
            intensity={2.5} 
            color="#f59e0b" 
            penumbra={1}
          />
          
          {/* Rim Light - Back accent cyan */}
          <pointLight 
            position={[0, -2, -3]} 
            intensity={1.5} 
            color="#06b6d4" 
          />
          
          {/* Ground Bounce - Subtle */}
          <pointLight 
            position={[0, -4, 2]} 
            intensity={0.8} 
            color="#10b981" 
          />
          
          {/* Front fill light */}
          <directionalLight
            position={[0, 2, 8]}
            intensity={2.5}
            color="#ffffff"
          />
          
          {/* Ambient Particles */}
          <AmbientParticles />
          
          {/* Main C4 Model with exposed methods */}
          <C4Explosive 
            ref={c4Ref}
            mousePos={mousePos} 
          />

          {/* Clean Post-Processing - No flickering effects */}
          <EffectComposer disableNormalPass multisampling={4}>
            {/* Subtle Bloom for LED glow - higher threshold for less blur */}
            <Bloom 
              luminanceThreshold={0.7}
              luminanceSmoothing={0.3}
              intensity={0.4}
              radius={0.3}
              levels={4}
            />
            
            {/* Subtle Vignette for focus */}
            <Vignette 
              eskil={false} 
              offset={0.3} 
              darkness={0.5} 
            />
            
            {/* Tone Mapping for realistic colors */}
            <ToneMapping
              mode={ToneMappingMode.ACES_FILMIC}
            />
          </EffectComposer>
        </Suspense>
      </Canvas>
    </div>
  );
};

export default SceneOrchestrator;
