<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { getBookings, type Booking, type BookingStatus } from "../../../lib/bookings";
import { activeServices, getWorkspaceSettings, serviceLabel } from "../../../lib/settings";
import { showToast } from "../../../lib/toast";
import DonutChart from "../reports/DonutChart.vue";
import { formatDuration } from "../../../lib/duration";
import type { DonutSegment } from "../../../lib/charts";

const emit = defineEmits<{ open: [booking: Booking] }>();
// Bumped by the parent after a booking is saved or deleted, so the
// calendar refetches instead of showing the old time.
const props = defineProps<{ refreshKey?: number }>();

// One place to see the schedule by day, month or year, with a summary of the
// same period beside it (how many bookings, of which service and type).

type View = "day" | "month" | "year";
const view = ref<View>("month");
const cursor = ref(startOfDay(new Date()));

const bookings = ref<Booking[]>([]);
const loading = ref(false);
const truncated = ref(false);

const PAGE = 200;
const MAX_PAGES = 30;

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
  pending: "bg-amber-100 text-amber-800 ring-1 ring-amber-300",
  confirmed: "bg-blue-100 text-blue-800 ring-1 ring-blue-300",
  in_progress: "bg-indigo-100 text-indigo-800 ring-1 ring-indigo-300",
  completed: "bg-green-100 text-green-800 ring-1 ring-green-300",
  cancelled: "bg-gray-200 text-gray-700 ring-1 ring-gray-300",
  no_show: "bg-red-100 text-red-800 ring-1 ring-red-300",
  rescheduled: "bg-purple-100 text-purple-800 ring-1 ring-purple-300",
};

// Same status colors as the Reports page, so a status looks alike everywhere.
const STATUS_COLORS: Record<BookingStatus, string> = {
  pending: "#D4AF37",
  confirmed: "#2563eb",
  in_progress: "#0ea5e9",
  completed: "#10b981",
  cancelled: "#94a3b8",
  no_show: "#f43f5e",
  rescheduled: "#8b5cf6",
};

// Categorical slots in fixed order (validated for colour-blind separation).
// A service keeps its slot by catalog position, not by how busy it is.
const SERIES = ["#2a78d6", "#eb6834", "#1baf7a", "#eda100", "#e87ba4", "#008300", "#6250d6"];
const OTHER_COLOR = "#94a3b8";

// Sequential (one hue, light → dark) for the year heat-map.
const HEAT = ["#cde2fb", "#9ec5f4", "#6da7ec", "#3987e5", "#256abf", "#184f95"];

const weekdays = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];

function startOfDay(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate());
}

function dayKey(d: Date): string {
  return `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`;
}

const range = computed(() => {
  const c = cursor.value;
  if (view.value === "day") {
    return { from: c, to: new Date(c.getFullYear(), c.getMonth(), c.getDate() + 1) };
  }
  if (view.value === "month") {
    return { from: new Date(c.getFullYear(), c.getMonth(), 1), to: new Date(c.getFullYear(), c.getMonth() + 1, 1) };
  }
  return { from: new Date(c.getFullYear(), 0, 1), to: new Date(c.getFullYear() + 1, 0, 1) };
});

const title = computed(() => {
  const c = cursor.value;
  if (view.value === "day") {
    return c.toLocaleDateString(undefined, { weekday: "long", day: "numeric", month: "long", year: "numeric" });
  }
  if (view.value === "month") return c.toLocaleDateString(undefined, { month: "long", year: "numeric" });
  return String(c.getFullYear());
});

function move(step: number) {
  const c = cursor.value;
  if (view.value === "day") cursor.value = new Date(c.getFullYear(), c.getMonth(), c.getDate() + step);
  else if (view.value === "month") cursor.value = new Date(c.getFullYear(), c.getMonth() + step, 1);
  else cursor.value = new Date(c.getFullYear() + step, 0, 1);
}

function goToday() {
  cursor.value = startOfDay(new Date());
}

