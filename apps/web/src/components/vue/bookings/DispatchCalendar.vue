<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { getBookings, type Booking, type BookingStatus } from "../../../lib/bookings";
import { serviceLabel } from "../../../lib/settings";
import DispatchBookingModal from "./DispatchBookingModal.vue";

const statusLabels: Record<BookingStatus, string> = {
  pending: "Pending",
  confirmed: "Confirmed",
  in_progress: "In Progress",
  completed: "Completed",
  cancelled: "Cancelled",
  no_show: "No Show",
  rescheduled: "Rescheduled",
};

const statusPills: Record<BookingStatus, string> = {
  pending: "bg-amber-100 text-amber-700",
  confirmed: "bg-blue-100 text-blue-700",
  in_progress: "bg-sky-100 text-sky-700",
  completed: "bg-emerald-100 text-emerald-700",
  cancelled: "bg-gray-100 text-gray-500",
  no_show: "bg-rose-100 text-rose-700",
  rescheduled: "bg-violet-100 text-violet-700",
};

const statusAccent: Record<BookingStatus, string> = {
  pending: "#f59e0b",
  confirmed: "#2563eb",
  in_progress: "#0ea5e9",
  completed: "#10b981",
  cancelled: "#cbd5e1",
  no_show: "#f43f5e",
  rescheduled: "#8b5cf6",
};

const serviceTypeLabels: Record<string, string> = {
  house_cleaning: "House Cleaning",
  condo_cleaning: "Condo Cleaning",
  deep_cleaning: "Deep Cleaning",
  move_in_out: "Move In/Out",
  after_renovation: "After Renovation",
  office_cleaning: "Office Cleaning",
  junk_removal: "Junk Removal",
  aircon_service: "Aircon Service",
};

function labelForService(id: string): string {
  const live = serviceLabel(id);
  return live !== id ? live : (serviceTypeLabels[id] ?? id);
}

const HOUR_PX = 56;
const DAY_LEN_MS = 24 * 60 * 60 * 1000;

const weekStart = ref(startOfWeek(new Date()));
const bookings = ref<Booking[]>([]);
const loading = ref(true);
const loadError = ref("");
const search = ref("");
const statusFilter = ref<BookingStatus | "">("");
const today = new Date();
const openModalBookings = ref<Booking[] | null>(null);

function startOfWeek(date: Date): Date {
  const d = new Date(date);
  d.setHours(0, 0, 0, 0);
  const day = d.getDay();
  const diff = day === 0 ? -6 : 1 - day;
  d.setDate(d.getDate() + diff);
  return d;
}

const days = computed(() => {
  const start = weekStart.value;
  return Array.from({ length: 7 }, (_, i) => new Date(start.getTime() + i * DAY_LEN_MS));
});

const weekLabel = computed(() => {
  const d = days.value;
  const fmt = (x: Date) =>
    x.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
  return `${fmt(d[0])} – ${fmt(d[6])}`;
});

const filtersApplied = computed(
  () => search.value.trim() !== "" || statusFilter.value !== "",
);

const visibleBookings = computed(() => {
  const q = search.value.trim().toLowerCase();
  return bookings.value
    .filter((b) => {
      if (statusFilter.value && b.status !== statusFilter.value) return false;
      return (
        !q ||
        b.customerName.toLowerCase().includes(q) ||
        b.bookingNumber.toLowerCase().includes(q) ||
        b.address.toLowerCase().includes(q)
      );
    })
    .sort((a, b) => a.scheduledFor.localeCompare(b.scheduledFor));
});

/** Visible hours window derived from the week's jobs (no dead night hours). */
const windowStart = computed(() => {
  if (visibleBookings.value.length === 0) return 7;
  const min = Math.min(
    ...visibleBookings.value.map((b) => {
      const d = new Date(b.scheduledFor);
      return d.getHours() + d.getMinutes() / 60;
    }),
  );
  return Math.max(0, Math.min(22, Math.floor(min) - 1));
});

const windowEnd = computed(() => {
  if (visibleBookings.value.length === 0) return 19;
  const max = Math.max(
    ...visibleBookings.value.map((b) => {
      const end = new Date(new Date(b.scheduledFor).getTime() + b.durationMinutes * 60_000);
      return end.getHours() + end.getMinutes() / 60;
    }),
  );
  return Math.max(windowStart.value + 6, Math.min(24, Math.ceil(max) + 1));
});

const hours = computed(() => {
  const out: number[] = [];
  for (let h = windowStart.value; h < windowEnd.value; h++) out.push(h);
  return out;
});

const gridHeight = computed(() => (windowEnd.value - windowStart.value) * HOUR_PX);

function hourLabel(h: number): string {
  const ap = h < 12 ? "AM" : "PM";
  const hh = h % 12 === 0 ? 12 : h % 12;
  return `${hh} ${ap}`;
}

