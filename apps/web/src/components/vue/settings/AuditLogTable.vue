<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ApiError } from "../../../lib/api";
import { clearSession } from "../../../lib/auth";
import { getAuditLogs, type AuditEntry } from "../../../lib/audit";
import { showToast } from "../../../lib/toast";

const entries = ref<AuditEntry[]>([]);
const loading = ref(true);
const search = ref("");
const forbidden = ref(false);
const authExpired = ref(false);
const loadError = ref<string | null>(null);

async function fetchLogs() {
  loading.value = true;
  forbidden.value = false;
  authExpired.value = false;
  loadError.value = null;
  try {
    entries.value = await getAuditLogs({ limit: 50 });
  } catch (e) {
    if (e instanceof ApiError && e.status === 403) {
      forbidden.value = true;
    } else if (e instanceof ApiError && e.status === 401) {
      clearSession();
      authExpired.value = true;
    } else {
      loadError.value =
        e instanceof Error ? e.message : "Failed to load audit log";
      showToast(loadError.value, "error");
    }
  } finally {
    loading.value = false;
  }
}

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase();
  if (!q) return entries.value;
  return entries.value.filter((e) =>
    `${e.action} ${e.resource} ${e.userEmail} ${e.userName}`
      .toLowerCase()
      .includes(q),
  );
});

function formatTime(iso: string): string {
  try {
    return new Date(iso).toLocaleString(undefined, {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return iso;
  }
}

function signInAgain() {
  clearSession();
  window.location.href = "/login?next=/settings";
}

onMounted(fetchLogs);
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-white">
    <div
      class="flex flex-col gap-3 border-b border-gray-200 px-6 py-4 sm:flex-row sm:items-center"
    >
      <div>
        <h2 class="text-base font-semibold text-gray-900">Audit log</h2>
        <p class="text-xs text-gray-500">
          Every create / update / delete, who did it, and when.
        </p>
      </div>
      <div class="sm:ml-auto flex gap-2">
        <input
          v-model="search"
          type="search"
          placeholder="Filter action, path, user..."
          class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500 sm:w-56"
        />
        <button
          type="button"
          class="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100"
          @click="fetchLogs"
        >
          Refresh
        </button>
      </div>
    </div>

    <div v-if="loading" class="px-6 py-12 text-center">
      <p class="text-sm text-gray-500">Loading audit log...</p>
    </div>
    <div v-else-if="forbidden" class="px-6 py-12 text-center">
      <p class="text-sm font-medium text-gray-900">Restricted to admins</p>
      <p class="mt-1 text-sm text-gray-500">
        Only ADMIN and SUPER_ADMIN can view the audit log.
      </p>
    </div>
    <div v-else-if="authExpired" class="px-6 py-12 text-center">
      <p class="text-sm font-medium text-gray-900">Session expired</p>
      <button
        type="button"
        class="mt-3 rounded-lg bg-navy-600 px-4 py-2 text-sm font-semibold text-white hover:bg-navy-700"
        @click="signInAgain"
      >
        Sign in again
      </button>
    </div>
    <div v-else-if="loadError" class="px-6 py-12 text-center">
      <p class="text-sm text-red-600">{{ loadError }}</p>
      <button
        type="button"
        class="mt-3 rounded-lg bg-red-600 px-4 py-2 text-sm font-semibold text-white hover:bg-red-700"
        @click="fetchLogs"
      >
        Retry
      </button>
    </div>
    <div v-else-if="filtered.length === 0" class="px-6 py-12 text-center">
      <p class="text-sm font-medium text-gray-900">No audit entries yet</p>
      <p class="mt-1 text-sm text-gray-500">
        Entries appear here after creates, updates, or deletes.
      </p>
    </div>

    <ul v-else class="divide-y divide-navy-100">
      <li
        v-for="e in filtered"
        :key="e.id"
        class="flex flex-wrap items-center gap-x-3 gap-y-1 px-6 py-3 text-sm"
      >
        <span
          class="rounded-md px-2 py-0.5 font-mono text-xs font-semibold"
          :class="
            e.action === 'DELETE'
              ? 'bg-red-50 text-red-700'
              : e.action === 'POST'
                ? 'bg-green-50 text-green-700'
                : 'bg-amber-50 text-amber-700'
          "
          >{{ e.action }}</span
        >
        <code class="truncate font-mono text-xs text-gray-600">{{
          e.resource
        }}</code>
        <span class="ml-auto text-xs text-gray-500">
          {{ e.userEmail || "system" }} · {{ formatTime(e.createdAt) }}
        </span>
      </li>
    </ul>
  </div>
</template>
