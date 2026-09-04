<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  createApplicant,
  deleteApplicant,
  getApplicants,
  updateApplicant,
  type Applicant,
  type ApplicantStatus,
  type CreateApplicantInput,
} from "../../../lib/applicants";
import { showToast } from "../../../lib/toast";
import ApplicantForm from "./ApplicantForm.vue";

const statusLabels: Record<ApplicantStatus, string> = {
  screening: "Screening",
  document_collection: "Document Collection",
  submitted: "Submitted",
  processing: "Processing",
  approved: "Approved",
  rejected: "Rejected",
};

const statusStyles: Record<ApplicantStatus, string> = {
  screening: "bg-sky-50 text-sky-700",
  document_collection: "bg-violet-50 text-violet-700",
  submitted: "bg-indigo-50 text-indigo-700",
  processing: "bg-amber-50 text-amber-700",
  approved: "bg-emerald-50 text-emerald-700",
  rejected: "bg-red-50 text-red-700",
};

const applicants = ref<Applicant[]>([]);
const loading = ref(true);
const search = ref("");
const statusFilter = ref<ApplicantStatus | "">("");
const page = ref(1);
const pageSize = 5;

const showForm = ref(false);
const editingApplicant = ref<Applicant | undefined>(undefined);
const saving = ref(false);
const deletingId = ref<number | null>(null);

const filteredApplicants = computed(() => {
  const query = search.value.trim().toLowerCase();
  return applicants.value.filter((applicant) => {
    const matchesStatus =
      !statusFilter.value || applicant.status === statusFilter.value;
    const matchesQuery =
      !query ||
      `${applicant.firstName} ${applicant.lastName}`
        .toLowerCase()
        .includes(query) ||
      applicant.email.toLowerCase().includes(query);
    return matchesStatus && matchesQuery;
  });
});

const totalPages = computed(() =>
  Math.max(1, Math.ceil(filteredApplicants.value.length / pageSize)),
);

const pagedApplicants = computed(() => {
  const start = (page.value - 1) * pageSize;
  return filteredApplicants.value.slice(start, start + pageSize);
});

async function fetchApplicants() {
  loading.value = true;
  try {
    applicants.value = await getApplicants();
  } catch {
    showToast("Failed to load applicants", "error");
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  editingApplicant.value = undefined;
  showForm.value = true;
}

function openEdit(applicant: Applicant) {
  editingApplicant.value = applicant;
  showForm.value = true;
}

function closeForm() {
  showForm.value = false;
  editingApplicant.value = undefined;
}

async function handleSave(input: CreateApplicantInput) {
  saving.value = true;
  try {
    if (editingApplicant.value) {
      const updated = await updateApplicant(editingApplicant.value.id, input);
      applicants.value = applicants.value.map((a) =>
        a.id === updated.id ? updated : a,
      );
      showToast("Applicant updated", "success");
    } else {
      const created = await createApplicant(input);
      applicants.value = [created, ...applicants.value];
      showToast("Applicant created", "success");
    }
    closeForm();
  } catch {
    showToast("Failed to save applicant", "error");
  } finally {
    saving.value = false;
  }
}

async function handleDelete(applicant: Applicant) {
  deletingId.value = applicant.id;
  try {
    await deleteApplicant(applicant.id);
    applicants.value = applicants.value.filter((a) => a.id !== applicant.id);
    showToast("Applicant deleted", "success");
  } catch {
    showToast("Failed to delete applicant", "error");
  } finally {
    deletingId.value = null;
  }
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

onMounted(fetchApplicants);
</script>

<template>
  <div class="rounded-xl border border-slate-200 bg-white">
    <div
      class="flex flex-col gap-3 border-b border-slate-200 px-6 py-4 sm:flex-row sm:items-center"
    >
      <h2 class="text-base font-semibold text-slate-900">Applicants</h2>
      <span
        class="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-600"
      >
        {{ filteredApplicants.length }}
      </span>

      <div class="sm:ml-auto flex flex-col gap-2 sm:flex-row">
        <input
          v-model="search"
          @input="page = 1"
          type="search"
          placeholder="Search name or email..."
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
          New Applicant
        </button>
      </div>
    </div>

    <div v-if="loading" class="px-6 py-16 text-center">
      <p class="text-sm text-slate-500">Loading applicants...</p>
    </div>

    <div
      v-else-if="filteredApplicants.length === 0"
      class="px-6 py-16 text-center"
    >
      <p class="text-sm font-medium text-slate-900">No applicants found</p>
      <p class="mt-1 text-sm text-slate-500">
        Try adjusting your filters or create a new applicant.
      </p>
    </div>

    <div v-else class="overflow-x-auto">
      <table class="w-full text-left text-sm">
        <thead>
          <tr
            class="border-b border-slate-200 text-xs uppercase tracking-wide text-slate-500"
          >
            <th class="px-6 py-3 font-medium">Name</th>
            <th class="px-6 py-3 font-medium">Email</th>
            <th class="px-6 py-3 font-medium">Nationality</th>
            <th class="px-6 py-3 font-medium">Visa Type</th>
            <th class="px-6 py-3 font-medium">Status</th>
            <th class="px-6 py-3 font-medium">Created</th>
            <th class="px-6 py-3 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          <tr
            v-for="applicant in pagedApplicants"
            :key="applicant.id"
            class="hover:bg-slate-50"
          >
            <td class="px-6 py-4">
              <a
                href="/applicants"
                class="font-medium text-indigo-600 hover:text-indigo-700"
              >
                {{ applicant.firstName }} {{ applicant.lastName }}
              </a>
            </td>
            <td class="px-6 py-4 text-slate-600">{{ applicant.email }}</td>
            <td class="px-6 py-4 text-slate-600">
              {{ applicant.nationality }}
            </td>
            <td class="px-6 py-4 text-slate-600">{{ applicant.visaType }}</td>
            <td class="px-6 py-4">
              <span
                :class="[
                  'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
                  statusStyles[applicant.status],
                ]"
              >
                {{ statusLabels[applicant.status] }}
              </span>
            </td>
            <td class="px-6 py-4 text-slate-600">
              {{ formatDate(applicant.createdAt) }}
            </td>
            <td class="px-6 py-4 text-right whitespace-nowrap">
              <button
                type="button"
                class="rounded-lg px-2 py-1 text-sm font-medium text-slate-600 hover:bg-slate-100 hover:text-slate-900"
                @click="openEdit(applicant)"
              >
                Edit
              </button>
              <button
                type="button"
                :disabled="deletingId === applicant.id"
                class="rounded-lg px-2 py-1 text-sm font-medium text-red-600 hover:bg-red-50 disabled:opacity-50"
                @click="handleDelete(applicant)"
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

  <ApplicantForm
    v-if="showForm"
    :key="editingApplicant?.id ?? 'new'"
    :applicant="editingApplicant"
    @save="handleSave"
    @cancel="closeForm"
  />
</template>
