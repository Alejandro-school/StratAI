import reference from "../../../../../ai_coach/interpretation/maps/de_mirage_place_volumes.json";

export const REPLAY_PLACE_REFERENCE = reference.reference_id;
const EPSILON = 0.0001; // Numeric tolerance, not a gameplay distance.
const displayNames = { Apartments: "Apartamentos B", Catwalk: "Corta B", BombsiteA: "Site A", BombsiteB: "Site B" };
export const replayPlaceName = (label) => displayNames[label] || label;

// The generated, checked-in reference is validated against source hulls by the
// producer and Python reader. Resolve only positions already allowed to this team.
export function resolveReplayPlace(position, mapName = "de_mirage", xyPrecision = 0) {
  const result = { schema_id: "stratai.replay_place@1", status: "unavailable", zone: null,
    candidates: [], reference_id: reference.reference_id, build_compatibility: "unverified",
    reason: "outside_place_volumes" };
  if (mapName !== reference.map_name) return { ...result, reference_id: null, reason: "map_reference_unavailable" };
  const point = [position?.x, position?.y, position?.z];
  if (!point.every(Number.isFinite) || !Number.isFinite(xyPrecision) || xyPrecision < 0) {
    return { ...result, reason: "position_unavailable" };
  }
  for (const region of reference.regions) {
    let relation = null;
    for (const hull of region.hulls) {
      const distances = hull.planes.map(([x, y, z, offset]) => x * point[0] + y * point[1] + z * point[2] - offset);
      const tolerances = hull.planes.map(([x, y]) => EPSILON + xyPrecision * (Math.abs(x) + Math.abs(y)));
      if (distances.every((d, i) => d <= tolerances[i])) {
        relation = distances.every((d, i) => d < -tolerances[i]) || relation === "inside" ? "inside" : "boundary";
      }
    }
    if (relation) result.candidates.push({ id: region.id, label: region.label, relation });
  }
  const labels = [...new Set(result.candidates.map((c) => c.label))];
  if (labels.length > 1) return { ...result, status: "ambiguous", reason: "overlapping_place_volumes" };
  if (labels.length) {
    const interior = result.candidates.some((c) => c.relation === "inside");
    return { ...result, status: interior ? "mapped" : "boundary", zone: interior ? labels[0] : null,
      reason: interior ? "unique_local_volume" : "place_volume_boundary" };
  }
  return result;
}

export function projectReplayPlaces(players, mapName) {
  return players.map((player) => {
    // Do not forward an untrusted/raw place label or locate a corpse as a live player.
    const { place: _rawPlace, ...safe } = player;
    if (!safe.alive) return safe;
    const precision = safe.knowledge === "own_team_snapshot" ? 0.5 : 0;
    return { ...safe, place: { ...resolveReplayPlace(safe, mapName, precision),
      observation_tick: safe.observation_tick, availability_tick: safe.availability_tick, xy_precision: precision } };
  });
}

export function describeReplayPlace(place) {
  if (!place) return null;
  if (place.status === "mapped") return `${replayPlaceName(place.zone)} · mapa de referencia`;
  const names = [...new Set(place.candidates.map((c) => replayPlaceName(c.label)))].join(" / ");
  if (place.status === "ambiguous") return `Zona ambigua: ${names}`;
  if (place.status === "boundary") return `Límite: ${names}`;
  return "Zona no determinada";
}
