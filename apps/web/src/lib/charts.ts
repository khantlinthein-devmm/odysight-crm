export interface DonutSegment {
  label: string;
  value: number;
  color: string;
}

export interface DonutSlice extends DonutSegment {
  /** Length of the slice on a 100-unit circle. */
  length: number;
  /** Dash offset that positions the slice after previous ones (top = 25). */
  offset: number;
  percent: number;
}

/**
 * Converts values into stroke-dasharray slices for an SVG circle with
 * `pathLength="100"`. A small gap keeps adjacent slices visually separated.
 */
export function donutSlices(
  segments: DonutSegment[],
  gap = 1.2,
): DonutSlice[] {
  const total = segments.reduce(
    (s, g) => s + (Number.isFinite(g.value) && g.value > 0 ? g.value : 0),
    0,
  );
  if (total <= 0) return [];
  let start = 0;
  return segments
    .filter((g) => g.value > 0)
    .map((g) => {
      const raw = (g.value / total) * 100;
      const slice: DonutSlice = {
        ...g,
        length: Math.max(raw - gap, 0.5),
        offset: 25 - start,
        percent: (g.value / total) * 100,
      };
      start += raw;
      return slice;
    });
}

export interface Point {
  x: number;
  y: number;
}

export interface TrendLayout {
  points: Point[];
  min: number;
  max: number;
}

/** Maps values into chart coordinates inside a width×height viewBox. */
export function trendLayout(
  values: number[],
  width: number,
  height: number,
  padX = 28,
  padTop = 18,
  padBottom = 30,
): TrendLayout {
  const min = Math.min(0, ...values);
  const max = Math.max(1, ...values);
  const span = max - min || 1;
  const innerW = width - padX * 2;
  const innerH = height - padTop - padBottom;
  const points = values.map((v, i) => ({
    x: values.length === 1 ? width / 2 : padX + (i / (values.length - 1)) * innerW,
    y: padTop + (1 - (v - min) / span) * innerH,
  }));
  return { points, min, max };
}

/** Smooth Catmull-Rom → cubic Bézier path through points. */
export function smoothPath(points: Point[]): string {
  if (points.length === 0) return "";
  if (points.length === 1) return `M ${points[0]!.x} ${points[0]!.y}`;
  let d = `M ${points[0]!.x} ${points[0]!.y}`;
  for (let i = 0; i < points.length - 1; i++) {
    const p0 = points[Math.max(i - 1, 0)]!;
    const p1 = points[i]!;
    const p2 = points[i + 1]!;
    const p3 = points[Math.min(i + 2, points.length - 1)]!;
    const c1x = p1.x + (p2.x - p0.x) / 6;
    const c1y = p1.y + (p2.y - p0.y) / 6;
    const c2x = p2.x - (p3.x - p1.x) / 6;
    const c2y = p2.y - (p3.y - p1.y) / 6;
    d += ` C ${c1x} ${c1y}, ${c2x} ${c2y}, ${p2.x} ${p2.y}`;
  }
  return d;
}

/** Closes a line path into a filled area down to baseY. */
export function areaPath(line: string, firstX: number, lastX: number, baseY: number): string {
  if (!line) return "";
  return `${line} L ${lastX} ${baseY} L ${firstX} ${baseY} Z`;
}

/** Evenly spaced gridlines between min and max (inclusive). */
export function gridlines(min: number, max: number, count = 4): number[] {
  if (count < 2) return [min];
  const step = (max - min) / (count - 1);
  return Array.from({ length: count }, (_, i) => min + step * i);
}
