import React, { useEffect, useRef } from 'react';
import {
  Check,
  ChevronDown,
  Crosshair,
  FlaskConical,
  Footprints,
  MapPin,
  Radar,
} from 'lucide-react';

const LENSES = [
  { id: 'briefing', label: 'Briefing', icon: Radar },
  { id: 'positioning', label: 'Posicionamiento', icon: Footprints },
  { id: 'combat', label: 'Combate', icon: Crosshair },
  { id: 'utility', label: 'Utilidad', icon: FlaskConical },
];

const focusAt = (elements, index) => {
  const available = elements.filter(Boolean);
  if (!available.length) return;
  available[(index + available.length) % available.length]?.focus();
};

const TacticalToolbar = ({
  maps,
  currentMap,
  currentMapInfo,
  activeLens,
  activeSide,
  currentLevel,
  hasLevels,
  isMapMenuOpen,
  onMapMenuChange,
  onMapChange,
  onLensChange,
  onSideChange,
  onLevelChange,
}) => {
  const menuRef = useRef(null);
  const triggerRef = useRef(null);
  const menuItemsRef = useRef([]);
  const tabRefs = useRef([]);

  useEffect(() => {
    if (!isMapMenuOpen) return undefined;

    const handlePointerDown = (event) => {
      if (!menuRef.current?.contains(event.target)) onMapMenuChange(false);
    };
    const handleKeyDown = (event) => {
      if (event.key !== 'Escape') return;
      onMapMenuChange(false);
      triggerRef.current?.focus();
    };
    const focusFrame = window.requestAnimationFrame(() => {
      const activeIndex = Math.max(0, maps.findIndex(({ id }) => id === currentMap));
      focusAt(menuItemsRef.current, activeIndex);
    });

    document.addEventListener('pointerdown', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      window.cancelAnimationFrame(focusFrame);
      document.removeEventListener('pointerdown', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [currentMap, isMapMenuOpen, maps, onMapMenuChange]);

  const handleTabKeyDown = (event, index) => {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
    event.preventDefault();
    const nextIndex = event.key === 'Home'
      ? 0
      : event.key === 'End'
        ? LENSES.length - 1
        : index + (event.key === 'ArrowRight' ? 1 : -1);
    const normalizedIndex = (nextIndex + LENSES.length) % LENSES.length;
    onLensChange(LENSES[normalizedIndex].id);
    tabRefs.current[normalizedIndex]?.focus();
  };

  const handleMenuKeyDown = (event, index) => {
    if (!['ArrowUp', 'ArrowDown', 'Home', 'End'].includes(event.key)) return;
    event.preventDefault();
    const nextIndex = event.key === 'Home'
      ? 0
      : event.key === 'End'
        ? maps.length - 1
        : index + (event.key === 'ArrowDown' ? 1 : -1);
    focusAt(menuItemsRef.current, nextIndex);
  };

  return (
    <div className="tactical-toolbar" aria-label="Controles del mapa táctico">
      <div className="tactical-map-picker" ref={menuRef}>
        <button
          ref={triggerRef}
          type="button"
          className="tactical-map-trigger"
          onClick={() => onMapMenuChange(!isMapMenuOpen)}
          onKeyDown={(event) => {
            if (event.key === 'ArrowDown') {
              event.preventDefault();
              onMapMenuChange(true);
            }
          }}
          aria-haspopup="menu"
          aria-controls="tactical-map-menu"
          aria-expanded={isMapMenuOpen}
        >
          <MapPin size={17} aria-hidden="true" />
          <span>{currentMapInfo.name}</span>
          <ChevronDown
            size={15}
            aria-hidden="true"
            className={isMapMenuOpen ? 'is-rotated' : ''}
          />
        </button>

        {isMapMenuOpen ? (
          <div id="tactical-map-menu" className="tactical-map-menu" role="menu" aria-label="Seleccionar mapa">
            {maps.map((map, index) => (
              <button
                ref={(element) => { menuItemsRef.current[index] = element; }}
                key={map.id}
                type="button"
                role="menuitemradio"
                aria-checked={map.id === currentMap}
                className={map.id === currentMap ? 'is-active' : ''}
                onKeyDown={(event) => handleMenuKeyDown(event, index)}
                onClick={() => onMapChange(map.id)}
              >
                <span>{map.name}</span>
                {map.id === currentMap ? <Check size={14} aria-hidden="true" /> : null}
              </button>
            ))}
          </div>
        ) : null}
      </div>

      <div className="tactical-lens-tabs" role="tablist" aria-label="Vistas de análisis">
        {LENSES.map(({ id, label, icon: Icon }, index) => (
          <button
            ref={(element) => { tabRefs.current[index] = element; }}
            id={`tactical-tab-${id}`}
            key={id}
            type="button"
            role="tab"
            tabIndex={activeLens === id ? 0 : -1}
            aria-selected={activeLens === id}
            aria-controls="tactical-lens-panel"
            className={activeLens === id ? 'is-active' : ''}
            onKeyDown={(event) => handleTabKeyDown(event, index)}
            onClick={() => onLensChange(id)}
          >
            <Icon size={16} aria-hidden="true" />
            <span>{label}</span>
          </button>
        ))}
      </div>

      <div className="tactical-filter-cluster">
        <div className="tactical-side-filter" role="group" aria-label="Filtrar por bando">
          {[
            ['ct', 'CT', 'Defensa'],
            ['t', 'T', 'Ataque'],
          ].map(([side, code, role]) => (
            <button
              key={side}
              type="button"
              aria-label={`Analizar ${code} · ${role}`}
              aria-pressed={activeSide === side}
              className={`${side} ${activeSide === side ? 'is-active' : ''}`}
              onClick={() => onSideChange(side)}
            >
              <strong>{code}</strong>
              <span>{role}</span>
            </button>
          ))}
        </div>

        {hasLevels ? (
          <div className="tactical-level-filter" role="group" aria-label="Planta del mapa">
            {[
              ['upper', 'Superior'],
              ['lower', 'Inferior'],
            ].map(([level, label]) => (
              <button
                key={level}
                type="button"
                aria-pressed={currentLevel === level}
                className={currentLevel === level ? 'is-active' : ''}
                onClick={() => onLevelChange(level)}
              >
                {label}
              </button>
            ))}
          </div>
        ) : null}
      </div>
    </div>
  );
};

export default React.memo(TacticalToolbar);
