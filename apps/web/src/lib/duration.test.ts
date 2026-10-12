import { describe, expect, it } from "vitest";
import { formatDuration, hoursToMinutes, minutesToHours } from "./duration";

describe("minutesToHours", () => {
  it("converts and rounds to 2 decimals", () => {
    expect(minutesToHours(150)).toBe(2.5);
    expect(minutesToHours(20)).toBe(0.33);
    expect(minutesToHours(NaN)).toBe(0);
  });
});

describe("hoursToMinutes", () => {
  it("converts hour input, including typed strings", () => {
    expect(hoursToMinutes(2.5)).toBe(150);
    expect(hoursToMinutes("1.25")).toBe(75);
  });
  it("treats blanks and negatives as zero", () => {
    expect(hoursToMinutes("")).toBe(0);
    expect(hoursToMinutes(-1)).toBe(0);
  });
  it("round-trips common durations", () => {
    for (const m of [30, 60, 90, 120, 180, 210, 480]) {
      expect(hoursToMinutes(minutesToHours(m))).toBe(m);
    }
  });
});

describe("formatDuration", () => {
  it("shows hours from one hour up, minutes below", () => {
    expect(formatDuration(120)).toBe("2 h");
    expect(formatDuration(210)).toBe("3.5 h");
    expect(formatDuration(45)).toBe("45 min");
    expect(formatDuration(0)).toBe("—");
  });
});
