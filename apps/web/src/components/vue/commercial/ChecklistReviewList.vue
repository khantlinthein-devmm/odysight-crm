<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import {
  CHECKLIST_STATE_BADGE as stateBadge,
  checklistState,
  listChecklists,
  type ChecklistState,
  type ChecklistSummary,
} from "../../../lib/checklists";
import { getCleaners, type Cleaner } from "../../../lib/cleaners";
import { serviceLabel } from "../../../lib/settings";
import { showToast } from "../../../lib/toast";
import AppIcon, { type IconName } from "../ui/AppIcon.vue";

// Office view of every booking checklist: who did it, how far they got, the
// scores, and which ones are waiting for review.

const emit = defineEmits<{ open: [bookingId: number] }>();

const PAGE = 50;
const rows = ref<ChecklistSummary[]>([]);
const total = ref(0);
const loading = ref(false);
const state = ref<ChecklistState | "">("submitted");
const cleanerId = ref<number | "">("");
const from = ref("");
const to = ref("");
const search = ref("");
const offset = ref(0);
const cleaners = ref<Cleaner[]>([]);
const counts = ref<Partial<Record<ChecklistState, number>>>({});

// The state tiles double as the filter tabs.
const tiles: { value: ChecklistState; label: string; icon: IconName; tone: string; ring: string }[] = [
  { value: "missing", label: "Missing checklist", icon: "warning", tone: "bg-red-100 text-red-600", ring: "ring-red-300" },
  { value: "submitted", label: "Waiting review", icon: "inbox", tone: "bg-amber-100 text-amber-600", ring: "ring-amber-300" },
  { value: "rework", label: "Sent back", icon: "undo", tone: "bg-red-100 text-red-600", ring: "ring-red-300" },
  { value: "approved", label: "Approved", icon: "badge", tone: "bg-emerald-100 text-emerald-600", ring: "ring-emerald-300" },
  { value: "in_progress", label: "In progress", icon: "progress", tone: "bg-blue-100 text-blue-600", ring: "ring-blue-300" },
  { value: "pending", label: "Not started", icon: "clock", tone: "bg-gray-100 text-gray-500", ring: "ring-gray-300" },
];

const filters = () => ({ cleanerId: cleanerId.value, from: from.value, to: to.value, search: search.value.trim() });

async function load() {
  loading.value = true;
  try {
    const page = await listChecklists({ ...filters(), state: state.value, limit: PAGE, offset: offset.value });
    rows.value = page.data;
    total.value = page.total;
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to load checklists", "error");
  } finally {
    loading.value = false;
  }
}

async function loadCounts() {
  const f = filters();
  const res = await Promise.all(
    tiles.map((tile) =>
      listChecklists({ ...f, state: tile.value, limit: 1 }).then((p) => [tile.value, p.total] as const).catch(() => null),
    ),
  );
  counts.value = Object.fromEntries(res.filter((x): x is readonly [ChecklistState, number] => !!x));
}

function reload() {
  offset.value = 0;
  void load();
  void loadCounts();
}

let searchTimer: ReturnType<typeof setTimeout> | undefined;
watch(state, () => {
  offset.value = 0;
  void load();
});
watch([cleanerId, from, to], reload);
watch(search, () => {
  clearTimeout(searchTimer);
  searchTimer = setTimeout(reload, 300);
});

function page(dir: -1 | 1) {
  offset.value = Math.max(0, offset.value + dir * PAGE);
  void load();
}

function dayNum(iso: string): string {
  return String(new Date(iso).getDate());
}
function monthShort(iso: string): string {
  return new Date(iso).toLocaleDateString("en-GB", { month: "short" });
}
function timeShort(iso: string): string {
  return new Date(iso).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit" });
}
function initials(name: string): string {
  return name.split(" ").filter(Boolean).slice(0, 2).map((w) => w[0]!.toUpperCase()).join("");
}
function scoreTone(v: number): string {
  return v >= 4 ? "text-emerald-600" : v >= 3 ? "text-amber-600" : "text-red-600";
}

onMounted(() => {
  void load();
  void loadCounts();
  getCleaners({ limit: 200 })
    .then((c) => (cleaners.value = c))
    .catch(() => {});
});
</script>

