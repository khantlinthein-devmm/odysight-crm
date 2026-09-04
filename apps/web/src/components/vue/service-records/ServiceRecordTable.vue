<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  createServiceRecord,
  deleteServiceRecord,
  getServiceRecords,
  updateServiceRecord,
  type ServiceRecord,
  type ServiceRecordStatus,
  type CreateServiceRecordInput,
} from "../../../lib/service-records";
import { showToast } from "../../../lib/toast";
import ServiceRecordForm from "./ServiceRecordForm.vue";
import ConfirmDialog from "../ui/ConfirmDialog.vue";

const statusLabels: Record<ServiceRecordStatus, string> = {
  pending: "Pending",
  completed: "Completed",
  rescheduled: "Rescheduled",
  cancelled: "Cancelled",
};

const statusStyles: Record<ServiceRecordStatus, string> = {
  pending: "bg-amber-50 text-amber-700",
  completed: "bg-emerald-50 text-emerald-700",
  rescheduled: "bg-violet-50 text-violet-700",
  cancelled: "bg-slate-100 text-slate-600",
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

const records = ref<ServiceRecord[]>([]);
const loading = ref(true);
const search = ref("");
const statusFilter = ref<ServiceRecordStatus | "">("");
const page = ref(1);
const pageSize = 5;

const showForm = ref(false);
const editingRecord = ref<ServiceRecord | undefined>(undefined);
const saving = ref(false);
const pendingDelete = ref<ServiceRecord | null>(null);
const deleting = ref(false);

const filteredRecords = computed(() => {
  const query = search.value.trim().toLowerCase();
  return records.value.filter((record) => {
    const matchesStatus =
      !statusFilter.value || record.status === statusFilter.value;
    const matchesQuery =
      !query ||
      record.cleanerName.toLowerCase().includes(query) ||
      record.bookingNumber.toLowerCase().includes(query);
    return matchesStatus && matchesQuery;
  });
});

const totalPages = computed(() =>
  Math.max(1, Math.ceil(filteredRecords.value.length / pageSize)),
);

const pagedRecords = computed(() => {
  const start = (page.value - 1) * pageSize;
  return filteredRecords.value.slice(start, start + pageSize);
});

async function fetchRecords() {
  loading.value = true;
  try {
    records.value = await getServiceRecords();
  } catch {
    showToast("Failed to load service records", "error");
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  editingRecord.value = undefined;
  showForm.value = true;
}

function openEdit(record: ServiceRecord) {
  editingRecord.value = record;
  showForm.value = true;
}

function closeForm() {
  showForm.value = false;
  editingRecord.value = undefined;
}

async function handleSave(input: CreateServiceRecordInput) {
  saving.value = true;
  try {
    if (editingRecord.value) {
      const updated = await updateServiceRecord(editingRecord.value.id, input);
      records.value = records.value.map((r) =>
        r.id === updated.id ? updated : r,
      );
      showToast("Service record updated", "success");
    } else {
      const created = await createServiceRecord(input);
      records.value = [created, ...records.value];
      showToast("Service record created", "success");
    }
    closeForm();
  } catch {
    showToast("Failed to save service record", "error");
  } finally {
    saving.value = false;
  }
}

async function handleDelete() {
  if (!pendingDelete.value) return;
  deleting.value = true;
  try {
    await deleteServiceRecord(pendingDelete.value.id);
    records.value = records.value.filter(
      (r) => r.id !== pendingDelete.value!.id,
    );
    showToast("Service record deleted", "success");
  } catch {
    showToast("Failed to delete service record", "error");
  } finally {
    deleting.value = false;
    pendingDelete.value = null;
  }
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString();
}

onMounted(fetchRecords);
</script>

<template>
  <div class="rounded-xl border border-slate-200 bg-white">
    <div
      class="flex flex-col gap-3 border-b border-slate-200 px-6 py-4 sm:flex-row sm:items-center"
    >
      <h2 class="text-base font-semibold text-slate-900">Service Records</h2>
      <span
        class="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-600"
      >
        {{ filteredRecords.length }}
      </span>

      <div class="sm:ml-auto flex flex-col gap-2 sm:flex-row">
        <input
          v-model="search"
          @input="page = 1"
          type="search"
          placeholder="Search booking or cleaner..."
          class="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 sm:w-56"
        />
        <select
          v-model="statusFilter"
          @change="page = 1"
          class="rounded-lg border border-slate-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
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
          type="button"
          class="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
          @click="openCreate"
        >
          Upload
        </button>
      </div>
    </div>

    <div v-if="loading" class="px-6 py-16 text-center">
      <p class="text-sm text-slate-500">Loading service records...</p>
    </div>

    <div
      v-else-if="filteredRecords.length === 0"
      class="px-6 py-16 text-center"
    >
      <p class="text-sm font-medium text-slate-900">No service records found</p>
      <p class="mt-1 text-sm text-slate-500">
        Try adjusting your filters or add a new service record.
      </p>
    </div>

    <div v-else class="overflow-x-auto">
      <table class="w-full text-left text-sm">
        <thead>
          <tr
            class="border-b border-slate-200 text-xs uppercase tracking-wide text-slate-500"
          >
            <th class="px-6 py-3 font-medium">Booking #</th>
            <th class="px-6 py-3 font-medium">Cleaner</th>
            <th class="px-6 py-3 font-medium">Service Type</th>
            <th class="px-6 py-3 font-medium">Rating</th>
            <th class="px-6 py-3 font-medium">Status</th>
            <th class="px-6 py-3 font-medium">Completed</th>
            <th class="px-6 py-3 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          <tr
            v-for="record in pagedRecords"
            :key="record.id"
            class="hover:bg-slate-50"
          >
            <td class="px-6 py-4 font-mono text-xs text-indigo-600">
              {{ record.bookingNumber }}
            </td>
            <td class="px-6 py-4 font-medium text-slate-900">
              {{ record.cleanerName }}
            </td>
            <td class="px-6 py-4 text-slate-600">
              {{ serviceTypeLabels[record.serviceType] ?? record.serviceType }}
            </td>
            <td class="px-6 py-4 text-slate-600">
              {{ record.rating != null ? `${record.rating}/5` : "—" }}
            </td>
            <td class="px-6 py-4">
              <span
                :class="[
                  'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
                  statusStyles[record.status],
                ]"
              >
                {{ statusLabels[record.status] }}
              </span>
            </td>
            <td class="px-6 py-4 text-slate-600">
              {{ record.completedAt ? formatDate(record.completedAt) : "—" }}
            </td>
            <td class="px-6 py-4 text-right whitespace-nowrap">
              <button
                type="button"
                class="rounded-lg px-2 py-1 text-sm font-medium text-slate-600 hover:bg-slate-100 hover:text-slate-900"
                @click="openEdit(record)"
              >
                Edit
              </button>
              <button
                type="button"
                class="rounded-lg px-2 py-1 text-sm font-medium text-red-600 hover:bg-red-50"
                @click="pendingDelete = record"
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
      class="flex items-center justify-between border-t border-slate-200 px-6 py-3"
    >
      <p class="text-sm text-slate-500">Page {{ page }} of {{ totalPages }}</p>
      <div class="flex gap-2">
        <button
          type="button"
          :disabled="page <= 1"
          class="rounded-lg border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-40 disabled:hover:bg-transparent"
          @click="page--"
        >
          Previous
        </button>
        <button
          type="button"
          :disabled="page >= totalPages"
          class="rounded-lg border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-40 disabled:hover:bg-transparent"
          @click="page++"
        >
          Next
        </button>
      </div>
    </div>
  </div>

  <ServiceRecordForm
    v-if="showForm"
    :key="editingRecord?.id ?? 'new'"
    :record="editingRecord"
    @save="handleSave"
    @cancel="closeForm"
  />

  <ConfirmDialog
    v-if="pendingDelete"
    title="Delete service record"
    :message="`Are you sure you want to delete ${pendingDelete.bookingNumber}? This action cannot be undone.`"
    confirm-label="Delete service record"
    :busy="deleting"
    @confirm="handleDelete"
    @cancel="pendingDelete = null"
  />
</template>
