export const OVERVIEW = { camera: [-6, 17, 34], target: [3, 0, 26] };
// Illustrative utility stays in the open site, clear of Palace's native walls.
export const UTILITY_ARC_POINTS = [[4.8, .8, 25], [6, 4, 25], [7, 3, 25], [8, .4, 25]];

// Match the expanded panel and its right gutter in content.css. Utility is
// framed in the remaining scene area instead of aiming beneath the panel.
export function utilityPanelOcclusion(width) {
  if (width <= 760) return 0;
  if (width >= 1900) return Math.min(1200, width * .62) + 60;
  if (width <= 1100) return width * .64 + 26;
  return Math.min(1080, width * .68) + 34;
}

// Ground heights raycast against the aligned native CS2 geometry.
// Player height corresponds to about 69 Source units in this .02 presentation.
export const AGENTS = {
  ct: { path: '/images/Landing/CT_model.glb', height: 1.38, position: [4.5, -.4, 25], rotation: Math.PI, clip: 'ct_loadout_pistol01_walkup', pose: .995 },
  t: { path: '/images/Landing/Tmodel-v1.glb', height: 1.38, position: [4.15, 2.4, 14.56], rotation: .03, clip: 'Animacion', pose: .99 },
};

export const FRAMING_LANDMARKS = {
  siteA: [4.5, 1, 25], stairs: [4.15, 3.4, 14.56], ctBoxes: [-4, 1, 29],
};
