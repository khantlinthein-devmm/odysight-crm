<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  createCleaner,
  deleteCleaner,
  getCleaners,
  updateCleaner,
  type Cleaner,
  type CleanerStatus,
  type CreateCleanerInput,
} from "../../../lib/cleaners";
import { showToast } from "../../../lib/toast";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";
import CleanerForm from "./CleanerForm.vue";
import ConfirmDialog from "../ui/ConfirmDialog.vue";

const role = getSessionUser()?.role;
const canCreate = computed(() => hasPermission(role, "cleaners.create"));
const canEdit = computed(() => hasPermission(role, "cleaners.update"));
const canDelete = computed(() => hasPermission(role, "cleaners.delete"));

const statusLabels: Record<CleanerStatus, string> = {
  available: "Available",
  assigned: "Assigned",
  on_leave: "On Leave",
  inactive: "Inactive",
};

const statusStyles: Record<CleanerStatus, string> = {
  available: "bg-green-100 text-green-800 ring-1 ring-green-300",
  assigned: "bg-blue-100 text-blue-800 ring-1 ring-blue-300",
  on_leave: "bg-amber-100 text-amber-800 ring-1 ring-amber-300",
  inactive: "bg-gray-200 text-gray-700 ring-1 ring-gray-300",
};

const statusAccent: Record<CleanerStatus, string> = {
  available: "bg-green-500",
  assigned: "bg-blue-500",
  on_leave: "bg-amber-400",
  inactive: "bg-gray-300",
};

const cleaners = ref<Cleaner[]>([]);
const loading = ref(true);
const search = ref("");
const statusFilter = ref<CleanerStatus | "">("");
const skillFilter = ref<string>("");
const page = ref(1);
const pageSize = 9;

const showForm = ref(false);
const editingCleaner = ref<Cleaner | undefined>(undefined);
const saving = ref(false);
const pendingDelete = ref<Cleaner | null>(null);
const deleting = ref(false);

function parseSkills(raw: string): string[] {
  return (raw ?? "")
    .split(",")
    .map((s) => s.trim())
    .filter((s) => s.length > 0);
}

function skillKey(raw: string): string {
  return raw.trim().toLowerCase();
}

// Distinct skill groups derived from cleaner data, e.g. "aircon" -> "Aircon (3)".
// Clicking a group filters the list to specialists with that skill.
const skillGroups = computed(() => {
  const counts = new Map<string, { label: string; count: number }>();
  for (const cleaner of cleaners.value) {
    const seen = new Set<string>();
    for (const skill of parseSkills(cleaner.skills)) {
      const key = skillKey(skill);
      if (seen.has(key)) continue;
      seen.add(key);
      const entry = counts.get(key);
      if (entry) entry.count++;
      else counts.set(key, { label: skill, count: 1 });
    }
  }
  return [...counts.entries()]
    .map(([key, v]) => ({ key, ...v }))
    .sort((a, b) => b.count - a.count || a.label.localeCompare(b.label));
});

function selectSkill(key: string) {
  skillFilter.value = skillFilter.value === key ? "" : key;
  page.value = 1;
}

const filteredCleaners = computed(() => {
  const query = search.value.trim().toLowerCase();
  return cleaners.value.filter((cleaner) => {
    const matchesStatus =
      !statusFilter.value || cleaner.status === statusFilter.value;
    const matchesSkill =
      !skillFilter.value ||
      parseSkills(cleaner.skills).some((s) => skillKey(s) === skillFilter.value);
    const matchesQuery =
      !query ||
      `${cleaner.firstName} ${cleaner.lastName}`
        .toLowerCase()
        .includes(query) ||
      cleaner.email.toLowerCase().includes(query);
    return matchesStatus && matchesSkill && matchesQuery;
  });
});

const totalPages = computed(() =>
  Math.max(1, Math.ceil(filteredCleaners.value.length / pageSize)),
);

const pagedCleaners = computed(() => {
  const start = (page.value - 1) * pageSize;
  return filteredCleaners.value.slice(start, start + pageSize);
});

