// Pure helpers for the dispatch board's drag-and-drop rescheduling.

/** Snap a drop position in a day column to a start time on that day. */
export function dropTime(
  day: Date,
  offsetPx: number,
  hourPx: number,
  windowStartHour: number,
  snapMinutes = 15,
): Date {
  const minutes = windowStartHour * 60 + (offsetPx / hourPx) * 60;
  const snapped = Math.round(minutes / snapMinutes) * snapMinutes;
  const clamped = Math.min(Math.max(snapped, 0), 24 * 60 - snapMinutes);
  const d = new Date(day);
  d.setHours(0, 0, 0, 0);
  d.setMinutes(clamped);
  return d;
}

/** Same time of day as `from`, moved onto `day`. */
export function onDay(from: Date, day: Date): Date {
  const d = new Date(day);
  d.setHours(from.getHours(), from.getMinutes(), 0, 0);
  return d;
}

/**
 * Crew after dragging a job from one cleaner's row to another's: the target
 * becomes primary (first), the source leaves, other crew stay. Dropping on
 * the unassigned row (toId null) removes the source only.
 */
export function reassignCrew(crew: number[], fromId: number | null, toId: number | null): number[] {
  const rest = crew.filter((id) => id !== fromId && id !== toId);
  return toId == null ? rest : [toId, ...rest];
}
