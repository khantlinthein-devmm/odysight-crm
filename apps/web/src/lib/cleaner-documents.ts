import { ApiError, USE_MOCKS, apiFetch, delay, getApiBaseUrl, toQuery } from "./api";

export type DocumentType =
  | "passport"
  | "visa"
  | "work_permit"
  | "pink_card"
  | "id_card"
  | "resume"
  | "other";

export type ExpiryStatus = "none" | "valid" | "expiring" | "expired";

export const DOCUMENT_TYPE_LABELS: Record<DocumentType, string> = {
  passport: "Passport",
  visa: "Visa",
  work_permit: "Work permit",
  pink_card: "Pink card",
  id_card: "ID card",
  resume: "Resume",
  other: "Other",
};

export interface CleanerDocument {
  id: number;
  cleanerId: number;
  type: DocumentType;
  number: string;
  numberMasked: boolean;
  issueDate: string | null;
  expiryDate: string | null;
  expiryStatus: ExpiryStatus;
  daysToExpiry: number | null;
  notes: string;
  hasFile: boolean;
  originalName: string;
  contentType: string;
  sizeBytes: number;
  createdAt: string;
  updatedAt: string;
}

/** Non-sensitive view for roles with cleaners.read only. */
export interface DocumentStatus {
  id: number;
  type: DocumentType;
  expiryDate: string | null;
  expiryStatus: ExpiryStatus;
  daysToExpiry: number | null;
}

export interface ExpiringDocument {
  documentId: number;
  cleanerId: number;
  cleanerName: string;
  type: DocumentType;
  expiryDate: string;
  expiryStatus: ExpiryStatus;
  daysToExpiry: number;
}

/** Dates are YYYY-MM-DD; "" clears a date on update. */
export interface DocumentFields {
  type?: DocumentType;
  number?: string;
  issueDate?: string;
  expiryDate?: string;
  notes?: string;
}

/** Types that carry no number or validity dates (only a file + notes). */
export const FILE_ONLY_TYPES: DocumentType[] = ["resume"];

export function maskNumber(value: string): string {
  const chars = [...value];
  if (chars.length === 0) return "";
  if (chars.length <= 6) return "•".repeat(chars.length);
  return chars.slice(0, 2).join("") + "•".repeat(chars.length - 5) + chars.slice(-3).join("");
}

export function computeExpiry(expiryDate: string | null, now = new Date()): { status: ExpiryStatus; days: number | null } {
  if (!expiryDate) return { status: "none", days: null };
  const [y, m, d] = expiryDate.split("-").map(Number);
  const today = Date.UTC(now.getFullYear(), now.getMonth(), now.getDate());
  const days = Math.round((Date.UTC(y, m - 1, d) - today) / 86_400_000);
  if (days < 0) return { status: "expired", days };
  if (days <= 60) return { status: "expiring", days };
  return { status: "valid", days };
}

// ---- mocks (dev only) ----

let mockId = 900;
const mockDocs: (CleanerDocument & { fullNumber: string })[] = [
  mockDoc(901, 601, "passport", "MD1234567", "2022-03-01", "2032-02-28"),
  mockDoc(902, 601, "work_permit", "WP99887766", "2025-11-20", offsetDate(25)),
  mockDoc(903, 602, "pink_card", "0012345678901", "2024-01-15", offsetDate(-3)),
];

function offsetDate(days: number): string {
  const d = new Date();
  d.setDate(d.getDate() + days);
  return d.toISOString().slice(0, 10);
}

function mockDoc(
  id: number, cleanerId: number, type: DocumentType, number: string, issueDate: string, expiryDate: string,
): CleanerDocument & { fullNumber: string } {
  const exp = computeExpiry(expiryDate);
  return {
    id, cleanerId, type, fullNumber: number, number: maskNumber(number), numberMasked: true,
    issueDate, expiryDate, expiryStatus: exp.status, daysToExpiry: exp.days, notes: "",
    hasFile: false, originalName: "", contentType: "", sizeBytes: 0,
    createdAt: new Date().toISOString(), updatedAt: new Date().toISOString(),
  };
}

function publicMock(d: CleanerDocument & { fullNumber: string }, reveal = false): CleanerDocument {
  const { fullNumber, ...rest } = d;
  const exp = computeExpiry(rest.expiryDate);
  return {
    ...rest,
    number: reveal ? fullNumber : maskNumber(fullNumber),
    numberMasked: !reveal,
    expiryStatus: exp.status,
    daysToExpiry: exp.days,
  };
}

function findMock(id: number) {
  const doc = mockDocs.find((d) => d.id === id);
  if (!doc) throw new ApiError(404, "document not found");
  return doc;
}

// ---- API ----

export async function getCleanerDocuments(cleanerId: number): Promise<CleanerDocument[]> {
  if (USE_MOCKS) {
    await delay(200);
    return mockDocs.filter((d) => d.cleanerId === cleanerId).map((d) => publicMock(d));
  }
  return apiFetch<CleanerDocument[]>("/api/v1/cleaner-documents" + toQuery({ cleanerId }));
}

