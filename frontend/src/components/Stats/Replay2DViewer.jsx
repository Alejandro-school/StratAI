// frontend/src/components/Stats/Replay2DViewer.jsx
// INNOVATIVE 2D Replay Viewer - Cinematic Esports Experience
// Features: Framer Motion animations, dual team panels, glassmorphism controls

import React, { useRef, useEffect, useState, useCallback, useMemo } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Play, Pause, SkipBack, SkipForward, Maximize2, ChevronLeft, ChevronRight, ZoomIn, ZoomOut, RotateCcw, Shield, Wrench, Skull, Bomb, MonitorPlay, Users } from "lucide-react";
import useReplaySyncStore from "../../stores/useReplaySyncStore";
import { API_URL } from "../../utils/api";
import "../../styles/Stats/replay2DViewer.css";

// ============================================================================
// CONFIGURATION
// ============================================================================

const MAP_CONFIGS = {
  de_dust2:   { pos_x: -2476, pos_y: 3239, scale: 4.4  },
  de_mirage:  { pos_x: -3230, pos_y: 1713, scale: 5.0  },
  de_inferno: { pos_x: -2087, pos_y: 3870, scale: 4.9  },
  de_ancient: { pos_x: -2953, pos_y: 2164, scale: 5.0  },
  de_anubis:  { pos_x: -2796, pos_y: 3328, scale: 5.22 },
  de_nuke:    { pos_x: -3453, pos_y: 2887, scale: 7.0  },
  de_overpass:{ pos_x: -4831, pos_y: 1781, scale: 5.2  },
  de_vertigo: { pos_x: -3168, pos_y: 1762, scale: 4.0  },
  de_train:   { pos_x: -2477, pos_y: 2392, scale: 4.7  },
};

const THEME = {
  ct: { primary: "#60a5fa", glow: "rgba(96, 165, 250, 0.45)", dark: "#083344" },
  t: { primary: "#eab308", glow: "rgba(234, 179, 8, 0.45)", dark: "#422006" },
};

const WEAPON_ICON_MAP = {
  // Rifles
  'AK-47': 'weapon_ak47', 'M4A4': 'weapon_m4a1', 'M4A1-S': 'weapon_m4a1_silencer',
  'AWP': 'weapon_awp', 'AUG': 'weapon_aug', 'FAMAS': 'weapon_famas', 
  'Galil AR': 'weapon_galilar', 'SSG 08': 'weapon_ssg08', 'SG 553': 'weapon_sg556',
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
  'Zeus x27': 'weapon_taser', 'Zeus': 'weapon_taser',
  'HE Grenade': 'weapon_hegrenade', 'Flashbang': 'weapon_flashbang',
  'Smoke Grenade': 'weapon_smokegrenade', 'Molotov': 'weapon_molotov',
  'Incendiary Grenade': 'weapon_incgrenade', 'Decoy Grenade': 'weapon_decoy',
};

const PRIMARY_WEAPONS = new Set([
  'AK-47', 'M4A4', 'M4A1-S', 'AWP', 'AUG', 'FAMAS', 'Galil AR', 'SSG 08', 'SG 553',
  'SCAR-20', 'G3SG1', 'MAC-10', 'MP9', 'MP7', 'UMP-45', 'PP-Bizon', 'P90', 'MP5-SD',
  'Nova', 'XM1014', 'Sawed-Off', 'MAG-7', 'M249', 'Negev',
]);

const PISTOLS = new Set([
  'Desert Eagle', 'USP-S', 'Glock-18', 'P2000', 'P250', 'Five-SeveN', 'Tec-9',
  'CZ75-Auto', 'Dual Berettas', 'R8 Revolver',
]);

const UTILITY_WEAPONS = new Set([
  'HE Grenade', 'Flashbang', 'Smoke Grenade', 'Molotov', 'Incendiary Grenade', 'Decoy Grenade',
  'C4', 'Zeus x27', 'Zeus',
]);

// ============================================================================
// UTILITIES
// ============================================================================

