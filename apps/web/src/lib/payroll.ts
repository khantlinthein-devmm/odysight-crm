import { ApiError, apiFetch, getApiBaseUrl, toQuery } from "./api";

export type PayType = "hourly" | "daily" | "per_job" | "monthly";

export const PAY_TYPE_LABELS: Record<PayType, string> = {
  hourly: "Per hour",
  daily: "Per day",
  per_job: "Per job",
  monthly: "Monthly salary",
};

export interface PayrollLine {
  cleanerId: number;
  name: string;
  payType: PayType;
  rate: number;
  hasRate: boolean;
  daysWorked: number;
  /** Days checked in but never checked out — not paid until fixed. */
  openDays: number;
  hours: number;
  regularHours: number;
  otHours: number;
  jobs: number;
  basePay: number;
  otPay: number;
  total: number;
}

export interface PayrollReport {
  from: string;
  to: string;
  lines: PayrollLine[];
  total: number;
}

export interface PayRate {
  cleanerId: number;
  payType: PayType;
  rate: number;
  otMultiplier: number;
  standardHours: number;
  updatedAt: string;
}

export function getPayroll(from: string, to: string): Promise<PayrollReport> {
  return apiFetch<PayrollReport>("/api/v1/payroll" + toQuery({ from, to }));
}

export function getPayRates(): Promise<PayRate[]> {
  return apiFetch<PayRate[]>("/api/v1/payroll/rates");
}

export function setPayRate(
  cleanerId: number,
  input: { payType: PayType; rate: number; otMultiplier: number; standardHours: number },
): Promise<PayRate> {
  return apiFetch<PayRate>(`/api/v1/payroll/rates/${cleanerId}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

/** Downloads the payroll CSV for the period (keeps the session cookie). */
export async function downloadPayrollCsv(from: string, to: string): Promise<void> {
  const api = getApiBaseUrl();
  if (!api) throw new ApiError(0, "PUBLIC_API_URL is not configured");
  const res = await fetch(`${api}/api/v1/payroll/export.csv${toQuery({ from, to })}`, { credentials: "include" });
  if (!res.ok) throw new ApiError(res.status, "Failed to export payroll");
  const url = URL.createObjectURL(await res.blob());
  const a = document.createElement("a");
  a.href = url;
  a.download = `payroll_${from}_${to}.csv`;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}
