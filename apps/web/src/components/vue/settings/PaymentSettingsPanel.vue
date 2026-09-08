<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ApiError } from "../../../lib/api";
import {
  getWorkspaceSettings,
  updateWorkspaceSettings,
} from "../../../lib/settings";
import { showToast } from "../../../lib/toast";

import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";

const editable = computed(() =>
  hasPermission(getSessionUser()?.role, "settings.manage"),
);

const taxRate = ref(7);
const methodsText = ref("");
const loading = ref(true);
const saving = ref(false);
const error = ref<string | null>(null);

onMounted(async () => {
  try {
    const p = (await getWorkspaceSettings()).payments;
    taxRate.value = p.taxRatePercent;
    methodsText.value = p.methods.join("\n");
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to load";
  } finally {
    loading.value = false;
  }
});

async function save() {
  error.value = null;
  const methods = methodsText.value
    .split("\n")
    .map((m) => m.trim().toLowerCase().replace(/[^a-z0-9]+/g, "_"))
    .filter(Boolean);
  if (methods.length === 0) {
    error.value = "At least one payment method is required.";
    return;
  }
  saving.value = true;
  try {
    await updateWorkspaceSettings({
      payments: { taxRatePercent: Number(taxRate.value), methods },
    });
    showToast("Payment settings saved — New Payment form uses these methods", "success");
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
    <h2 class="text-base font-semibold text-gray-900">Payments</h2>
    <p class="mt-1 text-sm text-gray-500">
      Currency comes from Localization. Methods feed the New Payment form.
    </p>
    <div v-if="loading" class="mt-4 text-sm text-gray-500">Loading…</div>
    <div v-else class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Tax rate (%)</span>
        <input v-model.number="taxRate" type="number" min="0" max="100" step="0.01" :class="input" :disabled="!editable" />
      </label>
      <div class="hidden sm:block"></div>
      <label class="block sm:col-span-2">
        <span class="mb-1 block text-xs font-medium text-gray-600">Payment methods (one id per line)</span>
        <textarea v-model="methodsText" rows="6" :class="input" :disabled="!editable"></textarea>
        <span class="mt-1 block text-xs text-gray-400">Ids are lowercased with underscores, e.g. bank_transfer. Labels render automatically.</span>
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
