import { describe, expect, it } from "vitest";
import { hasPermission, permissionsFor, ROLE_ORDER } from "./roles";

describe("roles permission matrix", () => {
  it("grants settings.manage only to SUPER_ADMIN and ADMIN", () => {
    expect(hasPermission("SUPER_ADMIN", "settings.manage")).toBe(true);
    expect(hasPermission("ADMIN", "settings.manage")).toBe(true);
    expect(hasPermission("MANAGER", "settings.manage")).toBe(false);
    expect(hasPermission("DISPATCH", "settings.manage")).toBe(false);
    expect(hasPermission(undefined, "settings.manage")).toBe(false);
  });

  it("rejects unknown permission keys", () => {
    expect(hasPermission("SUPER_ADMIN", "books.ban")).toBe(false);
  });

  it("gates bookings.create to dispatchers and above", () => {
    expect(hasPermission("DISPATCH", "bookings.create")).toBe(true);
    expect(hasPermission("CLEANER", "bookings.create")).toBe(false);
    expect(hasPermission("ACCOUNTANT", "bookings.create")).toBe(false);
  });

  it("keeps users.manage and audit.read SUPER_ADMIN-only (POST-1 trim)", () => {
    for (const role of ROLE_ORDER) {
      const expectedSA = role === "SUPER_ADMIN";
      expect(hasPermission(role, "users.manage"), role).toBe(expectedSA);
      expect(hasPermission(role, "audit.read"), role).toBe(expectedSA);
    }
  });

  it("permissionsFor collects the granted key set", () => {
    const p = permissionsFor("ADMIN");
    expect(p["leads.delete"]).toBe(true);
    expect(p["settings.manage"]).toBe(true);
    expect(p["users.manage"]).toBeUndefined();
  });

  it("gates invoices to accountant and above read-only for managers", () => {
    expect(hasPermission("ACCOUNTANT", "invoices.create")).toBe(true);
    expect(hasPermission("ACCOUNTANT", "invoices.update")).toBe(true);
    expect(hasPermission("MANAGER", "invoices.read")).toBe(true);
    expect(hasPermission("MANAGER", "invoices.create")).toBe(false);
    expect(hasPermission("DISPATCH", "invoices.read")).toBe(false);
  });
});