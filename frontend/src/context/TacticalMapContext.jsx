import React, { createContext, useContext, useMemo, useReducer } from 'react';
import { TACTICAL_MAPS } from '../utils/mapConfig';

const TacticalMapContext = createContext(null);

const VALID_LENSES = new Set(['briefing', 'positioning', 'combat', 'utility']);
const VALID_COMBAT_METRICS = new Set(['volume', 'efficiency', 'impact', 'risk']);
const LEGACY_LENSES = {
  overview: { activeSection: 'briefing' },
  presence: { activeSection: 'positioning' },
  duels: { activeSection: 'combat', combatMetric: 'volume' },
  impact: { activeSection: 'combat', combatMetric: 'impact' },
};
const VALID_SIDES = new Set(['ct', 't']);
const VALID_LEVELS = new Set(['upper', 'lower']);
const VALID_MAPS = new Set(TACTICAL_MAPS.map(({ id }) => id));

const baseState = {
  currentMap: 'de_dust2',
  currentLevel: 'upper',
  showMapDropdown: false,
  selectedCallout: null,
  activeSection: 'briefing',
  combatMetric: 'volume',
  activeSide: 'ct',
  grenadeVisibleTypes: {
    smoke: true,
    flash: true,
    he: true,
    molotov: true,
  },
  selectedGrenadeCluster: null,
  selectedGrenadeType: null,
  movementHeatmapIntensity: 70,
  showMovementHeatmap: true,
  showMovementRoutes: false,
  zoomLevel: 1,
};

export const parseTacticalRouteState = (source) => {
  const params = source instanceof URLSearchParams
    ? source
    : new URLSearchParams(source);
  const map = params.get('map');
  const lens = params.get('lens');
  const legacyLens = LEGACY_LENSES[lens];
  const activeSection = legacyLens?.activeSection
    ?? (VALID_LENSES.has(lens) ? lens : baseState.activeSection);
  const requestedMetric = params.get('metric');
  const combatMetric = legacyLens?.combatMetric
    ?? (VALID_COMBAT_METRICS.has(requestedMetric) ? requestedMetric : baseState.combatMetric);
  const requestedSide = params.get('side');
  const side = requestedSide === 'all' ? 'ct' : requestedSide;
  const level = params.get('level');

  return {
    currentMap: VALID_MAPS.has(map) ? map : baseState.currentMap,
    activeSection,
    combatMetric,
    activeSide: VALID_SIDES.has(side) ? side : baseState.activeSide,
    currentLevel: VALID_LEVELS.has(level) ? level : baseState.currentLevel,
    selectedCallout: params.get('zone') || null,
  };
};

const createInitialState = () => {
  if (typeof window === 'undefined') return baseState;

  return {
    ...baseState,
    ...parseTacticalRouteState(window.location.search),
  };
};

