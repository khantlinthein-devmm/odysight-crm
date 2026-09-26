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

const LEAD_COLORS = ["#D4AF37", "#2563eb", "#0ea5e9", "#8b5cf6", "#10b981", "#f43f5e"];

const BOOKING_COLORS: Record<string, string> = {
  Pending: "#D4AF37",
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
      delta: "+12.4% vs last month",
      up: true,
      href: "/leads",
      tile: "from-amber-400 to-yellow-600",
      icon: "M17 20h5v-2a4 4 0 00-3-3.87M9 20H4v-2a4 4 0 013-3.87m6-1.13a4 4 0 10-4-4 4 4 0 004 4zm6-4a3 3 0 11-3-3",
    },
    {
      label: "Active Customers",
      value: String(report.value.activeCustomers),
      sub: "In customer base",
      delta: "+8.1% vs last month",
      up: true,
      href: "/customers",
      tile: "from-navy-500 to-blue-700",
      icon: "M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197",
    },
    {
      label: "Upcoming Bookings",
      value: String(report.value.upcomingBookings),
      sub: "Scheduled ahead",
      delta: "+5.6% this week",
      up: true,
      href: "/bookings",
      tile: "from-violet-500 to-purple-700",
      icon: "M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z",
    },
    {
      label: "Revenue (Month)",
      value: formatMoneyFull(report.value.monthlyRevenue),
      sub: "Collected this month",
      delta: "+18.2% growth",
      up: true,
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
    <!-- Banking hero: total balance card -->
    <section
      class="relative overflow-hidden rounded-3xl bg-gradient-to-br from-slate-950 via-navy-900 to-slate-900 p-6 text-white shadow-xl ring-1 ring-slate-800 sm:p-8"
    >
      <div class="pointer-events-none absolute -right-20 -top-24 h-72 w-72 rounded-full bg-amber-400/20 blur-3xl"></div>
      <div class="pointer-events-none absolute -bottom-28 -left-16 h-72 w-72 rounded-full bg-blue-500/20 blur-3xl"></div>
      <div class="pointer-events-none absolute right-10 top-6 hidden select-none gap-1.5 sm:flex">
        <span class="h-1.5 w-1.5 rounded-full bg-white/30"></span>
        <span class="h-1.5 w-1.5 rounded-full bg-white/30"></span>
        <span class="h-1.5 w-1.5 rounded-full bg-white/30"></span>
      </div>

      <div class="relative flex flex-wrap items-start justify-between gap-6">
        <div class="min-w-[240px]">
          <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-amber-300/90">Total balance · collected revenue</p>
          <p class="mt-2 text-4xl font-bold tracking-tight sm:text-5xl">
            {{ formatMoneyFull(totalRevenue) }}
          </p>
          <p class="mt-2 flex items-center gap-2 text-sm">
            <span class="inline-flex items-center gap-1 rounded-full bg-emerald-400/15 px-2.5 py-1 text-xs font-semibold text-emerald-300 ring-1 ring-emerald-300/30">▲ +18.2%</span>
            <span class="text-slate-300">vs previous period</span>
          </p>
          <div class="mt-5 flex flex-wrap gap-2">
            <a href="/payments" class="rounded-xl bg-amber-400 px-4 py-2 text-sm font-bold text-slate-950 shadow-sm transition hover:brightness-110">＋ Add funds</a>
            <a href="/invoices" class="rounded-xl bg-white/10 px-4 py-2 text-sm font-semibold text-white ring-1 ring-white/20 transition hover:bg-white/20">Invoices</a>
            <a href="/payments" class="rounded-xl bg-white/10 px-4 py-2 text-sm font-semibold text-white ring-1 ring-white/20 transition hover:bg-white/20">Transactions</a>
          </div>
          <p class="mt-5 font-mono text-xs tracking-[0.3em] text-slate-400">•••• •••• •••• {{ String(totalBookings).padStart(4, "0") }}</p>
        </div>

        <div class="grid w-full max-w-md flex-1 grid-cols-3 gap-3">
          <div class="rounded-2xl bg-white/[0.07] p-4 ring-1 ring-white/10 backdrop-blur">
            <p class="text-[11px] uppercase tracking-wide text-slate-400">Income</p>
            <p class="mt-1 truncate text-lg font-bold text-emerald-300">{{ formatMoneyFull(report.monthlyRevenue) }}</p>
            <p class="mt-0.5 text-[11px] text-slate-400">this month</p>
          </div>
          <div class="rounded-2xl bg-white/[0.07] p-4 ring-1 ring-white/10 backdrop-blur">
            <p class="text-[11px] uppercase tracking-wide text-slate-400">Avg / mo</p>
            <p class="mt-1 truncate text-lg font-bold text-amber-300">{{ formatMoneyFull(Math.round(avgRevenue)) }}</p>
            <p class="mt-0.5 text-[11px] text-slate-400">period average</p>
          </div>
          <div class="rounded-2xl bg-white/[0.07] p-4 ring-1 ring-white/10 backdrop-blur">
            <p class="text-[11px] uppercase tracking-wide text-slate-400">Upcoming</p>
            <p class="mt-1 text-lg font-bold text-white">{{ report.upcomingBookings }}</p>
            <p class="mt-0.5 text-[11px] text-slate-400">bookings</p>
          </div>
        </div>
      </div>
    </section>

    <!-- Banking KPI stat cards -->
    <div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <a
        v-for="kpi in kpis"
        :key="kpi.label"
        :href="kpi.href"
        class="group relative overflow-hidden rounded-3xl bg-white p-5 shadow-sm ring-1 ring-gray-100 transition hover:-translate-y-0.5 hover:shadow-md"
      >
        <div class="absolute inset-x-0 top-0 h-1 bg-gradient-to-r" :class="kpi.tile"></div>
        <div class="flex items-center justify-between">
          <span
            :class="[
              'flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br text-white shadow-sm',
              kpi.tile,
            ]"
          >
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" :d="kpi.icon" />
            </svg>
          </span>
          <span class="rounded-full bg-emerald-50 px-2 py-0.5 text-[11px] font-bold text-emerald-700 ring-1 ring-emerald-200">{{ kpi.delta }}</span>
        </div>
        <p class="mt-4 text-[26px] font-bold leading-none tracking-tight text-gray-900">{{ kpi.value }}</p>
        <p class="mt-1.5 text-sm font-semibold text-gray-800">{{ kpi.label }}</p>
        <p class="text-xs text-gray-400">{{ kpi.sub }} <span class="text-gray-300 transition group-hover:translate-x-0.5 group-hover:text-amber-500">→</span></p>
      </a>
    </div>

    <!-- Cash flow + spending breakdown (banking: income chart + expense donut) -->
    <div class="mt-4 grid grid-cols-1 gap-4 xl:grid-cols-5">
      <section class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7 xl:col-span-3">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p class="text-[11px] font-bold uppercase tracking-[0.18em] text-amber-600">Cash flow</p>
            <h2 class="mt-1 text-lg font-bold tracking-tight text-gray-900">Money movement</h2>
            <p class="text-xs text-gray-400">Collected payments per month</p>
          </div>
          <div class="flex items-center gap-4">
            <span class="flex items-center gap-1.5 text-xs text-gray-500"><span class="h-2.5 w-2.5 rounded-full bg-emerald-500"></span> Income</span>
            <span class="flex items-center gap-1.5 text-xs text-gray-500"><span class="h-2.5 w-2.5 rounded-full bg-amber-400"></span> Avg</span>
            <div class="text-right">
              <p class="text-xs text-gray-400">Total</p>
              <p class="text-lg font-bold tracking-tight text-gray-900">{{ formatMoneyFull(totalRevenue) }}</p>
            </div>
          </div>
        </div>
        <TrendChart
          class="mt-4"
          :values="report.revenueByMonth.map((m) => m.collected)"
          :labels="report.revenueByMonth.map((m) => m.month)"
          :format-value="formatCompact"
          line-color="#D4AF37"
        />
        <div class="mt-4 grid grid-cols-3 gap-2">
          <div v-for="m in report.revenueByMonth.slice(-3)" :key="m.month" class="rounded-2xl bg-gradient-to-b from-amber-50 to-white p-3 text-center ring-1 ring-amber-100">
            <p class="text-[11px] font-semibold uppercase text-gray-400">{{ m.month }}</p>
            <p class="mt-0.5 text-sm font-bold text-gray-900">{{ formatCompact(m.collected) }}</p>
          </div>
        </div>
      </section>

      <section class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7 xl:col-span-2">
        <div class="flex items-baseline justify-between">
          <div>
            <p class="text-[11px] font-bold uppercase tracking-[0.18em] text-amber-600">Spending activity</p>
            <h2 class="mt-1 text-lg font-bold tracking-tight text-gray-900">Bookings by status</h2>
            <p class="text-xs text-gray-400">Operational mix · {{ totalBookings }} total</p>
          </div>
          <a href="/bookings" class="text-xs font-semibold text-amber-700 hover:underline">Details →</a>
        </div>
        <DonutChart
          class="mt-5"
          :segments="bookingSegments"
          :center-top="String(totalBookings)"
          center-bottom="transactions"
          compact
        />
      </section>
    </div>

    <!-- Pipeline + leaderboard + credit score -->
    <div class="mt-4 grid grid-cols-1 gap-4 xl:grid-cols-5">
      <section class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7 xl:col-span-2">
        <div class="flex items-baseline justify-between">
          <div>
            <p class="text-[11px] font-bold uppercase tracking-[0.18em] text-amber-600">Pipeline</p>
            <h2 class="mt-1 text-lg font-bold tracking-tight text-gray-900">Leads by status</h2>
            <p class="text-xs text-gray-400">{{ totalLeads }} total leads</p>
          </div>
          <a href="/leads" class="text-xs font-semibold text-amber-700 hover:underline">View →</a>
        </div>
        <DonutChart class="mt-5" :segments="leadSegments" :center-top="String(totalLeads)" center-bottom="total leads" compact />
      </section>

      <section class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7 xl:col-span-2">
        <div class="flex items-baseline justify-between">
          <div>
            <p class="text-[11px] font-bold uppercase tracking-[0.18em] text-amber-600">Top accounts</p>
            <h2 class="mt-1 text-lg font-bold tracking-tight text-gray-900">Team leaderboard</h2>
            <p class="text-xs text-gray-400">Completed vs upcoming</p>
          </div>
          <a href="/cleaners" class="text-xs font-semibold text-amber-700 hover:underline">Team →</a>
        </div>
        <ul class="mt-5 space-y-2.5">
          <li
            v-for="(c, i) in report.cleanerProductivity"
            :key="c.cleanerName"
            class="flex items-center gap-3 rounded-2xl bg-gradient-to-r from-slate-50 to-amber-50/50 p-3 ring-1 ring-gray-100"
          >
            <span
              class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br text-xs font-bold text-white shadow-sm"
              :class="i === 0 ? 'from-amber-400 to-yellow-600' : 'from-slate-700 to-slate-900'"
            >
              {{ initials(c.cleanerName) }}
            </span>
            <div class="min-w-0 flex-1">
              <div class="flex items-baseline justify-between gap-2">
                <p class="truncate text-sm font-semibold text-gray-900">{{ c.cleanerName }}</p>
                <p class="shrink-0 text-xs tabular-nums text-gray-500">
                  <span class="font-bold text-gray-900">{{ c.completedBookings }}</span> done · {{ c.upcomingBookings }} upcoming
                </p>
              </div>
              <div class="mt-1.5 h-2 overflow-hidden rounded-full bg-gray-200/70">
                <div
                  class="h-full rounded-full bg-gradient-to-r from-amber-400 to-yellow-600 transition-all duration-700"
                  :style="{ width: `${(c.completedBookings / maxCompleted) * 100}%` }"
                ></div>
              </div>
            </div>
            <span v-if="i === 0" class="shrink-0 rounded-full bg-amber-100 px-2.5 py-1 text-[11px] font-bold text-amber-800 ring-1 ring-amber-300">★ Top</span>
          </li>
        </ul>
      </section>

      <!-- Credit-score style conversion card -->
      <section class="relative overflow-hidden rounded-3xl bg-gradient-to-b from-slate-950 to-slate-900 p-6 text-white shadow-xl ring-1 ring-slate-800 sm:p-7">
        <div class="pointer-events-none absolute -right-14 -top-14 h-48 w-48 rounded-full bg-amber-400/25 blur-3xl"></div>
        <p class="relative text-[11px] font-bold uppercase tracking-[0.2em] text-amber-300">Credit score · conversion</p>
        <h2 class="relative mt-1 text-lg font-bold">Lead conversion</h2>
        <div v-if="conversion !== null" class="relative mt-4 flex flex-col items-center gap-4">
          <svg viewBox="0 0 120 120" class="h-32 w-32 shrink-0">
            <circle cx="60" cy="60" r="52" fill="none" class="stroke-white/10" stroke-width="12" />
            <circle
              cx="60" cy="60" r="52" fill="none" stroke="url(#conv-grad)" stroke-width="12"
              stroke-linecap="round" pathLength="100"
              :stroke-dasharray="`${conversion} ${100 - conversion}`"
              stroke-dashoffset="25" transform="rotate(0 60 60)" class="transition-all duration-700"
            />
            <defs>
              <linearGradient id="conv-grad" x1="0" y1="0" x2="1" y2="1">
                <stop offset="0%" stop-color="#D4AF37" />
                <stop offset="100%" stop-color="#34d399" />
              </linearGradient>
            </defs>
            <text x="60" y="58" text-anchor="middle" font-size="22" font-weight="700" class="fill-white">{{ conversion.toFixed(1) }}%</text>
            <text x="60" y="74" text-anchor="middle" font-size="10" class="fill-gray-400">score</text>
          </svg>
          <p class="text-center text-sm leading-relaxed text-slate-300">
            {{ conversion.toFixed(1) }}% of leads turn into paying bookings. Nurture quotes to push this higher.
          </p>
        </div>
        <p v-else class="relative mt-4 text-sm text-gray-400">Conversion data is not available yet.</p>
        <a href="/leads" class="relative mt-5 block rounded-xl bg-amber-400 px-4 py-2.5 text-center text-sm font-bold text-slate-950 transition hover:brightness-110">
          Work the pipeline →
        </a>
      </section>
    </div>
  </template>
</template>
