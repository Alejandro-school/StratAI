/**
 * AgentBackground — Fixed CT model as subtle landing page background
 *
 * Renders the GLB agent in hologram mode (revealProgress=0) centered-right,
 * with slow idle animation and scroll-responsive parallax.
 * Disabled on mobile for performance.
 */
import React, { useRef, useEffect, useState, Suspense } from 'react';
import { Canvas, useFrame } from '@react-three/fiber';
import { useGLTF, useAnimations } from '@react-three/drei';
import * as THREE from 'three';

/* ── Hologram shader ─────────────────────────────────────────────────── */
const createHologramMaterial = (original) => {
  const mat = original.clone();
  mat.transparent = true;
  mat.side = THREE.DoubleSide;

  mat.onBeforeCompile = (shader) => {
    shader.uniforms.time = { value: 0 };

    shader.vertexShader = shader.vertexShader.replace(
      '#include <common>',
      `#include <common>
       varying vec3 vWorldPos;`
    );
    shader.vertexShader = shader.vertexShader.replace(
      '#include <worldpos_vertex>',
      `#include <worldpos_vertex>
       vWorldPos = (modelMatrix * vec4(transformed, 1.0)).xyz;`
    );

    shader.fragmentShader = shader.fragmentShader.replace(
      '#include <common>',
      `#include <common>
       uniform float time;
       varying vec3 vWorldPos;`
    );
    shader.fragmentShader = shader.fragmentShader.replace(
      '#include <dithering_fragment>',
      `#include <dithering_fragment>
       float fresnel = pow(1.0 - saturate(dot(normal, normalize(vViewPosition))), 2.5);
       vec3 holo = vec3(0.0, 0.5, 0.8) * 0.18 + fresnel * vec3(0.0, 0.8, 1.0) * 0.45;
       holo += step(0.94, sin(vWorldPos.y * 300.0 + time * 2.0)) * vec3(0.0, 0.3, 0.4) * 0.12;
       gl_FragColor.rgb = holo;
       gl_FragColor.a = 0.30 + fresnel * 0.45;`
    );

    mat.userData.shader = shader;
  };

  return mat;
};

/* ── Model component ─────────────────────────────────────────────────── */
const HologramAgent = ({
  modelPath,
  idleAnimationName,
  fallbackAnimationNames = [],
  reverseAnimation = false,
  basePos,
  baseRot,
  scale,
  scrollRef,
  swayPhase = 0,
}) => {
  const groupRef = useRef();
  const { scene, animations } = useGLTF(modelPath);
  const { actions, names } = useAnimations(animations, groupRef);
  const materialsRef = useRef(new Map());
  const timeRef = useRef(0);

  // Apply hologram materials
  useEffect(() => {
    if (!scene) return;
    scene.traverse((child) => {
      if (child.isMesh && child.material && !materialsRef.current.has(child.uuid)) {
        const holoMat = createHologramMaterial(child.material);
        materialsRef.current.set(child.uuid, holoMat);
        child.material = holoMat;
      }
    });
  }, [scene]);

  // Play idle animation
  useEffect(() => {
    if (!actions || names.length === 0) return;

    const candidates = [idleAnimationName, ...fallbackAnimationNames].filter(Boolean);
    const selectedName = candidates.find((name) => actions[name]) || names[0];
    const idle = actions[selectedName];

    if (idle) {
      Object.values(actions).forEach((action) => action.stop());
      idle.reset();

      if (reverseAnimation) {
        idle.setLoop(THREE.LoopRepeat, Infinity);
        idle.timeScale = -1;
        idle.time = idle.getClip().duration;
      } else {
        idle.setLoop(THREE.LoopPingPong, Infinity);
        idle.timeScale = 1;
        idle.time = 0;
      }

      idle.play();
    }

    return () => {
      if (idle) idle.stop();
    };
  }, [actions, names, idleAnimationName, fallbackAnimationNames, reverseAnimation]);

  useFrame((_, delta) => {
    timeRef.current += delta;

    // Update shader uniforms
    materialsRef.current.forEach((mat) => {
      if (mat.userData.shader) {
        mat.userData.shader.uniforms.time.value = timeRef.current;
      }
    });

    // Remove vertical scroll drift per user request, keep it fixed
    if (groupRef.current) {
      groupRef.current.position.y = basePos.y;
    }

    // Slow rotation
    if (groupRef.current) {
      groupRef.current.rotation.y = baseRot.y + Math.sin(timeRef.current * 0.15 + swayPhase) * 0.03;
    }
  });

  return (
    <group
      ref={groupRef}
      position={[basePos.x, basePos.y, basePos.z]}
      scale={scale}
      rotation={[baseRot.x, baseRot.y, baseRot.z]}
    >
      <primitive object={scene} />
    </group>
  );
};