function openDay(d: Date) {
  cursor.value = startOfDay(d);
  view.value = "day";
}

function openMonth(month: number) {
  cursor.value = new Date(cursor.value.getFullYear(), month, 1);
  view.value = "month";
}

let loadSeq = 0;
async function load() {
  const seq = ++loadSeq;
  loading.value = true;
  truncated.value = false;
  const { from, to } = range.value;
  const rows: Booking[] = [];
  try {
    for (let page = 0; page < MAX_PAGES; page++) {
      const batch = await getBookings({
        from: from.toISOString(),
        to: to.toISOString(),
        limit: PAGE,
        offset: page * PAGE,
      });
      rows.push(...batch);
      if (batch.length < PAGE) break;
      if (page === MAX_PAGES - 1) truncated.value = true;
    }
    if (seq !== loadSeq) return;
    bookings.value = rows;
  } catch (err) {
    if (seq !== loadSeq) return;
    showToast(err instanceof Error ? err.message : "Could not load bookings", "error");
  } finally {
    if (seq === loadSeq) loading.value = false;
  }
}

watch(range, load);
watch(() => props.refreshKey, load);
onMounted(async () => {
  await getWorkspaceSettings().catch(() => null);
  await load();
});

const byDay = computed(() => {
  const map = new Map<string, Booking[]>();
  for (const b of bookings.value) {
    const k = dayKey(new Date(b.scheduledFor));
    const list = map.get(k);
    if (list) list.push(b);
    else map.set(k, [b]);
  }
  for (const list of map.values()) {
    list.sort((a, b) => a.scheduledFor.localeCompare(b.scheduledFor));
  }
  return map;
});

function bookingsOn(d: Date): Booking[] {
  return byDay.value.get(dayKey(d)) ?? [];
}

// Month grid: whole weeks, Monday first.
const monthCells = computed(() => {
  const first = new Date(cursor.value.getFullYear(), cursor.value.getMonth(), 1);
  const lead = (first.getDay() + 6) % 7;
  const start = new Date(first.getFullYear(), first.getMonth(), 1 - lead);
  const days = new Date(first.getFullYear(), first.getMonth() + 1, 0).getDate();
  const total = Math.ceil((lead + days) / 7) * 7;
  return Array.from({ length: total }, (_, i) => {
    const date = new Date(start.getFullYear(), start.getMonth(), start.getDate() + i);
    return { date, inMonth: date.getMonth() === first.getMonth() };
  });
});

const todayKey = dayKey(new Date());

// Year: one mini month per card, each day shaded by how many jobs it has.
const yearMonths = computed(() => {
  const y = cursor.value.getFullYear();
  return Array.from({ length: 12 }, (_, m) => {
    const first = new Date(y, m, 1);
    const lead = (first.getDay() + 6) % 7;
    const days = new Date(y, m + 1, 0).getDate();
    const cells: ({ date: Date; count: number } | null)[] = Array.from({ length: lead }, () => null);
    let count = 0;
    for (let d = 1; d <= days; d++) {
      const date = new Date(y, m, d);
      const n = bookingsOn(date).length;
      count += n;
      cells.push({ date, count: n });
    }
    return { month: m, name: first.toLocaleDateString(undefined, { month: "long" }), count, cells };
  });
});

const busiestDay = computed(() =>
  Math.max(1, ...Array.from(byDay.value.values(), (l) => l.length)),
);

function heatColor(n: number): string {
  if (n === 0) return "#f3f4f6";
  const i = Math.min(HEAT.length - 1, Math.floor(((n - 1) / busiestDay.value) * HEAT.length));
  return HEAT[i]!;
}

// --- Summary for the period on screen ---
const OPEN: BookingStatus[] = ["pending", "confirmed", "rescheduled"];

