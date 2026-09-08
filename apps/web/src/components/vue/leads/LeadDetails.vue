<script setup lang="ts">
import { onMounted, ref } from "vue";
import {
  getLead,
  type CreateLeadInput,
  type Lead,
  type LeadStatus,
} from "../../../lib/leads";
import { showToast } from "../../../lib/toast";
import LeadForm from "./LeadForm.vue";

const statusLabels: Record<LeadStatus, string> = {
  new: "New",
  contacted: "Contacted",
  qualified: "Qualified",
  proposal: "Proposal",
  won: "Won",
  lost: "Lost",
};

const sourceLabels: Record<string, string> = {
  website: "Website",
  referral: "Referral",
  social_media: "Social Media",
  walk_in: "Walk-in",
  campaign: "Campaign",
};

const props = defineProps<{
  leadId: number;
}>();

const lead = ref<Lead | null>(null);
const loading = ref(true);
const error = ref(false);
const showForm = ref(false);
const saving = ref(false);

async function fetchLead() {
  loading.value = true;
  error.value = false;
  try {
    lead.value = await getLead(props.leadId);
  } catch {
    error.value = true;
  } finally {
    loading.value = false;
  }
}

async function handleSave(input: CreateLeadInput) {
  if (!lead.value) return;
  saving.value = true;
  try {
    const { updateLead } = await import("../../../lib/leads");
    lead.value = await updateLead(lead.value.id, input);
    showToast("Lead updated", "success");
    showForm.value = false;
  } catch {
    showToast("Failed to update lead", "error");
  } finally {
    saving.value = false;
  }
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  });
}

onMounted(fetchLead);
</script>

<template>
  <div
    v-if="loading"
    class="rounded-xl border border-gray-200 bg-white px-6 py-16 text-center"
  >
    <p class="text-sm text-gray-500">Loading lead...</p>
  </div>

  <div
    v-else-if="error || !lead"
    class="rounded-xl border border-gray-200 bg-white px-6 py-16 text-center"
  >
    <p class="text-sm font-medium text-gray-900">Lead not found</p>
    <a
      href="/leads"
      class="mt-2 inline-block text-sm font-medium text-navy-600 hover:text-navy-700"
    >
      Back to leads
    </a>
  </div>

  <template v-else>
    <nav class="mb-4 flex items-center gap-2 text-sm text-gray-500">
      <a href="/leads" class="hover:text-navy-700">Leads</a>
      <span>/</span>
      <span class="font-medium text-gray-900"
        >{{ lead.firstName }} {{ lead.lastName }}</span
      >
    </nav>

    <div class="rounded-xl border border-gray-200 bg-white">
      <div
        class="flex flex-col gap-3 border-b border-gray-200 px-6 py-4 sm:flex-row sm:items-center"
      >
        <div class="flex items-center gap-3">
          <div
            class="flex h-11 w-11 items-center justify-center rounded-full bg-gray-100 text-base font-semibold text-gray-700"
          >
            {{ lead.firstName[0] }}{{ lead.lastName[0] }}
          </div>
          <div>
            <h2 class="text-base font-semibold text-gray-900">
              {{ lead.firstName }} {{ lead.lastName }}
            </h2>
            <p class="text-xs text-gray-500">
              Lead #{{ lead.id }} · created {{ formatDate(lead.createdAt) }}
            </p>
          </div>
        </div>

        <div class="sm:ml-auto flex items-center gap-3">
          <span
            :class="[
              'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
              'bg-navy-50 text-navy-700',
            ]"
          >
            {{ statusLabels[lead.status] }}
          </span>
          <button
            type="button"
            :disabled="saving"
            class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50"
            @click="showForm = true"
          >
            Edit
          </button>
        </div>
      </div>

      <dl class="grid grid-cols-1 gap-x-6 gap-y-5 px-6 py-6 sm:grid-cols-2">
        <div>
          <dt
            class="text-xs font-medium uppercase tracking-wide text-gray-500"
          >
            Email
          </dt>
          <dd class="mt-1 text-sm text-gray-900">{{ lead.email }}</dd>
        </div>
        <div>
          <dt
            class="text-xs font-medium uppercase tracking-wide text-gray-500"
          >
            Phone
          </dt>
          <dd class="mt-1 text-sm text-gray-900">{{ lead.phone || "—" }}</dd>
        </div>
        <div>
          <dt
            class="text-xs font-medium uppercase tracking-wide text-gray-500"
          >
            Source
          </dt>
          <dd class="mt-1 text-sm text-gray-900">
            {{ sourceLabels[lead.source] ?? lead.source }}
          </dd>
        </div>
        <div>
          <dt
            class="text-xs font-medium uppercase tracking-wide text-gray-500"
          >
            Status
          </dt>
          <dd class="mt-1 text-sm text-gray-900">
            {{ statusLabels[lead.status] }}
          </dd>
        </div>
      </dl>

      <div class="border-t border-gray-200 px-6 py-4">
        <p class="text-xs text-gray-400">
          Activities, notes and conversion to customer will appear here in
          later phases.
        </p>
      </div>
    </div>
  </template>

  <LeadForm
    v-if="showForm && lead"
    :key="lead.id"
    :lead="lead"
    @save="handleSave"
    @cancel="showForm = false"
  />
</template>