const stats = computed(() => {
  const list = visibleBookings.value;
  return {
    jobs: list.length,
    unassigned: list.filter((b) => (b.cleaners ?? []).length === 0 && !b.assignedCleaner).length,
    completed: list.filter((b) => b.status === "completed").length,
  };
});

function sameDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}

function bookingsForDay(day: Date): Booking[] {
  return visibleBookings.value.filter((b) => sameDay(new Date(b.scheduledFor), day));
}

function bookedCount(day: Date): number {
  return bookingsForDay(day).length;
}

function openBooking(b: Booking) {
  openModalBookings.value = [b];
}

function openDay(day: Date) {
  const list = bookingsForDay(day);
  if (list.length === 0) return;
  openModalBookings.value = list;
}

function topOf(iso: string): number {
  const d = new Date(iso);
  return (d.getHours() * 60 + d.getMinutes() - windowStart.value * 60) * (HOUR_PX / 60);
}

function heightOf(durationMinutes: number): number {
  return Math.max((durationMinutes / 60) * HOUR_PX, 20);
}

function timeRange(iso: string, durationMinutes: number): string {
  const start = new Date(iso);
  const end = new Date(start.getTime() + durationMinutes * 60_000);
  const fmt = (x: Date) =>
    x.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
  return `${fmt(start)} – ${fmt(end)}`;
}

function crewName(b: Booking): string {
  return (
    b.cleaners?.find((c) => c.role === "primary")?.name ??
    b.cleaners?.[0]?.name ??
    b.assignedCleaner ??
    "Unassigned"
  );
}

