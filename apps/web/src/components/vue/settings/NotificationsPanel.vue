<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ApiError } from "../../../lib/api";
import {
  getWorkspaceSettings,
  updateWorkspaceSettings,
} from "../../../lib/settings";
import { showToast } from "../../../lib/toast";

import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";

const editable = computed(() =>
  hasPermission(getSessionUser()?.role, "settings.manage"),
);

const newLead = ref(true);
const bookingChange = ref(true);
const paymentFail = ref(true);
const overdueReminder = ref(true);
const overdueDays = ref(7);
const recipientsText = ref("");
const loading = ref(true);
const saving = ref(false);
const error = ref<string | null>(null);

onMounted(async () => {
  try {
    const n = (await getWorkspaceSettings()).notifications;
    newLead.value = n.newLeadEmail;
    bookingChange.value = n.bookingChangeEmail;
    paymentFail.value = n.paymentFailEmail;
    overdueReminder.value = n.overdueReminderEmail;
    overdueDays.value = n.overdueReminderDays;
    recipientsText.value = n.recipients.join("\n");
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Failed to load";
  } finally {
    loading.value = false;
  }
});

async function save() {
  error.value = null;
  const recipients = recipientsText.value
    .split("\n")
    .map((e) => e.trim().toLowerCase())
    .filter(Boolean);
  const days = Number(overdueDays.value);
  if (!Number.isInteger(days) || days < 1 || days > 365) {
    error.value = "Reminder delay must be a whole number between 1 and 365 days.";
    return;
  }
  saving.value = true;
  try {
    await updateWorkspaceSettings({
      notifications: {
        newLeadEmail: newLead.value,
        bookingChangeEmail: bookingChange.value,
        paymentFailEmail: paymentFail.value,
        overdueReminderEmail: overdueReminder.value,
        overdueReminderDays: days,
        recipients,
      },
    });
    showToast("Notification preferences saved", "success");
  } catch (e) {
    error.value =
      e instanceof ApiError ? `API error ${e.status}: ${e.message}` : "Save failed";
  } finally {
    saving.value = false;
  }
}

function row(label: string, desc: string) {
  return { label, desc };
}
const rows = [
  row("New lead", "Alert when a lead is created"),
  row("Booking change", "Alert on booking create / update / cancel"),
  row("Payment failure", "Alert when a payment fails"),
  row(
    "Overdue invoice reminder",
    "Email the customer (with the invoice PDF) when a bill stays unpaid",
  ),
];

const input =
  "w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500 disabled:bg-gray-100 disabled:text-gray-500";
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-white p-6">
    <h2 class="text-base font-semibold text-gray-900">Notifications</h2>
    <p class="mt-1 text-sm text-gray-500">
      Which events alert the team, plus automated overdue-invoice reminders to
      customers. Emails are sent through the SMTP settings on this page — no
      emails go out until a mail server is configured.
    </p>
    <div v-if="loading" class="mt-4 text-sm text-gray-500">Loading…</div>
    <div v-else class="mt-4 space-y-3">
      <label class="flex items-start gap-3 rounded-lg border border-gray-100 px-4 py-3">
        <input v-model="newLead" type="checkbox" class="mt-1 h-4 w-4 accent-navy-600" :disabled="!editable" />
        <span>
          <span class="block text-sm font-medium text-gray-900">{{ rows[0]!.label }}</span>
          <span class="block text-xs text-gray-500">{{ rows[0]!.desc }}</span>
        </span>
      </label>
      <label class="flex items-start gap-3 rounded-lg border border-gray-100 px-4 py-3">
        <input v-model="bookingChange" type="checkbox" class="mt-1 h-4 w-4 accent-navy-600" :disabled="!editable" />
        <span>
          <span class="block text-sm font-medium text-gray-900">{{ rows[1]!.label }}</span>
          <span class="block text-xs text-gray-500">{{ rows[1]!.desc }}</span>
        </span>
      </label>
      <label class="flex items-start gap-3 rounded-lg border border-gray-100 px-4 py-3">
        <input v-model="paymentFail" type="checkbox" class="mt-1 h-4 w-4 accent-navy-600" :disabled="!editable" />
        <span>
          <span class="block text-sm font-medium text-gray-900">{{ rows[2]!.label }}</span>
          <span class="block text-xs text-gray-500">{{ rows[2]!.desc }}</span>
        </span>
      </label>
      <label class="flex items-start gap-3 rounded-lg border border-gray-100 px-4 py-3">
        <input v-model="overdueReminder" type="checkbox" class="mt-1 h-4 w-4 accent-navy-600" :disabled="!editable" />
        <span class="min-w-0">
          <span class="block text-sm font-medium text-gray-900">{{ rows[3]!.label }}</span>
          <span class="block text-xs text-gray-500">{{ rows[3]!.desc }}</span>
          <span class="mt-2 flex items-center gap-2">
            <label class="text-xs font-medium text-gray-600" for="overdue-days">Remind after</label>
            <input
              id="overdue-days"
              v-model.number="overdueDays"
              type="number"
              min="1"
              max="365"
              :disabled="!editable || !overdueReminder"
              class="w-20 rounded-lg border border-gray-200 bg-white px-2 py-1.5 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500 disabled:bg-gray-100 disabled:text-gray-500"
            />
            <span class="text-xs text-gray-500">days from issue date</span>
          </span>
        </span>
      </label>
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Notify these emails (one per line)</span>
        <textarea v-model="recipientsText" rows="3" placeholder="ops@company.com" :class="input" :disabled="!editable"></textarea>
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
