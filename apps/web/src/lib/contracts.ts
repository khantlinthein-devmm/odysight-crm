import { ApiError, USE_MOCKS, apiFetch, delay, toQuery, unwrapPage, type Page } from "./api";

export type ContractStatus = "draft" | "active" | "expiring" | "expired" | "cancelled" | "renewed";
export type BillingFrequency = "monthly" | "quarterly" | "annual" | "custom";

export interface Contract {
  id: number;
  contractNumber: string;
  customerId: number;
  title: string;
  status: ContractStatus;
  startDate: string;
  endDate: string;
  renewalDate?: string | null;
  contractValue: number;
  billingFrequency: BillingFrequency;
  slaTerms: string;
  notes: string;
  siteIds: number[];
  createdAt: string;
  updatedAt: string;
}

export type CreateContractInput = Omit<Contract, "id" | "contractNumber" | "createdAt" | "updatedAt">;
export type UpdateContractInput = Partial<Omit<CreateContractInput, "customerId">> & { clearSiteIds?: boolean };

let mockId = 700;
const mockContracts: Contract[] = [
  {
    id: 701, contractNumber: "CT-2026-0701", customerId: 101, title: "Sukhumvit + Silom — 12-month",
    status: "active", startDate: "2026-01-01", endDate: "2026-12-31", renewalDate: "2026-12-01",
    contractValue: 480000, billingFrequency: "monthly", slaTerms: "Same-day response",
    notes: "", siteIds: [501, 502],
    createdAt: "2026-08-19T10:00:00Z", updatedAt: "2026-08-19T10:00:00Z",
  },
];

export interface ContractListParams { search?: string; status?: string; customerId?: number; limit?: number; offset?: number }

export async function getContracts(params: ContractListParams = {}): Promise<Contract[]> {
  if (USE_MOCKS) {
    await delay(250);
    const q = params.search?.trim().toLowerCase() ?? "";
    return mockContracts
      .filter((c) => (params.customerId ? c.customerId === params.customerId : true))
      .filter((c) => (params.status ? c.status === params.status : true))
      .filter((c) => (!q || `${c.contractNumber} ${c.title}`.toLowerCase().includes(q)))
      .map((c) => ({ ...c }));
  }
  const json = await apiFetch<Contract[] | Page<Contract>>("/api/v1/contracts" + toQuery(params as Record<string, string | number | undefined>));
  return unwrapPage(json);
}

export async function createContract(input: CreateContractInput): Promise<Contract> {
  if (USE_MOCKS) {
    await delay(300);
    const now = new Date().toISOString();
    const c: Contract = { ...input, id: ++mockId, contractNumber: `CT-2026-0${mockId}`, createdAt: now, updatedAt: now };
    mockContracts.unshift(c);
    return { ...c };
  }
  return apiFetch<Contract>("/api/v1/contracts", { method: "POST", body: JSON.stringify(input) });
}

export async function updateContract(id: number, input: UpdateContractInput): Promise<Contract> {
  if (USE_MOCKS) {
    await delay(300);
    const i = mockContracts.findIndex((c) => c.id === id);
    if (i === -1) throw new ApiError(404, `Contract ${id} not found`);
    mockContracts[i] = { ...mockContracts[i], ...input, updatedAt: new Date().toISOString() };
    return { ...mockContracts[i] };
  }
  return apiFetch<Contract>(`/api/v1/contracts/${id}`, { method: "PATCH", body: JSON.stringify(input) });
}

export async function renewContract(id: number, startDate: string, endDate: string): Promise<Contract> {
  if (USE_MOCKS) {
    await delay(300);
    const src = mockContracts.find((c) => c.id === id);
    if (!src) throw new ApiError(404, `Contract ${id} not found`);
    src.status = "renewed";
    const next: Contract = {
      ...src, id: ++mockId, contractNumber: `CT-2026-0${mockId}`,
      status: "draft", startDate, endDate,
      createdAt: new Date().toISOString(), updatedAt: new Date().toISOString(),
    };
    mockContracts.unshift(next);
    return { ...next };
  }
  return apiFetch<Contract>(`/api/v1/contracts/${id}/renew`, {
    method: "POST", body: JSON.stringify({ startDate, endDate }),
  });
}
