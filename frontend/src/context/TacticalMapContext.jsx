import React, { createContext, useContext, useMemo, useReducer } from 'react';

const TacticalMapContext = createContext(null);

const initialState = {
  currentMap: 'de_dust2',
  currentLevel: 'upper',
  showMapDropdown: false,
  selectedCallout: null,
  activeSection: 'hotpoints',
  activeSide: 'all',
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
  zoomLevel: 1,
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
      return { ...state, currentLevel: action.payload };
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
      return { ...state, activeSide: action.payload };
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
    case 'SET_ZOOM':
      return { ...state, zoomLevel: action.payload };
    default:
      return state;
  }
};

export const TacticalMapProvider = ({ children }) => {
  const [state, dispatch] = useReducer(tacticalMapReducer, initialState);

  const value = useMemo(() => {
    const setCurrentMap = (mapName) => dispatch({ type: 'SET_MAP', payload: mapName });
    const setCurrentLevel = (level) => dispatch({ type: 'SET_LEVEL', payload: level });
    const setShowMapDropdown = (isOpen) => dispatch({ type: 'SET_DROPDOWN', payload: isOpen });
    const setActiveSection = (section) => dispatch({ type: 'SET_SECTION', payload: section });
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
    const setZoomLevel = (zoomLevel) => dispatch({ type: 'SET_ZOOM', payload: zoomLevel });

    return {
      state,
      dispatch,
      setCurrentMap,
      setCurrentLevel,
      setShowMapDropdown,
      setActiveSection,
      setActiveSide,
      setSelectedCallout,
      toggleSelectedCallout,
      setSelectedGrenade,
      clearSelectedGrenade,
      toggleGrenadeType,
      setMovementHeatmapIntensity,
      toggleMovementHeatmap,
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
