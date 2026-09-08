<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ApiError } from "../../../lib/api";
import {
  getWorkspaceSettings,
  updateWorkspaceSettings,
  type BookingSettings,
} from "../../../lib/settings";
import { showToast } from "../../../lib/toast";

import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";

const editable = computed(() =>
  hasPermission(getSessionUser()?.role, "settings.manage"),
);

const form = ref<BookingSettings>({
  defaultDurationMinutes: 120,
  bufferMinutes: 30,
  workStart: "08:00",
  workEnd: "18:00",
  holidays: [],
});
const holidaysText = ref("");
const loading = ref(true);
const saving = ref(false);
const error = ref<string | null>(null);

onMounted(async () => {
  try {
    const b = (await getWorkspaceSettings()).booking;
    form.value = { ...b, holidays: [...(b.holidays ?? [])] };
    holidaysText.value = form.value.holidays.join("\n");
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to load";
  } finally {
    loading.value = false;
  }
});

async function save() {
  error.value = null;
  form.value.holidays = holidaysText.value
    .split("\n")
    .map((d) => d.trim())
    .filter(Boolean);
  saving.value = true;
  try {
    await updateWorkspaceSettings({ booking: form.value });
    showToast("Booking defaults saved", "success");
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
    <h2 class="text-base font-semibold text-gray-900">Booking defaults</h2>
    <p class="mt-1 text-sm text-gray-500">
      Prefills the New Booking form — schedulers can still override per booking.
    </p>
    <div v-if="loading" class="mt-4 text-sm text-gray-500">Loading…</div>
    <div v-else class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Default duration (minutes)</span>
        <input v-model.number="form.defaultDurationMinutes" type="number" min="15" max="1440" :class="input" :disabled="!editable" />
      </label>
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Buffer between bookings (minutes)</span>
        <input v-model.number="form.bufferMinutes" type="number" min="0" max="1440" :class="input" :disabled="!editable" />
      </label>
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Working hours start</span>
        <input v-model="form.workStart" type="time" :class="input" :disabled="!editable" />
      </label>
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Working hours end</span>
        <input v-model="form.workEnd" type="time" :class="input" :disabled="!editable" />
      </label>
      <label class="block sm:col-span-2">
        <span class="mb-1 block text-xs font-medium text-gray-600">Holidays (one YYYY-MM-DD per line)</span>
        <textarea v-model="holidaysText" rows="3" placeholder="2026-12-25&#10;2026-12-31" :class="input" :disabled="!editable"></textarea>
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
