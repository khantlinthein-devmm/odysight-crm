<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { getReportSummary, type ReportSummary } from "../../../lib/reports";
import {
  currencyCode,
  formatMoney as formatMoneyFull,
  getWorkspaceSettings,
} from "../../../lib/settings";
import { showToast } from "../../../lib/toast";

const report = ref<ReportSummary | null>(null);
const loading = ref(true);

const kpis = computed(() => {
  if (!report.value) return [];
  return [
    {
      label: "Total Leads",
      value: String(report.value.totalLeads),
      href: "/leads",
    },
    {
      label: "Active Customers",
      value: String(report.value.activeCustomers),
      href: "/customers",
    },
    {
      label: "Upcoming Bookings",
      value: String(report.value.upcomingBookings),
      href: "/bookings",
    },
    {
      label: "Revenue (Month)",
      value: formatMoneyFull(report.value.monthlyRevenue),
      href: "/payments",
    },
  ];
});

const maxLeadCount = computed(() =>
  report.value
    ? Math.max(...report.value.leadsByStatus.map((s) => s.count))
    : 1,
);

const maxBookingCount = computed(() =>
  report.value
    ? Math.max(...report.value.bookingsByStatus.map((s) => s.count))
    : 1,
);

const maxRevenue = computed(() =>
  report.value
    ? Math.max(...report.value.revenueByMonth.map((m) => m.collected))
    : 1,
);

const leadBarColors: Record<string, string> = {
  New: "bg-navy-500",
  Contacted: "bg-navy-500",
  "Quote Sent": "bg-navy-500",
  Booked: "bg-amber-500",
  Won: "bg-green-500",
  Lost: "bg-navy-400",
};

const bookingBarColors: Record<string, string> = {
  Pending: "bg-navy-400",
  Confirmed: "bg-navy-500",
  "In Progress": "bg-navy-500",
  Completed: "bg-green-500",
  Cancelled: "bg-navy-600",
  "No Show": "bg-navy-700",
  Rescheduled: "bg-amber-500",
};

function formatMoney(value: number): string {
  const code = currencyCode();
  if (value >= 1000) return `${code} ${(value / 1000).toFixed(0)}k`;
  return `${code} ${value}`;
}

onMounted(async () => {
  try {
    // Warm the workspace settings cache (currency) first.
    await getWorkspaceSettings();
    report.value = await getReportSummary();
  } catch {
    showToast("Failed to load reports", "error");
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <div
    v-if="loading"
    class="rounded-xl border border-gray-200 bg-white px-6 py-16 text-center"
  >
    <p class="text-sm text-gray-500">Loading reports...</p>
  </div>

  <template v-else-if="report">
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <a
        v-for="kpi in kpis"
        :key="kpi.label"
        :href="kpi.href"
        class="rounded-xl border border-gray-200 bg-white p-6 transition-shadow hover:shadow-md"
      >
        <p class="text-sm font-medium text-gray-500">{{ kpi.label }}</p>
        <p class="mt-2 text-3xl font-semibold text-gray-900">
          {{ kpi.value }}
        </p>
      </a>
    </div>

    <div class="mt-6 grid grid-cols-1 gap-4 xl:grid-cols-2">
      <div class="rounded-xl border border-gray-200 bg-white p-6">
        <h2 class="text-base font-semibold text-gray-900">Leads by Status</h2>
        <div class="mt-5 space-y-3">
          <div
            v-for="item in report.leadsByStatus"
            :key="item.status"
            class="flex items-center gap-3"
          >
            <span class="w-24 shrink-0 text-sm text-gray-600">{{
              item.status
            }}</span>
            <div class="h-6 flex-1 overflow-hidden rounded-md bg-gray-100">
              <div
                :class="[
                  'h-full rounded-md transition-all duration-500',
                  leadBarColors[item.status] ?? 'bg-navy-400',
                ]"
                :style="{ width: `${(item.count / maxLeadCount) * 100}%` }"
              ></div>
            </div>
            <span
              class="w-8 shrink-0 text-right text-sm font-medium text-gray-900"
              >{{ item.count }}</span
            >
          </div>
        </div>
      </div>

      <div class="rounded-xl border border-gray-200 bg-white p-6">
        <h2 class="text-base font-semibold text-gray-900">Monthly Revenue</h2>
        <div class="mt-5 flex h-44 items-end gap-4">
          <div
            v-for="month in report.revenueByMonth"
            :key="month.month"
            class="flex flex-1 flex-col items-center gap-2"
          >
            <span class="text-xs font-medium text-gray-600">{{
              formatMoney(month.collected)
            }}</span>
            <div
              class="w-full rounded-t-md bg-green-500/80 transition-all duration-500 hover:bg-green-500"
              :style="{
                height: `${Math.max((month.collected / maxRevenue) * 100, 4)}%`,
              }"
            ></div>
            <span class="text-xs text-gray-500">{{ month.month }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="mt-4 rounded-xl border border-gray-200 bg-white p-6">
      <h2 class="text-base font-semibold text-gray-900">
        Bookings by Status
      </h2>
      <div class="mt-5 space-y-3">
        <div
          v-for="item in report.bookingsByStatus"
          :key="item.status"
          class="flex items-center gap-3"
        >
          <span class="w-28 shrink-0 text-sm text-gray-600">{{
            item.status
          }}</span>
          <div class="h-6 flex-1 overflow-hidden rounded-md bg-gray-100">
            <div
              :class="[
                'h-full rounded-md transition-all duration-500',
                bookingBarColors[item.status] ?? 'bg-navy-400',
              ]"
              :style="{ width: `${(item.count / maxBookingCount) * 100}%` }"
            ></div>
          </div>
          <span
            class="w-8 shrink-0 text-right text-sm font-medium text-gray-900"
            >{{ item.count }}</span
          >
        </div>
      </div>
    </div>

    <p class="mt-4 text-xs text-gray-400">
      Live data from the Go API · GET /api/v1/reports/summary
    </p>
  </template>
</template>
