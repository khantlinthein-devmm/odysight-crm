<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ApiError } from "../../../lib/api";
import {
  getWorkspaceSettings,
  updateWorkspaceSettings,
  type SmtpSettings,
} from "../../../lib/settings";
import { showToast } from "../../../lib/toast";

import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";

const editable = computed(() =>
  hasPermission(getSessionUser()?.role, "settings.manage"),
);

const cfg = ref<SmtpSettings>({
  enabled: false,
  host: "",
  port: 587,
  username: "",
  password: "",
  fromEmail: "",
  fromName: "",
  encryption: "starttls",
});
const passwordDirty = ref(false);
const loading = ref(true);
const saving = ref(false);
const error = ref<string | null>(null);

onMounted(async () => {
  try {
    cfg.value = (await getWorkspaceSettings()).smtp;
    passwordDirty.value = false;
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to load";
  } finally {
    loading.value = false;
  }
});

async function save() {
  error.value = null;
  if (cfg.value.enabled && !cfg.value.host.trim()) {
    error.value = "Host is required when SMTP is enabled.";
    return;
  }
  if (cfg.value.enabled && !cfg.value.fromEmail.trim()) {
    error.value = "From email is required when SMTP is enabled.";
    return;
  }
  saving.value = true;
  try {
    const payload: SmtpSettings = { ...cfg.value };
    if (!passwordDirty.value) {
      // Password is write-only and masked on read: send blank to keep the stored value.
      payload.password = "";
    }
    await updateWorkspaceSettings({ smtp: payload });
    passwordDirty.value = false;
    showToast(
      cfg.value.enabled
        ? "Email delivery enabled — invoices send automatically"
        : "Email delivery disabled",
      "success",
    );
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
    <h2 class="text-base font-semibold text-gray-900">Email (SMTP)</h2>
    <p class="mt-1 text-sm text-gray-500">
      Delivers invoice PDFs and payment receipts to customers. Use
      <code class="rounded bg-gray-100 px-1 text-xs">none</code> /
      <code class="rounded bg-gray-100 px-1 text-xs">starttls</code> /
      <code class="rounded bg-gray-100 px-1 text-xs">ssl</code> encryption.
    </p>
    <div v-if="loading" class="mt-4 text-sm text-gray-500">Loading…</div>
    <div v-else class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
      <label class="block sm:col-span-2">
        <input v-model="cfg.enabled" type="checkbox" :disabled="!editable" class="mr-2 rounded border-gray-300" />
        <span class="text-sm text-gray-700">Send invoice emails automatically</span>
      </label>
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Host</span>
        <input v-model="cfg.host" type="text" placeholder="smtp.example.com" :class="input" :disabled="!editable" />
      </label>
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Port</span>
        <input v-model.number="cfg.port" type="number" min="1" max="65535" :class="input" :disabled="!editable" />
      </label>
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Encryption</span>
        <select v-model="cfg.encryption" :class="input" :disabled="!editable">
          <option value="none">None</option>
          <option value="starttls">STARTTLS</option>
          <option value="ssl">SSL/TLS</option>
        </select>
      </label>
      <div class="hidden sm:block"></div>
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Username</span>
        <input v-model="cfg.username" type="text" autocomplete="off" :class="input" :disabled="!editable" />
      </label>
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Password</span>
        <input
          v-model="cfg.password"
          type="password"
          autocomplete="new-password"
          :placeholder="passwordDirty ? '' : '•••••••• (unchanged)'"
          :class="input"
          :disabled="!editable"
          @input="passwordDirty = true"
        />
      </label>
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">From email</span>
        <input v-model="cfg.fromEmail" type="email" placeholder="invoices@example.com" :class="input" :disabled="!editable" />
      </label>
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">From name</span>
        <input v-model="cfg.fromName" type="text" placeholder="Smile Clean" :class="input" :disabled="!editable" />
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