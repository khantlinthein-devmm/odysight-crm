import { describe, expect, it } from "vitest";
import { hasPermission } from "./roles";
import { createQuote } from "./quotes";
import { createSite } from "./sites";
import { completeChecklistItem, uploadItemPhoto } from "./checklists";
import { getCommercialReport } from "./reports";

describe("commercial domain (mocks)", () => {
  it("creates a site for a customer without a contract", async () => {
    const site = await createSite({
      customerId: 101, name: "Asoke Branch", address: "Asoke 10",
      contactName: "", phone: "", email: "", notes: "", status: "active", isDefault: false,
    });
    expect(site.customerId).toBe(101);
    expect(site.isDefault).toBe(false);
  });

  it("creates a quote without a contract and totals it", async () => {
    const q = await createQuote({
      customerId: 101, items: [{ serviceName: "Lobby", quantity: 2, unitPrice: 500 }],
    });
    expect(q.total).toBe(1000);
    expect(q.status).toBe("draft");
  });

  it("completes checklist items toward completion", async () => {
    const c = await completeChecklistItem(2, true, "Nok");
    expect(["in_progress", "completed"]).toContain(c.status);
  });

  it("attaches before/after photos and loads the commercial report", async () => {
    const file = new File(["x"], "before.jpg", { type: "image/jpeg" });
    const c = await uploadItemPhoto(2, "before", file);
    expect(c.items.find((i) => i.id === 2)?.beforePhotoUrl).toBeTruthy();
    const r = await getCommercialReport();
    expect(r.revenueBySite.length).toBeGreaterThan(0);
    expect(r.quoteWinRate.total).toBeGreaterThan(0);
  });

  it("gates commercial permissions by role", () => {
    expect(hasPermission("ADMIN", "contracts.create")).toBe(true);
    expect(hasPermission("MANAGER", "contracts.create")).toBe(false);
    expect(hasPermission("MANAGER", "quotes.create")).toBe(true);
    expect(hasPermission("ADMIN", "quotes.approve")).toBe(true);
    expect(hasPermission("MANAGER", "quotes.approve")).toBe(false);
    expect(hasPermission("CLEANER", "checklists.manage")).toBe(true);
    expect(hasPermission("DISPATCH", "sites.read")).toBe(true);
    expect(hasPermission("DISPATCH", "sites.create")).toBe(false);
  });
});
