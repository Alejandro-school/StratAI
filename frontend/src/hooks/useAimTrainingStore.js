/**
 * useAimTrainingStore - Zustand store for 3D Aim Training
 * 
 * Manages all state and metrics for the professional aim training experience:
 * - TTK (Time to Kill)
 * - Precision tracking
 * - Mouse pathing for AI flick analysis
 * - Heatmap data for impact visualization
 */
import { create } from 'zustand';

// Test Configuration
export const AIM_CONFIG = {
  TOTAL_TARGETS: 10,
  TARGET_LIFETIME: 2000, // ms before target disappears
  MIN_SPAWN_DELAY: 300,
  MAX_SPAWN_DELAY: 600,
  COUNTDOWN_SECONDS: 3,
  ARENA_RADIUS: 6, // 3D space radius for target spawning
  TARGET_HEIGHT_MIN: 0.5,
  TARGET_HEIGHT_MAX: 2.5,
  RECOIL_INTENSITY: 0.02,
  RECOIL_DURATION: 0.1,
};

// Test States
export const TEST_STATES = {
  START: 'START',
  COUNTDOWN: 'COUNTDOWN',
  PLAYING: 'PLAYING',
  RESULTS: 'RESULTS',
};

const initialState = {
  // Test State
  testState: TEST_STATES.START,
  countdown: AIM_CONFIG.COUNTDOWN_SECONDS,
  
  // Current Target
  currentTarget: null, // { id, position, spawnTime, modelType }
  targetsSpawned: 0,
  
  // Core Metrics
  kills: 0,
  totalClicks: 0,
  
  // TTK Data
  ttkData: [], // [{ targetId, spawnTime, killTime, ttk }]
  
  // Precision Data
  precision: {
    hits: 0,
    misses: 0,
    headshots: 0,
    bodyshots: 0,
    accuracy: 0,
  },
  
  // Mouse Pathing (for AI flick analysis)
  mousePath: [], // [{ x, y, timestamp, targetVisible }]
  
  // Heatmap Data
  impactPoints: [], // [{ relativeX, relativeY, zone }]
  
  // Visual Feedback
  lastHitPosition: null,
  showHitmarker: false,
  hitmarkerType: 'normal', // 'normal', 'headshot', 'kill'
  
  // Camera State
  cameraShake: { x: 0, y: 0 },
};

