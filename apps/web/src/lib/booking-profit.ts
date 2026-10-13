import { USE_MOCKS, apiFetch, delay } from "./api";

/** One cleaner's estimated pay for a job. Rate/payType/cost are only sent to
 *  roles that may see pay (payroll.read); others get totals only. */
export interface LaborLine {
  cleanerId: number;
  name: string;
  role: "primary" | "crew";
  payType?: "hourly" | "daily" | "per_job" | "monthly";
  rate?: number;
  hours: number;
  cost: number;
  hasRate: boolean;
}

/** Estimate: price (before VAT) − team pay (planned hours × pay rate) − supplies used. */
export interface BookingProfit {
  bookingId: number;
  price: number | null;
  labor: number;
  supplies: number;
  cost: number;
  profit: number | null;
  margin: number | null;
  missingRates: number;
  team: LaborLine[];
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
