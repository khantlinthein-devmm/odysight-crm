// Durations are stored in minutes (API, database) but staff think in hours:
// "2.5 hours" reads better than "150 min". These convert at the UI edge.

/** 150 → 2.5 (rounded to 2 decimals, so 20 min → 0.33). */
export function minutesToHours(minutes: number): number {
  const n = Number(minutes);
  return Number.isFinite(n) ? Math.round((n / 60) * 100) / 100 : 0;
}

/** 2.5 → 150, rounded to whole minutes; blanks and negatives become 0. */
export function hoursToMinutes(hours: number | string): number {
  const n = Number(hours);
  return Number.isFinite(n) && n > 0 ? Math.round(n * 60) : 0;
}

/** 120 → "2 h", 150 → "2.5 h", 45 → "45 min", 0 → "—". */
export function formatDuration(minutes: number): string {
  const n = Number(minutes);
  if (!Number.isFinite(n) || n <= 0) return "—";
  if (n < 60) return `${Math.round(n)} min`;
  return `${minutesToHours(n)} h`;
}
