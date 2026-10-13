import { describe, expect, it } from "vitest";
import { previewProfit } from "./booking-profit";

describe("previewProfit", () => {
  it("subtracts cleaner fees, transport and other costs from the price", () => {
    const r = previewProfit(3000, { teamFee: null, cleanerFees: { 1: 600, 2: 400 }, transport: 150, other: 70, otherNote: "" });
    expect(r).toEqual({ cost: 1220, profit: 1780, margin: 59.33 });
  });

  it("uses the team fee instead of per-cleaner fees when set", () => {
    const r = previewProfit(5000, { teamFee: 2000, cleanerFees: { 1: 999 }, transport: 100, other: 0, otherNote: "" });
    expect(r).toEqual({ cost: 2100, profit: 2900, margin: 58 });
  });

  it("shows a loss and no profit without a price", () => {
    expect(previewProfit(500, { teamFee: null, cleanerFees: { 1: 600 }, transport: 0, other: 0, otherNote: "" })).toEqual({
      cost: 600,
      profit: -100,
      margin: -20,
    });
    expect(previewProfit(null, { teamFee: null, cleanerFees: {}, transport: 100, other: 0, otherNote: "" })).toEqual({
      cost: 100,
      profit: null,
      margin: null,
    });
  });
});
