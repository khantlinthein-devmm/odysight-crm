<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { getReportSummary, type ReportSummary } from "../../../lib/reports";
import {
  currencyCode,
  formatMoney as formatMoneyFull,
  getWorkspaceSettings,
} from "../../../lib/settings";
import { showToast } from "../../../lib/toast";
import DonutChart from "./DonutChart.vue";
import TrendChart from "./TrendChart.vue";

const report = ref<ReportSummary | null>(null);
const loading = ref(true);

const LEAD_COLORS = ["#2563eb", "#0ea5e9", "#8b5cf6", "#f59e0b", "#10b981", "#f43f5e"];

const BOOKING_COLORS: Record<string, string> = {
  Pending: "#f59e0b",
  Confirmed: "#2563eb",
  "In Progress": "#0ea5e9",
  Completed: "#10b981",
  Cancelled: "#94a3b8",
  "No Show": "#f43f5e",
  Rescheduled: "#8b5cf6",
};

const kpis = computed(() => {
  if (!report.value) return [];
  return [
    {
      label: "Total Leads",
      value: String(report.value.totalLeads),
      sub: "Across all stages",
      href: "/leads",
      tile: "from-blue-500 to-navy-600",
      icon: "M17 20h5v-2a4 4 0 00-3-3.87M9 20H4v-2a4 4 0 013-3.87m6-1.13a4 4 0 10-4-4 4 4 0 004 4zm6-4a3 3 0 11-3-3",
    },
    {
      label: "Active Customers",
      value: String(report.value.activeCustomers),
      sub: "In customer base",
      href: "/customers",
      tile: "from-violet-500 to-purple-700",
      icon: "M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197",
    },
    {
      label: "Upcoming Bookings",
      value: String(report.value.upcomingBookings),
      sub: "Scheduled ahead",
      href: "/bookings",
      tile: "from-amber-400 to-orange-600",
      icon: "M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z",
    },
    {
      label: "Revenue (Month)",
      value: formatMoneyFull(report.value.monthlyRevenue),
      sub: "Collected this month",
      href: "/payments",
      tile: "from-emerald-400 to-green-600",
      icon: "M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z",
    },
  ];
});

const leadSegments = computed(
  () =>
    report.value?.leadsByStatus.map((s, i) => ({
      label: s.status,
      value: s.count,
      color: LEAD_COLORS[i % LEAD_COLORS.length]!,
    })) ?? [],
);

const bookingSegments = computed(
  () =>
    report.value?.bookingsByStatus.map((s) => ({
      label: s.status,
      value: s.count,
      color: BOOKING_COLORS[s.status] ?? "#64748b",
    })) ?? [],
);

const totalLeads = computed(() =>
  leadSegments.value.reduce((s, g) => s + g.value, 0),
);
const totalBookings = computed(() =>
  bookingSegments.value.reduce((s, g) => s + g.value, 0),
);
const totalRevenue = computed(
  () => report.value?.revenueByMonth.reduce((s, m) => s + m.collected, 0) ?? 0,
);
const avgRevenue = computed(() => {
  const months = report.value?.revenueByMonth.length ?? 0;
  return months > 0 ? totalRevenue.value / months : 0;
});

const maxCompleted = computed(() =>
  Math.max(
    1,
    ...(report.value?.cleanerProductivity.map((c) => c.completedBookings) ?? [1]),
  ),
);

