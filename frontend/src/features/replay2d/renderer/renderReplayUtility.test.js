import { describe, expect, it } from "vitest";
import { fitIconDimensions } from "./renderReplayUtility";

describe("fitIconDimensions", () => {
  it("fits official icons inside a constant 20px box without distortion", () => {
    expect(fitIconDimensions(80, 20)).toEqual({ width: 20, height: 5 });
    expect(fitIconDimensions(20, 40)).toEqual({ width: 10, height: 20 });
    expect(fitIconDimensions(0, 0)).toEqual({ width: 20, height: 20 });
  });
});
