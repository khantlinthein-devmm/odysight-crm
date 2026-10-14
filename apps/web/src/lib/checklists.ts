import { ApiError, USE_MOCKS, apiFetch, delay, getApiBaseUrl, toQuery } from "./api";
import { OfflineQueued, enqueue, isOnline } from "./offline";

/** How a cleaner answers an item: tick, 1-5 score, or pass / fail / n/a. */
export type ItemType = "check" | "score" | "pass_fail";
export type ItemResult = "pass" | "fail" | "na";
export type ReviewStatus = "" | "approved" | "rework";

export interface TemplateItem {
  id?: number;
  label: string;
  itemType: ItemType;
  requirePhoto: boolean;
  sortOrder?: number;
}

export interface ChecklistTemplate {
  id: number;
  name: string;
  serviceType: string;
  isActive: boolean;
  items: TemplateItem[];
}

export interface ChecklistItem {
  id: number;
  label: string;
  itemType: ItemType;
  requirePhoto: boolean;
  isCompleted: boolean;
  score?: number | null;
  result?: ItemResult | null;
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
  clientConfirmedAt?: string | null;
  submittedAt?: string | null;
  submittedBy?: string | null;
  reviewStatus: ReviewStatus;
  reviewNote: string;
  reviewedBy?: string | null;
  reviewedAt?: string | null;
  locked: boolean;
  avgScore?: number | null;
  items: ChecklistItem[];
}

/** One row of the office review list. */
export interface ChecklistSummary {
  id: number;
  bookingId: number;
  bookingNumber: string;
  customerName: string;
  serviceType: string;
  scheduledFor: string;
  cleaners: string;
  status: Checklist["status"];
  submittedAt?: string | null;
  reviewStatus: ReviewStatus;
  itemsTotal: number;
  itemsDone: number;
  avgScore?: number | null;
  failCount: number;
  photoCount: number;
}

/** Compact state for a job card badge. */
export interface ChecklistStatus {
  bookingId: number;
  status: Checklist["status"];
  submitted: boolean;
  reviewStatus: ReviewStatus;
  reviewNote: string;
  itemsTotal: number;
  itemsDone: number;
}

export type ChecklistState = "pending" | "in_progress" | "submitted" | "approved" | "rework";

/** The one state a checklist is in, for badges and filters. */
export function checklistState(c: {
  status: Checklist["status"];
  submittedAt?: string | null;
  submitted?: boolean;
  reviewStatus: ReviewStatus;
}): ChecklistState {
  if (c.reviewStatus === "approved") return "approved";
  if (c.reviewStatus === "rework") return "rework";
  if (c.submittedAt || c.submitted) return "submitted";
  return c.status === "pending" ? "pending" : "in_progress";
}

/** Office-facing label and colour per state. */
export const CHECKLIST_STATE_BADGE: Record<ChecklistState, { label: string; tone: string }> = {
  pending: { label: "Not started", tone: "bg-gray-100 text-gray-700" },
  in_progress: { label: "In progress", tone: "bg-blue-50 text-blue-700" },
  submitted: { label: "Waiting review", tone: "bg-amber-50 text-amber-800" },
  approved: { label: "Approved", tone: "bg-green-50 text-green-700" },
  rework: { label: "Sent back", tone: "bg-red-50 text-red-700" },
};

/** What still stops a submit (mirrors the API's check). */
export function submitProblems(c: Checklist): ChecklistItem[] {
  return c.items.filter(
    (it) =>
      !it.isCompleted ||
      (it.requirePhoto && !it.afterPhotoUrl) ||
      (it.result === "fail" && !it.notes.trim()),
  );
}

let mockId = 900;
const mockChecklists: Checklist[] = [
  {
    id: 901, bookingId: 204, templateId: 1, status: "in_progress", reviewStatus: "", reviewNote: "", locked: false,
    items: [
      { id: 1, label: "Production area cleaned", itemType: "check", requirePhoto: false, isCompleted: true, notes: "", sortOrder: 0 },
      { id: 2, label: "Floor cleaned", itemType: "check", requirePhoto: false, isCompleted: false, notes: "", sortOrder: 1 },
      { id: 3, label: "Waste removed", itemType: "pass_fail", requirePhoto: true, isCompleted: false, notes: "", sortOrder: 2 },
      { id: 4, label: "Overall finish", itemType: "score", requirePhoto: false, isCompleted: false, notes: "", sortOrder: 3 },
    ],
  },
];

export async function getChecklistTemplates(): Promise<ChecklistTemplate[]> {
  if (USE_MOCKS) {
    await delay(200);
    return [
      { id: 1, name: "Factory Deep Cleaning", serviceType: "deep_cleaning", isActive: true,
        items: [
          { id: 1, label: "Production area cleaned", itemType: "check", requirePhoto: false, sortOrder: 0 },
          { id: 2, label: "Floor cleaned", itemType: "check", requirePhoto: false, sortOrder: 1 },
          { id: 3, label: "Toilets cleaned", itemType: "pass_fail", requirePhoto: true, sortOrder: 2 },
          { id: 4, label: "Waste removed", itemType: "check", requirePhoto: false, sortOrder: 3 },
        ] },
    ];
  }
  const json = await apiFetch<{ data: ChecklistTemplate[] }>("/api/v1/checklists/templates?limit=100");
  return Array.isArray(json) ? (json as unknown as ChecklistTemplate[]) : json.data;
}

export interface TemplateInput {
  name: string;
  serviceType: string;
  isActive?: boolean;
  items: Pick<TemplateItem, "label" | "itemType" | "requirePhoto">[];
}

