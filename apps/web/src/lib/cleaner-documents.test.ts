import { describe, expect, it } from "vitest";
import { computeExpiry, maskNumber } from "./cleaner-documents";

describe("cleaner documents helpers", () => {
  it("masks numbers like the API", () => {
    expect(maskNumber("")).toBe("");
    expect(maskNumber("AB12")).toBe("••••");
    expect(maskNumber("MA1234567")).toBe("MA••••567");
  });

  it("classifies expiry against the local calendar date", () => {
    const now = new Date(2026, 9, 10, 23, 30);
    expect(computeExpiry(null, now)).toEqual({ status: "none", days: null });
    expect(computeExpiry("2026-10-09", now)).toEqual({ status: "expired", days: -1 });
    expect(computeExpiry("2026-10-10", now)).toEqual({ status: "expiring", days: 0 });
    expect(computeExpiry("2026-12-09", now)).toEqual({ status: "expiring", days: 60 });
    expect(computeExpiry("2026-12-10", now)).toEqual({ status: "valid", days: 61 });
  });
});
