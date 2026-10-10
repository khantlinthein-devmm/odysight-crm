<script setup lang="ts">
import { onMounted, ref } from "vue";
import { DOCUMENT_TYPE_LABELS, getExpiringDocuments, type ExpiringDocument } from "../../../lib/cleaner-documents";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";

// Office-wide "renew soon" list on the Cleaners page. Stays hidden for roles
// without document access, when nothing is due, or when storage is disabled.
const canRead = hasPermission(getSessionUser()?.role, "cleaner_documents.read");
const items = ref<ExpiringDocument[]>([]);

function dueLabel(d: ExpiringDocument): string {
  if (d.daysToExpiry < 0) return `expired ${Math.abs(d.daysToExpiry)}d ago`;
  if (d.daysToExpiry === 0) return "expires today";
  return `in ${d.daysToExpiry}d`;
}

onMounted(async () => {
  if (!canRead) return;
  try {
    items.value = await getExpiringDocuments(60);
  } catch {
    items.value = [];
  }
});
</script>

<template>
  <div v-if="items.length > 0" class="mx-6 mt-6 rounded-lg border border-amber-200 bg-amber-50 px-5 py-4">
    <p class="text-sm font-semibold text-amber-800">
      {{ items.length }} document{{ items.length === 1 ? "" : "s" }} expiring within 60 days
    </p>
    <ul class="mt-2 space-y-1 text-sm">
      <li v-for="d in items" :key="d.documentId">
        <a :href="`/cleaners/${d.cleanerId}`" class="font-medium text-gray-900 hover:underline">{{ d.cleanerName }}</a>
        <span class="text-gray-600"> · {{ DOCUMENT_TYPE_LABELS[d.type] }} · </span>
        <span :class="d.daysToExpiry < 0 ? 'font-medium text-red-700' : 'text-amber-800'">{{ dueLabel(d) }}</span>
      </li>
    </ul>
  </div>
</template>