export async function getDocumentStatuses(cleanerId: number): Promise<DocumentStatus[]> {
  if (USE_MOCKS) {
    await delay(150);
    return mockDocs
      .filter((d) => d.cleanerId === cleanerId)
      .map((d) => {
        const exp = computeExpiry(d.expiryDate);
        return { id: d.id, type: d.type, expiryDate: d.expiryDate, expiryStatus: exp.status, daysToExpiry: exp.days };
      });
  }
  return apiFetch<DocumentStatus[]>("/api/v1/cleaner-documents/status" + toQuery({ cleanerId }));
}

export async function getExpiringDocuments(days = 60): Promise<ExpiringDocument[]> {
  if (USE_MOCKS) {
    await delay(200);
    return mockDocs
      .map((d) => ({ d, exp: computeExpiry(d.expiryDate) }))
      .filter(({ exp }) => exp.days !== null && exp.days <= days)
      .sort((a, b) => (a.exp.days ?? 0) - (b.exp.days ?? 0))
      .map(({ d, exp }) => ({
        documentId: d.id, cleanerId: d.cleanerId, cleanerName: `Cleaner #${d.cleanerId}`, type: d.type,
        expiryDate: d.expiryDate!, expiryStatus: exp.status, daysToExpiry: exp.days!,
      }));
  }
  return apiFetch<ExpiringDocument[]>("/api/v1/cleaner-documents/expiring" + toQuery({ days }));
}

/** Full (unmasked) document. Every call is written to the audit log. */
export async function revealDocument(id: number): Promise<CleanerDocument> {
  if (USE_MOCKS) {
    await delay(150);
    return publicMock(findMock(id), true);
  }
  return apiFetch<CleanerDocument>(`/api/v1/cleaner-documents/${id}`);
}

export async function createDocument(cleanerId: number, fields: DocumentFields, file?: File | null): Promise<CleanerDocument> {
  if (USE_MOCKS) {
    await delay(300);
    const doc = mockDoc(++mockId, cleanerId, fields.type ?? "other", (fields.number ?? "").toUpperCase(),
      fields.issueDate ?? "", fields.expiryDate ?? "");
    doc.issueDate = fields.issueDate || null;
    doc.expiryDate = fields.expiryDate || null;
    doc.notes = fields.notes ?? "";
    if (file) Object.assign(doc, { hasFile: true, originalName: file.name, contentType: file.type, sizeBytes: file.size });
    mockDocs.push(doc);
    return publicMock(doc);
  }
  const base = getApiBaseUrl();
  if (!base) throw new ApiError(0, "PUBLIC_API_URL is not configured");
  const form = new FormData();
  form.append("cleanerId", String(cleanerId));
  for (const [k, v] of Object.entries(fields)) {
    if (v !== undefined) form.append(k, v);
  }
  if (file) form.append("file", file);
  const res = await fetch(`${base}/api/v1/cleaner-documents`, { method: "POST", body: form, credentials: "include" });
  if (!res.ok) throw await toApiError(res, "Upload failed");
  return (await res.json()) as CleanerDocument;
}

export async function updateDocument(id: number, fields: DocumentFields): Promise<CleanerDocument> {
  if (USE_MOCKS) {
    await delay(250);
    const doc = findMock(id);
    if (fields.number !== undefined) doc.fullNumber = fields.number.trim().toUpperCase();
    if (fields.type !== undefined) doc.type = fields.type;
    if (fields.issueDate !== undefined) doc.issueDate = fields.issueDate || null;
    if (fields.expiryDate !== undefined) doc.expiryDate = fields.expiryDate || null;
    if (fields.notes !== undefined) doc.notes = fields.notes;
    return publicMock(doc);
  }
  return apiFetch<CleanerDocument>(`/api/v1/cleaner-documents/${id}`, {
    method: "PATCH",
    body: JSON.stringify(fields),
  });
}

export async function deleteDocument(id: number): Promise<void> {
  if (USE_MOCKS) {
    await delay(200);
    mockDocs.splice(mockDocs.indexOf(findMock(id)), 1);
    return;
  }
  await apiFetch<void>(`/api/v1/cleaner-documents/${id}`, { method: "DELETE" });
}

/** Downloads the decrypted scan as a Blob (audited server-side). */
export async function fetchDocumentFile(id: number): Promise<Blob> {
  if (USE_MOCKS) {
    await delay(150);
    throw new ApiError(404, "Mock documents have no stored file");
  }
  const base = getApiBaseUrl();
  if (!base) throw new ApiError(0, "PUBLIC_API_URL is not configured");
  const res = await fetch(`${base}/api/v1/cleaner-documents/${id}/file`, { credentials: "include", cache: "no-store" });
  if (!res.ok) throw await toApiError(res, "Download failed");
  return res.blob();
}

async function toApiError(res: Response, fallback: string): Promise<ApiError> {
  let message = `${fallback}: ${res.statusText || res.status}`;
  try {
    const body = (await res.clone().json()) as { error?: string; message?: string };
    if (body?.error) message = body.error;
    else if (body?.message) message = body.message;
  } catch { /* keep default */ }
  return new ApiError(res.status, message);
}
