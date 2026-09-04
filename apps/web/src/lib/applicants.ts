import { ApiError, USE_MOCKS, apiFetch, delay } from "./api";

export type ApplicantStatus =
  | "screening"
  | "document_collection"
  | "submitted"
  | "processing"
  | "approved"
  | "rejected";

export interface Applicant {
  id: number;
  firstName: string;
  lastName: string;
  email: string;
  phone: string;
  nationality: string;
  visaType: string;
  status: ApplicantStatus;
  createdAt: string;
}

export type CreateApplicantInput = Omit<Applicant, "id" | "createdAt">;

export type UpdateApplicantInput = Partial<CreateApplicantInput>;

let mockId = 200;

const mockApplicants: Applicant[] = [
  {
    id: 101,
    firstName: "John",
    lastName: "Doe",
    email: "john.doe@example.com",
    phone: "+1 555-0101",
    nationality: "USA",
    visaType: "Student Visa",
    status: "document_collection",
    createdAt: "2026-08-19T10:00:00Z",
  },
  {
    id: 102,
    firstName: "Jane",
    lastName: "Smith",
    email: "jane.smith@example.com",
    phone: "+1 555-0102",
    nationality: "UK",
    visaType: "Work Permit",
    status: "submitted",
    createdAt: "2026-08-17T09:30:00Z",
  },
  {
    id: 103,
    firstName: "Omar",
    lastName: "Farouk",
    email: "omar.farouk@example.com",
    phone: "+1 555-0103",
    nationality: "Egypt",
    visaType: "Tourist Visa",
    status: "approved",
    createdAt: "2026-08-14T15:20:00Z",
  },
  {
    id: 104,
    firstName: "Maria",
    lastName: "Gomez",
    email: "maria.gomez@example.com",
    phone: "+1 555-0104",
    nationality: "Spain",
    visaType: "Student Visa",
    status: "processing",
    createdAt: "2026-08-11T12:00:00Z",
  },
  {
    id: 105,
    firstName: "Wei",
    lastName: "Chen",
    email: "wei.chen@example.com",
    phone: "+1 555-0105",
    nationality: "China",
    visaType: "Business Visa",
    status: "rejected",
    createdAt: "2026-08-09T08:45:00Z",
  },
  {
    id: 106,
    firstName: "Aisha",
    lastName: "Khan",
    email: "aisha.khan@example.com",
    phone: "+1 555-0106",
    nationality: "Pakistan",
    visaType: "Student Visa",
    status: "screening",
    createdAt: "2026-08-07T17:10:00Z",
  },
];

function clone(applicant: Applicant): Applicant {
  return { ...applicant };
}

export async function getApplicants(): Promise<Applicant[]> {
  if (USE_MOCKS) {
    await delay(300);
    return mockApplicants.map(clone);
  }
  return apiFetch<Applicant[]>("/api/v1/applicants");
}

export async function getApplicant(id: number): Promise<Applicant> {
  if (USE_MOCKS) {
    await delay(200);
    const applicant = mockApplicants.find((a) => a.id === id);
    if (!applicant) throw new ApiError(404, `Applicant ${id} not found`);
    return clone(applicant);
  }
  return apiFetch<Applicant>(`/api/v1/applicants/${id}`);
}

export async function createApplicant(
  input: CreateApplicantInput,
): Promise<Applicant> {
  if (USE_MOCKS) {
    await delay(400);
    const applicant: Applicant = {
      ...input,
      id: ++mockId,
      createdAt: new Date().toISOString(),
    };
    mockApplicants.unshift(applicant);
    return clone(applicant);
  }
  return apiFetch<Applicant>("/api/v1/applicants", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateApplicant(
  id: number,
  input: UpdateApplicantInput,
): Promise<Applicant> {
  if (USE_MOCKS) {
    await delay(400);
    const index = mockApplicants.findIndex((a) => a.id === id);
    if (index === -1) throw new ApiError(404, `Applicant ${id} not found`);
    const applicant = { ...mockApplicants[index], ...input };
    mockApplicants[index] = applicant;
    return clone(applicant);
  }
  return apiFetch<Applicant>(`/api/v1/applicants/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function deleteApplicant(id: number): Promise<void> {
  if (USE_MOCKS) {
    await delay(300);
    const index = mockApplicants.findIndex((a) => a.id === id);
    if (index === -1) throw new ApiError(404, `Applicant ${id} not found`);
    mockApplicants.splice(index, 1);
    return;
  }
  await apiFetch<void>(`/api/v1/applicants/${id}`, { method: "DELETE" });
}
