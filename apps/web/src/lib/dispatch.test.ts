import { describe, expect, it } from "vitest";
import { dropTime, onDay, reassignCrew } from "./dispatch";

describe("dispatch drag helpers", () => {
  const day = new Date(2026, 9, 5);

  it("snaps a drop to 15-minute steps from the visible window start", () => {
    // 56px per hour, window starts 07:00; 140px down = 2.5h → 09:30
    const t = dropTime(day, 140, 56, 7);
    expect([t.getHours(), t.getMinutes()]).toEqual([9, 30]);
    const t2 = dropTime(day, 150, 56, 7); // 2h41m → snaps to 09:45
    expect([t2.getHours(), t2.getMinutes()]).toEqual([9, 45]);
    expect(t.getDate()).toBe(5);
  });

  it("clamps drops above the grid to midnight", () => {
    const t = dropTime(day, -500, 56, 0);
    expect([t.getHours(), t.getMinutes()]).toEqual([0, 0]);
  });

  it("keeps the time of day when moving to another day", () => {
    const t = onDay(new Date(2026, 9, 5, 14, 15), new Date(2026, 9, 8));
    expect([t.getDate(), t.getHours(), t.getMinutes()]).toEqual([8, 14, 15]);
  });

  it("reassigns crew with the target as primary", () => {
    expect(reassignCrew([1, 2, 3], 1, 4)).toEqual([4, 2, 3]);
    expect(reassignCrew([1, 2], 1, 2)).toEqual([2]);
    expect(reassignCrew([1, 2], 1, null)).toEqual([2]);
    expect(reassignCrew([], null, 5)).toEqual([5]);
  });
});
