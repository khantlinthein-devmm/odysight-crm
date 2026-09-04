<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  createVisaCase,
  getVisaCases,
  updateVisaCase,
  type CreateVisaCaseInput,
  type VisaCase,
  type VisaCaseStatus,
} from "../../../lib/visa-cases";
import { showToast } from "../../../lib/toast";
import VisaCaseForm from "./VisaCaseForm.vue";

const statusLabels: Record<VisaCaseStatus, string> = {
  draft: "Draft",
  in_review: "In Review",
  submitted: "Submitted",
  additional_docs_required: "Docs Required",
  approved: "Approved",
  rejected: "Rejected",
  closed: "Closed",
};

const statusStyles: Record<VisaCaseStatus, string> = {
  draft: "bg-slate-100 text-slate-600",
  in_review: "bg-sky-50 text-sky-700",
  submitted: "bg-indigo-50 text-indigo-700",
  additional_docs_required: "bg-amber-50 text-amber-700",
  approved: "bg-emerald-50 text-emerald-700",
  rejected: "bg-red-50 text-red-700",
  closed: "bg-violet-50 text-violet-700",
};

const visaCases = ref<VisaCase[]>([]);
const loading = ref(true);
const search = ref("");
const statusFilter = ref<VisaCaseStatus | "">("");
const page = ref(1);
const pageSize = 5;

const showForm = ref(false);
const editingCase = ref<VisaCase | undefined>(undefined);
const saving = ref(false);

const filteredCases = computed(() => {
  const query = search.value.trim().toLowerCase();
  return visaCases.value.filter((visaCase) => {
    const matchesStatus =
      !statusFilter.value || visaCase.status === statusFilter.value;
    const matchesQuery =
      !query ||
      visaCase.applicantName.toLowerCase().includes(query) ||
      visaCase.caseNumber.toLowerCase().includes(query);
    return matchesStatus && matchesQuery;
  });
});

const totalPages = computed(() =>
  Math.max(1, Math.ceil(filteredCases.value.length / pageSize)),
);

const pagedCases = computed(() => {
  const start = (page.value - 1) * pageSize;
  return filteredCases.value.slice(start, start + pageSize);
});

async function fetchCases() {
  loading.value = true;
  try {
    visaCases.value = await getVisaCases();
  } catch {
    showToast("Failed to load visa cases", "error");
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  editingCase.value = undefined;
  showForm.value = true;
}

function openEdit(visaCase: VisaCase) {
  editingCase.value = visaCase;
  showForm.value = true;
}

function closeForm() {
  showForm.value = false;
  editingCase.value = undefined;
}

async function handleSave(input: CreateVisaCaseInput) {
  saving.value = true;
  try {
    if (editingCase.value) {
      const updated = await updateVisaCase(editingCase.value.id, input);
      visaCases.value = visaCases.value.map((c) =>
        c.id === updated.id ? updated : c,
      );
      showToast("Visa case updated", "success");
    } else {
      const created = await createVisaCase(input);
      visaCases.value = [created, ...visaCases.value];
      showToast("Visa case created", "success");
    }
    closeForm();
  } catch {
    showToast("Failed to save visa case", "error");
  } finally {
    saving.value = false;
  }
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

onMounted(fetchCases);
</script>

<template>
  <div class="rounded-xl border border-slate-200 bg-white">
    <div
      class="flex flex-col gap-3 border-b border-slate-200 px-6 py-4 sm:flex-row sm:items-center"
    >
      <h2 class="text-base font-semibold text-slate-900">Visa Cases</h2>
      <span
        class="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-600"
      >
        {{ filteredCases.length }}
      </span>

      <div class="sm:ml-auto flex flex-col gap-2 sm:flex-row">
        <input
          v-model="search"
          @input="page = 1"
          type="search"
          placeholder="Search case or applicant..."
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
          New Case
        </button>
      </div>
    </div>

    <div v-if="loading" class="px-6 py-16 text-center">
      <p class="text-sm text-slate-500">Loading visa cases...</p>
    </div>

    <div v-else-if="filteredCases.length === 0" class="px-6 py-16 text-center">
      <p class="text-sm font-medium text-slate-900">No visa cases found</p>
      <p class="mt-1 text-sm text-slate-500">
        Try adjusting your filters or create a new case.
      </p>
    </div>

    <div v-else class="overflow-x-auto">
      <table class="w-full text-left text-sm">
        <thead>
          <tr
            class="border-b border-slate-200 text-xs uppercase tracking-wide text-slate-500"
          >
            <th class="px-6 py-3 font-medium">Case #</th>
            <th class="px-6 py-3 font-medium">Applicant</th>
            <th class="px-6 py-3 font-medium">Visa Type</th>
            <th class="px-6 py-3 font-medium">Destination</th>
            <th class="px-6 py-3 font-medium">Assigned To</th>
            <th class="px-6 py-3 font-medium">Status</th>
            <th class="px-6 py-3 font-medium">Opened</th>
            <th class="px-6 py-3 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          <tr
            v-for="visaCase in pagedCases"
            :key="visaCase.id"
            class="hover:bg-slate-50"
          >
            <td class="px-6 py-4 font-mono text-xs text-indigo-600">
              {{ visaCase.caseNumber }}
            </td>
            <td class="px-6 py-4 font-medium text-slate-900">
              {{ visaCase.applicantName }}
            </td>
            <td class="px-6 py-4 text-slate-600">{{ visaCase.visaType }}</td>
            <td class="px-6 py-4 text-slate-600">{{ visaCase.destination }}</td>
            <td class="px-6 py-4 text-slate-600">{{ visaCase.assignedTo }}</td>
            <td class="px-6 py-4">
              <span
                :class="[
                  'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
                  statusStyles[visaCase.status],
                ]"
              >
                {{ statusLabels[visaCase.status] }}
              </span>
            </td>
            <td class="px-6 py-4 text-slate-600">
              {{ formatDate(visaCase.createdAt) }}
            </td>
            <td class="px-6 py-4 text-right whitespace-nowrap">
              <button
                type="button"
                class="rounded-lg px-2 py-1 text-sm font-medium text-slate-600 hover:bg-slate-100 hover:text-slate-900"
                @click="openEdit(visaCase)"
              >
                Edit
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

  <VisaCaseForm
    v-if="showForm"
    :key="editingCase?.id ?? 'new'"
    :visa-case="editingCase"
    @save="handleSave"
    @cancel="closeForm"
  />
</template>
