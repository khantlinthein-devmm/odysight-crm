import {
  ApiError,
  USE_MOCKS,
  apiFetch,
  delay,
  getApiBaseUrl,
  toQuery,
  unwrapPage,
  type Page,
} from "./api";
import { DEFAULT_SETTINGS } from "./settings";

export type InvoiceStatus = "draft" | "issued" | "paid" | "void";

export interface Invoice {
  id: number;
  invoiceNumber: string;
  bookingId: number;
  bookingNumber: string;
  customerName: string;
  customerEmail: string;
  address: string;
  serviceType: string;
  serviceName: string;
  subtotal: number;
  taxRate: number;
  taxAmount: number;
  total: number;
  currency: string;
  status: InvoiceStatus;
  /** Tax-invoice snapshot taken when the invoice was issued. */
  customerTaxId?: string;
  customerTaxBranch?: string;
  withholdingRate?: number;
  withholdingAmount?: number;
  /** Total minus withholding tax — what the customer actually transfers. */
  netPayable?: number;
  contractId?: number | null;
  idempotencyKey?: string | null;
  billingPeriodStart?: string | null;
  billingPeriodEnd?: string | null;
  issuedAt: string;
  paidAt: string | null;
  createdAt: string;
}

export type CreateInvoiceInput = {
  bookingId: number;
  subtotal?: number;
  contractId?: number | null;
  idempotencyKey?: string;
  billingPeriodStart?: string;
  billingPeriodEnd?: string;
};

export type UpdateInvoiceInput = { status: InvoiceStatus };

let mockId = 700;

// Mirrors mockBookings[204] (completed booking) so the demo invoice is real.
function mockCatalog() {
  const items = DEFAULT_SETTINGS.services ?? [];
  const taxRate = DEFAULT_SETTINGS.payments?.taxRatePercent ?? 7;
  const currency = DEFAULT_SETTINGS.localization?.currency ?? "THB";
  return { items, taxRate, currency };
}

const mockInvoices: Invoice[] = (() => {
  const { items, taxRate, currency } = mockCatalog();
  const service = items[0];
  const bookingNumber = "BK-2026-0204";
  const subtotal = Math.round(service.basePrice * 100) / 100;
  const taxAmount = Math.round(subtotal * taxRate * 100) / 10000;
  return [
    {
      id: 701,
      invoiceNumber: "INV-2026-0001",
      bookingId: 204,
      bookingNumber,
      customerName: "Maria Gomez",
      customerEmail: "maria@example.com",
      address: "Asok 4, Khlong Toei, Bangkok",
      serviceType: service.id,
      serviceName: service.name,
      subtotal,
      taxRate,
      taxAmount,
      total: Math.round((subtotal + taxAmount) * 100) / 100,
      currency,
      status: "issued",
      issuedAt: "2026-09-04T09:00:00Z",
      paidAt: null,
      createdAt: "2026-09-04T09:00:00Z",
    },
  ];
})();

export type InvoiceListParams = {
  search?: string;
  status?: InvoiceStatus;
  limit?: number;
  offset?: number;
};

export async function getInvoices(
  params: InvoiceListParams = {},
): Promise<Invoice[]> {
  if (USE_MOCKS) {
    await delay(300);
    let rows = mockInvoices.filter((inv) => {
      if (params.status && inv.status !== params.status) return false;
      const q = params.search?.trim().toLowerCase() ?? "";
      if (!q) return true;
      return [inv.invoiceNumber, inv.customerName, inv.bookingNumber]
        .join(" ")
        .toLowerCase()
        .includes(q);
    });
    const offset = params.offset ?? 0;
    const limit = params.limit ?? rows.length;
    rows = rows.slice(offset, offset + limit);
    return rows.map((inv) => ({ ...inv }));
  }
  const json = await apiFetch<Invoice[] | Page<Invoice>>(
    "/api/v1/invoices" + toQuery(params as Record<string, string | number | undefined>),
  );
  return unwrapPage(json);
}

export async function getInvoice(id: number): Promise<Invoice> {
  if (USE_MOCKS) {
    await delay(200);
    const inv = mockInvoices.find((i) => i.id === id);
    if (!inv) throw new ApiError(404, `Invoice ${id} not found`);
    return { ...inv };
  }
  return apiFetch<Invoice>(`/api/v1/invoices/${id}`);
}

export async function createInvoice(input: CreateInvoiceInput): Promise<Invoice> {
  if (USE_MOCKS) {
    await delay(400);
    const { items, taxRate, currency } = mockCatalog();
    const service = items[0];
    const subtotal =
      input.subtotal !== undefined ? input.subtotal : service.basePrice;
    const taxAmount = Math.round(subtotal * taxRate * 100) / 10000;
    const inv: Invoice = {
      id: ++mockId,
      invoiceNumber: `INV-2026-${String(mockId).padStart(4, "0")}`,
      bookingId: input.bookingId,
      bookingNumber: "BK-2026-0204",
      customerName: "Maria Gomez",
      customerEmail: "maria@example.com",
      address: "Asok 4, Khlong Toei, Bangkok",
      serviceType: service.id,
      serviceName: service.name,
      subtotal,
      taxRate,
      taxAmount,
      total: Math.round((subtotal + taxAmount) * 100) / 100,
      currency,
      status: "issued",
      issuedAt: new Date().toISOString(),
      paidAt: null,
      createdAt: new Date().toISOString(),
    };
    mockInvoices.unshift(inv);
    return { ...inv };
  }
  return apiFetch<Invoice>("/api/v1/invoices", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateInvoice(
  id: number,
  input: UpdateInvoiceInput,
): Promise<Invoice> {
  if (USE_MOCKS) {
    await delay(300);
    const index = mockInvoices.findIndex((i) => i.id === id);
    if (index === -1) throw new ApiError(404, `Invoice ${id} not found`);
    const inv = { ...mockInvoices[index], ...input };
    if (input.status === "paid" && !inv.paidAt) {
      inv.paidAt = new Date().toISOString();
    }
    mockInvoices[index] = inv;
    return { ...inv };
  }
  return apiFetch<Invoice>(`/api/v1/invoices/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

/**
 * Object URL of the invoice's PromptPay QR (PNG), or null when PromptPay is
 * not configured. Callers should URL.revokeObjectURL it when done.
 */
export async function getPromptPayQrUrl(id: number): Promise<string | null> {
  if (USE_MOCKS) return null;
  const api = getApiBaseUrl();
  if (!api) return null;
  const response = await fetch(`${api}/api/v1/invoices/${id}/promptpay.png`, {
    credentials: "include",
  });
  if (!response.ok) return null;
  return URL.createObjectURL(await response.blob());
}

export async function downloadInvoicePdf(id: number, fileName?: string): Promise<void> {
  if (USE_MOCKS) {
    await delay(300);
    return;
  }
  const api = getApiBaseUrl();
  if (!api) throw new ApiError(0, "PUBLIC_API_URL is not configured");
  const response = await fetch(`${api}/api/v1/invoices/${id}/pdf`, {
    credentials: "include",
  });
  if (!response.ok) {
    throw new ApiError(response.status, "Failed to download invoice PDF");
  }
  const blob = await response.blob();
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = fileName || `invoice-${id}.pdf`;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}

export async function sendInvoiceEmail(id: number): Promise<void> {
  if (USE_MOCKS) {
    await delay(300);
    return;
  }
  await apiFetch<{ status: string }>(`/api/v1/invoices/${id}/email`, {
    method: "POST",
  });
}