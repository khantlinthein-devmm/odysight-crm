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

export interface CleanerProductivity {
  cleanerName: string;
  completedBookings: number;
  upcomingBookings: number;
}

export interface ReportSummary {
  totalLeads: number;
  activeCustomers: number;
  upcomingBookings: number;
  monthlyRevenue: number;
  leadsByStatus: LeadStatusCount[];
  revenueByMonth: MonthlyRevenue[];
  bookingsByStatus: BookingStatusCount[];
  leadConversionRate: number | null;
  cleanerProductivity: CleanerProductivity[];
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
  leadConversionRate: 12.5,
  cleanerProductivity: [
    { cleanerName: "Nuch Srisai", completedBookings: 34, upcomingBookings: 5 },
    { cleanerName: "Pimchanok W.", completedBookings: 28, upcomingBookings: 8 },
    { cleanerName: "Somchai Prasert", completedBookings: 22, upcomingBookings: 3 },
    { cleanerName: "Areeya Kaeo", completedBookings: 17, upcomingBookings: 6 },
  ],
};

export async function getReportSummary(): Promise<ReportSummary> {
  if (USE_MOCKS) {
    await delay(300);
    return JSON.parse(JSON.stringify(mockReport)) as ReportSummary;
  }
  return apiFetch<ReportSummary>("/api/v1/reports/summary");
}

export interface SiteRevenue {
  siteId: number;
  customerId: number;
  siteName: string;
  bookings: number;
  completed: number;
  billed: number;
}

export interface ContractRevenue {
  contractId: number;
  contractNumber: string;
  title: string;
  status: string;
  contractValue: number;
  bookings: number;
  billed: number;
}

export interface CommercialReport {
  revenueBySite: SiteRevenue[];
  revenueByContract: ContractRevenue[];
  quoteWinRate: { total: number; accepted: number; rate: number | null };
  checklists: { total: number; completed: number; itemsTotal: number; itemsDone: number; completionRate: number | null };
}

const mockCommercial: CommercialReport = {
  revenueBySite: [
    { siteId: 501, customerId: 101, siteName: "Default Site", bookings: 3, completed: 2, billed: 9650 },
    { siteId: 502, customerId: 101, siteName: "Silom Branch", bookings: 1, completed: 0, billed: 0 },
  ],
  revenueByContract: [
    { contractId: 701, contractNumber: "CT-2026-0701", title: "Sukhumvit + Silom — 12-month", status: "active", contractValue: 480000, bookings: 2, billed: 6400 },
  ],
  quoteWinRate: { total: 4, accepted: 1, rate: 25 },
  checklists: { total: 1, completed: 0, itemsTotal: 3, itemsDone: 1, completionRate: 33.3 },
};

export async function getCommercialReport(): Promise<CommercialReport> {
  if (USE_MOCKS) {
    await delay(300);
    return JSON.parse(JSON.stringify(mockCommercial)) as CommercialReport;
  }
  return apiFetch<CommercialReport>("/api/v1/reports/commercial");
}