function initials(name: string): string {
  return name
    .split(" ")
    .map((w) => w[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("")
    .toUpperCase();
}

function formatCompact(value: number): string {
  const code = currencyCode();
  if (value >= 1000) return `${code} ${(value / 1000).toFixed(0)}k`;
  return `${code} ${value}`;
}

const conversion = computed(() => report.value?.leadConversionRate ?? null);

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
    class="rounded-3xl bg-white p-8 shadow-sm ring-1 ring-gray-100"
  >
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <div v-for="i in 4" :key="i" class="rounded-2xl bg-gray-50 p-6">
        <div class="h-10 w-10 animate-pulse rounded-xl bg-gray-200"></div>
        <div class="mt-4 h-7 w-24 animate-pulse rounded-lg bg-gray-200"></div>
        <div class="mt-2 h-4 w-32 animate-pulse rounded-lg bg-gray-100"></div>
      </div>
    </div>
  </div>

  <template v-else-if="report">
    <!-- KPI cards -->
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <a
        v-for="kpi in kpis"
        :key="kpi.label"
        :href="kpi.href"
        class="group rounded-3xl bg-white p-5 shadow-sm ring-1 ring-gray-100 transition hover:-translate-y-0.5 hover:shadow-md"
      >
        <div class="flex items-center justify-between">
          <span
            :class="[
              'flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br text-white shadow-sm',
              kpi.tile,
            ]"
          >
            <svg
              class="h-5 w-5"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                :d="kpi.icon"
              />
            </svg>
          </span>
          <span
            class="text-gray-300 transition group-hover:translate-x-0.5 group-hover:text-navy-500"
            >→</span
          >
        </div>
        <p class="mt-4 text-[26px] font-semibold leading-none tracking-tight text-gray-900">
          {{ kpi.value }}
        </p>
        <p class="mt-1.5 text-sm font-medium text-gray-700">{{ kpi.label }}</p>
        <p class="text-xs text-gray-400">{{ kpi.sub }}</p>
      </a>
    </div>

    <!-- Distribution donuts -->
    <div class="mt-4 grid grid-cols-1 gap-4 xl:grid-cols-2">
      <section class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7">
        <div class="flex items-baseline justify-between">
          <div>
            <h2 class="font-semibold tracking-tight text-gray-900">Leads by status</h2>
            <p class="text-xs text-gray-400">Pipeline distribution</p>
          </div>
          <a href="/leads" class="text-xs font-medium text-navy-600 hover:underline">
            View leads →
          </a>
        </div>
        <DonutChart
          class="mt-5"
          :segments="leadSegments"
          :center-top="String(totalLeads)"
          center-bottom="total leads"
        />
      </section>

      <section class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7">
        <div class="flex items-baseline justify-between">
          <div>
            <h2 class="font-semibold tracking-tight text-gray-900">Bookings by status</h2>
            <p class="text-xs text-gray-400">Operational mix</p>
          </div>
          <a href="/bookings" class="text-xs font-medium text-navy-600 hover:underline">
            View bookings →
          </a>
        </div>
        <DonutChart
          class="mt-5"
          :segments="bookingSegments"
          :center-top="String(totalBookings)"
          center-bottom="total bookings"
        />
      </section>
    </div>

    <!-- Revenue trend -->
    <section class="mt-4 rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 class="font-semibold tracking-tight text-gray-900">Revenue trend</h2>
          <p class="text-xs text-gray-400">Collected payments per month</p>
        </div>
        <div class="flex gap-6">
          <div class="text-right">
            <p class="text-xs text-gray-400">Total (period)</p>
            <p class="text-lg font-semibold tracking-tight text-gray-900">
              {{ formatMoneyFull(totalRevenue) }}
            </p>
          </div>
          <div class="text-right">
            <p class="text-xs text-gray-400">Monthly avg</p>
            <p class="text-lg font-semibold tracking-tight text-emerald-600">
              {{ formatMoneyFull(Math.round(avgRevenue)) }}
            </p>
          </div>
        </div>
      </div>
      <TrendChart
        class="mt-4"
        :values="report.revenueByMonth.map((m) => m.collected)"
        :labels="report.revenueByMonth.map((m) => m.month)"
        :format-value="formatCompact"
        line-color="#10b981"
      />
    </section>

    <div class="mt-4 grid grid-cols-1 gap-4 xl:grid-cols-5">
      <!-- Team leaderboard -->
      <section class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7 xl:col-span-3">
        <div class="flex items-baseline justify-between">
          <div>
            <h2 class="font-semibold tracking-tight text-gray-900">Team leaderboard</h2>
            <p class="text-xs text-gray-400">Completed vs upcoming bookings</p>
          </div>
          <a href="/cleaners" class="text-xs font-medium text-navy-600 hover:underline">
            View team →
          </a>
        </div>
        <ul class="mt-5 space-y-3">
          <li
            v-for="(c, i) in report.cleanerProductivity"
            :key="c.cleanerName"
            class="flex items-center gap-3 rounded-2xl bg-gray-50/70 p-3 ring-1 ring-gray-100"
          >
            <span
              class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br text-xs font-semibold text-white"
              :class="i === 0 ? 'from-amber-400 to-orange-500' : 'from-navy-500 to-blue-600'"
            >
              {{ initials(c.cleanerName) }}
            </span>
            <div class="min-w-0 flex-1">
              <div class="flex items-baseline justify-between gap-2">
                <p class="truncate text-sm font-medium text-gray-900">{{ c.cleanerName }}</p>
                <p class="shrink-0 text-xs tabular-nums text-gray-500">
                  <span class="font-semibold text-gray-900">{{ c.completedBookings }}</span> done ·
                  {{ c.upcomingBookings }} upcoming
                </p>
              </div>
              <div class="mt-1.5 h-2 overflow-hidden rounded-full bg-gray-200/70">
                <div
                  class="h-full rounded-full bg-gradient-to-r from-navy-500 to-blue-400 transition-all duration-700"
                  :style="{ width: `${(c.completedBookings / maxCompleted) * 100}%` }"
                ></div>
              </div>
            </div>
            <span
              v-if="i === 0"
              class="shrink-0 rounded-full bg-amber-100 px-2.5 py-1 text-[11px] font-semibold text-amber-700"
            >
              ★ Top
            </span>
          </li>
        </ul>
      </section>

      <!-- Conversion -->
      <section
        class="relative overflow-hidden rounded-3xl bg-gray-950 p-6 text-white shadow-lg ring-1 ring-gray-900 sm:p-7 xl:col-span-2"
      >
        <div
          class="pointer-events-none absolute -right-14 -top-14 h-48 w-48 rounded-full bg-navy-500/40 blur-3xl"
        ></div>
        <p class="relative text-xs font-medium uppercase tracking-widest text-gray-400">
          Lead conversion
        </p>
        <div v-if="conversion !== null" class="relative mt-4 flex items-center gap-5">
          <svg viewBox="0 0 120 120" class="h-28 w-28 shrink-0">
            <circle cx="60" cy="60" r="52" fill="none" class="stroke-white/10" stroke-width="12" />
            <circle
              cx="60"
              cy="60"
              r="52"
              fill="none"
              stroke="url(#conv-grad)"
              stroke-width="12"
              stroke-linecap="round"
              pathLength="100"
              :stroke-dasharray="`${conversion} ${100 - conversion}`"
              stroke-dashoffset="25"
              transform="rotate(0 60 60)"
              class="transition-all duration-700"
            />
            <defs>
              <linearGradient id="conv-grad" x1="0" y1="0" x2="1" y2="1">
                <stop offset="0%" stop-color="#34d399" />
                <stop offset="100%" stop-color="#3b82f6" />
              </linearGradient>
            </defs>
            <text
              x="60"
              y="58"
              text-anchor="middle"
              font-size="22"
              font-weight="600"
              class="fill-white"
            >
              {{ conversion.toFixed(1) }}%
            </text>
            <text x="60" y="74" text-anchor="middle" font-size="10" class="fill-gray-400">
              converted
            </text>
          </svg>
          <p class="text-sm leading-relaxed text-gray-300">
            {{ conversion.toFixed(1) }}% of leads turn into paying
            bookings. Nurture quotes to push this higher.
          </p>
        </div>
        <p v-else class="relative mt-4 text-sm text-gray-400">
          Conversion data is not available yet.
        </p>
        <a
          href="/leads"
          class="relative mt-5 inline-block rounded-xl bg-white/10 px-4 py-2 text-xs font-semibold text-white ring-1 ring-white/15 transition hover:bg-white/20"
        >
          Work the pipeline →
        </a>
      </section>
    </div>

    <p class="mt-4 text-xs text-gray-400">
      Live data from the Go API · GET /api/v1/reports/summary
    </p>
  </template>
</template>
