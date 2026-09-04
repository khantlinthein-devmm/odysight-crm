import { describe, expect, it } from "vitest";
import {
  createApplicant,
  deleteApplicant,
  getApplicant,
  getApplicants,
} from "./applicants";
import {
  createPayment,
  getPayments,
  updatePayment,
  type CreatePaymentInput,
} from "./payments";
import {
  createVisaCase,
  getVisaCases,
  type CreateVisaCaseInput,
} from "./visa-cases";
import { uploadDocument, getDocuments, deleteDocument } from "./documents";

describe("applicants mock API", () => {
  it("lists and creates applicants", async () => {
    const before = await getApplicants();
    const created = await createApplicant({
      firstName: "Test",
      lastName: "Applicant",
      email: "test.applicant@example.com",
      phone: "",
      nationality: "Brazil",
      visaType: "Student Visa",
      status: "screening",
    });
    const after = await getApplicants();

    expect(created.id).toBeGreaterThan(0);
    expect(after.length).toBe(before.length + 1);
    await expect(getApplicant(999999)).rejects.toThrow(/not found/i);
    void (await deleteApplicant(created.id));
  });
});

describe("visa cases mock API", () => {
  it("creates a case with generated case number", async () => {
    const before = await getVisaCases();
    const input: CreateVisaCaseInput = {
      applicantName: "Test Applicant",
      visaType: "Work Permit",
      destination: "Germany",
      assignedTo: "Someone",
      status: "draft",
    };
    const created = await createVisaCase(input);
    const after = await getVisaCases();

    expect(created.caseNumber).toMatch(/^VC-/);
    expect(after.length).toBe(before.length + 1);
  });
});

describe("documents mock API", () => {
  it("uploads and deletes documents", async () => {
    const before = await getDocuments();
    const doc = await uploadDocument({
      name: "test.pdf",
      type: "Passport",
      applicantName: "Test Applicant",
      status: "pending",
    });

    expect(doc.fileSizeKb).toBeGreaterThan(0);
    expect((await getDocuments()).length).toBe(before.length + 1);
    await deleteDocument(doc.id);
    expect((await getDocuments()).length).toBe(before.length);
  });
});

describe("payments mock API", () => {
  it("creates a payment with generated invoice number", async () => {
    const before = await getPayments();
    const input: CreatePaymentInput = {
      payerName: "Test Payer",
      amount: 100,
      currency: "USD",
      method: "Cash",
      status: "pending",
    };
    const created = await createPayment(input);

    expect(created.invoiceNumber).toMatch(/^INV-/);
    expect((await getPayments()).length).toBe(before.length + 1);
  });

  it("marks a payment as paid", async () => {
    const payments = await getPayments();
    const pending = payments.find((p) => p.status === "pending");
    if (!pending) return;
    const updated = await updatePayment(pending.id, { status: "paid" });
    expect(updated.status).toBe("paid");
  });
});
