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

const statusStyles: Record<BookingStatus, string> = {
  pending: "bg-amber-50 text-amber-700",
  confirmed: "bg-navy-50 text-navy-700",
  in_progress: "bg-green-50 text-green-700",
  completed: "bg-gray-100 text-gray-600",
  cancelled: "bg-gray-100 text-gray-400",
  no_show: "bg-red-50 text-red-700",
  rescheduled: "bg-navy-50 text-navy-700",
};

const statusAccent: Record<BookingStatus, string> = {
  pending: "border-l-amber-400",
  confirmed: "border-l-navy-600",
  in_progress: "border-l-green-500",
  completed: "border-l-gray-400",
  cancelled: "border-l-gray-300",
  no_show: "border-l-red-400",
  rescheduled: "border-l-navy-300",
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

const HOUR_PX = 48;
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
  return (d.getHours() * 60 + d.getMinutes()) * (HOUR_PX / 60);
}

function heightOf(durationMinutes: number): number {
  return Math.max((durationMinutes / 60) * HOUR_PX, 16);
}

function timeRange(iso: string, durationMinutes: number): string {
  const start = new Date(iso);
  const end = new Date(start.getTime() + durationMinutes * 60_000);
  const fmt = (x: Date) =>
    x.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
  return `${fmt(start)} – ${fmt(end)}`;
}

function crewSummary(b: Booking): string {
  const primary =
    b.cleaners?.find((c) => c.role === "primary")?.name ??
    b.cleaners?.[0]?.name ??
    b.assignedCleaner ??
    "Unassigned";
  const crewCount = (b.cleaners ?? []).filter((c) => c.role === "crew").length;
  return crewCount > 0 ? `${primary} +${crewCount}` : primary;
}

function isPast(day: Date): boolean {
  return day.getTime() < startOfWeek(today).getTime();
}

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
</script>

