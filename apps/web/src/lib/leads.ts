import { ApiError, USE_MOCKS, apiFetch, delay, toQuery, unwrapPage, type Page } from "./api";

export type LeadStatus =
  | "new"
  | "contacted"
  | "quote_sent"
  | "booked"
  | "won"
  | "lost";

export type LeadSource =
  | "website"
  | "referral"
  | "line"
  | "facebook"
  | "walk_in"
  | "campaign";

export interface Lead {
  id: number;
  firstName: string;
  lastName: string;
  email: string;
  phone: string;
  status: LeadStatus;
  source: LeadSource;
  /** LINE OA identity (empty for non-LINE leads). */
  lineUserId?: string;
  linePictureUrl?: string;
  createdAt: string;
}

export type CreateLeadInput = Omit<
  Lead,
  "id" | "createdAt" | "lastName" | "email"
> & { lastName?: string; email?: string };

export type UpdateLeadInput = Partial<CreateLeadInput>;

let mockId = 100;

const mockLeads: Lead[] = [
  {
    id: 1,
    firstName: "Somchai",
    lastName: "Prasert",
    email: "somchai.p@example.com",
    phone: "+66 91 234 5678",
    status: "new",
    source: "line",
    createdAt: "2026-08-20T09:00:00Z",
  },
  {
    id: 2,
    firstName: "Jane",
    lastName: "Smith",
    email: "jane.smith@example.com",
    phone: "+66 81 555-0102",
    status: "contacted",
    source: "referral",
    createdAt: "2026-08-18T14:30:00Z",
  },
  {
    id: 3,
    firstName: "Omar",
    lastName: "Farouk",
    email: "omar.farouk@example.com",
    phone: "+66 92 555-0103",
    status: "quote_sent",
    source: "facebook",
    createdAt: "2026-08-15T11:15:00Z",
  },
  {
    id: 4,
    firstName: "Maria",
    lastName: "Gomez",
    email: "maria.gomez@example.com",
    phone: "+66 84 555-0104",
    status: "booked",
    source: "campaign",
    createdAt: "2026-08-12T16:45:00Z",
  },
  {
    id: 5,
    firstName: "Wei",
    lastName: "Chen",
    email: "wei.chen@example.com",
    phone: "+66 93 555-0105",
    status: "won",
    source: "walk_in",
    createdAt: "2026-08-10T08:20:00Z",
  },
  {
    id: 6,
    firstName: "Aisha",
    lastName: "Khan",
    email: "aisha.khan@example.com",
    phone: "+66 85 555-0106",
    status: "lost",
    source: "website",
    createdAt: "2026-08-08T13:10:00Z",
  },
];

function clone(lead: Lead): Lead {
  return { ...lead };
}

function filterMocks(params: ListParams): Lead[] {
  const q = params.search?.trim().toLowerCase() ?? "";
  let rows = mockLeads.filter((l) => {
    if (params.status && l.status !== params.status) return false;
    if (!q) return true;
    return [l.firstName, l.lastName, l.email, l.phone]
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

export async function getLeads(params: ListParams = {}): Promise<Lead[]> {
  if (USE_MOCKS) {
    await delay(300);
    return filterMocks(params);
  }
  const json = await apiFetch<Lead[] | Page<Lead>>("/api/v1/leads"+toQuery(params as Record<string, string|number|undefined>));
  return unwrapPage(json);
}

export async function getLead(id: number): Promise<Lead> {
  if (USE_MOCKS) {
    await delay(200);
    const lead = mockLeads.find((l) => l.id === id);
    if (!lead) throw new ApiError(404, `Lead ${id} not found`);
    return clone(lead);
  }
  return apiFetch<Lead>(`/api/v1/leads/${id}`);
}

export async function createLead(input: CreateLeadInput): Promise<Lead> {
  if (USE_MOCKS) {
    await delay(400);
    const lead: Lead = {
      ...input,
      lastName: input.lastName ?? "",
      email: input.email ?? "",
      id: ++mockId,
      createdAt: new Date().toISOString(),
    };
    mockLeads.unshift(lead);
    return clone(lead);
  }
  return apiFetch<Lead>("/api/v1/leads", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateLead(
  id: number,
  input: UpdateLeadInput,
): Promise<Lead> {
  if (USE_MOCKS) {
    await delay(400);
    const index = mockLeads.findIndex((l) => l.id === id);
    if (index === -1) throw new ApiError(404, `Lead ${id} not found`);
    const lead = { ...mockLeads[index], ...input };
    mockLeads[index] = lead;
    return clone(lead);
  }
  return apiFetch<Lead>(`/api/v1/leads/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function deleteLead(id: number): Promise<void> {
  if (USE_MOCKS) {
    await delay(300);
    const index = mockLeads.findIndex((l) => l.id === id);
    if (index === -1) throw new ApiError(404, `Lead ${id} not found`);
    mockLeads.splice(index, 1);
    return;
  }
  await apiFetch<void>(`/api/v1/leads/${id}`, { method: "DELETE" });
}