async function fetchCleaners() {
  loading.value = true;
  try {
    cleaners.value = await getCleaners();
  } catch {
    showToast("Failed to load cleaners", "error");
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  editingCleaner.value = undefined;
  showForm.value = true;
}

function openEdit(cleaner: Cleaner) {
  editingCleaner.value = cleaner;
  showForm.value = true;
}

function openDetail(cleaner: Cleaner) {
  window.location.href = `/cleaners/${cleaner.id}`;
}

function closeForm() {
  showForm.value = false;
  editingCleaner.value = undefined;
}

async function handleSave(input: CreateCleanerInput) {
  saving.value = true;
  try {
    if (editingCleaner.value) {
      const updated = await updateCleaner(editingCleaner.value.id, input);
      cleaners.value = cleaners.value.map((c) =>
        c.id === updated.id ? updated : c,
      );
      showToast("Cleaner updated", "success");
    } else {
      const created = await createCleaner(input);
      cleaners.value = [created, ...cleaners.value];
      showToast("Cleaner created", "success");
    }
    closeForm();
  } catch {
    showToast("Failed to save cleaner", "error");
  } finally {
    saving.value = false;
  }
}

async function handleDelete() {
  if (!pendingDelete.value) return;
  deleting.value = true;
  try {
    await deleteCleaner(pendingDelete.value.id);
    cleaners.value = cleaners.value.filter(
      (c) => c.id !== pendingDelete.value!.id,
    );
    showToast("Cleaner deleted", "success");
  } catch {
    showToast("Failed to delete cleaner", "error");
  } finally {
    deleting.value = false;
    pendingDelete.value = null;
  }
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

onMounted(fetchCleaners);
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-gray-100/70">
    <div
      class="flex flex-col gap-3 rounded-t-xl border-b border-gray-200 bg-white px-6 py-4 sm:flex-row sm:items-center"
    >
      <h2 class="text-base font-semibold text-gray-900">Cleaners</h2>
      <span
        class="rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-600"
      >
        {{ filteredCleaners.length }}
      </span>

      <div class="sm:ml-auto flex flex-col gap-2 sm:flex-row">
        <input
          v-model="search"
          @input="page = 1"
          type="search"
          placeholder="Search name or email..."
          class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500 sm:w-56"
        />
        <select
          v-model="statusFilter"
          @change="page = 1"
          class="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
        >
          <option value="">All statuses</option>
          <option
            v-for="(label, value) in statusLabels"
            :key="value"
            :value="value"
          >
            {{ label }}
          </option>
        </select>
        <button
          v-if="canCreate"
          type="button"
          class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700"
          @click="openCreate"
        >
          New Cleaner
        </button>
      </div>
    </div>

    <div
      v-if="!loading && cleaners.length > 0"
      class="flex flex-wrap items-center gap-2 border-b border-gray-200 bg-white px-6 py-3"
    >
      <span class="text-xs font-semibold uppercase tracking-wide text-gray-500">Groups:</span>
      <button
        type="button"
        class="rounded-full px-3 py-1 text-xs font-semibold ring-1 transition"
        :class="!skillFilter ? 'bg-navy-600 text-white ring-navy-600' : 'bg-white text-gray-700 ring-gray-200 hover:bg-gray-100'"
        @click="selectSkill('');"
      >
        All ({{ cleaners.length }})
      </button>
      <button
        v-for="group in skillGroups"
        :key="group.key"
        type="button"
        class="rounded-full px-3 py-1 text-xs font-semibold ring-1 transition"
        :class="skillFilter === group.key ? 'bg-navy-600 text-white ring-navy-600' : 'bg-white text-gray-700 ring-gray-200 hover:bg-gray-100'"
        @click="selectSkill(group.key)"
        :title="`Show ${group.label} specialists`"
      >
        {{ group.label }} ({{ group.count }})
      </button>
      <span v-if="skillFilter" class="ml-auto text-xs text-gray-500">
        Filtering by “{{ skillGroups.find((g) => g.key === skillFilter)?.label }}”
        <button type="button" class="ml-1 font-semibold text-navy-600 hover:underline" @click="selectSkill(skillFilter)">Clear</button>
      </span>
    </div>

    <div v-if="loading" class="px-6 py-16 text-center">
      <p class="text-sm text-gray-500">Loading cleaners...</p>
    </div>

    <div
      v-else-if="filteredCleaners.length === 0"
      class="px-6 py-16 text-center"
    >
      <p class="text-sm font-medium text-gray-900">No cleaners found</p>
      <p class="mt-1 text-sm text-gray-500">
        Try adjusting your filters or create a new cleaner.
      </p>
    </div>

    <div v-else class="grid grid-cols-1 gap-4 p-4 sm:p-6 md:grid-cols-2 xl:grid-cols-3">
      <article
        v-for="cleaner in pagedCleaners"
        :key="cleaner.id"
        class="flex flex-col overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm transition hover:shadow-md"
      >
        <div :class="['h-1.5 w-full', statusAccent[cleaner.status]]"></div>
        <div class="flex flex-1 flex-col p-5">
          <div class="flex items-start justify-between gap-2">
            <span class="rounded bg-gray-900 px-2 py-0.5 font-mono text-[11px] font-semibold tracking-wide text-white">
              #{{ cleaner.id }}
            </span>
            <span
              :class="[
                'inline-flex shrink-0 items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-bold',
                statusStyles[cleaner.status],
              ]"
            >
              <span :class="['h-1.5 w-1.5 rounded-full', statusAccent[cleaner.status]]"></span>
              {{ statusLabels[cleaner.status] }}
            </span>
          </div>

          <div class="mt-3 flex items-center gap-3">
            <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-navy-600 text-base font-bold text-white">
              {{ (cleaner.firstName || "?").charAt(0).toUpperCase() }}{{ (cleaner.lastName || "").charAt(0).toUpperCase() }}
            </span>
            <div class="min-w-0">
              <button
                type="button"
                class="block max-w-full truncate text-left text-base font-bold text-gray-900 hover:text-navy-700"
                @click="openDetail(cleaner)"
                :title="`View ${cleaner.firstName} ${cleaner.lastName}`"
              >
                {{ cleaner.firstName }} {{ cleaner.lastName }}
              </button>
              <p class="truncate text-xs text-gray-500">Joined {{ formatDate(cleaner.createdAt) }}</p>
            </div>
          </div>

          <dl class="mt-4 space-y-2 rounded-lg bg-gray-50 p-3 text-sm ring-1 ring-gray-100">
            <div class="flex items-center gap-2">
              <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-blue-100 text-blue-700"><svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" /></svg></span>
              <dd class="min-w-0 truncate font-medium text-gray-800">{{ cleaner.phone || "—" }}</dd>
            </div>
            <div class="flex items-center gap-2">
              <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-green-100 text-green-700"><svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" /></svg></span>
              <dd class="min-w-0 truncate text-gray-700">{{ cleaner.email || "—" }}</dd>
            </div>
            <div v-if="cleaner.lineId" class="flex items-center gap-2">
              <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-emerald-100 text-emerald-700"><svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" /></svg></span>
              <dd class="min-w-0 truncate text-gray-700">{{ cleaner.lineId }}</dd>
            </div>
          </dl>

          <div class="mt-3">
            <p class="mb-1.5 text-[11px] font-semibold uppercase tracking-wide text-gray-400">Specialties</p>
            <span class="flex flex-wrap gap-1.5">
              <button
                v-for="skill in parseSkills(cleaner.skills)"
                :key="skill"
                type="button"
                class="rounded-md px-2 py-1 text-xs font-semibold ring-1 transition"
                :class="skillKey(skill) === skillFilter ? 'bg-navy-600 text-white ring-navy-600' : 'bg-slate-100 text-slate-700 ring-slate-200 hover:bg-slate-200'"
                @click="selectSkill(skillKey(skill))"
                :title="`Filter by ${skill}`"
              >
                {{ skill }}
              </button>
              <span v-if="parseSkills(cleaner.skills).length === 0" class="text-xs text-gray-400">No skills listed</span>
            </span>
          </div>

          <div class="mt-4 flex flex-wrap items-center gap-2 border-t border-gray-100 pt-3">
            <button
              type="button"
              class="rounded-lg border border-navy-200 bg-navy-50 px-3 py-1.5 text-sm font-semibold text-navy-700 hover:bg-navy-100"
              @click="openDetail(cleaner)"
            >
              View
            </button>
            <button
              v-if="canEdit"
              type="button"
              class="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-sm font-semibold text-gray-700 hover:bg-gray-100"
              @click="openEdit(cleaner)"
            >
              Edit
            </button>
            <button
              v-if="canDelete"
              type="button"
              class="ml-auto rounded-lg px-2.5 py-1.5 text-sm font-medium text-red-600 hover:bg-red-50"
              @click="pendingDelete = cleaner"
            >
              Delete
            </button>
          </div>
        </div>
      </article>
    </div>

    <div
      v-if="!loading && totalPages > 1"
      class="flex items-center justify-between border-t border-gray-200 px-6 py-3"
    >
      <p class="text-sm text-gray-500">Page {{ page }} of {{ totalPages }}</p>
      <div class="flex gap-2">
        <button
          type="button"
          :disabled="page <= 1"
          class="rounded-lg border border-gray-200 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-100 disabled:opacity-40 disabled:hover:bg-transparent"
          @click="page--"
        >
          Previous
        </button>
        <button
          type="button"
          :disabled="page >= totalPages"
          class="rounded-lg border border-gray-200 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-100 disabled:opacity-40 disabled:hover:bg-transparent"
          @click="page++"
        >
          Next
        </button>
      </div>
    </div>
  </div>

  <CleanerForm
    v-if="showForm"
    :key="editingCleaner?.id ?? 'new'"
    :cleaner="editingCleaner"
    @save="handleSave"
    @cancel="closeForm"
  />

  <ConfirmDialog
    v-if="pendingDelete"
    title="Delete cleaner"
    :message="`Are you sure you want to delete ${pendingDelete.firstName} ${pendingDelete.lastName}? This action cannot be undone.`"
    confirm-label="Delete cleaner"
    :busy="deleting"
    @confirm="handleDelete"
    @cancel="pendingDelete = null"
  />
</template>
