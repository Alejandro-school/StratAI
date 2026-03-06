// frontend/src/components/Stats/Replay2DViewer.jsx
// INNOVATIVE 2D Replay Viewer - Cinematic Esports Experience
// Features: Framer Motion animations, dual team panels, glassmorphism controls

import React, { useRef, useEffect, useState, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Play, Pause, SkipBack, SkipForward, Maximize2, ChevronLeft, ChevronRight, ZoomIn, ZoomOut, RotateCcw, Shield, PenTool, Skull } from "lucide-react";
import useReplaySyncStore from "../../stores/useReplaySyncStore";
import "../../styles/Stats/replay2DViewer.css";

// ============================================================================
// CONFIGURATION
// ============================================================================

const MAP_CONFIGS = {
  de_dust2: { pos_x: -2476, pos_y: 3239, scale: 4.4 },
  de_mirage: { pos_x: -3230, pos_y: 1713, scale: 5.0 },
  de_inferno: { pos_x: -2087, pos_y: 3870, scale: 4.9 },
  de_ancient: { pos_x: -2953, pos_y: 2164, scale: 5.0 },
  de_anubis: { pos_x: -2796, pos_y: 3328, scale: 5.22 },
  de_nuke: { pos_x: -3453, pos_y: 2887, scale: 7.0 },
  de_overpass: { pos_x: -4831, pos_y: 1781, scale: 5.2 },
  de_vertigo: { pos_x: -3168, pos_y: 1762, scale: 4.0 },
  de_train: { pos_x: -2477, pos_y: 2392, scale: 4.7 },
};

const THEME = {
  ct: { primary: "#22d3ee", glow: "rgba(34, 211, 238, 0.45)", dark: "#083344" },
  t: { primary: "#eab308", glow: "rgba(234, 179, 8, 0.45)", dark: "#422006" },
};

const WEAPON_ICON_MAP = {
  // Rifles
  'AK-47': 'weapon_ak47', 'M4A4': 'weapon_m4a1', 'M4A1-S': 'weapon_m4a1_silencer',
  'AWP': 'weapon_awp', 'AUG': 'weapon_aug', 'FAMAS': 'weapon_famas', 
  'Galil AR': 'weapon_galilar', 'SSG 08': 'weapon_ssg08', 'SG 553': 'weapon_sg553',
  'SCAR-20': 'weapon_scar20', 'G3SG1': 'weapon_g3sg1',
  // Pistols  
  'Desert Eagle': 'weapon_deagle', 'USP-S': 'weapon_usp_silencer', 'Glock-18': 'weapon_glock',
  'P2000': 'weapon_hkp2000', 'P250': 'weapon_p250', 'Five-SeveN': 'weapon_fiveseven',
  'Tec-9': 'weapon_tec9', 'CZ75-Auto': 'weapon_cz75a', 'Dual Berettas': 'weapon_elite',
  'R8 Revolver': 'weapon_revolver',
  // SMGs
  'MAC-10': 'weapon_mac10', 'MP9': 'weapon_mp9', 'MP7': 'weapon_mp7', 
  'UMP-45': 'weapon_ump45', 'PP-Bizon': 'weapon_bizon', 'P90': 'weapon_p90',
  'MP5-SD': 'weapon_mp5sd',
  // Shotguns & Heavy
  'Nova': 'weapon_nova', 'XM1014': 'weapon_xm1014', 'Sawed-Off': 'weapon_sawedoff',
  'MAG-7': 'weapon_mag7', 'M249': 'weapon_m249', 'Negev': 'weapon_negev',
  // Equipment
  'Knife': 'weapon_knife', 'C4': 'weapon_c4',
  'HE Grenade': 'weapon_hegrenade', 'Flashbang': 'weapon_flashbang',
  'Smoke Grenade': 'weapon_smokegrenade', 'Molotov': 'weapon_molotov',
  'Incendiary Grenade': 'weapon_incgrenade', 'Decoy Grenade': 'weapon_decoy',
};

// ============================================================================
// UTILITIES
// ============================================================================

function translateCoords(gameX, gameY, mapConfig, canvasSize) {
  const { pos_x, pos_y, scale } = mapConfig;
  const pixelX = (gameX - pos_x) / scale;
  const pixelY = (pos_y - gameY) / scale;
  const ratio = canvasSize / 1024;
  return { x: pixelX * ratio, y: pixelY * ratio };
}

function lerp(a, b, t) { return a + (b - a) * t; }

function lerpAngle(a, b, t) {
  while (a > 180) a -= 360; while (a < -180) a += 360;
  while (b > 180) b -= 360; while (b < -180) b += 360;
  let diff = b - a;
  if (diff > 180) diff -= 360; if (diff < -180) diff += 360;
  return a + diff * t;
}



function getWeaponIconPath(weapon) {
  const iconName = WEAPON_ICON_MAP[weapon];
  return iconName ? `/images/weapons/${iconName}.png` : null;
}

function parseTimestampToSeconds(timestamp) {
  if (!timestamp || typeof timestamp !== 'string') return null;
  const parts = timestamp.trim().split(':').map(Number);
  if (parts.some(Number.isNaN)) return null;
  if (parts.length === 1) return parts[0];
  if (parts.length === 2) return parts[0] * 60 + parts[1];
  return null;
}

function getClosestFrameIndexByTick(frames, tick) {
  if (!Array.isArray(frames) || frames.length === 0 || tick == null) return -1;
  let bestIndex = 0;
  let bestDistance = Number.POSITIVE_INFINITY;
  frames.forEach((frame, index) => {
    const distance = Math.abs((frame?.tick ?? 0) - tick);
    if (distance < bestDistance) {
      bestDistance = distance;
      bestIndex = index;
    }
  });
  return bestIndex;
}

function getClosestFrameIndexByTimeRemaining(frames, targetSeconds) {
  if (!Array.isArray(frames) || frames.length === 0 || targetSeconds == null) return -1;
  let bestIndex = 0;
  let bestDistance = Number.POSITIVE_INFINITY;
  frames.forEach((frame, index) => {
    const distance = Math.abs((frame?.time_remaining ?? 0) - targetSeconds);
    if (distance < bestDistance) {
      bestDistance = distance;
      bestIndex = index;
    }
  });
  return bestIndex;
}

const weaponIconCache = new Map();
function loadWeaponIcon(weapon) {
  const iconPath = getWeaponIconPath(weapon);
  if (!iconPath) return null;
  if (weaponIconCache.has(iconPath)) return weaponIconCache.get(iconPath);
  const img = new Image();
  img.src = iconPath;
  weaponIconCache.set(iconPath, img);
  return img;
}

// ============================================================================
// ANIMATED PLAYER CARD COMPONENT
// ============================================================================

const PlayerCard = React.memo(({ player, team }) => {
  const isDead = !player.alive;
  const hp = player.health || 0;
  const weaponIcon = loadWeaponIcon(player.weapon);
  const tc = team === 'CT' ? THEME.ct : THEME.t;

  // Health colour
  const hpColor = hp > 60 ? '#22c55e' : hp > 25 ? '#eab308' : '#ef4444';

  return (
    <div
      className={`rv-player ${team.toLowerCase()} ${isDead ? 'dead' : ''}`}
      style={{ '--tc': tc.primary }}
    >
      {/* Row 1: avatar dot + name + hp number */}
      <div className="rv-player-top">
        <div className="rv-avatar" />
        <span className="rv-name">{player.name?.substring(0, 14) || 'Player'}</span>
        <span className="rv-hp-num" style={{ color: isDead ? '#64748b' : hpColor }}>
          {isDead ? <Skull size={11} /> : hp}
        </span>
      </div>

      {/* Row 2: health bar full-width */}
      <div className="rv-hp-bar-bg">
        <div
          className="rv-hp-bar"
          style={{ width: `${hp}%`, background: isDead ? '#334155' : hpColor }}
        />
      </div>

      {/* Row 3: weapon + indicators + money */}
      <div className="rv-player-bottom">
        <div className="rv-weapon-row">
          {weaponIcon && weaponIcon.complete && (
            <img src={weaponIcon.src} alt="" className="rv-weapon-img" />
          )}
          {player.armor > 0 && <Shield size={10} color={tc.primary} />}
          {player.has_defuse_kit && <PenTool size={10} color="#94a3b8" />}
        </div>
        <span className="rv-money">${player.money || 0}</span>
      </div>
    </div>
  );
});

// ============================================================================
// KILL FEED OVERLAY COMPONENT
// ============================================================================

