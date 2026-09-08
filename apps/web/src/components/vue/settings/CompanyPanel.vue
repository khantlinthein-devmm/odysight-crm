<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ApiError } from "../../../lib/api";
import {
  getWorkspaceSettings,
  updateWorkspaceSettings,
  type CompanySettings,
} from "../../../lib/settings";
import { showToast } from "../../../lib/toast";

import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";

const editable = computed(() =>
  hasPermission(getSessionUser()?.role, "settings.manage"),
);

const form = ref<CompanySettings>({ name: "", phone: "", address: "", invoiceFooter: "", logoUrl: "" });
const loading = ref(true);
const saving = ref(false);
const error = ref<string | null>(null);

onMounted(async () => {
  try {
    form.value = { ...(await getWorkspaceSettings()).company };
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to load";
  } finally {
    loading.value = false;
  }
});

async function save() {
  error.value = null;
  if (!form.value.name.trim()) {
    error.value = "Company name is required.";
    return;
  }
  saving.value = true;
  try {
    await updateWorkspaceSettings({ company: form.value });
    showToast("Company profile saved", "success");
  } catch (e) {
    error.value =
      e instanceof ApiError ? `API error ${e.status}: ${e.message}` : "Save failed";
  } finally {
    saving.value = false;
  }
}

const input =
  "w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500 disabled:bg-gray-100 disabled:text-gray-500";
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-white p-6">
    <h2 class="text-base font-semibold text-gray-900">Company profile</h2>
    <p class="mt-1 text-sm text-gray-500">
      Shown across the workspace and used on future invoices / receipts.
    </p>
    <div v-if="loading" class="mt-4 text-sm text-gray-500">Loading…</div>
    <div v-else class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Company name *</span>
        <input v-model="form.name" type="text" :class="input" :disabled="!editable" />
      </label>
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Phone</span>
        <input v-model="form.phone" type="tel" :class="input" :disabled="!editable" />
      </label>
      <label class="block sm:col-span-2">
        <span class="mb-1 block text-xs font-medium text-gray-600">Address</span>
        <input v-model="form.address" type="text" :class="input" :disabled="!editable" />
      </label>
      <label class="block sm:col-span-2">
        <span class="mb-1 block text-xs font-medium text-gray-600">Invoice footer</span>
        <input v-model="form.invoiceFooter" type="text" :class="input" :disabled="!editable" />
      </label>
      <label class="block sm:col-span-2">
        <span class="mb-1 block text-xs font-medium text-gray-600">Logo URL (optional)</span>
        <input v-model="form.logoUrl" type="url" placeholder="https://…" :class="input" :disabled="!editable" />
      </label>
    </div>
    <p v-if="error" class="mt-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{{ error }}</p>
    <div class="mt-4">
      <button
        v-if="editable"
        type="button"
        :disabled="saving || loading"
        class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50"
        @click="save"
      >
        {{ saving ? "Saving…" : "Save" }}
      </button>
      <p v-else class="text-xs text-gray-400">Restricted to admins.</p>
    </div>
  </div>
</template>