const CT_CONFIG = {
  modelPath: '/images/Landing/CT_model.glb',
  idleAnimationName: 'ct_loadout_pistol01_walkup',
  fallbackAnimationNames: [],
  reverseAnimation: false,
  basePos: { x: 2.1, y: -1.45, z: 1.0 },
  baseRot: { x: 0, y: -0.35, z: 0 },
  scale: 1.45,
  swayPhase: 0,
};

const T_CONFIG = {
  modelPath: '/images/Landing/Tmodel.glb',
  idleAnimationName: 'ct_loadout_pistol01_walkup',
  fallbackAnimationNames: ['tm_professional_varf_balkanidle_balkan_idle'],
  reverseAnimation: false,
  basePos: { x: -3.1, y: -1.45, z: 1.0 },
  baseRot: { x: 0, y: 0.35, z: 0 },
  scale: 1.45,
  swayPhase: Math.PI,
};

/* ── Wrapper ─────────────────────────────────────────────────────────── */
const AgentBackground = ({ scrollY }) => {
  const scrollRef = useRef(0);
  const [isVisible, setIsVisible] = useState(true);

  useEffect(() => {
    if (scrollY !== undefined) scrollRef.current = scrollY;
  }, [scrollY]);

  // Disable on mobile & reduced motion
  useEffect(() => {
    const mql = window.matchMedia('(max-width: 768px)');
    const mqMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
    const check = () => setIsVisible(!mql.matches && !mqMotion.matches);
    check();
    mql.addEventListener('change', check);
    mqMotion.addEventListener('change', check);
    return () => {
      mql.removeEventListener('change', check);
      mqMotion.removeEventListener('change', check);
    };
  }, []);

  if (!isVisible) return null;

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 1,
        pointerEvents: 'none',
        opacity: 0.85,
      }}
      aria-hidden="true"
    >
      <Canvas
        dpr={[1, 1.5]}
        camera={{ position: [0.2, 0.95, 8], fov: 36 }}
        gl={{ antialias: true, alpha: true, powerPreference: 'default' }}
        frameloop="always"
      >
        <ambientLight intensity={0.3} />
        <directionalLight position={[3, 4, 5]} intensity={0.4} color="#06b6d4" />
        <Suspense fallback={null}>
          <HologramAgent
            modelPath={T_CONFIG.modelPath}
            idleAnimationName={T_CONFIG.idleAnimationName}
            fallbackAnimationNames={T_CONFIG.fallbackAnimationNames}
            reverseAnimation={T_CONFIG.reverseAnimation}
            basePos={T_CONFIG.basePos}
            baseRot={T_CONFIG.baseRot}
            scale={T_CONFIG.scale}
            swayPhase={T_CONFIG.swayPhase}
            scrollRef={scrollRef}
          />
          <HologramAgent
            modelPath={CT_CONFIG.modelPath}
            idleAnimationName={CT_CONFIG.idleAnimationName}
            fallbackAnimationNames={CT_CONFIG.fallbackAnimationNames}
            reverseAnimation={CT_CONFIG.reverseAnimation}
            basePos={CT_CONFIG.basePos}
            baseRot={CT_CONFIG.baseRot}
            scale={CT_CONFIG.scale}
            swayPhase={CT_CONFIG.swayPhase}
            scrollRef={scrollRef}
          />
        </Suspense>
      </Canvas>
    </div>
  );
};

useGLTF.preload('/images/Landing/CT_model.glb');
useGLTF.preload('/images/Landing/Tmodel.glb');

export default React.memo(AgentBackground);