<template>
  <div class="flex h-full flex-col">
    <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-6 py-4">
      <div class="flex items-center gap-3">
        <span class="text-lg font-semibold text-gray-900">Dispatch</span>
        <span class="text-sm text-gray-500">{{ weekLabel }}</span>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          class="rounded-lg border border-gray-200 px-3 py-2 text-sm font-medium text-gray-600 hover:bg-gray-100"
          @click="shiftWeek(-1)"
        >
          ‹ Prev
        </button>
        <button
          type="button"
          class="rounded-lg border border-gray-200 px-3 py-2 text-sm font-medium text-gray-600 hover:bg-gray-100"
          @click="goToday"
        >
          This Week
        </button>
        <button
          type="button"
          class="rounded-lg border border-gray-200 px-3 py-2 text-sm font-medium text-gray-600 hover:bg-gray-100"
          @click="shiftWeek(1)"
        >
          Next ›
        </button>
        <input
          v-model="search"
          type="search"
          placeholder="Search week…"
          class="w-44 rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
        />
        <select
          v-model="statusFilter"
          class="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
        >
          <option value="">All statuses</option>
          <option v-for="(label, value) in statusLabels" :key="value" :value="value">
            {{ label }}
          </option>
        </select>
        <a
          href="/bookings"
          class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700"
        >
          Manage Bookings
        </a>
      </div>
    </div>

    <div v-if="loading" class="flex flex-1 items-center justify-center">
      <p class="text-sm text-gray-500">Loading week…</p>
    </div>

    <div
      v-else-if="visibleBookings.length === 0"
      class="flex flex-1 flex-col items-center justify-center"
    >
      <template v-if="loadError">
        <p class="text-sm font-medium text-red-700">Couldn't load bookings</p>
        <p class="mt-1 text-sm text-red-600">{{ loadError }}</p>
        <button
          type="button"
          class="mt-3 rounded-lg border border-gray-200 px-3 py-1.5 text-sm font-medium text-gray-600 hover:bg-gray-100"
          @click="load"
        >
          Retry
        </button>
      </template>
      <template v-else>
        <p class="text-sm font-medium text-gray-900">No bookings this week</p>
        <p class="mt-1 text-sm text-gray-500">
          {{
            filtersApplied
              ? "Try adjusting your filters."
              : "Create bookings from the Bookings page to see them on the calendar."
          }}
        </p>
      </template>
    </div>

    <div v-else class="flex flex-1 flex-col">
      <div class="grid grid-cols-7 border-b border-gray-200">
        <div
          v-for="(day, i) in days"
          :key="i"
          role="button"
          :tabindex="bookedCount(day) > 0 ? 0 : -1"
          :aria-label="`View bookings for ${day.toLocaleDateString(undefined, { weekday: 'long', month: 'long', day: 'numeric' })}`"
          :class="[
            'flex flex-col items-center gap-0.5 border-l border-gray-200 py-2 text-center first:border-l-0',
            sameDay(day, today) ? 'bg-navy-50' : '',
            isPast(day) && !sameDay(day, today) ? 'text-gray-400' : '',
            bookedCount(day) > 0
              ? 'cursor-pointer hover:bg-navy-100/70 focus:outline-none focus:ring-2 focus:ring-inset focus:ring-navy-500'
              : '',
          ]"
          @click="openDay(day)"
          @keydown.enter.prevent="openDay(day)"
          @keydown.space.prevent="openDay(day)"
        >
          <span class="text-xs font-medium uppercase tracking-wide text-gray-500">
            {{ day.toLocaleDateString(undefined, { weekday: "short" }) }}
          </span>
          <span class="flex items-center gap-1.5">
            <span
              :class="[
                'text-sm font-semibold',
                sameDay(day, today) ? 'text-navy-700' : 'text-gray-900',
              ]"
            >
              {{ day.toLocaleDateString(undefined, { day: "numeric" }) }}
            </span>
          </span>
          <span
            v-if="bookedCount(day) > 0"
            class="mt-0.5 inline-flex items-center gap-1 rounded-full bg-green-100 px-2 py-0.5 text-[10px] font-medium text-green-700"
          >
            <span class="h-1.5 w-1.5 rounded-full bg-green-500"></span>
            {{ bookedCount(day) }} booked
          </span>
        </div>
      </div>

      <div class="relative flex-1 overflow-auto">
        <div class="grid grid-cols-7">
          <div v-for="(day, i) in days" :key="i" class="relative border-l border-gray-200 first:border-l-0">
            <div
              class="pointer-events-none absolute left-0 right-0 border-t border-gray-100"
              v-for="hour in 24"
              :key="hour"
              :style="{ top: `${(hour * HOUR_PX) - 1}px` }"
            />
            <div v-if="sameDay(day, today)" class="pointer-events-none absolute left-0 right-0 top-0 bottom-0 bg-navy-50/40" />

            <div
              v-for="b in bookingsForDay(day)"
              :key="b.id"
              role="button"
              tabindex="0"
              aria-haspopup="dialog"
              :aria-label="`${b.customerName}, ${labelForService(b.serviceType)}, ${timeRange(b.scheduledFor, b.durationMinutes)}`"
              class="absolute left-0.5 right-0.5 z-10 cursor-pointer overflow-hidden rounded-md border border-navy-100 border-l-4 bg-white p-1.5 shadow-sm transition-shadow hover:shadow-md focus:outline-none focus:ring-2 focus:ring-navy-500"
              :class="[
                sameDay(day, today) ? 'border-navy-200' : '',
                statusAccent[b.status],
              ]"
              :style="{ top: `${topOf(b.scheduledFor)}px`, height: `${heightOf(b.durationMinutes)}px` }"
              :title="`${timeRange(b.scheduledFor, b.durationMinutes)} — ${b.customerName} — click for details`"
              @click="openBooking(b)"
              @keydown.enter.prevent="openBooking(b)"
              @keydown.space.prevent="openBooking(b)"
            >
              <p class="truncate text-xs font-semibold text-gray-900">
                {{ b.customerName }}
              </p>
              <p class="truncate text-[11px] text-gray-500">
                {{ timeRange(b.scheduledFor, b.durationMinutes) }} · {{ labelForService(b.serviceType) }}
              </p>
              <p class="truncate text-[11px] font-medium text-gray-600">
                {{ crewSummary(b) }}
              </p>
              <span
                :class="[
                  'mt-0.5 inline-flex rounded-full px-1.5 py-px text-[10px] font-medium',
                  statusStyles[b.status],
                ]"
              >
                {{ statusLabels[b.status] }}
              </span>
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