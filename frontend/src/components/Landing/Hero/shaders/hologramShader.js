/**
 * Hologram Shader for C4 Explosive - AUTHENTIC HOLOGRAM EFFECT
 * 
 * Inspired by Subnautica/Sci-Fi holograms:
 * - Wireframe/edge detection for holographic look
 * - Visible scanlines with glow
 * - True transparency (see-through effect)
 * - Fresnel edge glow
 * - Flicker and interference
 * - Clipping plane for materialization effect
 */

export const hologramVertexShader = /* glsl */ `
  varying vec2 vUv;
  varying vec3 vNormal;
  varying vec3 vPosition;
  varying vec3 vWorldPosition;
  varying vec3 vViewDirection;
  varying vec3 vLocalPosition;
  
  uniform float uTime;
  uniform float uGlitchIntensity;
  
  float hash(float n) {
    return fract(sin(n) * 43758.5453123);
  }
  
  void main() {
    vUv = uv;
    vNormal = normalize(normalMatrix * normal);
    vPosition = position;
    vLocalPosition = position;
    
    vec4 worldPosition = modelMatrix * vec4(position, 1.0);
    vWorldPosition = worldPosition.xyz;
    vViewDirection = normalize(cameraPosition - worldPosition.xyz);
    
    // Hologram jitter/displacement
    vec3 pos = position;
    
    // Subtle vertical wave distortion
    float wave = sin(position.y * 10.0 + uTime * 3.0) * 0.002;
    pos.x += wave;
    pos.z += wave * 0.5;
    
    // Glitch displacement - horizontal slices
    if (uGlitchIntensity > 0.5) {
      float glitchTime = floor(uTime * 30.0);
      float sliceY = floor(position.y * 20.0) / 20.0;
      float sliceRandom = hash(sliceY + glitchTime);
      
      if (sliceRandom > 0.92) {
        pos.x += (hash(glitchTime) - 0.5) * 0.15 * uGlitchIntensity;
      }
    }
    
    gl_Position = projectionMatrix * modelViewMatrix * vec4(pos, 1.0);
  }
`;

