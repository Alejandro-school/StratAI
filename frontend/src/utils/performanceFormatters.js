const DASH = '—';

const toNumber = (value) => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
};

export const formatDecimal = (value, decimals = 1, suffix = '') => {
  const numericValue = toNumber(value);
  if (numericValue === null) return DASH;
  return `${numericValue.toFixed(decimals)}${suffix}`;
};

export const formatInteger = (value) => {
  const numericValue = toNumber(value);
  if (numericValue === null) return DASH;
  return numericValue.toLocaleString();
};

export const formatPercent = (value, decimals = 1) => formatDecimal(value, decimals, '%');

export const formatMilliseconds = (value) => {
  const numericValue = toNumber(value);
  if (numericValue === null || numericValue <= 0) return 'N/A';
  return `${numericValue.toFixed(0)} ms`;
};

export const formatDegrees = (value, decimals = 2) => {
  const numericValue = toNumber(value);
  if (numericValue === null || numericValue <= 0) return 'N/A';
  return `${numericValue.toFixed(decimals)}°`;
};

/**
 * Relative date formatter — "hace 2 días", "hace 1 semana", etc.
 * Falls back to the raw string if parsing fails.
 */
export const formatRelativeDate = (dateStr) => {
  if (!dateStr) return 'Sin fecha';

  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) return dateStr;

  const now = new Date();
  const diffMs = now - date;
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHr = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHr / 24);

  if (diffDay < 0) return dateStr;
  if (diffDay === 0) {
    if (diffHr < 1) return 'Ahora mismo';
    return `Hace ${diffHr}h`;
  }
  if (diffDay === 1) return 'Ayer';
  if (diffDay < 7) return `Hace ${diffDay} días`;
  if (diffDay < 30) {
    const weeks = Math.floor(diffDay / 7);
    return `Hace ${weeks} sem`;
  }
  if (diffDay < 365) {
    const months = Math.floor(diffDay / 30);
    return `Hace ${months} ${months === 1 ? 'mes' : 'meses'}`;
  }
  return dateStr;
};

export const formatMapName = (mapName) => {
  if (!mapName) return 'Mapa desconocido';

  const cleanedMap = String(mapName).replace(/^de_/i, '').toLowerCase();

  const mapNames = {
    dust2: 'Dust II',
    inferno: 'Inferno',
    mirage: 'Mirage',
    nuke: 'Nuke',
    ancient: 'Ancient',
    anubis: 'Anubis',
    vertigo: 'Vertigo',
    overpass: 'Overpass',
    train: 'Train',
    cache: 'Cache',
  };

  return mapNames[cleanedMap] || cleanedMap.charAt(0).toUpperCase() + cleanedMap.slice(1);
};

export const formatWeaponName = (weaponName) => {
  if (!weaponName) return 'Unknown';

  const normalized = String(weaponName).replace(/^weapon_/i, '').toLowerCase();

  const displayNames = {
    ak47: 'AK-47',
    m4a1: 'M4A4',
    m4a1_silencer: 'M4A1-S',
    awp: 'AWP',
    galilar: 'Galil AR',
    famas: 'FAMAS',
    ssg08: 'SSG 08',
    g3sg1: 'G3SG1',
    scar20: 'SCAR-20',
    aug: 'AUG',
    sg556: 'SG 553',
    glock: 'Glock-18',
    usp_silencer: 'USP-S',
    hkp2000: 'P2000',
    p250: 'P250',
    tec9: 'Tec-9',
    fiveseven: 'Five-SeveN',
    elite: 'Dual Berettas',
    deagle: 'Desert Eagle',
    revolver: 'R8 Revolver',
    cz75a: 'CZ75-Auto',
    mac10: 'MAC-10',
    mp9: 'MP9',
    mp7: 'MP7',
    ump45: 'UMP-45',
    p90: 'P90',
    bizon: 'PP-Bizon',
    nova: 'Nova',
    xm1014: 'XM1014',
    sawedoff: 'Sawed-Off',
    mag7: 'MAG-7',
    m249: 'M249',
    negev: 'Negev',
    hegrenade: 'HE Grenade',
    flashbang: 'Flashbang',
    smokegrenade: 'Smoke',
    molotov: 'Molotov',
    incgrenade: 'Incendiary',
    decoy: 'Decoy',
    knife: 'Knife',
    taser: 'Zeus x27',
  };

  return displayNames[normalized] || normalized.replace(/_/g, ' ').replace(/\b\w/g, (char) => char.toUpperCase());
};

export const RECHARTS_TOOLTIP_STYLE = {
  background: 'rgba(15, 23, 42, 0.92)',
  backdropFilter: 'blur(12px)',
  border: '1px solid rgba(148, 163, 184, 0.22)',
  borderRadius: 10,
  padding: '10px 14px',
  boxShadow: '0 8px 24px rgba(0, 0, 0, 0.45)',
  color: '#e2e8f0',
  fontSize: 12,
  fontWeight: 600,
};

export const TOOLTIP_LABEL_STYLE = {
  color: '#94a3b8',
  fontSize: 11,
  fontWeight: 600,
  marginBottom: 4,
};

export const TOOLTIP_ITEM_STYLE = {
  color: '#e2e8f0',
  fontSize: 12,
  fontWeight: 600,
};

/* Reusable chart axis / grid constants */
export const CHART_AXIS_TICK = { fill: '#64748b', fontSize: 11 };
export const CHART_GRID_STROKE = 'rgba(148, 163, 184, 0.06)';
export const CHART_CURSOR_STYLE = { stroke: 'rgba(139, 92, 246, 0.3)', strokeWidth: 1 };
export const CHART_CURSOR_BAR = { fill: 'rgba(139, 92, 246, 0.08)' };

/**
 * Compute a delta trend (current value vs average of last N entries).
 * Returns null if insufficient data; otherwise a numeric delta.
 */
export const computeTrend = (matchHistory = [], field, currentValue, sampleSize = 5) => {
  if (!matchHistory || matchHistory.length < 2) return null;

  const recent = matchHistory.slice(0, sampleSize);
  const sum = recent.reduce((acc, m) => acc + Number(m[field] || 0), 0);
  const avg = sum / recent.length;

  const cv = Number(currentValue);
  if (!Number.isFinite(cv) || !Number.isFinite(avg) || avg === 0) return null;

  return cv - avg;
};
