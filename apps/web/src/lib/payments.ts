import { ApiError, USE_MOCKS, apiFetch, delay } from "./api";

export type PaymentStatus = "pending" | "paid" | "failed" | "refunded";

export interface Payment {
  id: number;
  invoiceNumber: string;
  payerName: string;
  amount: number;
  currency: string;
  method: string;
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
    payerName: "John Doe",
    amount: 1500,
    currency: "USD",
    method: "Credit Card",
    status: "paid",
    createdAt: "2026-08-19T12:00:00Z",
  },
  {
    id: 402,
    invoiceNumber: "INV-2026-0402",
    payerName: "Jane Smith",
    amount: 2200,
    currency: "USD",
    method: "Bank Transfer",
    status: "pending",
    createdAt: "2026-08-17T10:30:00Z",
  },
  {
    id: 403,
    invoiceNumber: "INV-2026-0403",
    payerName: "Omar Farouk",
    amount: 800,
    currency: "USD",
    method: "Credit Card",
    status: "failed",
    createdAt: "2026-08-14T09:15:00Z",
  },
  {
    id: 404,
    invoiceNumber: "INV-2026-0404",
    payerName: "Maria Gomez",
    amount: 3100,
    currency: "USD",
    method: "Bank Transfer",
    status: "paid",
    createdAt: "2026-08-11T15:45:00Z",
  },
  {
    id: 405,
    invoiceNumber: "INV-2026-0405",
    payerName: "Wei Chen",
    amount: 1200,
    currency: "USD",
    method: "Credit Card",
    status: "refunded",
    createdAt: "2026-08-06T11:20:00Z",
  },
];

function clone(payment: Payment): Payment {
  return { ...payment };
}

export async function getPayments(): Promise<Payment[]> {
  if (USE_MOCKS) {
    await delay(300);
    return mockPayments.map(clone);
  }
  return apiFetch<Payment[]>("/api/v1/payments");
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
