import { useCallback, useMemo } from 'react';
import { TACTICAL_MAPS } from '../../../utils/mapConfig';
import {
  filterCalloutsByLevel,
  filterCalloutsBySide,
  filterItemsByLevel,
} from '../../../utils/tacticalFilters';
import { buildTacticalInsights } from '../domain/tacticalInsights';
import {
  buildTacticalPresentation,
  getMovementMetricsForSide,
  getMovementSideDistribution,
} from '../domain/tacticalViewModel';
import { useCalloutStats, useGrenadeStats, useMovementStats } from './useTacticalStats';

const CALLOUT_LENSES = new Set(['briefing', 'combat']);
const MOVEMENT_LENSES = new Set(['briefing', 'positioning']);

export const useTacticalExperienceData = ({
  currentMap,
  currentLevel,
  activeSection,
  activeSide,
  selectedCallout,
  enabled = true,
}) => {
  const calloutQuery = useCalloutStats(currentMap, {
    enabled: enabled && CALLOUT_LENSES.has(activeSection),
  });
  const movementQuery = useMovementStats(currentMap, {
    enabled: enabled && MOVEMENT_LENSES.has(activeSection),
  });
  const grenadeQuery = useGrenadeStats(currentMap, {
    enabled: enabled && activeSection === 'utility',
  });

  const mapInfo = TACTICAL_MAPS.find(({ id }) => id === currentMap) ?? TACTICAL_MAPS[0];
  const hasLevels = Boolean(mapInfo.levels && Number.isFinite(mapInfo.zThreshold));
  const mapImage = hasLevels ? mapInfo.levels[currentLevel] ?? mapInfo.img : mapInfo.img;

  const calloutsForView = useMemo(() => {
    const bySide = filterCalloutsBySide(calloutQuery.sortedCallouts, activeSide);
    return filterCalloutsByLevel({
      callouts: bySide,
      hasLevels,
      zThreshold: mapInfo.zThreshold,
      currentLevel,
      currentMap,
    });
  }, [activeSide, calloutQuery.sortedCallouts, currentLevel, currentMap, hasLevels, mapInfo.zThreshold]);

  const movementMetricsForLevel = useMemo(() => {
    if (!hasLevels) return movementQuery.metrics;
    const positions = filterItemsByLevel({
      items: movementQuery.metrics.top_positions ?? [],
      hasLevels,
      zThreshold: mapInfo.zThreshold,
      currentLevel,
      mapName: currentMap,
      missingZDefaultsToUpper: false,
    });
    const totalSamples = positions.reduce((sum, position) => sum + Number(position.sample_count ?? 0), 0);
    return {
      ...movementQuery.metrics,
      total_samples: totalSamples,
      top_positions: positions.map((position) => ({
        ...position,
        time_percent: totalSamples > 0 ? Number(position.sample_count ?? 0) / totalSamples * 100 : 0,
      })),
    };
  }, [currentLevel, currentMap, hasLevels, mapInfo.zThreshold, movementQuery.metrics]);

  const movementMetrics = useMemo(
    () => getMovementMetricsForSide(movementMetricsForLevel, activeSide),
    [activeSide, movementMetricsForLevel]
  );
  const sideDistribution = useMemo(
    () => getMovementSideDistribution(movementMetricsForLevel),
    [movementMetricsForLevel]
  );
  const tacticalModel = useMemo(() => buildTacticalInsights({
    callouts: calloutsForView,
    matchesAnalyzed: calloutQuery.matchesAnalyzed,
    movementMatchesAnalyzed: movementQuery.matchesAnalyzed,
    movementMetrics,
  }), [calloutQuery.matchesAnalyzed, calloutsForView, movementMetrics, movementQuery.matchesAnalyzed]);
  const presentation = useMemo(() => buildTacticalPresentation({
    tacticalInsights: tacticalModel,
    movementMetrics,
  }), [movementMetrics, tacticalModel]);
  const selectedCalloutData = useMemo(
    () => tacticalModel.callouts.find(({ name }) => name === selectedCallout) ?? null,
    [selectedCallout, tacticalModel.callouts]
  );

  const matchesAnalyzed = activeSection === 'positioning'
    ? movementQuery.matchesAnalyzed
    : activeSection === 'utility'
      ? grenadeQuery.matchesAnalyzed
      : activeSection === 'briefing'
        ? Math.max(calloutQuery.matchesAnalyzed, movementQuery.matchesAnalyzed)
        : calloutQuery.matchesAnalyzed;
  const loading = activeSection === 'briefing'
    ? calloutQuery.loading || movementQuery.loading
    : activeSection === 'positioning'
      ? movementQuery.loading
      : activeSection === 'utility'
        ? grenadeQuery.loading
        : calloutQuery.loading;
  const error = activeSection === 'briefing'
    ? calloutQuery.error || movementQuery.error
    : activeSection === 'positioning'
      ? movementQuery.error
      : activeSection === 'utility'
        ? grenadeQuery.error
        : calloutQuery.error;

  const retry = useCallback(() => {
    if (activeSection === 'briefing') {
      calloutQuery.refetch();
      movementQuery.refetch();
    } else if (activeSection === 'positioning') {
      movementQuery.refetch();
    } else if (activeSection === 'utility') {
      grenadeQuery.refetch();
    } else {
      calloutQuery.refetch();
    }
  }, [activeSection, calloutQuery, grenadeQuery, movementQuery]);

  return {
    mapInfo,
    hasLevels,
    mapImage,
    calloutQuery,
    movementQuery,
    grenadeQuery,
    movementMetrics,
    sideDistribution,
    tacticalModel,
    presentation,
    selectedCalloutData,
    matchesAnalyzed,
    loading,
    error,
    retry,
  };
};