function crewInitial(name: string): string {
  return name
    .split(" ")
    .map((w) => w[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("")
    .toUpperCase();
}

function isPast(day: Date): boolean {
  return day.getTime() < startOfWeek(today).getTime();
}

/** Position of the "now" line inside today's column (null when off-screen). */
const nowTop = computed(() => {
  if (!days.value.some((d) => sameDay(d, today))) return null;
  const mins = today.getHours() * 60 + today.getMinutes() - windowStart.value * 60;
  const top = mins * (HOUR_PX / 60);
  if (top < 0 || top > gridHeight.value) return null;
  return top;
});

async function load() {
  loading.value = true;
  loadError.value = "";
  try {
    const from = weekStart.value.toISOString();
    const to = new Date(weekStart.value.getTime() + 7 * DAY_LEN_MS).toISOString();
    bookings.value = await getBookings({ from, to, limit: 200 });
  } catch (e) {
    loadError.value =
      e instanceof Error ? e.message : "Failed to load bookings";
  } finally {
    loading.value = false;
  }
}

function shiftWeek(weeks: number) {
  weekStart.value = new Date(weekStart.value.getTime() + weeks * 7 * DAY_LEN_MS);
  load();
}

function goToday() {
  weekStart.value = startOfWeek(new Date());
  load();
}

onMounted(load);

const gridCols = "grid-template-columns: 3.5rem repeat(7, minmax(8.5rem, 1fr));";
</script>

<template>
  <div class="space-y-4">
    <!-- Toolbar -->
    <div class="rounded-3xl bg-white p-4 shadow-sm ring-1 ring-gray-100 sm:p-5">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex items-center gap-3">
          <span
            class="flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-navy-600 to-blue-500 text-white shadow-sm"
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
                d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
              />
            </svg>
          </span>
          <div>
            <h2 class="font-semibold tracking-tight text-gray-900">Dispatch board</h2>
            <p class="text-xs text-gray-400">{{ weekLabel }}</p>
          </div>
          <div class="ml-2 hidden items-center gap-2 lg:flex">
            <span class="inline-flex items-center gap-1.5 rounded-full bg-navy-50 px-3 py-1 text-xs font-semibold text-navy-700 ring-1 ring-navy-100">
              {{ stats.jobs }} jobs
            </span>
            <span
              v-if="stats.unassigned > 0"
              class="inline-flex items-center gap-1.5 rounded-full bg-amber-100 px-3 py-1 text-xs font-semibold text-amber-700 ring-1 ring-amber-200"
            >
              {{ stats.unassigned }} unassigned
            </span>
            <span
              class="inline-flex items-center gap-1.5 rounded-full bg-emerald-100 px-3 py-1 text-xs font-semibold text-emerald-700 ring-1 ring-emerald-200"
            >
              {{ stats.completed }} done
            </span>
          </div>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <div class="flex overflow-hidden rounded-xl ring-1 ring-gray-200">
            <button
              type="button"
              class="bg-white px-3 py-2 text-sm font-medium text-gray-600 transition hover:bg-gray-50"
              @click="shiftWeek(-1)"
              aria-label="Previous week"
            >
              ‹
            </button>
            <button
              type="button"
              class="border-x border-gray-200 bg-white px-3 py-2 text-xs font-semibold text-navy-700 transition hover:bg-navy-50"
              @click="goToday"
            >
              Today
            </button>
            <button
              type="button"
              class="bg-white px-3 py-2 text-sm font-medium text-gray-600 transition hover:bg-gray-50"
              @click="shiftWeek(1)"
              aria-label="Next week"
            >
              ›
            </button>
          </div>
          <input
            v-model="search"
            type="search"
            placeholder="Search week…"
            class="w-40 rounded-xl bg-white px-3.5 py-2 text-sm text-gray-900 shadow-sm ring-1 ring-gray-200 transition focus:outline-none focus:ring-2 focus:ring-navy-500"
          />
          <select
            v-model="statusFilter"
            class="rounded-xl bg-white py-2 pl-3.5 text-sm shadow-sm ring-1 ring-gray-200 transition focus:outline-none focus:ring-2 focus:ring-navy-500"
          >
            <option value="">All statuses</option>
            <option v-for="(label, value) in statusLabels" :key="value" :value="value">
              {{ label }}
            </option>
          </select>
          <a
            href="/bookings"
            class="rounded-xl bg-gradient-to-r from-navy-600 to-blue-500 px-4 py-2 text-sm font-semibold text-white shadow-sm transition hover:brightness-110"
          >
            Manage Bookings
          </a>
        </div>
      </div>
      <div class="mt-3 flex flex-wrap gap-2 lg:hidden">
        <span class="inline-flex items-center gap-1.5 rounded-full bg-navy-50 px-3 py-1 text-xs font-semibold text-navy-700 ring-1 ring-navy-100">
          {{ stats.jobs }} jobs
        </span>
        <span
          v-if="stats.unassigned > 0"
          class="inline-flex items-center gap-1.5 rounded-full bg-amber-100 px-3 py-1 text-xs font-semibold text-amber-700 ring-1 ring-amber-200"
        >
          {{ stats.unassigned }} unassigned
        </span>
        <span
          class="inline-flex items-center gap-1.5 rounded-full bg-emerald-100 px-3 py-1 text-xs font-semibold text-emerald-700 ring-1 ring-emerald-200"
        >
          {{ stats.completed }} done
        </span>
      </div>
    </div>

    <!-- Calendar -->
    <div class="overflow-hidden rounded-3xl bg-white shadow-sm ring-1 ring-gray-100">
      <div v-if="loading" class="space-y-3 p-8">
        <div class="h-12 animate-pulse rounded-2xl bg-gray-100"></div>
        <div class="h-64 animate-pulse rounded-2xl bg-gray-50"></div>
      </div>

      <div
        v-else-if="visibleBookings.length === 0"
        class="flex flex-col items-center justify-center px-6 py-20 text-center"
      >
        <template v-if="loadError">
          <span class="flex h-12 w-12 items-center justify-center rounded-2xl bg-red-100 text-xl">⚠️</span>
          <p class="mt-3 text-sm font-semibold text-gray-900">Couldn't load bookings</p>
          <p class="mt-1 text-sm text-red-600">{{ loadError }}</p>
          <button
            type="button"
            class="mt-4 rounded-xl bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-700"
            @click="load"
          >
            Retry
          </button>
        </template>
        <template v-else>
          <span class="flex h-12 w-12 items-center justify-center rounded-2xl bg-navy-50 text-xl">🗓️</span>
          <p class="mt-3 text-sm font-semibold text-gray-900">No bookings this week</p>
          <p class="mt-1 text-sm text-gray-500">
            {{
              filtersApplied
                ? "Try adjusting your filters."
                : "Create bookings from the Bookings page to see them on the board."
            }}
          </p>
        </template>
      </div>

      <div v-else class="overflow-x-auto">
        <div class="min-w-[920px]">
          <!-- Day headers -->
          <div class="grid border-b border-gray-100 bg-gray-50/60" :style="gridCols">
            <div></div>
            <div
              v-for="(day, i) in days"
              :key="i"
              role="button"
              :tabindex="bookedCount(day) > 0 ? 0 : -1"
              :aria-label="`View bookings for ${day.toLocaleDateString(undefined, { weekday: 'long', month: 'long', day: 'numeric' })}`"
              :class="[
                'flex flex-col items-center gap-0.5 border-l border-gray-100 py-2.5 text-center',
                sameDay(day, today) ? 'bg-navy-50/70' : '',
                isPast(day) && !sameDay(day, today) ? 'opacity-60' : '',
                bookedCount(day) > 0
                  ? 'cursor-pointer hover:bg-navy-50 focus:outline-none focus:ring-2 focus:ring-inset focus:ring-navy-500'
                  : '',
              ]"
              @click="openDay(day)"
              @keydown.enter.prevent="openDay(day)"
              @keydown.space.prevent="openDay(day)"
            >
              <span class="text-[11px] font-semibold uppercase tracking-widest text-gray-400">
                {{ day.toLocaleDateString(undefined, { weekday: "short" }) }}
              </span>
              <span
                :class="[
                  'flex h-8 w-8 items-center justify-center rounded-full text-sm font-semibold',
                  sameDay(day, today)
                    ? 'bg-navy-600 text-white shadow-sm'
                    : 'text-gray-900',
                ]"
              >
                {{ day.toLocaleDateString(undefined, { day: "numeric" }) }}
              </span>
              <span
                v-if="bookedCount(day) > 0"
                class="mt-0.5 inline-flex items-center gap-1 rounded-full bg-emerald-100 px-2 py-px text-[10px] font-semibold text-emerald-700"
              >
                <span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
                {{ bookedCount(day) }}
              </span>
              <span v-else class="mt-0.5 h-[18px] text-[10px] text-gray-300">—</span>
            </div>
          </div>

          <!-- Time grid -->
          <div class="grid" :style="gridCols">
            <!-- Gutter -->
            <div class="relative" :style="{ height: `${gridHeight}px` }">
              <span
                v-for="h in hours"
                :key="h"
                class="absolute right-2 -translate-y-1/2 text-[11px] font-medium tabular-nums text-gray-400"
                :style="{ top: `${(h - windowStart) * HOUR_PX}px` }"
              >
                {{ hourLabel(h) }}
              </span>
            </div>

            <!-- Day columns -->
            <div
              v-for="(day, i) in days"
              :key="i"
              class="relative border-l border-gray-100"
              :style="{ height: `${gridHeight}px` }"
            >
              <div
                v-for="h in hours"
                :key="h"
                class="pointer-events-none absolute left-0 right-0 border-t border-gray-100"
                :style="{ top: `${(h - windowStart) * HOUR_PX}px` }"
              />
              <div
                v-if="sameDay(day, today)"
                class="pointer-events-none absolute inset-0 bg-navy-50/50"
              />
              <div
                v-if="sameDay(day, today) && nowTop !== null"
                class="pointer-events-none absolute left-0 right-0 z-20"
                :style="{ top: `${nowTop}px` }"
              >
                <div class="h-0.5 rounded bg-rose-500 shadow-sm"></div>
                <div class="absolute -top-[3px] left-1 h-2 w-2 rounded-full bg-rose-500 ring-2 ring-rose-200"></div>
              </div>

              <div
                v-for="b in bookingsForDay(day)"
                :key="b.id"
                role="button"
                tabindex="0"
                aria-haspopup="dialog"
                :aria-label="`${b.customerName}, ${labelForService(b.serviceType)}, ${timeRange(b.scheduledFor, b.durationMinutes)}`"
                class="absolute left-1 right-1 z-10 cursor-pointer overflow-hidden rounded-xl border-l-4 bg-white p-2 shadow-sm ring-1 ring-gray-200 transition hover:-translate-y-px hover:shadow-md focus:outline-none focus:ring-2 focus:ring-navy-500"
                :style="{
                  top: `${topOf(b.scheduledFor)}px`,
                  height: `${heightOf(b.durationMinutes)}px`,
                  borderLeftColor: statusAccent[b.status],
                }"
                :title="`${timeRange(b.scheduledFor, b.durationMinutes)} — ${b.customerName} — click for details`"
                @click="openBooking(b)"
                @keydown.enter.prevent="openBooking(b)"
                @keydown.space.prevent="openBooking(b)"
              >
                <p class="truncate text-[11px] font-semibold text-navy-700">
                  {{ timeRange(b.scheduledFor, b.durationMinutes) }}
                </p>
                <p class="truncate text-xs font-semibold text-gray-900">
                  {{ b.customerName }}
                </p>
                <p class="truncate text-[11px] text-gray-500">
                  {{ labelForService(b.serviceType) }}
                </p>
                <div class="mt-1 flex items-center gap-1.5">
                  <span
                    class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-navy-100 text-[9px] font-bold text-navy-700"
                  >
                    {{ crewInitial(crewName(b)) }}
                  </span>
                  <span class="min-w-0 flex-1 truncate text-[11px] text-gray-600">
                    {{ crewName(b) }}
                  </span>
                  <span
                    :class="[
                      'shrink-0 rounded-full px-1.5 py-px text-[10px] font-semibold',
                      statusPills[b.status],
                    ]"
                  >
                    {{ statusLabels[b.status] }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <DispatchBookingModal
      v-if="openModalBookings"
      :bookings="openModalBookings"
      @close="openModalBookings = null"
    />
  </div>
</template>
