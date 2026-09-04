<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  createLead,
  deleteLead,
  getLeads,
  updateLead,
  type CreateLeadInput,
  type Lead,
  type LeadStatus,
} from "./../../../lib/leads";
import { showToast } from "./../../../lib/toast";
import LeadForm from "./LeadForm.vue";
import ConfirmDialog from "../ui/ConfirmDialog.vue";

const statusLabels: Record<LeadStatus, string> = {
  new: "New",
  contacted: "Contacted",
  qualified: "Qualified",
  proposal: "Proposal",
  won: "Won",
  lost: "Lost",
};

const statusStyles: Record<LeadStatus, string> = {
  new: "bg-sky-50 text-sky-700",
  contacted: "bg-indigo-50 text-indigo-700",
  qualified: "bg-violet-50 text-violet-700",
  proposal: "bg-amber-50 text-amber-700",
  won: "bg-emerald-50 text-emerald-700",
  lost: "bg-slate-100 text-slate-600",
};

const sourceLabels: Record<string, string> = {
  website: "Website",
  referral: "Referral",
  social_media: "Social Media",
  walk_in: "Walk-in",
  campaign: "Campaign",
};

const leads = ref<Lead[]>([]);
const loading = ref(true);
const search = ref("");
const statusFilter = ref<LeadStatus | "">("");
const page = ref(1);
const pageSize = 5;

const showForm = ref(false);
const editingLead = ref<Lead | undefined>(undefined);
const saving = ref(false);
const pendingDelete = ref<Lead | null>(null);
const deleting = ref(false);

const filteredLeads = computed(() => {
  const query = search.value.trim().toLowerCase();
  return leads.value.filter((lead) => {
    const matchesStatus =
      !statusFilter.value || lead.status === statusFilter.value;
    const matchesQuery =
      !query ||
      `${lead.firstName} ${lead.lastName}`.toLowerCase().includes(query) ||
      lead.email.toLowerCase().includes(query);
    return matchesStatus && matchesQuery;
  });
});

const totalPages = computed(() =>
  Math.max(1, Math.ceil(filteredLeads.value.length / pageSize)),
);

const pagedLeads = computed(() => {
  const start = (page.value - 1) * pageSize;
  return filteredLeads.value.slice(start, start + pageSize);
});

async function fetchLeads() {
  loading.value = true;
  try {
    leads.value = await getLeads();
  } catch {
    showToast("Failed to load leads", "error");
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  editingLead.value = undefined;
  showForm.value = true;
}

function openEdit(lead: Lead) {
  editingLead.value = lead;
  showForm.value = true;
}

function closeForm() {
  showForm.value = false;
  editingLead.value = undefined;
}

async function handleSave(input: CreateLeadInput) {
  saving.value = true;
  try {
    if (editingLead.value) {
      const updated = await updateLead(editingLead.value.id, input);
      leads.value = leads.value.map((l) => (l.id === updated.id ? updated : l));
      showToast("Lead updated", "success");
    } else {
      const created = await createLead(input);
      leads.value = [created, ...leads.value];
      showToast("Lead created", "success");
    }
    closeForm();
  } catch {
    showToast("Failed to save lead", "error");
  } finally {
    saving.value = false;
  }
}

async function handleDelete() {
  if (!pendingDelete.value) return;
  deleting.value = true;
  try {
    await deleteLead(pendingDelete.value.id);
    leads.value = leads.value.filter((l) => l.id !== pendingDelete.value!.id);
    showToast("Lead deleted", "success");
  } catch {
    showToast("Failed to delete lead", "error");
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

onMounted(fetchLeads);
</script>

<template>
  <div class="rounded-xl border border-slate-200 bg-white">
    <div
      class="flex flex-col gap-3 border-b border-slate-200 px-6 py-4 sm:flex-row sm:items-center"
    >
      <h2 class="text-base font-semibold text-slate-900">Leads</h2>
      <span
        class="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-600"
      >
        {{ filteredLeads.length }}
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
          class="rounded-lg border border-slate-200 px-3 py-2 text-sm capitalize focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
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
          New Lead
        </button>
      </div>
    </div>

    <div v-if="loading" class="px-6 py-16 text-center">
      <p class="text-sm text-slate-500">Loading leads...</p>
    </div>

    <div v-else-if="filteredLeads.length === 0" class="px-6 py-16 text-center">
      <p class="text-sm font-medium text-slate-900">No leads found</p>
      <p class="mt-1 text-sm text-slate-500">
        Try adjusting your filters or create a new lead.
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
            <th class="px-6 py-3 font-medium">Phone</th>
            <th class="px-6 py-3 font-medium">Source</th>
            <th class="px-6 py-3 font-medium">Status</th>
            <th class="px-6 py-3 font-medium">Created</th>
            <th class="px-6 py-3 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          <tr
            v-for="lead in pagedLeads"
            :key="lead.id"
            class="hover:bg-slate-50"
          >
            <td class="px-6 py-4">
              <a
                :href="`/leads/${lead.id}`"
                class="font-medium text-indigo-600 hover:text-indigo-700"
              >
                {{ lead.firstName }} {{ lead.lastName }}
              </a>
            </td>
            <td class="px-6 py-4 text-slate-600">{{ lead.email }}</td>
            <td class="px-6 py-4 text-slate-600">{{ lead.phone }}</td>
            <td class="px-6 py-4 text-slate-600">
              {{ sourceLabels[lead.source] ?? lead.source }}
            </td>
            <td class="px-6 py-4">
              <span
                :class="[
                  'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
                  statusStyles[lead.status],
                ]"
              >
                {{ statusLabels[lead.status] }}
              </span>
            </td>
            <td class="px-6 py-4 text-slate-600">
              {{ formatDate(lead.createdAt) }}
            </td>
            <td class="px-6 py-4 text-right whitespace-nowrap">
              <button
                type="button"
                class="rounded-lg px-2 py-1 text-sm font-medium text-slate-600 hover:bg-slate-100 hover:text-slate-900"
                @click="openEdit(lead)"
              >
                Edit
              </button>
              <button
                type="button"
                class="rounded-lg px-2 py-1 text-sm font-medium text-red-600 hover:bg-red-50"
                @click="pendingDelete = lead"
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

  <LeadForm
    v-if="showForm"
    :key="editingLead?.id ?? 'new'"
    :lead="editingLead"
    @save="handleSave"
    @cancel="closeForm"
  />

  <ConfirmDialog
    v-if="pendingDelete"
    title="Delete lead"
    :message="`Are you sure you want to delete ${pendingDelete.firstName} ${pendingDelete.lastName}? This action cannot be undone.`"
    confirm-label="Delete lead"
    :busy="deleting"
    @confirm="handleDelete"
    @cancel="pendingDelete = null"
  />
</template>