<template>
  <div class="space-y-4">
    <!-- State tiles -->
    <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-6">
      <button
        v-for="tile in tiles"
        :key="tile.value"
        type="button"
        :class="[
          'group rounded-3xl bg-white p-4 text-left shadow-sm ring-1 transition hover:-translate-y-0.5 hover:shadow-md',
          state === tile.value ? `ring-2 ${tile.ring}` : 'ring-gray-100',
        ]"
        @click="state = state === tile.value ? '' : tile.value"
      >
        <span :class="['flex h-10 w-10 items-center justify-center rounded-2xl', tile.tone]">
          <AppIcon :name="tile.icon" :size="20" />
        </span>
        <p class="mt-3 text-[26px] font-semibold leading-none tracking-tight text-gray-900">{{ counts[tile.value] ?? "–" }}</p>
        <p class="mt-1.5 text-sm font-medium text-gray-600">{{ tile.label }}</p>
      </button>
    </div>

    <!-- Filters -->
    <section class="rounded-3xl bg-white p-4 shadow-sm ring-1 ring-gray-100">
      <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
        <label class="relative block">
          <span class="pointer-events-none absolute inset-y-0 left-3 flex items-center text-gray-400"><AppIcon name="search" :size="17" /></span>
          <input v-model="search" type="search" placeholder="Booking no. or customer" class="w-full rounded-2xl border-0 bg-gray-50 py-2.5 pl-10 pr-3 text-sm ring-1 ring-gray-200 focus:ring-2 focus:ring-navy-500" />
        </label>
        <select v-model="cleanerId" class="w-full text-sm" aria-label="Cleaner">
          <option value="">All cleaners</option>
          <option v-for="c in cleaners" :key="c.id" :value="c.id">{{ c.firstName }} {{ c.lastName }}</option>
        </select>
        <label class="relative block">
          <span class="pointer-events-none absolute -top-2 left-3 bg-white px-1 text-[10px] font-semibold uppercase tracking-wide text-gray-400">From</span>
          <input v-model="from" type="date" class="w-full rounded-2xl border-0 bg-gray-50 px-3 py-2.5 text-sm ring-1 ring-gray-200 focus:ring-2 focus:ring-navy-500" aria-label="From date" />
        </label>
        <label class="relative block">
          <span class="pointer-events-none absolute -top-2 left-3 bg-white px-1 text-[10px] font-semibold uppercase tracking-wide text-gray-400">To</span>
          <input v-model="to" type="date" class="w-full rounded-2xl border-0 bg-gray-50 px-3 py-2.5 text-sm ring-1 ring-gray-200 focus:ring-2 focus:ring-navy-500" aria-label="To date" />
        </label>
      </div>
      <div class="mt-3 flex items-center justify-between text-xs text-gray-500">
        <span>
          <template v-if="state">{{ stateBadge[state].label }} · </template>{{ total }} checklist{{ total === 1 ? "" : "s" }}
        </span>
        <button v-if="state" type="button" class="font-semibold text-navy-600 hover:underline" @click="state = ''">Show all</button>
      </div>
    </section>

    <!-- Rows -->
    <div v-if="loading" class="space-y-2.5">
      <div v-for="i in 4" :key="i" class="h-20 animate-pulse rounded-3xl bg-white ring-1 ring-gray-100"></div>
    </div>
    <div v-else-if="rows.length === 0" class="flex flex-col items-center rounded-3xl bg-white px-6 py-14 text-center shadow-sm ring-1 ring-gray-100">
      <span class="flex h-14 w-14 items-center justify-center rounded-2xl bg-gray-50 text-gray-400"><AppIcon name="inbox" :size="28" /></span>
      <p class="mt-3 text-sm font-medium text-gray-700">Nothing here</p>
      <p class="text-xs text-gray-400">No checklists match these filters.</p>
    </div>
    <ul v-else class="space-y-2.5">
      <li v-for="r in rows" :key="r.id">
        <div
          role="button"
          tabindex="0"
          class="group flex cursor-pointer flex-wrap items-center gap-x-4 gap-y-3 rounded-3xl bg-white p-3.5 shadow-sm ring-1 ring-gray-100 transition hover:-translate-y-0.5 hover:shadow-md sm:flex-nowrap sm:p-4"
          @click="emit('open', r.bookingId)"
          @keydown.enter="emit('open', r.bookingId)"
        >
          <span class="flex w-12 shrink-0 flex-col items-center rounded-2xl bg-navy-50 py-1.5 text-navy-700">
            <span class="text-lg font-bold leading-none">{{ dayNum(r.scheduledFor) }}</span>
            <span class="text-[10px] font-semibold uppercase">{{ monthShort(r.scheduledFor) }}</span>
          </span>

          <div class="min-w-0 flex-1">
            <p class="truncate font-semibold text-gray-900">{{ r.customerName }}</p>
            <p class="truncate text-xs text-gray-500">{{ r.bookingNumber }} · {{ serviceLabel(r.serviceType) }} · {{ timeShort(r.scheduledFor) }}</p>
            <div v-if="r.cleaners" class="mt-1.5 flex items-center gap-1.5">
              <span class="flex -space-x-1.5">
                <span
                  v-for="n in r.cleaners.split(', ').slice(0, 3)"
                  :key="n"
                  class="flex h-6 w-6 items-center justify-center rounded-full bg-gradient-to-br from-navy-500 to-blue-400 text-[9px] font-bold text-white ring-2 ring-white"
                  :title="n"
                >{{ initials(n) }}</span>
              </span>
              <span class="truncate text-xs text-gray-500">{{ r.cleaners }}</span>
            </div>
          </div>

          <div class="w-full sm:w-36">
            <div class="flex items-center justify-between text-[11px] font-semibold text-gray-500">
              <span>{{ r.itemsDone }}/{{ r.itemsTotal }}</span>
              <span v-if="!r.id" class="text-red-600">No checklist</span>
              <span v-else>{{ r.itemsTotal ? Math.round((r.itemsDone / r.itemsTotal) * 100) : 0 }}%</span>
            </div>
            <div class="mt-1 h-1.5 overflow-hidden rounded-full bg-gray-100">
              <div class="h-full rounded-full bg-gradient-to-r from-navy-500 to-emerald-500" :style="{ width: `${r.itemsTotal ? (r.itemsDone / r.itemsTotal) * 100 : 0}%` }"></div>
            </div>
          </div>

          <div class="flex items-center gap-2">
            <span v-if="r.avgScore != null" :class="['inline-flex items-center gap-1 rounded-full bg-amber-50 px-2.5 py-1 text-xs font-bold', scoreTone(r.avgScore)]" title="Average score">
              <AppIcon name="star" :size="13" class="text-amber-400" /> {{ r.avgScore.toFixed(1) }}
            </span>
            <span v-if="r.failCount" class="inline-flex items-center gap-1 rounded-full bg-red-50 px-2.5 py-1 text-xs font-bold text-red-600" title="Failed items">
              <AppIcon name="x" :size="13" :stroke="2.5" /> {{ r.failCount }}
            </span>
            <span v-if="r.photoCount" class="inline-flex items-center gap-1 rounded-full bg-gray-50 px-2.5 py-1 text-xs font-semibold text-gray-600" title="Photos">
              <AppIcon name="photo" :size="13" /> {{ r.photoCount }}
            </span>
          </div>

          <span :class="['whitespace-nowrap rounded-full px-2.5 py-1 text-[11px] font-bold', stateBadge[checklistState(r)].tone]">{{ stateBadge[checklistState(r)].label }}</span>

          <div class="ml-auto flex items-center gap-1">
            <a
              :href="`/checklists/print?booking=${r.bookingId}&lang=en`"
              target="_blank"
              rel="noopener"
              class="flex h-9 w-9 items-center justify-center rounded-xl text-gray-400 transition hover:bg-navy-50 hover:text-navy-600"
              title="PDF report"
              aria-label="PDF report"
              @click.stop
            ><AppIcon name="document" :size="18" /></a>
            <span class="text-gray-300 transition group-hover:translate-x-0.5 group-hover:text-navy-500"><AppIcon name="chevronRight" :size="18" /></span>
          </div>
        </div>
      </li>
    </ul>

    <div v-if="total > PAGE" class="flex items-center justify-between text-sm text-gray-600">
      <span>{{ offset + 1 }}–{{ Math.min(offset + PAGE, total) }} of {{ total }}</span>
      <span class="flex gap-2">
        <button type="button" class="inline-flex items-center gap-1 rounded-xl bg-white px-3 py-1.5 ring-1 ring-gray-200 disabled:opacity-40" :disabled="offset === 0" @click="page(-1)"><AppIcon name="chevronLeft" :size="16" /> Previous</button>
        <button type="button" class="inline-flex items-center gap-1 rounded-xl bg-white px-3 py-1.5 ring-1 ring-gray-200 disabled:opacity-40" :disabled="offset + PAGE >= total" @click="page(1)">Next <AppIcon name="chevronRight" :size="16" /></button>
      </span>
    </div>
  </div>
</template>
