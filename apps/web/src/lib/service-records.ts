import { ApiError, USE_MOCKS, apiFetch, delay, toQuery, unwrapPage, type Page } from "./api";

export type ServiceRecordStatus =
  | "pending"
  | "completed"
  | "rescheduled"
  | "cancelled";

export interface ServiceRecord {
  id: number;
  bookingNumber: string;
  cleanerName: string;
  serviceType: string;
  rating: number | null;
  status: ServiceRecordStatus;
  notes: string;
  completedAt: string | null;
  createdAt: string;
}

export type CreateServiceRecordInput = Omit<
  ServiceRecord,
  "id" | "completedAt" | "createdAt"
>;

export type UpdateServiceRecordInput = Partial<
  Pick<
    ServiceRecord,
    "bookingNumber" | "cleanerName" | "serviceType" | "rating" | "status" | "notes"
  >
>;

let mockId = 400;

const mockServiceRecords: ServiceRecord[] = [
  {
    id: 301,
    bookingNumber: "BK-2026-0201",
    cleanerName: "Nok Srisuwan",
    serviceType: "condo_cleaning",
    rating: 5,
    status: "completed",
    notes: "Kitchen and bathroom spotless",
    completedAt: "2026-09-04T12:00:00Z",
    createdAt: "2026-09-04T09:05:00Z",
  },
  {
    id: 302,
    bookingNumber: "BK-2026-0202",
    cleanerName: "Daeng Chaiya",
    serviceType: "deep_cleaning",
    rating: null,
    status: "pending",
    notes: "",
    completedAt: null,
    createdAt: "2026-09-03T14:20:00Z",
  },
  {
    id: 303,
    bookingNumber: "BK-2026-0204",
    cleanerName: "Som Intarakamhaeng",
    serviceType: "house_cleaning",
    rating: 4,
    status: "completed",
    notes: "Client happy with result",
    completedAt: "2026-09-02T13:00:00Z",
    createdAt: "2026-09-02T10:10:00Z",
  },
  {
    id: 304,
    bookingNumber: "BK-2026-0205",
    cleanerName: "Pim Jiraroj",
    serviceType: "office_cleaning",
    rating: null,
    status: "cancelled",
    notes: "Postponed by client",
    completedAt: null,
    createdAt: "2026-09-01T16:40:00Z",
  },
  {
    id: 305,
    bookingNumber: "BK-2026-0203",
    cleanerName: "Mali Kongdee",
    serviceType: "move_in_out",
    rating: 5,
    status: "completed",
    notes: "Renters moved out, unit clean",
    completedAt: "2026-08-30T14:00:00Z",
    createdAt: "2026-08-30T08:25:00Z",
  },
];

function clone(record: ServiceRecord): ServiceRecord {
  return { ...record, rating: record.rating, completedAt: record.completedAt };
}

function filterMocks(params: ListParams): ServiceRecord[] {
  const q = params.search?.trim().toLowerCase() ?? "";
  let rows = mockServiceRecords.filter((r) => {
    if (params.status && r.status !== params.status) return false;
    if (!q) return true;
    return [r.bookingNumber, r.cleanerName, r.serviceType, r.notes]
      .join(" ")
      .toLowerCase()
      .includes(q);
  });
  const offset = params.offset ?? 0;
  const limit = params.limit ?? rows.length;
  rows = rows.slice(offset, offset + limit);
  return rows.map(clone);
}

export interface ListParams { search?: string; status?: string; limit?: number; offset?: number }

export async function getServiceRecords(params: ListParams = {}): Promise<ServiceRecord[]> {
  if (USE_MOCKS) {
    await delay(300);
    return filterMocks(params);
  }
  const json = await apiFetch<ServiceRecord[] | Page<ServiceRecord>>("/api/v1/service-records"+toQuery(params as Record<string, string|number|undefined>));
  return unwrapPage(json);
}

export async function getServiceRecord(id: number): Promise<ServiceRecord> {
  if (USE_MOCKS) {
    await delay(200);
    const record = mockServiceRecords.find((r) => r.id === id);
    if (!record) throw new ApiError(404, `Service record ${id} not found`);
    return clone(record);
  }
  return apiFetch<ServiceRecord>(`/api/v1/service-records/${id}`);
}

export async function createServiceRecord(
  input: CreateServiceRecordInput,
): Promise<ServiceRecord> {
  if (USE_MOCKS) {
    await delay(500);
    const record: ServiceRecord = {
      ...input,
      id: ++mockId,
      completedAt: null,
      createdAt: new Date().toISOString(),
    };
    mockServiceRecords.unshift(record);
    return clone(record);
  }
  return apiFetch<ServiceRecord>("/api/v1/service-records", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateServiceRecord(
  id: number,
  input: UpdateServiceRecordInput,
): Promise<ServiceRecord> {
  if (USE_MOCKS) {
    await delay(300);
    const index = mockServiceRecords.findIndex((r) => r.id === id);
    if (index === -1) throw new ApiError(404, `Service record ${id} not found`);
    const record = { ...mockServiceRecords[index], ...input };
    mockServiceRecords[index] = record;
    return clone(record);
  }
  return apiFetch<ServiceRecord>(`/api/v1/service-records/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function deleteServiceRecord(id: number): Promise<void> {
  if (USE_MOCKS) {
    await delay(300);
    const index = mockServiceRecords.findIndex((r) => r.id === id);
    if (index === -1) throw new ApiError(404, `Service record ${id} not found`);
    mockServiceRecords.splice(index, 1);
    return;
  }
  await apiFetch<void>(`/api/v1/service-records/${id}`, { method: "DELETE" });
}
