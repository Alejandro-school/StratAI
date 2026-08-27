import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Edges, useTexture } from '@react-three/drei';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';

const createMapLabelTexture = (
  context,
  accent,
  compareMode,
  comparisonContext,
  comparisonName,
) => {
  const canvas = document.createElement('canvas');
  canvas.width = 1024;
  canvas.height = 192;
  const ctx = canvas.getContext('2d');

  ctx.fillStyle = 'rgba(5, 17, 24, 0.94)';
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  ctx.textBaseline = 'middle';

  if (compareMode && comparisonContext) {
    const difference = context.rating - comparisonContext.rating;
    ctx.fillStyle = '#edf9fb';
    ctx.font = '600 52px "Chakra Petch", sans-serif';
    ctx.fillText(context.name.toUpperCase(), 42, 48);

    ctx.fillStyle = '#8faab4';
    ctx.font = '600 25px "Chakra Petch", sans-serif';
    ctx.fillText('TÚ', 42, 105);
    ctx.fillStyle = '#edf9fb';
    ctx.font = '600 38px "Chakra Petch", sans-serif';
    ctx.fillText(context.rating.toFixed(2), 100, 105);

    ctx.fillStyle = '#8faab4';
    ctx.font = '600 25px "Chakra Petch", sans-serif';
    ctx.fillText(comparisonName.toUpperCase(), 245, 105);
    ctx.fillStyle = '#d4e1e5';
    ctx.font = '600 38px "Chakra Petch", sans-serif';
    ctx.fillText(comparisonContext.rating.toFixed(2), 390, 105);

    ctx.fillStyle = '#78939d';
    ctx.font = '600 23px "Chakra Petch", sans-serif';
    ctx.fillText(
      `${context.matches} / ${comparisonContext.matches} PARTIDAS  ·  ${context.winRate}% / ${comparisonContext.winRate}% WR`,
      42,
      157,
    );

    ctx.fillStyle = accent;
    ctx.font = '700 60px "Chakra Petch", sans-serif';
    ctx.textAlign = 'right';
    ctx.fillText(`${difference >= 0 ? '+' : ''}${difference.toFixed(2)}`, 920, 80);
    ctx.fillStyle = '#77929d';
    ctx.font = '600 22px "Chakra Petch", sans-serif';
    ctx.fillText('DIF. RATING', 920, 130);
  } else {
    ctx.fillStyle = '#edf9fb';
    ctx.font = '600 70px "Chakra Petch", sans-serif';
    ctx.fillText(context.name.toUpperCase(), 42, 72);
    ctx.fillStyle = '#adc2ca';
    ctx.font = '600 40px "Chakra Petch", sans-serif';
    ctx.fillText(`${context.matches} PARTIDAS  ·  ${context.winRate}% WR`, 42, 145);

    ctx.fillStyle = accent;
    ctx.font = '600 74px "Chakra Petch", sans-serif';
    ctx.textAlign = 'right';
    ctx.fillText(context.rating.toFixed(2), 920, 68);
    ctx.fillStyle = '#77929d';
    ctx.font = '600 25px "Chakra Petch", sans-serif';
    ctx.letterSpacing = '4px';
    ctx.fillText('RATING', 920, 132);
  }

  ctx.fillStyle = accent;
  ctx.fillRect(42, 180, 70, 4);

  const labelTexture = new THREE.CanvasTexture(canvas);
  labelTexture.colorSpace = THREE.SRGBColorSpace;
  labelTexture.anisotropy = 8;
  return labelTexture;
};

