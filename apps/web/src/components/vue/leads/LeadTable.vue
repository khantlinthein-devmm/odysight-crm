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
  } catch (err) {
    showToast(
      err instanceof Error ? `Failed to delete lead: ${err.message}` : "Failed to delete lead",
      "error",
    );
  } finally {
    deleting.value = false;
    pendingDelete.value = null;
  }
}

// Pipeline steps in order; "lost" sits outside it.
const PIPELINE: LeadStatus[] = ["new", "contacted", "quote_sent", "booked", "won"];

function stageIndex(lead: Lead): number {
  return Math.max(0, PIPELINE.indexOf(lead.status));
}

const sourceTone: Record<string, string> = {
  website: "bg-slate-100 text-slate-700",
  referral: "bg-violet-100 text-violet-700",
  line: "bg-green-100 text-green-700",
  facebook: "bg-blue-100 text-blue-700",
  walk_in: "bg-amber-100 text-amber-800",
  campaign: "bg-pink-100 text-pink-700",
};

// A lead still "new" after arriving in the last two days gets highlighted.
function isFresh(lead: Lead): boolean {
  return lead.status === "new" && Date.now() - new Date(lead.createdAt).getTime() < 2 * 86_400_000;
}

function timeAgo(iso: string): string {
  const mins = Math.max(0, Math.round((Date.now() - new Date(iso).getTime()) / 60_000));
  if (mins < 60) return `${mins || 1} min ago`;
  const hours = Math.round(mins / 60);
  if (hours < 24) return `${hours} h ago`;
  const days = Math.round(hours / 24);
  if (days < 30) return `${days} day${days === 1 ? "" : "s"} ago`;
  return formatDate(iso);
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
        :class="[
          'group relative flex flex-col rounded-2xl bg-white p-5 shadow-sm ring-1 transition duration-200 hover:-translate-y-0.5 hover:shadow-lg',
          isFresh(lead) ? 'ring-amber-300' : 'ring-gray-200 hover:ring-navy-200',
        ]"
      >
        <!-- Where it came from and how long it has waited -->
        <div class="flex items-center justify-between gap-2">
          <span
            :class="[
              'inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-bold',
              (sourceTone[lead.source] ?? sourceTone.website)!,
            ]"
          >
            <span class="h-1.5 w-1.5 rounded-full bg-current opacity-70"></span>
            {{ sourceLabels[lead.source] ?? lead.source }}
          </span>
          <span
            v-if="isFresh(lead)"
            class="inline-flex items-center gap-1.5 text-[11px] font-semibold text-amber-700"
            title="New lead: reply quickly"
          >
            <span class="relative flex h-2 w-2">
              <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-amber-400 opacity-75"></span>
              <span class="relative inline-flex h-2 w-2 rounded-full bg-amber-500"></span>
            </span>
            {{ timeAgo(lead.createdAt) }}
          </span>
          <span v-else class="text-[11px] font-medium text-gray-400">{{ timeAgo(lead.createdAt) }}</span>
        </div>

        <div class="mt-4 flex items-center gap-3">
          <img
            v-if="lead.linePictureUrl"
            :src="lead.linePictureUrl"
            alt=""
            class="h-12 w-12 shrink-0 rounded-full object-cover ring-2 ring-white shadow"
          />
          <span
            v-else
            class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-navy-100 to-navy-200 text-base font-bold text-navy-700"
          >
            {{ (lead.firstName || "?").charAt(0).toUpperCase() }}{{ (lead.lastName || "").charAt(0).toUpperCase() }}
          </span>
          <div class="min-w-0">
            <a
              :href="`/leads/${lead.id}`"
              class="block truncate text-base font-semibold text-gray-900 hover:text-navy-700"
            >
              {{ lead.firstName }} {{ lead.lastName }}
            </a>
            <p class="truncate text-sm text-gray-500">
              <span class="tabular-nums">{{ lead.phone || "No phone" }}</span>
              <span v-if="lead.email" class="text-gray-400"> &middot; {{ lead.email }}</span>
            </p>
          </div>
        </div>

        <!-- Pipeline: how far along this lead is -->
        <div class="mt-5">
          <div class="mb-1.5 flex items-center justify-between text-xs">
            <span :class="['font-semibold', lead.status === 'lost' ? 'text-gray-500' : 'text-gray-900']">
              {{ statusLabels[lead.status] ?? lead.status }}
            </span>
            <span v-if="lead.status !== 'lost'" class="text-gray-400">
              Step {{ stageIndex(lead) + 1 }} of {{ PIPELINE.length }}
            </span>
            <span v-else class="font-semibold text-gray-400">Closed</span>
          </div>
          <div class="flex gap-1" role="img" :aria-label="`Stage: ${statusLabels[lead.status] ?? lead.status}`">
            <span
              v-for="(stage, i) in PIPELINE"
              :key="stage"
              :class="[
                'h-1.5 flex-1 rounded-full transition-colors',
                lead.status === 'lost'
                  ? 'bg-gray-200'
                  : i <= stageIndex(lead)
                    ? (statusAccent[lead.status] ?? 'bg-navy-500')
                    : 'bg-gray-100',
              ]"
              :title="statusLabels[stage]"
            ></span>
          </div>
        </div>

        <div class="mt-auto flex items-center gap-2 pt-5">
          <a
            :href="`/leads/${lead.id}`"
            class="flex-1 rounded-xl bg-gray-900 px-3 py-2 text-center text-sm font-semibold text-white hover:bg-gray-800"
          >
            Open lead
          </a>
          <a
            v-if="lead.phone"
            :href="`tel:${lead.phone}`"
            class="rounded-xl bg-gray-100 p-2 text-gray-600 hover:bg-gray-200"
            :aria-label="`Call ${lead.firstName}`"
            :title="`Call ${lead.phone}`"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" /></svg>
          </a>
          <button
            type="button"
            class="rounded-xl bg-gray-100 px-3 py-2 text-sm font-semibold text-gray-700 hover:bg-gray-200"
            @click="openEdit(lead)"
          >
            Edit
          </button>
          <button
            type="button"
            class="rounded-xl p-2 text-gray-400 transition hover:bg-red-50 hover:text-red-600"
            :aria-label="`Delete ${lead.firstName} ${lead.lastName}`.trim()"
            title="Delete"
            @click="pendingDelete = lead"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
          </button>
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
    :title="`${pendingDelete.firstName} ${pendingDelete.lastName}`.trim()"
    message="Are you sure you want to delete this lead? This action cannot be undone."
    confirm-label="Delete lead"
    :busy="deleting"
    @confirm="handleDelete"
    @cancel="pendingDelete = null"
  />
</template>