const useAimTrainingStore = create((set, get) => ({
  ...initialState,
  
  // ═══════════════════════════════════════════════════════════════════════════
  // State Management
  // ═══════════════════════════════════════════════════════════════════════════
  
  startCountdown: () => {
    set({ testState: TEST_STATES.COUNTDOWN, countdown: AIM_CONFIG.COUNTDOWN_SECONDS });
  },
  
  decrementCountdown: () => {
    const { countdown } = get();
    if (countdown > 1) {
      set({ countdown: countdown - 1 });
    } else {
      set({ testState: TEST_STATES.PLAYING, countdown: 0 });
    }
  },
  
  startPlaying: () => {
    set({ testState: TEST_STATES.PLAYING });
  },
  
  endTest: () => {
    const { precision, kills, totalClicks } = get();
    // Calculate final accuracy
    const finalAccuracy = totalClicks > 0 
      ? Math.round((precision.hits / totalClicks) * 100) 
      : 0;
    
    set({ 
      testState: TEST_STATES.RESULTS,
      precision: { ...precision, accuracy: finalAccuracy }
    });
  },
  
  resetTest: () => {
    set(initialState);
  },
  
  // ═══════════════════════════════════════════════════════════════════════════
  // Target Management
  // ═══════════════════════════════════════════════════════════════════════════
  
  spawnTarget: () => {
    const { targetsSpawned } = get();
    
    // Random position in 3D space - limited range for visibility
    // X between -3 and +3 (within FOV 70 at distance 5-8)
    // Y between 0.8 and 2.2 (standing height range)
    // Z between -5 and -8 (in front of camera)
    const x = (Math.random() - 0.5) * 6; // -3 to +3
    const y = 0.8 + Math.random() * 1.4; // 0.8 to 2.2
    const z = -5 - Math.random() * 3; // -5 to -8
    
    const position = { x, y, z };
    
    // Alternate between terrorist and agent models
    const modelType = targetsSpawned % 2 === 0 ? 'terrorist' : 'agent';
    
    console.log('🎯 Spawning target:', { position, modelType, targetsSpawned });
    
    const newTarget = {
      id: Date.now(),
      position,
      spawnTime: Date.now(),
      modelType,
    };
    
    set({ 
      currentTarget: newTarget,
      targetsSpawned: targetsSpawned + 1,
    });
    
    return newTarget;
  },
  
  clearTarget: () => {
    set({ currentTarget: null });
  },
  
  // ═══════════════════════════════════════════════════════════════════════════
  // Hit Detection & Scoring
  // ═══════════════════════════════════════════════════════════════════════════
  
  registerHit: (hitData) => {
    const { currentTarget, kills, precision, ttkData, impactPoints } = get();
    
    if (!currentTarget) return;
    
    const killTime = Date.now();
    // Use reactionTime from hitData (from visibility) if available, else calculate from spawn
    const ttk = hitData.reactionTime || (killTime - currentTarget.spawnTime);
    
    // Determine hit zone (based on Y position relative to model)
    const zone = hitData.localY > 1.5 ? 'headshot' : 'bodyshot';
    const isHeadshot = zone === 'headshot';
    
    // Update TTK data
    const newTtkData = [...ttkData, {
      targetId: currentTarget.id,
      spawnTime: currentTarget.spawnTime,
      killTime,
      ttk, // Now uses visibility-based reaction time
    }];
    
    // Update precision
    const newPrecision = {
      ...precision,
      hits: precision.hits + 1,
      headshots: precision.headshots + (isHeadshot ? 1 : 0),
      bodyshots: precision.bodyshots + (isHeadshot ? 0 : 1),
    };
    
    // Update heatmap
    const newImpactPoints = [...impactPoints, {
      relativeX: hitData.localX || 0,
      relativeY: hitData.localY || 0,
      zone,
    }];
    
    set({
      kills: kills + 1,
      precision: newPrecision,
      ttkData: newTtkData,
      impactPoints: newImpactPoints,
      currentTarget: null,
      lastHitPosition: hitData.worldPosition,
      showHitmarker: true,
      hitmarkerType: isHeadshot ? 'headshot' : 'kill',
    });
    
    // Auto-hide hitmarker
    setTimeout(() => {
      set({ showHitmarker: false });
    }, 150);
    
    return { isHeadshot, ttk };
  },
  
  registerMiss: (position) => {
    const { totalClicks, precision } = get();
    
    set({
      totalClicks: totalClicks + 1,
      precision: {
        ...precision,
        misses: precision.misses + 1,
      },
    });
  },
  
  registerClick: () => {
    set(state => ({ totalClicks: state.totalClicks + 1 }));
  },
  
  // ═══════════════════════════════════════════════════════════════════════════
  // Mouse Tracking (for AI analysis)
  // ═══════════════════════════════════════════════════════════════════════════
  
  recordMousePosition: (x, y) => {
    const { mousePath, currentTarget, testState } = get();
    
    if (testState !== TEST_STATES.PLAYING) return;
    
    // Throttle to reduce data size (every 16ms = ~60fps)
    const lastEntry = mousePath[mousePath.length - 1];
    if (lastEntry && Date.now() - lastEntry.timestamp < 16) return;
    
    set({
      mousePath: [...mousePath, {
        x,
        y,
        timestamp: Date.now(),
        targetVisible: !!currentTarget,
      }],
    });
  },
  
  // ═══════════════════════════════════════════════════════════════════════════
  // Camera Effects
  // ═══════════════════════════════════════════════════════════════════════════
  
  triggerRecoil: () => {
    const intensity = AIM_CONFIG.RECOIL_INTENSITY;
    set({
      cameraShake: {
        x: (Math.random() - 0.5) * intensity,
        y: Math.random() * intensity * 2,
      },
    });
    
    // Reset after duration
    setTimeout(() => {
      set({ cameraShake: { x: 0, y: 0 } });
    }, AIM_CONFIG.RECOIL_DURATION * 1000);
  },
  
  // ═══════════════════════════════════════════════════════════════════════════
  // Computed Values
  // ═══════════════════════════════════════════════════════════════════════════
  
  getAverageRTT: () => {
    const { ttkData } = get();
    if (ttkData.length === 0) return 0;
    const total = ttkData.reduce((sum, d) => sum + d.ttk, 0);
    return Math.round(total / ttkData.length);
  },
  
  getAccuracy: () => {
    const { precision, totalClicks } = get();
    if (totalClicks === 0) return 0;
    return Math.round((precision.hits / totalClicks) * 100);
  },
  
  getHeadshotRate: () => {
    const { precision } = get();
    const totalHits = precision.hits;
    if (totalHits === 0) return 0;
    return Math.round((precision.headshots / totalHits) * 100);
  },
  
  // Get results for saving
  getResults: () => {
    const state = get();
    return {
      kills: state.kills,
      totalTargets: AIM_CONFIG.TOTAL_TARGETS,
      accuracy: state.getAccuracy(),
      avgTTK: state.getAverageRTT(),
      headshotRate: state.getHeadshotRate(),
      ttkData: state.ttkData,
      impactPoints: state.impactPoints,
      mousePath: state.mousePath,
    };
  },
}));

export default useAimTrainingStore;