const KillFeedItem = ({ kill, onComplete }) => {
  useEffect(() => {
    const timer = setTimeout(onComplete, 5000);
    return () => clearTimeout(timer);
  }, [onComplete]);
  
  const killerColor = kill.killer_team === 'CT' ? THEME.ct.primary : THEME.t.primary;
  const victimColor = kill.victim_team === 'CT' ? THEME.ct.primary : THEME.t.primary;
  const weaponIcon = loadWeaponIcon(kill.weapon);
  
  return (
    <motion.div
      initial={{ opacity: 0, x: 30, scale: 0.9 }}
      animate={{ opacity: 1, x: 0, scale: 1 }}
      exit={{ opacity: 0, x: -20, scale: 0.9 }}
      transition={{ type: "spring", stiffness: 400, damping: 25 }}
      className="kill-feed-item"
    >
      <span className="kill-name killer" style={{ color: killerColor }}>{kill.killer_name?.substring(0, 10)}</span>
      <div className="kill-weapon-container">
        {weaponIcon && weaponIcon.complete ? (
          <img src={weaponIcon.src} alt="" className="kill-weapon-icon" />
        ) : (
          <span className="kill-weapon-text">{kill.weapon || '?'}</span>
        )}
        {kill.headshot && <span className="headshot-badge">HS</span>}
      </div>
      <span className="kill-name victim" style={{ color: victimColor }}>{kill.victim_name?.substring(0, 10)}</span>
    </motion.div>
  );
};

