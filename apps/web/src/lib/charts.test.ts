import { describe, expect, it } from "vitest";
import {
  areaPath,
  donutSlices,
  gridlines,
  smoothPath,
  trendLayout,
} from "./charts";

describe("donutSlices", () => {
  it("splits a circle proportionally", () => {
    const slices = donutSlices(
      [
        { label: "A", value: 50, color: "red" },
        { label: "B", value: 50, color: "blue" },
      ],
      0,
    );
    expect(slices).toHaveLength(2);
    expect(slices[0]!.length).toBeCloseTo(50, 6);
    expect(slices[0]!.offset).toBeCloseTo(25, 6);
    expect(slices[1]!.offset).toBeCloseTo(-25, 6);
    expect(slices[0]!.percent).toBeCloseTo(50, 6);
  });

  it("skips non-positive values and empty totals", () => {
    expect(donutSlices([])).toEqual([]);
    expect(
      donutSlices([{ label: "A", value: 0, color: "red" }]),
    ).toEqual([]);
    const slices = donutSlices([
      { label: "A", value: 0, color: "red" },
      { label: "B", value: 10, color: "blue" },
    ]);
    expect(slices).toHaveLength(1);
    expect(slices[0]!.percent).toBeCloseTo(100, 6);
  });
});

describe("trendLayout", () => {
  it("maps values inside the viewBox", () => {
    const { points, min, max } = trendLayout([0, 50, 100], 600, 240);
    expect(points).toHaveLength(3);
    expect(min).toBe(0);
    expect(max).toBe(100);
    for (const p of points) {
      expect(p.x).toBeGreaterThanOrEqual(0);
      expect(p.x).toBeLessThanOrEqual(600);
      expect(p.y).toBeGreaterThanOrEqual(0);
      expect(p.y).toBeLessThanOrEqual(240);
    }
    // Higher value → smaller y (top of chart).
    expect(points[2]!.y).toBeLessThan(points[0]!.y);
  });
});

describe("smoothPath / areaPath", () => {
  it("builds a smooth curve through points", () => {
    const d = smoothPath([
      { x: 0, y: 10 },
      { x: 5, y: 0 },
      { x: 10, y: 10 },
    ]);
    expect(d.startsWith("M 0 10")).toBe(true);
    expect(d).toContain("C ");
  });

  it("closes an area path", () => {
    const line = smoothPath([
      { x: 0, y: 10 },
      { x: 10, y: 0 },
    ]);
    const area = areaPath(line, 0, 10, 100);
    expect(area.endsWith("Z")).toBe(true);
    expect(smoothPath([])).toBe("");
    expect(areaPath("", 0, 0, 0)).toBe("");
  });
});

describe("gridlines", () => {
  it("returns evenly spaced lines", () => {
    expect(gridlines(0, 100, 5)).toEqual([0, 25, 50, 75, 100]);
  });
});
