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