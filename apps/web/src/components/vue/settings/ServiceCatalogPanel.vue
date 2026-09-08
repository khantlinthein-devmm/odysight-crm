<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ApiError } from "../../../lib/api";
import {
  formatMoney,
  getWorkspaceSettings,
  updateWorkspaceSettings,
  type ServiceItem,
} from "../../../lib/settings";
import { showToast } from "../../../lib/toast";

import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";

const editable = computed(() =>
  hasPermission(getSessionUser()?.role, "settings.manage"),
);

const items = ref<ServiceItem[]>([]);
const loading = ref(true);
const saving = ref(false);
const error = ref<string | null>(null);

const newId = ref("");
const newName = ref("");
const newDuration = ref(120);
const newPrice = ref(0);

function slugify(s: string): string {
  return s.trim().toLowerCase().replace(/[^a-z0-9]+/g, "_").replace(/^_+|_+$/g, "");
}

onMounted(async () => {
  try {
    items.value = [...(await getWorkspaceSettings()).services];
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to load";
  } finally {
    loading.value = false;
  }
});

function addItem() {
  error.value = null;
  const id = slugify(newId.value || newName.value);
  if (!id) {
    error.value = "Service needs an id (or a name to derive it from).";
    return;
  }
  if (items.value.some((s) => s.id === id)) {
    error.value = `Duplicate id "${id}".`;
    return;
  }
  if (!newName.value.trim()) {
    error.value = "Service needs a display name.";
    return;
  }
  items.value.push({
    id,
    name: newName.value.trim(),
    durationMinutes: Number(newDuration.value) || 60,
    basePrice: Number(newPrice.value) || 0,
    active: true,
  });
  newId.value = "";
  newName.value = "";
  newDuration.value = 120;
  newPrice.value = 0;
}

function removeItem(id: string) {
  items.value = items.value.filter((s) => s.id !== id);
}

async function save() {
  error.value = null;
  if (items.value.length === 0) {
    error.value = "Catalog needs at least one service.";
    return;
  }
  saving.value = true;
  try {
    await updateWorkspaceSettings({ services: items.value });
    showToast("Service catalog saved — booking forms use it immediately", "success");
  } catch (e) {
    error.value =
      e instanceof ApiError ? `API error ${e.status}: ${e.message}` : "Save failed";
  } finally {
    saving.value = false;
  }
}

const input =
  "rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500 disabled:bg-gray-100 disabled:text-gray-500";
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-white p-6">
    <h2 class="text-base font-semibold text-gray-900">Service catalog</h2>
    <p class="mt-1 text-sm text-gray-500">
      Services offered, with default durations and base prices. The New Booking
      form lists active ones — no deploy needed for price changes.
    </p>
    <div v-if="loading" class="mt-4 text-sm text-gray-500">Loading…</div>
    <template v-else>
      <div class="mt-4 overflow-x-auto">
        <table class="w-full min-w-[640px] text-left text-sm">
          <thead>
            <tr class="border-b border-gray-100 text-xs uppercase tracking-wide text-gray-400">
              <th class="py-2 pr-2 font-medium">ID</th>
              <th class="py-2 pr-2 font-medium">Name</th>
              <th class="py-2 pr-2 font-medium">Duration (min)</th>
              <th class="py-2 pr-2 font-medium">Base price</th>
              <th class="py-2 pr-2 text-center font-medium">Active</th>
              <th v-if="editable" class="py-2 text-right font-medium">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-navy-100">
            <tr v-for="s in items" :key="s.id">
              <td class="py-2 pr-2 font-mono text-xs text-gray-500">{{ s.id }}</td>
              <td class="py-2 pr-2">
                <input v-model="s.name" :class="input + ' w-full'" :disabled="!editable" />
              </td>
              <td class="py-2 pr-2">
                <input v-model.number="s.durationMinutes" type="number" min="15" max="1440" :class="input + ' w-24'" :disabled="!editable" />
              </td>
              <td class="py-2 pr-2">
                <input v-model.number="s.basePrice" type="number" min="0" step="0.01" :class="input + ' w-28'" :disabled="!editable" />
                <span class="ml-1 text-xs text-gray-400">{{ formatMoney(s.basePrice) }}</span>
              </td>
              <td class="py-2 pr-2 text-center">
                <input v-model="s.active" type="checkbox" class="h-4 w-4 accent-navy-600" :disabled="!editable" />
              </td>
              <td v-if="editable" class="py-2 text-right">
                <button
                  type="button"
                  class="rounded-lg border border-red-200 px-3 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50"
                  @click="removeItem(s.id)"
                >
                  Remove
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="editable" class="mt-4 rounded-lg bg-gray-50 p-4">
        <p class="text-xs font-semibold uppercase tracking-wide text-gray-500">Add service</p>
        <div class="mt-2 grid grid-cols-1 gap-2 sm:grid-cols-5">
          <input v-model="newName" placeholder="Display name" :class="input" />
          <input v-model="newId" placeholder="id (auto from name)" :class="input" />
          <input v-model.number="newDuration" type="number" min="15" placeholder="Minutes" :class="input" />
          <input v-model.number="newPrice" type="number" min="0" placeholder="Base price" :class="input" />
          <button
            type="button"
            class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700"
            @click="addItem"
          >
            Add
          </button>
        </div>
      </div>

      <p v-if="error" class="mt-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{{ error }}</p>
      <div class="mt-4">
        <button
          v-if="editable"
          type="button"
          :disabled="saving"
          class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50"
          @click="save"
        >
          {{ saving ? "Saving…" : "Save catalog" }}
        </button>
        <p v-else class="text-xs text-gray-400">Restricted to admins.</p>
      </div>
    </template>
  </div>
</template>
