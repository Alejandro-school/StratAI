// frontend/src/components/Dashboard/LevelSelector.jsx
// Professional Multi-Level Map Selector with Mini-Preview
import React, { useState } from 'react';
import { Layers, ChevronUp, ChevronDown, MapPin } from 'lucide-react';
import '../../styles/Dashboard/levelSelector.css';

/**
 * Level configuration for multi-level maps
 */
const LEVEL_CONFIG = {
  de_nuke: {
    upper: {
      label: 'Upper',
      sublabel: 'Roof / Outside',
      icon: ChevronUp,
      color: '#60a5fa',
      zones: ['Lobby', 'Roof', 'Outside', 'Silo', 'Mini', 'Secret', 'Garage']
    },
    lower: {
      label: 'Lower',
      sublabel: 'Tunnels / Ramp',
      icon: ChevronDown,
      color: '#f59e0b',
      zones: ['Ramp', 'Tunnels', 'B Site', 'Decon', 'Vent']
    }
  },
  de_vertigo: {
    upper: {
      label: 'Upper',
      sublabel: 'A Site / CT',
      icon: ChevronUp,
      color: '#60a5fa',
      zones: ['A Site', 'CT Spawn', 'Ramp', 'Elevator']
    },
    lower: {
      label: 'Lower',
      sublabel: 'B Site / T Spawn',
      icon: ChevronDown,
      color: '#f59e0b',
      zones: ['B Site', 'T Spawn', 'Bottom Mid', 'Tunnels']
    }
  },
  de_train: {
    upper: {
      label: 'Upper',
      sublabel: 'Main Level',
      icon: ChevronUp,
      color: '#60a5fa',
      zones: ['A Site', 'B Site', 'Ivy', 'Connector']
    },
    lower: {
      label: 'Lower',
      sublabel: 'Pop Dog / Under',
      icon: ChevronDown,
      color: '#f59e0b',
      zones: ['Pop Dog', 'Under Trains', 'Z Connector']
    }
  }
};

/**
 * LevelSelector - Professional level toggle for multi-floor maps
 * 
 * @param {string} mapName - Current map (e.g., 'de_nuke')
 * @param {string} currentLevel - 'upper' or 'lower'
 * @param {function} onLevelChange - Callback when level changes
 * @param {Object} levelStats - Optional stats per level { upper: {...}, lower: {...} }
 */
const LevelSelector = ({ 
  mapName, 
  currentLevel, 
  onLevelChange, 
  levelStats = null,
  compact = false
}) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const config = LEVEL_CONFIG[mapName];
  
  if (!config) return null;
  
  const levels = ['upper', 'lower'];
  const currentConfig = config[currentLevel] || config.upper;
  const CurrentIcon = currentConfig.icon;
  
  // Compact mode - simple toggle buttons
  if (compact) {
    return (
      <div className="level-selector-compact">
        {levels.map(level => {
          const levelConfig = config[level];
          const LevelIcon = levelConfig.icon;
          const isActive = currentLevel === level;
          
          return (
            <button
              key={level}
              className={`level-btn-compact ${isActive ? 'active' : ''}`}
              onClick={() => onLevelChange(level)}
              title={`${levelConfig.label}: ${levelConfig.sublabel}`}
              style={{ '--level-color': levelConfig.color }}
            >
              <LevelIcon size={14} />
              <span>{levelConfig.label}</span>
            </button>
          );
        })}
      </div>
    );
  }
  
  // Full mode - expandable with zone preview
  return (
    <div className={`level-selector ${isExpanded ? 'expanded' : ''}`}>
      {/* Current Level Button */}
      <button 
        className="level-current"
        onClick={() => setIsExpanded(!isExpanded)}
        style={{ '--level-color': currentConfig.color }}
      >
        <div className="level-indicator">
          <Layers size={16} />
          <div className="level-stack">
            <div className={`stack-layer upper ${currentLevel === 'upper' ? 'active' : ''}`} />
            <div className={`stack-layer lower ${currentLevel === 'lower' ? 'active' : ''}`} />
          </div>
        </div>
        
        <div className="level-info">
          <span className="level-label">{currentConfig.label}</span>
          <span className="level-sublabel">{currentConfig.sublabel}</span>
        </div>
        
        <CurrentIcon 
          size={16} 
          className={`level-chevron ${isExpanded ? 'rotated' : ''}`}
        />
      </button>
      
      {/* Expanded Panel */}
      {isExpanded && (
        <div className="level-panel">
          {levels.map(level => {
            const levelConfig = config[level];
            const LevelIcon = levelConfig.icon;
            const isActive = currentLevel === level;
            const stats = levelStats?.[level];
            
            return (
              <button
                key={level}
                className={`level-option ${isActive ? 'active' : ''}`}
                onClick={() => {
                  onLevelChange(level);
                  setIsExpanded(false);
                }}
                style={{ '--level-color': levelConfig.color }}
              >
                <div className="option-header">
                  <LevelIcon size={18} className="option-icon" />
                  <div className="option-text">
                    <span className="option-label">{levelConfig.label}</span>
                    <span className="option-sublabel">{levelConfig.sublabel}</span>
                  </div>
                  {isActive && (
                    <div className="option-active-indicator" />
                  )}
                </div>
                
                {/* Zone Tags */}
                <div className="option-zones">
                  {levelConfig.zones.slice(0, 4).map((zone, idx) => (
                    <span key={idx} className="zone-tag">
                      <MapPin size={8} />
                      {zone}
                    </span>
                  ))}
                  {levelConfig.zones.length > 4 && (
                    <span className="zone-more">+{levelConfig.zones.length - 4}</span>
                  )}
                </div>
                
                {/* Optional Stats */}
                {stats && (
                  <div className="option-stats">
                    <span className="stat">
                      <span className="stat-value">{stats.kills || 0}</span>
                      <span className="stat-label">K</span>
                    </span>
                    <span className="stat-divider">/</span>
                    <span className="stat">
                      <span className="stat-value">{stats.deaths || 0}</span>
                      <span className="stat-label">D</span>
                    </span>
                  </div>
                )}
              </button>
            );
          })}
          
          {/* Visual hint */}
          <div className="level-hint">
            <Layers size={12} />
            <span>Datos filtrados por nivel Z</span>
          </div>
        </div>
      )}
    </div>
  );
};

/**
 * LevelIndicator - Minimal floating indicator showing current level
 * Placed on the map for quick reference
 */
export const LevelIndicator = ({ mapName, currentLevel }) => {
  const config = LEVEL_CONFIG[mapName]?.[currentLevel];
  if (!config) return null;
  
  const LevelIcon = config.icon;
  
  return (
    <div 
      className="level-floating-indicator"
      style={{ '--level-color': config.color }}
    >
      <LevelIcon size={14} />
      <span>{config.label}</span>
    </div>
  );
};

export default LevelSelector;
