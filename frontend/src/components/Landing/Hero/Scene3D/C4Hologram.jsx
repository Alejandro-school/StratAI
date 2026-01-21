/**
 * C4Hologram - AUTHENTIC Holographic C4 model
 * 
 * Features:
 * - TRUE hologram effect (transparent, scanlines, flicker, wireframe edges)
 * - Large size filling background
 * - Mouse parallax rotation
 * - Dynamic point light following cursor
 * - Subnautica-style materialization (clipping plane bottom-to-top)
 * - Zoom to LCD screen after materialization
 */
import React, { useRef, useEffect, useState, useImperativeHandle, forwardRef, useMemo } from 'react';
import { useGLTF, Html } from '@react-three/drei';
import { useFrame, useThree, createPortal } from '@react-three/fiber';
import * as THREE from 'three';
import gsap from 'gsap';
import { hologramVertexShader, hologramFragmentShader } from '../shaders/hologramShader';
import { useLanding } from '../../LandingContext';

// ═══════════════════════════════════════════════════════════════════════════
// LCD DISPLAY COMPONENT - Same as in C4Explosive
// ═══════════════════════════════════════════════════════════════════════════
const DECODE_CHARS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@#$%&*';

const CS2BombDisplay = ({ text, maxLength = 10, showCursor, isBooting, isSubmitting }) => {
  const [bootPhase, setBootPhase] = useState(0);
  const [decodingChars, setDecodingChars] = useState({});
  const containerRef = useRef(null);
  const prevTextRef = useRef('');

  // Boot sequence
  useEffect(() => {
    if (isBooting) {
      setBootPhase(1);
      
      const timeline = gsap.timeline();
      
      timeline.to({}, {
        duration: 0.2,
        onComplete: () => setBootPhase(2)
      });
      
      timeline.to({}, {
        duration: 0.8,
        onComplete: () => setBootPhase(3)
      });
      
      timeline.to({}, {
        duration: 1.2,
        onComplete: () => setBootPhase(4)
      });

      return () => timeline.kill();
    }
  }, [isBooting]);

  // Decode animation when new characters are typed
  useEffect(() => {
    const prevText = prevTextRef.current || '';
    const currentText = text || '';
    
    if (currentText.length > prevText.length) {
      const newCharIndex = currentText.length - 1;
      setDecodingChars(prev => ({ ...prev, [newCharIndex]: true }));
      
      let scrambleCount = 0;
      const scrambleInterval = setInterval(() => {
        scrambleCount++;
        if (scrambleCount >= 6) {
          clearInterval(scrambleInterval);
          setDecodingChars(prev => {
            const updated = { ...prev };
            delete updated[newCharIndex];
            return updated;
          });
        }
      }, 40);
    }
    
    prevTextRef.current = currentText;
  }, [text]);

  const generateDisplay = () => {
    const typedChars = text ? text.toUpperCase() : '';
    const typedLength = typedChars.length;
    if (typedLength === 0) return '*'.repeat(maxLength);
    return typedChars + '*'.repeat(Math.max(0, maxLength - typedLength));
  };

  const getDisplayChar = (char, index) => {
    if (decodingChars[index]) {
      return DECODE_CHARS[Math.floor(Math.random() * DECODE_CHARS.length)];
    }
    return char;
  };

  const displayContent = generateDisplay();
  const hasText = text && text.length > 0;

  const getBootContent = () => {
    switch (bootPhase) {
      case 1: return "SCANNING...";
      case 2: return "CONNECTED";
      case 3: return "ENTER TAG";
      default: return null;
    }
  };

  const bootContent = getBootContent();
  const showNormalContent = bootPhase >= 4 || !isBooting;

  return (
    <div ref={containerRef} className={`c4-lcd-container ${isBooting && bootPhase < 4 ? 'c4-lcd-boot' : ''}`}>
      <div className="c4-lcd-display-area">
        {isBooting && bootPhase > 0 && bootPhase < 4 && (
          <div className={`c4-lcd-text c4-lcd-boot-text ${bootPhase === 3 ? 'c4-lcd-instruction' : ''}`}>
            {bootContent}
          </div>
        )}
        
        {showNormalContent && (
          <div className={`c4-lcd-text ${!hasText ? 'c4-lcd-waiting' : ''}`}>
            {isSubmitting ? (
              <span className="c4-lcd-status-text">SENDING...</span>
            ) : (
              displayContent.split('').map((char, index) => {
                const isTypedChar = text && index < text.length;
                const isAsterisk = char === '*';
                const displayChar = isTypedChar ? getDisplayChar(char, index) : char;
                return (
                  <span 
                    key={index}
                    className={`c4-lcd-char ${isAsterisk ? 'c4-lcd-asterisk' : 'c4-lcd-revealed'} ${decodingChars[index] ? 'c4-lcd-decoding' : ''}`}
                  >
                    {displayChar}
                  </span>
                );
              })
            )}
          </div>
        )}
      </div>
    </div>
  );
};

