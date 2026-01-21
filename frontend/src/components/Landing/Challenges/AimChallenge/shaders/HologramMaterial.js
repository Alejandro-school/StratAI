/**
 * HologramMaterial - Custom GLSL shader for holographic CS2 targets
 * 
 * Features:
 * - Fresnel edge glow effect
 * - Animated horizontal scanlines
 * - Team-based color schemes (blue CT / orange T)
 * - Hit flash effect
 * - Transparency with depth
 */
import * as THREE from 'three';
import { shaderMaterial } from '@react-three/drei';

// Vertex Shader
const vertexShader = /* glsl */ `
  varying vec3 vNormal;
  varying vec3 vViewPosition;
  varying vec2 vUv;
  varying vec3 vWorldPosition;
  
  void main() {
    vUv = uv;
    vNormal = normalize(normalMatrix * normal);
    
    vec4 worldPosition = modelMatrix * vec4(position, 1.0);
    vWorldPosition = worldPosition.xyz;
    
    vec4 mvPosition = modelViewMatrix * vec4(position, 1.0);
    vViewPosition = -mvPosition.xyz;
    
    gl_Position = projectionMatrix * mvPosition;
  }
`;

// Fragment Shader
const fragmentShader = /* glsl */ `
  uniform float uTime;
  uniform vec3 uColor;
  uniform vec3 uFresnelColor;
  uniform float uFresnelPower;
  uniform float uScanlineSpeed;
  uniform float uScanlineCount;
  uniform float uOpacity;
  uniform float uHitFlash;
  uniform float uDisintegrate;
  
  varying vec3 vNormal;
  varying vec3 vViewPosition;
  varying vec2 vUv;
  varying vec3 vWorldPosition;
  
  // Noise function for disintegration
  float rand(vec2 co) {
    return fract(sin(dot(co.xy, vec2(12.9898, 78.233))) * 43758.5453);
  }
  
  void main() {
    // Fresnel effect (edge glow)
    vec3 viewDir = normalize(vViewPosition);
    float fresnel = pow(1.0 - abs(dot(viewDir, vNormal)), uFresnelPower);
    
    // Scanline effect
    float scanline = sin((vWorldPosition.y + uTime * uScanlineSpeed) * uScanlineCount) * 0.5 + 0.5;
    scanline = smoothstep(0.3, 0.7, scanline);
    
    // Base hologram color with scanlines
    vec3 baseColor = mix(uColor * 0.3, uColor, scanline * 0.7 + 0.3);
    
    // Add fresnel glow
    vec3 finalColor = mix(baseColor, uFresnelColor, fresnel * 0.8);
    
    // Hit flash effect (white flash)
    finalColor = mix(finalColor, vec3(1.0), uHitFlash);
    
    // Alpha calculation
    float alpha = uOpacity * (0.4 + fresnel * 0.6 + scanline * 0.2);
    
    // Disintegration effect
    if (uDisintegrate > 0.0) {
      float noise = rand(vUv * 100.0 + uTime);
      if (noise < uDisintegrate) {
        discard;
      }
      alpha *= (1.0 - uDisintegrate);
    }
    
    gl_FragColor = vec4(finalColor, alpha);
  }
`;

// Create the shader material using drei's shaderMaterial helper
const HologramMaterialImpl = shaderMaterial(
  {
    uTime: 0,
    uColor: new THREE.Color(0x00f0ff),
    uFresnelColor: new THREE.Color(0x00ffff),
    uFresnelPower: 2.0,
    uScanlineSpeed: 0.5,
    uScanlineCount: 50.0,
    uOpacity: 0.85,
    uHitFlash: 0.0,
    uDisintegrate: 0.0,
  },
  vertexShader,
  fragmentShader
);

// Extend Three.js to recognize our material
HologramMaterialImpl.key = THREE.MathUtils.generateUUID();

// Color presets for different team types
export const HOLOGRAM_COLORS = {
  terrorist: {
    base: new THREE.Color(0xff6600), // Orange
    fresnel: new THREE.Color(0xff9933),
  },
  agent: {
    base: new THREE.Color(0x00a8ff), // Blue
    fresnel: new THREE.Color(0x00ffff),
  },
  neutral: {
    base: new THREE.Color(0x00ff88), // Green
    fresnel: new THREE.Color(0x88ffcc),
  },
};

export { HologramMaterialImpl };
export default HologramMaterialImpl;