function translateCoords(gameX, gameY, mapConfig, canvasSize = 1024) {
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

function buildInventory(player) {
  const sourceWeapons = Array.isArray(player.weapons) && player.weapons.length > 0
    ? player.weapons
    : [player.weapon].filter(Boolean);
  const weapons = Array.from(new Set(sourceWeapons.filter(Boolean)));
  const activeWeapon = player.weapon;

  return {
    primary: weapons.find(weapon => PRIMARY_WEAPONS.has(weapon)) || null,
    pistol: weapons.find(weapon => PISTOLS.has(weapon)) || null,
    utility: weapons.filter(weapon => UTILITY_WEAPONS.has(weapon) && weapon !== 'C4'),
    activeWeapon,
  };
}

function parseTimestampToSeconds(timestamp) {
  if (!timestamp || typeof timestamp !== 'string') return null;
  const parts = timestamp.trim().split(':').map(Number);
  if (parts.some(Number.isNaN)) return null;
  if (parts.length === 1) return parts[0];
  if (parts.length === 2) return parts[0] * 60 + parts[1];
  return null;
}

function formatRoundClock(seconds) {
  if (typeof seconds !== 'number' || Number.isNaN(seconds)) return '';
  const safeSeconds = Math.max(0, seconds);
  const minutes = Math.floor(safeSeconds / 60);
  const remainingSeconds = Math.floor(safeSeconds % 60);
  return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
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

// ============================================================================
// ANIMATED PLAYER CARD COMPONENT
// ============================================================================

const PlayerCard = React.memo(({ player, team }) => {
  const isDead = !player.alive;
  const hp = player.health || 0;
  const inventory = buildInventory(player);
  const tc = team === 'CT' ? THEME.ct : THEME.t;
  const hpColor = hp > 60 ? '#22c55e' : hp > 25 ? '#eab308' : '#ef4444';
  const armor = Math.max(0, Math.min(100, player.armor || 0));
  const playerName = String(player.name || 'Jugador').trim();
  const activeStates = [
    player.is_defusing ? 'DEFUSING' : null,
    player.is_reloading ? 'RELOAD' : null,
    player.is_scoped ? 'SCOPED' : null,
    player.is_walking ? 'SHIFT' : null,
    player.flash_duration > 0 ? 'FLASHED' : null,
  ].filter(Boolean);

  return (
    <div
      className={`rv-player ${team.toLowerCase()} ${isDead ? 'dead' : ''}`}
      style={{ '--tc': tc.primary, '--hp-color': hpColor, '--armor-pct': `${armor}%` }}
      aria-label={`${playerName}, ${hp} de vida, $${player.money || 0}, ${armor} de kevlar`}
    >
      <div className="rv-hp-badge" style={{ borderColor: isDead ? '#475569' : hpColor }}>
        {isDead ? <Skull size={14} color="#64748b" /> : (
          <span style={{ color: hpColor }}>{hp}</span>
        )}
      </div>

      <div className="rv-info">
        <div className="rv-name-row">
          <span className="rv-name" title={playerName}>{playerName}</span>
          {activeStates.length > 0 && (
            <span className="rv-state-tag">{activeStates[0]}</span>
          )}
        </div>
        <div className="rv-meta">
          <span className="rv-money">${(player.money || 0).toLocaleString()}</span>
          <span className={`rv-armor-meter ${armor > 0 ? 'active' : ''}`} title={`Kevlar ${armor}`}>
            <Shield size={11} aria-hidden="true" />
            <span>{armor}</span>
          </span>
          {player.has_defuse_kit && (
            <span className="rv-kit-badge" title="Defuse kit">
              <Wrench size={10} aria-hidden="true" />
              <span>KIT</span>
            </span>
          )}
          {player.has_c4 && (
            <span className="rv-c4-badge" title="C4">
              <Bomb size={11} aria-hidden="true" />
              <span>C4</span>
            </span>
          )}
        </div>
      </div>

      <div className="rv-inventory">
        <WeaponIcon weapon={inventory.primary} active={inventory.activeWeapon === inventory.primary} type="primary" />
        <div className="rv-inventory-secondary">
          <WeaponIcon weapon={inventory.pistol} active={inventory.activeWeapon === inventory.pistol} type="pistol" />
          <div className="rv-utility-row">
              {inventory.utility.slice(0, 5).map((weapon, index) => (
              <WeaponIcon key={`${weapon}-${index}`} weapon={weapon} active={inventory.activeWeapon === weapon} type="utility" />
              ))}
            {player.has_c4 && <WeaponIcon weapon="C4" active={inventory.activeWeapon === 'C4'} type="utility c4" />}
          </div>
        </div>
      </div>
    </div>
  );
});

const WeaponIcon = ({ weapon, active = false, type = '' }) => {
  const iconPath = getWeaponIconPath(weapon);

  if (!weapon) {
    return <span className={`rv-weapon-placeholder ${type}`} />;
  }

  if (!iconPath) {
    return <span className={`rv-weapon-text ${active ? 'active' : ''} ${type}`}>{weapon}</span>;
  }

  return (
    <img
      src={iconPath}
      alt=""
      className={`rv-weapon-img ${active ? 'active' : ''} ${type}`}
      title={weapon}
    />
  );
};

// ============================================================================
// KILL FEED OVERLAY COMPONENT
// ============================================================================

const getKillFeedKey = (kill) => [
  kill.tick,
  kill.killer_id || kill.killer_name,
  kill.assister_id || kill.assister_name || kill.assister,
  kill.victim_id || kill.victim_name,
  kill.weapon || 'weapon'
].join('-');

const KillFeedItem = ({ kill }) => {
  const killerColor = kill.killer_team === 'CT' ? THEME.ct.primary : THEME.t.primary;
  const victimColor = kill.victim_team === 'CT' ? THEME.ct.primary : THEME.t.primary;
  const assisterColor = kill.assister_team === 'CT' ? THEME.ct.primary : kill.assister_team === 'T' ? THEME.t.primary : killerColor;
  const weaponIconPath = getWeaponIconPath(kill.weapon);
  
  const rawKiller = kill.killer_name || kill.attacker_name || '';
  const rawVictim = kill.victim_name || kill.target_name || '';
  const rawAssister = kill.assister_name || kill.assister || '';
  
  const killerName = String(rawKiller).trim().length > 0 
    ? String(rawKiller).trim()
    : 'Jugador';
    
  const victimName = String(rawVictim).trim().length > 0 
    ? String(rawVictim).trim()
    : 'Jugador';
  const assisterName = String(rawAssister).trim();
  const isHeadshot = kill.headshot || kill.is_headshot;
  const isWallbang = kill.wallbang || kill.is_wallbang || kill.penetrated_objects > 0;
  const isNoScope = kill.noscope || kill.no_scope;
  const modifiers = [
    isHeadshot ? 'HS' : null,
    kill.through_smoke ? 'SMOKE' : null,
    isWallbang ? 'WALL' : null,
    isNoScope ? 'NS' : null,
    kill.attacker_blind ? 'BLIND' : null,
  ].filter(Boolean);

  return (
    <motion.div
      layout="position"
      className="kill-feed-item"
      initial={{ opacity: 0, x: 24 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: 18 }}
      transition={{
        layout: { duration: 0.18, ease: [0.2, 0, 0, 1] },
        opacity: { duration: 0.14 },
        x: { duration: 0.18, ease: [0.2, 0, 0, 1] }
      }}
      style={{ '--killer-color': killerColor, '--assister-color': assisterColor, '--victim-color': victimColor }}
    >
      <span className="kill-time">{kill.display_time}</span>
      <div className="kill-actors">
        <span className="kill-name killer" title={killerName}>{killerName}</span>
        {assisterName && (
          <>
          <span className="kill-assist-plus">+</span>
          <span className="kill-name assister" title={assisterName}>{assisterName}</span>
          </>
        )}
      </div>
      <div className="kill-weapon-container">
        {weaponIconPath ? (
          <img src={weaponIconPath} alt="" className="kill-weapon-icon" />
        ) : (
          <span className="kill-weapon-text">{kill.weapon || '?'}</span>
        )}
      </div>
      {modifiers.length > 0 && (
        <div className="kill-modifiers">
          {modifiers.slice(0, 2).map(modifier => (
            <span key={modifier} className={`kill-modifier ${modifier.toLowerCase()}`}>
              {modifier === 'HS' ? <Skull size={15} strokeWidth={2.6} /> : modifier}
            </span>
          ))}
        </div>
      )}
      <span className="kill-name victim" title={victimName}>{victimName}</span>
    </motion.div>
  );
};

const KillFeedOverlay = ({ events, players, frames, playerNameLookup, currentTick }) => {
  const [visibleKills, setVisibleKills] = useState([]);
  
  useEffect(() => {
    if (!events) return;
    const recentKills = events.filter(e => 
      e.type === 'kill' && 
      currentTick - e.tick >= 0 && 
      currentTick - e.tick < 512
    ).map(kill => {
      const killer = players?.find(p => String(p.steam_id) === String(kill.killer_id));
      const victim = players?.find(p => String(p.steam_id) === String(kill.victim_id));
      const assister = players?.find(p => String(p.steam_id) === String(kill.assister_id));
      const killFrameIndex = getClosestFrameIndexByTick(frames, kill.tick);
      const killFrame = killFrameIndex >= 0 ? frames[killFrameIndex] : null;
      const killerName = String(kill.killer_name || '').trim()
        || String(killer?.name || '').trim()
        || String(playerNameLookup?.get(String(kill.killer_id)) || '').trim();
      const victimName = String(kill.victim_name || '').trim()
        || String(victim?.name || '').trim()
        || String(playerNameLookup?.get(String(kill.victim_id)) || '').trim();
      const assisterName = String(kill.assister_name || kill.assister || '').trim()
        || String(assister?.name || '').trim()
        || String(playerNameLookup?.get(String(kill.assister_id)) || '').trim();

      return {
        ...kill,
        killer_name: killerName,
        victim_name: victimName,
        assister_name: assisterName,
        display_time: formatRoundClock(killFrame?.time_remaining),
      };
    });
    setVisibleKills(recentKills.slice(-5));
  }, [events, players, frames, playerNameLookup, currentTick]);
  
  return (
    <div className="kill-feed-overlay">
      <AnimatePresence initial={false}>
        {visibleKills.map((kill, i) => (
          <KillFeedItem 
            key={getKillFeedKey(kill)}
            kill={kill}
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
  scenarioContext = null,
  fitMode = 'focus',
  compactTeams = false,
  onAvailabilityChange
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
  const pendingAiClipRef = useRef(null);
  
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
  const [canvasDim, setCanvasDim] = useState({ w: 800, h: 800 });
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [zoom, setZoom] = useState(1);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [isDraggingTimeline, setIsDraggingTimeline] = useState(false);
  const [isTeamsPanelOpen, setIsTeamsPanelOpen] = useState(false);

  useEffect(() => {
    onAvailabilityChange?.(Boolean(metadata));
  }, [metadata, onAvailabilityChange]);
  
  // Derived
  const mapName = metadata?.map_name;
  const mapConfig = MAP_CONFIGS[mapName] || MAP_CONFIGS.de_mirage;
  const totalFrames = currentRoundData?.frames?.length || 0;
  const sampleRateMs = metadata?.sample_rate_ms || 62.5;
  const ctScore = useMemo(() => roundsSummary.slice(0, currentRound).filter(r => r.winner === 'CT').length, [roundsSummary, currentRound]);
  const tScore = useMemo(() => roundsSummary.slice(0, currentRound).filter(r => r.winner === 'T').length, [roundsSummary, currentRound]);
  const playerNameLookup = useMemo(() => {
    const namesById = new Map();

    currentRoundData?.frames?.forEach(frame => {
      frame.players?.forEach(player => {
        const playerId = String(player?.steam_id || '');
        const playerName = String(player?.name || '').trim();

        if (playerId && playerName && !namesById.has(playerId)) {
          namesById.set(playerId, playerName);
        }
      });
    });

    return namesById;
  }, [currentRoundData]);

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
        const res = await fetch(`${API_URL}/match/${matchId}/replay/metadata`, { credentials: 'include' });
        if (!res.ok) throw new Error('Error al cargar la repetición');
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
    const cached = roundCacheRef.current.get(roundNum);
    if (cached?.frames?.length) {
      setCurrentRoundData(cached);
      return;
    }
    if (preloadedData?.rounds?.[roundNum - 1]?.frames?.length) {
      setCurrentRoundData(preloadedData.rounds[roundNum - 1]);
      return;
    }
    
    setLoadingRound(true);
    try {
      const res = await fetch(`${API_URL}/match/${matchId}/replay/round/${roundNum}`, { credentials: 'include' });
      if (!res.ok) throw new Error('Error al cargar la ronda');
      const data = await res.json();
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
    if (!activeClip) return;
    pendingAiClipRef.current = activeClip;

    if (typeof activeClip.round !== 'number') return;
    const summaryIndex = roundsSummary.findIndex((round) => round.round === activeClip.round);
    const targetRoundIndex = summaryIndex >= 0 ? summaryIndex + 1 : activeClip.round;

    setCurrentRound((current) => {
      if (current === targetRoundIndex) return current;
      setCurrentTime(0);
      currentTimeRef.current = 0;
      setIsPlaying(false);
      return targetRoundIndex;
    });
  }, [activeClip, roundsSummary]);

  useEffect(() => {
    const clip = pendingAiClipRef.current;
    if (!clip || !currentRoundData?.frames?.length) return;

    if (typeof clip.round === 'number') {
      const summaryIndex = roundsSummary.findIndex((round) => round.round === clip.round);
      const targetRoundIndex = summaryIndex >= 0 ? summaryIndex + 1 : clip.round;
      if (targetRoundIndex !== currentRound) return;
    }

    if (clip.startTick <= 1) {
      setCurrentTime(clip.startTick);
      currentTimeRef.current = clip.startTick;
      setIsPlaying(true);
    } else {
      const targetIndex = getClosestFrameIndexByTick(currentRoundData.frames, clip.startTick);
      if (targetIndex >= 0) {
        seekToFrameIndex(targetIndex, currentRoundData.frames.length);
        setIsPlaying(true);
      }
    }

    pendingAiClipRef.current = null;
  }, [activeClip, currentRound, currentRoundData, roundsSummary, seekToFrameIndex]);

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

  // Canvas resize — fill the container width and height
  useEffect(() => {
    const updateSize = () => {
      setDpr(window.devicePixelRatio || 1);
      if (containerRef.current) {
        const mapArea = containerRef.current.querySelector('.replay-map-area');
        if (mapArea) {
          const rect = mapArea.getBoundingClientRect();
          setCanvasDim({ w: Math.max(rect.width, 400), h: Math.max(rect.height, 400) });
        }
      }
    };
    updateSize();
    // Small delay to let the layout settle after first render
    const initialTimer = setTimeout(updateSize, 100);
    window.addEventListener('resize', updateSize);
    return () => {
      clearTimeout(initialTimer);
      window.removeEventListener('resize', updateSize);
    };
  }, [isFullscreen]);

  // Auto-fit the view to the playable area whenever the round data or canvas dims change.
  // Scans all player positions in the round to compute the tightest bounding box,
  // then calculates the zoom and pan that centre it inside the canvas.
  useEffect(() => {
    if (!currentRoundData?.frames?.length || !mapConfig || canvasDim.w < 100) return;

    if (fitMode === 'contain') {
      setZoom(1);
      setPan({ x: 0, y: 0 });
      return;
    }

    let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
    currentRoundData.frames.forEach(frame => {
      frame.players?.forEach(p => {
        // Only include valid coordinates (filter out exact 0,0 which can be dead/unspawned outliers)
        if (p.x !== undefined && p.y !== undefined && (Math.abs(p.x) > 10 || Math.abs(p.y) > 10)) {
          minX = Math.min(minX, p.x); maxX = Math.max(maxX, p.x);
          minY = Math.min(minY, p.y); maxY = Math.max(maxY, p.y);
        }
      });
    });
    if (minX === Infinity) return;

    // Tight fit: minimal padding.
    const pad = 0.05;
    const rx = (maxX - minX) * pad;
    const ry = (maxY - minY) * pad;
    // Compute bounding box in 1024 virtual space coordinates
    const tl = translateCoords(minX - rx, maxY + ry, mapConfig, 1024);
    const br = translateCoords(maxX + rx, minY - ry, mapConfig, 1024);
    const bw = br.x - tl.x;
    const bh = br.y - tl.y;
    if (bw <= 0 || bh <= 0) return;

    // Determine how much of the virtual space is visible on screen
    const baseScale = Math.min(canvasDim.w / 1024, canvasDim.h / 1024);
    const visibleW = canvasDim.w / baseScale;
    const visibleH = canvasDim.h / baseScale;

    const rawFitZoom = Math.min(visibleW / bw, visibleH / bh);
    const fitZoom = Math.max(1.4, rawFitZoom * 1.2);
    const midX   = (tl.x + br.x) / 2;
    const midY   = (tl.y + br.y) / 2;
    setZoom(fitZoom);
    setPan({ x: (512 - midX) * fitZoom, y: (512 - midY) * fitZoom });
  }, [currentRoundData, mapConfig, canvasDim, fitMode]);

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
    const size = 1024; // Virtual logic size is always 1024
    const { w: cw, h: ch } = canvasDim;
    
    // Clear using physical pixels
    ctx.clearRect(0, 0, cw * dpr, ch * dpr);
    ctx.save();
    ctx.scale(dpr, dpr);

    // Background filling entire screen
    ctx.fillStyle = '#0b0e13';
    ctx.fillRect(0, 0, cw, ch);

    // Base coordinate system mapping virtual 1024 space to actual screen,
    // centering it properly so we can utilize the full width/height.
    const baseScale = Math.min(cw / 1024, ch / 1024);
    const offsetX = (cw - 1024 * baseScale) / 2;
    const offsetY = (ch - 1024 * baseScale) / 2;
    ctx.translate(offsetX, offsetY);
    ctx.scale(baseScale, baseScale);

    const effectiveZoom = Math.max(0.01, zoom);
    const now = performance.now();

    // ── Pan/zoom transform — applies to ALL objects including the map ────────
    ctx.translate(pan.x + 512, pan.y + 512);
    ctx.scale(effectiveZoom, effectiveZoom);
    ctx.translate(-512, -512);

    if (mapImageRef.current?.complete) {
      ctx.globalAlpha = 0.9;
      ctx.drawImage(mapImageRef.current, 0, 0, 1024, 1024);
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
          const smokeRadius = baseRadius * 0.9;
          const timeRemaining = effect.time_remaining || 15;
          const maxDuration = 18;
          const progress = Math.max(0, Math.min(1, timeRemaining / maxDuration));
          const pulse = 0.95 + Math.sin(nowSeconds * 1.5) * 0.05;
          
          const gradient = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, smokeRadius * pulse);
          gradient.addColorStop(0, 'rgba(140, 145, 155, 0.75)');
          gradient.addColorStop(0.7, 'rgba(100, 105, 115, 0.45)');
          gradient.addColorStop(1, 'rgba(80, 85, 90, 0)');
          
          ctx.beginPath();
          ctx.arc(pos.x, pos.y, smokeRadius * pulse, 0, Math.PI * 2);
          ctx.fillStyle = gradient;
          ctx.fill();
          
          if (progress > 0) {
            const startAngle = -Math.PI / 2;
            const endAngle = startAngle + (progress * Math.PI * 2);
            
            ctx.beginPath();
            ctx.arc(pos.x, pos.y, smokeRadius + 4, 0, Math.PI * 2);
            ctx.strokeStyle = 'rgba(20, 25, 30, 0.5)';
            ctx.lineWidth = 2.5;
            ctx.stroke();
            
            ctx.beginPath();
            ctx.arc(pos.x, pos.y, smokeRadius + 4, startAngle, endAngle, false); 
            ctx.strokeStyle = 'rgba(200, 220, 255, 0.8)';
            ctx.lineWidth = 2;
            ctx.lineCap = 'round';
            ctx.stroke();
            ctx.lineCap = 'butt';
          }
          
        } else if (effect.type === 'inferno') {
          const fireRadius = baseRadius * 0.85;
          const timeRemaining = effect.time_remaining || 7;
          const maxDuration = 7;
          const age = maxDuration - timeRemaining;
          
          const spreadProgress = Math.min(1, Math.max(0, age / 0.6));
          const spreadEased = 1 - Math.pow(1 - spreadProgress, 3);
          const currentRadius = fireRadius * spreadEased;
          
          const intensity = Math.min(1, timeRemaining / maxDuration);
          const flicker = 0.85 + Math.sin(nowSeconds * 10) * 0.15;
          
          const fireGradient = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, currentRadius);
          fireGradient.addColorStop(0, `rgba(255, 200, 80, ${0.85 * intensity * flicker})`);
          fireGradient.addColorStop(0.5, `rgba(255, 100, 20, ${0.6 * intensity})`);
          fireGradient.addColorStop(1, `rgba(180, 40, 0, 0)`);
          
          ctx.beginPath();
          ctx.arc(pos.x, pos.y, currentRadius, 0, Math.PI * 2);
          ctx.fillStyle = fireGradient;
          ctx.fill();
          
          const outerGlow = ctx.createRadialGradient(pos.x, pos.y, currentRadius * 0.7, pos.x, pos.y, currentRadius * 1.15);
          outerGlow.addColorStop(0, 'rgba(255, 80, 0, 0)');
          outerGlow.addColorStop(1, `rgba(255, 60, 0, ${0.15 * intensity})`);
          ctx.beginPath();
          ctx.arc(pos.x, pos.y, currentRadius * 1.15, 0, Math.PI * 2);
          ctx.fillStyle = outerGlow;
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
        
        // Draw smooth trajectory using dashed curves
        ctx.strokeStyle = color;
        ctx.lineWidth = 1.5;
        ctx.globalAlpha = 0.45;
        ctx.lineCap = 'round';
        ctx.lineJoin = 'round';
        ctx.setLineDash([4, 4]);
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
        ctx.setLineDash([]);
        ctx.globalAlpha = 1;
        ctx.lineCap = 'butt';
        ctx.lineJoin = 'miter';
        
        const currentPos = translateCoords(proj.x, proj.y, mapConfig, size);
        ctx.beginPath();
        ctx.arc(currentPos.x, currentPos.y, 4, 0, Math.PI * 2);
        ctx.fillStyle = color;
        ctx.globalAlpha = 0.85;
        ctx.fill();
        ctx.globalAlpha = 1;
      });
    }
    
    // =========================================================================
    // CINEMATIC SHOT ANIMATION SYSTEM - Ultra-realistic visual effects
    // =========================================================================
    
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
            duration: grenadeType === 'flashbang' ? 200 : 
                      grenadeType === 'he' || grenadeType === 'hegrenade' ? 400 : 300
          });
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
        const flashIntensity = Math.pow(1 - progress, 2);
        const radius = explosionRadius * 0.6 * (1 + progress * 0.3);
        
        const flashGradient = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, radius);
        flashGradient.addColorStop(0, `rgba(255, 255, 255, ${flashIntensity * 0.9})`);
        flashGradient.addColorStop(0.5, `rgba(255, 255, 220, ${flashIntensity * 0.4})`);
        flashGradient.addColorStop(1, 'rgba(255, 255, 255, 0)');
        
        ctx.beginPath();
        ctx.arc(pos.x, pos.y, radius, 0, Math.PI * 2);
        ctx.fillStyle = flashGradient;
        ctx.fill();
        
      } else if (explosion.type === 'he' || explosion.type === 'hegrenade') {
        const maxRadius = explosionRadius * 0.5;
        
        if (progress < 0.7) {
          const coreProgress = progress / 0.7;
          const scale = 1 - Math.pow(1 - coreProgress, 3);
          const coreAlpha = 1 - coreProgress;
          const currentRadius = maxRadius * scale;
          
          const fireGradient = ctx.createRadialGradient(pos.x, pos.y, 0, pos.x, pos.y, currentRadius);
          fireGradient.addColorStop(0, `rgba(255, 240, 160, ${coreAlpha})`);
          fireGradient.addColorStop(0.4, `rgba(255, 100, 20, ${coreAlpha * 0.8})`);
          fireGradient.addColorStop(1, `rgba(80, 30, 0, 0)`);
          
          ctx.beginPath();
          ctx.arc(pos.x, pos.y, currentRadius, 0, Math.PI * 2);
          ctx.fillStyle = fireGradient;
          ctx.fill();
        }
        
        if (progress < 0.6) {
           const shockAlpha = 1 - (progress / 0.6);
           const shockRadius = maxRadius * 1.3 * Math.pow(progress / 0.6, 0.5);
           ctx.beginPath();
           ctx.arc(pos.x, pos.y, shockRadius, 0, Math.PI * 2);
           ctx.strokeStyle = `rgba(180, 180, 180, ${shockAlpha * 0.4})`;
           ctx.lineWidth = 1.5;
           ctx.stroke();
        }
        
      } else if (explosion.type === 'decoy') {
        const puffAlpha = 1 - progress;
        ctx.beginPath();
        ctx.arc(pos.x, pos.y, 10 + progress * 12, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(150, 150, 150, ${puffAlpha * 0.4})`;
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
              duration: 600
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
      ctx.setLineDash([6, 4]);
      ctx.beginPath();
      ctx.moveTo(killerPos.x, killerPos.y);
      ctx.lineTo(victimPos.x, victimPos.y);
      ctx.strokeStyle = lineColor;
      ctx.lineWidth = killLine.headshot ? 2 : 1.5;
      ctx.globalAlpha = alpha * 0.7;
      ctx.stroke();
      ctx.setLineDash([]);
      
      // Small cross at victim position
      const cs = 4;
      ctx.beginPath();
      ctx.moveTo(victimPos.x - cs, victimPos.y - cs);
      ctx.lineTo(victimPos.x + cs, victimPos.y + cs);
      ctx.moveTo(victimPos.x + cs, victimPos.y - cs);
      ctx.lineTo(victimPos.x - cs, victimPos.y + cs);
      ctx.strokeStyle = '#FF4444';
      ctx.lineWidth = 2;
      ctx.globalAlpha = alpha;
      ctx.stroke();
      
      ctx.globalAlpha = 1;
    });
    
    // =========================================================================
    // BOMB STATE VISUALIZATION
    // =========================================================================
    
    if (frameData?.bomb) {
      const bomb = frameData.bomb;
      const bombPos = translateCoords(bomb.x, bomb.y, mapConfig, size);
      
      if (bomb.state === 'planted' || bomb.state === 'defusing') {
        const pulsePhase = (now / 250) % (Math.PI * 2);
        const pulseIntensity = 0.5 + Math.sin(pulsePhase) * 0.25;
        const bombRadius = 14;
        
        // Subtle danger zone
        const dangerGradient = ctx.createRadialGradient(bombPos.x, bombPos.y, 0, bombPos.x, bombPos.y, bombRadius * 1.8);
        dangerGradient.addColorStop(0, `rgba(255, 50, 50, ${pulseIntensity * 0.3})`);
        dangerGradient.addColorStop(1, 'rgba(200, 0, 0, 0)');
        
        ctx.beginPath();
        ctx.arc(bombPos.x, bombPos.y, bombRadius * 1.8, 0, Math.PI * 2);
        ctx.fillStyle = dangerGradient;
        ctx.fill();
        
        // Bomb dot
        ctx.beginPath();
        ctx.arc(bombPos.x, bombPos.y, 5, 0, Math.PI * 2);
        ctx.fillStyle = '#EF4444';
        ctx.fill();
        ctx.strokeStyle = 'rgba(255,255,255,0.7)';
        ctx.lineWidth = 1.5;
        ctx.stroke();
        
        // Site label
        if (bomb.site) {
          ctx.font = 'bold 10px Inter, sans-serif';
          ctx.textAlign = 'center';
          ctx.textBaseline = 'bottom';
          ctx.fillStyle = '#FFFFFF';
          ctx.fillText(bomb.site, bombPos.x, bombPos.y - 8);
        }
        
        // Defusing ring
        if (bomb.state === 'defusing') {
          const defusePhase = (now / 120) % (Math.PI * 2);
          ctx.beginPath();
          ctx.arc(bombPos.x, bombPos.y, 18, 0, Math.PI * 2);
          ctx.strokeStyle = `rgba(88, 166, 255, ${0.6 + Math.sin(defusePhase) * 0.3})`;
          ctx.lineWidth = 2;
          ctx.stroke();
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
      const radius = Math.max(10, Math.min(14, size / 72));
      const fontSize = Math.max(14, Math.round(size / 68));
      
      // ── Dead players first (X marks) ──
      frameData.players.forEach(player => {
        if (player.alive) return;
        const pos = translateCoords(player.x, player.y, mapConfig, size);
        ctx.strokeStyle = 'rgba(100, 100, 100, 0.3)';
        ctx.lineWidth = 1.5;
        const cs = 3.5;
        ctx.beginPath();
        ctx.moveTo(pos.x - cs, pos.y - cs); ctx.lineTo(pos.x + cs, pos.y + cs);
        ctx.moveTo(pos.x + cs, pos.y - cs); ctx.lineTo(pos.x - cs, pos.y + cs);
        ctx.stroke();
      });
      
      // ── PASS A: Draw all dots, cones, arrows (below names) ──
      frameData.players.forEach(player => {
        if (!player.alive) return;
        const pos = translateCoords(player.x, player.y, mapConfig, size);
        const theme = player.team === 'CT' ? THEME.ct : THEME.t;
        
        // Blindness indicator
        let flashVal = 0;
        if (typeof player.flash_duration === 'number' && player.flash_duration > 0) {
            flashVal = Math.min(1, player.flash_duration / 5.0);
        } else if (player.is_blinded) {
            flashVal = 0.7;
        }
        if (flashVal > 0) {
           const blindRadius = radius + 3 + (flashVal * 3);
           ctx.beginPath();
           ctx.arc(pos.x, pos.y, blindRadius, 0, Math.PI * 2);
           ctx.strokeStyle = `rgba(255, 255, 200, ${flashVal * 0.7})`;
           ctx.lineWidth = 1.5;
           ctx.stroke();
        }

        // View direction: subtle FOV cone
        const yawRad = (-player.yaw * Math.PI) / 180;
        const fovHalf = (25 * Math.PI) / 180;
        const coneLen = radius + 16;
        ctx.beginPath();
        ctx.moveTo(pos.x, pos.y);
        ctx.arc(pos.x, pos.y, coneLen, yawRad - fovHalf, yawRad + fovHalf);
        ctx.closePath();
        const coneGrad = ctx.createRadialGradient(pos.x, pos.y, radius, pos.x, pos.y, coneLen);
        coneGrad.addColorStop(0, player.team === 'CT' ? 'rgba(34,211,238,0.08)' : 'rgba(234,179,8,0.08)');
        coneGrad.addColorStop(1, 'rgba(0,0,0,0)');
        ctx.fillStyle = coneGrad;
        ctx.fill();

        // Player circle
        ctx.shadowColor = theme.glow;
        ctx.shadowBlur = 8;
        ctx.beginPath();
        ctx.arc(pos.x, pos.y, radius, 0, Math.PI * 2);
        ctx.fillStyle = theme.primary;
        ctx.fill();
        ctx.strokeStyle = 'rgba(255,255,255,0.45)';
        ctx.lineWidth = 1.2;
        ctx.stroke();
        ctx.shadowBlur = 0;
        
        // Directional arrow (triangle on circle edge)
        const arrowLen = radius * 0.65;
        const arrowWidth = radius * 0.4;
        const arrowDist = radius + 2;
        const tipX = pos.x + Math.cos(yawRad) * (arrowDist + arrowLen);
        const tipY = pos.y + Math.sin(yawRad) * (arrowDist + arrowLen);
        const baseX = pos.x + Math.cos(yawRad) * arrowDist;
        const baseY = pos.y + Math.sin(yawRad) * arrowDist;
        const perpX = -Math.sin(yawRad) * arrowWidth;
        const perpY = Math.cos(yawRad) * arrowWidth;
        ctx.beginPath();
        ctx.moveTo(tipX, tipY);
        ctx.lineTo(baseX + perpX, baseY + perpY);
        ctx.lineTo(baseX - perpX, baseY - perpY);
        ctx.closePath();
        ctx.fillStyle = theme.primary;
        ctx.globalAlpha = 0.85;
        ctx.fill();
        ctx.globalAlpha = 1;
      });
      
      // ── PASS B: Draw ALL name labels on top (always visible) ──
      frameData.players.forEach(player => {
        if (!player.alive) return;
        const pos = translateCoords(player.x, player.y, mapConfig, size);
        const theme = player.team === 'CT' ? THEME.ct : THEME.t;
        
        const name = (player.name || 'Jugador').substring(0, 10);
        ctx.font = `600 ${fontSize}px Inter, sans-serif`;
        ctx.textAlign = 'center';
        ctx.textBaseline = 'bottom';
        const textWidth = ctx.measureText(name).width;
        const pillPadX = 6;
        const pillPadY = 3;
        const pillH = fontSize + pillPadY * 2;
        const pillW = textWidth + pillPadX * 2;
        const labelY = pos.y - radius - 6;
        
        // Dark pill background
        const pillX = pos.x - pillW / 2;
        const pillTop = labelY - pillH;
        ctx.fillStyle = 'rgba(0, 0, 0, 0.7)';
        ctx.beginPath();
        ctx.roundRect(pillX, pillTop, pillW, pillH, 2);
        ctx.fill();
        
        // Name text
        ctx.fillStyle = theme.primary;
        ctx.fillText(name, pos.x, labelY - pillPadY);
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

  }, [canvasDim, dpr, currentRoundData, totalFrames, mapConfig, zoom, pan, annotations]);

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

  const handleTimelineKeyDown = (event) => {
    const step = event.shiftKey ? 0.1 : 0.02;
    let nextTime = currentTimeRef.current;

    if (event.key === 'ArrowLeft' || event.key === 'ArrowDown') nextTime -= step;
    else if (event.key === 'ArrowRight' || event.key === 'ArrowUp') nextTime += step;
    else if (event.key === 'Home') nextTime = 0;
    else if (event.key === 'End') nextTime = 1;
    else return;

    event.preventDefault();
    const clampedTime = Math.max(0, Math.min(1, nextTime));
    setCurrentTime(clampedTime);
    currentTimeRef.current = clampedTime;
  };
  
  useEffect(() => {
    if (!isDraggingTimeline) return;
    const handleMove = (e) => updateTimeFromMouse(e);
    const handleUp = () => setIsDraggingTimeline(false);
    window.addEventListener('pointermove', handleMove);
    window.addEventListener('pointerup', handleUp);
    return () => {
      window.removeEventListener('pointermove', handleMove);
      window.removeEventListener('pointerup', handleUp);
    };
  }, [isDraggingTimeline]);

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.target.closest('input, select, textarea, button, [contenteditable="true"]')) return;
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
      <div className="replay-container replay-loading" role="status" aria-live="polite">
        <motion.div 
          className="replay-loader"
          animate={{ rotate: 360 }}
          transition={{ repeat: Infinity, duration: 1, ease: "linear" }}
        />
        <p>Cargando repetición…</p>
      </div>
    );
  }

  if (!metadata) {
    return (
      <div className="replay-container replay-empty" role="status">
        <div className="replay-empty-icon"><MonitorPlay size={28} aria-hidden="true" /></div>
        <strong>Replay 2D todavía no disponible</strong>
        <p>Esta demo no incluye fotogramas reproducibles o sigue procesándose.</p>
        <span>Puedes revisar los hallazgos y preparar tu plan mientras tanto.</span>
      </div>
    );
  }

  const maxRounds = roundsSummary.length || 0;
  const currentFrameData = getInterpolatedFrame();
  const ctPlayers = (currentFrameData?.players?.filter(p => p.team === 'CT') || [])
    .sort((a, b) => String(a.steam_id).localeCompare(String(b.steam_id)));
  const tPlayers = (currentFrameData?.players?.filter(p => p.team === 'T') || [])
    .sort((a, b) => String(a.steam_id).localeCompare(String(b.steam_id)));

  // Compute timer display
  const timeRemaining = currentFrameData?.time_remaining || 0;
  const timerMinutes = Math.floor(timeRemaining / 60);
  const timerSeconds = Math.floor(timeRemaining % 60);
  const timerDisplay = `${timerMinutes}:${timerSeconds.toString().padStart(2, '0')}`;

  return (
    <div className={`replay-container ${isFullscreen ? 'fullscreen' : ''} ${compactTeams ? 'compact-teams' : ''}`} ref={containerRef}>
      {/* Loading overlay */}
      <AnimatePresence>
        {loadingRound && (
          <motion.div 
            className="replay-round-loading"
            role="status"
            aria-live="polite"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          >
            <div className="replay-loader-small" />
            <span>Cargando ronda {currentRound}…</span>
          </motion.div>
        )}
      </AnimatePresence>

      {/* ═══════════════════════════════════════════════════════════
          TOP BAR: Round chips
          ═══════════════════════════════════════════════════════════ */}
      <div className="replay-top-bar">
        <button type="button" className="dock-btn round-nav-btn" aria-label="Ronda anterior" onClick={() => handleRoundChange(-1)} disabled={currentRound <= 1}>
          <ChevronLeft size={14} />
        </button>
        <div className="dock-round-chips">
          {roundsSummary.map((round, idx) => (
            <button
              key={idx}
              className={`dock-round-chip ${currentRound === idx + 1 ? 'active' : ''} ${round.winner?.toLowerCase() || ''}`}
              aria-label={`Ir a la ronda ${idx + 1}`}
              aria-current={currentRound === idx + 1 ? 'true' : undefined}
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
        <button type="button" className="dock-btn round-nav-btn" aria-label="Ronda siguiente" onClick={() => handleRoundChange(1)} disabled={currentRound >= maxRounds}>
          <ChevronRight size={14} />
        </button>
        {compactTeams && (
          <button
            type="button"
            className={`dock-btn team-toggle-btn ${isTeamsPanelOpen ? 'active' : ''}`}
            aria-label={isTeamsPanelOpen ? 'Ocultar equipos' : 'Mostrar equipos'}
            aria-expanded={isTeamsPanelOpen}
            onClick={() => setIsTeamsPanelOpen((isOpen) => !isOpen)}
          >
            <Users size={14} aria-hidden="true" />
            <span>Equipos</span>
          </button>
        )}
      </div>

      {/* ═══════════════════════════════════════════════════════════
          MAIN AREA: Map (left) | Info Panel (right)
          ═══════════════════════════════════════════════════════════ */}
      <div className="replay-main-layout">
        
        {/* LEFT: Map area */}
        <div className="replay-map-area">
          {/* Round info overlay */}
          <div className="replay-round-overlay">
            <span className="round-overlay-label">ROUND {currentRound}</span>
            <span className={`round-overlay-timer ${timeRemaining <= 10 ? 'danger' : ''}`}>{timerDisplay}</span>
          </div>

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
              width={canvasDim.w * dpr}
              height={canvasDim.h * dpr}
              style={{ width: canvasDim.w, height: canvasDim.h }}
              className="replay-canvas"
              role="img"
              aria-label={`Replay táctico de la ronda ${currentRound}. Usa el panel de jugadores y el timeline para consultar el estado de la ronda.`}
            />
            
            {/* Kill Feed Overlay — top-right of canvas */}
            <div className="replay-killfeed-overlay">
              <KillFeedOverlay 
                events={currentRoundData?.events}
                players={currentFrameData?.players}
                frames={currentRoundData?.frames}
                playerNameLookup={playerNameLookup}
                currentTick={currentFrameData?.interpolatedTick || currentFrameData?.tick || 0}
              />
            </div>

            {/* Zoom Controls */}
            <div className="replay-zoom-controls">
              <button type="button" aria-label="Acercar mapa" onClick={() => setZoom(z => Math.min(4, z * 1.25))}><ZoomIn size={14} aria-hidden="true" /></button>
              <button type="button" aria-label="Alejar mapa" onClick={() => setZoom(z => Math.max(0.5, z / 1.25))}><ZoomOut size={14} aria-hidden="true" /></button>
              <button type="button" aria-label="Restablecer vista del mapa" onClick={() => { setZoom(1); setPan({ x: 0, y: 0 }); }}><RotateCcw size={14} aria-hidden="true" /></button>
            </div>
          </div>
        </div>

        {/* RIGHT: Info panel — both teams */}
        <div className={`replay-info-panel ${isTeamsPanelOpen ? 'open' : ''}`}>
          {/* CT Section */}
          <div className="team-section ct">
            <div className="team-section-header">
              <span className="team-header-label">TEAM A</span>
              <span className="team-header-sub">Counter-Terrorists</span>
              <span className="team-header-score">{ctScore}</span>
            </div>
            <div className="team-players">
              {ctPlayers.map((player, i) => (
                <PlayerCard key={player.steam_id || i} player={player} team="CT" />
              ))}
            </div>
          </div>

          {/* T Section */}
          <div className="team-section t">
            <div className="team-section-header">
              <span className="team-header-label">TEAM B</span>
              <span className="team-header-sub">Terrorists</span>
              <span className="team-header-score">{tScore}</span>
            </div>
            <div className="team-players">
              {tPlayers.map((player, i) => (
                <PlayerCard key={player.steam_id || i} player={player} team="T" />
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* ═══════════════════════════════════════════════════════════
          BOTTOM DOCK: Timeline + Transport Controls
          ═══════════════════════════════════════════════════════════ */}
      <div className="replay-bottom-dock">
        {/* Timeline — full width */}
        <div 
          ref={timelineRef}
          className={`replay-timeline ${isDraggingTimeline ? 'dragging' : ''}`}
          onPointerDown={handleTimelineMouseDown}
          onKeyDown={handleTimelineKeyDown}
          role="slider"
          tabIndex="0"
          aria-label="Posición de la ronda"
          aria-valuemin="0"
          aria-valuemax="100"
          aria-valuenow={Math.round(currentTime * 100)}
          aria-valuetext={`${timerDisplay} restantes en la ronda ${currentRound}`}
        >
          <div className="timeline-track" />
          <div className="timeline-progress" style={{ width: `${currentTime * 100}%` }} />
          <div className="timeline-handle" style={{ left: `${currentTime * 100}%` }} />
        </div>

        {/* Playback row */}
        <div className="dock-controls-row">
          {/* Left: Play + speed */}
          <div className="dock-section">
            <button 
              type="button"
              className={`dock-play-btn ${isPlaying ? 'playing' : ''}`}
              aria-label={isPlaying ? 'Pausar replay' : 'Reproducir replay'}
              onClick={() => setIsPlaying(!isPlaying)}
            >
              {isPlaying ? <Pause size={16} /> : <Play size={16} />}
            </button>
            <select 
              className="dock-speed"
              name="playback-speed"
              aria-label="Velocidad de reproducción"
              value={playbackSpeed}
              onChange={(e) => setPlaybackSpeed(parseFloat(e.target.value))}
            >
              <option value="0.25">0.25×</option>
              <option value="0.5">0.5×</option>
              <option value="1">1×</option>
              <option value="2">2×</option>
              <option value="4">4×</option>
            </select>
          </div>

          {/* Center: round navigation */}
          <div className="dock-section dock-transport">
            <button
              type="button"
              className="dock-btn"
              onClick={() => handleRoundChange(-1)}
              disabled={currentRound <= 1}
              title="Ronda anterior"
              aria-label="Ronda anterior"
            >
              <SkipBack size={14} />
            </button>
            <button
              type="button"
              className="dock-btn"
              onClick={() => handleRoundChange(1)}
              disabled={currentRound >= maxRounds}
              title="Ronda siguiente"
              aria-label="Ronda siguiente"
            >
              <SkipForward size={14} />
            </button>
          </div>

          {/* Right: Fullscreen */}
          <div className="dock-section">
            <button type="button" className="dock-btn" aria-label="Ver replay a pantalla completa" onClick={toggleFullscreen}>
              <Maximize2 size={14} />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
