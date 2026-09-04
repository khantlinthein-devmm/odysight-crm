import { describe, expect, it } from "vitest";
import { createLead, deleteLead, getLead, getLeads, updateLead } from "./leads";

describe("leads mock API", () => {
  it("returns a list of leads", async () => {
    const leads = await getLeads();
    expect(leads.length).toBeGreaterThan(0);
    expect(leads[0]).toMatchObject({
      id: expect.any(Number),
      firstName: expect.any(String),
      lastName: expect.any(String),
      email: expect.any(String),
      status: expect.any(String),
      source: expect.any(String),
      createdAt: expect.any(String),
    });
  });

  it("creates a lead with generated id and createdAt", async () => {
    const before = await getLeads();
    const created = await createLead({
      firstName: "Test",
      lastName: "Person",
      email: "test.person@example.com",
      phone: "",
      status: "new",
      source: "website",
    });
    const after = await getLeads();

    expect(created.id).toBeGreaterThan(0);
    expect(created.createdAt).toBeTruthy();
    expect(after.length).toBe(before.length + 1);
    expect(after.some((l) => l.id === created.id)).toBe(true);
  });

  it("updates an existing lead", async () => {
    const leads = await getLeads();
    const target = leads[0];
    const updated = await updateLead(target.id, { firstName: "Renamed" });

    expect(updated.firstName).toBe("Renamed");
    expect(updated.lastName).toBe(target.lastName);
  });

  it("deletes a lead", async () => {
    const created = await createLead({
      firstName: "Delete",
      lastName: "Me",
      email: "delete.me@example.com",
      phone: "",
      status: "new",
      source: "referral",
    });
    await deleteLead(created.id);
    const leads = await getLeads();
    expect(leads.some((l) => l.id === created.id)).toBe(false);
  });

  it("throws when fetching a missing lead", async () => {
    await expect(getLead(999999)).rejects.toThrow(/not found/i);
  });
});
