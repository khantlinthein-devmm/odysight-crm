import { USE_MOCKS, apiFetch, delay } from "./api";

export interface LeadStatusCount {
  status: string;
  count: number;
}

export interface MonthlyRevenue {
  month: string;
  collected: number;
}

export interface ReportSummary {
  totalLeads: number;
  activeApplicants: number;
  openVisaCases: number;
  monthlyRevenue: number;
  leadsByStatus: LeadStatusCount[];
  revenueByMonth: MonthlyRevenue[];
}

const mockReport: ReportSummary = {
  totalLeads: 128,
  activeApplicants: 86,
  openVisaCases: 42,
  monthlyRevenue: 24500,
  leadsByStatus: [
    { status: "New", count: 32 },
    { status: "Contacted", count: 28 },
    { status: "Qualified", count: 22 },
    { status: "Proposal", count: 18 },
    { status: "Won", count: 16 },
    { status: "Lost", count: 12 },
  ],
  revenueByMonth: [
    { month: "Mar", collected: 14200 },
    { month: "Apr", collected: 16800 },
    { month: "May", collected: 15400 },
    { month: "Jun", collected: 19700 },
    { month: "Jul", collected: 21300 },
    { month: "Aug", collected: 24500 },
  ],
};

export async function getReportSummary(): Promise<ReportSummary> {
  if (USE_MOCKS) {
    await delay(300);
    return JSON.parse(JSON.stringify(mockReport)) as ReportSummary;
  }
  return apiFetch<ReportSummary>("/api/v1/reports/summary");
}
