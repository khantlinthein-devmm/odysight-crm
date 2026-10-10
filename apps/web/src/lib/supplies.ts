import { apiFetch, toQuery } from "./api";

export const UNITS = ["piece", "bottle", "litre", "kg", "pack", "roll", "box", "gallon"] as const;

export interface Supply {
  id: number;
  name: string;
  unit: string;
  unitCost: number;
  stockQty: number;
  reorderLevel: number;
  active: boolean;
  lowStock: boolean;
  updatedAt: string;
}

export type MovementKind = "purchase" | "usage" | "adjustment";

export interface SupplyMovement {
  id: number;
  supplyId: number;
  supplyName: string;
  unit: string;
  kind: MovementKind;
  /** Stock change: + purchase, − usage, ± adjustment. */
  quantity: number;
  unitCost: number;
  cost: number;
  siteId: number | null;
  siteName: string;
  bookingId: number | null;
  bookingNumber: string;
  note: string;
  createdAt: string;
}

export interface SiteSupplyCost {
  siteId: number;
  siteName: string;
  customerName: string;
  supplyCost: number;
  revenue: number;
  costShare: number;
}

export function getSupplies(all = false): Promise<Supply[]> {
  return apiFetch<Supply[]>("/api/v1/supplies" + toQuery({ all: all ? "true" : undefined }));
}

export function createSupply(input: {
  name: string;
  unit: string;
  unitCost: number;
  stockQty: number;
  reorderLevel: number;
}): Promise<Supply> {
  return apiFetch<Supply>("/api/v1/supplies", { method: "POST", body: JSON.stringify(input) });
}

export function updateSupply(
  id: number,
  input: Partial<Pick<Supply, "name" | "unit" | "unitCost" | "reorderLevel" | "active">>,
): Promise<Supply> {
  return apiFetch<Supply>(`/api/v1/supplies/${id}`, { method: "PATCH", body: JSON.stringify(input) });
}

export function recordMovement(
  id: number,
  input: { kind: MovementKind; quantity: number; unitCost?: number; siteId?: number; bookingId?: number; note?: string },
): Promise<{ movement: SupplyMovement; supply: Supply }> {
  return apiFetch(`/api/v1/supplies/${id}/movements`, { method: "POST", body: JSON.stringify(input) });
}

export function getMovements(params: { from?: string; to?: string; siteId?: number; kind?: string } = {}): Promise<SupplyMovement[]> {
  return apiFetch<SupplyMovement[]>("/api/v1/supplies/movements" + toQuery(params));
}

export function getSiteCosts(from: string, to: string): Promise<SiteSupplyCost[]> {
  return apiFetch<SiteSupplyCost[]>("/api/v1/supplies/site-costs" + toQuery({ from, to }));
}
