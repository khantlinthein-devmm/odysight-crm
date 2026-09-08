<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ApiError } from "../../../lib/api";
import {
  getWorkspaceSettings,
  updateWorkspaceSettings,
  type LocalizationSettings,
} from "../../../lib/settings";
import { showToast } from "../../../lib/toast";

import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";

const editable = computed(() =>
  hasPermission(getSessionUser()?.role, "settings.manage"),
);

const timezones = [
  "Asia/Bangkok",
  "Asia/Singapore",
  "Asia/Kuala_Lumpur",
  "Asia/Tokyo",
  "UTC",
  "Europe/London",
  "America/New_York",
];
const dateFormats = ["DD/MM/YYYY", "MM/DD/YYYY", "YYYY-MM-DD"];
const languages = [
  { value: "en", label: "English" },
  { value: "th", label: "ไทย (Thai)" },
];
const currencies = ["THB", "USD", "EUR", "GBP", "JPY", "SGD", "MYR"];

const form = ref<LocalizationSettings>({ timezone: "", dateFormat: "", language: "", currency: "" });
const loading = ref(true);
const saving = ref(false);
const error = ref<string | null>(null);

onMounted(async () => {
  try {
    form.value = { ...(await getWorkspaceSettings()).localization };
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to load";
  } finally {
    loading.value = false;
  }
});

async function save() {
  error.value = null;
  saving.value = true;
  try {
    await updateWorkspaceSettings({ localization: form.value });
    showToast("Localization saved — amounts now display in " + form.value.currency, "success");
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
    <h2 class="text-base font-semibold text-gray-900">Localization</h2>
    <p class="mt-1 text-sm text-gray-500">
      Timezone drives scheduling; currency drives every amount display.
    </p>
    <div v-if="loading" class="mt-4 text-sm text-gray-500">Loading…</div>
    <div v-else class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Timezone</span>
        <select v-model="form.timezone" :class="input" :disabled="!editable">
          <option v-for="tz in timezones" :key="tz" :value="tz">{{ tz }}</option>
        </select>
      </label>
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Date format</span>
        <select v-model="form.dateFormat" :class="input" :disabled="!editable">
          <option v-for="f in dateFormats" :key="f" :value="f">{{ f }}</option>
        </select>
      </label>
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Language</span>
        <select v-model="form.language" :class="input" :disabled="!editable">
          <option v-for="l in languages" :key="l.value" :value="l.value">{{ l.label }}</option>
        </select>
      </label>
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Currency</span>
        <select v-model="form.currency" :class="input" :disabled="!editable">
          <option v-for="c in currencies" :key="c" :value="c">{{ c }}</option>
        </select>
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
