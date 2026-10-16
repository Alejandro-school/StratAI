import { describe, expect, it } from "vitest";
import { resolveReplayPlace } from "./replayPlaces";

// Real env_cs_place hulls from de_mirage_texto.vmap. These points were checked
// independently from face vertices and outward normals, without using the
// runtime resolver, exported planes, AABB membership, or nearest-label lookup.
const REFERENCE_ID = "a0b0d249fe7fafcbfdf4f51b14dcc1ca6859441e34a6c2bb4a6b6fdbd06602f7";

describe("replay place resolution", () => {
  it.each([
    undefined,
    null,
    {},
    { x: 0, y: 0 },
    { x: null, y: 0, z: 0 },
    { x: 0, y: undefined, z: 0 },
    { x: Number.NaN, y: 0, z: 0 },
    { x: 0, y: Number.POSITIVE_INFINITY, z: 0 },
    { x: 0, y: 0, z: Number.NEGATIVE_INFINITY },
  ])("keeps a missing or nonfinite XYZ unavailable: %j", (position) => {
    expect(resolveReplayPlace(position)).toMatchObject({
      status: "unavailable", zone: null, candidates: [], build_compatibility: "unverified",
    });
  });

  it("does not use Mirage source references to assign a place on another map", () => {
    // Real local source point: de_mirage_places.json, position.ladder0.
    // Its point label is not a certified callout or a volume-membership fixture.
    const localReference = { x: -915.1489868164, y: -88, z: -96 };
    expect(resolveReplayPlace(localReference, "de_inferno")).toMatchObject({
      status: "unavailable", zone: null, candidates: [], build_compatibility: "unverified",
    });
  });

  it.each([
    [{ x: -1500, y: 750, z: 0 }, "71", "Apartments"],
    [{ x: -400, y: -400, z: -100 }, "1820", "Catwalk"],
    [{ x: -1226.5, y: -815.5, z: -114.7 }, "1829", "SnipersNest"],
  ])("maps the real XYZ interior %j to source label %s/%s", (position, id, label) => {
    const result = resolveReplayPlace(position);
    expect(result).toMatchObject({
      status: "mapped", zone: label, reference_id: REFERENCE_ID,
      build_compatibility: "unverified",
    });
    expect(result.candidates).toHaveLength(1);
    expect(result.candidates[0]).toMatchObject({ id, label });
  });

  it("does not assign Apartments when the same XY is above the actual volume", () => {
    // Apartments upper face is z=58.5313034058; no other hull contains this XYZ.
    const result = resolveReplayPlace({ x: -1500, y: 750, z: 100 });
    expect(result).toMatchObject({ status: "unavailable", zone: null, candidates: [] });
  });

  it("uses the oblique Apartments face instead of its axis-aligned bounding box", () => {
    // This point is inside Apartments' XYZ bounds but outside the diagonal
    // face joining (-2075,675) and (-1208,408); its z is above BombsiteB.
    const result = resolveReplayPlace({ x: -2000, y: 450, z: 30 });
    expect(result).toMatchObject({ status: "unavailable", zone: null, candidates: [] });
  });

  it("retains both source candidates in the real Apartments and BombsiteB overlap", () => {
    const result = resolveReplayPlace({ x: -1500, y: 500, z: -50 });
    expect(result).toMatchObject({
      status: "ambiguous", zone: null, reference_id: REFERENCE_ID,
      build_compatibility: "unverified",
    });
    expect(result.candidates).toHaveLength(2);
    expect(result.candidates).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: "71", label: "Apartments" }),
      expect.objectContaining({ id: "1833", label: "BombsiteB" }),
    ]));
  });

  it("resolves the lower of the same XY positions to BombsiteB without flattening Z", () => {
    // Apartments starts at z=-88.0000076294, while BombsiteB extends to z=-176.
    const result = resolveReplayPlace({ x: -1500, y: 500, z: -140 });
    expect(result).toMatchObject({ status: "mapped", zone: "BombsiteB" });
    expect(result.candidates).toHaveLength(1);
    expect(result.candidates[0]).toMatchObject({ id: "1833", label: "BombsiteB" });
  });

  it("keeps an exact BackAlley face as a boundary without inventing a unique zone", () => {
    // The real positive-X face of BackAlley is x=-128.
    const result = resolveReplayPlace({ x: -128, y: 600, z: -50 });
    expect(result).toMatchObject({ status: "boundary", zone: null });
    expect(result.candidates).toHaveLength(1);
    expect(result.candidates[0]).toMatchObject({ id: "267", label: "BackAlley" });
  });

  it("does not round an exterior point into the nearby BackAlley volume", () => {
    const result = resolveReplayPlace({ x: -127.75, y: 600, z: -50 });
    expect(result).toMatchObject({ status: "unavailable", zone: null, candidates: [] });
  });

  it("reports a boundary when the XY precision interval crosses a real face", () => {
    const point = { x: -128.25, y: 600, z: -50 };
    expect(resolveReplayPlace(point, "de_mirage", 0)).toMatchObject({ status: "mapped", zone: "BackAlley" });
    const imprecise = resolveReplayPlace(point, "de_mirage", 0.5);
    expect(imprecise).toMatchObject({ status: "boundary", zone: null });
    expect(imprecise.candidates).toHaveLength(1);
    expect(imprecise.candidates[0]).toMatchObject({ id: "267", label: "BackAlley" });
  });

  it("never promotes an internal position reference to a callout label", () => {
    // Exact local reference coordinates from de_mirage_places.json. They lie
    // outside every env_cs_place hull and are not themselves a region volume.
    const reference = { name: "position.actspawn", x: 854, y: 802, z: 32 };
    expect(resolveReplayPlace(reference)).toMatchObject({ status: "unavailable", zone: null, candidates: [] });
  });
});
