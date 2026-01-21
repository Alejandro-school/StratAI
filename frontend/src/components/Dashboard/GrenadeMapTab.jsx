// frontend/src/components/Dashboard/GrenadeMapTab.jsx
// Grenade Intel System - Professional CS2 Analysis Dashboard
import React, { useState, useMemo } from 'react';
import { useGrenadeStats } from '../../hooks/useGrenadeStats';
import { 
  TrendingUp, AlertTriangle, Target, Zap,
  Eye, EyeOff, ChevronRight, X, MapPin, Award, Crosshair
} from 'lucide-react';
import '../../styles/Dashboard/grenadeMapTab.css';

// ============================================
// CS2 OFFICIAL GRENADE ICONS (PNG Images)
// ============================================

// Image paths for real CS2 grenade icons
const GRENADE_IMAGES = {
  smoke: '/images/weapons/weapon_smokegrenade.png',
  flash: '/images/weapons/weapon_flashbang.png',
  he: '/images/weapons/weapon_hegrenade.png',
  molotov: '/images/weapons/weapon_molotov.png',
  incendiary: '/images/weapons/weapon_incgrenade.png'
};

// CS2 Grenade Image Component
const GrenadeImage = ({ type, size = 24, className = '' }) => (
  <img 
    src={GRENADE_IMAGES[type] || GRENADE_IMAGES.he}
    alt={type}
    width={size}
    height={size}
    className={`grenade-icon-img ${className}`}
    style={{ objectFit: 'contain', filter: 'drop-shadow(0 2px 4px rgba(0,0,0,0.3))' }}
  />
);

// Grenade type configuration
const GRENADE_TYPES = {
  smoke: { 
    label: 'Smoke', 
    labelEs: 'Humos',
    type: 'smoke',
    color: '#60a5fa',
    glowColor: 'rgba(96, 165, 250, 0.5)',
    bgColor: 'rgba(96, 165, 250, 0.15)',
    gradientStart: '#93c5fd',
    gradientEnd: '#3b82f6',
    effectLabel: 'Efectividad',
    effectUnit: '%',
    explosionColor: 'rgba(96, 165, 250, 0.8)'
  },
  flash: { 
    label: 'Flash', 
    labelEs: 'Flashes',
    type: 'flash',
    color: '#fbbf24',
    glowColor: 'rgba(251, 191, 36, 0.5)',
    bgColor: 'rgba(251, 191, 36, 0.15)',
    gradientStart: '#fde68a',
    gradientEnd: '#f59e0b',
    effectLabel: 'Cegados/Flash',
    effectUnit: '',
    explosionColor: 'rgba(255, 255, 255, 0.95)'
  },
  he: { 
    label: 'HE', 
    labelEs: 'HE',
    type: 'he',
    color: '#f87171',
    glowColor: 'rgba(248, 113, 113, 0.5)',
    bgColor: 'rgba(248, 113, 113, 0.15)',
    gradientStart: '#fca5a5',
    gradientEnd: '#ef4444',
    effectLabel: 'Daño Promedio',
    effectUnit: ' dmg',
    explosionColor: 'rgba(248, 113, 113, 0.9)'
  },
  molotov: { 
    label: 'Molotov', 
    labelEs: 'Mollys',
    type: 'molotov',
    color: '#fb923c',
    glowColor: 'rgba(251, 146, 60, 0.5)',
    bgColor: 'rgba(251, 146, 60, 0.15)',
    gradientStart: '#fdba74',
    gradientEnd: '#ea580c',
    effectLabel: 'Daño Promedio',
    effectUnit: ' dmg',
    explosionColor: 'rgba(251, 146, 60, 0.85)'
  }
};

// ============================================
// SMART GRENADE MARKER - Contextual with efficiency bar
// ============================================