const MapNode = ({
  context,
  position,
  rotationY,
  selected,
  dimmed,
  comparisonContext,
  comparisonName,
  compareMode,
  onHover,
  onSelect,
}) => {
  const groupRef = useRef(null);
  const radarRef = useRef(null);
  const pointerTarget = useRef(new THREE.Vector2());
  const [hovered, setHovered] = useState(false);
  const texture = useTexture(context.radar);
  const [initialPosition] = useState(position);
  const [initialRotation] = useState(rotationY);
  const baseY = position[1];

  useEffect(() => {
    texture.colorSpace = THREE.SRGBColorSpace;
    texture.anisotropy = 8;
    texture.needsUpdate = true;
  }, [texture]);

  useFrame((_, delta) => {
    if (!groupRef.current) return;
    const targetScale = hovered ? 1.1 : selected ? 1.01 : dimmed ? 0.96 : 1;
    const damping = 1 - Math.exp(-delta * 7);
    groupRef.current.scale.lerp(
      new THREE.Vector3(targetScale, targetScale, targetScale),
      damping,
    );
    groupRef.current.position.x = THREE.MathUtils.damp(
      groupRef.current.position.x,
      position[0],
      5.5,
      delta,
    );
    groupRef.current.position.y = THREE.MathUtils.damp(
      groupRef.current.position.y,
      baseY + (hovered ? 0.12 : 0),
      7,
      delta,
    );
    groupRef.current.position.z = THREE.MathUtils.damp(
      groupRef.current.position.z,
      position[2] + (hovered ? 1.35 : selected ? 0.28 : 0),
      7,
      delta,
    );
    groupRef.current.rotation.y = THREE.MathUtils.damp(
      groupRef.current.rotation.y,
      hovered ? 0 : rotationY,
      7,
      delta,
    );

    if (radarRef.current) {
      radarRef.current.position.x = THREE.MathUtils.damp(
        radarRef.current.position.x,
        hovered ? pointerTarget.current.x * 0.09 : 0,
        8,
        delta,
      );
      radarRef.current.position.y = THREE.MathUtils.damp(
        radarRef.current.position.y,
        0.28 + (hovered ? pointerTarget.current.y * 0.06 : 0),
        8,
        delta,
      );
    }
  });

  useEffect(() => {
    if (!groupRef.current) return;

    const renderOrder = hovered ? 100 : selected ? 50 : 0;
    groupRef.current.traverse((child) => {
      if (child.isMesh) child.renderOrder = renderOrder;
    });
  }, [hovered, selected]);

  useEffect(() => () => {
    document.body.style.cursor = 'default';
    onHover(null);
  }, [onHover]);

  const accent = compareMode && comparisonContext
    ? context.rating >= comparisonContext.rating ? '#5cf2cd' : '#ff6f82'
    : context.trend >= 0 ? '#5cf2cd' : '#ff6f82';
  const labelTexture = useMemo(
    () => createMapLabelTexture(
      context,
      accent,
      compareMode,
      comparisonContext,
      comparisonName,
    ),
    [accent, compareMode, comparisonContext, comparisonName, context],
  );

  useEffect(() => () => labelTexture.dispose(), [labelTexture]);

  return (
    <group ref={groupRef} position={initialPosition} rotation-y={initialRotation}>
      <mesh
        onClick={(event) => {
          event.stopPropagation();
          onSelect(context.id);
        }}
        onPointerMove={(event) => {
          if (!event.uv) return;
          pointerTarget.current.set(event.uv.x - 0.5, event.uv.y - 0.5);
        }}
        onPointerOver={(event) => {
          event.stopPropagation();
          setHovered(true);
          onHover(context.id);
          document.body.style.cursor = 'pointer';
        }}
        onPointerOut={() => {
          setHovered(false);
          pointerTarget.current.set(0, 0);
          onHover(null);
          document.body.style.cursor = 'default';
        }}
      >
        <boxGeometry args={[3.62, 2.92, 0.11]} />
        <meshStandardMaterial
          color="#0a1b24"
          emissive={selected || hovered ? '#123d49' : '#06151c'}
          emissiveIntensity={selected ? 0.38 : hovered ? 0.25 : 0.08}
          metalness={0.38}
          roughness={0.48}
          transparent
          opacity={dimmed ? 0.46 : hovered ? 0.92 : 0.78}
        />
        <Edges
          color={selected || hovered ? '#75dced' : '#28505e'}
          threshold={15}
        />
      </mesh>

      <group ref={radarRef} position={[0, 0.28, 0]}>
        <mesh position={[0.04, -0.03, 0.08]}>
          <planeGeometry args={[3.22, 2.12]} />
          <meshBasicMaterial
            map={texture}
            color={accent}
            transparent
            opacity={hovered ? 0.18 : selected ? 0.12 : 0.05}
            depthWrite={false}
            toneMapped={false}
          />
        </mesh>
        <mesh position={[0, 0, 0.13]}>
          <planeGeometry args={[3.22, 2.12]} />
          <meshBasicMaterial
            map={texture}
            color={selected || hovered ? '#ffffff' : '#c5d8dc'}
            transparent
            opacity={selected || hovered ? 1 : dimmed ? 0.38 : 0.8}
            alphaTest={0.015}
            toneMapped={false}
          />
        </mesh>
      </group>

      <mesh position={[0, -1.07, 0.1]}>
        <planeGeometry args={[3.18, 0.76]} />
        <meshBasicMaterial
          map={labelTexture}
          transparent
          opacity={dimmed ? 0.54 : 1}
          toneMapped={false}
        />
      </mesh>

      <mesh position={[0, -1.43, 0.13]}>
        <planeGeometry args={[3.18, 0.022]} />
        <meshBasicMaterial
          color={accent}
          transparent
          opacity={selected || hovered ? 0.95 : 0.38}
          toneMapped={false}
        />
      </mesh>
      {selected && (
        <pointLight position={[0, 0, 1]} color="#50dcff" intensity={0.85} distance={4} />
      )}
    </group>
  );
};

export default MapNode;
