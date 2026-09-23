import { USE_MOCKS, apiFetch, delay, getApiBaseUrl } from "./api";

export interface PortalCustomer {
  id: number;
  name: string;
  email: string;
  phone: string;
  address: string;
  area: string;
}

export interface PortalBooking {
  id: number;
  bookingNumber: string;
  serviceType: string;
  scheduledFor: string;
  durationMinutes: number;
  address: string;
  assignee: string;
  status: string;
  notes: string;
}

export interface PortalFeedback {
  bookingId: number;
  rating: number;
  comment: string;
}

export interface PortalSite {
  id: number;
  name: string;
  address: string;
  isDefault: boolean;
}

export interface PortalService {
  id: string;
  name: string;
  durationMinutes: number;
  basePrice: number;
  active: boolean;
}

export interface PortalCreateBookingInput {
  serviceType: string;
  scheduledFor: string;
  durationMinutes?: number;
  siteId?: number | null;
  address?: string;
  notes?: string;
}

interface PortalLoginResponse {
  token: string;
  customer: PortalCustomer;
}

const CUSTOMER_KEY = "odysight_portal_customer";

export function getPortalCustomer(): PortalCustomer | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(CUSTOMER_KEY);
    return raw ? (JSON.parse(raw) as PortalCustomer) : null;
  } catch {
    return null;
  }
}

export function clearPortalCustomer(): void {
  try {
    localStorage.removeItem(CUSTOMER_KEY);
  } catch { /* ignore */ }
}

// Dev-only mock data (never used in production builds).
const mockCustomer: PortalCustomer = {
  id: 7,
  name: "Somchai Prasert",
  email: "somchai@smileclean.com",
  phone: "+66 812345678",
  address: "Sukhumvit 38, Bangkok",
  area: "Sukhumvit",
};

const mockBookings: PortalBooking[] = [
  {
    id: 201,
    bookingNumber: "BK-2026-0201",
    serviceType: "condo_cleaning",
    scheduledFor: "2026-09-05T09:00:00Z",
    durationMinutes: 120,
    address: "Sukhumvit 38, Bangkok",
    assignee: "Nok Srisuwan",
    status: "completed",
    notes: "",
  },
  {
    id: 205,
    bookingNumber: "BK-2026-0205",
    serviceType: "deep_cleaning",
    scheduledFor: "2026-09-18T10:00:00Z",
    durationMinutes: 240,
    address: "Sukhumvit 38, Bangkok",
    assignee: "Pichai Wong",
    status: "confirmed",
    notes: "Please bring extra cloths for the kitchen.",
  },
];

function cloneBooking(b: PortalBooking): PortalBooking {
  return { ...b };
}

export async function portalLogin(
  email: string,
  password: string,
): Promise<PortalCustomer> {
  if (USE_MOCKS) {
    await delay(400);
    if (
      email.trim().toLowerCase() !== mockCustomer.email ||
      password !== "portal123"
    ) {
      throw new Error("Invalid email or password");
    }
    return { ...mockCustomer };
  }
  const response = await apiFetch<PortalLoginResponse>(
    "/api/v1/portal/auth/login",
    { method: "POST", body: JSON.stringify({ email, password }) },
  );
  return response.customer;
}

export async function portalLogout(): Promise<void> {
  if (USE_MOCKS) {
    await delay(200);
    return;
  }
  const api = getApiBaseUrl();
  if (api && typeof window !== "undefined") {
    await fetch(`${api}/api/v1/portal/auth/logout`, {
      method: "POST",
      credentials: "include",
    }).catch(() => { /* ignore */ });
  }
}

export async function portalMe(): Promise<PortalCustomer> {
  if (USE_MOCKS) {
    await delay(150);
    return { ...mockCustomer };
  }
  return apiFetch<PortalCustomer>("/api/v1/portal/me");
}

export async function portalBookings(): Promise<PortalBooking[]> {
  if (USE_MOCKS) {
    await delay(300);
    return mockBookings.map(cloneBooking);
  }
  const json = await apiFetch<PortalBooking[]>("/api/v1/portal/bookings");
  return Array.isArray(json) ? json : [];
}

export async function portalChangePassword(
  currentPassword: string,
  newPassword: string,
): Promise<void> {
  if (USE_MOCKS) {
    await delay(300);
    if (newPassword.length < 8) {
      throw new Error("New password must be at least 8 characters");
    }
    return;
  }
  await apiFetch<{ status: string }>("/api/v1/portal/me/password", {
    method: "PATCH",
    body: JSON.stringify({ currentPassword, newPassword }),
  });
}

export async function portalSubmitFeedback(
  input: PortalFeedback,
): Promise<void> {
  if (USE_MOCKS) {
    await delay(300);
    return;
  }
  await apiFetch<{ status: string }>(
    `/api/v1/portal/bookings/${input.bookingId}/feedback`,
    {
      method: "POST",
      body: JSON.stringify({ rating: input.rating, comment: input.comment }),
    },
  );
}

export async function portalSites(): Promise<PortalSite[]> {
  if (USE_MOCKS) {
    await delay(200);
    return [
      { id: 501, name: "Default Site", address: "Sukhumvit 38, Bangkok", isDefault: true },
      { id: 502, name: "Silom Branch", address: "Silom 5, Bangkok", isDefault: false },
    ];
  }
  const json = await apiFetch<PortalSite[]>("/api/v1/portal/sites");
  return Array.isArray(json) ? json : [];
}

export async function portalServices(): Promise<PortalService[]> {
  if (USE_MOCKS) {
    await delay(200);
    return [
      { id: "house_cleaning", name: "House Cleaning", durationMinutes: 180, basePrice: 1500, active: true },
      { id: "condo_cleaning", name: "Condo Cleaning", durationMinutes: 120, basePrice: 1200, active: true },
      { id: "deep_cleaning", name: "Deep Cleaning", durationMinutes: 240, basePrice: 2500, active: true },
      { id: "office_cleaning", name: "Office Cleaning", durationMinutes: 210, basePrice: 2200, active: true },
    ];
  }
  const json = await apiFetch<PortalService[]>("/api/v1/portal/services");
  return Array.isArray(json) ? json : [];
}

export async function portalCreateBooking(
  input: PortalCreateBookingInput,
): Promise<PortalBooking> {
  if (USE_MOCKS) {
    await delay(400);
    const booking: PortalBooking = {
      id: Math.floor(Math.random() * 10000) + 300,
      bookingNumber: `BK-2026-${String(Math.floor(Math.random() * 9000) + 1000)}`,
      serviceType: input.serviceType,
      scheduledFor: input.scheduledFor,
      durationMinutes: input.durationMinutes ?? 120,
      address: input.address || "Sukhumvit 38, Bangkok",
      assignee: "",
      status: "pending",
      notes: input.notes ?? "",
    };
    mockBookings.unshift(booking);
    return { ...booking };
  }
  return apiFetch<PortalBooking>("/api/v1/portal/bookings", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function portalCancelBooking(bookingId: number): Promise<void> {
  if (USE_MOCKS) {
    await delay(300);
    const b = mockBookings.find((x) => x.id === bookingId);
    if (b) b.status = "cancelled";
    return;
  }
  await apiFetch<{ status: string }>(`/api/v1/portal/bookings/${bookingId}/cancel`, {
    method: "POST",
  });
}