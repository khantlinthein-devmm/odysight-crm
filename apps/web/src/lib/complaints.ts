import { apiFetch, toQuery, unwrapPage, type Page } from "./api";

export type ComplaintStatus = "open" | "in_progress" | "resolved" | "closed";
export type Severity = "low" | "medium" | "high";

export const CATEGORY_LABELS: Record<string, string> = {
  quality: "Cleaning quality",
  missed_area: "Missed area",
  late: "Late arrival",
  no_show: "No-show",
  damage: "Damage / breakage",
  staff_behaviour: "Staff behaviour",
  other: "Other",
};

export const CHANNEL_LABELS: Record<string, string> = {
  phone: "Phone",
  line: "LINE",
  email: "Email",
  portal: "Customer portal",
  in_person: "In person",
};

export const SLA_HOURS: Record<Severity, number> = { high: 24, medium: 48, low: 72 };

export interface Complaint {
  id: number;
  complaintNumber: string;
  customerId: number;
  customerName: string;
  siteId: number | null;
  siteName: string;
  bookingId: number | null;
  bookingNumber: string;
  category: string;
  severity: Severity;
  channel: string;
  description: string;
  status: ComplaintStatus;
  dueAt: string;
  overdue: boolean;
  resolution: string;
  resolvedAt: string | null;
  recleanBookingId: number | null;
  recleanBookingNumber: string;
  createdAt: string;
  updatedAt: string;
}

export interface ComplaintFilters {
  status?: "" | "active" | ComplaintStatus;
  overdue?: boolean;
  customerId?: number;
  search?: string;
}

export async function getComplaints(f: ComplaintFilters = {}): Promise<Complaint[]> {
  const json = await apiFetch<Complaint[] | Page<Complaint>>(
    "/api/v1/complaints" +
      toQuery({
        status: f.status || undefined,
        overdue: f.overdue ? "true" : undefined,
        customerId: f.customerId,
        search: f.search,
        limit: 200,
      }),
  );
  return unwrapPage(json);
}

export function createComplaint(input: {
  customerId?: number;
  bookingId?: number;
  siteId?: number;
  category: string;
  severity: Severity;
  channel: string;
  description: string;
}): Promise<Complaint> {
  return apiFetch<Complaint>("/api/v1/complaints", { method: "POST", body: JSON.stringify(input) });
}

export function updateComplaint(
  id: number,
  input: Partial<Pick<Complaint, "status" | "severity" | "category" | "description" | "resolution">>,
): Promise<Complaint> {
  return apiFetch<Complaint>(`/api/v1/complaints/${id}`, { method: "PATCH", body: JSON.stringify(input) });
}

export function bookReclean(
  id: number,
  input: { scheduledFor: string; durationMinutes?: number; sameCrew?: boolean },
): Promise<Complaint> {
  return apiFetch<Complaint>(`/api/v1/complaints/${id}/reclean`, { method: "POST", body: JSON.stringify(input) });
}
