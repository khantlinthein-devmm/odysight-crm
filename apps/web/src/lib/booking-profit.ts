import { USE_MOCKS, apiFetch, delay } from "./api";

/** One team member's fee for the job, as entered on the booking. */
export interface LaborLine {
  cleanerId: number;
  name: string;
  role: "primary" | "crew";
  fee: number;
  hasFee: boolean;
}

/** price (before VAT) − cleaner fees − transportation − other − supplies used. */
export interface BookingProfit {
  bookingId: number;
  price: number | null;
  labor: number;
  transport: number;
  other: number;
  otherNote: string;
  supplies: number;
  cost: number;
  profit: number | null;
  margin: number | null;
  missingFees: number;
  team: LaborLine[];
}

/** Costs typed on the booking form. Fees are keyed by cleaner id. */
export interface BookingCostsInput {
  cleanerFees: Record<number, number>;
  transport: number;
  other: number;
  otherNote: string;
}

/** What the booking form holds while typing ("" = empty input). */
export interface CostDraft {
  fees: Record<number, number | "">;
  transport: number | "";
  other: number | "";
  otherNote: string;
}

const MAX_IDS = 200;

export async function getBookingProfits(ids: number[]): Promise<Record<number, BookingProfit>> {
  const unique = [...new Set(ids)].filter((id) => id > 0);
  const out: Record<number, BookingProfit> = {};
  if (unique.length === 0 || USE_MOCKS) {
    if (USE_MOCKS) await delay(150);
    return out;
  }
  for (let i = 0; i < unique.length; i += MAX_IDS) {
    const chunk = unique.slice(i, i + MAX_IDS);
    const rows = await apiFetch<BookingProfit[]>(`/api/v1/reports/booking-profit?ids=${chunk.join(",")}`);
    for (const row of rows) out[row.bookingId] = row;
  }
  return out;
}

export async function getBookingProfit(id: number): Promise<BookingProfit | null> {
  const map = await getBookingProfits([id]);
  return map[id] ?? null;
}

export async function saveBookingCosts(bookingId: number, costs: BookingCostsInput): Promise<BookingProfit | null> {
  if (USE_MOCKS) {
    await delay(150);
    return null;
  }
  return apiFetch<BookingProfit>(`/api/v1/reports/booking-costs/${bookingId}`, {
    method: "PUT",
    body: JSON.stringify(costs),
  });
}

/** Profit preview while typing (supplies are added server-side later). */
export function previewProfit(price: number | null, costs: BookingCostsInput): {
  cost: number;
  profit: number | null;
  margin: number | null;
} {
  const fees = Object.values(costs.cleanerFees).reduce((s, v) => s + (Number(v) || 0), 0);
  const cost = Math.round((fees + (Number(costs.transport) || 0) + (Number(costs.other) || 0)) * 100) / 100;
  if (price == null) return { cost, profit: null, margin: null };
  const profit = Math.round((price - cost) * 100) / 100;
  const margin = price > 0 ? Math.round((profit / price) * 10000) / 100 : null;
  return { cost, profit, margin };
}