export const hologramFragmentShader = /* glsl */ `
  uniform float uTime;
  uniform float uTransition;
  uniform vec3 uHologramColor;
  uniform vec3 uHologramColor2;
  uniform float uFresnelPower;
  uniform float uScanlineSpeed;
  uniform float uScanlineCount;
  uniform float uGlitchIntensity;
  uniform float uFlickerSpeed;
  uniform sampler2D uTexture;
  uniform bool uHasTexture;
  uniform vec3 uMouseLightPos;
  uniform float uMouseLightIntensity;
  uniform float uClipY;  // For materialization effect
  uniform float uModelHeight;
  
  varying vec2 vUv;
  varying vec3 vNormal;
  varying vec3 vPosition;
  varying vec3 vWorldPosition;
  varying vec3 vViewDirection;
  varying vec3 vLocalPosition;
  
  // Noise functions
  float random(vec2 st) {
    return fract(sin(dot(st.xy, vec2(12.9898, 78.233))) * 43758.5453123);
  }
  
  float noise(vec2 st) {
    vec2 i = floor(st);
    vec2 f = fract(st);
    float a = random(i);
    float b = random(i + vec2(1.0, 0.0));
    float c = random(i + vec2(0.0, 1.0));
    float d = random(i + vec2(1.0, 1.0));
    vec2 u = f * f * (3.0 - 2.0 * f);
    return mix(a, b, u.x) + (c - a) * u.y * (1.0 - u.x) + (d - b) * u.x * u.y;
  }
  
  void main() {
    // ═══════════════════════════════════════════════════════════════════════
    // CLIP PLANE - Discard fragments below the materialization line
    // ═══════════════════════════════════════════════════════════════════════
    float normalizedY = (vLocalPosition.y + uModelHeight * 0.5) / uModelHeight;
    
    // If transition is active, clip fragments that are below the clip line
    // This creates a "reveal from bottom" effect
    if (uTransition > 0.0 && normalizedY < uClipY) {
      discard;
    }
    
    // Edge glow at materialization boundary
    float edgeDist = abs(normalizedY - uClipY);
    float edgeGlow = smoothstep(0.06, 0.0, edgeDist) * uTransition;
    
    // ═══════════════════════════════════════════════════════════════════════
    // FRESNEL - Edge detection for holographic rim
    // ═══════════════════════════════════════════════════════════════════════
    float fresnel = pow(1.0 - abs(dot(vViewDirection, vNormal)), uFresnelPower);
    
    // ═══════════════════════════════════════════════════════════════════════
    // SCANLINES - Horizontal moving lines (key hologram feature)
    // ═══════════════════════════════════════════════════════════════════════
    // Primary scanlines - thick, visible
    float scanY = vWorldPosition.y * uScanlineCount + uTime * uScanlineSpeed;
    float scanline1 = sin(scanY) * 0.5 + 0.5;
    scanline1 = pow(scanline1, 4.0);
    
    // Secondary scanlines - finer detail
    float scanline2 = sin(scanY * 3.0 - uTime * uScanlineSpeed * 1.5) * 0.5 + 0.5;
    scanline2 = pow(scanline2, 8.0) * 0.5;
    
    // Combine scanlines
    float scanlines = scanline1 * 0.6 + scanline2;
    
    // ═══════════════════════════════════════════════════════════════════════
    // FLICKER - Random intensity variation
    // ═══════════════════════════════════════════════════════════════════════
    float flicker = 1.0;
    float flickerTime = uTime * uFlickerSpeed;
    flicker *= 0.95 + 0.05 * sin(flickerTime * 50.0);
    flicker *= 0.97 + 0.03 * sin(flickerTime * 120.0 + 1.5);
    
    // Occasional strong flicker
    float strongFlicker = random(vec2(floor(uTime * 8.0), 0.0));
    if (strongFlicker > 0.95) {
      flicker *= 0.7 + random(vec2(floor(uTime * 30.0), 1.0)) * 0.3;
    }
    
    // ═══════════════════════════════════════════════════════════════════════
    // GLITCH BANDS - Horizontal interference
    // ═══════════════════════════════════════════════════════════════════════
    float glitch = 0.0;
    if (uGlitchIntensity > 0.0) {
      float glitchTime = floor(uTime * 10.0);
      float glitchRand = random(vec2(glitchTime, 0.0));
      
      if (glitchRand > 0.85) {
        float bandY = random(vec2(glitchTime, 1.0));
        float bandWidth = 0.02 + random(vec2(glitchTime, 2.0)) * 0.08;
        float distToBand = abs(fract(vWorldPosition.y * 0.5) - bandY);
        glitch = smoothstep(bandWidth, 0.0, distToBand) * uGlitchIntensity;
      }
    }
    
    // ═══════════════════════════════════════════════════════════════════════
    // HOLOGRAM COLOR - Much dimmer for subtle look
    // ═══════════════════════════════════════════════════════════════════════
    // Base color with vertical gradient - moderately visible
    vec3 holoColor = mix(uHologramColor, uHologramColor2, vUv.y * 0.5) * 0.4;
    
    // Scanlines - moderately visible
    holoColor += uHologramColor * scanlines * 0.15;
    
    // Fresnel rim glow - subtle edge highlight
    holoColor += uHologramColor * fresnel * 0.35;
    
    // Mouse reactive light - very subtle
    float distToMouse = length(vWorldPosition.xy - uMouseLightPos.xy);
    float mouseGlow = 1.0 / (1.0 + distToMouse * distToMouse * 0.5);
    holoColor += uHologramColor * mouseGlow * uMouseLightIntensity * 0.05;
    
    // Apply flicker
    holoColor *= flicker;
    
    // Glitch color shift - no white
    if (glitch > 0.0) {
      holoColor.r += glitch * 0.3;
      holoColor.g -= glitch * 0.1;
      // NO white mix - this causes white lines
      // holoColor = mix(holoColor, vec3(1.0), glitch * 0.3);
    }
    
    // ═══════════════════════════════════════════════════════════════════════
    // ALPHA - Clean transparent look without white edge lines
    // ═══════════════════════════════════════════════════════════════════════
    // Low-moderate base alpha for visible but not solid look
    float baseAlpha = 0.12;
    
    // Subtle fresnel alpha for edge visibility
    float fresnelAlpha = fresnel * 0.15;
    
    // Scanlines add minimal opacity
    float scanlineAlpha = scanlines * 0.03;
    
    // Combine - with subtle fresnel
    float alpha = baseAlpha + scanlineAlpha + fresnelAlpha;
    
    // Flicker affects alpha too
    alpha *= flicker;
    
    // Glitch - minimal effect on alpha
    alpha += glitch * 0.05;
    
    // Clamp - moderate max for visible but translucent look
    alpha = clamp(alpha, 0.0, 0.45);
    
    // ═══════════════════════════════════════════════════════════════════════
    // FINAL OUTPUT - Pure hologram with edge glow during transition
    // ═══════════════════════════════════════════════════════════════════════
    // Minimal noise for clean look
    float digitalNoise = random(vUv * 500.0 + uTime) * 0.01;
    holoColor += vec3(digitalNoise);
    
    // Add bright edge glow at clip boundary during materialization
    vec3 finalColor = holoColor;
    finalColor += uHologramColor * edgeGlow * 3.0;
    finalColor += vec3(1.0) * edgeGlow * 0.5; // White hot edge
    
    // Increase alpha at edge for visibility
    float finalAlpha = alpha + edgeGlow * 0.4;
    finalAlpha = clamp(finalAlpha, 0.0, 0.7);
    
    gl_FragColor = vec4(finalColor, finalAlpha);
  }
`;