const tacticalMapReducer = (state, action) => {
  switch (action.type) {
    case 'SET_MAP':
      return {
        ...state,
        currentMap: action.payload,
        currentLevel: 'upper',
        showMapDropdown: false,
        selectedCallout: null,
        selectedGrenadeCluster: null,
        selectedGrenadeType: null,
      };
    case 'SET_LEVEL':
      return {
        ...state,
        currentLevel: action.payload,
        selectedCallout: null,
        selectedGrenadeCluster: null,
        selectedGrenadeType: null,
      };
    case 'SET_DROPDOWN':
      return { ...state, showMapDropdown: action.payload };
    case 'SET_SECTION':
      return {
        ...state,
        activeSection: action.payload,
        selectedCallout: null,
        selectedGrenadeCluster: null,
        selectedGrenadeType: null,
      };
    case 'SET_SIDE':
      return {
        ...state,
        activeSide: action.payload,
        selectedCallout: null,
        selectedGrenadeCluster: null,
        selectedGrenadeType: null,
      };
    case 'SET_COMBAT_METRIC':
      return {
        ...state,
        combatMetric: VALID_COMBAT_METRICS.has(action.payload)
          ? action.payload
          : state.combatMetric,
        selectedCallout: null,
      };
    case 'SET_SELECTED_CALLOUT':
      return { ...state, selectedCallout: action.payload };
    case 'TOGGLE_SELECTED_CALLOUT':
      return {
        ...state,
        selectedCallout: state.selectedCallout === action.payload ? null : action.payload,
      };
    case 'SET_SELECTED_GRENADE':
      return {
        ...state,
        selectedGrenadeCluster: action.payload.cluster,
        selectedGrenadeType: action.payload.type,
      };
    case 'CLEAR_SELECTED_GRENADE':
      return { ...state, selectedGrenadeCluster: null, selectedGrenadeType: null };
    case 'TOGGLE_GRENADE_TYPE':
      return {
        ...state,
        grenadeVisibleTypes: {
          ...state.grenadeVisibleTypes,
          [action.payload]: !state.grenadeVisibleTypes[action.payload],
        },
      };
    case 'SET_HEATMAP_INTENSITY':
      return { ...state, movementHeatmapIntensity: action.payload };
    case 'TOGGLE_HEATMAP':
      return { ...state, showMovementHeatmap: !state.showMovementHeatmap };
    case 'TOGGLE_ROUTES':
      return { ...state, showMovementRoutes: !state.showMovementRoutes };
    case 'SET_ZOOM':
      return { ...state, zoomLevel: action.payload };
    case 'SYNC_ROUTE_STATE': {
      const routeState = action.payload;
      const isSame = Object.entries(routeState).every(
        ([key, value]) => state[key] === value
      );
      if (isSame) return state;

      return {
        ...state,
        ...routeState,
        showMapDropdown: false,
        selectedGrenadeCluster: null,
        selectedGrenadeType: null,
        zoomLevel: 1,
      };
    }
    default:
      return state;
  }
};

export const TacticalMapProvider = ({ children }) => {
  const [state, dispatch] = useReducer(tacticalMapReducer, undefined, createInitialState);

  const value = useMemo(() => {
    const setCurrentMap = (mapName) => dispatch({ type: 'SET_MAP', payload: mapName });
    const setCurrentLevel = (level) => dispatch({ type: 'SET_LEVEL', payload: level });
    const setShowMapDropdown = (isOpen) => dispatch({ type: 'SET_DROPDOWN', payload: isOpen });
    const setActiveSection = (section) => dispatch({ type: 'SET_SECTION', payload: section });
    const setCombatMetric = (metric) => dispatch({ type: 'SET_COMBAT_METRIC', payload: metric });
    const setActiveSide = (side) => dispatch({ type: 'SET_SIDE', payload: side });
    const setSelectedCallout = (callout) => dispatch({ type: 'SET_SELECTED_CALLOUT', payload: callout });
    const toggleSelectedCallout = (callout) => dispatch({ type: 'TOGGLE_SELECTED_CALLOUT', payload: callout });
    const setSelectedGrenade = (cluster, type) =>
      dispatch({ type: 'SET_SELECTED_GRENADE', payload: { cluster, type } });
    const clearSelectedGrenade = () => dispatch({ type: 'CLEAR_SELECTED_GRENADE' });
    const toggleGrenadeType = (type) => dispatch({ type: 'TOGGLE_GRENADE_TYPE', payload: type });
    const setMovementHeatmapIntensity = (intensity) =>
      dispatch({ type: 'SET_HEATMAP_INTENSITY', payload: intensity });
    const toggleMovementHeatmap = () => dispatch({ type: 'TOGGLE_HEATMAP' });
    const toggleMovementRoutes = () => dispatch({ type: 'TOGGLE_ROUTES' });
    const setZoomLevel = (zoomLevel) => dispatch({ type: 'SET_ZOOM', payload: zoomLevel });

    return {
      state,
      dispatch,
      setCurrentMap,
      setCurrentLevel,
      setShowMapDropdown,
      setActiveSection,
      setCombatMetric,
      setActiveSide,
      setSelectedCallout,
      toggleSelectedCallout,
      setSelectedGrenade,
      clearSelectedGrenade,
      toggleGrenadeType,
      setMovementHeatmapIntensity,
      toggleMovementHeatmap,
      toggleMovementRoutes,
      setZoomLevel,
    };
  }, [state]);

  return <TacticalMapContext.Provider value={value}>{children}</TacticalMapContext.Provider>;
};

export const useTacticalMapState = () => {
  const context = useContext(TacticalMapContext);
  if (!context) {
    throw new Error('useTacticalMapState must be used within TacticalMapProvider');
  }
  return context;
};
