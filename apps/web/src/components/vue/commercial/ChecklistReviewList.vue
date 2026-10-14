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

const stateTabs: { value: ChecklistState | ""; label: string }[] = [
  { value: "submitted", label: "Waiting review" },
  { value: "rework", label: "Sent back" },
  { value: "approved", label: "Approved" },
  { value: "in_progress", label: "In progress" },
  { value: "pending", label: "Not started" },
  { value: "", label: "All" },
];

async function load() {
  loading.value = true;
  try {
    const page = await listChecklists({
      state: state.value, cleanerId: cleanerId.value, from: from.value, to: to.value,
      search: search.value.trim(), limit: PAGE, offset: offset.value,
    });
    rows.value = page.data;
    total.value = page.total;
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to load checklists", "error");
  } finally {
    loading.value = false;
  }
}

let searchTimer: ReturnType<typeof setTimeout> | undefined;
watch([state, cleanerId, from, to], () => {
  offset.value = 0;
  void load();
});
watch(search, () => {
  clearTimeout(searchTimer);
  searchTimer = setTimeout(() => {
    offset.value = 0;
    void load();
  }, 300);
});

function page(dir: -1 | 1) {
  offset.value = Math.max(0, offset.value + dir * PAGE);
  void load();
}

function day(iso: string): string {
  return new Date(iso).toLocaleDateString("en-GB", { day: "2-digit", month: "short", year: "numeric" });
}

onMounted(() => {
  void load();
  getCleaners({ limit: 200 })
    .then((c) => (cleaners.value = c))
    .catch(() => {});
});
</script>

<template>
  <div class="px-6 py-4">
    <div class="flex flex-wrap gap-1.5">
      <button
        v-for="tab in stateTabs"
        :key="tab.value"
        type="button"
        :class="['rounded-full px-3 py-1 text-sm font-medium', state === tab.value ? 'bg-navy-600 text-white' : 'bg-gray-100 text-gray-700 hover:bg-gray-200']"
        @click="state = tab.value"
      >{{ tab.label }}</button>
    </div>
    <div class="mt-3 grid gap-2 sm:grid-cols-4">
      <select v-model="cleanerId" class="rounded-lg border border-gray-200 px-3 py-2 text-sm" aria-label="Cleaner">
        <option value="">All cleaners</option>
        <option v-for="c in cleaners" :key="c.id" :value="c.id">{{ c.firstName }} {{ c.lastName }}</option>
      </select>
      <input v-model="from" type="date" class="rounded-lg border border-gray-200 px-3 py-2 text-sm" aria-label="From date" />
      <input v-model="to" type="date" class="rounded-lg border border-gray-200 px-3 py-2 text-sm" aria-label="To date" />
      <input v-model="search" type="search" placeholder="Booking no. or customer" class="rounded-lg border border-gray-200 px-3 py-2 text-sm" />
    </div>

    <div class="mt-4 overflow-x-auto rounded-lg border border-gray-200">
      <table class="w-full text-sm">
        <thead class="bg-gray-50 text-left text-xs font-medium text-gray-500">
          <tr>
            <th class="px-3 py-2">Date</th>
            <th class="px-3 py-2">Booking</th>
            <th class="px-3 py-2">Cleaners</th>
            <th class="px-3 py-2">Done</th>
            <th class="px-3 py-2">Avg score</th>
            <th class="px-3 py-2">Fails</th>
            <th class="px-3 py-2">Photos</th>
            <th class="px-3 py-2">State</th>
            <th class="px-3 py-2"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-if="loading"><td colspan="9" class="px-3 py-8 text-center text-gray-500">Loading…</td></tr>
          <tr v-else-if="rows.length === 0"><td colspan="9" class="px-3 py-8 text-center text-gray-500">No checklists match.</td></tr>
          <tr v-for="r in rows" v-else :key="r.id" class="cursor-pointer hover:bg-gray-50" @click="emit('open', r.bookingId)">
            <td class="whitespace-nowrap px-3 py-2 text-gray-700">{{ day(r.scheduledFor) }}</td>
            <td class="px-3 py-2">
              <p class="font-medium text-gray-900">{{ r.bookingNumber }} — {{ r.customerName }}</p>
              <p class="text-xs text-gray-500">{{ serviceLabel(r.serviceType) }}</p>
            </td>
            <td class="px-3 py-2 text-gray-700">{{ r.cleaners || "—" }}</td>
            <td class="whitespace-nowrap px-3 py-2 text-gray-700">{{ r.itemsDone }}/{{ r.itemsTotal }}</td>
            <td class="px-3 py-2">
              <span v-if="r.avgScore != null" :class="['font-semibold', r.avgScore >= 4 ? 'text-green-700' : r.avgScore >= 3 ? 'text-amber-700' : 'text-red-700']">{{ r.avgScore.toFixed(1) }}</span>
              <span v-else class="text-gray-400">—</span>
            </td>
            <td class="px-3 py-2"><span :class="r.failCount ? 'font-semibold text-red-700' : 'text-gray-400'">{{ r.failCount || "—" }}</span></td>
            <td class="px-3 py-2 text-gray-700">{{ r.photoCount || "—" }}</td>
            <td class="px-3 py-2">
              <span :class="['whitespace-nowrap rounded-full px-2 py-0.5 text-[11px] font-semibold', stateBadge[checklistState(r)].tone]">{{ stateBadge[checklistState(r)].label }}</span>
            </td>
            <td class="whitespace-nowrap px-3 py-2 text-right">
              <a :href="`/checklists/print?booking=${r.bookingId}&lang=en`" target="_blank" rel="noopener" class="mr-3 text-gray-500 hover:text-navy-600" @click.stop>PDF</a>
              <span class="font-medium text-navy-600">Open →</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-if="total > PAGE" class="mt-3 flex items-center justify-between text-sm text-gray-600">
      <span>{{ offset + 1 }}–{{ Math.min(offset + PAGE, total) }} of {{ total }}</span>
      <span class="flex gap-2">
        <button type="button" class="rounded border border-gray-200 px-3 py-1 disabled:opacity-40" :disabled="offset === 0" @click="page(-1)">Previous</button>
        <button type="button" class="rounded border border-gray-200 px-3 py-1 disabled:opacity-40" :disabled="offset + PAGE >= total" @click="page(1)">Next</button>
      </span>
    </div>
  </div>
</template>
