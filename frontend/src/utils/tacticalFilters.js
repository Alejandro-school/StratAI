import { guessCalloutLevel } from './mapLevelFallbacks';

export const filterCalloutsBySide = (callouts, activeSide) => {
  if (activeSide === 'all') return callouts;

  return callouts
    .map((callout) => {
      const sideSplit = callout.ct_t_split || {};
      const sideKills = activeSide === 'ct' ? sideSplit.ct_kills || 0 : sideSplit.t_kills || 0;
      const sideDeaths = activeSide === 'ct' ? sideSplit.ct_deaths || 0 : sideSplit.t_deaths || 0;
      const total = sideKills + sideDeaths;
      if (total === 0) return null;

      const kd = sideDeaths > 0 ? (sideKills / sideDeaths).toFixed(2) : sideKills.toFixed(2);
      const winRate = Math.round((sideKills / total) * 100);
      const rating = winRate >= 55 ? 'good' : winRate <= 45 ? 'bad' : 'neutral';

      return {
        ...callout,
        kills: sideKills,
        deaths: sideDeaths,
        kd,
        win_rate: winRate,
        rating,
        sample_size: total,
      };
    })
    .filter(Boolean);
};

export const hasValidPosition = (item) =>
  item?.position && item.position.x !== undefined && item.position.y !== undefined;

export const filterItemsByLevel = ({
  items,
  hasLevels,
  zThreshold,
  currentLevel,
  mapName,
  fallbackNameResolver,
  missingZDefaultsToUpper = true,
}) => {
  if (!hasLevels || zThreshold === undefined || zThreshold === null) {
    return items;
  }

  return items.filter((item) => {
    const avgZ = item?.avg_z;

    if (avgZ !== null && avgZ !== undefined) {
      const isUpper = avgZ >= zThreshold;
      return currentLevel === 'upper' ? isUpper : !isUpper;
    }

    const resolver = fallbackNameResolver || ((entry) => entry?.name || '');
    const fallbackName = resolver(item);

    if (!fallbackName) {
      return missingZDefaultsToUpper ? currentLevel === 'upper' : false;
    }

    const guessedLevel = guessCalloutLevel(fallbackName, mapName);
    return guessedLevel === currentLevel;
  });
};

export const filterCalloutsByLevel = ({ callouts, hasLevels, zThreshold, currentLevel, currentMap }) => {
  const withPosition = callouts.filter(hasValidPosition);

  return filterItemsByLevel({
    items: withPosition,
    hasLevels,
    zThreshold,
    currentLevel,
    mapName: currentMap,
    fallbackNameResolver: (entry) => entry?.name,
    missingZDefaultsToUpper: true,
  });
};

export const filterGrenadeClusters = ({
  grenadeData,
  activeSide,
  visibleTypes,
  hasLevels,
  zThreshold,
  currentLevel,
  mapName,
}) => {
  const result = {};

  Object.entries(grenadeData || {}).forEach(([type, clusters]) => {
    if (visibleTypes && !visibleTypes[type]) {
      result[type] = [];
      return;
    }

    const bySide = activeSide === 'all'
      ? clusters || []
      : (clusters || []).filter((cluster) => cluster.side?.toLowerCase() === activeSide);

    result[type] = filterItemsByLevel({
      items: bySide,
      hasLevels,
      zThreshold,
      currentLevel,
      mapName,
      fallbackNameResolver: (entry) => entry?.areas?.[0] || entry?.lineup_name,
      missingZDefaultsToUpper: true,
    });
  });

  return result;
};
