<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  getWorkspaceSettings,
  updateWorkspaceSettings,
  type SmsSettings,
} from "../../../lib/settings";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";
import { showToast } from "../../../lib/toast";

const props = defineProps<{
  channel: "sms" | "whatsapp";
}>();

const editable = computed(() =>
  hasPermission(getSessionUser()?.role, "settings.manage"),
);

const label = computed(() =>
  props.channel === "sms" ? "SMS" : "WhatsApp",
);

const cfg = ref<SmsSettings>({
  enabled: false,
  provider: "",
  webhookUrl: "",
  apiKey: "",
  fromNumber: "",
});
const loading = ref(true);
const saving = ref(false);
const error = ref("");
const apiKeyDirty = ref(false);

const MASKED = "••••••••";

onMounted(async () => {
  try {
    const ws = await getWorkspaceSettings();
    cfg.value = { ...ws[props.channel] };
    if (cfg.value.apiKey && cfg.value.apiKey !== MASKED) {
      cfg.value.apiKey = MASKED;
    }
  } catch {
    error.value = "Failed to load settings";
  } finally {
    loading.value = false;
  }
});

async function save() {
  error.value = "";
  if (cfg.value.enabled && (!cfg.value.provider.trim() || !cfg.value.webhookUrl.trim())) {
    error.value = "Provider and webhook URL are required when enabled";
    return;
  }
  saving.value = true;
  try {
    const payload: SmsSettings = { ...cfg.value };
    // Write-only API key: if untouched, send empty so the server keeps the stored key.
    if (!apiKeyDirty.value) payload.apiKey = "";
    const ws = await updateWorkspaceSettings({ [props.channel]: payload });
    cfg.value = { ...ws[props.channel] };
    cfg.value.apiKey = cfg.value.apiKey ? MASKED : "";
    apiKeyDirty.value = false;
    showToast(`${label.value} settings saved`);
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to save settings";
  } finally {
    saving.value = false;
  }
}

const inputClass =
  "w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500";
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-white p-6">
    <div class="flex items-center justify-between">
      <div>
        <h3 class="text-sm font-semibold text-gray-900">{{ label }} settings</h3>
        <p class="mt-1 text-sm text-gray-500">
          Send booking reminders and invoice alerts via {{ label }}.
        </p>
      </div>
      <label class="flex items-center gap-2 text-sm text-gray-700">
        <input
          v-model="cfg.enabled"
          type="checkbox"
          :disabled="!editable || loading"
          class="h-4 w-4 rounded border-gray-300 text-navy-600 focus:ring-navy-500"
        />
        Enabled
      </label>
    </div>

    <div v-if="loading" class="mt-4 h-24 animate-pulse rounded-lg bg-navy-50" />

    <div v-else class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
      <div>
        <label class="mb-1 block text-sm font-medium text-gray-700">Provider</label>
        <select
          v-model="cfg.provider"
          :disabled="!editable"
          class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
        >
          <option value="">Not set</option>
          <option value="http">Generic webhook (Twilio-style)</option>
        </select>
      </div>

      <div>
        <label class="mb-1 block text-sm font-medium text-gray-700">From number</label>
        <input
          v-model="cfg.fromNumber"
          :disabled="!editable"
          class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
          placeholder="+66901234567"
        />
      </div>

      <div class="sm:col-span-2">
        <label class="mb-1 block text-sm font-medium text-gray-700">Webhook URL</label>
        <input
          v-model="cfg.webhookUrl"
          :disabled="!editable"
          class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
          placeholder="https://api.your-sms-provider.com/send"
        />
      </div>

      <div class="sm:col-span-2">
        <label class="mb-1 block text-sm font-medium text-gray-700">API key</label>
        <input
          v-model="cfg.apiKey"
          type="password"
          :disabled="!editable"
          placeholder="Stored securely; leave blank to keep the current key"
          :class="inputClass"
          @input="apiKeyDirty = true"
        />
        <p class="mt-1 text-xs text-gray-500">
          The API key is write-only: it is masked here and never returned.
        </p>
      </div>
    </div>

    <p v-if="error" class="mt-3 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700">
      {{ error }}
    </p>

    <div class="mt-4 flex justify-end">
      <button
        type="button"
        :disabled="!editable || saving || loading"
        class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50"
        @click="save"
      >
        {{ saving ? "Saving…" : "Save" }}
      </button>
    </div>
  </div>
</template>