const kpis = computed(() => {
  const list = bookings.value;
  const customers = new Set(list.map((b) => (b.customerId ? `id:${b.customerId}` : `name:${b.customerName.trim().toLowerCase()}`)));
  const cleaners = new Set(list.flatMap((b) => b.cleaners.map((c) => c.id)));
  const needsCrew = list.filter((b) => b.cleaners.length === 0 && OPEN.includes(b.status)).length;
  return [
    { label: "Bookings", value: list.length },
    { label: "Customers", value: customers.size },
    { label: "Cleaners working", value: cleaners.size },
    { label: "Needs crew", value: needsCrew, warn: needsCrew > 0 },
  ];
});

const serviceSegments = computed<DonutSegment[]>(() => {
  const counts = new Map<string, number>();
  for (const b of bookings.value) counts.set(b.serviceType, (counts.get(b.serviceType) ?? 0) + 1);
  // Slot follows the catalog order; unknown or overflow services fold into "Other".
  const order = activeServices().map((s) => s.id);
  for (const id of counts.keys()) if (!order.includes(id)) order.push(id);
  const segments: DonutSegment[] = [];
  let other = 0;
  order.forEach((id, i) => {
    const value = counts.get(id) ?? 0;
    if (!value) return;
    if (i < SERIES.length) segments.push({ label: serviceLabel(id), value, color: SERIES[i]! });
    else other += value;
  });
  if (other) segments.push({ label: "Other", value: other, color: OTHER_COLOR });
  return segments;
});

const typeSegments = computed<DonutSegment[]>(() => {
  let oneTime = 0;
  let recurring = 0;
  let contract = 0;
  for (const b of bookings.value) {
    if (b.contractId) contract++;
    else if (b.isRecurring) recurring++;
    else oneTime++;
  }
  return [
    { label: "One-time", value: oneTime, color: SERIES[0]! },
    { label: "Recurring", value: recurring, color: SERIES[1]! },
    { label: "Contract", value: contract, color: SERIES[2]! },
  ].filter((s) => s.value > 0);
});

const statusSegments = computed<DonutSegment[]>(() => {
  const counts = new Map<BookingStatus, number>();
  for (const b of bookings.value) counts.set(b.status, (counts.get(b.status) ?? 0) + 1);
  return (Object.keys(statusLabels) as BookingStatus[])
    .filter((s) => counts.get(s))
    .map((s) => ({ label: statusLabels[s], value: counts.get(s)!, color: STATUS_COLORS[s] }));
});

function time(iso: string): string {
  return new Date(iso).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
}

function crew(b: Booking): string {
  return b.cleaners.map((c) => c.name).join(", ");
}

const views: { value: View; label: string }[] = [
  { value: "day", label: "Day" },
  { value: "month", label: "Month" },
  { value: "year", label: "Year" },
];
</script>

