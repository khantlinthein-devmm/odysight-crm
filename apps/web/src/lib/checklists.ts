import { ApiError, USE_MOCKS, apiFetch, delay, getApiBaseUrl } from "./api";
import { OfflineQueued, enqueue, isOnline } from "./offline";

export interface ChecklistTemplate {
  id: number;
  name: string;
  serviceType: string;
  isActive: boolean;
  items: { id: number; label: string; sortOrder: number }[];
}

export interface ChecklistItem {
  id: number;
  label: string;
  isCompleted: boolean;
  completedBy?: string | null;
  completedAt?: string | null;
  notes: string;
  beforePhotoUrl?: string | null;
  afterPhotoUrl?: string | null;
  sortOrder: number;
}

export interface Checklist {
  id: number;
  bookingId: number;
  templateId?: number | null;
  status: "pending" | "in_progress" | "completed";
  clientSignature?: string | null;
  items: ChecklistItem[];
}

let mockId = 900;
const mockChecklists: Checklist[] = [
  {
    id: 901, bookingId: 204, templateId: 1, status: "in_progress",
    items: [
      { id: 1, label: "Production area cleaned", isCompleted: true, notes: "", sortOrder: 0 },
      { id: 2, label: "Floor cleaned", isCompleted: false, notes: "", sortOrder: 1 },
      { id: 3, label: "Waste removed", isCompleted: false, notes: "", sortOrder: 2 },
    ],
  },
];

export async function getChecklistTemplates(): Promise<ChecklistTemplate[]> {
  if (USE_MOCKS) {
    await delay(200);
    return [
      { id: 1, name: "Factory Deep Cleaning", serviceType: "deep_cleaning", isActive: true,
        items: [
          { id: 1, label: "Production area cleaned", sortOrder: 0 },
          { id: 2, label: "Floor cleaned", sortOrder: 1 },
          { id: 3, label: "Toilets cleaned", sortOrder: 2 },
          { id: 4, label: "Waste removed", sortOrder: 3 },
        ] },
    ];
  }
  const json = await apiFetch<{ data: ChecklistTemplate[] }>("/api/v1/checklists/templates");
  return Array.isArray(json) ? (json as unknown as ChecklistTemplate[]) : json.data;
}

export async function getChecklistByBooking(bookingId: number): Promise<Checklist> {
  if (USE_MOCKS) {
    await delay(200);
    const c = mockChecklists.find((x) => x.bookingId === bookingId);
    if (!c) throw new ApiError(404, "checklist not found");
    return { ...c };
  }
  return apiFetch<Checklist>(`/api/v1/checklists/bookings/${bookingId}`);
}

export async function createChecklist(bookingId: number, templateId?: number): Promise<Checklist> {
  if (USE_MOCKS) {
    await delay(300);
    const c: Checklist = { id: ++mockId, bookingId, templateId: templateId ?? null, status: "pending", items: [] };
    mockChecklists.unshift(c);
    return { ...c };
  }
  return apiFetch<Checklist>("/api/v1/checklists", { method: "POST", body: JSON.stringify({ bookingId, templateId }) });
}
export async function completeChecklistItem(itemId: number, isCompleted: boolean, completedBy?: string): Promise<Checklist> {
  if (USE_MOCKS) {
    await delay(250);
    for (const c of mockChecklists) {
      const it = c.items.find((x) => x.id === itemId);
      if (it) {
        it.isCompleted = isCompleted;
        it.completedBy = completedBy ?? null;
        c.status = c.items.every((x) => x.isCompleted) ? "completed" : "in_progress";
        return { ...c };
      }
    }
    throw new ApiError(404, "item not found");
  }
  if (!isOnline()) {
    const entryId = await enqueue("checklist.item", { itemId, isCompleted, completedBy });
    throw new OfflineQueued("checklist.item", entryId);
  }
  return apiFetch<Checklist>(`/api/v1/checklists/items/${itemId}`, {
    method: "PATCH", body: JSON.stringify({ isCompleted, completedBy }),
  });
}

export async function uploadItemPhoto(itemId: number, kind: "before" | "after", file: File): Promise<Checklist> {
  if (USE_MOCKS) {
    await delay(300);
    for (const c of mockChecklists) {
      const it = c.items.find((x) => x.id === itemId);
      if (it) {
        if (kind === "before") it.beforePhotoUrl = `mock://${file.name}`;
        else it.afterPhotoUrl = `mock://${file.name}`;
        return { ...c };
      }
    }
    throw new ApiError(404, "item not found");
  }
  const base = getApiBaseUrl();
  if (!base) throw new ApiError(0, "PUBLIC_API_URL is not configured");
  if (!isOnline()) {
    // Photos are stored on-device (IndexedDB) and uploaded automatically
    // when the connection returns — single before/after slot per item means
    // retries overwrite instead of duplicating.
    const entryId = await enqueue("checklist.photo", {
      itemId,
      kind,
      fileName: file.name,
      blob: file,
    });
    throw new OfflineQueued("checklist.photo", entryId);
  }
  const form = new FormData();
  form.append("kind", kind);
  form.append("file", file);
  const res = await fetch(`${base}/api/v1/checklists/items/${itemId}/photo`, {
    method: "POST", body: form, credentials: "include",
  });
  if (!res.ok) {
    let message = `Upload failed: ${res.statusText || res.status}`;
    try {
      const body = (await res.clone().json()) as { error?: string; message?: string };
      if (body?.error) message = body.error;
      else if (body?.message) message = body.message;
    } catch { /* keep default */ }
    throw new ApiError(res.status, message);
  }
  return (await res.json()) as Checklist;
}

export async function confirmChecklist(bookingId: number, clientSignature: string): Promise<Checklist> {
  if (USE_MOCKS) {
    await delay(300);
    const c = mockChecklists.find((x) => x.bookingId === bookingId);
    if (!c) throw new ApiError(404, "checklist not found");
    c.clientSignature = clientSignature;
    c.status = "completed";
    return { ...c };
  }
  return apiFetch<Checklist>(`/api/v1/checklists/bookings/${bookingId}/confirm`, {
    method: "POST",
    body: JSON.stringify({ clientSignature }),
  });
}
