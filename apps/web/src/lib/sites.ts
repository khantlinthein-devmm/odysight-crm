import { ApiError, USE_MOCKS, apiFetch, delay, toQuery, unwrapPage, type Page } from "./api";

// Commercial cleaning covers more than the residential set customers alone
// were limited to.
export type SitePropertyType =
  | "house"
  | "condo"
  | "apartment"
  | "office"
  | "restaurant"
  | "factory"
  | "mall"
  | "retail"
  | "other";

export type SiteStatus = "active" | "inactive";

export interface Site {
  id: number;
  customerId: number;
  customerName: string;
  name: string;
  address: string;
  area: string;
  propertyType: SitePropertyType;
  // Per-site overrides. Blank means "use the customer's own contact details".
  contactName: string;
  contactPhone: string;
  contactEmail: string;
  notes: string;
  status: SiteStatus;
  lat: number | null;
  lng: number | null;
  createdAt: string;
  updatedAt: string;
}

export interface SiteListParams {
  customer?: number;
  search?: string;
  status?: SiteStatus | "";
  area?: string;
  limit?: number;
  offset?: number;
}

export type CreateSiteInput = Pick<
  Site,
  "customerId" | "name" | "address" | "area" | "propertyType"
> &
  Partial<
    Pick<Site, "contactName" | "contactPhone" | "contactEmail" | "notes" | "status" | "lat" | "lng">
  >;

export type UpdateSiteInput = Partial<Omit<CreateSiteInput, "customerId">>;

let mockId = 900;

const mockSites: Site[] = [
  {
    id: 901,
    customerId: 101,
    customerName: "Somchai Prasert",
    name: "Main site",
    address: "Sukhumvit 38, Klongton, Bangkok",
    area: "Sukhumvit",
    propertyType: "condo",
    contactName: "",
    contactPhone: "",
    contactEmail: "",
    notes: "",
    status: "active",
    lat: null,
    lng: null,
    createdAt: "2026-08-19T10:00:00Z",
    updatedAt: "2026-08-19T10:00:00Z",
  },
  {
    id: 902,
    customerId: 102,
    customerName: "Jane Smith",
    name: "Main site",
    address: "On Nut 10, Suan Luang, Bangkok",
    area: "On Nut",
    propertyType: "house",
    contactName: "",
    contactPhone: "",
    contactEmail: "",
    notes: "",
    status: "active",
    lat: null,
    lng: null,
    createdAt: "2026-08-17T09:30:00Z",
    updatedAt: "2026-08-17T09:30:00Z",
  },
];

function clone(site: Site): Site {
  return { ...site };
}

function filterMocks(params: SiteListParams): Site[] {
  const q = params.search?.trim().toLowerCase() ?? "";
  let rows = mockSites.filter((s) => {
    if (params.customer !== undefined && s.customerId !== params.customer) return false;
    if (params.status && s.status !== params.status) return false;
    if (params.area && s.area !== params.area) return false;
    if (!q) return true;
    return [s.name, s.address, s.customerName].join(" ").toLowerCase().includes(q);
  });
  rows = [...rows].sort((a, b) => a.customerId - b.customerId || a.name.localeCompare(b.name));
  const offset = params.offset ?? 0;
  const limit = params.limit ?? rows.length;
  rows = rows.slice(offset, offset + limit);
  return rows.map(clone);
}

export async function getSites(params: SiteListParams = {}): Promise<Site[]> {
  if (USE_MOCKS) {
    await delay(250);
    return filterMocks(params);
  }
  const json = await apiFetch<Site[] | Page<Site>>(
    "/api/v1/sites" + toQuery(params as Record<string, string | number | undefined>),
  );
  return unwrapPage(json);
}

export async function getSite(id: number): Promise<Site> {
  if (USE_MOCKS) {
    await delay(150);
    const site = mockSites.find((s) => s.id === id);
    if (!site) throw new ApiError(404, `Site ${id} not found`);
    return clone(site);
  }
  return apiFetch<Site>(`/api/v1/sites/${id}`);
}

export async function createSite(input: CreateSiteInput): Promise<Site> {
  if (USE_MOCKS) {
    await delay(350);
    const owner = mockSites.find((s) => s.customerId === input.customerId);
    const site: Site = {
      contactName: "",
      contactPhone: "",
      contactEmail: "",
      notes: "",
      status: "active",
      lat: null,
      lng: null,
      ...input,
      customerName: owner?.customerName ?? `Customer ${input.customerId}`,
      id: ++mockId,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    mockSites.unshift(site);
    return clone(site);
  }
  return apiFetch<Site>("/api/v1/sites", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateSite(id: number, input: UpdateSiteInput): Promise<Site> {
  if (USE_MOCKS) {
    await delay(300);
    const index = mockSites.findIndex((s) => s.id === id);
    if (index === -1) throw new ApiError(404, `Site ${id} not found`);
    const site = { ...mockSites[index]!, ...input, updatedAt: new Date().toISOString() };
    mockSites[index] = site;
    return clone(site);
  }
  return apiFetch<Site>(`/api/v1/sites/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function deleteSite(id: number): Promise<void> {
  if (USE_MOCKS) {
    await delay(250);
    const index = mockSites.findIndex((s) => s.id === id);
    if (index === -1) throw new ApiError(404, `Site ${id} not found`);
    mockSites.splice(index, 1);
    return;
  }
  await apiFetch<void>(`/api/v1/sites/${id}`, { method: "DELETE" });
}
