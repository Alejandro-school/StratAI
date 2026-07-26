const EQUIPMENT_ROOT = "/images/cs2/equipment";

export const EQUIPMENT_CATALOG = {
  ak47: { label: "AK-47", slot: "primary", aliases: ["AK47"] },
  m4a1: { label: "M4A4", slot: "primary", aliases: ["M4A1"] },
  m4a1_silencer: { label: "M4A1-S", slot: "primary", aliases: ["M4A1S"] },
  awp: { label: "AWP", slot: "primary" },
  aug: { label: "AUG", slot: "primary" },
  famas: { label: "FAMAS", slot: "primary" },
  galilar: { label: "Galil AR", slot: "primary", aliases: ["Galil"] },
  ssg08: { label: "SSG 08", slot: "primary", aliases: ["Scout"] },
  sg556: { label: "SG 553", slot: "primary", aliases: ["SG556"] },
  scar20: { label: "SCAR-20", slot: "primary" },
  g3sg1: { label: "G3SG1", slot: "primary" },
  mac10: { label: "MAC-10", slot: "primary" },
  mp9: { label: "MP9", slot: "primary" },
  mp7: { label: "MP7", slot: "primary" },
  ump45: { label: "UMP-45", slot: "primary" },
  bizon: { label: "PP-Bizon", slot: "primary", aliases: ["Bizon"] },
  p90: { label: "P90", slot: "primary" },
  mp5sd: { label: "MP5-SD", slot: "primary" },
  nova: { label: "Nova", slot: "primary" },
  xm1014: { label: "XM1014", slot: "primary" },
  sawedoff: { label: "Sawed-Off", slot: "primary" },
  mag7: { label: "MAG-7", slot: "primary" },
  m249: { label: "M249", slot: "primary" },
  negev: { label: "Negev", slot: "primary" },
  deagle: { label: "Desert Eagle", slot: "pistol", aliases: ["Deagle"] },
  usp_silencer: { label: "USP-S", slot: "pistol", aliases: ["USP"] },
  glock: { label: "Glock-18", slot: "pistol", aliases: ["Glock"] },
  hkp2000: { label: "P2000", slot: "pistol", aliases: ["HKP2000"] },
  p250: { label: "P250", slot: "pistol" },
  fiveseven: { label: "Five-SeveN", slot: "pistol", aliases: ["Five Seven"] },
  tec9: { label: "Tec-9", slot: "pistol" },
  cz75a: { label: "CZ75-Auto", slot: "pistol", aliases: ["CZ75"] },
  elite: { label: "Dual Berettas", slot: "pistol", aliases: ["Dual Elites"] },
  revolver: { label: "R8 Revolver", slot: "pistol", aliases: ["R8"] },
  hegrenade: { label: "HE Grenade", slot: "utility", aliases: ["HEGrenade", "HE"] },
  flashbang: { label: "Flashbang", slot: "utility", aliases: ["Flash"] },
  smokegrenade: { label: "Smoke Grenade", slot: "utility", aliases: ["SmokeGrenade", "Smoke"] },
  molotov: { label: "Molotov", slot: "utility" },
  incgrenade: { label: "Incendiary Grenade", slot: "utility", aliases: ["IncendiaryGrenade", "Incendiary"] },
  decoy: { label: "Decoy Grenade", slot: "utility", aliases: ["DecoyGrenade", "Decoy"] },
  taser: { label: "Zeus x27", slot: "utility", aliases: ["Zeus"] },
  knife: { label: "Knife", slot: "knife" },
  c4: { label: "C4", slot: "objective", aliases: ["Bomb"] },
  defuser: { label: "Defuse Kit", slot: "equipment", aliases: ["Kit"] },
  kevlar: { label: "Kevlar", slot: "equipment", aliases: ["Armor"] },
  helmet: { label: "Helmet", slot: "equipment" },
};

const normalizeEquipmentName = (value) => String(value || "")
  .trim()
  .toLowerCase()
  .replace(/[^a-z0-9]+/g, "");

const EQUIPMENT_ALIASES = new Map();
Object.entries(EQUIPMENT_CATALOG).forEach(([id, equipment]) => {
  [id, equipment.label, ...(equipment.aliases || [])].forEach((alias) => {
    EQUIPMENT_ALIASES.set(normalizeEquipmentName(alias), id);
  });
});

export const equipmentIdFor = (equipment) => EQUIPMENT_ALIASES.get(normalizeEquipmentName(equipment)) || null;

export const equipmentLabel = (equipment) => {
  const id = EQUIPMENT_CATALOG[equipment] ? equipment : equipmentIdFor(equipment);
  return id ? EQUIPMENT_CATALOG[id].label : String(equipment || "");
};

export const equipmentIconPath = (equipment) => {
  const id = EQUIPMENT_CATALOG[equipment] ? equipment : equipmentIdFor(equipment);
  return id ? `${EQUIPMENT_ROOT}/${id}.svg` : null;
};

export const weaponIconPath = equipmentIconPath;

const normalizedInventory = (player) => {
  const items = (player?.weapons || [])
    .map(equipmentIdFor)
    .filter(Boolean);
  const active = equipmentIdFor(player?.weapon);
  if (active && !items.includes(active)) items.push(active);
  return { items, active };
};

export const buildPlayerLoadout = (player) => {
  const { items, active } = normalizedInventory(player);
  const bySlot = (slot) => items.filter((id) => EQUIPMENT_CATALOG[id]?.slot === slot);
  return {
    primary: bySlot("primary")[0] || null,
    pistol: bySlot("pistol")[0] || null,
    utility: bySlot("utility"),
    knife: bySlot("knife")[0] || null,
    active,
  };
};

export const describePlayerLoadout = (player) => {
  const loadout = buildPlayerLoadout(player);
  const equipment = [
    loadout.primary,
    loadout.pistol,
    ...loadout.utility,
    player?.has_c4 ? "c4" : null,
    player?.has_defuse_kit ? "defuser" : null,
  ].filter(Boolean);
  return equipment.length
    ? equipment.map(equipmentLabel).join(", ")
    : "Sin equipamiento";
};

export const PROJECTILE_ICON_IDS = {
  he: "hegrenade",
  flashbang: "flashbang",
  smoke: "smokegrenade",
  molotov: "molotov",
  incendiary: "incgrenade",
  decoy: "decoy",
};

export const projectileIconPath = (type) => equipmentIconPath(PROJECTILE_ICON_IDS[type]);

export const playerIdentity = (player) => {
  const steamId = String(player?.steam_id || "");
  return steamId && steamId !== "0" ? steamId : `${player?.team || "?"}:${player?.name || "bot"}`;
};

export const stableRosterPlayers = (players, team) => players
  .filter((player) => player.team === team)
  .sort((left, right) => playerIdentity(left).localeCompare(playerIdentity(right)));
