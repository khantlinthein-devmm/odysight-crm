<script setup lang="ts">
import { computed, ref } from "vue";
import { changePassword, getSessionUser, logout } from "../../../lib/auth";
import { showToast } from "../../../lib/toast";

const user = computed(() => getSessionUser());

const current = ref("");
const next = ref("");
const confirm = ref("");
const saving = ref(false);
const error = ref<string | null>(null);

async function handleChange() {
  error.value = null;
  if (next.value.length < 8) {
    error.value = "New password must be at least 8 characters.";
    return;
  }
  if (next.value !== confirm.value) {
    error.value = "New passwords do not match.";
    return;
  }
  saving.value = true;
  try {
    await changePassword(current.value, next.value);
    showToast("Password changed", "success");
    current.value = "";
    next.value = "";
    confirm.value = "";
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to change password";
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-white p-6">
    <h2 class="text-base font-semibold text-gray-900">Security</h2>
    <p class="mt-1 text-sm text-gray-500">
      Signed in as
      <span class="font-medium text-gray-700">{{ user?.name ?? "…" }}</span>
      <span
        class="ml-1 rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600"
        >{{ user?.role ?? "" }}</span
      >
      · sessions expire after 8 hours.
    </p>

    <h3 class="mt-6 text-sm font-semibold text-gray-900">Change password</h3>
    <div class="mt-3 grid max-w-lg grid-cols-1 gap-3">
      <input
        v-model="current"
        type="password"
        autocomplete="current-password"
        placeholder="Current password"
        class="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
      />
      <input
        v-model="next"
        type="password"
        autocomplete="new-password"
        placeholder="New password (min 8 characters)"
        class="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
      />
      <input
        v-model="confirm"
        type="password"
        autocomplete="new-password"
        placeholder="Confirm new password"
        class="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
      />
    </div>
    <p
      v-if="error"
      class="mt-3 max-w-lg rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700"
    >
      {{ error }}
    </p>
    <div class="mt-4 flex max-w-lg flex-wrap gap-2">
      <button
        type="button"
        :disabled="saving"
        class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50"
        @click="handleChange"
      >
        {{ saving ? "Saving..." : "Change password" }}
      </button>
      <button
        type="button"
        class="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100"
        @click="logout()"
      >
        Sign out
      </button>
    </div>
  </div>
</template>