export async function createChecklistTemplate(input: TemplateInput): Promise<ChecklistTemplate> {
  return apiFetch<ChecklistTemplate>("/api/v1/checklists/templates", {
    method: "POST",
    body: JSON.stringify({ name: input.name, serviceType: input.serviceType, items: input.items }),
  });
}

export async function updateChecklistTemplate(id: number, input: TemplateInput): Promise<ChecklistTemplate> {
  return apiFetch<ChecklistTemplate>(`/api/v1/checklists/templates/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export interface ChecklistListParams {
  state?: ChecklistState | "";
  cleanerId?: number | "";
  from?: string;
  to?: string;
  search?: string;
  limit?: number;
  offset?: number;
}

export async function listChecklists(params: ChecklistListParams = {}): Promise<{ data: ChecklistSummary[]; total: number }> {
  if (USE_MOCKS) {
    await delay(200);
    return { data: [], total: 0 };
  }
  const q = toQuery({
    state: params.state || undefined,
    cleanerId: params.cleanerId || undefined,
    from: params.from || undefined,
    to: params.to || undefined,
    search: params.search || undefined,
    limit: params.limit,
    offset: params.offset,
  });
  return apiFetch<{ data: ChecklistSummary[]; total: number }>(`/api/v1/checklists${q}`);
}

export async function getChecklistStatuses(bookingIds: number[]): Promise<ChecklistStatus[]> {
  if (USE_MOCKS || bookingIds.length === 0) return [];
  const json = await apiFetch<{ data: ChecklistStatus[] }>(
    `/api/v1/checklists/status?bookings=${bookingIds.slice(0, 200).join(",")}`,
  );
  return json.data ?? [];
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
    const c: Checklist = {
      id: ++mockId, bookingId, templateId: templateId ?? null, status: "pending",
      reviewStatus: "", reviewNote: "", locked: false, items: [],
    };
    mockChecklists.unshift(c);
    return { ...c };
  }
  return apiFetch<Checklist>("/api/v1/checklists", { method: "POST", body: JSON.stringify({ bookingId, templateId }) });
}

/** Item answer. isCompleted drives tick items; score (0 clears) and result
 * ("" clears) drive the other types; notes is optional. */
export interface ItemAnswer {
  isCompleted: boolean;
  score?: number;
  result?: ItemResult | "";
  notes?: string;
}

export async function answerChecklistItem(itemId: number, answer: ItemAnswer): Promise<Checklist> {
  if (USE_MOCKS) {
    await delay(250);
    for (const c of mockChecklists) {
      const it = c.items.find((x) => x.id === itemId);
      if (it) {
        if (answer.score !== undefined) it.score = answer.score || null;
        if (answer.result !== undefined) it.result = answer.result || null;
        if (answer.notes !== undefined) it.notes = answer.notes;
        it.isCompleted =
          it.itemType === "score" ? !!it.score : it.itemType === "pass_fail" ? !!it.result : answer.isCompleted;
        c.status = c.items.every((x) => x.isCompleted) ? "completed" : "in_progress";
        return { ...c };
      }
    }
    throw new ApiError(404, "item not found");
  }
  if (!isOnline()) {
    const entryId = await enqueue("checklist.item", { itemId, ...answer });
    throw new OfflineQueued("checklist.item", entryId);
  }
  return apiFetch<Checklist>(`/api/v1/checklists/items/${itemId}`, {
    method: "PATCH", body: JSON.stringify(answer),
  });
}

export async function completeChecklistItem(itemId: number, isCompleted: boolean): Promise<Checklist> {
  return answerChecklistItem(itemId, { isCompleted });
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
    return { ...c };
  }
  return apiFetch<Checklist>(`/api/v1/checklists/bookings/${bookingId}/confirm`, {
    method: "POST",
    body: JSON.stringify({ clientSignature }),
  });
}

export async function submitChecklist(bookingId: number): Promise<Checklist> {
  if (USE_MOCKS) {
    await delay(300);
    const c = mockChecklists.find((x) => x.bookingId === bookingId);
    if (!c) throw new ApiError(404, "checklist not found");
    Object.assign(c, { submittedAt: new Date().toISOString(), locked: true, reviewStatus: "" });
    return { ...c };
  }
  return apiFetch<Checklist>(`/api/v1/checklists/bookings/${bookingId}/submit`, { method: "POST" });
}

export async function reviewChecklist(
  bookingId: number,
  decision: "approved" | "rework",
  note: string,
): Promise<Checklist> {
  if (USE_MOCKS) {
    await delay(300);
    const c = mockChecklists.find((x) => x.bookingId === bookingId);
    if (!c) throw new ApiError(404, "checklist not found");
    Object.assign(c, { reviewStatus: decision, reviewNote: note }, decision === "rework" ? { submittedAt: null, locked: false } : {});
    return { ...c };
  }
  return apiFetch<Checklist>(`/api/v1/checklists/bookings/${bookingId}/review`, {
    method: "POST",
    body: JSON.stringify({ decision, note }),
  });
}

const photoCache = new Map<string, Promise<string>>();

/** Loads a proof-of-work photo with the session cookie (the API serves them
 * privately) and returns a blob: URL; on-device and mock URLs pass through. */
export function checklistPhotoUrl(url: string): Promise<string> {
  if (/^(blob:|mock:|data:)/.test(url)) return Promise.resolve(url);
  let p = photoCache.get(url);
  if (!p) {
    const src = /^https?:/.test(url) ? url : `${getApiBaseUrl() ?? ""}${url}`;
    p = fetch(src, { credentials: "include" }).then(async (r) => {
      if (!r.ok) throw new ApiError(r.status, "photo not available");
      return URL.createObjectURL(await r.blob());
    });
    p.catch(() => photoCache.delete(url));
    photoCache.set(url, p);
  }
  return p;
}