// Wireframe edge shader for hologram outline effect
export const hologramWireframeVertexShader = /* glsl */ `
  varying vec2 vUv;
  varying vec3 vNormal;
  varying vec3 vWorldPosition;
  varying vec3 vViewDirection;
  
  uniform float uTime;
  
  void main() {
    vUv = uv;
    vNormal = normalize(normalMatrix * normal);
    
    vec4 worldPosition = modelMatrix * vec4(position, 1.0);
    vWorldPosition = worldPosition.xyz;
    vViewDirection = normalize(cameraPosition - worldPosition.xyz);
    
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
  }
`;

export const hologramWireframeFragmentShader = /* glsl */ `
  uniform float uTime;
  uniform vec3 uHologramColor;
  uniform float uOpacity;
  
  varying vec2 vUv;
  varying vec3 vNormal;
  varying vec3 vWorldPosition;
  varying vec3 vViewDirection;
  
  void main() {
    float fresnel = pow(1.0 - abs(dot(vViewDirection, vNormal)), 2.0);
    
    // Scanline for wireframe
    float scan = sin(vWorldPosition.y * 60.0 + uTime * 2.0) * 0.5 + 0.5;
    scan = pow(scan, 6.0);
    
    vec3 color = uHologramColor * (1.0 + fresnel * 1.5 + scan * 0.3);
    float alpha = (0.3 + fresnel * 0.5 + scan * 0.2) * uOpacity;
    
    // Flicker
    alpha *= 0.9 + 0.1 * sin(uTime * 60.0);
    
    gl_FragColor = vec4(color, alpha);
  }
`;

// Default uniform values
export const hologramUniforms = {
  uTime: { value: 0 },
  uTransition: { value: 0 },
  uHologramColor: { value: [0.0, 0.92, 0.95] }, // Bright Cyan
  uHologramColor2: { value: [0.0, 0.6, 1.0] }, // Blue accent
  uFresnelPower: { value: 2.0 },
  uScanlineSpeed: { value: 1.5 },
  uScanlineCount: { value: 40.0 },
  uGlitchIntensity: { value: 0.8 },
  uFlickerSpeed: { value: 1.0 },
  uTexture: { value: null },
  uHasTexture: { value: false },
  uMouseLightPos: { value: [0, 0, 5] },
  uMouseLightIntensity: { value: 1.5 },
  uClipY: { value: 0.0 },
  uModelHeight: { value: 2.0 }
};
