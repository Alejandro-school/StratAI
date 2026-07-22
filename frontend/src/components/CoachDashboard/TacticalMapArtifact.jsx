import React, { useEffect, useMemo, useRef } from 'react';
import { Edges, useTexture } from '@react-three/drei';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';

const VERTEX_SHADER = `
  varying vec2 vUv;

  void main() {
    vUv = uv;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
  }
`;

const FRAGMENT_SHADER = `
  uniform sampler2D uMap;
  uniform vec3 uAccent;
  uniform float uTime;
  uniform float uReveal;
  varying vec2 vUv;

  float hash(vec2 point) {
    return fract(sin(dot(point, vec2(127.1, 311.7))) * 43758.5453123);
  }

  void main() {
    vec4 source = texture2D(uMap, vUv);
    float luminance = dot(source.rgb, vec3(0.299, 0.587, 0.114));
    vec3 graded = mix(source.rgb, vec3(luminance), 0.38);
    graded = mix(graded * vec3(0.48, 0.56, 0.68), graded, 0.58);

    float fineGrid = step(0.985, max(fract(vUv.x * 26.0), fract(vUv.y * 16.0)));
    float scanPosition = fract(uTime * 0.075);
    float scan = 1.0 - smoothstep(0.0, 0.025, abs(vUv.y - scanPosition));
    float vignette = smoothstep(0.78, 0.22, distance(vUv, vec2(0.5)));
    float noise = hash(floor(vUv * vec2(72.0, 42.0)));
    float reveal = smoothstep(noise - 0.16, noise + 0.14, uReveal);

    vec3 color = graded * (0.72 + vignette * 0.34);
    color += uAccent * fineGrid * 0.08;
    color += uAccent * scan * 0.24;

    gl_FragColor = vec4(color, source.a * reveal * 0.94);
  }
`;

const createSignalPositions = () => {
  const positions = new Float32Array(180 * 3);

  for (let index = 0; index < 180; index += 1) {
    const offset = index * 3;
    const angle = index * 2.39996;
    const radius = 2.7 + ((index * 37) % 100) / 68;
    positions[offset] = Math.cos(angle) * radius;
    positions[offset + 1] = Math.sin(angle) * radius * 0.62;
    positions[offset + 2] = -0.35 + ((index * 17) % 31) / 42;
  }

  return positions;
};

const MapSurface = ({ image, accent }) => {
  const materialRef = useRef(null);
  const revealRef = useRef(0);
  const texture = useTexture(image);
  const uniforms = useMemo(() => ({
    uMap: { value: texture },
    uAccent: { value: new THREE.Color(accent) },
    uTime: { value: 0 },
    uReveal: { value: 0 }
  }), [accent, texture]);

  useEffect(() => {
    texture.colorSpace = THREE.SRGBColorSpace;
    texture.anisotropy = 4;
    texture.needsUpdate = true;
  }, [texture]);

  useFrame((state, delta) => {
    if (!materialRef.current) return;
    revealRef.current = Math.min(1, revealRef.current + delta * 1.45);
    materialRef.current.uniforms.uTime.value = state.clock.elapsedTime;
    materialRef.current.uniforms.uReveal.value = revealRef.current;
  });

  return (
    <mesh position={[0, 0, 0.075]} renderOrder={3}>
      <planeGeometry args={[5.7, 3.2, 1, 1]} />
      <shaderMaterial
        ref={materialRef}
        uniforms={uniforms}
        vertexShader={VERTEX_SHADER}
        fragmentShader={FRAGMENT_SHADER}
        transparent
        depthWrite={false}
        toneMapped={false}
      />
    </mesh>
  );
};

const SignalField = ({ accent }) => {
  const pointsRef = useRef(null);
  const positions = useMemo(createSignalPositions, []);

  useFrame((_, delta) => {
    if (!pointsRef.current) return;
    pointsRef.current.rotation.z += delta * 0.018;
  });

  return (
    <points ref={pointsRef} position={[0, 0, 0.16]}>
      <bufferGeometry>
        <bufferAttribute attach="attributes-position" args={[positions, 3]} />
      </bufferGeometry>
      <pointsMaterial color={accent} size={0.022} transparent opacity={0.48} sizeAttenuation />
    </points>
  );
};

const ScanBeam = ({ accent }) => {
  const beamRef = useRef(null);

  useFrame((state) => {
    if (!beamRef.current) return;
    beamRef.current.position.y = ((state.clock.elapsedTime * 0.72) % 3.8) - 1.9;
  });

  return (
    <mesh ref={beamRef} position={[0, -1.7, 0.11]} renderOrder={4}>
      <planeGeometry args={[5.9, 0.035]} />
      <meshBasicMaterial color={accent} transparent opacity={0.72} depthWrite={false} />
    </mesh>
  );
};

const TacticalMapArtifact = ({ image, isWin }) => {
  const accent = isWin ? '#7cf0bd' : '#ff7e74';

  return (
    <group>
      {[0.18, 0.3, 0.42].map((depth, index) => (
        <mesh key={depth} position={[0, 0, -depth]} scale={1 - index * 0.028}>
          <boxGeometry args={[5.9, 3.4, 0.055]} />
          <meshBasicMaterial color="#17202b" transparent opacity={0.18 - index * 0.035} />
          <Edges color={accent} threshold={18} opacity={0.12} transparent />
        </mesh>
      ))}

      <mesh position={[0, 0, 0]}>
        <boxGeometry args={[5.92, 3.42, 0.12]} />
        <meshStandardMaterial color="#101720" metalness={0.68} roughness={0.34} />
        <Edges color={accent} threshold={18} opacity={0.56} transparent />
      </mesh>

      <MapSurface key={image} image={image} accent={accent} />
      <ScanBeam accent={accent} />
      <SignalField accent={accent} />
    </group>
  );
};

export default TacticalMapArtifact;
