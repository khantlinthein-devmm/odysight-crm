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
  quote_sent: "Quote Sent",
  booked: "Booked",
  won: "Won",
  lost: "Lost",
};

const statusStyles: Record<LeadStatus, string> = {
  new: "bg-amber-100 text-amber-800 ring-1 ring-amber-300",
  contacted: "bg-blue-100 text-blue-800 ring-1 ring-blue-300",
  quote_sent: "bg-purple-100 text-purple-800 ring-1 ring-purple-300",
  booked: "bg-orange-100 text-orange-800 ring-1 ring-orange-300",
  won: "bg-green-100 text-green-800 ring-1 ring-green-300",
  lost: "bg-gray-200 text-gray-700 ring-1 ring-gray-300",
};

const statusAccent: Record<LeadStatus, string> = {
  new: "bg-amber-400",
  contacted: "bg-blue-500",
  quote_sent: "bg-purple-500",
  booked: "bg-orange-500",
  won: "bg-green-500",
  lost: "bg-gray-300",
};

const sourceLabels: Record<string, string> = {
  website: "Website",
  referral: "Referral",
  line: "LINE",
  facebook: "Facebook",
  walk_in: "Walk-in",
  campaign: "Campaign",
};

const leads = ref<Lead[]>([]);
const loading = ref(true);
const search = ref("");
const statusFilter = ref<LeadStatus | "">("");
const page = ref(1);
const pageSize = 9;

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
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to save lead", "error");
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
  <div class="rounded-xl border border-gray-200 bg-gray-100/70">
    <div
      class="flex flex-col gap-3 rounded-t-xl border-b border-gray-200 bg-white px-6 py-4 sm:flex-row sm:items-center"
    >
      <h2 class="text-base font-semibold text-gray-900">Leads</h2>
      <span
        class="rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-600"
      >
        {{ filteredLeads.length }}
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
          class="rounded-lg border border-gray-200 px-3 py-2 text-sm capitalize focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
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
          class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700"
          @click="openCreate"
        >
          New Lead
        </button>
      </div>
    </div>

    <div v-if="loading" class="px-6 py-16 text-center">
      <p class="text-sm text-gray-500">Loading leads...</p>
    </div>

    <div v-else-if="filteredLeads.length === 0" class="px-6 py-16 text-center">
      <p class="text-sm font-medium text-gray-900">No leads found</p>
      <p class="mt-1 text-sm text-gray-500">
        Try adjusting your filters or create a new lead.
      </p>
    </div>

    <div v-else class="grid grid-cols-1 gap-4 p-4 sm:p-6 md:grid-cols-2 xl:grid-cols-3">
      <article
        v-for="lead in pagedLeads"
        :key="lead.id"
        class="flex flex-col overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm transition hover:shadow-md"
      >
        <div :class="['h-1.5 w-full', statusAccent[lead.status] ?? 'bg-gray-300']"></div>
        <div class="flex flex-1 flex-col p-5">
          <div class="flex items-center justify-between gap-2">
            <span class="rounded bg-gray-900 px-2 py-0.5 font-mono text-[11px] font-semibold tracking-wide text-white">
              #{{ lead.id }}
            </span>
            <span
              :class="[
                'inline-flex shrink-0 items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-bold',
                statusStyles[lead.status] ?? 'bg-gray-100 text-gray-600',
              ]"
            >
              <span :class="['h-1.5 w-1.5 rounded-full', statusAccent[lead.status] ?? 'bg-gray-400']"></span>
              {{ statusLabels[lead.status] ?? lead.status }}
            </span>
          </div>

          <div class="mt-3 flex items-center gap-3">
            <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-navy-600 text-base font-bold text-white shadow-sm">
              {{ (lead.firstName || "?").charAt(0).toUpperCase() }}{{ (lead.lastName || "").charAt(0).toUpperCase() }}
            </span>
            <div class="min-w-0">
              <a
                :href="`/leads/${lead.id}`"
                class="block truncate text-base font-bold text-gray-900 hover:text-navy-700"
              >
                {{ lead.firstName }} {{ lead.lastName }}
              </a>
              <p class="mt-0.5 flex flex-wrap items-center gap-1.5">
                <span class="inline-flex rounded-md bg-slate-100 px-2 py-0.5 text-xs font-semibold text-slate-700 ring-1 ring-slate-200">
                  {{ sourceLabels[lead.source] ?? lead.source }}
                </span>
                <span
                  v-if="lead.lineUserId"
                  class="inline-flex rounded-md bg-green-100 px-2 py-0.5 text-xs font-semibold text-green-800 ring-1 ring-green-200"
                >
                  LINE
                </span>
              </p>
            </div>
          </div>

          <dl class="mt-4 space-y-2 rounded-lg bg-gray-50 p-3 text-sm ring-1 ring-gray-100">
            <div class="flex items-center gap-2">
              <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-blue-100 text-blue-700"><svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" /></svg></span>
              <dd class="min-w-0 truncate text-gray-700">{{ lead.email || "—" }}</dd>
            </div>
            <div class="flex items-center gap-2">
              <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-green-100 text-green-700"><svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" /></svg></span>
              <dd class="min-w-0 truncate font-medium text-gray-800">{{ lead.phone || "—" }}</dd>
            </div>
            <div class="flex items-center gap-2">
              <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-gray-100 text-gray-500"><svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg></span>
              <dd class="min-w-0 text-gray-600">{{ formatDate(lead.createdAt) }}</dd>
            </div>
          </dl>

          <div class="mt-4 flex flex-wrap items-center gap-2 border-t border-gray-100 pt-3">
            <a
              :href="`/leads/${lead.id}`"
              class="rounded-lg border border-navy-200 bg-navy-50 px-3 py-1.5 text-sm font-semibold text-navy-700 hover:bg-navy-100"
            >
              View
            </a>
            <button
              type="button"
              class="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-sm font-semibold text-gray-700 hover:bg-gray-100"
              @click="openEdit(lead)"
            >
              Edit
            </button>
            <button
              type="button"
              class="ml-auto rounded-lg px-2.5 py-1.5 text-sm font-medium text-red-600 hover:bg-red-50"
              @click="pendingDelete = lead"
            >
              Delete
            </button>
          </div>
        </div>
      </article>
    </div>

    <div
      v-if="!loading && totalPages > 1"
      class="flex items-center justify-between rounded-b-xl border-t border-gray-200 bg-white px-6 py-3"
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