const KillFeedOverlay = ({ events, currentTick }) => {
  const [visibleKills, setVisibleKills] = useState([]);
  
  useEffect(() => {
    if (!events) return;
    const recentKills = events.filter(e => 
      e.type === 'kill' && 
      currentTick - e.tick >= 0 && 
      currentTick - e.tick < 256
    );
    setVisibleKills(recentKills.slice(-5));
  }, [events, currentTick]);
  
  return (
    <div className="kill-feed-overlay">
      <AnimatePresence>
        {visibleKills.map((kill, i) => (
          <KillFeedItem 
            key={`${kill.tick}-${i}`} 
            kill={kill}
            onComplete={() => {}}
          />
        ))}
      </AnimatePresence>
    </div>
  );
};

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export default function Replay2DViewer({ 
  matchId, 
  replayData: preloadedData, 
  initialRound = 1,
  externalControl = null,
  scenarioContext = null
}) {
  // Global Store Sync
  const { isPlaying: isAiPlaying, activeClip, annotations, updateCurrentTick } = useReplaySyncStore();

  // Refs
  const canvasRef = useRef(null);
  const containerRef = useRef(null);
  const animationRef = useRef(null);
  const mapImageRef = useRef(null);
  const currentTimeRef = useRef(0);
  const roundCacheRef = useRef(new Map());
  const tracersRef = useRef([]);
  const prevFrameRef = useRef(null);
  const explosionsRef = useRef([]);
  const killLinesRef = useRef([]);
  const prevEventsRef = useRef(new Set());
  const pendingTickRef = useRef(null);
  const pendingTimestampRef = useRef(null);
  const screenShakeRef = useRef({ active: false, startTime: 0, intensity: 0 });
  
  // State
  const [metadata, setMetadata] = useState(null);
  const [roundsSummary, setRoundsSummary] = useState([]);
  const [currentRoundData, setCurrentRoundData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [loadingRound, setLoadingRound] = useState(false);
  const [currentRound, setCurrentRound] = useState(initialRound);
  const [isPlaying, setIsPlaying] = useState(false);
  const [playbackSpeed, setPlaybackSpeed] = useState(1);
  const [currentTime, setCurrentTime] = useState(0);
  const [canvasSize, setCanvasSize] = useState(700);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [zoom, setZoom] = useState(1);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [isDraggingTimeline, setIsDraggingTimeline] = useState(false);
  
  // Derived
  const mapName = metadata?.map_name;
  const mapConfig = MAP_CONFIGS[mapName] || MAP_CONFIGS.de_mirage;
  const totalFrames = currentRoundData?.frames?.length || 0;
  const sampleRateMs = metadata?.sample_rate_ms || 62.5;

  // ============================================================================
  // DATA LOADING
  // ============================================================================
  
  useEffect(() => {
    if (preloadedData) {
      setMetadata(preloadedData.metadata);
      const summary = preloadedData.rounds?.map((r, i) => ({
        round: r.round || i + 1,
        winner: r.winner,
        events: r.events || []
      })) || [];
      setRoundsSummary(summary);
      if (preloadedData.rounds?.[initialRound - 1]) {
        setCurrentRoundData(preloadedData.rounds[initialRound - 1]);
        preloadedData.rounds.forEach((r, i) => roundCacheRef.current.set(i + 1, r));
      }
      setLoading(false);
      return;
    }
    
    if (!matchId) return;
    
    const fetchMetadata = async () => {
      setLoading(true);
      try {
        const res = await fetch(`http://localhost:8000/match/${matchId}/replay/metadata`, { credentials: 'include' });
        if (!res.ok) throw new Error('Error loading replay');
        const data = await res.json();
        setMetadata(data.metadata);
        if (data.rounds_summary) {
          setRoundsSummary(data.rounds_summary);
          // Load the first available round using its actual round number
          const firstRoundNum = data.rounds_summary[0]?.round || initialRound;
          setCurrentRound(1); // Index 1 = first round in summary
          await loadRoundData(firstRoundNum);
        } else {
          await loadRoundData(initialRound);
        }
      } catch (err) {
        console.error(err);
      } finally {
        setLoading(false);
      }
    };
    fetchMetadata();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [matchId, preloadedData, initialRound]);

  const loadRoundData = useCallback(async (roundNum) => {
    console.log('[Replay2D] loadRoundData called, round:', roundNum);
    const cached = roundCacheRef.current.get(roundNum);
    if (cached?.frames?.length) {
      console.log('[Replay2D] Using cached data for round', roundNum, 'frames:', cached.frames.length);
      setCurrentRoundData(cached);
      return;
    }
    if (preloadedData?.rounds?.[roundNum - 1]?.frames?.length) {
      console.log('[Replay2D] Using preloaded data for round', roundNum);
      setCurrentRoundData(preloadedData.rounds[roundNum - 1]);
      return;
    }
    
    setLoadingRound(true);
    try {
      const res = await fetch(`http://localhost:8000/match/${matchId}/replay/round/${roundNum}`, { credentials: 'include' });
      if (!res.ok) throw new Error('Error loading round');
      const data = await res.json();
      console.log('[Replay2D] Fetched round', roundNum, 'from API, frames:', data?.frames?.length);
      roundCacheRef.current.set(roundNum, data);
      setCurrentRoundData(data);
    } catch (err) { console.error(err); }
    finally { setLoadingRound(false); }
  }, [matchId, preloadedData]);

  useEffect(() => { 
    if (metadata && currentRound > 0 && roundsSummary.length > 0) {
      // Resolve actual round number from the summary (currentRound is 1-based index)
      const actualRoundNum = roundsSummary[currentRound - 1]?.round || currentRound;
      console.log('[Replay2D] Round index:', currentRound, '=> actual round num:', actualRoundNum);
      loadRoundData(actualRoundNum);
      // Clear shot deduplication when changing rounds
      prevEventsRef.current.clear();
      tracersRef.current = [];
    }
  }, [currentRound, metadata, roundsSummary, loadRoundData]);

  const seekToFrameIndex = useCallback((frameIndex, framesLength) => {
    if (frameIndex < 0 || framesLength <= 1) return;
    const normalizedTime = Math.max(0, Math.min(1, frameIndex / (framesLength - 1)));
    setCurrentTime(normalizedTime);
    currentTimeRef.current = normalizedTime;
  }, []);

  useEffect(() => {
    if (!currentRoundData?.frames?.length) return;
    const frames = currentRoundData.frames;

    if (pendingTickRef.current != null) {
      const targetIndex = getClosestFrameIndexByTick(frames, pendingTickRef.current);
      if (targetIndex >= 0) {
        seekToFrameIndex(targetIndex, frames.length);
      }
      pendingTickRef.current = null;
    }

    if (pendingTimestampRef.current) {
      const seconds = parseTimestampToSeconds(pendingTimestampRef.current);
      if (seconds != null) {
        const targetIndex = getClosestFrameIndexByTimeRemaining(frames, seconds);
        if (targetIndex >= 0) {
          seekToFrameIndex(targetIndex, frames.length);
        }
      }
      pendingTimestampRef.current = null;
    }
  }, [currentRoundData, seekToFrameIndex]);

  useEffect(() => {
    if (!externalControl) return;

    if (typeof externalControl.round === 'number' && externalControl.round !== currentRound) {
      setCurrentRound(externalControl.round);
      setCurrentTime(0);
      currentTimeRef.current = 0;
      setIsPlaying(false);
    }

    if (typeof externalControl.time === 'number') {
      const normalizedTime = Math.max(0, Math.min(1, externalControl.time));
      setCurrentTime(normalizedTime);
      currentTimeRef.current = normalizedTime;
    }

    if (typeof externalControl.seekDeltaSeconds === 'number' && totalFrames > 1) {
      const roundDurationSeconds = (totalFrames * sampleRateMs) / 1000;
      const deltaNormalized = externalControl.seekDeltaSeconds / Math.max(roundDurationSeconds, 1);
      const nextTime = Math.max(0, Math.min(1, currentTimeRef.current + deltaNormalized));
      setCurrentTime(nextTime);
      currentTimeRef.current = nextTime;
    }

    if (externalControl.tick != null) {
      if (currentRoundData?.frames?.length) {
        const targetIndex = getClosestFrameIndexByTick(currentRoundData.frames, externalControl.tick);
        if (targetIndex >= 0) {
          seekToFrameIndex(targetIndex, currentRoundData.frames.length);
        }
      } else {
        pendingTickRef.current = externalControl.tick;
      }
    }

    if (externalControl.timestamp) {
      if (currentRoundData?.frames?.length) {
        const seconds = parseTimestampToSeconds(externalControl.timestamp);
        if (seconds != null) {
          const targetIndex = getClosestFrameIndexByTimeRemaining(currentRoundData.frames, seconds);
          if (targetIndex >= 0) {
            seekToFrameIndex(targetIndex, currentRoundData.frames.length);
          }
        }
      } else {
        pendingTimestampRef.current = externalControl.timestamp;
      }
    }

    if (typeof externalControl.play === 'boolean') {
      setIsPlaying(externalControl.play);
    }
  }, [
    externalControl,
    currentRound,
    currentRoundData,
    totalFrames,
    sampleRateMs,
    seekToFrameIndex
  ]);

  // ============================================================================
  // AI CLIP LISTENER (Zustand Global State)
  // ============================================================================
  useEffect(() => {
    if (activeClip && currentRoundData?.frames?.length > 0) {
      // Si el startTick es pequeño (ej. 0.35), asumimos que es un tiempo normalizado (0.0 a 1.0)
      if (activeClip.startTick <= 1) {
        setCurrentTime(activeClip.startTick);
        currentTimeRef.current = activeClip.startTick;
        setIsPlaying(true);
      } else {
        // Find frame index closest to the AI's requested startTick if it's an absolute tick
        const targetIndex = getClosestFrameIndexByTick(currentRoundData.frames, activeClip.startTick);
        if (targetIndex >= 0) {
          seekToFrameIndex(targetIndex, currentRoundData.frames.length);
          setIsPlaying(true); // Ensure local playback starts when AI dictates
        }
      }
    }
  }, [activeClip, currentRoundData, seekToFrameIndex]);

  // Sync local isPlaying with global AI playback state (auto-pause when endTick reached)
  useEffect(() => {
    if (!isAiPlaying && activeClip) {
      // Global store auto-paused the clip — stop local playback too
      setIsPlaying(false);
    }
  }, [isAiPlaying, activeClip]);

  // Load map image
  useEffect(() => {
    if (!mapName) return;
    const img = new Image();
    img.onload = () => { mapImageRef.current = img; };
    img.src = `/maps/${mapName}_radar_psd.png`;
  }, [mapName]);

  const [dpr, setDpr] = useState(1);

  // Canvas resize
  useEffect(() => {
    const updateSize = () => {
      setDpr(window.devicePixelRatio || 1);
      if (containerRef.current) {
        const rect = containerRef.current.getBoundingClientRect();
        // Subtract side panels (260px each) + gaps
        const availableWidth = rect.width - 520 - 48;
        // Subtract top bar (~48px) + bottom dock (~120px)
        const availableHeight = rect.height - 170;
        // Canvas is square — fit into whichever is smaller
        const size = Math.min(availableWidth, availableHeight);
        setCanvasSize(Math.max(size, 280));
      }
    };
    updateSize();
    window.addEventListener('resize', updateSize);
    return () => window.removeEventListener('resize', updateSize);
  }, [isFullscreen]);

  // ============================================================================
  // INTERPOLATION
  // ============================================================================
  
  const getInterpolatedFrame = useCallback(() => {
    if (!currentRoundData?.frames || totalFrames === 0) return null;
    const frameProgress = currentTime * (totalFrames - 1);
    const frameIndex = Math.floor(frameProgress);
    const t = frameProgress - frameIndex;
    const frame = currentRoundData.frames[frameIndex];
    const nextFrame = currentRoundData.frames[Math.min(frameIndex + 1, totalFrames - 1)];
    if (!frame || frameIndex >= totalFrames - 1 || !nextFrame) return frame;
    
    const interpolatedPlayers = frame.players.map(player => {
      const nextPlayer = nextFrame.players.find(p => p.steam_id === player.steam_id);
      if (!nextPlayer) return player;
      return {
        ...player,
        x: lerp(player.x, nextPlayer.x, t),
        y: lerp(player.y, nextPlayer.y, t),
        yaw: lerpAngle(player.yaw, nextPlayer.yaw, t),
      };
    });
    
    return { ...frame, players: interpolatedPlayers, interpolatedTick: lerp(frame.tick, nextFrame.tick, t) };
  }, [currentRoundData, totalFrames, currentTime]);

  // ============================================================================
  // CANVAS DRAWING
  // ============================================================================
  
  const draw = useCallback((time) => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    const size = canvasSize; // Logic size
    
    // Clear using physical pixels
    ctx.clearRect(0, 0, size * dpr, size * dpr);
    ctx.save();
    
    // Scale for High DPI
    ctx.scale(dpr, dpr);

    // Apply screen shake if active
    let shakeX = 0, shakeY = 0;
    if (screenShakeRef.current.active) {
      const shakeAge = performance.now() - screenShakeRef.current.startTime;
      if (shakeAge < screenShakeRef.current.duration) {
        const shakeFade = 1 - (shakeAge / screenShakeRef.current.duration);
        const shakeIntensity = screenShakeRef.current.intensity * shakeFade;
        shakeX = (Math.random() - 0.5) * shakeIntensity * 2;
        shakeY = (Math.random() - 0.5) * shakeIntensity * 2;
      } else {
        screenShakeRef.current.active = false;
      }
    }
    
    ctx.translate(pan.x + size / 2 + shakeX, pan.y + size / 2 + shakeY);
    ctx.scale(zoom, zoom);
    ctx.translate(-size / 2, -size / 2);
    
    // Draw map
    ctx.fillStyle = '#0D1117';
    ctx.fillRect(0, 0, size, size);
    if (mapImageRef.current) {
      ctx.globalAlpha = 0.9;
      ctx.drawImage(mapImageRef.current, 0, 0, size, size);
      ctx.globalAlpha = 1;
    }
    
    const frameData = currentRoundData?.frames ? (() => {
      const frameProgress = time * (totalFrames - 1);
      const frameIndex = Math.floor(frameProgress);
      const t = frameProgress - frameIndex;
      const frame = currentRoundData.frames[frameIndex];
      const nextFrame = currentRoundData.frames[Math.min(frameIndex + 1, totalFrames - 1)];
      if (!frame) return null;
      if (!nextFrame || frameIndex >= totalFrames - 1) return frame;
      
      const interpolatedPlayers = frame.players.map(player => {
        const nextPlayer = nextFrame.players.find(p => p.steam_id === player.steam_id);
        if (!nextPlayer) return player;
        return { ...player, x: lerp(player.x, nextPlayer.x, t), y: lerp(player.y, nextPlayer.y, t), yaw: lerpAngle(player.yaw, nextPlayer.yaw, t) };
      });
      return { ...frame, players: interpolatedPlayers };
    })() : null;
    
    // Draw effects (smokes, fires) - Professional style with timers
    if (frameData?.active_effects) {
      const nowSeconds = performance.now() / 1000;
      frameData.active_effects.forEach(effect => {
        const pos = translateCoords(effect.x, effect.y, mapConfig, size);
        const baseRadius = (144 / mapConfig.scale) * (size / 1024);
        
        if (effect.type === 'smoke') {
          // =====================================================
          // SMOKE - Gray circle with animated timer ring
          // =====================================================
          const smokeRadius = baseRadius * 0.9;
          const timeRemaining = effect.time_remaining || 15;
          const maxDuration = 18; // Smoke lasts ~18 seconds
          const progress = Math.max(0, Math.min(1, timeRemaining / maxDuration));
          
          // Animated rotation for the smoke cloud
          const rotationAngle = (nowSeconds * 0.2) % (Math.PI * 2);

          ctx.save();
          ctx.translate(pos.x, pos.y);
          ctx.rotate(rotationAngle);
          
          // Main smoke fill - smoother and more voluminous
          const gradient = ctx.createRadialGradient(0, 0, 0, 0, 0, smokeRadius);
          gradient.addColorStop(0, 'rgba(130, 135, 145, 0.9)');
          gradient.addColorStop(0.4, 'rgba(110, 115, 125, 0.75)');
          gradient.addColorStop(0.8, 'rgba(90, 95, 105, 0.4)');
          gradient.addColorStop(1, 'rgba(80, 85, 90, 0)');
          
          // Draw a stylized puffy cloud using overlapping circles
          ctx.beginPath();
          ctx.arc(0, 0, smokeRadius * 0.8, 0, Math.PI * 2);
          ctx.arc(smokeRadius * 0.3, smokeRadius * 0.2, smokeRadius * 0.6, 0, Math.PI * 2);
          ctx.arc(-smokeRadius * 0.25, -smokeRadius * 0.3, smokeRadius * 0.7, 0, Math.PI * 2);
          ctx.arc(-smokeRadius * 0.4, smokeRadius * 0.3, smokeRadius * 0.5, 0, Math.PI * 2);
          ctx.arc(smokeRadius * 0.4, -smokeRadius * 0.2, smokeRadius * 0.55, 0, Math.PI * 2);
          ctx.fillStyle = gradient;
          ctx.fill();

          // Subtle inner swirling details
          ctx.beginPath();
          ctx.arc(0, 0, smokeRadius * 0.5, 0, Math.PI * 2);
          ctx.fillStyle = 'rgba(255, 255, 255, 0.05)';
          ctx.fill();
          
          ctx.restore();
          
          // Timer ring (white stroke showing remaining time) around the smoke
          if (progress > 0) {
            const startAngle = -Math.PI / 2;
            const endAngle = startAngle + (progress * Math.PI * 2);
            
            // Background ring (dark)
            ctx.beginPath();
            ctx.arc(pos.x, pos.y, smokeRadius + 6, 0, Math.PI * 2);
            ctx.strokeStyle = 'rgba(20, 25, 30, 0.7)';
            ctx.lineWidth = 4;
            ctx.stroke();
            
            // Progress ring (white/cyan)
            ctx.beginPath();
            ctx.arc(pos.x, pos.y, smokeRadius + 6, startAngle, endAngle, false); 
            ctx.strokeStyle = 'rgba(200, 220, 255, 0.9)';
            ctx.lineWidth = 3;
            ctx.lineCap = 'round';
            ctx.stroke();
            ctx.lineCap = 'butt';
            
            // Central icon (Smoke Cloud)
            ctx.fillStyle = 'rgba(255, 255, 255, 0.9)';
            ctx.font = '14px Arial';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.shadowColor = 'rgba(0,0,0,0.8)';
            ctx.shadowBlur = 4;
            ctx.fillText('☁', pos.x, pos.y + 1);
            ctx.shadowBlur = 0;
          }
          
        } else if (effect.type === 'inferno') {
          // =====================================================
          // MOLOTOV/INCENDIARY - Realistic fire with particles
          // =====================================================
          const fireRadius = baseRadius * 0.85;
          const timeRemaining = effect.time_remaining || 7;
          const maxDuration = 7;
          const age = maxDuration - timeRemaining;
          
          // Spreading animation (grows from center in first 0.6s)
          const spreadProgress = Math.min(1, Math.max(0, age / 0.6));
          // Easing out cubic for spread
          const spreadEased = 1 - Math.pow(1 - spreadProgress, 3);
          const currentRadius = fireRadius * spreadEased;
          
          const intensity = Math.min(1, timeRemaining / maxDuration);
          
          // Multiple fire layers for depth
          const flicker1 = 0.7 + Math.sin(nowSeconds * 12) * 0.3;
          const flicker2 = 0.8 + Math.sin(nowSeconds * 8 + 1) * 0.2;
          
          // Outer fire glow
          const outerGradient = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, currentRadius * 1.2);
          outerGradient.addColorStop(0, `rgba(255, 120, 30, ${0.4 * intensity * flicker1})`);
          outerGradient.addColorStop(0.6, `rgba(255, 80, 0, ${0.2 * intensity})`);
          outerGradient.addColorStop(1, 'rgba(200, 50, 0, 0)');
          
          ctx.beginPath();
          ctx.arc(pos.x, pos.y, currentRadius * 1.2, 0, Math.PI * 2);
          ctx.fillStyle = outerGradient;
          ctx.fill();
          
          // Main fire core (dynamic shape)
          ctx.save();
          ctx.translate(pos.x, pos.y);
          // Rotate slightly over time
          ctx.rotate(nowSeconds * 0.5);
          
          const coreGradient = ctx.createRadialGradient(0, 0, 0, 0, 0, currentRadius * 0.8);
          coreGradient.addColorStop(0, `rgba(255, 220, 100, ${0.9 * intensity * flicker2})`);
          coreGradient.addColorStop(0.3, `rgba(255, 160, 40, ${0.8 * intensity * flicker1})`);
          coreGradient.addColorStop(0.7, `rgba(255, 80, 0, ${0.5 * intensity})`);
          coreGradient.addColorStop(1, `rgba(200, 40, 0, ${0.1 * intensity})`);
          
          ctx.beginPath();
          // Draw an irregular fire blob instead of perfect circle
          for(let i=0; i<8; i++) {
             const angle = (i / 8) * Math.PI * 2;
             // Adds some spiky randomness based on time and angle
             const spike = 1 + Math.sin(angle * 3 + nowSeconds * 5) * 0.15;
             const r = currentRadius * 0.8 * spike;
             if (i === 0) ctx.moveTo(r * Math.cos(angle), r * Math.sin(angle));
             else ctx.lineTo(r * Math.cos(angle), r * Math.sin(angle));
          }
          ctx.closePath();
          ctx.fillStyle = coreGradient;
          ctx.fill();
          ctx.restore();
          
          // Smooth pulsing glow instead of jittery particles
          const glowPhase = (nowSeconds * 4) % (Math.PI * 2);
          const pulseRadius = currentRadius * (0.6 + Math.sin(glowPhase) * 0.1);
          ctx.beginPath();
          ctx.arc(pos.x, pos.y, pulseRadius, 0, Math.PI * 2);
          ctx.fillStyle = `rgba(255, 180, 50, ${0.4 * intensity})`;
          ctx.fill();
        }
      });
    }
    
    // Draw grenade trajectories from projectiles
    if (frameData?.projectiles) {
      frameData.projectiles.forEach(proj => {
        if (!proj.trajectory || proj.trajectory.length < 4) return;
        
        const grenadeColors = {
          smoke: '#AAAAAA', flashbang: '#FFFF88', he: '#FF6644',
          molotov: '#FF8800', decoy: '#88FF88', incendiary: '#FF8800'
        };
        const color = grenadeColors[proj.type] || '#FFFFFF';
        
        // Convert trajectory to canvas points array
        const pts = [];
        for (let i = 0; i < proj.trajectory.length; i += 2) {
          pts.push(translateCoords(proj.trajectory[i], proj.trajectory[i + 1], mapConfig, size));
        }
        
        // Draw smooth trajectory using quadratic Bézier curves
        ctx.strokeStyle = color;
        ctx.lineWidth = 2;
        ctx.globalAlpha = 0.5;
        ctx.lineCap = 'round';
        ctx.lineJoin = 'round';
        ctx.beginPath();
        ctx.moveTo(pts[0].x, pts[0].y);
        if (pts.length === 2) {
          ctx.lineTo(pts[1].x, pts[1].y);
        } else {
          for (let i = 1; i < pts.length - 1; i++) {
            const cpX = (pts[i].x + pts[i + 1].x) / 2;
            const cpY = (pts[i].y + pts[i + 1].y) / 2;
            ctx.quadraticCurveTo(pts[i].x, pts[i].y, cpX, cpY);
          }
          const last = pts[pts.length - 1];
          ctx.lineTo(last.x, last.y);
        }
        ctx.stroke();
        ctx.globalAlpha = 1;
        ctx.lineCap = 'butt';
        ctx.lineJoin = 'miter';
        
        // Draw current grenade position
        const currentPos = translateCoords(proj.x, proj.y, mapConfig, size);
        ctx.beginPath();
        ctx.arc(currentPos.x, currentPos.y, 5, 0, Math.PI * 2);
        ctx.fillStyle = color;
        ctx.globalAlpha = 0.9;
        ctx.fill();
        ctx.globalAlpha = 1;
      });
    }
    
    // =========================================================================
    // CINEMATIC SHOT ANIMATION SYSTEM - Ultra-realistic visual effects
    // =========================================================================
    const now = performance.now();
    
    // WEAPONS TO EXCLUDE (melee, utility, grenades)
    const EXCLUDED_WEAPONS = new Set([
      'Knife', 'C4', 'Zeus x27',
      'HE Grenade', 'Flashbang', 'Smoke Grenade', 
      'Molotov', 'Incendiary Grenade', 'Decoy Grenade',
      'knife', 'c4', 'taser',
      'hegrenade', 'flashbang', 'smokegrenade',
      'molotov', 'incgrenade', 'decoy'
    ]);
    
    // Weapon classification for visual effects
    const SNIPER_RIFLES = new Set(['AWP', 'SSG 08', 'SCAR-20', 'G3SG1']);
    const RIFLES = new Set(['AK-47', 'M4A4', 'M4A1-S', 'AUG', 'FAMAS', 'Galil AR', 'SG 553']);
    const SMGS = new Set(['MAC-10', 'MP9', 'MP7', 'UMP-45', 'PP-Bizon', 'P90', 'MP5-SD']);
    
    // Add new shot tracers when shots occur
    if (frameData?.shots) {
      frameData.shots.forEach(shot => {
        if (EXCLUDED_WEAPONS.has(shot.weapon)) return;
        
        // Deduplication by shot properties (same shot appears in multiple frames from backend)
        // Use rounded coordinates to create unique key for each actual shot
        const shotKey = `${shot.shooter_id}-${shot.from_x.toFixed(0)}-${shot.from_y.toFixed(0)}-${shot.to_x.toFixed(0)}-${shot.to_y.toFixed(0)}`;
        if (prevEventsRef.current.has(shotKey)) return;
        prevEventsRef.current.add(shotKey);
        
        // Clean old entries periodically (every 100 shots) to prevent memory leak
        if (prevEventsRef.current.size > 500) {
          prevEventsRef.current.clear();
        }
        
        const shooter = frameData.players?.find(p => p.steam_id === shot.shooter_id);
        const weapon = shot.weapon;
        
        // Determine weapon category for visual effects
        const isSniper = SNIPER_RIFLES.has(weapon);
        const isRifle = RIFLES.has(weapon);
        const isSMG = SMGS.has(weapon);
        
        // Calculate shot distance for effect intensity
        const dx = shot.to_x - shot.from_x;
        const dy = shot.to_y - shot.from_y;
        const shotDistance = Math.sqrt(dx * dx + dy * dy);
        
        // Create tracer data structure
        tracersRef.current.push({
          // Position data
          fromX: shot.from_x,
          fromY: shot.from_y,
          toX: shot.to_x,
          toY: shot.to_y,
          
          // Visual properties
          team: shooter?.team || 'T',
          weapon: weapon,
          isSniper: isSniper,
          isRifle: isRifle,
          isSMG: isSMG,
          distance: shotDistance,
          
          // Timing - shorter for automatic weapons to distinguish each shot
          startTime: now,
          tracerDuration: isSniper ? 150 : (isRifle ? 80 : 60), // Sniper 150ms, Rifle 80ms, SMG/Pistol 60ms
        });
      });
    }
    prevFrameRef.current = frameData;
    
    // =========================================================================
    // RENDER SHOT TRACERS AND EFFECTS
    // =========================================================================
    
    tracersRef.current = tracersRef.current.filter(tracer => {
      const age = now - tracer.startTime;
      
      // Remove expired tracers
      if (age > tracer.tracerDuration) {
        return false;
      }
      
      // Convert positions to canvas coordinates
      const fromPos = translateCoords(tracer.fromX, tracer.fromY, mapConfig, size);
      const toPos = translateCoords(tracer.toX, tracer.toY, mapConfig, size);
      
      // Extend the line to cover full map (backend uses 1500 units, we extend 5x more)
      const dx = toPos.x - fromPos.x;
      const dy = toPos.y - fromPos.y;
      const extendedToX = fromPos.x + dx * 5;
      const extendedToY = fromPos.y + dy * 5;
      
      // ---------------------------------------------------------------------------
      // TRACER LINE - Simple clean line (no effects)
      // ---------------------------------------------------------------------------
      // Team colors: Blue for CT, Yellow for T
      const tracerColor = tracer.team === 'CT' 
        ? 'rgba(91, 155, 213, 1)'   // Blue for CT
        : 'rgba(255, 200, 80, 1)';  // Yellow/Gold for T
      
      ctx.beginPath();
      ctx.moveTo(fromPos.x, fromPos.y);
      ctx.lineTo(extendedToX, extendedToY);
      ctx.strokeStyle = tracerColor;
      ctx.lineWidth = 1.5;
      ctx.stroke();
      
      return true; // Keep this tracer
    });
    
    
    // =========================================================================
    // GRENADE EXPLOSION EFFECTS SYSTEM
    // =========================================================================
    
    // Process grenade_explode events and add to explosions ref
    if (currentRoundData?.events && prevFrameRef.current?.tick !== frameData?.tick) {
      const currentTick = frameData?.tick || 0;
      currentRoundData.events.forEach(event => {
        if (event.type === 'grenade_explode' && 
            event.tick <= currentTick && 
            event.tick > (prevFrameRef.current?.tick || 0) &&
            !prevEventsRef.current.has(`${event.tick}-${event.grenade_type}-${event.x}-${event.y}`)) {
          prevEventsRef.current.add(`${event.tick}-${event.grenade_type}-${event.x}-${event.y}`);
          
          const grenadeType = event.grenade_type;
          explosionsRef.current.push({
            x: event.x,
            y: event.y,
            type: grenadeType,
            startTime: now,
            duration: grenadeType === 'flashbang' ? 250 : 
                      grenadeType === 'he' || grenadeType === 'hegrenade' ? 600 : 350
          });
          
          // Trigger screen shake for HE grenades
          if (grenadeType === 'he' || grenadeType === 'hegrenade') {
            screenShakeRef.current = {
              active: true,
              startTime: now,
              intensity: 6,
              duration: 250
            };
          }
        }
      });
    }
    
    // Draw and expire explosions
    explosionsRef.current = explosionsRef.current.filter(exp => now - exp.startTime < exp.duration);
    
    explosionsRef.current.forEach(explosion => {
      const pos = translateCoords(explosion.x, explosion.y, mapConfig, size);
      const age = now - explosion.startTime;
      const progress = age / explosion.duration;
      const explosionRadius = (144 / mapConfig.scale) * (size / 1024);
      
      if (explosion.type === 'flashbang' || explosion.type === 'flash') {
        // Flashbang: Small intense white burst that leaves a quick star glare
        const flashIntensity = progress < 0.1 ? 1 : Math.pow(1 - (progress - 0.1) / 0.9, 2);
        // Expand very quickly, then lock radius
        const expandScale = progress < 0.1 ? (progress / 0.1) : 1 + (progress * 0.2);
        const radius = explosionRadius * 0.8 * expandScale; // A bit smaller baseline
        
        // Outer glow
        const flashGradient = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, radius);
        flashGradient.addColorStop(0, `rgba(255, 255, 255, ${flashIntensity * 0.95})`);
        flashGradient.addColorStop(0.2, `rgba(255, 255, 240, ${flashIntensity * 0.8})`);
        flashGradient.addColorStop(0.5, `rgba(255, 255, 200, ${flashIntensity * 0.4})`);
        flashGradient.addColorStop(1, 'rgba(255, 255, 255, 0)');
        
        ctx.beginPath();
        ctx.arc(pos.x, pos.y, radius, 0, Math.PI * 2);
        ctx.fillStyle = flashGradient;
        ctx.fill();
        
        // Characteristic 4-point star glare for flash
        if (progress < 0.4) {
          const starIntensity = 1 - (progress / 0.4);
          ctx.save();
          ctx.translate(pos.x, pos.y);
          ctx.rotate(Math.PI / 4 + progress * 2);
          
          ctx.beginPath();
          ctx.ellipse(0, 0, radius * 1.5, radius * 0.15, 0, 0, Math.PI * 2);
          ctx.ellipse(0, 0, radius * 0.15, radius * 1.5, 0, 0, Math.PI * 2);
          ctx.fillStyle = `rgba(255, 255, 255, ${starIntensity})`;
          ctx.fill();
          ctx.restore();
        }
        
        // Inner bright core
        if (progress < 0.2) {
          ctx.beginPath();
          ctx.arc(pos.x, pos.y, radius * 0.4, 0, Math.PI * 2);
          ctx.fillStyle = `rgba(255, 255, 255, ${(1 - progress / 0.2)})`;
          ctx.fill();
        }
        
      } else if (explosion.type === 'he' || explosion.type === 'hegrenade') {
        // HE Grenade: Small, punchy blast
        // Reduce the overall radius
        const maxRadius = explosionRadius * 0.6;
        
        // Quick expanding fireball that dissipates
        if (progress < 0.6) {
          const coreProgress = progress / 0.6;
          // Ease out cubic
          const scale = 1 - Math.pow(1 - coreProgress, 3);
          const coreAlpha = 1 - coreProgress;
          const currentRadius = maxRadius * scale;
          
          const fireGradient = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, currentRadius);
          fireGradient.addColorStop(0, `rgba(255, 255, 180, ${coreAlpha})`);
          fireGradient.addColorStop(0.3, `rgba(255, 120, 30, ${coreAlpha * 0.9})`);
          fireGradient.addColorStop(0.7, `rgba(200, 40, 0, ${coreAlpha * 0.6})`);
          fireGradient.addColorStop(1, `rgba(50, 50, 50, 0)`);
          
          ctx.beginPath();
          ctx.arc(pos.x, pos.y, currentRadius, 0, Math.PI * 2);
          ctx.fillStyle = fireGradient;
          ctx.fill();
          
          // Small debris / sparks
          for (let i = 0; i < 6; i++) {
             const angle = (i / 6) * Math.PI * 2 + (progress * 2);
             const sparkDist = currentRadius * 1.2 * scale;
             ctx.beginPath();
             ctx.arc(pos.x + Math.cos(angle) * sparkDist, pos.y + Math.sin(angle) * sparkDist, 1.5, 0, Math.PI * 2);
             ctx.fillStyle = `rgba(255, 200, 100, ${coreAlpha})`;
             ctx.fill();
          }
        }
        
        // Single sharp shockwave
        if (progress < 0.8) {
           const shockAlpha = 1 - (progress / 0.8);
           const shockScale = Math.pow(progress / 0.8, 0.5); // Fast out easing
           const shockRadius = maxRadius * 1.5 * shockScale;
           
           ctx.beginPath();
           ctx.arc(pos.x, pos.y, shockRadius, 0, Math.PI * 2);
           ctx.strokeStyle = `rgba(180, 180, 180, ${shockAlpha * 0.5})`;
           ctx.lineWidth = 2;
           ctx.stroke();
        }
        
      } else if (explosion.type === 'decoy') {
        // Decoy: small puff
        const puffAlpha = 1 - progress;
        const puffRadius = 15 + progress * 20;
        
        ctx.beginPath();
        ctx.arc(pos.x, pos.y, puffRadius, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(150, 150, 150, ${puffAlpha * 0.5})`;
        ctx.fill();
      }
    });
    
    // =========================================================================
    // KILL LINES - Show shooter to victim lines on kills
    // =========================================================================
    
    // Add new kill lines from events
    if (currentRoundData?.events && prevFrameRef.current?.tick !== frameData?.tick) {
      const currentTick = frameData?.tick || 0;
      currentRoundData.events.forEach(event => {
        if (event.type === 'kill' && 
            event.tick <= currentTick && 
            event.tick > (prevFrameRef.current?.tick || 0)) {
          const eventKey = `kill-${event.tick}-${event.killer_id}-${event.victim_id}`;
          if (!prevEventsRef.current.has(eventKey)) {
            prevEventsRef.current.add(eventKey);
            killLinesRef.current.push({
              killerX: event.killer_x,
              killerY: event.killer_y,
              victimX: event.victim_x,
              victimY: event.victim_y,
              killerTeam: event.killer_team,
              headshot: event.headshot,
              startTime: now,
              duration: 800
            });
          }
        }
      });
    }
    
    // Draw and expire kill lines
    killLinesRef.current = killLinesRef.current.filter(k => now - k.startTime < k.duration);
    
    killLinesRef.current.forEach(killLine => {
      const age = now - killLine.startTime;
      const progress = age / killLine.duration;
      const alpha = progress < 0.3 ? 1 : (1 - (progress - 0.3) / 0.7);
      
      if (alpha <= 0) return;
      
      const killerPos = translateCoords(killLine.killerX, killLine.killerY, mapConfig, size);
      const victimPos = translateCoords(killLine.victimX, killLine.victimY, mapConfig, size);
      
      // Kill line color based on team and headshot
      const lineColor = killLine.headshot ? '#FF4444' : 
                        (killLine.killerTeam === 'CT' ? '#5B9BD5' : '#E6B422');
      
      // Draw dashed kill line
      ctx.setLineDash([8, 4]);
      ctx.beginPath();
      ctx.moveTo(killerPos.x, killerPos.y);
      ctx.lineTo(victimPos.x, victimPos.y);
      ctx.strokeStyle = lineColor;
      ctx.lineWidth = killLine.headshot ? 3 : 2;
      ctx.globalAlpha = alpha * 0.8;
      ctx.stroke();
      ctx.setLineDash([]);
      
      // Cross icon at victim position
      ctx.beginPath();
      ctx.moveTo(victimPos.x - 5, victimPos.y - 5);
      ctx.lineTo(victimPos.x + 5, victimPos.y + 5);
      ctx.moveTo(victimPos.x + 5, victimPos.y - 5);
      ctx.lineTo(victimPos.x - 5, victimPos.y + 5);
      ctx.strokeStyle = '#FF4444';
      ctx.lineWidth = 3;
      ctx.globalAlpha = alpha;
      ctx.stroke();
      
      // Headshot indicator
      if (killLine.headshot) {
        ctx.font = 'bold 10px Arial';
        ctx.fillStyle = '#FF4444';
        ctx.fillText('HS', victimPos.x, victimPos.y - 14);
      }
      
      ctx.globalAlpha = 1;
    });
    
    // =========================================================================
    // BOMB STATE VISUALIZATION
    // =========================================================================
    
    if (frameData?.bomb) {
      const bomb = frameData.bomb;
      const bombPos = translateCoords(bomb.x, bomb.y, mapConfig, size);
      
      if (bomb.state === 'planted' || bomb.state === 'defusing') {
        // Planted bomb - pulsing red glow
        const pulsePhase = (now / 200) % (Math.PI * 2);
        const pulseIntensity = 0.5 + Math.sin(pulsePhase) * 0.3;
        const bombRadius = 18;
        
        // Outer danger zone
        const dangerGradient = ctx.createRadialGradient(bombPos.x, bombPos.y, 0, bombPos.x, bombPos.y, bombRadius * 3);
        dangerGradient.addColorStop(0, `rgba(255, 50, 50, ${pulseIntensity * 0.4})`);
        dangerGradient.addColorStop(0.5, `rgba(255, 0, 0, ${pulseIntensity * 0.2})`);
        dangerGradient.addColorStop(1, 'rgba(200, 0, 0, 0)');
        
        ctx.beginPath();
        ctx.arc(bombPos.x, bombPos.y, bombRadius * 3, 0, Math.PI * 2);
        ctx.fillStyle = dangerGradient;
        ctx.fill();
        
        // Bomb Danger Triangle
        ctx.beginPath();
        ctx.moveTo(bombPos.x, bombPos.y - 12);
        ctx.lineTo(bombPos.x + 10, bombPos.y + 6);
        ctx.lineTo(bombPos.x - 10, bombPos.y + 6);
        ctx.closePath();
        ctx.fillStyle = '#EF4444';
        ctx.fill();
        ctx.font = 'bold 12px Arial';
        ctx.fillStyle = '#FFF';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText('!', bombPos.x, bombPos.y + 3);
        
        // Site indicator
        if (bomb.site) {
          ctx.font = 'bold 12px Arial';
          ctx.fillStyle = '#FFFFFF';
          ctx.fillText(bomb.site, bombPos.x, bombPos.y - 20);
        }
        
        // Defusing indicator
        if (bomb.state === 'defusing') {
          const defusePhase = (now / 100) % (Math.PI * 2);
          ctx.beginPath();
          ctx.arc(bombPos.x, bombPos.y, 25, 0, Math.PI * 2);
          ctx.strokeStyle = `rgba(88, 166, 255, ${0.7 + Math.sin(defusePhase) * 0.3})`;
          ctx.lineWidth = 3;
          ctx.stroke();
          
          ctx.font = 'bold 10px Arial';
          ctx.fillStyle = '#58A6FF';
          ctx.fillText('DEFUSING', bombPos.x, bombPos.y + 30);
        }
        
      } else if (bomb.state === 'dropped') {
        // Dropped bomb - subtle indicator
        ctx.beginPath();
        ctx.arc(bombPos.x, bombPos.y, 6, 0, Math.PI * 2);
        ctx.fillStyle = '#F59E0B';
        ctx.fill();
        ctx.strokeStyle = '#FFF';
        ctx.lineWidth = 1.5;
        ctx.stroke();
        
      } else if (bomb.state === 'exploded') {
        // Explosion effect
        const explodeRadius = 60;
        const explodeGradient = ctx.createRadialGradient(bombPos.x, bombPos.y, 0, bombPos.x, bombPos.y, explodeRadius);
        explodeGradient.addColorStop(0, 'rgba(255, 200, 100, 0.9)');
        explodeGradient.addColorStop(0.5, 'rgba(255, 100, 0, 0.6)');
        explodeGradient.addColorStop(1, 'rgba(200, 50, 0, 0)');
        
        ctx.beginPath();
        ctx.arc(bombPos.x, bombPos.y, explodeRadius, 0, Math.PI * 2);
        ctx.fillStyle = explodeGradient;
        ctx.fill();
      }
    }
    
    // Draw players
    if (frameData?.players) {
      frameData.players.forEach(player => {
        const pos = translateCoords(player.x, player.y, mapConfig, size);
        const theme = player.team === 'CT' ? THEME.ct : THEME.t;
        
        if (!player.alive) {
          ctx.strokeStyle = 'rgba(100, 100, 100, 0.5)';
          ctx.lineWidth = 2;
          ctx.beginPath();
          ctx.moveTo(pos.x - 5, pos.y - 5); ctx.lineTo(pos.x + 5, pos.y + 5);
          ctx.moveTo(pos.x + 5, pos.y - 5); ctx.lineTo(pos.x - 5, pos.y + 5);
          ctx.stroke();
          return;
        }
        
        const radius = 9;
        
        // If player is blinded, draw blindness indicator
        let flashVal = 0;
        if (typeof player.flash_duration === 'number' && player.flash_duration > 0) {
            flashVal = Math.min(1, player.flash_duration / 5.0); // max duration ~ 5s
        } else if (player.is_blinded) {
            flashVal = 0.7; // fallback
        }
        
        if (flashVal > 0) {
           const blindRadius = radius + 4 + (flashVal * 4);
           
           // Blinding ring around player
           ctx.beginPath();
           ctx.arc(pos.x, pos.y, blindRadius, 0, Math.PI * 2);
           ctx.fillStyle = `rgba(255, 255, 255, ${flashVal * 0.2})`;
           ctx.strokeStyle = `rgba(255, 255, 200, ${flashVal * 0.9})`;
           ctx.lineWidth = 1.5;
           ctx.fill();
           ctx.stroke();
           
           // Blind icon
           ctx.fillStyle = `rgba(255, 255, 255, ${flashVal})`;
           ctx.font = '10px Arial';
           ctx.textAlign = 'center';
           ctx.fillText('👁', pos.x, pos.y - blindRadius - 4);
           
           // Cross line over eye to indicate blind
           ctx.beginPath();
           ctx.moveTo(pos.x - 4, pos.y - blindRadius - 8);
           ctx.lineTo(pos.x + 4, pos.y - blindRadius - 1);
           ctx.strokeStyle = `rgba(255, 50, 50, ${flashVal})`;
           ctx.lineWidth = 1.5;
           ctx.stroke();
        }

        // Glow
        ctx.shadowColor = theme.glow;
        ctx.shadowBlur = 15;
        
        // Player circle
        ctx.beginPath();
        ctx.arc(pos.x, pos.y, radius, 0, Math.PI * 2);
        ctx.fillStyle = theme.primary;
        ctx.fill();
        ctx.strokeStyle = 'rgba(255,255,255,0.8)';
        ctx.lineWidth = 2;
        ctx.stroke();
        ctx.shadowBlur = 0;
        
        // Direction triangle
        const yawRad = (-player.yaw * Math.PI) / 180;
        const tipX = pos.x + Math.cos(yawRad) * (radius + 7);
        const tipY = pos.y + Math.sin(yawRad) * (radius + 7);
        const perpAngle = yawRad + Math.PI / 2;
        const baseX = pos.x + Math.cos(yawRad) * (radius - 2);
        const baseY = pos.y + Math.sin(yawRad) * (radius - 2);
        
        ctx.beginPath();
        ctx.moveTo(tipX, tipY);
        ctx.lineTo(baseX + Math.cos(perpAngle) * 5, baseY + Math.sin(perpAngle) * 5);
        ctx.lineTo(baseX - Math.cos(perpAngle) * 5, baseY - Math.sin(perpAngle) * 5);
        ctx.closePath();
        ctx.fillStyle = '#FFF';
        ctx.fill();
        
        // Draw weapon icon to the right of player
        const weaponIcon = loadWeaponIcon(player.weapon);
        if (weaponIcon && weaponIcon.complete && weaponIcon.naturalWidth > 0) {
          const iconSize = 18;
          const iconX = pos.x + radius + 6;
          const iconY = pos.y - iconSize / 2;
          ctx.globalAlpha = 0.9;
          ctx.drawImage(weaponIcon, iconX, iconY, iconSize * 1.5, iconSize);
          ctx.globalAlpha = 1;
        }
      });
    }

    // =========================================================================
    // AI ANNOTATIONS (Global State Sync)
    // =========================================================================
    if (annotations && annotations.length > 0) {
      annotations.forEach(anim => {
        if (anim.type === 'DANGER_ZONE') {
          const pos = translateCoords(anim.x, anim.y, mapConfig, size);
          const scaledRadius = (anim.radius / mapConfig.scale) * (size / 1024);
          
          ctx.beginPath();
          ctx.arc(pos.x, pos.y, scaledRadius, 0, Math.PI * 2);
          ctx.fillStyle = anim.color || 'rgba(255, 0, 0, 0.4)';
          // Pulsing effect
          ctx.globalAlpha = 0.5 + Math.sin(now / 300) * 0.3;
          ctx.fill();
          
          ctx.strokeStyle = anim.color ? anim.color.replace(/[\d.]+\)$/g, '1)') : 'red';
          ctx.lineWidth = 2;
          ctx.stroke();
          ctx.globalAlpha = 1;
        } 
        else if (anim.type === 'SUGGESTED_PATH' && anim.points && anim.points.length > 0) {
          ctx.beginPath();
          anim.points.forEach((pt, i) => {
            const pos = translateCoords(pt.x, pt.y, mapConfig, size);
            if (i === 0) ctx.moveTo(pos.x, pos.y);
            else ctx.lineTo(pos.x, pos.y);
          });
          
          ctx.strokeStyle = anim.color || 'rgba(0, 255, 0, 0.8)';
          ctx.lineWidth = 4;
          ctx.lineCap = 'round';
          ctx.lineJoin = 'round';
          
          // Flowing line dash animation
          ctx.setLineDash([10, 10]);
          ctx.lineDashOffset = -(now / 20) % 20;
          
          ctx.stroke();
          ctx.setLineDash([]); // Reset
        }
      });
    }

    ctx.restore();

  }, [canvasSize, dpr, currentRoundData, totalFrames, mapConfig, zoom, pan, annotations]);

  // Animation loop
  useEffect(() => {
    let lastTime = performance.now();
    
    const animate = (now) => {
      const deltaTime = now - lastTime;
      lastTime = now;
      
      let localTime = currentTimeRef.current;
      
      // Update global store with current tick/percentage to allow AI auto-pause
      // We send the normalized percentage (0.0 to 1.0) back to the store so the 
      // chat can stop playback if it exceeds endTick
      updateCurrentTick(localTime);

      // Check if AI is commanding playback
      if (isAiPlaying || (isPlaying && totalFrames > 1)) {
        // ... (We maintain standard logic, but allow AI to override play state)
        const frameDuration = sampleRateMs / playbackSpeed;
        const totalDuration = totalFrames * frameDuration;
        localTime += deltaTime / totalDuration;
        localTime = Math.min(localTime, 1);
        currentTimeRef.current = localTime;
        
        if (now % 100 < deltaTime) setCurrentTime(localTime);
        if (localTime >= 1) { setIsPlaying(false); setCurrentTime(1); }
      }
      
      draw(localTime);
      animationRef.current = requestAnimationFrame(animate);
    };
    
    animationRef.current = requestAnimationFrame(animate);
    return () => { if (animationRef.current) cancelAnimationFrame(animationRef.current); };
  }, [isPlaying, isAiPlaying, playbackSpeed, totalFrames, sampleRateMs, draw, currentRoundData, updateCurrentTick]);

  // Sync ref with state
  useEffect(() => { currentTimeRef.current = currentTime; }, [currentTime]);

  // ============================================================================
  // EVENT HANDLERS
  // ============================================================================
  
  const handleRoundChange = (delta) => {
    const maxRounds = roundsSummary.length || 0;
    const newRound = Math.max(1, Math.min(maxRounds, currentRound + delta));
    if (newRound !== currentRound) {
      setCurrentRound(newRound);
      setCurrentTime(0);
      currentTimeRef.current = 0;
      setIsPlaying(false);
      // Clear all visual effect refs when changing rounds
      tracersRef.current = [];
      explosionsRef.current = [];
      killLinesRef.current = [];
      prevEventsRef.current = new Set();
      prevFrameRef.current = null;
    }
  };

  const timelineRef = useRef(null);
  
  const handleTimelineMouseDown = (e) => {
    e.preventDefault();
    setIsDraggingTimeline(true);
    updateTimeFromMouse(e);
  };
  
  const updateTimeFromMouse = (e) => {
    if (!timelineRef.current) return;
    const rect = timelineRef.current.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const newTime = Math.max(0, Math.min(1, x / rect.width));
    setCurrentTime(newTime);
    currentTimeRef.current = newTime;
  };
  
  useEffect(() => {
    if (!isDraggingTimeline) return;
    const handleMove = (e) => updateTimeFromMouse(e);
    const handleUp = () => setIsDraggingTimeline(false);
    window.addEventListener('mousemove', handleMove);
    window.addEventListener('mouseup', handleUp);
    return () => {
      window.removeEventListener('mousemove', handleMove);
      window.removeEventListener('mouseup', handleUp);
    };
  }, [isDraggingTimeline]);

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.target.tagName === 'INPUT') return;
      switch (e.code) {
        case 'Space': e.preventDefault(); setIsPlaying(p => !p); break;
        case 'ArrowLeft': setCurrentTime(t => Math.max(0, t - 0.05)); break;
        case 'ArrowRight': setCurrentTime(t => Math.min(1, t + 0.05)); break;
        case 'KeyR': setCurrentTime(0); currentTimeRef.current = 0; break;
        default: break;
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  const toggleFullscreen = () => {
    if (!document.fullscreenElement) {
      containerRef.current?.requestFullscreen();
      setIsFullscreen(true);
    } else {
      document.exitFullscreen();
      setIsFullscreen(false);
    }
  };

  // ============================================================================
  // RENDER
  // ============================================================================
  
  if (loading) {
    return (
      <div className="replay-container replay-loading">
        <motion.div 
          className="replay-loader"
          animate={{ rotate: 360 }}
          transition={{ repeat: Infinity, duration: 1, ease: "linear" }}
        />
        <p>Loading replay...</p>
      </div>
    );
  }

  if (!metadata) {
    return <div className="replay-container replay-empty"><p>No replay data available</p></div>;
  }

  const maxRounds = roundsSummary.length || 0;
  const currentFrameData = getInterpolatedFrame();
  const ctPlayers = (currentFrameData?.players?.filter(p => p.team === 'CT') || [])
    .sort((a, b) => String(a.steam_id).localeCompare(String(b.steam_id)));
  const tPlayers = (currentFrameData?.players?.filter(p => p.team === 'T') || [])
    .sort((a, b) => String(a.steam_id).localeCompare(String(b.steam_id)));
  const currentRoundInfo = roundsSummary[currentRound - 1] || {};

  // Compute timer display
  const timeRemaining = currentFrameData?.time_remaining || 0;
  const timerMinutes = Math.floor(timeRemaining / 60);
  const timerSeconds = Math.floor(timeRemaining % 60);
  const timerDisplay = `${timerMinutes}:${timerSeconds.toString().padStart(2, '0')}`;

  return (
    <div className={`replay-container ${isFullscreen ? 'fullscreen' : ''}`} ref={containerRef}>
      {/* Loading overlay */}
      <AnimatePresence>
        {loadingRound && (
          <motion.div 
            className="replay-round-loading"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          >
            <div className="replay-loader-small" />
            <span>Loading round {currentRound}...</span>
          </motion.div>
        )}
      </AnimatePresence>

      {/* ═══════════════════════════════════════════════════════════
          TOP BAR: Score / Round / Timer
          ═══════════════════════════════════════════════════════════ */}
      <div className="replay-top-bar">
        <div className="top-bar-team ct-side">
          <span className="top-bar-team-label">CT</span>
          <span className="top-bar-alive">{ctPlayers.filter(p => p.alive).length}<span className="alive-label"> alive</span></span>
        </div>
        <div className="top-bar-center">
          <span className={`top-bar-timer ${timeRemaining <= 10 ? 'danger' : ''}`}>{timerDisplay}</span>
          <span className="top-bar-round">Round {currentRound} / {maxRounds}</span>
          {currentRoundInfo.winner && (
            <span className={`top-bar-winner ${currentRoundInfo.winner?.toLowerCase()}`}>
              {currentRoundInfo.winner} win
            </span>
          )}
        </div>
        <div className="top-bar-team t-side">
          <span className="top-bar-alive"><span className="alive-label">alive </span>{tPlayers.filter(p => p.alive).length}</span>
          <span className="top-bar-team-label">T</span>
        </div>
      </div>

      {/* ═══════════════════════════════════════════════════════════
          MAIN AREA: CT Panel | Canvas | T Panel
          ═══════════════════════════════════════════════════════════ */}
      <div className="replay-main-layout">
        
        {/* CT Team Panel */}
        <div className="replay-team-panel ct">
          <div className="team-players">
            {ctPlayers.map((player, i) => (
              <PlayerCard key={player.steam_id || i} player={player} team="CT" />
            ))}
          </div>
        </div>

        {/* Center: Canvas */}
        <div className="replay-center">
          {/* Tactical context (only when provided by AI) */}
          {scenarioContext && (
            <div className="replay-scenario-header">
              <div className="scenario-title">
                <div className="scenario-dot" />
                {scenarioContext.title || "SITUACIÓN TÁCTICA"}
              </div>
              <p className="scenario-desc">{scenarioContext.description || "Analiza la jugada."}</p>
            </div>
          )}

          <div className="replay-canvas-container">
            <canvas
              ref={canvasRef}
              width={canvasSize * dpr}
              height={canvasSize * dpr}
              style={{ width: canvasSize, height: canvasSize }}
              className="replay-canvas"
            />
            
            {/* Kill Feed Overlay — top-right of canvas */}
            <div className="replay-killfeed-overlay">
              <KillFeedOverlay 
                events={currentRoundData?.events}
                currentTick={currentFrameData?.interpolatedTick || currentFrameData?.tick || 0}
              />
            </div>

            {/* Zoom Controls */}
            <div className="replay-zoom-controls">
              <button onClick={() => setZoom(z => Math.min(4, z * 1.25))}><ZoomIn size={14} /></button>
              <button onClick={() => setZoom(z => Math.max(0.5, z / 1.25))}><ZoomOut size={14} /></button>
              <button onClick={() => { setZoom(1); setPan({ x: 0, y: 0 }); }}><RotateCcw size={14} /></button>
            </div>
          </div>
        </div>

        {/* T Team Panel */}
        <div className="replay-team-panel t">
          <div className="team-players">
            {tPlayers.map((player, i) => (
              <PlayerCard key={player.steam_id || i} player={player} team="T" />
            ))}
          </div>
        </div>
      </div>

      {/* ═══════════════════════════════════════════════════════════
          BOTTOM DOCK: Timeline + Playback Controls + Round Selector
          ═══════════════════════════════════════════════════════════ */}
      <div className="replay-bottom-dock">
        {/* Timeline — full width */}
        <div 
          ref={timelineRef}
          className={`replay-timeline ${isDraggingTimeline ? 'dragging' : ''}`}
          onMouseDown={handleTimelineMouseDown}
        >
          <div className="timeline-track" />
          <div className="timeline-progress" style={{ width: `${currentTime * 100}%` }} />
          <div className="timeline-handle" style={{ left: `${currentTime * 100}%` }} />
        </div>

        {/* Playback row */}
        <div className="dock-controls-row">
          {/* Left: Round nav */}
          <div className="dock-section dock-round-nav">
            <button className="dock-btn" onClick={() => handleRoundChange(-1)} disabled={currentRound <= 1}>
              <ChevronLeft size={16} />
            </button>
            <span className="dock-round-label">Ronda {currentRound}</span>
            <button className="dock-btn" onClick={() => handleRoundChange(1)} disabled={currentRound >= maxRounds}>
              <ChevronRight size={16} />
            </button>
          </div>

          {/* Center: Transport */}
          <div className="dock-section dock-transport">
            <button className="dock-btn" onClick={() => { setCurrentTime(0); currentTimeRef.current = 0; }}>
              <SkipBack size={14} />
            </button>
            <button 
              className={`dock-play-btn ${isPlaying ? 'playing' : ''}`}
              onClick={() => setIsPlaying(!isPlaying)}
            >
              {isPlaying ? <Pause size={16} /> : <Play size={16} />}
            </button>
            <button className="dock-btn" onClick={() => { setCurrentTime(1); setIsPlaying(false); }}>
              <SkipForward size={14} />
            </button>
          </div>

          {/* Right: Speed + Fullscreen */}
          <div className="dock-section dock-extras">
            <select 
              className="dock-speed"
              value={playbackSpeed}
              onChange={(e) => setPlaybackSpeed(parseFloat(e.target.value))}
            >
              <option value="0.25">0.25×</option>
              <option value="0.5">0.5×</option>
              <option value="1">1×</option>
              <option value="2">2×</option>
              <option value="4">4×</option>
            </select>
            <button className="dock-btn" onClick={toggleFullscreen}>
              <Maximize2 size={14} />
            </button>
          </div>
        </div>

        {/* Round chips */}
        <div className="dock-round-chips">
          {roundsSummary.map((round, idx) => (
            <button
              key={idx}
              className={`dock-round-chip ${currentRound === idx + 1 ? 'active' : ''} ${round.winner?.toLowerCase() || ''}`}
              onClick={() => { 
                if (currentRound !== idx + 1) {
                  setCurrentRound(idx + 1); 
                  setCurrentTime(0); 
                  currentTimeRef.current = 0; 
                  setIsPlaying(false);
                  tracersRef.current = [];
                  explosionsRef.current = [];
                  killLinesRef.current = [];
                  prevEventsRef.current = new Set();
                  prevFrameRef.current = null;
                }
              }}
            >
              {idx + 1}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
