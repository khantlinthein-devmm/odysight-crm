import { ApiError, USE_MOCKS, apiFetch, delay, toQuery, unwrapPage, type Page } from "./api";
import { OfflineQueued, enqueue, isOnline } from "./offline";

export type BookingStatus =
  | "pending"
  | "confirmed"
  | "in_progress"
  | "completed"
  | "cancelled"
  | "no_show"
  | "rescheduled";

// Service types are admin-editable via Settings → Service catalog,
// so this is an open string (the API no longer enforces a fixed enum).
export type ServiceType = string;

export interface BookingCleaner {
  id: number;
  name: string;
  role: "primary" | "crew";
}

export type RecurrenceFreq = "weekly" | "biweekly" | "monthly";

export interface Booking {
  id: number;
  bookingNumber: string;
  customerName: string;
  customerEmail: string;
  customerId?: number | null;
  siteId?: number | null;
  contractId?: number | null;
  serviceType: ServiceType;
  scheduledFor: string;
  durationMinutes: number;
  address: string;
  assignedCleaner: string;
  status: BookingStatus;
  notes: string;
  isRecurring: boolean;
  recurrence: RecurrenceFreq | "";
  seriesId: string | null;
  createdAt: string;
  cleaners: BookingCleaner[];
}

export type CreateBookingInput = Omit<
  Booking,
  "id" | "bookingNumber" | "createdAt" | "cleaners"
> & {
  // Selects real cleaners; first = primary, rest = crew. Derived by the form.
  cleanerIds?: number[];
};

export type UpdateBookingInput = Partial<CreateBookingInput> & {
  clearSiteId?: boolean;
  clearContractId?: boolean;
};

let mockId = 300;

// Parity with mockCleaners (lib/cleaners.ts) so cleanerIds resolve to names.
const mockCleanerNames: Record<number, string> = {
  601: "Nok Srisuwan",
  602: "Pim Jiraroj",
  603: "Daeng Chaiya",
  604: "Mali Kongdee",
  605: "Som Intarakamhaeng",
};

function resolveMockCleaners(cleanerIds?: number[]): BookingCleaner[] | undefined {
  if (!cleanerIds || cleanerIds.length === 0) return undefined;
  return cleanerIds
    .map((id, i) => ({
      id,
      name: mockCleanerNames[id] ?? `Cleaner ${id}`,
      role: (i === 0 ? "primary" : "crew") as BookingCleaner["role"],
    }));
}

function primaryFromCleaners(cleaners?: BookingCleaner[]): string {
  return cleaners?.find((c) => c.role === "primary")?.name ?? cleaners?.[0]?.name ?? "";
}

