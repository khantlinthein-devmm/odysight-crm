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

const API_URL = import.meta.env.PUBLIC_API_URL as string | undefined;

const summary = ref<ReportSummary | null>(null);
const recentLeads = ref<Lead[]>([]);
const recentBookings = ref<Booking[]>([]);
const allBookings = ref<Booking[]>([]);
const allLeads = ref<Lead[]>([]);
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
      label: "Today's Jobs",
      value: String(todayJobs.value.length),
      sub: `${completedToday.value} done · ${remainingToday.value} left`,
      href: "/dispatch",
      tile: "from-amber-400 to-orange-600",
      icon: "M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z",
    },
    {
      label: "Needs Crew",
      value: String(unassignedJobs.value.length),
      sub: "Unassigned bookings",
      href: "/dispatch",
      tile: "from-rose-400 to-red-600",
      icon: "M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z",
    },
    {
      label: "New Leads",
      value: String(newLeadsCount.value),
      sub: "Waiting for contact",
      href: "/leads",
      tile: "from-blue-500 to-navy-600",
      icon: "M17 20h5v-2a4 4 0 00-3-3.87M9 20H4v-2a4 4 0 013-3.87m6-1.13a4 4 0 10-4-4 4 4 0 004 4zm6-4a3 3 0 11-3-3",
    },
    {
      label: "Upcoming",
      value: String(summary.value.upcomingBookings),
      sub: "Scheduled ahead",
      href: "/bookings",
      tile: "from-emerald-400 to-green-600",
      icon: "M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z",
    },
  ];
});

function isToday(iso: string): boolean {
  try {
    const d = new Date(iso);
    const now = new Date();
    return (
      d.getFullYear() === now.getFullYear() &&
      d.getMonth() === now.getMonth() &&
      d.getDate() === now.getDate()
    );
  } catch {
    return false;
  }
}

function isUnassigned(b: Booking): boolean {
  return (b.cleaners ?? []).length === 0 && !b.assignedCleaner;
}

const todayJobs = computed(() => allBookings.value.filter((b) => isToday(b.scheduledFor)));
const completedToday = computed(() => todayJobs.value.filter((b) => b.status === "completed").length);
const remainingToday = computed(() => todayJobs.value.length - completedToday.value);
const unassignedJobs = computed(() => allBookings.value.filter((b) => isUnassigned(b) && (b.status === "pending" || b.status === "confirmed")));
const newLeadsCount = computed(() => allLeads.value.filter((l) => l.status === "new").length);