<template>
  <div class="grid grid-cols-1 gap-6 xl:grid-cols-3">
    <!-- Calendar -->
    <section class="rounded-xl border border-gray-200 bg-white shadow-sm xl:col-span-2">
      <div class="flex flex-wrap items-center gap-3 border-b border-gray-200 px-4 py-3 sm:px-6">
        <div class="inline-flex rounded-lg bg-gray-100 p-0.5" role="tablist" aria-label="Calendar view">
          <button
            v-for="v in views"
            :key="v.value"
            type="button"
            role="tab"
            :aria-selected="view === v.value"
            :class="[
              'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
              view === v.value ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-800',
            ]"
            @click="view = v.value"
          >
            {{ v.label }}
          </button>
        </div>

        <div class="flex items-center gap-1">
          <button
            type="button"
            class="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100 hover:text-gray-800"
            :aria-label="`Previous ${view}`"
            @click="move(-1)"
          >
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" /></svg>
          </button>
          <button
            type="button"
            class="rounded-lg border border-gray-200 px-3 py-1 text-sm font-medium text-gray-700 hover:bg-gray-100"
            @click="goToday"
          >
            Today
          </button>
          <button
            type="button"
            class="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100 hover:text-gray-800"
            :aria-label="`Next ${view}`"
            @click="move(1)"
          >
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" /></svg>
          </button>
        </div>

        <h2 class="text-base font-semibold text-gray-900">{{ title }}</h2>
        <span v-if="loading" class="ml-auto text-xs text-gray-400">Loading…</span>
      </div>

      <p v-if="truncated" class="bg-amber-50 px-6 py-2 text-xs text-amber-800">
        Showing the first {{ PAGE * MAX_PAGES }} bookings of this period.
      </p>

      <!-- Month -->
      <div v-if="view === 'month'" class="p-2 sm:p-4">
        <div class="grid grid-cols-7 text-center text-xs font-medium text-gray-500">
          <div v-for="d in weekdays" :key="d" class="py-2">{{ d }}</div>
        </div>
        <div class="grid grid-cols-7 gap-px overflow-hidden rounded-lg bg-gray-200 ring-1 ring-gray-200">
          <button
            v-for="cell in monthCells"
            :key="cell.date.toISOString()"
            type="button"
            :class="[
              'flex min-h-20 flex-col gap-1 p-1.5 text-left align-top transition-colors hover:bg-navy-50 sm:min-h-28',
              cell.inMonth ? 'bg-white' : 'bg-gray-50 text-gray-400',
            ]"
            :aria-label="`${cell.date.toDateString()}: ${bookingsOn(cell.date).length} bookings`"
            @click="openDay(cell.date)"
          >
            <span class="flex items-center justify-between">
              <span
                :class="[
                  'flex h-6 w-6 items-center justify-center rounded-full text-xs font-medium',
                  dayKey(cell.date) === todayKey ? 'bg-navy-600 text-white' : '',
                ]"
              >{{ cell.date.getDate() }}</span>
              <span
                v-if="bookingsOn(cell.date).length"
                class="rounded-full bg-gray-100 px-1.5 text-[11px] font-semibold text-gray-700 sm:hidden"
              >{{ bookingsOn(cell.date).length }}</span>
            </span>
            <span
              v-for="b in bookingsOn(cell.date).slice(0, 3)"
              :key="b.id"
              :class="['hidden truncate rounded px-1.5 py-0.5 text-[11px] font-medium sm:block', statusStyles[b.status]]"
              :title="`${time(b.scheduledFor)} ${b.customerName} · ${serviceLabel(b.serviceType)}`"
            >
              {{ time(b.scheduledFor) }} {{ b.customerName }}
            </span>
            <span
              v-if="bookingsOn(cell.date).length > 3"
              class="hidden text-[11px] font-medium text-navy-600 sm:block"
            >+{{ bookingsOn(cell.date).length - 3 }} more</span>
          </button>
        </div>
      </div>

      <!-- Day -->
      <div v-else-if="view === 'day'" class="p-4 sm:p-6">
        <p v-if="!loading && bookingsOn(cursor).length === 0" class="py-12 text-center text-sm text-gray-500">
          No bookings on this day.
        </p>
        <ul class="space-y-2">
          <li
            v-for="b in bookingsOn(cursor)"
            :key="b.id"
            role="button"
            tabindex="0"
            class="flex cursor-pointer flex-col gap-2 rounded-xl p-3 ring-1 ring-gray-200 transition hover:bg-navy-50/50 hover:ring-navy-200 sm:flex-row sm:items-start sm:gap-4"
            @click="emit('open', b)"
            @keydown.enter="emit('open', b)"
          >
            <div class="w-24 shrink-0">
              <p class="text-sm font-semibold text-gray-900">{{ time(b.scheduledFor) }}</p>
              <p class="text-xs text-gray-500">{{ formatDuration(b.durationMinutes) }}</p>
            </div>
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium text-gray-900">
                {{ b.customerName }}
                <span class="font-normal text-gray-400">· {{ b.bookingNumber }}</span>
              </p>
              <p class="text-xs text-gray-600">
                {{ serviceLabel(b.serviceType) }}<span v-if="b.isRecurring"> · Recurring</span>
              </p>
              <p class="truncate text-xs text-gray-500">{{ b.address }}</p>
              <p class="mt-1 text-xs" :class="b.cleaners.length ? 'text-gray-600' : 'font-medium text-amber-700'">
                {{ b.cleaners.length ? crew(b) : "No cleaner assigned yet" }}
              </p>
            </div>
            <span :class="['self-start rounded-full px-2 py-0.5 text-xs font-medium', statusStyles[b.status]]">
              {{ statusLabels[b.status] }}
            </span>
          </li>
        </ul>
        <p v-if="bookingsOn(cursor).length" class="mt-4 text-xs text-gray-500">
          Click a booking to see its details and the customer.
          <a href="/dispatch" class="ml-1 font-medium text-navy-600 hover:underline">Assign cleaners in Dispatch</a>
        </p>
      </div>

      <!-- Year -->
      <div v-else class="grid grid-cols-1 gap-4 p-4 sm:grid-cols-2 sm:p-6 lg:grid-cols-3">
        <button
          v-for="m in yearMonths"
          :key="m.month"
          type="button"
          class="flex flex-col justify-start rounded-xl p-3 text-left ring-1 ring-gray-200 transition hover:ring-navy-300"
          @click="openMonth(m.month)"
        >
          <span class="mb-2 flex w-full items-baseline justify-between">
            <span class="text-sm font-semibold text-gray-900">{{ m.name }}</span>
            <span class="text-xs text-gray-500"><strong class="text-gray-900">{{ m.count }}</strong> bookings</span>
          </span>
          <span class="grid w-full grid-cols-7 gap-0.5">
            <span
              v-for="(c, i) in m.cells"
              :key="i"
              class="aspect-square rounded-sm"
              :style="c ? { backgroundColor: heatColor(c.count) } : undefined"
              :title="c ? `${c.date.toLocaleDateString()}: ${c.count} booking${c.count === 1 ? '' : 's'}` : undefined"
            ></span>
          </span>
        </button>
        <div class="flex items-center gap-2 text-xs text-gray-500 sm:col-span-2 lg:col-span-3">
          <span>Fewer</span>
          <span class="h-3 w-3 rounded-sm" style="background-color: #f3f4f6"></span>
          <span v-for="c in HEAT" :key="c" class="h-3 w-3 rounded-sm" :style="{ backgroundColor: c }"></span>
          <span>More bookings per day</span>
        </div>
      </div>
    </section>

    <!-- Summary -->
    <aside class="space-y-4">
      <div class="grid grid-cols-2 gap-3">
        <div
          v-for="k in kpis"
          :key="k.label"
          class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm"
        >
          <p class="text-xs font-medium text-gray-500">{{ k.label }}</p>
          <p :class="['mt-1 text-2xl font-semibold tracking-tight', k.warn ? 'text-amber-700' : 'text-gray-900']">
            {{ k.value }}
          </p>
        </div>
      </div>

      <section class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
        <h3 class="text-sm font-semibold text-gray-900">By service type</h3>
        <p v-if="!serviceSegments.length" class="py-6 text-center text-sm text-gray-400">No bookings in this period</p>
        <DonutChart
          v-else
          class="mt-4"
          :segments="serviceSegments"
          :size="160"
          :thickness="24"
          center-bottom="bookings"
          compact
        />
      </section>

      <section class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
        <h3 class="text-sm font-semibold text-gray-900">By booking type</h3>
        <p v-if="!typeSegments.length" class="py-6 text-center text-sm text-gray-400">No bookings in this period</p>
        <DonutChart
          v-else
          class="mt-4"
          :segments="typeSegments"
          :size="160"
          :thickness="24"
          center-bottom="bookings"
          compact
        />
      </section>

      <section class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
        <h3 class="text-sm font-semibold text-gray-900">By status</h3>
        <p v-if="!statusSegments.length" class="py-6 text-center text-sm text-gray-400">No bookings in this period</p>
        <DonutChart
          v-else
          class="mt-4"
          :segments="statusSegments"
          :size="160"
          :thickness="24"
          center-bottom="bookings"
          compact
        />
      </section>
    </aside>
  </div>
</template>
