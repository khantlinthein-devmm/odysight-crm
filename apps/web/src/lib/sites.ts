import { ApiError, USE_MOCKS, apiFetch, delay, toQuery, unwrapPage, type Page } from "./api";

export type SiteStatus = "active" | "inactive";

export interface Site {
  id: number;
  customerId: number;
  name: string;
  address: string;
  contactName: string;
  phone: string;
  email: string;
  notes: string;
  status: SiteStatus;
  latitude?: number | null;
  longitude?: number | null;
  isDefault: boolean;
  createdAt: string;
  updatedAt: string;
}

export type CreateSiteInput = Omit<Site, "id" | "createdAt" | "updatedAt">;
export type UpdateSiteInput = Partial<Omit<CreateSiteInput, "customerId">>;

let mockId = 500;
const mockSites: Site[] = [
  {
    id: 501, customerId: 101, name: "Default Site", address: "Sukhumvit 38, Klongton, Bangkok",
    contactName: "Somchai Prasert", phone: "+66 91 234 5678", email: "somchai.prasert@example.com",
    notes: "", status: "active", isDefault: true,
    createdAt: "2026-08-19T10:00:00Z", updatedAt: "2026-08-19T10:00:00Z",
  },
  {
    id: 502, customerId: 101, name: "Silom Branch", address: "Silom 5, Bangrak, Bangkok",
    contactName: "Somchai Prasert", phone: "+66 91 234 5678", email: "",
    notes: "Night access via rear door", status: "active", isDefault: false,
    createdAt: "2026-08-21T10:00:00Z", updatedAt: "2026-08-21T10:00:00Z",
  },
];

export interface SiteListParams { search?: string; status?: string; customerId?: number; limit?: number; offset?: number }

export async function getSites(params: SiteListParams = {}): Promise<Site[]> {
  if (USE_MOCKS) {
    await delay(250);
    const q = params.search?.trim().toLowerCase() ?? "";
    return mockSites
      .filter((s) => (params.customerId ? s.customerId === params.customerId : true))
      .filter((s) => (params.status ? s.status === params.status : true))
      .filter((s) => (!q || `${s.name} ${s.address}`.toLowerCase().includes(q)))
      .map((s) => ({ ...s }));
  }
  const json = await apiFetch<Site[] | Page<Site>>("/api/v1/sites" + toQuery(params as Record<string, string | number | undefined>));
  return unwrapPage(json);
}

export async function createSite(input: CreateSiteInput): Promise<Site> {
  if (USE_MOCKS) {
    await delay(300);
    const now = new Date().toISOString();
    const site: Site = { ...input, id: ++mockId, createdAt: now, updatedAt: now };
    mockSites.unshift(site);
    return { ...site };
  }
  return apiFetch<Site>("/api/v1/sites", { method: "POST", body: JSON.stringify(input) });
}

export async function updateSite(id: number, input: UpdateSiteInput): Promise<Site> {
  if (USE_MOCKS) {
    await delay(300);
    const i = mockSites.findIndex((s) => s.id === id);
    if (i === -1) throw new ApiError(404, `Site ${id} not found`);
    mockSites[i] = { ...mockSites[i], ...input, updatedAt: new Date().toISOString() };
    return { ...mockSites[i] };
  }
  return apiFetch<Site>(`/api/v1/sites/${id}`, { method: "PATCH", body: JSON.stringify(input) });
}

export async function deleteSite(id: number): Promise<void> {
  if (USE_MOCKS) {
    await delay(250);
    const i = mockSites.findIndex((s) => s.id === id);
    if (i === -1) throw new ApiError(404, `Site ${id} not found`);
    mockSites.splice(i, 1);
    return;
  }
  await apiFetch<void>(`/api/v1/sites/${id}`, { method: "DELETE" });
}
