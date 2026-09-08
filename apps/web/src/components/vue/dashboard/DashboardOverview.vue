<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ApiError } from "../../../lib/api";
import { clearSession } from "../../../lib/auth";
import {
  formatMoney,
  getWorkspaceSettings,
} from "../../../lib/settings";
import { getReportSummary, type ReportSummary } from "../../../lib/reports";
import { getLeads, type Lead } from "../../../lib/leads";
import { getBookings, type Booking } from "../../../lib/bookings";
import { showToast } from "../../../lib/toast";

const API_URL = import.meta.env.PUBLIC_API_URL as string | undefined;

const summary = ref<ReportSummary | null>(null);
const recentLeads = ref<Lead[]>([]);
const recentBookings = ref<Booking[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const errorStatus = ref<number | null>(null);

const isAuthError = computed(() => errorStatus.value === 401);

const kpis = computed(() => {
  if (!summary.value) return [];
  return [
    {
      label: "Total Leads",
      value: String(summary.value.totalLeads),
      href: "/leads",
    },
    {
      label: "Active Customers",
      value: String(summary.value.activeCustomers),
      href: "/customers",
    },
    {
      label: "Upcoming Bookings",
      value: String(summary.value.upcomingBookings),
      href: "/bookings",
    },
    {
      label: "Revenue (Month)",
      value: formatMoney(summary.value.monthlyRevenue),
      href: "/payments",
    },
  ];
});

const revenueMax = computed(() =>
  Math.max(0, ...(summary.value?.revenueByMonth.map((m) => m.collected) ?? [])),
);
const bookingsMax = computed(() =>
  Math.max(0, ...(summary.value?.bookingsByStatus.map((b) => b.count) ?? [])),
);
const leadsMax = computed(() =>
  Math.max(0, ...(summary.value?.leadsByStatus.map((l) => l.count) ?? [])),
);
const cleanersMax = computed(() =>
  Math.max(0, ...(summary.value?.cleanerProductivity.map((c) => c.completedBookings) ?? [])),
);

const conversionRate = computed(() => summary.value?.leadConversionRate ?? null);

function barPct(value: number, max: number): number {
  if (value <= 0 || max <= 0) return 0;
  return Math.min(100, Math.max(6, (value / max) * 100));
}

function formatDate(value: string): string {
  try {
    return new Date(value).toLocaleDateString(undefined, {
      month: "short",
      day: "numeric",
    });
  } catch {
    return value;
  }
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
    class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4"
  >
    <div
      v-for="i in 4"
      :key="i"
      class="animate-pulse rounded-xl border border-gray-200 bg-white p-6"
    >
      <div class="h-4 w-24 rounded bg-gray-100"></div>
      <div class="mt-3 h-8 w-16 rounded bg-gray-100"></div>
    </div>
  </div>

  <div
    v-else-if="error"
    class="rounded-xl border p-6"
    :class="
      isAuthError
        ? 'border-amber-200 bg-amber-50'
        : 'border-red-200 bg-red-50'
    "
  >
    <h2
      class="text-base font-semibold"
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
        class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-semibold text-white hover:bg-navy-700"
        @click="signInAgain"
      >
        Sign in again
      </button>
      <button
        type="button"
        class="rounded-lg px-4 py-2 text-sm font-semibold"
        :class="
          isAuthError
            ? 'border border-amber-300 bg-white text-amber-800 hover:bg-amber-100'
            : 'bg-red-600 text-white hover:bg-red-700'
        "
        @click="load"
      >
        Retry
      </button>
    </div>
  </div>

  <template v-else-if="summary">
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
        <div class="flex items-center justify-between">
          <h2 class="text-base font-semibold text-gray-900">Recent Leads</h2>
          <a
            href="/leads"
            class="text-sm font-medium text-navy-600 hover:text-navy-700"
            >View all →</a
          >
        </div>
        <ul v-if="recentLeads.length" class="mt-4 divide-y divide-navy-100">
          <li
            v-for="lead in recentLeads"
            :key="lead.id"
            class="flex items-center justify-between gap-3 py-2.5"
          >
            <div class="min-w-0">
              <p class="truncate text-sm font-medium text-gray-900">
                {{ lead.firstName }} {{ lead.lastName }}
              </p>
              <p class="truncate text-xs text-gray-500">{{ lead.email }}</p>
            </div>
            <span
              class="shrink-0 rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-600"
              >{{ lead.status }}</span
            >
          </li>
        </ul>
        <div v-else class="mt-4 rounded-lg bg-gray-50 p-4 text-sm text-gray-500">
          No leads yet.
          <a href="/leads" class="font-medium text-navy-600">Create your first lead →</a>
        </div>
      </div>

      <div class="rounded-xl border border-gray-200 bg-white p-6">
        <div class="flex items-center justify-between">
          <h2 class="text-base font-semibold text-gray-900">
            Upcoming Bookings
          </h2>
          <a
            href="/bookings"
            class="text-sm font-medium text-navy-600 hover:text-navy-700"
            >View all →</a
          >
        </div>
        <ul v-if="recentBookings.length" class="mt-4 divide-y divide-navy-100">
          <li
            v-for="b in recentBookings"
            :key="b.id"
            class="flex items-center justify-between gap-3 py-2.5"
          >
            <div class="min-w-0">
              <p class="truncate text-sm font-medium text-gray-900">
                {{ b.customerName }}
              </p>
              <p class="truncate text-xs text-gray-500">
                {{ b.bookingNumber }} · {{ formatDate(b.scheduledFor) }} ·
                {{ b.status }}
              </p>
            </div>
            <span
              class="shrink-0 rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-600"
              >{{ b.serviceType }}</span
            >
          </li>
        </ul>
        <div v-else class="mt-4 rounded-lg bg-gray-50 p-4 text-sm text-gray-500">
          No bookings yet.
          <a href="/bookings" class="font-medium text-navy-600">Create your first booking →</a>
        </div>
      </div>
    </div>

    <div class="mt-6">
      <h2 class="text-base font-semibold text-gray-900">Reporting</h2>
      <div class="mt-3 grid grid-cols-1 gap-4 xl:grid-cols-2">
        <div class="rounded-xl border border-gray-200 bg-white p-6">
          <h3 class="text-sm font-semibold text-gray-900">Revenue (6 months)</h3>
          <div v-if="summary.revenueByMonth.length" class="mt-4 flex">
            <div
              v-for="m in summary.revenueByMonth"
              :key="m.month"
              class="flex flex-1 flex-col items-center gap-1.5"
            >
              <div class="flex h-36 w-full items-end">
                <div
                  class="w-full rounded-t-md bg-emerald-500"
                  :style="{ height: barPct(m.collected, revenueMax) + '%' }"
                  :title="`${m.month}: ${formatMoney(m.collected)}`"
                ></div>
              </div>
              <span class="text-[11px] font-medium text-gray-500">{{
                m.month
              }}</span>
            </div>
          </div>
          <div
            v-else
            class="mt-4 rounded-lg bg-gray-50 p-4 text-sm text-gray-500"
          >
            No paid revenue yet.
          </div>
        </div>

        <div class="rounded-xl border border-gray-200 bg-white p-6">
          <h3 class="text-sm font-semibold text-gray-900">Bookings by status</h3>
          <ul v-if="summary.bookingsByStatus.length" class="mt-4 space-y-2.5">
            <li
              v-for="b in summary.bookingsByStatus"
              :key="b.status"
              class="flex items-center gap-3"
            >
              <span class="w-28 shrink-0 text-xs text-gray-600">{{
                b.status
              }}</span>
              <div class="h-2.5 flex-1 rounded-full bg-gray-100">
                <div
                  class="h-full rounded-full bg-navy-600"
                  :style="{ width: barPct(b.count, bookingsMax) + '%' }"
                ></div>
              </div>
              <span class="w-8 shrink-0 text-right text-xs font-semibold text-gray-700">{{
                b.count
              }}</span>
            </li>
          </ul>
          <div
            v-else
            class="mt-4 rounded-lg bg-gray-50 p-4 text-sm text-gray-500"
          >
            No bookings yet.
          </div>
        </div>

        <div class="rounded-xl border border-gray-200 bg-white p-6">
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-semibold text-gray-900">Leads pipeline</h3>
            <span
              v-if="conversionRate !== null"
              class="rounded-full bg-sky-50 px-2.5 py-1 text-xs font-semibold text-sky-700"
              >{{ conversionRate.toFixed(1) }}% won</span
            >
          </div>
          <ul v-if="summary.leadsByStatus.length" class="mt-4 space-y-2.5">
            <li
              v-for="l in summary.leadsByStatus"
              :key="l.status"
              class="flex items-center gap-3"
            >
              <span class="w-28 shrink-0 text-xs text-gray-600">{{
                l.status
              }}</span>
              <div class="h-2.5 flex-1 rounded-full bg-gray-100">
                <div
                  class="h-full rounded-full bg-sky-500"
                  :style="{ width: barPct(l.count, leadsMax) + '%' }"
                ></div>
              </div>
              <span class="w-8 shrink-0 text-right text-xs font-semibold text-gray-700">{{
                l.count
              }}</span>
            </li>
          </ul>
          <div
            v-else
            class="mt-4 rounded-lg bg-gray-50 p-4 text-sm text-gray-500"
          >
            No leads yet.
          </div>
        </div>

        <div class="rounded-xl border border-gray-200 bg-white p-6">
          <h3 class="text-sm font-semibold text-gray-900">
            Cleaner productivity
          </h3>
          <ul
            v-if="summary.cleanerProductivity.length"
            class="mt-4 divide-y divide-gray-100"
          >
            <li v-for="c in summary.cleanerProductivity" :key="c.cleanerName" class="py-2">
              <div class="flex items-center justify-between gap-3">
                <p class="truncate text-sm font-medium text-gray-900">
                  {{ c.cleanerName }}
                </p>
                <p class="shrink-0 text-xs text-gray-500">
                  {{ c.completedBookings }} completed
                  <template v-if="c.upcomingBookings > 0">
                    · {{ c.upcomingBookings }} upcoming
                  </template>
                </p>
              </div>
              <div class="mt-1.5 h-2 rounded-full bg-gray-100">
                <div
                  class="h-full rounded-full bg-navy-600"
                  :style="{ width: barPct(c.completedBookings, cleanersMax) + '%' }"
                ></div>
              </div>
            </li>
          </ul>
          <div
            v-else
            class="mt-4 rounded-lg bg-gray-50 p-4 text-sm text-gray-500"
          >
            No cleaner assignments yet.
          </div>
        </div>
      </div>
    </div>

    <p class="mt-4 text-xs text-gray-400">
      Live data from {{ API_URL ?? "the API" }} · updated just now.
    </p>
  </template>
</template>
