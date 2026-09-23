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
  available: "bg-green-50 text-green-700",
  assigned: "bg-navy-50 text-navy-700",
  on_leave: "bg-amber-50 text-amber-700",
  inactive: "bg-gray-100 text-gray-600",
};

const cleaners = ref<Cleaner[]>([]);
const loading = ref(true);
const search = ref("");
const statusFilter = ref<CleanerStatus | "">("");
const page = ref(1);
const pageSize = 10;

const showForm = ref(false);
const editingCleaner = ref<Cleaner | undefined>(undefined);
const saving = ref(false);
const pendingDelete = ref<Cleaner | null>(null);
const deleting = ref(false);

const filteredCleaners = computed(() => {
  const query = search.value.trim().toLowerCase();
  return cleaners.value.filter((cleaner) => {
    const matchesStatus =
      !statusFilter.value || cleaner.status === statusFilter.value;
    const matchesQuery =
      !query ||
      `${cleaner.firstName} ${cleaner.lastName}`
        .toLowerCase()
        .includes(query) ||
      cleaner.email.toLowerCase().includes(query);
    return matchesStatus && matchesQuery;
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
  <div class="rounded-xl border border-gray-200 bg-white">
    <div
      class="flex flex-col gap-3 border-b border-gray-200 px-6 py-4 sm:flex-row sm:items-center"
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

    <div v-else class="overflow-x-auto">
      <table class="w-full text-left text-sm">
        <thead>
          <tr
            class="border-b border-gray-200 text-xs uppercase tracking-wide text-gray-500"
          >
            <th class="px-6 py-3 font-medium">Name</th>
            <th class="px-6 py-3 font-medium">Email</th>
            <th class="px-6 py-3 font-medium">Phone</th>
            <th class="px-6 py-3 font-medium">Skills</th>
            <th class="px-6 py-3 font-medium">Status</th>
            <th class="px-6 py-3 font-medium">Created</th>
            <th class="px-6 py-3 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-navy-100">
          <tr
            v-for="cleaner in pagedCleaners"
            :key="cleaner.id"
            class="hover:bg-gray-100"
          >
            <td class="px-6 py-4">
              <button
                type="button"
                class="font-medium text-navy-600 hover:text-navy-700"
                @click="openDetail(cleaner)"
              >
                {{ cleaner.firstName }} {{ cleaner.lastName }}
              </button>
            </td>
            <td class="px-6 py-4 text-gray-600">{{ cleaner.email }}</td>
            <td class="px-6 py-4 text-gray-600">{{ cleaner.phone }}</td>
            <td class="px-6 py-4 text-gray-600">{{ cleaner.skills }}</td>
            <td class="px-6 py-4">
              <span
                :class="[
                  'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
                  statusStyles[cleaner.status],
                ]"
              >
                {{ statusLabels[cleaner.status] }}
              </span>
            </td>
            <td class="px-6 py-4 text-gray-600">
              {{ formatDate(cleaner.createdAt) }}
            </td>
            <td class="px-6 py-4 text-right whitespace-nowrap">
              <button
                v-if="canEdit"
                type="button"
                class="rounded-lg px-2 py-1 text-sm font-medium text-gray-600 hover:bg-gray-100 hover:text-gray-900"
                @click="openEdit(cleaner)"
              >
                Edit
              </button>
              <button
                v-if="canDelete"
                type="button"
                class="rounded-lg px-2 py-1 text-sm font-medium text-red-600 hover:bg-red-50"
                @click="pendingDelete = cleaner"
              >
                Delete
              </button>
            </td>
          </tr>
        </tbody>
      </table>
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
