import { describe, expect, it } from "vitest";
import { calculateQuote, lineTotal } from "./pricing";

describe("lineTotal", () => {
  it("adds base + sqm × rate", () => {
    expect(lineTotal({ basePrice: 1200, areaSqm: 80, pricePerSqm: 25 })).toBe(3200);
  });
  it("treats blanks/negatives as zero", () => {
    expect(lineTotal({ basePrice: -5, areaSqm: NaN, pricePerSqm: 25 })).toBe(0);
  });
});

describe("calculateQuote", () => {
  it("sums multiple services plus equipment/transport/labour and custom fees", () => {
    const r = calculateQuote({
      lines: [
        { serviceId: "a", serviceName: "A", basePrice: 1000, areaSqm: 50, pricePerSqm: 20 },
        { serviceId: "b", serviceName: "B", basePrice: 500, areaSqm: 0, pricePerSqm: 0 },
      ],
      equipmentFee: 200,
      transportFee: 150,
      labourFee: 300,
      extraFees: [{ id: "1", label: "Parking", amount: 100 }],
      discount: 0,
      taxRatePercent: 0,
    });
    // (1000+1000) + 500 = 2500 services; fees 200+150+300+100=750
    expect(r.servicesTotal).toBe(2500);
    expect(r.feesTotal).toBe(750);
    expect(r.subtotal).toBe(3250);
    expect(r.total).toBe(3250);
  });

  it("applies discount before tax and clamps discount to subtotal", () => {
    const r = calculateQuote({
      lines: [{ serviceId: "a", serviceName: "A", basePrice: 1000, areaSqm: 0, pricePerSqm: 0 }],
      equipmentFee: 0,
      transportFee: 0,
      labourFee: 0,
      extraFees: [],
      discount: 5000,
      taxRatePercent: 7,
    });
    expect(r.discount).toBe(1000);
    expect(r.taxable).toBe(0);
    expect(r.total).toBe(0);
  });

  it("computes 7% tax after discount", () => {
    const r = calculateQuote({
      lines: [{ serviceId: "a", serviceName: "A", basePrice: 1000, areaSqm: 10, pricePerSqm: 10 }],
      equipmentFee: 0,
      transportFee: 0,
      labourFee: 0,
      extraFees: [],
      discount: 100,
      taxRatePercent: 7,
    });
    // subtotal 1100, discount 100 → taxable 1000, tax 70, total 1070
    expect(r.total).toBeCloseTo(1070, 6);
  });
});
