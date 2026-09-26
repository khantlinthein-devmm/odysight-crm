import { ApiError, USE_MOCKS, apiFetch, delay, toQuery, unwrapPage, type Page } from "./api";

export type CleanerStatus =
  | "available"
  | "assigned"
  | "on_leave"
  | "inactive";

export interface Cleaner {
  id: number;
  firstName: string;
  lastName: string;
  phone: string;
  email: string;
  lineId: string;
  skills: string;
  status: CleanerStatus;
  createdAt: string;
}

export type CreateCleanerInput = Omit<
  Cleaner,
  "id" | "createdAt" | "lastName" | "email" | "lineId"
> & { lastName?: string; email?: string; lineId?: string };

export type UpdateCleanerInput = Partial<CreateCleanerInput>;

let mockId = 700;

const mockCleaners: Cleaner[] = [
  {
    id: 601,
    firstName: "Nok",
    lastName: "Srisuwan",
    phone: "+66 81 111 2233",
    email: "nok.s@smileclean.com",
    lineId: "nok_clean",
    skills: "Deep Cleaning, Condo",
    status: "available",
    createdAt: "2026-08-15T09:00:00Z",
  },
  {
    id: 602,
    firstName: "Pim",
    lastName: "Jiraroj",
    phone: "+66 82 222 3344",
    email: "pim.j@smileclean.com",
    lineId: "pimjira",
    skills: "House Cleaning, Aircon",
    status: "assigned",
    createdAt: "2026-08-15T09:05:00Z",
  },
  {
    id: 603,
    firstName: "Daeng",
    lastName: "Chaiya",
    phone: "+66 83 333 4455",
    email: "daeng.c@smileclean.com",
    lineId: "",
    skills: "Office Cleaning, Deep Cleaning",
    status: "available",
    createdAt: "2026-08-15T09:10:00Z",
  },
  {
    id: 604,
    firstName: "Mali",
    lastName: "Kongdee",
    phone: "+66 84 444 5566",
    email: "mali.k@smileclean.com",
    lineId: "mali.k",
    skills: "Move In/Out, Junk Removal",
    status: "on_leave",
    createdAt: "2026-08-15T09:15:00Z",
  },
  {
    id: 605,
    firstName: "Som",
    lastName: "Intarakamhaeng",
    phone: "+66 85 555 6677",
    email: "som.i@smileclean.com",
    lineId: "",
    skills: "General Cleaning",
    status: "available",
    createdAt: "2026-08-15T09:20:00Z",
  },
];

function clone(cleaner: Cleaner): Cleaner {
  return { ...cleaner };
}

function filterMocks(params: ListParams): Cleaner[] {
  const q = params.search?.trim().toLowerCase() ?? "";
  let rows = mockCleaners.filter((c) => {
    if (params.status && c.status !== params.status) return false;
    if (!q) return true;
    return [c.firstName, c.lastName, c.email, c.lineId, c.phone, c.skills]
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

export async function getCleaners(params: ListParams = {}): Promise<Cleaner[]> {
  if (USE_MOCKS) {
    await delay(300);
    return filterMocks(params);
  }
  const json = await apiFetch<Cleaner[] | Page<Cleaner>>("/api/v1/cleaners"+toQuery(params as Record<string, string|number|undefined>));
  return unwrapPage(json);
}

export async function getCleaner(id: number): Promise<Cleaner> {
  if (USE_MOCKS) {
    await delay(200);
    const cleaner = mockCleaners.find((c) => c.id === id);
    if (!cleaner) throw new ApiError(404, `Cleaner ${id} not found`);
    return clone(cleaner);
  }
  return apiFetch<Cleaner>(`/api/v1/cleaners/${id}`);
}

/** The cleaner profile linked to the signed-in login (null when none). */
export async function getMyCleanerProfile(): Promise<Cleaner | null> {
  if (USE_MOCKS) {
    await delay(200);
    return null;
  }
  try {
    return await apiFetch<Cleaner>("/api/v1/cleaners/me");
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null;
    throw err;
  }
}

export async function createCleaner(
  input: CreateCleanerInput,
): Promise<Cleaner> {
  if (USE_MOCKS) {
    await delay(400);
    const cleaner: Cleaner = {
      ...input,
      lastName: input.lastName ?? "",
      email: input.email ?? "",
      lineId: input.lineId ?? "",
      id: ++mockId,
      createdAt: new Date().toISOString(),
    };
    mockCleaners.unshift(cleaner);
    return clone(cleaner);
  }
  return apiFetch<Cleaner>("/api/v1/cleaners", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateCleaner(
  id: number,
  input: UpdateCleanerInput,
): Promise<Cleaner> {
  if (USE_MOCKS) {
    await delay(400);
    const index = mockCleaners.findIndex((c) => c.id === id);
    if (index === -1) throw new ApiError(404, `Cleaner ${id} not found`);
    const cleaner = { ...mockCleaners[index], ...input };
    mockCleaners[index] = cleaner;
    return clone(cleaner);
  }
  return apiFetch<Cleaner>(`/api/v1/cleaners/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function deleteCleaner(id: number): Promise<void> {
  if (USE_MOCKS) {
    await delay(300);
    const index = mockCleaners.findIndex((c) => c.id === id);
    if (index === -1) throw new ApiError(404, `Cleaner ${id} not found`);
    mockCleaners.splice(index, 1);
    delete mockPhones[id];
    return;
  }
  await apiFetch<void>(`/api/v1/cleaners/${id}`, { method: "DELETE" });
}

export interface PhoneNumber {
  id: number;
  label: string;
  phone: string;
}

let mockPhoneId = 810;

const mockPhones: Record<number, PhoneNumber[]> = {
  601: [
    { id: 801, label: "Emergency", phone: "+66 89 999 0000" },
    { id: 802, label: "Home", phone: "+66 2 123 4567" },
  ],
};

function clonePhone(phone: PhoneNumber): PhoneNumber {
  return { ...phone };
}

function requireMockCleaner(id: number): void {
  if (!mockCleaners.some((c) => c.id === id)) {
    throw new ApiError(404, `Cleaner ${id} not found`);
  }
}

export async function getCleanerPhones(id: number): Promise<PhoneNumber[]> {
  if (USE_MOCKS) {
    await delay(200);
    requireMockCleaner(id);
    return (mockPhones[id] ?? []).map(clonePhone);
  }
  return apiFetch<PhoneNumber[]>(`/api/v1/cleaners/${id}/phones`);
}

export async function addCleanerPhone(
  id: number,
  input: { label: string; phone: string },
): Promise<PhoneNumber> {
  if (USE_MOCKS) {
    await delay(300);
    requireMockCleaner(id);
    if (!input.phone.trim()) throw new ApiError(400, "Phone is required");
    const record: PhoneNumber = {
      id: ++mockPhoneId,
      label: input.label,
      phone: input.phone,
    };
    (mockPhones[id] ??= []).push(record);
    return clonePhone(record);
  }
  return apiFetch<PhoneNumber>(`/api/v1/cleaners/${id}/phones`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function removeCleanerPhone(
  id: number,
  phoneId: number,
): Promise<void> {
  if (USE_MOCKS) {
    await delay(300);
    requireMockCleaner(id);
    const rows = mockPhones[id] ?? [];
    const index = rows.findIndex((p) => p.id === phoneId);
    if (index === -1) throw new ApiError(404, `Phone ${phoneId} not found`);
    rows.splice(index, 1);
    return;
  }
  await apiFetch<void>(`/api/v1/cleaners/${id}/phones/${phoneId}`, {
    method: "DELETE",
  });
}
