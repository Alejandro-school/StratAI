/**
 * UnifiedCameraController - Handles camera for all scenes
 * 
 * Manages camera positions, transitions, and animations across:
 * - Hero scene (C4 view)
 * - Warp transition (zoom through)
 * - AIM challenge (FPS view)
 */
import { useEffect, useRef } from 'react';
import { useThree } from '@react-three/fiber';
import gsap from 'gsap';
import { useLanding, LANDING_STEPS } from '../LandingContext';

const UnifiedCameraController = ({ activeScene }) => {
  const { camera } = useThree();
  const { currentStep, showWarpTransition } = useLanding();
  const lastSceneRef = useRef(null);
  const isTransitioningRef = useRef(false);

  // Handle Warp Transition - Camera moves forward through C4
  useEffect(() => {
    if (showWarpTransition && !isTransitioningRef.current) {
      isTransitioningRef.current = true;
      console.log('🚀 Starting warp transition camera animation');
      
      // Kill any existing animations first
      gsap.killTweensOf(camera.position);
      gsap.killTweensOf(camera.rotation);
      gsap.killTweensOf(camera);
      
      // Simple, smooth camera advance forward
      gsap.to(camera.position, {
        x: 0,
        y: 0,
        z: -30, // Move forward past the C4
        duration: 3.5,
        ease: "power1.inOut"
      });
      
      // Subtle FOV increase for speed sensation
      gsap.to(camera, {
        fov: 65,
        duration: 1.5,
        ease: "power1.out",
        onUpdate: () => camera.updateProjectionMatrix()
      });
      
      // Normalize FOV towards the end
      gsap.to(camera, {
        fov: 50,
        duration: 1.5,
        delay: 2.0,
        ease: "power1.inOut",
        onUpdate: () => camera.updateProjectionMatrix()
      });
    }
    
    // When warp ends, reset the flag
    if (!showWarpTransition && isTransitioningRef.current) {
      isTransitioningRef.current = false;
    }
  }, [showWarpTransition, camera]);

  // Scene-specific camera positions - runs when activeScene changes
  useEffect(() => {
    // Skip if currently in warp transition
    if (showWarpTransition) {
      console.log('⏳ Skipping camera update during warp transition');
      return;
    }
    
    // Skip if scene hasn't changed
    if (activeScene === lastSceneRef.current) {
      return;
    }
    
    console.log(`📷 Camera switching to scene: ${activeScene}`);
    lastSceneRef.current = activeScene;

    // Kill any previous animations before starting new ones
    gsap.killTweensOf(camera.position);
    gsap.killTweensOf(camera.rotation);
    gsap.killTweensOf(camera);

    if (activeScene === 'hero') {
      if (currentStep === LANDING_STEPS.HERO) {
        // Hero shot - Camera positioned to see C4
        console.log('📷 Setting camera to Hero position');
        gsap.to(camera.position, {
          x: 0,
          y: 0.5,
          z: 8,
          duration: 1.5,
          ease: "power3.inOut"
        });
        gsap.to(camera.rotation, {
          x: 0,
          y: 0.2,
          z: 0,
          duration: 1.5,
          ease: "power3.inOut"
        });
        gsap.to(camera, {
          fov: 50,
          duration: 1.5,
          ease: "power3.inOut",
          onUpdate: () => camera.updateProjectionMatrix()
        });
      }
      // NICKNAME step - zoom is handled by C4Explosive.zoomToLCD()
    } else if (activeScene === 'aim') {
      // Position camera for FPS view - IMMEDIATELY set position, then animate if needed
      console.log('📷 Setting camera to AIM FPS position');
      
      // Set immediately to avoid black screen
      camera.position.set(0, 1.6, 1.5);
      camera.rotation.set(-0.05, 0, 0);
      camera.fov = 70;
      camera.lookAt(0, 1.4, -3);
      camera.updateProjectionMatrix();
      
      console.log('📷 Camera now at:', camera.position.toArray(), 'FOV:', camera.fov);
    }
    
    // No cleanup here - let animations complete
  }, [activeScene, showWarpTransition, currentStep, camera]);

  return null;
};

export default UnifiedCameraController;