const GrenadeMarkerSmart = ({ cluster, type, onClick, isSelected, isHighlighted }) => {
  const config = GRENADE_TYPES[type];
  
  // Calculate efficiency for visual feedback
  const getEfficiency = () => {
    if (type === 'smoke') return 100;
    if (type === 'flash') return Math.min(100, Math.round((cluster.avg_blinded || 0) * 40));
    if (type === 'he' || type === 'molotov') return Math.min(100, Math.round((cluster.avg_damage || 0) * 2));
    return 50;
  };
  
  const efficiency = getEfficiency();
  const isGood = efficiency >= 70;
  const isBad = efficiency < 40;
  
  // LARGER sizes for better visibility
  const baseSize = 48;
  const size = Math.min(64, baseSize + Math.floor(cluster.count / 3) * 4);
  const imgSize = size * 0.65;
  
  return (
    <button
      className={`grenade-marker-smart ${type} ${isSelected ? 'selected' : ''} ${isHighlighted ? 'highlighted' : ''} ${isGood ? 'good' : ''} ${isBad ? 'bad' : ''}`}
      style={{
        left: `${cluster.x}%`,
        top: `${cluster.y}%`,
        '--marker-color': config.color,
        '--marker-glow': config.glowColor,
        '--marker-bg': config.bgColor,
        '--efficiency': `${efficiency}%`
      }}
      onClick={() => onClick(cluster, type)}
    >
      <div className="marker-pulse" />
      <div className="marker-body" style={{ width: size, height: size }}>
        <GrenadeImage type={type} size={imgSize} />
      </div>
      <div className="marker-count-badge">
        {cluster.count}
      </div>
      <div className="marker-efficiency-bar">
        <div className="efficiency-fill" style={{ width: `${efficiency}%` }} />
      </div>
      {cluster.lineup_name && (
        <span className="marker-label">{cluster.lineup_name}</span>
      )}
    </button>
  );
};

// ============================================
// TRAJECTORY VISUALIZATION - Natural Flight with Grenade Icon
// ============================================

const TrajectoryVisualization = ({ cluster, type }) => {
  const config = GRENADE_TYPES[type];
  const [animationKey, setAnimationKey] = useState(0);
  
  // Reset animation when cluster changes
  React.useEffect(() => {
    setAnimationKey(prev => prev + 1);
  }, [cluster]);
  
  if (!cluster?.trajectories || cluster.trajectories.length === 0) return null;
  
  // Calculate distance for animation timing
  const getDistance = (traj) => {
    const dx = traj.x2 - traj.x1;
    const dy = traj.y2 - traj.y1;
    return Math.sqrt(dx * dx + dy * dy);
  };
  
  return (
    <div className="trajectory-container animated" key={animationKey}>
      {cluster.trajectories.slice(0, 3).map((traj, idx) => {
        const dist = getDistance(traj);
        const angle = Math.atan2(traj.y2 - traj.y1, traj.x2 - traj.x1) * (180 / Math.PI);
        
        // Animation timing
        const animationDelay = idx * 0.4;
        const flightDuration = 0.6 + (dist / 80) * 0.3; // Natural speed
        
        return (
          <div key={idx} className="trajectory-flight-group">
            {/* Dashed line trail */}
            <svg 
              className="trajectory-line-svg"
              style={{
                animationDelay: `${animationDelay}s`,
                '--flight-duration': `${flightDuration}s`
              }}
            >
              <line
                x1={`${traj.x1}%`}
                y1={`${traj.y1}%`}
                x2={`${traj.x2}%`}
                y2={`${traj.y2}%`}
                stroke={config.color}
                strokeWidth="2"
                strokeDasharray="6 4"
                strokeLinecap="round"
                opacity="0.6"
                className="trajectory-dashed-line"
              />
            </svg>
            
            {/* Flying grenade icon */}
            <div 
              className="flying-grenade"
              style={{
                '--start-x': `${traj.x1}%`,
                '--start-y': `${traj.y1}%`,
                '--end-x': `${traj.x2}%`,
                '--end-y': `${traj.y2}%`,
                '--flight-duration': `${flightDuration}s`,
                '--angle': `${angle}deg`,
                animationDelay: `${animationDelay}s`
              }}
            >
              <GrenadeImage type={type} size={24} />
            </div>
            
            {/* Explosion effect */}
            <div 
              className={`explosion-effect ${type}`}
              style={{
                left: `${traj.x2}%`,
                top: `${traj.y2}%`,
                '--explosion-color': config.explosionColor,
                '--glow-color': config.glowColor,
                animationDelay: `${animationDelay + flightDuration}s`
              }}
            >
              <div className="explosion-ring-1" />
              <div className="explosion-ring-2" />
              {type === 'flash' && <div className="flash-white-burst" />}
              {(type === 'he' || type === 'molotov') && <div className="fire-burst" />}
              {type === 'smoke' && <div className="smoke-puff" />}
            </div>
            
            {/* Landing indicator with grenade icon */}
            <div 
              className={`landing-marker ${type}`}
              style={{
                left: `${traj.x2}%`,
                top: `${traj.y2}%`,
                '--marker-color': config.color,
                '--marker-glow': config.glowColor,
                animationDelay: `${animationDelay + flightDuration + 0.2}s`
              }}
            >
              <GrenadeImage type={type} size={22} />
            </div>
          </div>
        );
      })}
      
      {/* Start point indicator */}
      {cluster.trajectories.length > 0 && (
        <div 
          className="throw-point"
          style={{
            left: `${cluster.trajectories[0].x1}%`,
            top: `${cluster.trajectories[0].y1}%`,
            '--point-color': config.gradientStart
          }}
        />
      )}
    </div>
  );
};


