<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ApiError } from "../../../lib/api";
import { clearSession } from "../../../lib/auth";
import {
  currencyCode,
  formatMoney,
  getWorkspaceSettings,
  serviceLabel,
} from "../../../lib/settings";
import { getReportSummary, type ReportSummary } from "../../../lib/reports";
import { getLeads, type Lead } from "../../../lib/leads";
import { getBookings, type Booking } from "../../../lib/bookings";
import { showToast } from "../../../lib/toast";
import DonutChart from "../reports/DonutChart.vue";
import TrendChart from "../reports/TrendChart.vue";

const API_URL = import.meta.env.PUBLIC_API_URL as string | undefined;

const summary = ref<ReportSummary | null>(null);
const recentLeads = ref<Lead[]>([]);
const recentBookings = ref<Booking[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const errorStatus = ref<number | null>(null);

const isAuthError = computed(() => errorStatus.value === 401);

const greeting = computed(() => {
  const h = new Date().getHours();
  if (h < 12) return "Good morning";
  if (h < 17) return "Good afternoon";
  return "Good evening";
});

const todayLine = computed(() =>
  new Date().toLocaleDateString(undefined, {
    weekday: "long",
    month: "long",
    day: "numeric",
  }),
);

const kpis = computed(() => {
  if (!summary.value) return [];
  return [
    {
      label: "Total Leads",
      value: String(summary.value.totalLeads),
      sub: "Across all stages",
      href: "/leads",
      tile: "from-blue-500 to-navy-600",
      icon: "M17 20h5v-2a4 4 0 00-3-3.87M9 20H4v-2a4 4 0 013-3.87m6-1.13a4 4 0 10-4-4 4 4 0 004 4zm6-4a3 3 0 11-3-3",
    },
    {
      label: "Active Customers",
      value: String(summary.value.activeCustomers),
      sub: "In customer base",
      href: "/customers",
      tile: "from-violet-500 to-purple-700",
      icon: "M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197",
    },
    {
      label: "Upcoming Bookings",
      value: String(summary.value.upcomingBookings),
      sub: "Scheduled ahead",
      href: "/bookings",
      tile: "from-amber-400 to-orange-600",
      icon: "M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z",
    },
    {
      label: "Revenue (Month)",
      value: formatMoney(summary.value.monthlyRevenue),
      sub: "Collected this month",
      href: "/payments",
      tile: "from-emerald-400 to-green-600",
      icon: "M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z",
    },
  ];
});

const BOOKING_COLORS: Record<string, string> = {
  Pending: "#f59e0b",
  Confirmed: "#2563eb",
  "In Progress": "#0ea5e9",
  Completed: "#10b981",
  Cancelled: "#94a3b8",
  "No Show": "#f43f5e",
  Rescheduled: "#8b5cf6",
};

const bookingSegments = computed(
  () =>
    summary.value?.bookingsByStatus.map((s) => ({
      label: s.status,
      value: s.count,
      color: BOOKING_COLORS[s.status] ?? "#64748b",
    })) ?? [],
);

const totalBookings = computed(() =>
  bookingSegments.value.reduce((s, g) => s + g.value, 0),
);

const bookingPill: Record<string, string> = {
  pending: "bg-amber-100 text-amber-700",
  confirmed: "bg-blue-100 text-blue-700",
  in_progress: "bg-sky-100 text-sky-700",
  completed: "bg-emerald-100 text-emerald-700",
  cancelled: "bg-gray-100 text-gray-500",
  no_show: "bg-rose-100 text-rose-700",
  rescheduled: "bg-violet-100 text-violet-700",
};

const leadPill: Record<string, string> = {
  new: "bg-blue-100 text-blue-700",
  contacted: "bg-sky-100 text-sky-700",
  quote_sent: "bg-violet-100 text-violet-700",
  booked: "bg-amber-100 text-amber-700",
  won: "bg-emerald-100 text-emerald-700",
  lost: "bg-gray-100 text-gray-500",
};

function statusText(s: string): string {
  return s.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

function initials(name: string): string {
  return name
    .split(" ")
    .map((w) => w[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("")
    .toUpperCase();
}

function bookingDay(value: string): string {
  try {
    return String(new Date(value).getDate());
  } catch {
    return "–";
  }
}

function bookingMonth(value: string): string {
  try {
    return new Date(value).toLocaleDateString(undefined, { month: "short" });
  } catch {
    return "";
  }
}

function bookingTime(value: string): string {
  try {
    return new Date(value).toLocaleTimeString(undefined, {
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return "";
  }
}

function formatCompact(value: number): string {
  const code = currencyCode();
  if (value >= 1000) return `${code} ${(value / 1000).toFixed(0)}k`;
  return `${code} ${value}`;
}

async function load() {
  loading.value = true;
  error.value = null;
  errorStatus.value = null;
  try {
    // Warm the workspace settings cache (currency, catalog) first.
    await getWorkspaceSettings();
    const [s, leads, bookings] = await Promise.all([
      getReportSummary(),
      getLeads({ limit: 5 }),
      getBookings({ limit: 5 }),
    ]);
    summary.value = s;
    recentLeads.value = leads.slice(0, 5);
    recentBookings.value = bookings.slice(0, 5);
  } catch (e) {
    const status = e instanceof ApiError ? e.status : null;
    errorStatus.value = status;
    const msg =
      e instanceof ApiError
        ? `API error ${e.status}: ${e.message}`
        : e instanceof Error
          ? e.message
          : "Failed to load dashboard";
    error.value = msg;
    if (status === 401) {
      // Stale token (e.g. old mock-token, expired JWT, or JWT signed with a
      // previous JWT_SECRET). Purge it so a fresh sign-in starts clean.
      // The Astro middleware only checks cookie presence, not validity,
      // so without this the app keeps resending the rejected token.
      clearSession();
    }
    showToast(msg, "error");
  } finally {
    loading.value = false;
  }
}

function signInAgain() {
  clearSession();
  window.location.href = "/login?next=/dashboard";
}

onMounted(load);
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

  <div
    v-else-if="error"
    class="rounded-3xl p-7 shadow-sm ring-1"
    :class="
      isAuthError
        ? 'bg-amber-50 ring-amber-200'
        : 'bg-red-50 ring-red-200'
    "
  >
    <h2
      class="font-semibold tracking-tight"
      :class="isAuthError ? 'text-amber-900' : 'text-red-900'"
    >
      {{ isAuthError ? "Your session has expired" : "Couldn't reach the API" }}
    </h2>
    <p
      class="mt-1 text-sm"
      :class="isAuthError ? 'text-amber-800' : 'text-red-700'"
    >
      {{
        isAuthError
          ? "The API rejected your stored token (invalid or expired). Your stale session was cleared — please sign in again to get a fresh token."
          : error
      }}
    </p>
    <p
      v-if="!isAuthError"
      class="mt-2 text-xs text-red-600"
    >
      API base URL:
      <code class="rounded bg-red-100 px-1">{{ API_URL ?? "(not set)" }}</code>
      · Check that the Go API is running (<code>docker compose ps</code>,
      <code>curl http://localhost:8080/ready</code>) and CORS allows this
      origin.
    </p>
    <div class="mt-4 flex flex-wrap gap-3">
      <button
        v-if="isAuthError"
        type="button"
        class="rounded-xl bg-navy-600 px-4 py-2 text-sm font-semibold text-white hover:bg-navy-700"
        @click="signInAgain"
      >
        Sign in again
      </button>
      <button
        type="button"
        class="rounded-xl px-4 py-2 text-sm font-semibold"
        :class="
          isAuthError
            ? 'bg-white text-amber-800 ring-1 ring-amber-300 hover:bg-amber-100'
            : 'bg-red-600 text-white hover:bg-red-700'
        "
        @click="load"
      >
        Retry
      </button>
    </div>
  </div>

  <template v-else-if="summary">
    <!-- Welcome hero -->
    <div
      class="relative overflow-hidden rounded-3xl bg-gradient-to-br from-navy-800 via-navy-600 to-blue-500 p-6 text-white shadow-lg sm:p-7"
    >
      <div
        class="pointer-events-none absolute -right-16 -top-16 h-56 w-56 rounded-full bg-white/10 blur-2xl"
      ></div>
      <div
        class="pointer-events-none absolute -bottom-20 left-1/3 h-48 w-48 rounded-full bg-blue-300/20 blur-2xl"
      ></div>
      <div class="relative flex flex-wrap items-center justify-between gap-4">
        <div>
          <p class="text-xs font-medium uppercase tracking-widest text-blue-100">
            {{ todayLine }}
          </p>
          <h2 class="mt-1 text-2xl font-semibold tracking-tight">
            {{ greeting }} 👋
          </h2>
          <p class="mt-1 text-sm text-blue-50/90">
            {{ summary.upcomingBookings }} upcoming bookings ·
            {{ summary.totalLeads }} open leads ·
            {{ formatMoney(summary.monthlyRevenue) }} collected this month
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <a
            href="/bookings"
            class="rounded-xl bg-white px-4 py-2 text-sm font-semibold text-navy-700 shadow transition hover:brightness-95"
          >
            + New booking
          </a>
          <a
            href="/calculator"
            class="rounded-xl bg-white/15 px-4 py-2 text-sm font-semibold text-white ring-1 ring-white/25 backdrop-blur transition hover:bg-white/25"
          >
            Calculator
          </a>
          <a
            href="/reports"
            class="rounded-xl bg-white/15 px-4 py-2 text-sm font-semibold text-white ring-1 ring-white/25 backdrop-blur transition hover:bg-white/25"
          >
            Reports
          </a>
        </div>
      </div>
    </div>

    <!-- KPI cards -->
    <div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
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

    <!-- Operations -->
    <div class="mt-4 grid grid-cols-1 gap-4 xl:grid-cols-3">
      <section class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7 xl:col-span-2">
        <div class="flex items-baseline justify-between">
          <div>
            <h2 class="font-semibold tracking-tight text-gray-900">Upcoming bookings</h2>
            <p class="text-xs text-gray-400">Next on the schedule</p>
          </div>
          <a href="/bookings" class="text-xs font-medium text-navy-600 hover:underline">
            View all →
          </a>
        </div>
        <ul v-if="recentBookings.length" class="mt-4 space-y-2.5">
          <li
            v-for="b in recentBookings"
            :key="b.id"
            class="flex items-center gap-3.5 rounded-2xl bg-gray-50/70 p-3 ring-1 ring-gray-100 transition hover:bg-gray-50"
          >
            <div
              class="flex h-12 w-12 shrink-0 flex-col items-center justify-center rounded-xl bg-navy-600/5 ring-1 ring-navy-100"
            >
              <span class="text-base font-semibold leading-none text-navy-700">
                {{ bookingDay(b.scheduledFor) }}
              </span>
              <span class="mt-0.5 text-[10px] font-medium uppercase leading-none text-navy-500">
                {{ bookingMonth(b.scheduledFor) }}
              </span>
            </div>
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium text-gray-900">{{ b.customerName }}</p>
              <p class="truncate text-xs text-gray-500">
                {{ b.bookingNumber }} · {{ serviceLabel(b.serviceType) }} · {{ bookingTime(b.scheduledFor) }}
              </p>
            </div>
            <span
              class="shrink-0 rounded-full px-2.5 py-1 text-[11px] font-semibold"
              :class="bookingPill[b.status] ?? 'bg-gray-100 text-gray-600'"
            >
              {{ statusText(b.status) }}
            </span>
          </li>
        </ul>
        <div v-else class="mt-4 rounded-2xl bg-gray-50 p-5 text-sm text-gray-500 ring-1 ring-gray-100">
          No bookings yet.
          <a href="/bookings" class="font-medium text-navy-600">Create your first booking →</a>
        </div>
      </section>

      <section class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7">
        <div class="flex items-baseline justify-between">
          <div>
            <h2 class="font-semibold tracking-tight text-gray-900">Latest leads</h2>
            <p class="text-xs text-gray-400">Fresh opportunities</p>
          </div>
          <a href="/leads" class="text-xs font-medium text-navy-600 hover:underline">
            View all →
          </a>
        </div>
        <ul v-if="recentLeads.length" class="mt-4 space-y-2.5">
          <li
            v-for="lead in recentLeads"
            :key="lead.id"
            class="flex items-center gap-3 rounded-2xl bg-gray-50/70 p-3 ring-1 ring-gray-100 transition hover:bg-gray-50"
          >
            <span
              class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-navy-500 to-blue-600 text-[11px] font-semibold text-white"
            >
              {{ initials(`${lead.firstName} ${lead.lastName}`) }}
            </span>
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium text-gray-900">
                {{ lead.firstName }} {{ lead.lastName }}
              </p>
              <p class="truncate text-xs text-gray-500">{{ lead.email }}</p>
            </div>
            <span
              class="shrink-0 rounded-full px-2.5 py-1 text-[11px] font-semibold"
              :class="leadPill[lead.status] ?? 'bg-gray-100 text-gray-600'"
            >
              {{ statusText(lead.status) }}
            </span>
          </li>
        </ul>
        <div v-else class="mt-4 rounded-2xl bg-gray-50 p-5 text-sm text-gray-500 ring-1 ring-gray-100">
          No leads yet.
          <a href="/leads" class="font-medium text-navy-600">Create your first lead →</a>
        </div>
      </section>
    </div>

    <!-- Analytics -->
    <div class="mt-4 grid grid-cols-1 gap-4 xl:grid-cols-3">
      <section class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7 xl:col-span-2">
        <div class="flex items-baseline justify-between">
          <div>
            <h2 class="font-semibold tracking-tight text-gray-900">Revenue trend</h2>
            <p class="text-xs text-gray-400">Collected payments per month</p>
          </div>
          <a href="/reports" class="text-xs font-medium text-navy-600 hover:underline">
            Full reports →
          </a>
        </div>
        <TrendChart
          class="mt-4"
          :values="summary.revenueByMonth.map((m) => m.collected)"
          :labels="summary.revenueByMonth.map((m) => m.month)"
          :format-value="formatCompact"
          line-color="#10b981"
        />
      </section>

      <section class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7">
        <div class="flex items-baseline justify-between">
          <div>
            <h2 class="font-semibold tracking-tight text-gray-900">Bookings mix</h2>
            <p class="text-xs text-gray-400">By status</p>
          </div>
        </div>
        <DonutChart
          class="mt-5"
          compact
          :segments="bookingSegments"
          :size="170"
          :center-top="String(totalBookings)"
          center-bottom="total bookings"
        />
      </section>
    </div>

    <p class="mt-4 text-xs text-gray-400">
      Live data from {{ API_URL ?? "the API" }} · updated just now.
    </p>
  </template>
</template>
