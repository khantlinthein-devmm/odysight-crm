import { ApiError, USE_MOCKS, apiFetch, delay } from "./api";

export type DocumentStatus = "pending" | "verified" | "rejected" | "expired";

export interface DocumentRecord {
  id: number;
  name: string;
  type: string;
  applicantName: string;
  fileSizeKb: number;
  status: DocumentStatus;
  uploadedAt: string;
}

export type UploadDocumentInput = Omit<
  DocumentRecord,
  "id" | "fileSizeKb" | "uploadedAt"
>;

export type UpdateDocumentInput = Partial<
  Pick<DocumentRecord, "name" | "type" | "status">
>;

let mockId = 400;

const mockDocuments: DocumentRecord[] = [
  {
    id: 301,
    name: "passport_john_doe.pdf",
    type: "Passport",
    applicantName: "John Doe",
    fileSizeKb: 1240,
    status: "verified",
    uploadedAt: "2026-08-19T11:00:00Z",
  },
  {
    id: 302,
    name: "diploma_jane_smith.pdf",
    type: "Education Certificate",
    applicantName: "Jane Smith",
    fileSizeKb: 830,
    status: "pending",
    uploadedAt: "2026-08-17T14:20:00Z",
  },
  {
    id: 303,
    name: "bank_statement_omar.pdf",
    type: "Financial Proof",
    applicantName: "Omar Farouk",
    fileSizeKb: 2210,
    status: "rejected",
    uploadedAt: "2026-08-15T09:10:00Z",
  },
  {
    id: 304,
    name: "photo_maria_gomez.jpg",
    type: "Photo",
    applicantName: "Maria Gomez",
    fileSizeKb: 420,
    status: "verified",
    uploadedAt: "2026-08-12T16:40:00Z",
  },
  {
    id: 305,
    name: "medical_wei_chen.pdf",
    type: "Medical Report",
    applicantName: "Wei Chen",
    fileSizeKb: 990,
    status: "expired",
    uploadedAt: "2026-08-05T13:25:00Z",
  },
];

function clone(doc: DocumentRecord): DocumentRecord {
  return { ...doc };
}

export async function getDocuments(): Promise<DocumentRecord[]> {
  if (USE_MOCKS) {
    await delay(300);
    return mockDocuments.map(clone);
  }
  return apiFetch<DocumentRecord[]>("/api/v1/documents");
}

export async function getDocument(id: number): Promise<DocumentRecord> {
  if (USE_MOCKS) {
    await delay(200);
    const doc = mockDocuments.find((d) => d.id === id);
    if (!doc) throw new ApiError(404, `Document ${id} not found`);
    return clone(doc);
  }
  return apiFetch<DocumentRecord>(`/api/v1/documents/${id}`);
}

export async function uploadDocument(
  input: UploadDocumentInput,
): Promise<DocumentRecord> {
  if (USE_MOCKS) {
    await delay(500);
    const doc: DocumentRecord = {
      ...input,
      id: ++mockId,
      fileSizeKb: Math.floor(Math.random() * 2000) + 100,
      uploadedAt: new Date().toISOString(),
    };
    mockDocuments.unshift(doc);
    return clone(doc);
  }
  return apiFetch<DocumentRecord>("/api/v1/documents", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateDocument(
  id: number,
  input: UpdateDocumentInput,
): Promise<DocumentRecord> {
  if (USE_MOCKS) {
    await delay(300);
    const index = mockDocuments.findIndex((d) => d.id === id);
    if (index === -1) throw new ApiError(404, `Document ${id} not found`);
    const doc = { ...mockDocuments[index], ...input };
    mockDocuments[index] = doc;
    return clone(doc);
  }
  return apiFetch<DocumentRecord>(`/api/v1/documents/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function deleteDocument(id: number): Promise<void> {
  if (USE_MOCKS) {
    await delay(300);
    const index = mockDocuments.findIndex((d) => d.id === id);
    if (index === -1) throw new ApiError(404, `Document ${id} not found`);
    mockDocuments.splice(index, 1);
    return;
  }
  await apiFetch<void>(`/api/v1/documents/${id}`, { method: "DELETE" });
}
