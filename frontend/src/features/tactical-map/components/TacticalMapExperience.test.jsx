import React from 'react';
import { render, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import TacticalMapExperience from './TacticalMapExperience';

const mocks = vi.hoisted(() => ({
  useAuth: vi.fn(),
  useQuery: vi.fn(),
  useTacticalExperienceData: vi.fn(),
  useTacticalMapState: vi.fn(),
  useTacticalRouteSync: vi.fn(),
}));

vi.mock('@tanstack/react-query', async () => ({
  ...await vi.importActual('@tanstack/react-query'),
  useQuery: mocks.useQuery,
}));
vi.mock('../../../auth/useAuth', () => ({ useAuth: mocks.useAuth }));
vi.mock('../../../context/TacticalMapContext', () => ({
  useTacticalMapState: mocks.useTacticalMapState,
}));
vi.mock('../hooks/useTacticalExperienceData', () => ({
  useTacticalExperienceData: mocks.useTacticalExperienceData,
}));
vi.mock('../hooks/useTacticalRouteSync', () => ({
  useTacticalRouteSync: mocks.useTacticalRouteSync,
}));
vi.mock('../../../components/Layout/NavigationFrame', () => ({
  default: ({ children }) => <div>{children}</div>,
}));
vi.mock('./TacticalMapStage', () => ({ default: () => <div /> }));
vi.mock('./TacticalPresencePanel', () => ({ default: () => <div /> }));
vi.mock('./TacticalToolbar', () => ({ default: () => <div /> }));
vi.mock('./TacticalUtilityPanel', () => ({ default: () => <div /> }));
vi.mock('./TacticalZonePanel', () => ({ default: () => <div /> }));
vi.mock('../../../components/TacticalMap/TacticalInsightsPanel', () => ({ default: () => <div /> }));
vi.mock('../../../components/TacticalMap/TacticalIntro', () => ({ default: () => null }));

const createMapContext = (currentMap) => ({
  state: {
    currentMap,
    currentLevel: 'upper',
    showMapDropdown: false,
    selectedCallout: null,
    activeSection: 'briefing',
    combatMetric: 'volume',
    activeSide: 'ct',
    grenadeVisibleTypes: { smoke: true, flash: true, he: true, molotov: true },
    selectedGrenadeCluster: null,
    selectedGrenadeType: null,
    movementHeatmapIntensity: 70,
    showMovementHeatmap: true,
    showMovementRoutes: false,
    zoomLevel: 1,
  },
  dispatch: vi.fn(),
  setCurrentMap: vi.fn(),
  setCurrentLevel: vi.fn(),
  setShowMapDropdown: vi.fn(),
  setActiveSection: vi.fn(),
  setCombatMetric: vi.fn(),
  setActiveSide: vi.fn(),
  setSelectedCallout: vi.fn(),
  toggleSelectedCallout: vi.fn(),
  setSelectedGrenade: vi.fn(),
  clearSelectedGrenade: vi.fn(),
  toggleGrenadeType: vi.fn(),
  setMovementHeatmapIntensity: vi.fn(),
  toggleMovementHeatmap: vi.fn(),
  toggleMovementRoutes: vi.fn(),
  setZoomLevel: vi.fn(),
});

const tacticalData = {
  mapInfo: { id: 'de_dust2', name: 'Dust II', img: 'radar.webp' },
  hasLevels: false,
  mapImage: 'radar.webp',
  calloutQuery: { matchesAnalyzed: 0 },
  movementQuery: { heatmapGrid: [], flowLines: [], matchesAnalyzed: 0 },
  grenadeQuery: { grenadeData: {}, loading: false, matchesAnalyzed: 0 },
  movementMetrics: {},
  sideDistribution: {},
  tacticalModel: { callouts: [], baselineWinRate: 50 },
  presentation: { signals: [], globalConfidence: 0, insights: [] },
  selectedCalloutData: null,
  matchesAnalyzed: 0,
  loading: false,
  error: null,
  retry: vi.fn(),
};

describe('TacticalMapExperience map bootstrap', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.useAuth.mockReturnValue({ user: { steam_id: '76561198116485358' } });
    mocks.useTacticalExperienceData.mockReturnValue(tacticalData);
  });

  it('adopts Nuke when it is the first supported map in the shared match list', async () => {
    const mapContext = createMapContext('de_dust2');
    mocks.useTacticalMapState.mockReturnValue(mapContext);
    mocks.useQuery.mockReturnValue({
      data: [
        { match_id: 'cache-match', map_name: 'de_cache' },
        { match_id: 'nuke-match', map_name: 'de_nuke' },
      ],
      isPending: false,
    });

    render(
      <MemoryRouter initialEntries={['/tactical-map']}>
        <TacticalMapExperience />
      </MemoryRouter>,
    );

    await waitFor(() => expect(mapContext.setCurrentMap).toHaveBeenCalledWith('de_nuke'));
    expect(mocks.useTacticalExperienceData).toHaveBeenCalledWith(
      expect.objectContaining({ currentMap: 'de_dust2', enabled: false }),
    );
    expect(mocks.useTacticalRouteSync).toHaveBeenCalledWith(
      expect.objectContaining({ deferRouteWrite: true }),
    );
  });

  it('keeps an explicit map deep link and skips the match-list request', () => {
    const mapContext = createMapContext('de_mirage');
    mocks.useTacticalMapState.mockReturnValue(mapContext);
    mocks.useQuery.mockReturnValue({ data: undefined, isPending: true });

    render(
      <MemoryRouter initialEntries={['/tactical-map?map=de_mirage']}>
        <TacticalMapExperience />
      </MemoryRouter>,
    );

    expect(mapContext.setCurrentMap).not.toHaveBeenCalled();
    expect(mocks.useQuery).toHaveBeenCalledWith(expect.objectContaining({ enabled: false }));
    expect(mocks.useTacticalExperienceData).toHaveBeenCalledWith(
      expect.objectContaining({ currentMap: 'de_mirage', enabled: true }),
    );
    expect(mocks.useTacticalRouteSync).toHaveBeenCalledWith(
      expect.objectContaining({ deferRouteWrite: false }),
    );
  });
});
