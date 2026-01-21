/**
 * ImageTarget - 2D CS2-style enemy target with texture
 * 
 * Uses actual CT/T model images for realistic targets.
 * Targets spawn BEHIND the wall edges and peek out from left/right.
 */
import React, { useRef, useEffect, useState, useCallback, useMemo } from 'react';
import { useFrame, useLoader, useThree } from '@react-three/fiber';
import * as THREE from 'three';

// CS2 Movement Speeds (units per second)
const CS2_SPEEDS = {
  CROUCH: 1.0,      // very slow peek
  WALK: 1.8,        // slow/careful peek  
  JOG: 2.5,         // medium speed
  RUN: 3.5,         // fast strafe
  SPRINT: 4.5,      // sprint
};

// Get random speed based on difficulty distribution
export const getRandomSpeed = () => {
  const rand = Math.random();
  if (rand < 0.15) return CS2_SPEEDS.CROUCH;
  if (rand < 0.40) return CS2_SPEEDS.WALK;
  if (rand < 0.70) return CS2_SPEEDS.JOG;
  if (rand < 0.90) return CS2_SPEEDS.RUN;
  return CS2_SPEEDS.SPRINT;
};

const ImageTarget = ({ 
  id,
  side = 'left',
  speed = 2.0,
  targetType = 'terrorist',
  wallPosition = [0, 0, -3],
  onHit,
  onMiss,
  isActive = true,
}) => {
  const groupRef = useRef();
  const meshRef = useRef();
  const [isHit, setIsHit] = useState(false);
  const [hasEscaped, setHasEscaped] = useState(false);
  const [opacity, setOpacity] = useState(1);
  const [scale, setScale] = useState(1);
  const { raycaster, camera, pointer } = useThree();
  
  // Load textures
  const ctTexture = useLoader(THREE.TextureLoader, '/images/Landing/CTModel.png');
  const tTexture = useLoader(THREE.TextureLoader, '/images/Landing/TModel.png');
  
  const texture = useMemo(() => {
    const tex = targetType === 'terrorist' ? tTexture : ctTexture;
    tex.colorSpace = THREE.SRGBColorSpace;
    return tex;
  }, [targetType, ctTexture, tTexture]);
  

  // Wall dimensions - wall is 3.6 units wide at z=-3
  const wallHalfWidth = 1.8;
  const wallX = wallPosition[0];
  const wallZ = wallPosition[2];
  
  // Target dimensions
  const targetHeight = 1.8;
  const targetWidth = 0.9;
  
  // PEEK MECHANIC: Targets start BEHIND the wall, hidden by the wall
  // They start far to one side and move toward center, becoming visible when they exit the wall edge
  // Z position: Same as wall or slightly behind so wall occludes them
  const zPos = wallZ; // Same Z as wall
  
  // Start position: Hidden by the wall (target center behind wall)
  // Start at center of wall (hidden) and move outward
  const startX = side === 'left'
    ? wallX - 0.3  // Start just left of center (hidden behind wall)
    : wallX + 0.3; // Start just right of center (hidden behind wall)
  
  // Wall edge positions (where target becomes visible)
  const wallEdgeX = side === 'left' 
    ? wallX - wallHalfWidth  // Left edge of wall
    : wallX + wallHalfWidth; // Right edge of wall
  
  // End position: Target peeks out and stays visible longer (extended distance)
  const endX = side === 'left' 
    ? wallX - wallHalfWidth - targetWidth * 4  // More time visible on left
    : wallX + wallHalfWidth + targetWidth * 4; // More time visible on right
  
  // Y position with slight variation
  const yPos = useRef(0.9 + (Math.random() - 0.5) * 0.15).current;
  
  // Current X position ref
  const posXRef = useRef(startX);
  
  // Track when target became visible (for accurate TTK)
  const visibleTimeRef = useRef(null);
  const [isVisible, setIsVisible] = useState(false);
  
  // Initialize position
  useEffect(() => {
    if (isActive && !isHit && !hasEscaped && groupRef.current) {
      posXRef.current = startX;
      visibleTimeRef.current = null;
      setIsVisible(false);
      groupRef.current.position.set(startX, yPos, zPos);
      setOpacity(1);
      setScale(1);
      console.log(`🎯 ImageTarget ${id}: side=${side}, speed=${speed.toFixed(1)}, type=${targetType}`);
    }
  }, [isActive, id, side, startX, speed, targetType, isHit, hasEscaped, yPos, zPos]);
  
  // Movement animation
  useFrame((state, delta) => {
    if (!groupRef.current || !isActive || isHit || hasEscaped) return;
    
    // Direction: left side moves LEFT (-X) to peek out, right side moves RIGHT (+X) to peek out
    const direction = side === 'left' ? -1 : 1;
    posXRef.current += direction * speed * delta;
    
    // Check if target just became visible (exited wall edge)
    if (!isVisible) {
      const nowVisible = side === 'left'
        ? posXRef.current < wallEdgeX  // Left target visible when past left edge
        : posXRef.current > wallEdgeX; // Right target visible when past right edge
      
      if (nowVisible) {
        visibleTimeRef.current = Date.now();
        setIsVisible(true);
        console.log(`👁️ Target ${id} now VISIBLE at x=${posXRef.current.toFixed(2)}`);
      }
    }
    
    // Check if escaped (passed the end position, fully visible and beyond)
    const escaped = side === 'left' 
      ? posXRef.current < endX 
      : posXRef.current > endX;
    
    if (escaped) {
      console.log(`❌ Target ${id} escaped!`);
      setHasEscaped(true);
      if (onMiss) onMiss(id);
      return;
    }
    
    // Update position
    groupRef.current.position.x = posXRef.current;
    
    // Subtle bobbing
    const bob = Math.sin(state.clock.elapsedTime * speed * 4) * 0.01;
    groupRef.current.position.y = yPos + bob;
  });
  
  // Hit/death animation
  useFrame((state, delta) => {
    if (!isHit || !groupRef.current) return;
    
    if (opacity > 0) {
      const newOpacity = Math.max(opacity - delta * 8, 0);
      const newScale = scale + delta * 4;
      setOpacity(newOpacity);
      setScale(newScale);
      
      // Update shader uniform for opacity
      if (meshRef.current && meshRef.current.material && meshRef.current.material.uniforms) {
        meshRef.current.material.uniforms.opacity.value = newOpacity;
      }
      groupRef.current.scale.setScalar(newScale);
    }
  });
  
  // Handle click - robust click handler
  const handleClick = useCallback((event) => {
    if (isHit || hasEscaped) return;
    
    event.stopPropagation();
    
    // Calculate TTK from when target became visible (more accurate)
    const hitTime = Date.now();
    const reactionTime = visibleTimeRef.current ? hitTime - visibleTimeRef.current : 0;
    
    console.log(`💥 Target ${id} HIT! Reaction time: ${reactionTime}ms`);
    
    setIsHit(true);
    
    if (onHit) {
      onHit({
        id,
        worldPosition: event.point || new THREE.Vector3(posXRef.current, yPos, zPos),
        isHeadshot: event.point ? event.point.y > yPos + targetHeight * 0.25 : false,
        speed,
        visibleTime: visibleTimeRef.current, // When target became visible
        hitTime: hitTime,                     // When hit occurred
        reactionTime: reactionTime,           // Time from visible to hit (ms)
      });
    }
  }, [id, isHit, hasEscaped, onHit, speed, yPos, zPos, targetHeight]);
  
  // Handle pointer down as backup for click
  const handlePointerDown = useCallback((event) => {
    handleClick(event);
  }, [handleClick]);
  
  if (!isActive) return null;
  if (hasEscaped) return null;
  if (isHit && opacity <= 0) return null;
  
  return (
    <group ref={groupRef} position={[startX, yPos, zPos]}>
      {/* Main target sprite with proper transparency */}
      <mesh 
        ref={meshRef}
        onClick={handleClick}
        onPointerDown={handlePointerDown}
      >
        <planeGeometry args={[targetWidth, targetHeight]} />
        <meshBasicMaterial 
          map={texture}
          transparent={true}
          alphaTest={0.5}
          opacity={opacity}
          side={THREE.DoubleSide}
          depthWrite={false}
        />
      </mesh>
      
      {/* Larger invisible hitbox for easier clicking */}
      <mesh 
        visible={false} 
        onClick={handleClick}
        onPointerDown={handlePointerDown}
      >
        <boxGeometry args={[targetWidth * 1.3, targetHeight * 1.1, 0.5]} />
        <meshBasicMaterial transparent opacity={0} />
      </mesh>
    </group>
  );
};

export default ImageTarget;
