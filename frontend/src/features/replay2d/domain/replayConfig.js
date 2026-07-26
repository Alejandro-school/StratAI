export const MAP_CONFIGS = {
  de_dust2: { pos_x: -2476, pos_y: 3239, scale: 4.4 },
  de_mirage: { pos_x: -3230, pos_y: 1713, scale: 5 },
  de_inferno: { pos_x: -2087, pos_y: 3870, scale: 4.9 },
  de_ancient: { pos_x: -2953, pos_y: 2164, scale: 5 },
  de_anubis: { pos_x: -2796, pos_y: 3328, scale: 5.22 },
  de_nuke: { pos_x: -3453, pos_y: 2887, scale: 7 },
  de_overpass: { pos_x: -4831, pos_y: 1781, scale: 5.2 },
  de_vertigo: { pos_x: -3168, pos_y: 1762, scale: 4 },
  de_train: { pos_x: -2308, pos_y: 2078, scale: 4.082077 },
};

const layered = (map) => ({
  upper: `/maps/${map}_radar_psd.webp`,
  lower: `/maps/${map}_lower_radar_psd.webp`,
});

export const MAP_LEVELS = {
  de_nuke: layered("de_nuke"),
  de_train: layered("de_train"),
  de_vertigo: layered("de_vertigo"),
};

export const getMapConfig = (metadata) => {
  const fallback = MAP_CONFIGS[metadata?.map_name] || MAP_CONFIGS.de_mirage;
  const supplied = metadata?.map_config;
  if (!supplied?.scale) return fallback;
  return { ...fallback, ...supplied };
};

export const getMapSource = (mapName, level = "upper") => {
  const levels = MAP_LEVELS[mapName];
  return levels?.[level] || `/maps/${mapName || "de_mirage"}_radar_psd.webp`;
};

export const getMapPngFallback = (source) => source.replace(/\.webp$/i, ".png");

export const TEAM_COLORS = {
  CT: { solid: "#69c7ff", soft: "rgba(105, 199, 255, .18)" },
  T: { solid: "#ffbe55", soft: "rgba(255, 190, 85, .18)" },
};

export const UTILITY_COLORS = {
  smoke: "#aab4c3",
  flashbang: "#fff3af",
  he: "#ff655f",
  molotov: "#ff8738",
  incendiary: "#ff8738",
  inferno: "#ff8738",
  decoy: "#71d6a0",
};

export const DEFAULT_LAYERS = {
  names: true,
  fov: true,
  shots: true,
  trajectories: true,
  utility: true,
  deaths: true,
  annotations: true,
};

export const ANNOTATION_COLORS = ["#63d7ff", "#ffbe55", "#ff6b6b", "#85e19b"];
