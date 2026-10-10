<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  deleteFeedback,
  getFeedback,
  type Feedback,
} from "../../../lib/feedback";
import { showToast } from "../../../lib/toast";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";
import ConfirmDialog from "../ui/ConfirmDialog.vue";

const role = getSessionUser()?.role;
const canView = computed(() => hasPermission(role, "feedback.read"));
const canDelete = computed(() => hasPermission(role, "feedback.delete"));

const items = ref<Feedback[]>([]);
const loading = ref(true);
const forbidden = ref(false);
const search = ref("");
const page = ref(1);
const pageSize = 10;
const pendingDelete = ref<Feedback | null>(null);
const deleting = ref(false);

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase();
  if (!q) return items.value;
  return items.value.filter(
    (f) =>
      f.customerName.toLowerCase().includes(q) ||
      f.bookingNumber.toLowerCase().includes(q) ||
      f.comment.toLowerCase().includes(q),
  );
});

const totalPages = computed(() =>
  Math.max(1, Math.ceil(filtered.value.length / pageSize)),
);

const paged = computed(() => {
  const start = (page.value - 1) * pageSize;
  return filtered.value.slice(start, start + pageSize);
});

async function fetchFeedback() {
  loading.value = true;
  try {
    items.value = await getFeedback();
  } catch {
    showToast("Failed to load feedback", "error");
  } finally {
    loading.value = false;
  }
}

onMounted(fetchFeedback);

async function handleDelete(fb: Feedback) {
  deleting.value = true;
  try {
    await deleteFeedback(fb.id);
    items.value = items.value.filter((f) => f.id !== fb.id);
    showToast("Feedback deleted");
  } catch (err) {
    showToast(
      err instanceof Error ? `Failed to delete feedback: ${err.message}` : "Failed to delete feedback",
      "error",
    );
  } finally {
    deleting.value = false;
    pendingDelete.value = null;
  }
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    day: "2-digit",
    month: "short",
    year: "numeric",
  });
}

function stars(fb: Feedback): string {
  return "★".repeat(fb.rating) + "☆".repeat(5 - fb.rating);
}
</script>

<template>
  <div v-if="!canView" class="rounded-xl bg-white p-16 text-center shadow-sm">
    <p class="text-sm font-medium text-gray-900">
      You do not have permission to view feedback.
    </p>
  </div>

  <div v-else class="rounded-xl bg-white shadow-sm">
    <div
      class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-6 py-4"
    >
      <div>
        <h2 class="text-base font-semibold text-gray-900">Customer Feedback</h2>
        <p class="text-sm text-gray-500">Ratings and comments from your customers</p>
      </div>
      <input
        v-model="search"
        type="text"
        placeholder="Search feedback…"
        class="w-56 rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
      />
    </div>

    <div v-if="loading" class="space-y-3 px-6 py-8">
      <div v-for="i in 3" :key="i" class="h-20 animate-pulse rounded-lg bg-navy-50" />
    </div>

    <div v-else-if="filtered.length === 0" class="px-6 py-16 text-center">
      <p class="text-sm font-medium text-gray-900">No feedback yet</p>
      <p class="mt-1 text-sm text-gray-500">
        Feedback submitted through the customer portal appears here.
      </p>
    </div>

    <div v-else class="divide-y divide-gray-100">
      <div
        v-for="fb in paged"
        :key="fb.id"
        class="flex flex-wrap items-start gap-4 px-6 py-4"
      >
        <div class="min-w-0 flex-1">
          <div class="flex flex-wrap items-center gap-2">
            <span class="font-medium text-gray-900">{{ fb.customerName }}</span>
            <span class="font-mono text-xs text-gray-400">{{ fb.bookingNumber }}</span>
            <span class="text-sm text-amber-500" :title="`${fb.rating} / 5`">
              {{ stars(fb) }}
            </span>
          </div>
          <p v-if="fb.comment" class="mt-1 text-sm text-gray-600">{{ fb.comment }}</p>
          <p class="mt-1 text-xs text-gray-400">{{ formatDate(fb.createdAt) }}</p>
        </div>
        <button
          v-if="canDelete"
          type="button"
          class="rounded-lg px-2 py-1 text-sm font-medium text-red-600 hover:bg-red-50"
          @click="pendingDelete = fb"
        >
          Delete
        </button>
      </div>
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

    <ConfirmDialog
      :open="pendingDelete !== null"
      title="Delete feedback?"
      :busy="deleting"
      @confirm="pendingDelete && handleDelete(pendingDelete)"
      @cancel="pendingDelete = null"
    >
      <p class="text-sm text-gray-500">
        This permanently removes this feedback entry.
      </p>
    </ConfirmDialog>
  </div>
</template>