import { ApiError, USE_MOCKS, apiFetch, delay, toQuery, unwrapPage, type Page } from "./api";

export type PaymentStatus = "pending" | "paid" | "failed" | "refunded";

// Payment methods are admin-editable via Settings → Payments,
// so this is an open string (the API no longer enforces a fixed enum).
export type PaymentMethod = string;

export interface Payment {
  id: number;
  invoiceNumber: string;
  customerName: string;
  bookingNumber: string;
  amount: number;
  currency: string;
  method: PaymentMethod;
  status: PaymentStatus;
  createdAt: string;
}

export type CreatePaymentInput = Omit<
  Payment,
  "id" | "invoiceNumber" | "createdAt"
>;

export type UpdatePaymentInput = Partial<Pick<Payment, "status" | "method">>;

let mockId = 500;

const mockPayments: Payment[] = [
  {
    id: 401,
    invoiceNumber: "INV-2026-0401",
    customerName: "Somchai Prasert",
    bookingNumber: "BK-2026-0201",
    amount: 1500,
    currency: "THB",
    method: "promptpay",
    status: "paid",
    createdAt: "2026-08-19T12:00:00Z",
  },
  {
    id: 402,
    invoiceNumber: "INV-2026-0402",
    customerName: "Jane Smith",
    bookingNumber: "BK-2026-0202",
    amount: 2200,
    currency: "THB",
    method: "bank_transfer",
    status: "pending",
    createdAt: "2026-08-17T10:30:00Z",
  },
  {
    id: 403,
    invoiceNumber: "INV-2026-0403",
    customerName: "Omar Farouk",
    bookingNumber: "BK-2026-0203",
    amount: 800,
    currency: "THB",
    method: "cash",
    status: "failed",
    createdAt: "2026-08-14T09:15:00Z",
  },
  {
    id: 404,
    invoiceNumber: "INV-2026-0404",
    customerName: "Maria Gomez",
    bookingNumber: "BK-2026-0204",
    amount: 3100,
    currency: "THB",
    method: "credit_card",
    status: "paid",
    createdAt: "2026-08-11T15:45:00Z",
  },
  {
    id: 405,
    invoiceNumber: "INV-2026-0405",
    customerName: "Wei Chen",
    bookingNumber: "BK-2026-0205",
    amount: 1200,
    currency: "THB",
    method: "line_pay",
    status: "refunded",
    createdAt: "2026-08-06T11:20:00Z",
  },
];

function clone(payment: Payment): Payment {
  return { ...payment };
}

function filterMocks(params: ListParams): Payment[] {
  const q = params.search?.trim().toLowerCase() ?? "";
  let rows = mockPayments.filter((p) => {
    if (params.status && p.status !== params.status) return false;
    if (!q) return true;
    return [p.customerName, p.invoiceNumber, p.bookingNumber]
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

export async function getPayments(params: ListParams = {}): Promise<Payment[]> {
  if (USE_MOCKS) {
    await delay(300);
    return filterMocks(params);
  }
  const json = await apiFetch<Payment[] | Page<Payment>>("/api/v1/payments"+toQuery(params as Record<string, string|number|undefined>));
  return unwrapPage(json);
}

export async function getPayment(id: number): Promise<Payment> {
  if (USE_MOCKS) {
    await delay(200);
    const payment = mockPayments.find((p) => p.id === id);
    if (!payment) throw new ApiError(404, `Payment ${id} not found`);
    return clone(payment);
  }
  return apiFetch<Payment>(`/api/v1/payments/${id}`);
}

export async function createPayment(
  input: CreatePaymentInput,
): Promise<Payment> {
  if (USE_MOCKS) {
    await delay(400);
    const payment: Payment = {
      ...input,
      id: ++mockId,
      invoiceNumber: `INV-2026-0${mockId}`,
      createdAt: new Date().toISOString(),
    };
    mockPayments.unshift(payment);
    return clone(payment);
  }
  return apiFetch<Payment>("/api/v1/payments", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updatePayment(
  id: number,
  input: UpdatePaymentInput,
): Promise<Payment> {
  if (USE_MOCKS) {
    await delay(300);
    const index = mockPayments.findIndex((p) => p.id === id);
    if (index === -1) throw new ApiError(404, `Payment ${id} not found`);
    const payment = { ...mockPayments[index], ...input };
    mockPayments[index] = payment;
    return clone(payment);
  }
  return apiFetch<Payment>(`/api/v1/payments/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}
