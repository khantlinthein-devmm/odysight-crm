export interface QuoteServiceLine {
  serviceId: string;
  serviceName: string;
  /** Editable base price for this quote (prefilled from catalog). */
  basePrice: number;
  /** Room / area size in square meters. */
  areaSqm: number;
  /** Editable rate per m² for this quote (prefilled from catalog). */
  pricePerSqm: number;
}

export interface QuoteExtraFee {
  id: string;
  label: string;
  amount: number;
}

export interface QuoteInput {
  lines: QuoteServiceLine[];
  equipmentFee: number;
  transportFee: number;
  labourFee: number;
  extraFees: QuoteExtraFee[];
  discount: number;
  taxRatePercent: number;
}

export interface QuoteLineResult extends QuoteServiceLine {
  lineTotal: number;
}

export interface QuoteResult {
  lines: QuoteLineResult[];
  servicesTotal: number;
  feesTotal: number;
  subtotal: number;
  discount: number;
  taxable: number;
  tax: number;
  total: number;
  estimatedMinutes: number;
}

function num(n: unknown): number {
  const v = typeof n === "string" ? Number(n) : (n as number);
  return Number.isFinite(v) && v > 0 ? v : 0;
}

export function lineTotal(line: Pick<QuoteServiceLine, "basePrice" | "areaSqm" | "pricePerSqm">): number {
  return num(line.basePrice) + num(line.areaSqm) * num(line.pricePerSqm);
}

export function emptyQuote(taxRatePercent = 7): QuoteInput {
  return {
    lines: [],
    equipmentFee: 0,
    transportFee: 0,
    labourFee: 0,
    extraFees: [],
    discount: 0,
    taxRatePercent,
  };
}

export function calculateQuote(input: QuoteInput, durationLookup?: (serviceId: string) => number): QuoteResult {
  const lines: QuoteLineResult[] = input.lines.map((l) => ({
    ...l,
    basePrice: num(l.basePrice),
    areaSqm: num(l.areaSqm),
    pricePerSqm: num(l.pricePerSqm),
    lineTotal: lineTotal(l),
  }));
  const servicesTotal = lines.reduce((s, l) => s + l.lineTotal, 0);
  const extrasTotal = input.extraFees.reduce((s, f) => s + num(f.amount), 0);
  const feesTotal =
    num(input.equipmentFee) + num(input.transportFee) + num(input.labourFee) + extrasTotal;
  const subtotal = servicesTotal + feesTotal;
  const discount = Math.min(num(input.discount), subtotal);
  const taxable = subtotal - discount;
  const tax = taxable * (num(input.taxRatePercent) / 100);
  const total = taxable + tax;
  const estimatedMinutes = durationLookup
    ? input.lines.reduce((s, l) => s + (durationLookup(l.serviceId) || 0), 0)
    : 0;
  return { lines, servicesTotal, feesTotal, subtotal, discount, taxable, tax, total, estimatedMinutes };
}

export function buildQuoteSummary(result: QuoteResult, currencyFmt: (n: number) => string): string {
  const parts: string[] = [];
  for (const l of result.lines) {
    const area = l.areaSqm > 0 ? ` ${l.areaSqm}m²` : "";
    parts.push(`${l.serviceName}${area}: ${currencyFmt(l.lineTotal)}`);
  }
  if (result.feesTotal > 0) parts.push(`Fees: ${currencyFmt(result.feesTotal)}`);
  if (result.discount > 0) parts.push(`Discount: -${currencyFmt(result.discount)}`);
  if (result.tax > 0) parts.push(`Tax: ${currencyFmt(result.tax)}`);
  parts.push(`Total: ${currencyFmt(result.total)}`);
  return parts.join("\n");
}

export function newFeeId(): string {
  return `fee_${Date.now().toString(36)}_${Math.floor(Math.random() * 1e6).toString(36)}`;
}
