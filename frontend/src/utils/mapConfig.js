const MAP_CATALOG = [
  { id: 'de_dust2', name: 'Dust II', radarImage: 'de_dust2_radar_psd.png', coverImage: '/images/maps/de_dust2.png', color: 'var(--color-primary-400)', availableInTacticalMap: true },
  { id: 'de_mirage', name: 'Mirage', radarImage: 'de_mirage_radar_psd.png', coverImage: '/images/maps/de_mirage.png', color: 'var(--color-primary-500)', availableInTacticalMap: true },
  { id: 'de_inferno', name: 'Inferno', radarImage: 'de_inferno_radar_psd.png', coverImage: '/images/maps/de_inferno.png', color: 'var(--color-secondary-500)', availableInTacticalMap: true },
  {
    id: 'de_nuke',
    name: 'Nuke',
    radarImage: 'de_nuke_radar_psd.png',
    coverImage: '/images/maps/de_nuke.png',
    color: 'var(--color-primary-700)',
    availableInTacticalMap: true,
    levels: { upper: 'de_nuke_radar_psd.png', lower: 'de_nuke_lower_radar_psd.png' },
    zThreshold: -500,
  },
  { id: 'de_overpass', name: 'Overpass', radarImage: 'de_overpass_radar_psd.png', coverImage: '/images/maps/de_overpass.png', color: 'var(--color-primary-300)', availableInTacticalMap: true },
  { id: 'de_train', name: 'Train', radarImage: 'de_train_radar_psd.png', coverImage: '/images/maps/de_train.png', color: 'var(--color-secondary-600)', availableInTacticalMap: true },
  { id: 'de_vertigo', name: 'Vertigo', radarImage: 'de_vertigo_radar_psd.png', coverImage: '/images/maps/de_vertigo.png', color: 'var(--color-primary-200)', availableInTacticalMap: false },
  { id: 'de_anubis', name: 'Anubis', radarImage: 'de_anubis_radar_psd.png', coverImage: '/images/maps/de_anubis.png', color: 'var(--color-primary-600)', availableInTacticalMap: false },
  { id: 'de_ancient', name: 'Ancient', radarImage: 'de_ancient_radar_psd.png', coverImage: '/images/maps/de_ancient.png', color: 'var(--color-primary-800)', availableInTacticalMap: true },
];

const FALLBACK_MAP_ID = 'de_dust2';

export const TACTICAL_MAPS = MAP_CATALOG
  .filter((map) => map.availableInTacticalMap)
  .map(({ id, name, radarImage, levels, zThreshold }) => ({
    id,
    name,
    img: radarImage,
    available: true,
    ...(levels ? { levels } : {}),
    ...(typeof zThreshold === 'number' ? { zThreshold } : {}),
  }));

export const MAP_FILTER_OPTIONS = MAP_CATALOG.map(({ id, name, color }) => ({
  id,
  name,
  color,
}));

export const getMapImage = (mapName) => {
  const match = MAP_CATALOG.find((map) => map.id === mapName);
  const fallback = MAP_CATALOG.find((map) => map.id === FALLBACK_MAP_ID);
  return match?.coverImage || fallback?.coverImage || '/images/maps/de_dust2.png';
};