// ═══════════════════════════════════════════════════════════════════════════
// C4 HOLOGRAM - Authentic holographic effect like Subnautica
// ═══════════════════════════════════════════════════════════════════════════

const C4Hologram = forwardRef(({ 
  mousePos = { x: 0, y: 0 },
  onTransitionStart,
  onTransitionComplete,
}, ref) => {
  const { scene, nodes } = useGLTF('/Images/Landing/c4_target.glb');
  const { camera, viewport, size } = useThree();
  const { nickname } = useLanding();
  
  // Refs
  const groupRef = useRef();
  const hologramMaterialsRef = useRef([]);
  const realMaterialsRef = useRef([]); // For real model clip plane
  const pointLightRef = useRef();
  const clipYRef = useRef({ value: 0 });
  const cameraTargetRef = useRef();
  const screenSurfaceRef = useRef();
  const isZoomingRef = useRef(false);
  
  // State
  const [isTransitioning, setIsTransitioning] = useState(false);
  const [isZooming, setIsZooming] = useState(false);
  const [isMaterialized, setIsMaterialized] = useState(false);
  const [materializeProgress, setMaterializeProgress] = useState(0); // 0 to 1 for smooth light transition
  const [hologramScene, setHologramScene] = useState(null); // Hologram version
  const [realScene, setRealScene] = useState(null); // Real PBR version
  const [modelInfo, setModelInfo] = useState({ height: 2, minY: -1, maxY: 1 });
  const [showLcdContent, setShowLcdContent] = useState(false);
  const [isLcdBooting, setIsLcdBooting] = useState(false);
  const [cursorVisible, setCursorVisible] = useState(true);

  // Clone scene on mount - create TWO versions: hologram and real
  useEffect(() => {
    if (!scene) return;
    
    // Clone for hologram (will get shader materials)
    const holoClone = scene.clone(true);
    // Clone for real model (keeps PBR materials)
    const realClone = scene.clone(true);
    
    // Calculate bounding box
    const box = new THREE.Box3().setFromObject(holoClone);
    const height = box.max.y - box.min.y;
    
    // Find CameraTarget and ScreenSurface in hologram clone
    const cameraTarget = holoClone.getObjectByName('CameraTarget');
    if (cameraTarget) {
      cameraTargetRef.current = cameraTarget;
      console.log('📍 Found CameraTarget in cloned scene');
    }
    
    const screenSurface = holoClone.getObjectByName('ScreenSurface');
    if (screenSurface) {
      screenSurfaceRef.current = screenSurface;
      console.log('📺 Found ScreenSurface in cloned scene');
    }
    
    setHologramScene(holoClone);
    setRealScene(realClone);
    setModelInfo({ height, minY: box.min.y, maxY: box.max.y });
  }, [scene]);

  // Calculate responsive scale - moderate size like reference
  const responsiveConfig = useMemo(() => {
    // Use size for pixel-based calculations
    const isMobile = size.width < 768;
    const isTablet = size.width < 1024;
    
    // Moderate size - visible but not overwhelming
    let scale, position, rotation, zoomFov, htmlScale;
    
    if (isMobile) {
      scale = 0.6;
      position = [0.5, -0.3, -2];
      rotation = [0.2, -0.4, 0.05];
      zoomFov = 35;
      htmlScale = 0.08;
    } else if (isTablet) {
      scale = 0.8;
      position = [1.2, -0.2, -1.5];
      rotation = [0.15, -0.45, 0.03];
      zoomFov = 30;
      htmlScale = 0.06;
    } else {
      // Desktop - positioned to the right, medium size
      scale = 0.8;
      position = [0, -4, -2];
      rotation = [0.1, -0.5, 0.02];
      zoomFov = 80;
      htmlScale = 0.045;
    }
    
    return { scale, position, rotation, zoomFov, htmlScale };
  }, [size]);

  // Cursor blink effect
  useEffect(() => {
    if (!showLcdContent) return;
    const interval = setInterval(() => {
      setCursorVisible(prev => !prev);
    }, 530);
    return () => clearInterval(interval);
  }, [showLcdContent]);

  // ═══════════════════════════════════════════════════════════════════════════
  // CREATE HOLOGRAM MATERIALS (for hologram scene)
  // ═══════════════════════════════════════════════════════════════════════════
  useEffect(() => {
    if (!hologramScene) return;
    
    const materials = [];
    const modelHeight = modelInfo.height;
    
    hologramScene.traverse((child) => {
      if (child.isMesh) {
        // Create authentic hologram shader material
        const hologramMaterial = new THREE.ShaderMaterial({
          uniforms: {
            uTime: { value: 0 },
            uTransition: { value: 0 },
            uHologramColor: { value: new THREE.Color(0x00EEFF) },  // Bright cyan
            uHologramColor2: { value: new THREE.Color(0x0099FF) }, // Blue accent
            uFresnelPower: { value: 2.0 },
            uScanlineSpeed: { value: 1.5 },
            uScanlineCount: { value: 40.0 },
            uGlitchIntensity: { value: 0.8 },
            uFlickerSpeed: { value: 1.0 },
            uTexture: { value: child.material.map },
            uHasTexture: { value: !!child.material.map },
            uMouseLightPos: { value: new THREE.Vector3(0, 0, 5) },
            uMouseLightIntensity: { value: 1.5 },
            uClipY: { value: 0 },
            uModelHeight: { value: modelHeight }
          },
          vertexShader: hologramVertexShader,
          fragmentShader: hologramFragmentShader,
          transparent: true,
          side: THREE.DoubleSide,
          depthWrite: false,
          blending: THREE.AdditiveBlending
        });
        
        child.material = hologramMaterial;
        materials.push(hologramMaterial);
      }
    });
    
    hologramMaterialsRef.current = materials;
    
  }, [hologramScene, modelInfo.height]);

  // ═══════════════════════════════════════════════════════════════════════════
  // SETUP REAL MODEL WITH CLIP PLANE (appears from bottom during materialization)
  // ═══════════════════════════════════════════════════════════════════════════
  useEffect(() => {
    if (!realScene || modelInfo.height === 2) return; // Wait for proper modelInfo
    
    const materials = [];
    
    // Create a clipping plane that reveals from bottom
    // Plane normal (0, 1, 0) = pointing UP
    // Plane equation: y - constant >= 0 to keep (keeps what's ABOVE the plane)
    // We want the opposite: keep what's BELOW, so we use normal (0, -1, 0)
    // Or we can use (0, 1, 0) with NEGATIVE constant
    // -y + (-constant) >= 0 => y <= -constant... confusing
    // Let's use (0, 1, 0) and set constant to -clipWorldY
    // dot(normal, P) + constant >= 0
    // y + constant >= 0, so keep points where y >= -constant
    // If constant = -minY, keep y >= minY (shows nothing at start)
    // If constant = -maxY, keep y >= maxY (shows nothing)
    // That's wrong... let me think again.
    //
    // For (0, -1, 0) normal:
    // -y + constant >= 0 => y <= constant (keep below constant)
    // constant = minY at start => keep y <= minY => nothing
    // constant = maxY at end => keep y <= maxY => everything
    //
    // That's exactly what we want! The issue might be the scale or transform.
    // Let's make sure clipShadows works and the constant is correct.
    
    realScene.traverse((child) => {
      if (child.isMesh) {
        // Clone the material to avoid sharing
        const mat = child.material.clone();
        // Use world-space clip plane
        const clipPlane = new THREE.Plane(new THREE.Vector3(0, -1, 0), modelInfo.minY);
        mat.clippingPlanes = [clipPlane];
        mat.clipShadows = true;
        mat.side = THREE.DoubleSide;
        mat.needsUpdate = true;
        child.material = mat;
        materials.push({ material: mat, clipPlane });
      }
    });
    
    // Initially hide the real model
    realScene.visible = false;
    realMaterialsRef.current = materials;
    
    console.log('📐 Real model clip planes configured:', { 
      minY: modelInfo.minY, 
      maxY: modelInfo.maxY, 
      height: modelInfo.height,
      materialsCount: materials.length 
    });
    
  }, [realScene, modelInfo]);

  // ═══════════════════════════════════════════════════════════════════════════
  // ZOOM TO LCD FUNCTION
  // ═══════════════════════════════════════════════════════════════════════════
  const startCameraZoom = () => {
    if (!cameraTargetRef.current) {
      console.warn('⚠️ No CameraTarget found, skipping zoom');
      onTransitionComplete?.();
      return;
    }
    
    console.log('🎥 Starting camera zoom to LCD...');
    setIsZooming(true);
    isZoomingRef.current = true;
    
    // Update world matrices
    if (groupRef.current) {
      groupRef.current.updateMatrixWorld(true);
    }
    
    // Get world position of CameraTarget - this IS where the camera should go
    const targetPos = new THREE.Vector3();
    cameraTargetRef.current.getWorldPosition(targetPos);
    
    // Get world rotation of CameraTarget - this IS how the camera should be oriented
    const targetQuat = new THREE.Quaternion();
    cameraTargetRef.current.getWorldQuaternion(targetQuat);
    const targetEuler = new THREE.Euler().setFromQuaternion(targetQuat);
    
    console.log('📍 Moving camera to CameraTarget position:', targetPos);
    
    // Animate camera position to CameraTarget
    gsap.to(camera.position, {
      x: targetPos.x,
      y: targetPos.y,
      z: targetPos.z,
      duration: 1.2,
      ease: "power4.inOut"
    });
    
    // Animate camera rotation to match CameraTarget
    gsap.to(camera.rotation, {
      x: targetEuler.x,
      y: targetEuler.y,
      z: targetEuler.z,
      duration: 1.2,
      ease: "power4.inOut",
      onComplete: () => {
        console.log('✅ Camera zoom complete, showing LCD');
        setShowLcdContent(true);
        setIsLcdBooting(true);
        
        // Boot sequence
        setTimeout(() => {
          setIsLcdBooting(false);
          onTransitionComplete?.();
        }, 2500);
      }
    });
    
    // Zoom FOV for close framing
    gsap.to(camera, {
      fov: responsiveConfig.zoomFov,
      duration: 1.4,
      ease: "power4.inOut",
      onUpdate: () => camera.updateProjectionMatrix()
    });
  };

  // ═══════════════════════════════════════════════════════════════════════════
  // IMPERATIVE API - External control
  // ═══════════════════════════════════════════════════════════════════════════
  useImperativeHandle(ref, () => ({
    // Full sequence: straighten → materialize → zoom
    materialize: () => {
      if (isTransitioning || isZoomingRef.current) return;
      
      console.log('🚀 Starting full materialization sequence...');
      setIsTransitioning(true);
      onTransitionStart?.();
      
      // Kill any existing animations
      gsap.killTweensOf(camera.position);
      gsap.killTweensOf(camera.rotation);
      if (groupRef.current) {
        gsap.killTweensOf(groupRef.current.rotation);
        gsap.killTweensOf(groupRef.current.position);
      }
      
      // STEP 1: Straighten the model (rotate to 0,0,0) - DO NOT change position
      if (groupRef.current) {
        console.log('📐 Step 1: Straightening model (rotation only)...');
        gsap.to(groupRef.current.rotation, {
          x: 0,
          y: 0,
          z: 0,
          duration: 1.2,
          ease: "power2.inOut",
          onComplete: () => {
            // STEP 2: Materialize (hologram to real)
            console.log('✨ Step 2: Materializing...');
            startMaterialization();
          }
        });
        // Position stays the same - don't move during straighten
      } else {
        startMaterialization();
      }
      
      function startMaterialization() {
        // Show the real model - it will be revealed by clip plane
        if (realScene) {
          realScene.visible = true;
        }
        
        // Animate clip plane from 0 to 1 (bottom to top)
        gsap.to(clipYRef.current, {
          value: 1.05, // Go slightly above 1 to ensure full reveal
          duration: 2.5,
          ease: "power2.inOut",
          onUpdate: () => {
            const progress = Math.min(clipYRef.current.value, 1.0);
            
            // Update hologram materials - clip disappears fragments ABOVE the line
            hologramMaterialsRef.current.forEach(mat => {
              if (mat.uniforms) {
                mat.uniforms.uClipY.value = clipYRef.current.value;
                mat.uniforms.uTransition.value = progress;
              }
            });
            
            // Update real model clip planes - reveal from bottom
            // Plane (0, -1, 0) keeps points where y <= constant
            const clipWorldY = modelInfo.minY + progress * modelInfo.height;
            realMaterialsRef.current.forEach(item => {
              if (item.clipPlane) {
                item.clipPlane.constant = clipWorldY;
              }
            });
            
            // Update progress state for smooth light transitions
            setMaterializeProgress(progress);
          },
          onComplete: () => {
            console.log('🎨 Materialization complete');
            
            // Hide hologram, show only real
            if (hologramScene) {
              hologramScene.visible = false;
            }
            
            // Remove clipping from real materials
            realMaterialsRef.current.forEach(item => {
              if (item.material) {
                item.material.clippingPlanes = [];
                item.material.needsUpdate = true;
              }
            });
            
            setIsMaterialized(true);
            setMaterializeProgress(1);
            setIsTransitioning(false);
            
            // STEP 3: Zoom to LCD
            console.log('🎥 Step 3: Zooming to LCD...');
            setTimeout(() => {
              startCameraZoom();
            }, 400);
          }
        });
        
        // Subtle camera approach during materialization
        gsap.to(camera.position, {
          z: camera.position.z - 1.5,
          duration: 2.0,
          ease: "power2.out"
        });
      }
    },
    
    // Reset to hologram state
    reset: () => {
      console.log('🔄 Resetting C4 to hologram state...');
      
      // Reset all state
      clipYRef.current.value = 0;
      setIsTransitioning(false);
      setIsZooming(false);
      setIsMaterialized(false);
      setMaterializeProgress(0);
      isZoomingRef.current = false;
      setShowLcdContent(false);
      setIsLcdBooting(false);
      
      // Kill any running animations
      gsap.killTweensOf(camera.position);
      gsap.killTweensOf(camera.rotation);
      gsap.killTweensOf(camera);
      gsap.killTweensOf(clipYRef.current);
      if (groupRef.current) {
        gsap.killTweensOf(groupRef.current.rotation);
        gsap.killTweensOf(groupRef.current.position);
      }
      
      // Reset group position and rotation
      if (groupRef.current) {
        gsap.to(groupRef.current.position, {
          x: responsiveConfig.position[0],
          y: responsiveConfig.position[1],
          z: responsiveConfig.position[2],
          duration: 1.0,
          ease: "power2.out"
        });
        
        gsap.to(groupRef.current.rotation, {
          x: responsiveConfig.rotation[0],
          y: responsiveConfig.rotation[1],
          z: responsiveConfig.rotation[2],
          duration: 1.0,
          ease: "power2.out"
        });
      }
      
      // Show hologram, hide real
      if (hologramScene) {
        hologramScene.visible = true;
      }
      if (realScene) {
        realScene.visible = false;
        // Reset clip planes
        realMaterialsRef.current.forEach(item => {
          if (item.clipPlane) {
            item.clipPlane.constant = modelInfo.minY;
          }
        });
      }
      
      // Reset hologram shader uniforms
      hologramMaterialsRef.current.forEach(mat => {
        if (mat.uniforms) {
          mat.uniforms.uClipY.value = 0;
          mat.uniforms.uTransition.value = 0;
        }
      });
      
      // Reset camera
      gsap.to(camera.position, {
        x: 0,
        y: 0.5,
        z: 8,
        duration: 1.5,
        ease: "power2.out"
      });
      
      gsap.to(camera.rotation, {
        x: 0,
        y: 0,
        z: 0,
        duration: 1.5,
        ease: "power2.out"
      });
      
      gsap.to(camera, {
        fov: 50,
        duration: 1.5,
        ease: "power2.out",
        onUpdate: () => camera.updateProjectionMatrix()
      });
    },
    
    // Trigger glitch burst
    glitch: (intensity = 2.0, duration = 0.25) => {
      hologramMaterialsRef.current.forEach(mat => {
        if (!mat.uniforms) return;
        const original = mat.uniforms.uGlitchIntensity.value;
        gsap.to(mat.uniforms.uGlitchIntensity, {
          value: intensity,
          duration: 0.04,
          onComplete: () => {
            gsap.to(mat.uniforms.uGlitchIntensity, {
              value: original,
              duration: duration
            });
          }
        });
      });
    }
  }));

  // ═══════════════════════════════════════════════════════════════════════════
  // ANIMATION FRAME - Parallax, lighting, shader updates
  // ═══════════════════════════════════════════════════════════════════════════
  useFrame((state) => {
    const elapsed = state.clock.getElapsedTime();
    
    // Use mousePos from props (comes from window events, not canvas)
    const mx = mousePos.x;
    const my = mousePos.y;
    
    // Update all shader uniforms
    hologramMaterialsRef.current.forEach(mat => {
      if (mat.uniforms) {
        mat.uniforms.uTime.value = elapsed;
        mat.uniforms.uMouseLightPos.value.set(
          mx * 6,
          my * 4,
          4
        );
      }
    });
    
    // Mouse parallax rotation - only when not transitioning, not zooming, and NOT materialized
    // Once materialized, model should stay perfectly still (at rotation 0,0,0)
    if (groupRef.current && !isTransitioning && !isZooming && !isMaterialized) {
      const baseRotY = responsiveConfig.rotation[1];
      const baseRotX = responsiveConfig.rotation[0];
      
      // Strong mouse influence for clearly visible interaction
      const targetRotY = baseRotY + mx * 0.5;
      const targetRotX = baseRotX - my * 0.3;
      
      // Apply rotation with lerp
      groupRef.current.rotation.y = THREE.MathUtils.lerp(
        groupRef.current.rotation.y,
        targetRotY + Math.sin(elapsed * 0.3) * 0.02,
        0.1
      );
      
      groupRef.current.rotation.x = THREE.MathUtils.lerp(
        groupRef.current.rotation.x,
        targetRotX + Math.cos(elapsed * 0.25) * 0.015,
        0.1
      );
      
      // Add subtle Z rotation for more dynamic feel
      groupRef.current.rotation.z = THREE.MathUtils.lerp(
        groupRef.current.rotation.z,
        mx * 0.05,
        0.05
      );
    }
    
    // Point light follows mouse smoothly
    if (pointLightRef.current) {
      pointLightRef.current.position.x = THREE.MathUtils.lerp(
        pointLightRef.current.position.x,
        mousePos.x * 8,
        0.08
      );
      pointLightRef.current.position.y = THREE.MathUtils.lerp(
        pointLightRef.current.position.y,
        mousePos.y * 5 + 2,
        0.08
      );
      
      // Very subtle intensity flicker
      pointLightRef.current.intensity = 0.25 + Math.sin(elapsed * 2) * 0.05;
    }
    
    // Random glitch trigger - occasional
    if (Math.random() > 0.996 && !isTransitioning) {
      const intensity = 1.5 + Math.random() * 1.5;
      hologramMaterialsRef.current.forEach(mat => {
        if (mat.uniforms) {
          gsap.to(mat.uniforms.uGlitchIntensity, {
            value: intensity,
            duration: 0.03,
            onComplete: () => {
              gsap.to(mat.uniforms.uGlitchIntensity, {
                value: 0.8,
                duration: 0.08 + Math.random() * 0.12
              });
            }
          });
        }
      });
    }
  });

  if (!hologramScene) return null;

  return (
    <group 
      ref={groupRef} 
      position={responsiveConfig.position}
    >
      {/* Main dynamic point light - follows cursor, intensity increases with materialization */}
      <pointLight
        ref={pointLightRef}
        color={materializeProgress > 0.5 ? "#FFFFFF" : "#00EEFF"}
        intensity={0.3 + materializeProgress * 1.2}
        distance={15}
        decay={2}
        position={[0, 2, 6]}
      />
      
      {/* Secondary rim light - increases with materialization */}
      <pointLight
        color={materializeProgress > 0.5 ? "#FFFFFF" : "#0066FF"}
        intensity={0.15 + materializeProgress * 0.85}
        distance={12}
        decay={2}
        position={[-5, 1, -4]}
      />
      
      {/* Back rim light - increases with materialization */}
      <pointLight
        color={materializeProgress > 0.5 ? "#FFFFFF" : "#00FFFF"}
        intensity={0.15 + materializeProgress * 0.65}
        distance={10}
        decay={2}
        position={[3, -4, -6]}
      />
      
      {/* Progressive ambient and directional lights during materialization */}
      <ambientLight intensity={0.3 + materializeProgress * 1.2} color="#FFFFFF" />
      <directionalLight 
        position={[5, 5, 5]} 
        intensity={0.5 + materializeProgress * 2.5} 
        color="#FFFFFF"
      />
      <directionalLight 
        position={[-3, 3, 2]} 
        intensity={0.3 + materializeProgress * 1.7} 
        color="#E0E8FF"
      />
      <directionalLight 
        position={[0, -3, 5]} 
        intensity={0.2 + materializeProgress * 1.3} 
        color="#FFEECC"
      />
      
      {/* Hologram model - fades out during materialization */}
      {hologramScene && (
        <primitive 
          object={hologramScene} 
          scale={responsiveConfig.scale} 
        />
      )}
      
      {/* Real PBR model - revealed during materialization */}
      {realScene && (
        <primitive 
          object={realScene} 
          scale={responsiveConfig.scale} 
        />
      )}
      
      {/* LCD Content - shown after zoom completes */}
      {showLcdContent && screenSurfaceRef.current && createPortal(
        <Html
          transform
          position={[-0.05, -0.8, 0.05]} 
          rotation={[0, 0, -1.57]} 
          scale={responsiveConfig.htmlScale}
          zIndexRange={[100, 0]}
          className="c4-html-wrapper"
        >
          <div 
            style={{ cursor: 'text' }}
            onClick={(e) => {
              e.stopPropagation();
              e.preventDefault();
              const input = document.getElementById('nickname-hidden-input');
              if (input) {
                try {
                  input.focus();
                  const val = input.value || '';
                  if (input.setSelectionRange) {
                    input.setSelectionRange(val.length, val.length);
                  }
                } catch (err) {
                  // Ignore focus errors
                }
              }
            }}
          >
            <CS2BombDisplay 
              text={nickname} 
              showCursor={cursorVisible} 
              maxLength={8}
              isBooting={isLcdBooting}
              isSubmitting={false}
            />
          </div>
        </Html>,
        screenSurfaceRef.current
      )}
    </group>
  );
});

C4Hologram.displayName = 'C4Hologram';

useGLTF.preload('/Images/Landing/c4_target.glb');

export default C4Hologram;
