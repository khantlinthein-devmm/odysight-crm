import { USE_MOCKS, apiFetch, delay } from "./api";

export interface LeadStatusCount {
  status: string;
  count: number;
}

export interface MonthlyRevenue {
  month: string;
  collected: number;
}

export interface BookingStatusCount {
  status: string;
  count: number;
}

export interface ReportSummary {
  totalLeads: number;
  activeCustomers: number;
  upcomingBookings: number;
  monthlyRevenue: number;
  leadsByStatus: LeadStatusCount[];
  revenueByMonth: MonthlyRevenue[];
  bookingsByStatus: BookingStatusCount[];
}

const mockReport: ReportSummary = {
  totalLeads: 128,
  activeCustomers: 86,
  upcomingBookings: 42,
  monthlyRevenue: 245000,
  leadsByStatus: [
    { status: "New", count: 32 },
    { status: "Contacted", count: 28 },
    { status: "Quote Sent", count: 22 },
    { status: "Booked", count: 18 },
    { status: "Won", count: 16 },
    { status: "Lost", count: 12 },
  ],
  revenueByMonth: [
    { month: "Mar", collected: 142000 },
    { month: "Apr", collected: 168000 },
    { month: "May", collected: 154000 },
    { month: "Jun", collected: 197000 },
    { month: "Jul", collected: 213000 },
    { month: "Aug", collected: 245000 },
  ],
  bookingsByStatus: [
    { status: "Pending", count: 8 },
    { status: "Confirmed", count: 15 },
    { status: "In Progress", count: 5 },
    { status: "Completed", count: 60 },
    { status: "Cancelled", count: 6 },
    { status: "No Show", count: 2 },
    { status: "Rescheduled", count: 3 },
  ],
};

export async function getReportSummary(): Promise<ReportSummary> {
  if (USE_MOCKS) {
    await delay(300);
    return JSON.parse(JSON.stringify(mockReport)) as ReportSummary;
  }
  return apiFetch<ReportSummary>("/api/v1/reports/summary");
}
