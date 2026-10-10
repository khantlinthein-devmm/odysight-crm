<script setup lang="ts">
import { computed, ref } from "vue";
import type { Cleaner } from "../../../lib/cleaners";

// One list for the whole booking team: each cleaner can be marked as the
// lead (primary, exactly one) or as crew (any number). Replaces the old
// ctrl-click multi-select, which most people never discovered.
const props = defineProps<{
  cleaners: Cleaner[];
  loading: boolean;
  error?: string;
}>();

const primary = defineModel<number | "">("primary", { required: true });
const crew = defineModel<number[]>("crew", { required: true });

const query = ref("");

function fullName(c: Cleaner): string {
  return `${c.firstName} ${c.lastName}`.trim();
}

function isSelected(id: number): boolean {
  return primary.value === id || crew.value.includes(id);
}

// Inactive cleaners only appear when already on this booking; on-leave ones
// show but can't be newly picked.
function unavailable(c: Cleaner): boolean {
  return (c.status === "inactive" || c.status === "on_leave") && !isSelected(c.id);
}

const visible = computed(() => {
  const q = query.value.trim().toLowerCase();
  return props.cleaners.filter((c) => {
    if (c.status === "inactive" && !isSelected(c.id)) return false;
    return !q || fullName(c).toLowerCase().includes(q);
  });
});

const byId = computed(() => new Map(props.cleaners.map((c) => [c.id, c])));

const leadName = computed(() => {
  const c = typeof primary.value === "number" ? byId.value.get(primary.value) : undefined;
  return c ? fullName(c) : "";
});

const crewNames = computed(() =>
  crew.value.map((id) => byId.value.get(id)).filter((c): c is Cleaner => !!c).map(fullName),
);

function setLead(id: number) {
  primary.value = id;
  crew.value = crew.value.filter((x) => x !== id);
}

function toggleCrew(id: number) {
  if (primary.value === id) return;
  crew.value = crew.value.includes(id) ? crew.value.filter((x) => x !== id) : [...crew.value, id];
}
</script>

<template>
  <div>
    <div class="mb-1 flex flex-wrap items-baseline justify-between gap-2">
      <span class="text-sm font-medium text-gray-700">Cleaning team</span>
      <span class="text-xs text-gray-500">1 lead (required) · crew optional</span>
    </div>

    <div
      :class="[
        'rounded-lg border bg-white',
        error ? 'border-red-300' : 'border-gray-200',
      ]"
    >
      <div class="flex flex-wrap items-center gap-1.5 border-b border-gray-100 px-3 py-2 text-xs">
        <span v-if="leadName" class="rounded-full bg-navy-600 px-2 py-0.5 font-medium text-white">Lead: {{ leadName }}</span>
        <span v-else class="rounded-full bg-amber-50 px-2 py-0.5 font-medium text-amber-700">No lead selected</span>
        <span v-for="name in crewNames" :key="name" class="rounded-full bg-navy-50 px-2 py-0.5 font-medium text-navy-700">
          Crew: {{ name }}
        </span>
      </div>

      <div v-if="cleaners.length > 6" class="border-b border-gray-100 px-3 py-2">
        <input
          v-model="query"
          type="search"
          placeholder="Search cleaners…"
          aria-label="Search cleaners"
          class="w-full rounded-md border border-gray-200 px-2.5 py-1.5 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
        />
      </div>

      <p v-if="loading" class="px-3 py-4 text-sm text-gray-500">Loading cleaners…</p>
      <p v-else-if="visible.length === 0" class="px-3 py-4 text-sm text-gray-500">No cleaners found.</p>
      <ul v-else class="max-h-64 divide-y divide-gray-100 overflow-y-auto">
        <li
          v-for="c in visible"
          :key="c.id"
          :class="['flex items-center justify-between gap-3 px-3 py-2', unavailable(c) ? 'opacity-50' : '']"
        >
          <span class="min-w-0 truncate text-sm text-gray-900">
            {{ fullName(c) }}
            <span v-if="c.status === 'on_leave'" class="text-xs text-amber-700">(on leave)</span>
            <span v-else-if="c.status === 'inactive'" class="text-xs text-gray-500">(inactive)</span>
            <span v-else-if="c.status === 'assigned'" class="text-xs text-gray-500">(busy)</span>
          </span>
          <span class="flex shrink-0 items-center gap-1">
            <button
              type="button"
              :disabled="unavailable(c)"
              :aria-pressed="primary === c.id"
              :class="[
                'rounded-md px-2.5 py-1 text-xs font-medium',
                primary === c.id
                  ? 'bg-navy-600 text-white'
                  : 'border border-gray-200 text-gray-600 hover:bg-gray-100 disabled:hover:bg-transparent',
              ]"
              @click="setLead(c.id)"
            >
              Lead
            </button>
            <button
              type="button"
              :disabled="unavailable(c) || primary === c.id"
              :aria-pressed="crew.includes(c.id)"
              :class="[
                'rounded-md px-2.5 py-1 text-xs font-medium',
                crew.includes(c.id)
                  ? 'bg-navy-100 text-navy-800 ring-1 ring-navy-300'
                  : 'border border-gray-200 text-gray-600 hover:bg-gray-100 disabled:hover:bg-transparent',
              ]"
              @click="toggleCrew(c.id)"
            >
              {{ crew.includes(c.id) ? "✓ Crew" : "+ Crew" }}
            </button>
          </span>
        </li>
      </ul>
    </div>
    <p v-if="error" class="mt-1 text-xs text-red-600">{{ error }}</p>
  </div>
</template>