// Actionable queues — what needs a human right now.
const needsAttention = computed(() => {
  const items: { label: string; detail: string; href: string; tone: string }[] = [];
  for (const b of unassignedJobs.value.slice(0, 3)) {
    items.push({
      label: `Assign crew: ${b.customerName}`,
      detail: `${b.bookingNumber} · ${bookingTime(b.scheduledFor)}`,
      href: "/dispatch",
      tone: "bg-rose-50 text-rose-700 ring-rose-200",
    });
  }
  for (const l of allLeads.value.filter((x) => x.status === "new").slice(0, 3)) {
    items.push({
      label: `Contact lead: ${l.firstName} ${l.lastName}`,
      detail: l.phone || l.email || "New inquiry",
      href: `/leads/${l.id}`,
      tone: "bg-blue-50 text-blue-700 ring-blue-200",
    });
  }
  return items.slice(0, 5);
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
    const [s, leads, bookings, opsLeads, opsBookings] = await Promise.all([
      getReportSummary(),
      getLeads({ limit: 5 }),
      getBookings({ limit: 5 }),
      getLeads({ limit: 100 }),
      getBookings({ limit: 100 }),
    ]);
    summary.value = s;
    recentLeads.value = leads.slice(0, 5);
    recentBookings.value = bookings.slice(0, 5);
    allLeads.value = opsLeads;
    allBookings.value = opsBookings;
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
            {{ todayJobs.length }} jobs today ·
            {{ unassignedJobs.length }} need crew ·
            {{ newLeadsCount }} new leads waiting
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
              {{ initials(`${lead.firstName} ${lead.lastName}`.trim()) }}
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

    <!-- Operations focus: needs attention + quick actions (analytics lives in Reports) -->
    <div class="mt-4 grid grid-cols-1 gap-4 xl:grid-cols-3">
      <section class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7 xl:col-span-2">
        <div class="flex items-baseline justify-between">
          <div>
            <h2 class="font-semibold tracking-tight text-gray-900">⚡ Needs attention</h2>
            <p class="text-xs text-gray-400">Unassigned jobs & new leads — act now</p>
          </div>
          <a href="/dispatch" class="text-xs font-medium text-navy-600 hover:underline">
            Open dispatch →
          </a>
        </div>
        <ul v-if="needsAttention.length" class="mt-4 space-y-2.5">
          <li
            v-for="(item, i) in needsAttention"
            :key="i"
            class="flex items-center gap-3 rounded-2xl bg-gray-50/70 p-3 ring-1 ring-gray-100 transition hover:bg-gray-50"
          >
            <span :class="['shrink-0 rounded-full px-2.5 py-1 text-[11px] font-bold ring-1', item.tone]">
              {{ i + 1 }}
            </span>
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-semibold text-gray-900">{{ item.label }}</p>
              <p class="truncate text-xs text-gray-500">{{ item.detail }}</p>
            </div>
            <a :href="item.href" class="shrink-0 rounded-lg bg-navy-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-navy-700">
              Handle
            </a>
          </li>
        </ul>
        <div v-else class="mt-4 rounded-2xl bg-emerald-50 p-5 text-sm text-emerald-700 ring-1 ring-emerald-200">
          ✅ All clear — every job has a crew and every lead has been contacted.
        </div>
      </section>

      <section class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7">
        <div>
          <h2 class="font-semibold tracking-tight text-gray-900">Quick actions</h2>
          <p class="text-xs text-gray-400">Daily workflows</p>
        </div>
        <div class="mt-4 grid grid-cols-2 gap-2.5">
          <a href="/bookings" class="rounded-2xl bg-navy-600 p-4 text-white shadow-sm transition hover:brightness-110">
            <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
              <path stroke-linecap="round" stroke-linejoin="round" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <p class="mt-2 text-sm font-semibold">New booking</p>
          </a>
          <a href="/dispatch" class="rounded-2xl bg-amber-400 p-4 text-slate-950 shadow-sm transition hover:brightness-105">
            <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7" />
            </svg>
            <p class="mt-2 text-sm font-semibold">Dispatch</p>
          </a>
          <a href="/leads" class="rounded-2xl bg-blue-100 p-4 text-blue-900 ring-1 ring-blue-200 transition hover:bg-blue-200">
            <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
              <path stroke-linecap="round" stroke-linejoin="round" d="M18 9v3m0 0v3m0-3h3m-3 0h-3m-2-5a4 4 0 11-8 0 4 4 0 018 0zM3 20a6 6 0 0112 0v1H3v-1z" />
            </svg>
            <p class="mt-2 text-sm font-semibold">New lead</p>
          </a>
          <a href="/calculator" class="rounded-2xl bg-emerald-100 p-4 text-emerald-900 ring-1 ring-emerald-200 transition hover:bg-emerald-200">
            <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9 7h6m-5 4h4M5 3h14a2 2 0 012 2v14a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2zm3 16h.01M9 19h.01M12 19h.01M15 19h.01M18 19h.01M6 19h.01" />
            </svg>
            <p class="mt-2 text-sm font-semibold">Calculator</p>
          </a>
          <a href="/checklists" class="rounded-2xl bg-violet-100 p-4 text-violet-900 ring-1 ring-violet-200 transition hover:bg-violet-200">
            <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
            </svg>
            <p class="mt-2 text-sm font-semibold">Checklists</p>
          </a>
          <a href="/reports" class="rounded-2xl bg-slate-900 p-4 text-amber-300 ring-1 ring-slate-700 transition hover:bg-slate-800">
            <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
            </svg>
            <p class="mt-2 text-sm font-semibold">Reports</p>
          </a>
        </div>
      </section>
    </div>

    <p class="mt-4 text-xs text-gray-400">
      Live data from {{ API_URL ?? "the API" }} · updated just now.
    </p>
  </template>
</template>
