const getSupportedMapIds = (supportedMaps) => new Set(
  (supportedMaps ?? []).map((map) => (typeof map === 'string' ? map : map?.id)).filter(Boolean)
);

export const selectInitialTacticalMap = (matches, supportedMaps) => {
  const supportedMapIds = getSupportedMapIds(supportedMaps);
  return (matches ?? []).find((match) => supportedMapIds.has(match?.map_name))?.map_name ?? null;
};

export const getTacticalMapBootstrap = ({
  currentMap,
  hasExplicitMap,
  isLoading,
  matches,
  supportedMaps,
}) => {
  if (hasExplicitMap) {
    return { targetMap: null, deferRouteWrite: false };
  }

  const preferredMap = selectInitialTacticalMap(matches, supportedMaps);
  const targetMap = preferredMap && preferredMap !== currentMap ? preferredMap : null;

  return {
    targetMap,
    deferRouteWrite: Boolean(isLoading || targetMap),
  };
};
