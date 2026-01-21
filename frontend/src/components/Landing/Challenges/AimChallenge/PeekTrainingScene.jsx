/**
 * PeekTrainingScene - CS2-style peek training environment
 * 
 * Features:
 * - Central wall for cover (positioned close to player at z=-3)
 * - Clean modern training arena aesthetic
 * - Optimized lighting for target visibility
 */
import React, { useRef, useMemo } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';


const PeekTrainingScene = ({ enableLights = true }) => {
  return (
    <>
      {enableLights && (
        <>
          {/* Ambient light - stronger for better visibility */}
          <ambientLight intensity={0.8} color="#c0d4e8" />
          
          {/* Main directional light (sun) */}
          <directionalLight
            position={[5, 15, 8]}
            intensity={1.8}
            color="#fff8e6"
            castShadow
            shadow-mapSize={[2048, 2048]}
            shadow-camera-far={30}
            shadow-camera-left={-8}
            shadow-camera-right={8}
            shadow-camera-top={8}
            shadow-camera-bottom={-8}
          />
          
          {/* Fill light from opposite side */}
          <directionalLight
            position={[-3, 8, 5]}
            intensity={0.6}
            color="#97c4eb"
          />
        </>
      )}
      
      {/* Ground - Training room floor */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, 0, 0]} receiveShadow>
        <planeGeometry args={[30, 30]} />
        <meshStandardMaterial 
          color="#6b7a85" 
          roughness={0.75}
          metalness={0.15}
        />
      </mesh>
      
      {/* Grid lines on floor */}
      <GridFloor />
      
      {/* Back wall */}
      <mesh position={[0, 2.5, -8]} receiveShadow>
        <boxGeometry args={[30, 5, 0.3]} />
        <meshStandardMaterial color="#9aa8b0" roughness={0.5} />
      </mesh>
      
      {/* Side walls with gradient */}
      <mesh position={[-12, 2.5, -2]} receiveShadow>
        <boxGeometry args={[0.3, 5, 14]} />
        <meshStandardMaterial color="#8a98a0" roughness={0.5} />
      </mesh>
      <mesh position={[12, 2.5, -2]} receiveShadow>
        <boxGeometry args={[0.3, 5, 14]} />
        <meshStandardMaterial color="#8a98a0" roughness={0.5} />
      </mesh>
      
      {/* CENTER COVER - Modern training wall - CLOSER to player */}
      <group position={[0, 0, -3]}>
        {/* Main wall with modern materials */}
        <mesh position={[0, 1.35, 0]} castShadow receiveShadow>
          <boxGeometry args={[3.6, 2.7, 0.25]} />
          <meshStandardMaterial 
            color="#e87000" 
            roughness={0.35}
            metalness={0.15}
            emissive="#ff6600"
            emissiveIntensity={0.08}
          />
        </mesh>
        
        {/* Top accent bar */}
        <mesh position={[0, 2.75, 0.13]}>
          <boxGeometry args={[3.6, 0.08, 0.02]} />
          <meshStandardMaterial 
            color="#ffffff" 
            emissive="#ffffff"
            emissiveIntensity={0.4}
          />
        </mesh>
        
        {/* Corner accents */}
        <mesh position={[-1.75, 0.15, 0.13]}>
          <boxGeometry args={[0.08, 0.3, 0.02]} />
          <meshStandardMaterial color="#ffffff" emissive="#ffffff" emissiveIntensity={0.3} />
        </mesh>
        <mesh position={[1.75, 0.15, 0.13]}>
          <boxGeometry args={[0.08, 0.3, 0.02]} />
          <meshStandardMaterial color="#ffffff" emissive="#ffffff" emissiveIntensity={0.3} />
        </mesh>
        
        {/* Base/floor contact */}
        <mesh position={[0, 0.04, 0]} castShadow>
          <boxGeometry args={[3.8, 0.08, 0.35]} />
          <meshStandardMaterial color="#2a2a2a" metalness={0.6} roughness={0.4} />
        </mesh>
      </group>
      
      {/* Left side cover block with modern styling */}
      <mesh position={[-5.5, 0.65, -5]} castShadow receiveShadow>
        <boxGeometry args={[1.8, 1.3, 2.5]} />
        <meshStandardMaterial color="#b8c2c8" roughness={0.6} metalness={0.1} />
      </mesh>
      
      {/* Right side cover block */}
      <mesh position={[5.5, 0.65, -5]} castShadow receiveShadow>
        <boxGeometry args={[1.8, 1.3, 2.5]} />
        <meshStandardMaterial color="#b8c2c8" roughness={0.6} metalness={0.1} />
      </mesh>
      
      {/* Sky background */}
      <mesh position={[0, 10, -12]}>
        <planeGeometry args={[60, 30]} />
        <meshBasicMaterial color="#7fc4eb" />
      </mesh>
    </>
  );
};

// Grid lines on the floor
const GridFloor = () => {
  const gridRef = useRef();
  
  const lines = useMemo(() => {
    const geometry = new THREE.BufferGeometry();
    const positions = [];
    const gridSize = 20;
    const divisions = 20;
    const step = gridSize / divisions;
    
    // Horizontal lines
    for (let i = -gridSize / 2; i <= gridSize / 2; i += step) {
      positions.push(-gridSize / 2, 0.01, i);
      positions.push(gridSize / 2, 0.01, i);
    }
    
    // Vertical lines
    for (let i = -gridSize / 2; i <= gridSize / 2; i += step) {
      positions.push(i, 0.01, -gridSize / 2);
      positions.push(i, 0.01, gridSize / 2);
    }
    
    geometry.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3));
    return geometry;
  }, []);
  
  return (
    <lineSegments ref={gridRef} geometry={lines}>
      <lineBasicMaterial color="#5a6a75" transparent opacity={0.3} />
    </lineSegments>
  );
};

export default PeekTrainingScene;
