// frontend/src/components/Stats/Replay2DViewer.jsx
// INNOVATIVE 2D Replay Viewer - Cinematic Esports Experience
// Features: Framer Motion animations, dual team panels, glassmorphism controls

import React, { useRef, useEffect, useState, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Play, Pause, SkipBack, SkipForward, Maximize2, ChevronLeft, ChevronRight, ZoomIn, ZoomOut, RotateCcw } from "lucide-react";
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
  ct: { primary: "#5B9BD5", glow: "rgba(91, 155, 213, 0.5)", dark: "#1a3a5c" },
  t: { primary: "#E6B422", glow: "rgba(230, 180, 34, 0.5)", dark: "#4a3a0a" },
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

const PlayerCard = React.memo(({ player, team, isHighlighted }) => {
  const isDead = !player.alive;
  const healthPercent = player.health || 0;
  const weaponIcon = loadWeaponIcon(player.weapon);
  
  const healthColor = healthPercent > 60 ? '#22C55E' : healthPercent > 30 ? '#EAB308' : '#EF4444';
  const teamColor = team === 'CT' ? THEME.ct : THEME.t;
  
  return (
    <motion.div
      layout
      initial={{ opacity: 0, x: team === 'CT' ? -20 : 20 }}
      animate={{ 
        opacity: isDead ? 0.4 : 1, 
        x: 0,
        scale: isHighlighted ? 1.02 : 1,
      }}
      transition={{ type: "spring", stiffness: 500, damping: 30 }}
      className={`replay-player-card ${team.toLowerCase()} ${isDead ? 'dead' : ''}`}
      style={{
        borderLeft: team === 'CT' ? `3px solid ${teamColor.primary}` : 'none',
        borderRight: team === 'T' ? `3px solid ${teamColor.primary}` : 'none',
      }}
    >
      {/* Player Name Row */}
      <div className="player-card-header">
        <span className="player-name" style={{ color: teamColor.primary }}>
          {player.name?.substring(0, 12) || 'Player'}
        </span>
        <div className="player-indicators">
          {player.armor > 0 && <span className="armor-icon">🛡</span>}
          {player.has_defuse_kit && <span className="defuse-icon">🔧</span>}
        </div>
      </div>
      
      {/* Health Bar */}
      <div className="player-health-container">
        <motion.div 
          className="player-health-bar"
          initial={{ width: 0 }}
          animate={{ width: `${healthPercent}%` }}
          transition={{ type: "spring", stiffness: 300, damping: 25 }}
          style={{ backgroundColor: isDead ? '#555' : healthColor }}
        />
        <span className="player-health-text">{isDead ? '☠' : healthPercent}</span>
      </div>
      
      {/* Bottom Row: Weapon + Money */}
      <div className="player-card-footer">
        <div className="player-weapon">
          {weaponIcon && weaponIcon.complete && (
            <img src={weaponIcon.src} alt="" className="weapon-icon" />
          )}
        </div>
        <motion.span 
          className="player-money"
          key={player.money}
          initial={{ scale: 1.2, color: '#4ade80' }}
          animate={{ scale: 1, color: '#22C55E' }}
          transition={{ duration: 0.3 }}
        >
          ${player.money || 0}
        </motion.span>
      </div>
    </motion.div>
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

export default function Replay2DViewer({ matchId, replayData: preloadedData, initialRound = 1 }) {
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
        if (data.rounds_summary) setRoundsSummary(data.rounds_summary);
        await loadRoundData(initialRound);
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
    const cached = roundCacheRef.current.get(roundNum);
    if (cached?.frames) { setCurrentRoundData(cached); return; }
    if (preloadedData?.rounds?.[roundNum - 1]) { setCurrentRoundData(preloadedData.rounds[roundNum - 1]); return; }
    
    setLoadingRound(true);
    try {
      const res = await fetch(`http://localhost:8000/match/${matchId}/replay/round/${roundNum}`, { credentials: 'include' });
      if (!res.ok) throw new Error('Error loading round');
      const data = await res.json();
      roundCacheRef.current.set(roundNum, data);
      setCurrentRoundData(data);
    } catch (err) { console.error(err); }
    finally { setLoadingRound(false); }
  }, [matchId, preloadedData]);

  useEffect(() => { 
    if (metadata && currentRound > 0) {
      loadRoundData(currentRound);
      // Clear shot deduplication when changing rounds
      prevEventsRef.current.clear();
      tracersRef.current = [];
    }
  }, [currentRound, metadata, loadRoundData]);

  // Load map image
  useEffect(() => {
    if (!mapName) return;
    const img = new Image();
    img.onload = () => { mapImageRef.current = img; };
    img.src = `/maps/${mapName}_radar_psd.png`;
  }, [mapName]);

  // Canvas resize
  useEffect(() => {
    const updateSize = () => {
      if (containerRef.current) {
        const rect = containerRef.current.getBoundingClientRect();
        // Leave space for side panels (280px each)
        const availableWidth = rect.width - 560 - 40;
        const maxHeight = window.innerHeight - 280;
        const size = Math.min(availableWidth, maxHeight, isFullscreen ? 1000 : 750);
        setCanvasSize(Math.max(size, 500));
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
    const size = canvasSize;
    
    ctx.clearRect(0, 0, size, size);
    ctx.save();
    
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
          
          // Main smoke fill - subtle movement
          const wobble = Math.sin(nowSeconds * 0.8) * 2;
          const gradient = ctx.createRadialGradient(pos.x + wobble, pos.y, 0, pos.x, pos.y, smokeRadius);
          gradient.addColorStop(0, 'rgba(140, 145, 155, 0.7)');
          gradient.addColorStop(0.7, 'rgba(120, 125, 135, 0.5)');
          gradient.addColorStop(1, 'rgba(100, 105, 115, 0)');
          
          ctx.beginPath();
          ctx.arc(pos.x, pos.y, smokeRadius, 0, Math.PI * 2);
          ctx.fillStyle = gradient;
          ctx.fill();
          
          // Timer ring (white stroke showing remaining time)
          if (progress > 0) {
            const startAngle = -Math.PI / 2;
            const endAngle = startAngle + (progress * Math.PI * 2);
            
            // Background ring (dark)
            ctx.beginPath();
            ctx.arc(pos.x, pos.y, smokeRadius + 4, 0, Math.PI * 2);
            ctx.strokeStyle = 'rgba(60, 65, 75, 0.6)';
            ctx.lineWidth = 3;
            ctx.stroke();
            
            // Progress ring (white)
            ctx.beginPath();
            ctx.arc(pos.x, pos.y, smokeRadius + 4, startAngle, endAngle);
            ctx.strokeStyle = 'rgba(255, 255, 255, 0.85)';
            ctx.lineWidth = 3;
            ctx.lineCap = 'round';
            ctx.stroke();
            ctx.lineCap = 'butt';
          }
          
        } else if (effect.type === 'inferno') {
          // =====================================================
          // MOLOTOV/INCENDIARY - Realistic fire with particles
          // =====================================================
          const fireRadius = baseRadius * 0.85;
          const timeRemaining = effect.time_remaining || 7;
          const maxDuration = 7;
          const intensity = Math.min(1, timeRemaining / maxDuration);
          
          // Multiple fire layers for depth
          const flicker1 = 0.7 + Math.sin(nowSeconds * 8) * 0.15;
          const flicker2 = 0.8 + Math.sin(nowSeconds * 6 + 1) * 0.1;
          
          // Outer fire glow
          const outerGradient = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, fireRadius * 1.2);
          outerGradient.addColorStop(0, `rgba(255, 120, 30, ${0.3 * intensity * flicker1})`);
          outerGradient.addColorStop(0.6, `rgba(255, 80, 0, ${0.15 * intensity})`);
          outerGradient.addColorStop(1, 'rgba(200, 50, 0, 0)');
          
          ctx.beginPath();
          ctx.arc(pos.x, pos.y, fireRadius * 1.2, 0, Math.PI * 2);
          ctx.fillStyle = outerGradient;
          ctx.fill();
          
          // Main fire core
          const coreGradient = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, fireRadius * 0.8);
          coreGradient.addColorStop(0, `rgba(255, 220, 100, ${0.85 * intensity * flicker2})`);
          coreGradient.addColorStop(0.3, `rgba(255, 160, 40, ${0.7 * intensity * flicker1})`);
          coreGradient.addColorStop(0.7, `rgba(255, 80, 0, ${0.4 * intensity})`);
          coreGradient.addColorStop(1, `rgba(200, 40, 0, ${0.1 * intensity})`);
          
          ctx.beginPath();
          ctx.arc(pos.x, pos.y, fireRadius * 0.8, 0, Math.PI * 2);
          ctx.fillStyle = coreGradient;
          ctx.fill();
          
          // Fire particles (small animated dots)
          const particleCount = 6;
          for (let i = 0; i < particleCount; i++) {
            const angle = (i / particleCount) * Math.PI * 2 + nowSeconds * 2;
            const distance = fireRadius * (0.3 + Math.sin(nowSeconds * 4 + i) * 0.3);
            const px = pos.x + Math.cos(angle) * distance;
            const py = pos.y + Math.sin(angle) * distance;
            const particleSize = 4 + Math.sin(nowSeconds * 5 + i * 2) * 2;
            
            ctx.beginPath();
            ctx.arc(px, py, particleSize, 0, Math.PI * 2);
            ctx.fillStyle = `rgba(255, ${200 - i * 20}, 50, ${0.6 * intensity})`;
            ctx.fill();
          }
          
          // Fire border ring
          ctx.beginPath();
          ctx.arc(pos.x, pos.y, fireRadius, 0, Math.PI * 2);
          ctx.strokeStyle = `rgba(255, 100, 0, ${0.4 * intensity})`;
          ctx.lineWidth = 2;
          ctx.stroke();
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
        
        // Draw trajectory line
        ctx.setLineDash([6, 4]);
        ctx.strokeStyle = color;
        ctx.lineWidth = 2;
        ctx.globalAlpha = 0.6;
        ctx.beginPath();
        
        for (let i = 0; i < proj.trajectory.length; i += 2) {
          const pos = translateCoords(proj.trajectory[i], proj.trajectory[i + 1], mapConfig, size);
          if (i === 0) ctx.moveTo(pos.x, pos.y);
          else ctx.lineTo(pos.x, pos.y);
        }
        ctx.stroke();
        ctx.setLineDash([]);
        ctx.globalAlpha = 1;
        
        // Draw current grenade position
        const currentPos = translateCoords(proj.x, proj.y, mapConfig, size);
        ctx.beginPath();
        ctx.arc(currentPos.x, currentPos.y, 6, 0, Math.PI * 2);
        ctx.fillStyle = color;
        ctx.fill();
        ctx.strokeStyle = '#FFF';
        ctx.lineWidth = 1.5;
        ctx.stroke();
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
        // Flashbang: intense white burst that fades quickly
        const flashIntensity = progress < 0.2 ? 1 : (1 - (progress - 0.2) / 0.8);
        const expandScale = 0.5 + progress * 1.5;
        const radius = explosionRadius * expandScale;
        
        // Outer glow
        const flashGradient = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, radius);
        flashGradient.addColorStop(0, `rgba(255, 255, 255, ${flashIntensity * 0.95})`);
        flashGradient.addColorStop(0.3, `rgba(255, 255, 220, ${flashIntensity * 0.7})`);
        flashGradient.addColorStop(0.7, `rgba(255, 255, 150, ${flashIntensity * 0.3})`);
        flashGradient.addColorStop(1, 'rgba(255, 255, 100, 0)');
        
        ctx.beginPath();
        ctx.arc(pos.x, pos.y, radius, 0, Math.PI * 2);
        ctx.fillStyle = flashGradient;
        ctx.fill();
        
        // Inner bright core
        if (progress < 0.3) {
          ctx.beginPath();
          ctx.arc(pos.x, pos.y, radius * 0.3, 0, Math.PI * 2);
          ctx.fillStyle = `rgba(255, 255, 255, ${(1 - progress / 0.3) * 0.9})`;
          ctx.fill();
        }
        
      } else if (explosion.type === 'he' || explosion.type === 'hegrenade') {
        // HE Grenade: expanding shockwave rings with fire core
        
        // Shockwave rings
        for (let ring = 0; ring < 3; ring++) {
          const ringProgress = Math.max(0, progress - ring * 0.15);
          if (ringProgress <= 0 || ringProgress > 1) continue;
          
          const ringRadius = explosionRadius * (0.3 + ringProgress * 1.5) * (1 + ring * 0.3);
          const ringAlpha = (1 - ringProgress) * 0.6;
          
          ctx.beginPath();
          ctx.arc(pos.x, pos.y, ringRadius, 0, Math.PI * 2);
          ctx.strokeStyle = `rgba(255, ${150 - ring * 40}, ${50 - ring * 20}, ${ringAlpha})`;
          ctx.lineWidth = 4 - ring;
          ctx.stroke();
        }
        
        // Fire core
        if (progress < 0.6) {
          const coreAlpha = (1 - progress / 0.6) * 0.9;
          const coreRadius = explosionRadius * 0.5 * (1 - progress * 0.5);
          const fireGradient = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, coreRadius);
          fireGradient.addColorStop(0, `rgba(255, 255, 200, ${coreAlpha})`);
          fireGradient.addColorStop(0.4, `rgba(255, 150, 50, ${coreAlpha * 0.8})`);
          fireGradient.addColorStop(1, `rgba(200, 50, 0, 0)`);
          
          ctx.beginPath();
          ctx.arc(pos.x, pos.y, coreRadius, 0, Math.PI * 2);
          ctx.fillStyle = fireGradient;
          ctx.fill();
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
      
      // Skull icon at victim position
      ctx.font = 'bold 16px Arial';
      ctx.fillStyle = '#FFFFFF';
      ctx.globalAlpha = alpha;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      ctx.fillText('☠', victimPos.x, victimPos.y);
      
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
        
        // Bomb icon
        ctx.font = 'bold 20px Arial';
        ctx.fillStyle = '#FF3333';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText('💣', bombPos.x, bombPos.y);
        
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
        ctx.font = '16px Arial';
        ctx.fillStyle = '#FFAA00';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText('💣', bombPos.x, bombPos.y);
        
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
        
        const radius = 12;
        
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
        const tipX = pos.x + Math.cos(yawRad) * (radius + 10);
        const tipY = pos.y + Math.sin(yawRad) * (radius + 10);
        const perpAngle = yawRad + Math.PI / 2;
        const baseX = pos.x + Math.cos(yawRad) * (radius - 2);
        const baseY = pos.y + Math.sin(yawRad) * (radius - 2);
        
        ctx.beginPath();
        ctx.moveTo(tipX, tipY);
        ctx.lineTo(baseX + Math.cos(perpAngle) * 6, baseY + Math.sin(perpAngle) * 6);
        ctx.lineTo(baseX - Math.cos(perpAngle) * 6, baseY - Math.sin(perpAngle) * 6);
        ctx.closePath();
        ctx.fillStyle = '#FFF';
        ctx.fill();
        
        // Draw weapon icon to the right of player
        const weaponIcon = loadWeaponIcon(player.weapon);
        if (weaponIcon && weaponIcon.complete && weaponIcon.naturalWidth > 0) {
          const iconSize = 22;
          const iconX = pos.x + radius + 8;
          const iconY = pos.y - iconSize / 2;
          ctx.globalAlpha = 0.9;
          ctx.drawImage(weaponIcon, iconX, iconY, iconSize * 1.5, iconSize);
          ctx.globalAlpha = 1;
        }
      });
    }
    
    ctx.restore();
    
    // Draw HUD overlay (timer)
    if (frameData) {
      const timeRemaining = frameData.time_remaining || 0;
      const minutes = Math.floor(timeRemaining / 60);
      const seconds = Math.floor(timeRemaining % 60);
      
      ctx.font = 'bold 32px "Inter", sans-serif';
      ctx.textAlign = 'center';
      ctx.fillStyle = timeRemaining <= 10 ? '#FF5555' : '#FFFFFF';
      ctx.fillText(`${minutes}:${seconds.toString().padStart(2, '0')}`, size / 2, 40);
      
      ctx.font = '14px "Inter", sans-serif';
      ctx.fillStyle = '#8B949E';
      ctx.fillText(`Round ${currentRound}`, size / 2, 60);
    }
    
  }, [canvasSize, currentRoundData, totalFrames, mapConfig, zoom, pan, currentRound]);

  // Animation loop
  useEffect(() => {
    let lastTime = performance.now();
    
    const animate = (now) => {
      const deltaTime = now - lastTime;
      lastTime = now;
      
      let localTime = currentTimeRef.current;
      
      if (isPlaying && totalFrames > 1) {
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
  }, [isPlaying, playbackSpeed, totalFrames, sampleRateMs, draw]);

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

      {/* Main Layout: CT Panel | Canvas | T Panel */}
      <div className="replay-main-layout">
        
        {/* CT Team Panel */}
        <motion.div 
          className="replay-team-panel ct"
          initial={{ opacity: 0, x: -30 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ delay: 0.1 }}
        >
          <div className="team-panel-header">
            <span className="team-label">COUNTER-TERRORISTS</span>
            <span className="team-alive">{ctPlayers.filter(p => p.alive).length} alive</span>
          </div>
          <div className="team-players">
            {ctPlayers.map((player, i) => (
              <PlayerCard key={player.steam_id || i} player={player} team="CT" />
            ))}
          </div>
        </motion.div>

        {/* Center: Canvas + Controls */}
        <div className="replay-center">
          <div className="replay-canvas-container">
            <canvas
              ref={canvasRef}
              width={canvasSize}
              height={canvasSize}
              className="replay-canvas"
            />
            
            {/* Zoom Controls */}
            <div className="replay-zoom-controls">
              <button onClick={() => setZoom(z => Math.min(4, z * 1.25))}><ZoomIn size={16} /></button>
              <button onClick={() => setZoom(z => Math.max(0.5, z / 1.25))}><ZoomOut size={16} /></button>
              <button onClick={() => { setZoom(1); setPan({ x: 0, y: 0 }); }}><RotateCcw size={16} /></button>
            </div>
          </div>

          {/* Controls Bar */}
          <motion.div 
            className="replay-controls-bar"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
          >
            {/* Round Navigation */}
            <div className="controls-section">
              <button className="control-btn" onClick={() => handleRoundChange(-1)} disabled={currentRound <= 1}>
                <ChevronLeft size={20} />
              </button>
              <div className="round-display">
                <span className="round-number">Round {currentRound}</span>
                <span className="round-total">/ {maxRounds}</span>
                <span className={`round-winner ${currentRoundInfo.winner?.toLowerCase()}`}>
                  {currentRoundInfo.winner}
                </span>
              </div>
              <button className="control-btn" onClick={() => handleRoundChange(1)} disabled={currentRound >= maxRounds}>
                <ChevronRight size={20} />
              </button>
            </div>

            {/* Timeline */}
            <div 
              ref={timelineRef}
              className={`replay-timeline ${isDraggingTimeline ? 'dragging' : ''}`}
              onMouseDown={handleTimelineMouseDown}
            >
              <div className="timeline-track" />
              <motion.div 
                className="timeline-progress"
                style={{ width: `${currentTime * 100}%` }}
              />
              <motion.div 
                className="timeline-handle"
                style={{ left: `${currentTime * 100}%` }}
                whileHover={{ scale: 1.3 }}
                whileTap={{ scale: 1.1 }}
              />
            </div>

            {/* Playback Controls */}
            <div className="controls-section">
              <button className="control-btn" onClick={() => { setCurrentTime(0); currentTimeRef.current = 0; }}>
                <SkipBack size={18} />
              </button>
              
              <motion.button 
                className={`play-btn ${isPlaying ? 'playing' : ''}`}
                onClick={() => setIsPlaying(!isPlaying)}
                whileHover={{ scale: 1.1 }}
                whileTap={{ scale: 0.95 }}
              >
                {isPlaying ? <Pause size={22} /> : <Play size={22} />}
              </motion.button>
              
              <button className="control-btn" onClick={() => { setCurrentTime(1); setIsPlaying(false); }}>
                <SkipForward size={18} />
              </button>
              
              <select 
                className="speed-select"
                value={playbackSpeed}
                onChange={(e) => setPlaybackSpeed(parseFloat(e.target.value))}
              >
                <option value="0.25">0.25x</option>
                <option value="0.5">0.5x</option>
                <option value="1">1x</option>
                <option value="2">2x</option>
                <option value="4">4x</option>
              </select>
              
              <button className="control-btn" onClick={toggleFullscreen}>
                <Maximize2 size={18} />
              </button>
            </div>
          </motion.div>

          {/* Round Chips */}
          <div className="replay-round-chips">
            {roundsSummary.map((round, idx) => (
              <motion.button
                key={idx}
                className={`round-chip ${currentRound === idx + 1 ? 'active' : ''} ${round.winner?.toLowerCase()}`}
                onClick={() => { 
                  if (currentRound !== idx + 1) {
                    setCurrentRound(idx + 1); 
                    setCurrentTime(0); 
                    currentTimeRef.current = 0; 
                    setIsPlaying(false);
                    // Clear visual effects for new round
                    tracersRef.current = [];
                    explosionsRef.current = [];
                    killLinesRef.current = [];
                    prevEventsRef.current = new Set();
                    prevFrameRef.current = null;
                  }
                }}
                whileHover={{ scale: 1.1 }}
                whileTap={{ scale: 0.95 }}
              >
                {idx + 1}
              </motion.button>
            ))}
          </div>
        </div>

        {/* T Team Panel + Kill Feed */}
        <motion.div 
          className="replay-team-panel t"
          initial={{ opacity: 0, x: 30 }}
          animate={{ opacity: 1, x: 0 }}
          transition={{ delay: 0.1 }}
        >
          {/* Kill Feed - Moved here from canvas */}
          <KillFeedOverlay 
            events={currentRoundData?.events}
            currentTick={currentFrameData?.interpolatedTick || currentFrameData?.tick || 0}
          />
          
          <div className="team-panel-header">
            <span className="team-label">TERRORISTS</span>
            <span className="team-alive">{tPlayers.filter(p => p.alive).length} alive</span>
          </div>
          <div className="team-players">
            {tPlayers.map((player, i) => (
              <PlayerCard key={player.steam_id || i} player={player} team="T" />
            ))}
          </div>
        </motion.div>
      </div>
    </div>
  );
}
