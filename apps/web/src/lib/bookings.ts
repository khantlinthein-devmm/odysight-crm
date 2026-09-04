import { ApiError, USE_MOCKS, apiFetch, delay } from "./api";

export type BookingStatus =
  | "pending"
  | "confirmed"
  | "in_progress"
  | "completed"
  | "cancelled"
  | "no_show"
  | "rescheduled";

export type ServiceType =
  | "house_cleaning"
  | "condo_cleaning"
  | "deep_cleaning"
  | "move_in_out"
  | "after_renovation"
  | "office_cleaning"
  | "junk_removal"
  | "aircon_service";

export interface Booking {
  id: number;
  bookingNumber: string;
  customerName: string;
  serviceType: ServiceType;
  scheduledFor: string;
  durationMinutes: number;
  address: string;
  assignedCleaner: string;
  status: BookingStatus;
  notes: string;
  createdAt: string;
}

export type CreateBookingInput = Omit<
  Booking,
  "id" | "bookingNumber" | "createdAt"
>;

export type UpdateBookingInput = Partial<CreateBookingInput>;

let mockId = 300;

const mockBookings: Booking[] = [
  {
    id: 201,
    bookingNumber: "BK-2026-0201",
    customerName: "Somchai Prasert",
    serviceType: "condo_cleaning",
    scheduledFor: "2026-09-06T09:00:00Z",
    durationMinutes: 180,
    address: "Sukhumvit 38, Klongton, Bangkok",
    assignedCleaner: "Nok Srisuwan",
    status: "confirmed",
    notes: "Focus on kitchen and bathroom",
    createdAt: "2026-08-18T10:00:00Z",
  },
  {
    id: 202,
    bookingNumber: "BK-2026-0202",
    customerName: "Jane Smith",
    serviceType: "deep_cleaning",
    scheduledFor: "2026-09-07T13:00:00Z",
    durationMinutes: 240,
    address: "On Nut 10, Suan Luang, Bangkok",
    assignedCleaner: "Daeng Chaiya",
    status: "pending",
    notes: "",
    createdAt: "2026-08-16T09:30:00Z",
  },
  {
    id: 203,
    bookingNumber: "BK-2026-0203",
    customerName: "Omar Farouk",
    serviceType: "move_in_out",
    scheduledFor: "2026-09-08T08:00:00Z",
    durationMinutes: 300,
    address: "Thonglor 13, Wattana, Bangkok",
    assignedCleaner: "Mali Kongdee",
    status: "confirmed",
    notes: "Renters moving out",
    createdAt: "2026-08-13T15:20:00Z",
  },
  {
    id: 204,
    bookingNumber: "BK-2026-0204",
    customerName: "Maria Gomez",
    serviceType: "house_cleaning",
    scheduledFor: "2026-09-09T10:00:00Z",
    durationMinutes: 180,
    address: "Asok 4, Khlong Toei, Bangkok",
    assignedCleaner: "Som Intarakamhaeng",
    status: "completed",
    notes: "",
    createdAt: "2026-08-10T12:00:00Z",
  },
  {
    id: 205,
    bookingNumber: "BK-2026-0205",
    customerName: "Wei Chen",
    serviceType: "office_cleaning",
    scheduledFor: "2026-09-05T17:00:00Z",
    durationMinutes: 210,
    address: "Sathorn 12, Sathorn, Bangkok",
    assignedCleaner: "Pim Jiraroj",
    status: "cancelled",
    notes: "Client postponed",
    createdAt: "2026-08-08T08:45:00Z",
  },
];

function clone(booking: Booking): Booking {
  return { ...booking };
}

export async function getBookings(): Promise<Booking[]> {
  if (USE_MOCKS) {
    await delay(300);
    return mockBookings.map(clone);
  }
  return apiFetch<Booking[]>("/api/v1/bookings");
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
    const booking: Booking = {
      ...input,
      id: ++mockId,
      bookingNumber: `BK-2026-0${mockId}`,
      createdAt: new Date().toISOString(),
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
    const booking = { ...mockBookings[index], ...input };
    mockBookings[index] = booking;
    return clone(booking);
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