// ============================================
// GRENADE ARSENAL - Compact Summary with Filters
// ============================================

const GrenadeArsenal = ({ summary, matchesAnalyzed, visibleTypes, onToggleType }) => {
  // Calculate stats for each type
  const getEffectStat = (type, stats) => {
    if (!stats) return { value: 0, label: '' };
    switch (type) {
      case 'smoke': 
        return { 
          value: stats.thrown || 0, 
          label: 'lanzados',
          unit: ''
        };
      case 'flash': 
        return { 
          value: stats.avg_blinded?.toFixed(1) || 0, 
          label: 'cegados/flash',
          unit: ''
        };
      case 'he': 
      case 'molotov':
        return { 
          value: Math.round(stats.avg_damage || 0), 
          label: 'daño prom.',
          unit: ''
        };
      default: 
        return { value: 0, label: '' };
    }
  };

  const totalGrenades = Object.values(summary).reduce((sum, s) => sum + (s?.thrown || 0), 0);

  return (
    <div className="grenade-arsenal">
      {/* Header */}
      <div className="arsenal-header">
        <div className="arsenal-title">
          <Crosshair size={18} className="arsenal-icon" />
          <h4>Arsenal de Granadas</h4>
        </div>
        <div className="arsenal-meta">
          <span className="total-count">{totalGrenades} granadas</span>
          <span className="matches-count">{matchesAnalyzed} partidas</span>
        </div>
      </div>

      {/* Stats Grid - Larger Icons */}
      <div className="arsenal-grid">
        {Object.entries(GRENADE_TYPES).map(([gType, config]) => {
          const stats = summary[gType] || {};
          const effect = getEffectStat(gType, stats);
          const isActive = visibleTypes[gType];
          
          return (
            <button
              key={gType}
              className={`arsenal-item ${isActive ? 'active' : 'inactive'}`}
              onClick={() => onToggleType(gType)}
              style={{ '--item-color': config.color, '--item-bg': config.bgColor }}
            >
              <div className="item-icon-large">
                <GrenadeImage type={gType} size={40} className={isActive ? '' : 'grayscale'} />
              </div>
              <div className="item-stats">
                <div className="item-count-large">{stats.thrown || 0}</div>
                <div className="item-effect-compact">
                  <span className="effect-value">{effect.value}{effect.unit}</span>
                  <span className="effect-label-small">{config.effectLabel.toUpperCase()}</span>
                </div>
              </div>
              <div className="item-visibility-icon">
                {isActive ? <Eye size={12} /> : <EyeOff size={12} />}
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
};

// ============================================
// LINEUP DETAIL PANEL - Rich Information
// ============================================

const LineupDetailPanel = ({ cluster, type, onClose }) => {
  if (!cluster || !type) {
    return (
      <div className="lineup-detail-panel empty">
        <div className="empty-content">
          <Target size={40} strokeWidth={1.5} />
          <h4>Selecciona un Lineup</h4>
          <p>Haz clic en un marcador del mapa para ver estadísticas detalladas y consejos</p>
        </div>
      </div>
    );
  }
  
  const config = GRENADE_TYPES[type];
  
  // Calculate metrics
  const getMetrics = () => {
    switch (type) {
      case 'smoke':
        return {
          primary: { value: cluster.count, label: 'Lanzados', unit: '', isGood: true },
          secondary: {}
        };
      case 'flash':
        return {
          primary: { value: (cluster.avg_blinded || 0).toFixed(1), label: 'Cegados/Flash', isGood: cluster.avg_blinded >= 1.5 },
          secondary: { value: cluster.total_blinded || 0, label: 'Total Cegados' }
        };
      case 'he':
      case 'molotov':
        return {
          primary: { value: cluster.avg_damage || 0, label: 'Daño Promedio', isGood: cluster.avg_damage >= 40 },
          secondary: { value: cluster.total_damage || 0, label: 'Daño Total' }
        };
      default:
        return { primary: {}, secondary: {} };
    }
  };
  
  const metrics = getMetrics();
  const lineupName = cluster.lineup_name || cluster.areas?.[0] || 'Zona del mapa';
  
  // Generate insight - only for actually useful feedback
  const getInsight = () => {

    // Flash success
    if (type === 'flash' && cluster.avg_blinded >= 1.5) {
      return { type: 'success', text: `Excelente flash! ${cluster.avg_blinded.toFixed(1)} cegados en promedio.` };
    }
    // HE/Molly with actual kills or high damage
    if ((type === 'he' || type === 'molotov') && cluster.avg_damage >= 50) {
      return { type: 'success', text: `Alto impacto: ${cluster.avg_damage} daño promedio.` };
    }
    // Don't show "bad" feedback for mollys/HE - they're often used for area denial, not damage
    return null;
  };
  
  const insight = getInsight();
  
  return (
    <div className="lineup-detail-panel" style={{ '--panel-color': config.color, '--panel-bg': config.bgColor }}>
      {/* Header */}
      <div className="panel-header">
        <div className="panel-title-row">
          <div className="panel-icon" style={{ background: config.bgColor }}>
            <GrenadeImage type={type} size={28} />
          </div>
          <div className="panel-title-text">
            <h3>{config.label.toUpperCase()}</h3>
            <span className="panel-location">
              <MapPin size={12} />
              {lineupName}
            </span>
          </div>
        </div>
        <button className="panel-close" onClick={onClose}>
          <X size={18} />
        </button>
      </div>
      
      {/* Route Info */}
      {cluster.from_area && (
        <div className="route-info">
          <span className="route-from">{cluster.from_area}</span>
          <ChevronRight size={16} />
          <span className="route-to">{cluster.areas?.[0] || 'Destino'}</span>
        </div>
      )}
      
      {/* Usage Badge */}
      <div className="usage-badge">
        <Zap size={14} />
        <span>Lanzadas <strong>{cluster.count}</strong> veces</span>
      </div>
      
      {/* Metrics */}
      <div className="metrics-grid">
        <div className={`metric-card ${metrics.primary.isGood ? 'good' : ''}`}>
          <span className="metric-value">{metrics.primary.value}{metrics.primary.unit || ''}</span>
          <span className="metric-label">{metrics.primary.label}</span>
        </div>
        <div className={`metric-card ${metrics.secondary.isBad ? 'bad' : ''}`}>
          <span className="metric-value">{metrics.secondary.value}{metrics.secondary.unit || ''}</span>
          <span className="metric-label">{metrics.secondary.label}</span>
        </div>
      </div>
      
      {/* Impact Zones */}
      {cluster.areas && cluster.areas.length > 1 && (
        <div className="zones-section">
          <h5>Zonas de Impacto</h5>
          <div className="zone-tags">
            {cluster.areas.slice(0, 4).map((area, idx) => (
              <span key={idx} className="zone-tag">{area}</span>
            ))}
          </div>
        </div>
      )}
      
      {/* Insight */}
      {insight && (
        <div className={`insight-box ${insight.type}`}>
          {insight.type === 'success' ? <TrendingUp size={16} /> : <AlertTriangle size={16} />}
          <p>{insight.text}</p>
        </div>
      )}
    </div>
  );
};

// ============================================
// TOP LINEUPS SECTION - Best & Worst
// ============================================

const TopLineupsSection = ({ grenadeData, summary, onLineupHover, onLineupClick }) => {
  // Flatten all clusters and sort by effectiveness
  const allLineups = useMemo(() => {
    const lineups = [];
    
    Object.entries(grenadeData).forEach(([type, clusters]) => {
      clusters.forEach(cluster => {
        let score = 0;
        let effectLabel = '';
        
        if (type === 'smoke') {
          score = (cluster.count || 0) * 10;
          effectLabel = `${cluster.count} lanzados`;
        } else if (type === 'flash') {
          score = (cluster.avg_blinded || 0) * 40;
          effectLabel = `${(cluster.avg_blinded || 0).toFixed(1)} cegados`;
        } else if (type === 'he' || type === 'molotov') {
          score = (cluster.avg_damage || 0) * 2;
          effectLabel = `${Math.round(cluster.avg_damage || 0)} dmg`;
        }
        
        if (cluster.count >= 2) { // Only show lineups used at least twice
          lineups.push({
            ...cluster,
            type,
            score,
            effectLabel,
            name: cluster.lineup_name || cluster.areas?.[0] || 'Unknown'
          });
        }
      });
    });
    
    return lineups.sort((a, b) => b.score - a.score);
  }, [grenadeData]);
  
  const bestLineups = allLineups.slice(0, 3);
  const worstLineups = allLineups.slice(-3).reverse().filter(l => l.score < 60);
  
  if (allLineups.length === 0) return null;
  
  return (
    <div className="top-lineups-section">
      {/* Best Lineups */}
      {bestLineups.length > 0 && (
        <div className="lineup-category best">
          <div className="category-header">
            <Award size={16} className="category-icon" />
            <h5>Mejores Lineups</h5>
          </div>
          <div className="lineup-list">
            {bestLineups.map((lineup, idx) => {
              const lineupConfig = GRENADE_TYPES[lineup.type];
              return (
                <button
                  key={idx}
                  className="lineup-item"
                  onMouseEnter={() => onLineupHover?.(lineup)}
                  onMouseLeave={() => onLineupHover?.(null)}
                  onClick={() => onLineupClick?.(lineup, lineup.type)}
                  style={{ '--lineup-color': lineupConfig.color }}
                >
                  <div className="lineup-rank">{idx + 1}</div>
                  <div className="lineup-icon">
                    <GrenadeImage type={lineup.type} size={18} />
                  </div>
                  <div className="lineup-info">
                    <span className="lineup-name">{lineup.name}</span>
                    <span className="lineup-effect">{lineup.effectLabel}</span>
                  </div>
                  <span className="lineup-count">×{lineup.count}</span>
                </button>
              );
            })}
          </div>
        </div>
      )}
      
      {/* Needs Practice */}
      {worstLineups.length > 0 && (
        <div className="lineup-category practice">
          <div className="category-header">
            <AlertTriangle size={16} className="category-icon" />
            <h5>Necesitan Práctica</h5>
          </div>
          <div className="lineup-list">
            {worstLineups.map((lineup, idx) => {
              const lineupConfig = GRENADE_TYPES[lineup.type];
              return (
                <button
                  key={idx}
                  className="lineup-item needs-practice"
                  onMouseEnter={() => onLineupHover?.(lineup)}
                  onMouseLeave={() => onLineupHover?.(null)}
                  onClick={() => onLineupClick?.(lineup, lineup.type)}
                  style={{ '--lineup-color': lineupConfig.color }}
                >
                  <div className="lineup-icon">
                    <GrenadeImage type={lineup.type} size={18} />
                  </div>
                  <div className="lineup-info">
                    <span className="lineup-name">{lineup.name}</span>
                    <span className="lineup-effect bad">{lineup.effectLabel}</span>
                  </div>
                  <span className="lineup-count">×{lineup.count}</span>
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
};

// ============================================
// MAIN GRENADE TAB COMPONENT - Redesigned
// ============================================

const GrenadeMapTab = ({ 
  mapName = 'de_dust2',
  activeSide = 'all',
  visibleTypes,
  onToggleType,
  selectedCluster,
  selectedType,
  onClusterSelect,
  onClusterClose
}) => {
  const { 
    grenadeData, 
    summary, 
    insights, 
    matchesAnalyzed, 
    loading 
  } = useGrenadeStats(mapName);
  
  const [localVisibleTypes, setLocalVisibleTypes] = useState({
    smoke: true,
    flash: true,
    he: true,
    molotov: true
  });
  
  const [hoveredLineup, setHoveredLineup] = useState(null);
  
  const effectiveVisibleTypes = visibleTypes || localVisibleTypes;
  
  const handleToggle = (type) => {
    if (onToggleType) {
      onToggleType(type);
    } else {
      setLocalVisibleTypes(prev => ({ ...prev, [type]: !prev[type] }));
    }
  };

  // Filter clusters by side
  const filteredClustersByType = useMemo(() => {
    const result = {};
    Object.entries(grenadeData).forEach(([type, clusters]) => {
      if (activeSide === 'all') {
        result[type] = clusters;
      } else {
        result[type] = clusters.filter(c => c.side?.toLowerCase() === activeSide);
      }
    });
    return result;
  }, [grenadeData, activeSide]);
  
  const visibleClusters = useMemo(() => {
    const result = [];
    Object.entries(filteredClustersByType).forEach(([type, clusters]) => {
      if (effectiveVisibleTypes[type]) {
        clusters.forEach(cluster => {
          result.push({ ...cluster, type });
        });
      }
    });
    return result;
  }, [filteredClustersByType, effectiveVisibleTypes]);
  
  if (loading) {
    return (
      <div className="grenade-intel-panel loading">
        <div className="loading-spinner" />
        <span>Analizando granadas...</span>
      </div>
    );
  }
  
  const hasData = visibleClusters.length > 0 || Object.values(summary).some(s => s?.thrown > 0);
  
  return (
    <div className="grenade-intel-panel">
      {/* Grenade Arsenal - Compact Summary with Filters */}
      <GrenadeArsenal 
        summary={summary}
        matchesAnalyzed={matchesAnalyzed}
        visibleTypes={effectiveVisibleTypes}
        onToggleType={handleToggle}
      />
      
      {/* Lineup Detail Panel */}
      <LineupDetailPanel 
        cluster={selectedCluster}
        type={selectedType}
        onClose={onClusterClose}
      />
      
      {/* Top Lineups Section */}
      <TopLineupsSection 
        grenadeData={filteredClustersByType}
        summary={summary}
        onLineupHover={setHoveredLineup}
        onLineupClick={(cluster, type) => {
          if (onClusterSelect) {
            onClusterSelect(cluster, type);
          }
        }}
      />
      
      {/* Empty state */}
      {!hasData && !loading && (
        <div className="empty-state">
          <GrenadeImage type="he" size={48} className="grayscale" />
          <p>No hay datos de granadas</p>
          <span className="hint">
            {activeSide === 'all' 
              ? 'Juega más partidas en este mapa para ver tus patrones de granada.'
              : `Sin datos para el lado ${activeSide.toUpperCase()}.`}
          </span>
        </div>
      )}
    </div>
  );
};

// ============================================
// GRENADE OVERLAY (for map) - Redesigned
// ============================================

export const GrenadeOverlay = ({ 
  mapName, 
  activeSide = 'all', 
  visibleTypes, 
  onClusterClick, 
  selectedCluster, 
  selectedType,
  highlightedLineup,
  currentLevel = 'upper',
  zThreshold = null
}) => {
  const { grenadeData, loading } = useGrenadeStats(mapName);
  
  const filteredClusters = useMemo(() => {
    const result = {};
    Object.entries(grenadeData).forEach(([type, clusters]) => {
      // 1. Filter by side
      let filtered = clusters;
      if (activeSide !== 'all') {
        filtered = filtered.filter(c => c.side?.toLowerCase() === activeSide);
      }
      
      // 2. Filter by level (Z coordinate) for multi-level maps
      if (zThreshold !== null) {
        filtered = filtered.filter(c => {
          const avgZ = c.avg_z;
          // If no Z data, show on upper by default
          if (avgZ === undefined || avgZ === null) {
            return currentLevel === 'upper';
          }
          // Filter based on map threshold
          if (currentLevel === 'upper') {
            return avgZ >= zThreshold;
          } else {
            return avgZ < zThreshold;
          }
        });
      }

      result[type] = filtered;
    });
    return result;
  }, [grenadeData, activeSide, currentLevel, zThreshold]);

  if (loading) return null;
  
  return (
    <div className="grenade-overlay-intel">
      {/* Trajectory visualization for selected cluster */}
      {selectedCluster && selectedType && (
        <TrajectoryVisualization cluster={selectedCluster} type={selectedType} />
      )}
      
      {/* Smart markers */}
      {Object.entries(filteredClusters).map(([type, clusters]) => {
        if (!visibleTypes[type]) return null;
        
        return clusters.map((cluster, idx) => (
          <GrenadeMarkerSmart
            key={`${type}-${idx}`}
            cluster={cluster}
            type={type}
            onClick={onClusterClick}
            isSelected={selectedCluster === cluster}
            isHighlighted={highlightedLineup === cluster}
          />
        ));
      })}
    </div>
  );
};

export default GrenadeMapTab;
