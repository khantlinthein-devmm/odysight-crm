import { describe, expect, it } from "vitest";
import { parseCoordinates } from "./geo";

describe("parseCoordinates", () => {
  it("reads plain lat,lng", () => {
    expect(parseCoordinates("13.7462, 100.5347")).toEqual({ latitude: 13.7462, longitude: 100.5347 });
  });
  it("reads Google Maps links", () => {
    expect(parseCoordinates("https://www.google.com/maps/@13.7462,100.5347,17z")).toEqual({ latitude: 13.7462, longitude: 100.5347 });
    expect(parseCoordinates("https://maps.google.com/?q=13.75,100.5")).toEqual({ latitude: 13.75, longitude: 100.5 });
    expect(parseCoordinates("https://www.google.com/maps/place/X/data=!3d13.7466!4d100.5393")).toEqual({ latitude: 13.7466, longitude: 100.5393 });
  });
  it("rejects text without coordinates", () => {
    expect(parseCoordinates("Siam Paragon")).toBeNull();
    expect(parseCoordinates("200, 100")).toBeNull();
  });
});
