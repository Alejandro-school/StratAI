import React, { useRef, useEffect, useState, useImperativeHandle, forwardRef } from 'react';
import { useGLTF, Float, Html } from '@react-three/drei';
import { useFrame, useThree, createPortal } from '@react-three/fiber';

import * as THREE from 'three';
import gsap from 'gsap';
import { useLanding } from '../../LandingContext';
import '../../../../styles/Landing/c4-explosive.css';

// ═══════════════════════════════════════════════════════════════════════════
// C4 TACTICAL EXPLOSIVE - CS2 Bomb Plant Style LCD
// ═══════════════════════════════════════════════════════════════════════════

let hotReloadCounter = 0;

if (import.meta.hot) {
  import.meta.hot.accept(() => {
    hotReloadCounter++;
    useGLTF.clear('/models/c4_target.glb');
  });
}

// ═══════════════════════════════════════════════════════════════════════════
// CS2 BOMB PLANT LCD - Asterisks reveal as you type with decode effect
// ═══════════════════════════════════════════════════════════════════════════

// Characters to use for decode scramble effect
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
  // ORIGINAL LOGIC: show boot during boot, show asterisks after OR if not booting
  const showNormalContent = bootPhase >= 4 || !isBooting;

  return (
    <div ref={containerRef} className={`c4-lcd-container ${isBooting && bootPhase < 4 ? 'c4-lcd-boot' : ''}`}>
      <div className="c4-lcd-display-area">
        {/* Boot content - show during phases 1-3 while booting */}
        {isBooting && bootPhase > 0 && bootPhase < 4 && (
          <div className={`c4-lcd-text c4-lcd-boot-text ${bootPhase === 3 ? 'c4-lcd-instruction' : ''}`}>
            {bootContent}
          </div>
        )}
        
        {/* Normal asterisks display */}
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

// Sound manager
const C4SoundManager = {
  audioContext: null,
  keySounds: [],
  explosionSound: null,
  initialized: false,
  
  init() {
    if (this.initialized) return;
    this.audioContext = new (window.AudioContext || window.webkitAudioContext)();
    
    const keyPaths = [
      '/sounds/C4_key_press1.mp3',
      '/sounds/C4_key_press2.mp3',
      '/sounds/C4_key_press3.mp3',
      '/sounds/C4_key_press4.mp3',
      '/sounds/C4_key_press5.mp3',
      '/sounds/C4_key_press6.mp3',
      '/sounds/C4_key_press7.mp3',
    ];
    
    keyPaths.forEach(path => {
      const audio = new Audio(path);
      audio.volume = 0.3;
      this.keySounds.push(audio);
    });
    
    this.explosionSound = new Audio('/sounds/bomb_planted.wav');
    this.explosionSound.volume = 0.5;
    
    this.initialized = true;
  },
  
  playRandomKeySound() {
    if (!this.initialized) this.init();
    const randomIndex = Math.floor(Math.random() * this.keySounds.length);
    const sound = this.keySounds[randomIndex];
    sound.currentTime = 0;
    sound.play().catch(() => {});
  },
  
  playExplosion() {
    if (!this.initialized) this.init();
    this.explosionSound.currentTime = 0;
    this.explosionSound.play().catch(() => {});
  }
};

// ═══════════════════════════════════════════════════════════════════════════
// MAIN C4 EXPLOSIVE COMPONENT
// ═══════════════════════════════════════════════════════════════════════════
const C4Explosive = forwardRef(({ mousePos = { x: 0, y: 0 } }, ref) => {
  const { scene, nodes } = useGLTF('/Images/Landing/c4_target.glb');
  const { camera } = useThree();
  const { nickname, isArmed } = useLanding();
  
  const groupRef = useRef();
  const floatRef = useRef();
  const cameraTargetRef = useRef();
  const screenSurfaceRef = useRef();
  
  const [version, setVersion] = useState(0);
  const [isZooming, setIsZooming] = useState(false);
  const [showLcdContent, setShowLcdContent] = useState(false);
  const [isLcdBooting, setIsLcdBooting] = useState(false);
  const [cursorVisible, setCursorVisible] = useState(true);
  
  // Responsive state
  const [responsiveConfig, setResponsiveConfig] = useState({
    modelScale: 0.55,
    modelPosition: [-0.5, -1, 0],
    htmlScale: 0.06,
    zoomFov: 80
  });

  // Calculate responsive values based on screen width
  useEffect(() => {
    const updateResponsive = () => {
      const width = window.innerWidth;
      
      let config;
      if (width <= 400) {
        // Very small mobile
        config = {
          modelScale: 0.25,
          modelPosition: [0.3, -0.5, 0],
          htmlScale: 0.06,
          zoomFov: 140
        };
      } else if (width <= 576) {
        // Mobile
        config = {
          modelScale: 0.30,
          modelPosition: [0.4, -0.2, 0],
          htmlScale: 0.09,
          zoomFov: 110
        };
      } else if (width <= 768) {
        // Small tablet
        config = {
          modelScale: 0.38,
          modelPosition: [0.5, -0.7, 0],
          htmlScale: 0.04,
          zoomFov: 90
        };
      } else if (width <= 992) {
        // Tablet
        config = {
          modelScale: 0.45,
          modelPosition: [0.7, -0.8, 0],
          htmlScale: 0.05,
          zoomFov: 85
        };
      } else if (width <= 1200) {
        // Small laptop
        config = {
          modelScale: 0.50,
          modelPosition: [0.8, -0.9, 0],
          htmlScale: 0.055,
          zoomFov: 82
        };
      } else {
        // Desktop
        config = {
          modelScale: 0.15,
          modelPosition: [-0.3, -0.8, 0],
          htmlScale: 0.06,
          zoomFov: 80
        };
      }
      
      setResponsiveConfig(config);
    };
    
    updateResponsive();
    window.addEventListener('resize', updateResponsive);
    return () => window.removeEventListener('resize', updateResponsive);
  }, []);

  useEffect(() => {
    C4SoundManager.init();
  }, []);
  
  useEffect(() => {
    setVersion(hotReloadCounter);
  }, []);

  useEffect(() => {
    if (scene && nodes) {
      const cameraTarget = nodes.CameraTarget || scene.getObjectByName('CameraTarget');
      if (cameraTarget) {
        cameraTargetRef.current = cameraTarget;
      }
      
      const screenSurface = nodes.ScreenSurface || scene.getObjectByName('ScreenSurface');
      if (screenSurface) {
        screenSurfaceRef.current = screenSurface;
      }
    }
  }, [scene, nodes, version]);

  useEffect(() => {
    if (!showLcdContent) return;
    const interval = setInterval(() => {
      setCursorVisible(prev => !prev);
    }, 530);
    return () => clearInterval(interval);
  }, [showLcdContent]);

  const isZoomingRef = useRef(false);

  useImperativeHandle(ref, () => ({
    zoomToLCD: () => {
      if (isZoomingRef.current) return;
      if (!cameraTargetRef.current) return;

      isZoomingRef.current = true;
      setIsZooming(true);
      
      gsap.killTweensOf(camera.position);
      gsap.killTweensOf(camera.rotation);
      gsap.killTweensOf(camera);
      
      if (groupRef.current) {
        gsap.killTweensOf(groupRef.current.rotation);
        gsap.killTweensOf(groupRef.current.position);
        
        // First: Animate rotation smoothly to straight position
        gsap.to(groupRef.current.rotation, {
          x: 0,
          y: 0,
          z: 0,
          duration: 0.8,
          ease: "power3.inOut",
          onComplete: () => {
            // After rotation completes, calculate target position and zoom
            startCameraZoom();
          }
        });
      } else {
        // No group ref, just zoom immediately
        startCameraZoom();
      }
      
      function startCameraZoom() {
        const target = cameraTargetRef.current;
        
        if (groupRef.current) {
          groupRef.current.updateMatrixWorld(true);
        }
        
        const worldPosition = new THREE.Vector3();
        target.getWorldPosition(worldPosition);
        
        gsap.to(camera.position, {
          x: worldPosition.x,
          y: worldPosition.y,
          z: worldPosition.z,
          duration: 1.2,
          ease: "power4.inOut",
          onComplete: () => {
            setShowLcdContent(true);
            setIsLcdBooting(true);
            
            // Boot sequence takes 2.2s (0.2 + 0.8 + 1.2), give it 2.5s
            setTimeout(() => {
              setIsLcdBooting(false);
            }, 2500);
          }
        });

        gsap.to(camera.rotation, {
          x: 0,
          y: 0,
          z: 0,
          duration: 1.2,
          ease: "power4.inOut"
        });

        gsap.to(camera, {
          fov: responsiveConfig.zoomFov,
          duration: 1.4,
          ease: "power4.inOut",
          onUpdate: () => camera.updateProjectionMatrix()
        });
      }
    },

    resetCamera: () => {
      isZoomingRef.current = false;
      setIsZooming(false);
      setShowLcdContent(false);
      setIsLcdBooting(false);
      
      gsap.killTweensOf(camera.position);
      gsap.killTweensOf(camera.rotation);
      gsap.killTweensOf(camera);
      
      gsap.to(camera.position, {
        x: 0,
        y: 0.5,
        z: 8,
        duration: 2,
        ease: "power3.inOut"
      });

      gsap.to(camera.rotation, {
        x: 0,
        y: 0.2,
        z: 0,
        duration: 2,
        ease: "power3.inOut"
      });

      gsap.to(camera, {
        fov: 50,
        duration: 2,
        ease: "power3.inOut",
        onUpdate: () => camera.updateProjectionMatrix()
      });
    }
  }));

  useFrame((state) => {
    if (groupRef.current && !isZooming) {
      const t = state.clock.getElapsedTime();
      const targetRotY = mousePos.x * 0.15;
      const targetRotX = mousePos.y * 0.1;
      
      groupRef.current.rotation.y = THREE.MathUtils.lerp(
        groupRef.current.rotation.y,
        targetRotY + Math.sin(t * 0.4) * 0.03,
        0.05
      );
      
      groupRef.current.rotation.x = THREE.MathUtils.lerp(
        groupRef.current.rotation.x,
        targetRotX,
        0.05
      );
    }
  });

  // Handle Warp Transition - Smooth fade out
  const { showWarpTransition } = useLanding();
  const modelRef = useRef();

  useEffect(() => {
    if (modelRef.current) {
      if (showWarpTransition) {
        // Simple fade out as the camera passes through
        modelRef.current.traverse((child) => {
          if (child.isMesh && child.material) {
            child.material.transparent = true;
            // Smooth fade out over the transition
            gsap.to(child.material, {
              opacity: 0,
              duration: 2.0,
              delay: 0.3, // Start fading shortly after transition begins
              ease: "power1.inOut"
            });
          }
        });
      } else {
        // Reset opacity when transition ends
        if (responsiveConfig.modelScale) {
          modelRef.current.scale.set(responsiveConfig.modelScale, responsiveConfig.modelScale, responsiveConfig.modelScale);
        }
        modelRef.current.traverse((child) => {
          if (child.isMesh && child.material) {
            child.material.opacity = 1;
          }
        });
      }
    }
  }, [showWarpTransition, responsiveConfig.modelScale]);

  return (
    <group ref={groupRef} position={responsiveConfig.modelPosition}>
      {!isZooming ? (
        <Float
          ref={floatRef}
          speed={1.0}
          rotationIntensity={0.2}
          floatIntensity={0.5}
        >
          <primitive 
            ref={modelRef}
            object={scene} 
            scale={responsiveConfig.modelScale} 
          />
        </Float>
      ) : (
        <>
          <primitive 
            ref={modelRef}
            object={scene} 
            scale={responsiveConfig.modelScale} 
          />
          
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
                      // Force selection at end with null check
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
                  isSubmitting={isArmed}
                />
              </div>
            </Html>,
            screenSurfaceRef.current
          )}
        </>
      )}
    </group>
  );
});

export { C4SoundManager };

useGLTF.preload('/Images/Landing/c4_target.glb');
export default C4Explosive;
