<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  getNotificationLog,
  type NotificationLogEntry,
} from "../../../lib/notifications";
import { showToast } from "../../../lib/toast";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";

const role = getSessionUser()?.role;
const canView = computed(() => hasPermission(role, "notifications.read"));

const entries = ref<NotificationLogEntry[]>([]);
const loading = ref(true);
const search = ref("");
const statusFilter = ref<NotificationLogEntry["status"] | "">("");
const page = ref(1);
const pageSize = 10;

const channelStyles: Record<string, string> = {
  email: "bg-navy-50 text-navy-700",
  sms: "bg-purple-50 text-purple-700",
  whatsapp: "bg-green-50 text-green-700",
};

const statusStyles: Record<string, string> = {
  sent: "bg-green-50 text-green-700",
  failed: "bg-red-50 text-red-700",
  skipped: "bg-gray-100 text-gray-600",
};

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase();
  return entries.value.filter((e) => {
    if (statusFilter.value && e.status !== statusFilter.value) return false;
    if (!q) return true;
    return (
      e.recipient.toLowerCase().includes(q) ||
      e.subject.toLowerCase().includes(q) ||
      e.eventType.toLowerCase().includes(q)
    );
  });
});

const totalPages = computed(() =>
  Math.max(1, Math.ceil(filtered.value.length / pageSize)),
);

const paged = computed(() => {
  const start = (page.value - 1) * pageSize;
  return filtered.value.slice(start, start + pageSize);
});

async function fetchLog() {
  loading.value = true;
  try {
    entries.value = await getNotificationLog();
  } catch {
    showToast("Failed to load notification log", "error");
  } finally {
    loading.value = false;
  }
}

onMounted(fetchLog);

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function eventLabel(e: NotificationLogEntry): string {
  return e.eventType.replace(/\./g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}
</script>

<template>
  <div v-if="!canView" class="rounded-xl bg-white p-16 text-center shadow-sm">
    <p class="text-sm font-medium text-gray-900">
      You do not have permission to view notification logs.
    </p>
  </div>

  <div v-else class="rounded-xl bg-white shadow-sm">
    <div
      class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-6 py-4"
    >
      <div>
        <h2 class="text-base font-semibold text-gray-900">Notification Log</h2>
        <p class="text-sm text-gray-500">Emails, SMS and WhatsApp messages sent</p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <select
          v-model="statusFilter"
          class="rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 focus:border-gray-500 focus:outline-none"
        >
          <option value="">All statuses</option>
          <option value="sent">Sent</option>
          <option value="failed">Failed</option>
          <option value="skipped">Skipped</option>
        </select>
        <input
          v-model="search"
          type="text"
          placeholder="Search…"
          class="w-52 rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-gray-500 focus:outline-none"
        />
      </div>
    </div>

    <div v-if="loading" class="space-y-3 px-6 py-8">
      <div v-for="i in 4" :key="i" class="h-12 animate-pulse rounded-lg bg-navy-50" />
    </div>

    <div v-else-if="filtered.length === 0" class="px-6 py-16 text-center">
      <p class="text-sm font-medium text-gray-900">No notifications logged</p>
      <p class="mt-1 text-sm text-gray-500">
        Sent emails, SMS and WhatsApp messages appear here.
      </p>
    </div>

    <div v-else class="overflow-x-auto">
      <table class="w-full text-left text-sm">
        <thead>
          <tr
            class="border-b border-gray-200 text-xs uppercase tracking-wide text-gray-500"
          >
            <th class="px-6 py-3 font-medium">Channel</th>
            <th class="px-6 py-3 font-medium">Event</th>
            <th class="px-6 py-3 font-medium">Recipient</th>
            <th class="px-6 py-3 font-medium">Subject</th>
            <th class="px-6 py-3 font-medium">Status</th>
            <th class="px-6 py-3 font-medium">Sent</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-navy-100">
          <tr v-for="e in paged" :key="e.id" class="hover:bg-gray-100">
            <td class="px-6 py-3">
              <span
                :class="[
                  'inline-flex rounded-full px-2.5 py-0.5 text-xs capitalize font-medium',
                  channelStyles[e.channel] ?? 'bg-gray-100 text-gray-600',
                ]"
              >
                {{ e.channel }}
              </span>
            </td>
            <td class="px-6 py-3 text-gray-600">{{ eventLabel(e) }}</td>
            <td class="px-6 py-3 font-mono text-xs text-gray-600">
              {{ e.recipient }}
            </td>
            <td class="max-w-64 truncate px-6 py-3 text-gray-600">
              {{ e.subject }}
            </td>
            <td class="px-6 py-3">
              <span
                :class="[
                  'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize',
                  statusStyles[e.status],
                ]"
              >
                {{ e.status }}
              </span>
            </td>
            <td class="px-6 py-3 whitespace-nowrap text-gray-500">
              {{ formatTime(e.createdAt) }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div
      v-if="!loading && totalPages > 1"
      class="flex items-center justify-between border-t border-gray-200 px-6 py-3"
    >
      <p class="text-sm text-gray-500">Page {{ page }} of {{ totalPages }}</p>
      <div class="flex gap-2">
        <button
          type="button"
          :disabled="page <= 1"
          class="rounded-lg border border-gray-200 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-100 disabled:opacity-40"
          @click="page--"
        >
          Previous
        </button>
        <button
          type="button"
          :disabled="page >= totalPages"
          class="rounded-lg border border-gray-200 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-100 disabled:opacity-40"
          @click="page++"
        >
          Next
        </button>
      </div>
    </div>
  </div>
</template>