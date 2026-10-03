import { ApiError, USE_MOCKS, apiFetch, delay, getApiBaseUrl, toQuery, unwrapPage, type Page } from "./api";
import { downloadPdf } from "./invoices";

export type QuoteStatus = "draft" | "sent" | "accepted" | "rejected" | "expired";

export interface QuoteItem {
  id?: number;
  serviceName: string;
  description?: string;
  quantity: number;
  unitPrice: number;
  lineTotal?: number;
  sortOrder?: number;
}

export interface Quote {
  id: number;
  quoteNumber: string;
  customerId: number;
  siteId?: number | null;
  status: QuoteStatus;
  validUntil?: string | null;
  subtotal: number;
  taxRate: number;
  total: number;
  /** Line prices include VAT; subtotal is the value before VAT. */
  pricesIncludeVat?: boolean;
  currency: string;
  notes: string;
  version: number;
  convertedBookingId?: number | null;
  convertedContractId?: number | null;
  items: QuoteItem[];
  createdAt: string;
  updatedAt: string;
}

export interface CreateQuoteInput {
  customerId: number;
  siteId?: number | null;
  status?: QuoteStatus;
  validUntil?: string | null;
  taxRate?: number;
  pricesIncludeVat?: boolean;
  currency?: string;
  notes?: string;
  items: QuoteItem[];
}

let mockId = 800;
const mockQuotes: Quote[] = [
  {
    id: 801, quoteNumber: "QT-2026-0801", customerId: 101, siteId: 501, status: "sent",
    validUntil: "2026-10-15", subtotal: 3000, taxRate: 7, total: 3210, currency: "THB",
    notes: "Weekly lobby cleaning", version: 1,
    items: [{ serviceName: "Lobby cleaning", quantity: 4, unitPrice: 750, lineTotal: 3000 }],
    createdAt: "2026-09-01T10:00:00Z", updatedAt: "2026-09-01T10:00:00Z",
  },
];

export interface QuoteListParams { search?: string; status?: string; customerId?: number; limit?: number; offset?: number }

export async function getQuotes(params: QuoteListParams = {}): Promise<Quote[]> {
  if (USE_MOCKS) {
    await delay(250);
    return mockQuotes
      .filter((q) => (params.customerId ? q.customerId === params.customerId : true))
      .filter((q) => (params.status ? q.status === params.status : true))
      .map((q) => ({ ...q }));
  }
  const json = await apiFetch<Quote[] | Page<Quote>>("/api/v1/quotes" + toQuery(params as Record<string, string | number | undefined>));
  return unwrapPage(json);
}

export async function createQuote(input: CreateQuoteInput): Promise<Quote> {
  if (USE_MOCKS) {
    await delay(300);
    const now = new Date().toISOString();
    const subtotal = input.items.reduce((s, i) => s + i.quantity * i.unitPrice, 0);
    const taxRate = input.taxRate ?? 0;
    const q: Quote = {
      ...input, status: input.status ?? "draft", currency: input.currency ?? "THB",
      notes: input.notes ?? "", subtotal, taxRate, total: subtotal + (subtotal * taxRate) / 100,
      id: ++mockId, quoteNumber: `QT-2026-0${mockId}`, version: 1, createdAt: now, updatedAt: now,
    };
    mockQuotes.unshift(q);
    return { ...q };
  }
  return apiFetch<Quote>("/api/v1/quotes", { method: "POST", body: JSON.stringify(input) });
}

export interface UpdateQuoteInput {
  status?: QuoteStatus;
  notes?: string;
  siteId?: number | null;
  clearSiteId?: boolean;
  validUntil?: string;
  clearValidUntil?: boolean;
  taxRate?: number;
  pricesIncludeVat?: boolean;
  items?: QuoteItem[];
  convertedBookingId?: number;
}

export async function getQuote(id: number): Promise<Quote> {
  if (USE_MOCKS) {
    await delay(150);
    const q = mockQuotes.find((x) => x.id === id);
    if (!q) throw new ApiError(404, `Quote ${id} not found`);
    return { ...q };
  }
  return apiFetch<Quote>(`/api/v1/quotes/${id}`);
}

export async function downloadQuotePdf(q: Pick<Quote, "id" | "quoteNumber">): Promise<void> {
  if (USE_MOCKS) {
    await delay(200);
    return;
  }
  const api = getApiBaseUrl();
  if (!api) throw new ApiError(0, "PUBLIC_API_URL is not configured");
  await downloadPdf(`${api}/api/v1/quotes/${q.id}/pdf`, `${q.quoteNumber}.pdf`);
}

export async function updateQuote(id: number, input: UpdateQuoteInput): Promise<Quote> {
  if (USE_MOCKS) {
    await delay(300);
    const q = mockQuotes.find((x) => x.id === id);
    if (!q) throw new ApiError(404, `Quote ${id} not found`);
    Object.assign(q, input);
    return { ...q };
  }
  return apiFetch<Quote>(`/api/v1/quotes/${id}`, { method: "PATCH", body: JSON.stringify(input) });
}

export async function approveQuote(id: number): Promise<Quote> {  if (USE_MOCKS) {
    await delay(300);
    const q = mockQuotes.find((x) => x.id === id);
    if (!q) throw new ApiError(404, `Quote ${id} not found`);
    q.status = "accepted";
    return { ...q };
  }
  return apiFetch<Quote>(`/api/v1/quotes/${id}/approve`, { method: "POST" });
}
