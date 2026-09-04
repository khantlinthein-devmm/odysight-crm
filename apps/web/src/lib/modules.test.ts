import { describe, expect, it } from "vitest";
import {
  createCustomer,
  deleteCustomer,
  getCustomer,
  getCustomers,
} from "./customers";
import {
  createBooking,
  deleteBooking,
  getBookings,
  type CreateBookingInput,
} from "./bookings";
import {
  createServiceRecord,
  deleteServiceRecord,
  getServiceRecords,
} from "./service-records";
import {
  createCleaner,
  deleteCleaner,
  getCleaners,
} from "./cleaners";
import {
  createPayment,
  getPayments,
  updatePayment,
  type CreatePaymentInput,
} from "./payments";

describe("customers mock API", () => {
  it("lists and creates customers", async () => {
    const before = await getCustomers();
    const created = await createCustomer({
      firstName: "Test",
      lastName: "Customer",
      email: "test.customer@example.com",
      phone: "+66 91 000 0000",
      address: "123 Test Street, Bangkok",
      propertyType: "condo",
      area: "Sukhumvit",
      status: "active",
    });
    const after = await getCustomers();

    expect(created.id).toBeGreaterThan(0);
    expect(after.length).toBe(before.length + 1);
    await expect(getCustomer(999999)).rejects.toThrow(/not found/i);
    await deleteCustomer(created.id);
  });
});

describe("bookings mock API", () => {
  it("creates a booking with generated booking number", async () => {
    const before = await getBookings();
    const input: CreateBookingInput = {
      customerName: "Test Customer",
      serviceType: "condo_cleaning",
      scheduledFor: "2026-09-10T09:00:00Z",
      durationMinutes: 180,
      address: "Sukhumvit 38, Bangkok",
      assignedCleaner: "Nok Srisuwan",
      status: "pending",
      notes: "",
    };
    const created = await createBooking(input);
    const after = await getBookings();

    expect(created.bookingNumber).toMatch(/^BK-/);
    expect(after.length).toBe(before.length + 1);
    await deleteBooking(created.id);
  });
});

describe("service records mock API", () => {
  it("creates and deletes service records", async () => {
    const before = await getServiceRecords();
    const created = await createServiceRecord({
      bookingNumber: "BK-2026-0201",
      cleanerName: "Nok Srisuwan",
      serviceType: "Condo Cleaning",
      rating: null,
      status: "pending",
      notes: "",
    });

    expect(created.id).toBeGreaterThan(0);
    expect((await getServiceRecords()).length).toBe(before.length + 1);
    await deleteServiceRecord(created.id);
    expect((await getServiceRecords()).length).toBe(before.length);
  });
});

describe("cleaners mock API", () => {
  it("lists, creates and deletes cleaners", async () => {
    const before = await getCleaners();
    const created = await createCleaner({
      firstName: "Test",
      lastName: "Cleaner",
      phone: "+66 81 000 0000",
      email: "test.cleaner@smileclean.com",
      skills: "General Cleaning",
      status: "available",
    });

    expect(created.id).toBeGreaterThan(0);
    expect((await getCleaners()).length).toBe(before.length + 1);
    await deleteCleaner(created.id);
    expect((await getCleaners()).length).toBe(before.length);
  });
});

describe("payments mock API", () => {
  it("creates a payment with generated invoice number", async () => {
    const before = await getPayments();
    const input: CreatePaymentInput = {
      customerName: "Test Customer",
      bookingNumber: "BK-2026-0201",
      amount: 100,
      currency: "THB",
      method: "promptpay",
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
