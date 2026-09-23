import { describe, expect, it } from "vitest";
import { ApiError } from "./api";
import {
  OfflineQueued,
  enqueue,
  isOfflineQueued,
  listDeadLetters,
  pendingCount,
  persistSettingsSnapshot,
  processOutboxWith,
  readSettingsSnapshot,
  type OutboxEntry,
} from "./offline";

describe("settings snapshot (offline calculator catalog)", () => {
  it("round-trips the last-synced workspace settings", () => {
    persistSettingsSnapshot({ payments: { taxRatePercent: 7 }, services: [{ id: "x" }] });
    const snap = readSettingsSnapshot<{ payments: { taxRatePercent: number } }>();
    expect(snap?.payments?.taxRatePercent).toBe(7);
  });

  it("returns null when nothing was cached", () => {
    localStorage.clear();
    expect(readSettingsSnapshot()).toBeNull();
  });
});

describe("outbox queue", () => {
  it("enqueues and counts pending entries", async () => {
    const before = await pendingCount();
    await enqueue("attendance.check-in", { personType: "cleaner", personId: 1 });
    expect(await pendingCount()).toBe(before + 1);
    // Drain with a successful sender so tests stay isolated.
    await processOutboxWith(() => Promise.resolve({ ok: true }));
    expect(await pendingCount()).toBe(0);
  });

  it("applies entries on success", async () => {
    await enqueue("checklist.item", { itemId: 5, isCompleted: true });
    const res = await processOutboxWith(() => Promise.resolve({ ok: true }));
    expect(res.applied).toBe(1);
    expect(res.remaining).toBe(0);
  });

  it("treats duplicate check-in 409 as already applied (idempotent retry)", async () => {
    await enqueue("attendance.check-in", { personType: "cleaner", personId: 2 });
    const res = await processOutboxWith(() =>
      Promise.reject(new ApiError(409, "already checked in for today")),
    );
    expect(res.applied).toBe(1);
    expect(res.remaining).toBe(0);
    expect(await listDeadLetters()).toHaveLength(0);
  });

  it("dead-letters a reassigned booking 409 instead of retrying forever", async () => {
    await enqueue("booking.patch", { id: 9, patch: { status: "confirmed" } });
    const res = await processOutboxWith(() =>
      Promise.reject(new ApiError(409, "booking is no longer available")),
    );
    expect(res.dead).toBe(1);
    expect(res.remaining).toBe(0);
    const dead = await listDeadLetters();
    expect(dead.some((d) => d.type === "booking.patch")).toBe(true);
  });

  it("keeps network failures queued for retry with backoff", async () => {
    const id = await enqueue("attendance.check-out", { personType: "cleaner", personId: 3 });
    void id;
    const res = await processOutboxWith(() => Promise.reject(new TypeError("down")));
    expect(res.applied).toBe(0);
    expect(res.remaining).toBe(1);
    // Drain.
    await processOutboxWith(() => Promise.resolve({ ok: true }));
    expect(await pendingCount()).toBe(0);
  });

  it("recognizes the OfflineQueued signal", () => {
    const err: unknown = new OfflineQueued("checklist.photo", "q_1");
    expect(isOfflineQueued(err)).toBe(true);
    expect(isOfflineQueued(new Error("nope"))).toBe(false);
    expect((err as OfflineQueued).outboxType).toBe("checklist.photo");
  });

  it("dead-letters a lost accept race instead of retrying forever", async () => {
    await enqueue("booking.accept", { id: 11 });
    const res = await processOutboxWith(() =>
      Promise.reject(new ApiError(409, "booking is no longer available")),
    );
    expect(res.dead).toBe(1);
    expect(res.remaining).toBe(0);
  });

  it("entry ids are unique", async () => {
    const seen = new Set<string>();
    const sender = async (e: OutboxEntry) => {
      seen.add(e.id);
      return { ok: true };
    };
    await enqueue("checklist.item", { itemId: 1, isCompleted: true });
    await enqueue("checklist.item", { itemId: 2, isCompleted: true });
    await processOutboxWith(sender);
    expect(seen.size).toBe(2);
  });
});
