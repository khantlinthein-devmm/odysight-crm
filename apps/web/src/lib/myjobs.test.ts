import { describe, expect, it } from "vitest";
import { myJobs, type Booking } from "./bookings";

function booking(over: Partial<Booking> = {}): Booking {
  return {
    id: 1,
    bookingNumber: "BK-1",
    customerName: "A",
    customerEmail: "",
    serviceType: "house_cleaning",
    scheduledFor: "2026-09-22T09:00:00Z",
    durationMinutes: 120,
    address: "",
    assignedCleaner: "",
    status: "pending",
    notes: "",
    isRecurring: false,
    recurrence: "",
    seriesId: null,
    createdAt: "2026-09-20T00:00:00Z",
    cleaners: [],
    ...over,
  };
}

const me = { email: "nok.s@smileclean.com", name: "Nok Srisuwan", cleanerProfileId: 601 };

describe("myJobs (mobile field list)", () => {
  it("shows my confirmed jobs by profile id", () => {
    const mine = booking({ id: 1, status: "confirmed", cleaners: [{ id: 601, name: "Nok", role: "primary" }] });
    const other = booking({ id: 2, status: "confirmed", cleaners: [{ id: 602, name: "Pim", role: "primary" }] });
    const res = myJobs([other, mine], me);
    expect(res.map((b) => b.id)).toEqual([1]);
  });

  it("matches by assigned name when no profile id is known", () => {
    const b = booking({ id: 3, status: "confirmed", assignedCleaner: "Nok Srisuwan" });
    const res = myJobs([b], { email: "", name: "Nok Srisuwan", cleanerProfileId: null });
    expect(res.map((x) => x.id)).toEqual([3]);
  });

  it("shows open jobs anyone can accept", () => {
    const open = booking({ id: 4, status: "pending" });
    expect(myJobs([open], me).map((b) => b.id)).toEqual([4]);
  });

  it("hides completed, cancelled, and other cleaners' jobs", () => {
    const list = [
      booking({ id: 5, status: "completed", cleaners: [{ id: 601, name: "Nok", role: "primary" }] }),
      booking({ id: 6, status: "cancelled", cleaners: [{ id: 601, name: "Nok", role: "primary" }] }),
      booking({ id: 7, status: "confirmed", assignedCleaner: "Pim Jiraroj" }),
    ];
    expect(myJobs(list, me)).toEqual([]);
  });

  it("sorts soonest first", () => {
    const a = booking({ id: 8, status: "pending", scheduledFor: "2026-09-25T09:00:00Z" });
    const b = booking({ id: 9, status: "pending", scheduledFor: "2026-09-22T09:00:00Z" });
    expect(myJobs([a, b], me).map((x) => x.id)).toEqual([9, 8]);
  });
});
