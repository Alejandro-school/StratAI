import React, { Suspense, useRef } from 'react';
import { AdaptiveDpr } from '@react-three/drei';
import { Canvas, useFrame } from '@react-three/fiber';
import * as THREE from 'three';
import TacticalMapArtifact from './TacticalMapArtifact';

const CameraResponsiveRig = ({ children }) => {
  const rigRef = useRef(null);

  useFrame((state, delta) => {
    if (!rigRef.current) return;

    const targetX = -0.16 + state.pointer.y * 0.045;
    const targetY = state.pointer.x * 0.055;
    rigRef.current.rotation.x = THREE.MathUtils.damp(rigRef.current.rotation.x, targetX, 4.2, delta);
    rigRef.current.rotation.y = THREE.MathUtils.damp(rigRef.current.rotation.y, targetY, 4.2, delta);
    rigRef.current.position.y = THREE.MathUtils.damp(
      rigRef.current.position.y,
      Math.sin(state.clock.elapsedTime * 0.42) * 0.045,
      2.8,
      delta
    );
  });

  return (
    <group ref={rigRef} rotation={[-0.16, 0, -0.035]}>
      {children}
    </group>
  );
};

const SceneEnvironment = ({ isWin }) => {
  const accent = isWin ? '#7cf0bd' : '#ff7e74';

  return (
    <>
      <ambientLight intensity={0.7} />
      <directionalLight position={[3, 4, 6]} intensity={1.2} color="#dceaff" />
      <pointLight position={[-4, -1, 3]} intensity={12} distance={9} color={accent} />

      <gridHelper
        args={[12, 24, '#263648', '#17212c']}
        position={[0, 0.05, -1.15]}
        rotation={[Math.PI / 2, 0, 0]}
      />

      {[2.25, 2.85, 3.5].map((radius, index) => (
        <mesh key={radius} position={[0, 0, -0.82 - index * 0.05]}>
          <ringGeometry args={[radius, radius + 0.008, 96]} />
          <meshBasicMaterial color={accent} transparent opacity={0.12 - index * 0.025} depthWrite={false} />
        </mesh>
      ))}
    </>
  );
};

const TacticalMatchScene = ({ mapImage, isWin }) => (
  <Canvas
    dpr={[1, 1.5]}
    camera={{ position: [0, 0.25, 8.6], fov: 35, near: 0.1, far: 40 }}
    gl={{ antialias: true, alpha: true, powerPreference: 'high-performance' }}
    performance={{ min: 0.65 }}
  >
    <AdaptiveDpr pixelated />
    <SceneEnvironment isWin={isWin} />
    <Suspense fallback={null}>
      <CameraResponsiveRig>
        <TacticalMapArtifact image={mapImage} isWin={isWin} />
      </CameraResponsiveRig>
    </Suspense>
  </Canvas>
);

export default TacticalMatchScene;