const mockBookings: Booking[] = [
  {
    id: 201,
    bookingNumber: "BK-2026-0201",
    customerName: "Somchai Prasert",
    customerEmail: "somchai@example.com",
    serviceType: "condo_cleaning",
    scheduledFor: "2026-09-06T09:00:00Z",
    durationMinutes: 180,
    address: "Sukhumvit 38, Klongton, Bangkok",
    assignedCleaner: "Nok Srisuwan",
    status: "confirmed",
    notes: "Focus on kitchen and bathroom",
    isRecurring: true,
    recurrence: "weekly",
    seriesId: "a1b2c3d4",
    createdAt: "2026-08-18T10:00:00Z",
    cleaners: [{ id: 601, name: "Nok Srisuwan", role: "primary" }],
  },
  {
    id: 202,
    bookingNumber: "BK-2026-0202",
    customerName: "Jane Smith",
    customerEmail: "jane.smith@example.com",
    serviceType: "deep_cleaning",
    scheduledFor: "2026-09-07T13:00:00Z",
    durationMinutes: 240,
    address: "On Nut 10, Suan Luang, Bangkok",
    assignedCleaner: "Daeng Chaiya",
    status: "pending",
    notes: "",
    isRecurring: false,
    recurrence: "",
    seriesId: null,
    createdAt: "2026-08-16T09:30:00Z",
    cleaners: [
      { id: 603, name: "Daeng Chaiya", role: "primary" },
      { id: 605, name: "Som Intarakamhaeng", role: "crew" },
    ],
  },
  {
    id: 203,
    bookingNumber: "BK-2026-0203",
    customerName: "Omar Farouk",
    customerEmail: "omar@example.com",
    serviceType: "move_in_out",
    scheduledFor: "2026-09-08T08:00:00Z",
    durationMinutes: 300,
    address: "Thonglor 13, Wattana, Bangkok",
    assignedCleaner: "Mali Kongdee",
    status: "confirmed",
    notes: "Renters moving out",
    isRecurring: false,
    recurrence: "",
    seriesId: null,
    createdAt: "2026-08-13T15:20:00Z",
    cleaners: [{ id: 604, name: "Mali Kongdee", role: "primary" }],
  },
  {
    id: 204,
    bookingNumber: "BK-2026-0204",
    customerName: "Maria Gomez",
    customerEmail: "maria.gomez@example.com",
    serviceType: "house_cleaning",
    scheduledFor: "2026-09-09T10:00:00Z",
    durationMinutes: 180,
    address: "Asok 4, Khlong Toei, Bangkok",
    assignedCleaner: "Som Intarakamhaeng",
    status: "completed",
    notes: "",
    isRecurring: true,
    recurrence: "biweekly",
    seriesId: "e5f6a7b8",
    createdAt: "2026-08-10T12:00:00Z",
    cleaners: [
      { id: 605, name: "Som Intarakamhaeng", role: "primary" },
      { id: 602, name: "Pim Jiraroj", role: "crew" },
    ],
  },
  {
    id: 205,
    bookingNumber: "BK-2026-0205",
    customerName: "Wei Chen",
    customerEmail: "wei.chen@example.com",
    serviceType: "office_cleaning",
    scheduledFor: "2026-09-05T17:00:00Z",
    durationMinutes: 210,
    address: "Sathorn 12, Sathorn, Bangkok",
    assignedCleaner: "Pim Jiraroj",
    status: "cancelled",
    notes: "Client postponed",
    isRecurring: false,
    recurrence: "",
    seriesId: null,
    createdAt: "2026-08-08T08:45:00Z",
    cleaners: [{ id: 602, name: "Pim Jiraroj", role: "primary" }],
  },
];

function clone(booking: Booking): Booking {
  return { ...booking };
}

function filterMocks(params: ListParams): Booking[] {
  const q = params.search?.trim().toLowerCase() ?? "";
  let rows = mockBookings.filter((b) => {
    if (params.status && b.status !== params.status) return false;
    if (params.from && b.scheduledFor < params.from) return false;
    if (params.to && b.scheduledFor >= params.to) return false;
    if (!q) return true;
    return [b.customerName, b.bookingNumber, b.address, b.assignedCleaner]
      .join(" ")
      .toLowerCase()
      .includes(q);
  });
  const offset = params.offset ?? 0;
  const limit = params.limit ?? rows.length;
  rows = rows.slice(offset, offset + limit);
  return rows.map(clone);
}

export interface ListParams { search?: string; status?: string; limit?: number; offset?: number; from?: string; to?: string; cleaner?: number }

export async function getBookings(params: ListParams = {}): Promise<Booking[]> {
  if (USE_MOCKS) {
    await delay(300);
    return filterMocks(params);
  }
  const json = await apiFetch<Booking[] | Page<Booking>>("/api/v1/bookings"+toQuery(params as Record<string, string|number|undefined>));
  return unwrapPage(json);
}

export async function getBooking(id: number): Promise<Booking> {
  if (USE_MOCKS) {
    await delay(200);
    const booking = mockBookings.find((b) => b.id === id);
    if (!booking) throw new ApiError(404, `Booking ${id} not found`);
    return clone(booking);
  }
  return apiFetch<Booking>(`/api/v1/bookings/${id}`);
}

