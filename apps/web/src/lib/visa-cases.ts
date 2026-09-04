import { ApiError, USE_MOCKS, apiFetch, delay } from "./api";

export type VisaCaseStatus =
  | "draft"
  | "in_review"
  | "submitted"
  | "additional_docs_required"
  | "approved"
  | "rejected"
  | "closed";

export interface VisaCase {
  id: number;
  caseNumber: string;
  applicantName: string;
  visaType: string;
  destination: string;
  assignedTo: string;
  status: VisaCaseStatus;
  createdAt: string;
}

export type CreateVisaCaseInput = Omit<
  VisaCase,
  "id" | "caseNumber" | "createdAt"
>;

export type UpdateVisaCaseInput = Partial<CreateVisaCaseInput>;

let mockId = 300;

const mockVisaCases: VisaCase[] = [
  {
    id: 201,
    caseNumber: "VC-2026-0201",
    applicantName: "John Doe",
    visaType: "Student Visa",
    destination: "Canada",
    assignedTo: "Sarah Miller",
    status: "in_review",
    createdAt: "2026-08-18T10:00:00Z",
  },
  {
    id: 202,
    caseNumber: "VC-2026-0202",
    applicantName: "Jane Smith",
    visaType: "Work Permit",
    destination: "Germany",
    assignedTo: "David Lee",
    status: "submitted",
    createdAt: "2026-08-16T09:30:00Z",
  },
  {
    id: 203,
    caseNumber: "VC-2026-0203",
    applicantName: "Omar Farouk",
    visaType: "Tourist Visa",
    destination: "USA",
    assignedTo: "Sarah Miller",
    status: "additional_docs_required",
    createdAt: "2026-08-13T15:20:00Z",
  },
  {
    id: 204,
    caseNumber: "VC-2026-0204",
    applicantName: "Maria Gomez",
    visaType: "Student Visa",
    destination: "Australia",
    assignedTo: "Priya Patel",
    status: "approved",
    createdAt: "2026-08-10T12:00:00Z",
  },
  {
    id: 205,
    caseNumber: "VC-2026-0205",
    applicantName: "Wei Chen",
    visaType: "Business Visa",
    destination: "Japan",
    assignedTo: "David Lee",
    status: "rejected",
    createdAt: "2026-08-08T08:45:00Z",
  },
];

function clone(visaCase: VisaCase): VisaCase {
  return { ...visaCase };
}

export async function getVisaCases(): Promise<VisaCase[]> {
  if (USE_MOCKS) {
    await delay(300);
    return mockVisaCases.map(clone);
  }
  return apiFetch<VisaCase[]>("/api/v1/visa-cases");
}

export async function getVisaCase(id: number): Promise<VisaCase> {
  if (USE_MOCKS) {
    await delay(200);
    const visaCase = mockVisaCases.find((c) => c.id === id);
    if (!visaCase) throw new ApiError(404, `Visa case ${id} not found`);
    return clone(visaCase);
  }
  return apiFetch<VisaCase>(`/api/v1/visa-cases/${id}`);
}

export async function createVisaCase(
  input: CreateVisaCaseInput,
): Promise<VisaCase> {
  if (USE_MOCKS) {
    await delay(400);
    const visaCase: VisaCase = {
      ...input,
      id: ++mockId,
      caseNumber: `VC-2026-0${mockId}`,
      createdAt: new Date().toISOString(),
    };
    mockVisaCases.unshift(visaCase);
    return clone(visaCase);
  }
  return apiFetch<VisaCase>("/api/v1/visa-cases", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateVisaCase(
  id: number,
  input: UpdateVisaCaseInput,
): Promise<VisaCase> {
  if (USE_MOCKS) {
    await delay(400);
    const index = mockVisaCases.findIndex((c) => c.id === id);
    if (index === -1) throw new ApiError(404, `Visa case ${id} not found`);
    const visaCase = { ...mockVisaCases[index], ...input };
    mockVisaCases[index] = visaCase;
    return clone(visaCase);
  }
  return apiFetch<VisaCase>(`/api/v1/visa-cases/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}
