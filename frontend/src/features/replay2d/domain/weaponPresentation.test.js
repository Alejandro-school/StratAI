import { describe, expect, it } from "vitest";
import {
  buildPlayerLoadout,
  describePlayerLoadout,
  equipmentIdFor,
  equipmentIconPath,
  projectileIconPath,
} from "./weaponPresentation";

describe("weaponPresentation", () => {
  it("normalizes a complete competitive loadout and preserves grenade quantities", () => {
    const player = {
      weapon: "MP7",
      weapons: [
        "MP7",
        "Five-SeveN",
        "Flashbang",
        "Flashbang",
        "Smoke Grenade",
        "HE Grenade",
        "Knife",
      ],
      has_c4: true,
      has_defuse_kit: true,
    };

    expect(buildPlayerLoadout(player)).toEqual({
      primary: "mp7",
      pistol: "fiveseven",
      utility: ["flashbang", "flashbang", "smokegrenade", "hegrenade"],
      knife: "knife",
      active: "mp7",
    });
    expect(describePlayerLoadout(player)).toBe(
      "MP7, Five-SeveN, Flashbang, Flashbang, Smoke Grenade, HE Grenade, C4, Defuse Kit",
    );
  });

  it("maps parser aliases and every projectile subtype to official assets", () => {
    expect(equipmentIdFor("Incendiary Grenade")).toBe("incgrenade");
    expect(equipmentIdFor("Five-SeveN")).toBe("fiveseven");
    expect(equipmentIconPath("AK-47")).toBe("/images/cs2/equipment/ak47.svg");
    expect(projectileIconPath("he")).toBe("/images/cs2/equipment/hegrenade.svg");
    expect(projectileIconPath("incendiary")).toBe("/images/cs2/equipment/incgrenade.svg");
  });
});