export async function createBooking(
  input: CreateBookingInput,
): Promise<Booking> {
  if (USE_MOCKS) {
    await delay(400);
    const cleaners = resolveMockCleaners(input.cleanerIds);
    const booking: Booking = {
      ...input,
      id: ++mockId,
      bookingNumber: `BK-2026-0${mockId}`,
      createdAt: new Date().toISOString(),
      assignedCleaner: input.assignedCleaner || primaryFromCleaners(cleaners),
      cleaners: cleaners ?? [],
    };
    mockBookings.unshift(booking);
    return clone(booking);
  }
  return apiFetch<Booking>("/api/v1/bookings", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateBooking(
  id: number,
  input: UpdateBookingInput,
): Promise<Booking> {
  if (USE_MOCKS) {
    await delay(400);
    const index = mockBookings.findIndex((b) => b.id === id);
    if (index === -1) throw new ApiError(404, `Booking ${id} not found`);
    const existing = mockBookings[index];
    const cleaners =
      input.cleanerIds !== undefined
        ? (resolveMockCleaners(input.cleanerIds) ?? [])
        : existing.cleaners;
    const booking: Booking = {
      ...existing,
      ...input,
      cleaners,
      assignedCleaner:
        input.cleanerIds !== undefined
          ? primaryFromCleaners(cleaners)
          : input.assignedCleaner ?? existing.assignedCleaner,
    };
    mockBookings[index] = booking;
    return clone(booking);
  }
  if (!isOnline()) {
    // Queued status/assignment edits replay as PATCH on reconnect. A 409
    // (e.g. double-booked cleaner, reassigned job) becomes a dead-letter the
    // user can review instead of a silent retry.
    const entryId = await enqueue("booking.patch", { id, patch: input });
    throw new OfflineQueued("booking.patch", entryId);
  }
  return apiFetch<Booking>(`/api/v1/bookings/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function deleteBooking(id: number): Promise<void> {
  if (USE_MOCKS) {
    await delay(300);
    const index = mockBookings.findIndex((b) => b.id === id);
    if (index === -1) throw new ApiError(404, `Booking ${id} not found`);
    mockBookings.splice(index, 1);
    return;
  }
  await apiFetch<void>(`/api/v1/bookings/${id}`, { method: "DELETE" });
}

/**
 * Cleaner first-tap-wins accept (POST /api/v1/bookings/:id/accept).
 * The server resolves the cleaner's profile from the login. Offline it
 * queues: a later 409 ("no longer available") becomes a dead-letter the
 * cleaner can review instead of a silent retry.
 */
export async function acceptBooking(id: number): Promise<Booking> {
  if (USE_MOCKS) {
    await delay(400);
    const booking = mockBookings.find((b) => b.id === id);
    if (!booking) throw new ApiError(404, `Booking ${id} not found`);
    if (booking.status !== "pending" || (booking.cleaners ?? []).length > 0) {
      throw new ApiError(409, "booking is no longer available");
    }
    return clone(booking);
  }
  if (!isOnline()) {
    const entryId = await enqueue("booking.accept", { id });
    throw new OfflineQueued("booking.accept", entryId);
  }
  return apiFetch<Booking>(`/api/v1/bookings/${id}/accept`, {
    method: "POST",
    body: JSON.stringify({}),
  });
}

export interface MyJobsIdentity {
  email: string;
  name: string;
  cleanerProfileId?: number | null;
}

/**
 * Pure field-list selector for the mobile "My Jobs" cards.
 * Cleaners see their own upcoming jobs plus open (pending, unassigned) jobs
 * they can tap Accept on; everyone else sees the same filtered list as the
 * desktop table. Sorted soonest first.
 */
export function myJobs(list: Booking[], me: MyJobsIdentity): Booking[] {
  const name = me.name.trim().toLowerCase();
  const mine = (b: Booking): boolean => {
    if (me.cleanerProfileId != null) {
      if ((b.cleaners ?? []).some((c) => c.id === me.cleanerProfileId)) return true;
    }
    // Fallback: office often assigns by name string before profiles link up.
    if (name && b.assignedCleaner.trim().toLowerCase() === name) return true;
    return false;
  };
  const open = (b: Booking): boolean =>
    b.status === "pending" && (b.cleaners ?? []).length === 0 && !b.assignedCleaner;
  const active = (b: Booking): boolean =>
    b.status === "confirmed" || b.status === "in_progress" || b.status === "pending";
  return list
    .filter((b) => active(b) && (mine(b) || open(b)))
    .sort((a, b) => a.scheduledFor.localeCompare(b.scheduledFor));
}
