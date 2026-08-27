import React, { useEffect, useMemo, useRef } from 'react';
import { Edges } from '@react-three/drei';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';

const createOverviewTexture = (
  context,
  comparison,
  comparisonName,
  compareMode,
) => {
  const canvas = document.createElement('canvas');
  canvas.width = 1280;
  canvas.height = 320;
  const ctx = canvas.getContext('2d');

  const gradient = ctx.createLinearGradient(0, 0, canvas.width, canvas.height);
  gradient.addColorStop(0, '#0b2530');
  gradient.addColorStop(0.56, '#081820');
  gradient.addColorStop(1, '#0a2028');
  ctx.fillStyle = gradient;
  ctx.fillRect(0, 0, canvas.width, canvas.height);

  ctx.strokeStyle = 'rgba(95, 227, 255, .22)';
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.arc(150, 160, 82, 0, Math.PI * 2);
  ctx.stroke();
  ctx.beginPath();
  ctx.arc(150, 160, 54, 0, Math.PI * 2);
  ctx.stroke();

  for (let index = 0; index < 7; index += 1) {
    const angle = (Math.PI * 2 * index) / 7 - Math.PI / 2;
    const x = 150 + Math.cos(angle) * 81;
    const y = 160 + Math.sin(angle) * 81;
    ctx.fillStyle = index < 4 ? '#5cf2cd' : '#3b7180';
    ctx.beginPath();
    ctx.arc(x, y, 8, 0, Math.PI * 2);
    ctx.fill();
  }

  ctx.fillStyle = '#59e3ff';
  ctx.font = '600 26px "Chakra Petch", sans-serif';
  ctx.letterSpacing = '5px';
  ctx.fillText(compareMode ? 'COMPARACIÓN GLOBAL' : 'POOL COMPLETO', 286, 72);

  if (compareMode && comparison) {
    const difference = context.rating - comparison.rating;
    ctx.fillStyle = '#edf9fb';
    ctx.font = '600 61px "Chakra Petch", sans-serif';
    ctx.fillText(`TÚ  ${context.rating.toFixed(2)}`, 282, 152);
    ctx.fillStyle = '#b2c5cc';
    ctx.font = '600 43px "Chakra Petch", sans-serif';
    ctx.fillText(`${comparisonName.toUpperCase()}  ${comparison.rating.toFixed(2)}`, 286, 215);
    ctx.fillStyle = '#78939d';
    ctx.font = '600 25px "Chakra Petch", sans-serif';
    ctx.fillText(
      `${context.matches} / ${comparison.matches} PARTIDAS  ·  ${context.winRate}% / ${comparison.winRate}% WR`,
      286,
      268,
    );

    ctx.textAlign = 'right';
    ctx.fillStyle = difference >= 0 ? '#5cf2cd' : '#ff6f82';
    ctx.font = '700 88px "Chakra Petch", sans-serif';
    ctx.fillText(`${difference >= 0 ? '+' : ''}${difference.toFixed(2)}`, 1218, 157);
    ctx.fillStyle = '#77929d';
    ctx.font = '600 25px "Chakra Petch", sans-serif';
    ctx.fillText('DIF. RATING', 1218, 211);
  } else {
    ctx.fillStyle = '#edf9fb';
    ctx.font = '600 72px "Chakra Petch", sans-serif';
    ctx.fillText('TODOS LOS MAPAS', 282, 157);
    ctx.fillStyle = '#8ca6b0';
    ctx.font = '600 30px "Chakra Petch", sans-serif';
    ctx.fillText(`${context.matches} PARTIDAS  ·  ${context.winRate}% WIN RATE`, 286, 220);

    ctx.textAlign = 'right';
    ctx.fillStyle = '#5cf2cd';
    ctx.font = '600 82px "Chakra Petch", sans-serif';
    ctx.fillText(context.rating.toFixed(2), 1218, 150);
    ctx.fillStyle = '#77929d';
    ctx.font = '600 25px "Chakra Petch", sans-serif';
    ctx.fillText('RATING GLOBAL', 1218, 204);
  }

  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  texture.anisotropy = 8;
  return texture;
};

const GeneralOverviewNode = ({
  context,
  comparison,
  comparisonName,
  compareMode,
}) => {
  const groupRef = useRef(null);
  const texture = useMemo(
    () => createOverviewTexture(context, comparison, comparisonName, compareMode),
    [comparison, comparisonName, compareMode, context],
  );

  useEffect(() => () => texture.dispose(), [texture]);

  useFrame(({ clock }, delta) => {
    if (!groupRef.current) return;
    const targetY = 4.52 + Math.sin(clock.elapsedTime * 0.7) * 0.025;
    groupRef.current.position.y = THREE.MathUtils.damp(
      groupRef.current.position.y,
      targetY,
      4,
      delta,
    );
  });

  const accent = compareMode && comparison && context.rating < comparison.rating
    ? '#ff6f82'
    : '#5cf2cd';

  return (
    <group ref={groupRef} position={[0, 4.52, -3.65]}>
      <mesh>
        <boxGeometry args={[5.65, 1.42, 0.1]} />
        <meshStandardMaterial
          color="#0a2029"
          emissive="#0b3844"
          emissiveIntensity={0.22}
          metalness={0.42}
          roughness={0.46}
          transparent
          opacity={0.9}
        />
        <Edges color="#4a9bad" threshold={15} />
      </mesh>
      <mesh position={[0, 0, 0.07]}>
        <planeGeometry args={[5.48, 1.25]} />
        <meshBasicMaterial map={texture} toneMapped={false} />
      </mesh>
      <mesh position={[0, -0.69, 0.08]}>
        <planeGeometry args={[5.48, 0.022]} />
        <meshBasicMaterial color={accent} toneMapped={false} />
      </mesh>
      <pointLight position={[0, 0, 1]} color="#59e3ff" intensity={0.55} distance={4} />
    </group>
  );
};

export default GeneralOverviewNode;
