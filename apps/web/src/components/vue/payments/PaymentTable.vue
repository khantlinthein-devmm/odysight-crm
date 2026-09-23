<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  createPayment,
  getPayments,
  updatePayment,
  type CreatePaymentInput,
  type Payment,
  type PaymentStatus,
} from "../../../lib/payments";
import { showToast } from "../../../lib/toast";
import {
  currencyCode,
  getWorkspaceSettings,
  paymentMethodLabel,
} from "../../../lib/settings";
import PaymentForm from "./PaymentForm.vue";

const statusLabels: Record<PaymentStatus, string> = {
  pending: "Pending",
  paid: "Paid",
  failed: "Failed",
  refunded: "Refunded",
};

const statusStyles: Record<PaymentStatus, string> = {
  pending: "bg-amber-50 text-amber-700",
  paid: "bg-green-50 text-green-700",
  failed: "bg-red-50 text-red-700",
  refunded: "bg-gray-100 text-gray-600",
};

const payments = ref<Payment[]>([]);
const loading = ref(true);
const search = ref("");
const statusFilter = ref<PaymentStatus | "">("");
const page = ref(1);
const pageSize = 10;

const showForm = ref(false);
const saving = ref(false);

const methodLabels: Record<string, string> = {
  cash: "Cash",
  bank_transfer: "Bank Transfer",
  promptpay: "PromptPay",
  credit_card: "Credit Card",
  line_pay: "LINE Pay",
  online_wallet: "Online Wallet",
};

function labelForMethod(id: string): string {
  const live = paymentMethodLabel(id);
  if (live !== id) return live;
  return methodLabels[id] ?? id;
}

const filteredPayments = computed(() => {
  const query = search.value.trim().toLowerCase();
  return payments.value.filter((payment) => {
    const matchesStatus =
      !statusFilter.value || payment.status === statusFilter.value;
    const matchesQuery =
      !query ||
      payment.customerName.toLowerCase().includes(query) ||
      payment.invoiceNumber.toLowerCase().includes(query);
    return matchesStatus && matchesQuery;
  });
});

const totalRevenue = computed(() =>
  filteredPayments.value
    .filter((p) => p.status === "paid")
    .reduce((sum, p) => sum + p.amount, 0),
);

const totalPages = computed(() =>
  Math.max(1, Math.ceil(filteredPayments.value.length / pageSize)),
);

const pagedPayments = computed(() => {
  const start = (page.value - 1) * pageSize;
  return filteredPayments.value.slice(start, start + pageSize);
});

async function fetchPayments() {
  loading.value = true;
  try {
    // Warm the workspace settings cache for currency + method labels.
    await getWorkspaceSettings().catch(() => null);
    payments.value = await getPayments();
  } catch {
    showToast("Failed to load payments", "error");
  } finally {
    loading.value = false;
  }
}

function closeForm() {
  showForm.value = false;
}

async function handleSave(input: CreatePaymentInput) {
  saving.value = true;
  try {
    const created = await createPayment(input);
    payments.value = [created, ...payments.value];
    showToast("Payment created", "success");
    closeForm();
  } catch {
    showToast("Failed to create payment", "error");
  } finally {
    saving.value = false;
  }
}

async function markAsPaid(payment: Payment) {
  try {
    const updated = await updatePayment(payment.id, { status: "paid" });
    payments.value = payments.value.map((p) =>
      p.id === updated.id ? updated : p,
    );
    showToast(`Invoice ${updated.invoiceNumber} marked as paid`, "success");
  } catch {
    showToast("Failed to update payment", "error");
  }
}

function formatMoney(amount: number, currency: string): string {
  try {
    return new Intl.NumberFormat(undefined, {
      style: "currency",
      currency,
    }).format(amount);
  } catch {
    return `${currency} ${amount.toLocaleString()}`;
  }
}

function workspaceTotal(amount: number): string {
  return formatMoney(amount, currencyCode());
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

onMounted(fetchPayments);
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-white">
    <div
      class="flex flex-col gap-3 border-b border-gray-200 px-6 py-4 sm:flex-row sm:items-center"
    >
      <h2 class="text-base font-semibold text-gray-900">Payments</h2>
      <span
        class="rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-600"
      >
        {{ filteredPayments.length }}
      </span>
      <span class="text-sm text-gray-500">
        Collected:
        <strong class="font-semibold text-green-600">{{
          workspaceTotal(totalRevenue)
        }}</strong>
      </span>

      <div class="sm:ml-auto flex flex-col gap-2 sm:flex-row">
        <input
          v-model="search"
          @input="page = 1"
          type="search"
          placeholder="Search invoice or customer..."
          class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500 sm:w-56"
        />
        <select
          v-model="statusFilter"
          @change="page = 1"
          class="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
        >
          <option value="">All statuses</option>
          <option
            v-for="(label, value) in statusLabels"
            :key="value"
            :value="value"
          >
            {{ label }}
          </option>
        </select>
        <button
          type="button"
          :disabled="saving"
          class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50"
          @click="showForm = true"
        >
          New Payment
        </button>
      </div>
    </div>

    <div v-if="loading" class="px-6 py-16 text-center">
      <p class="text-sm text-gray-500">Loading payments...</p>
    </div>

    <div
      v-else-if="filteredPayments.length === 0"
      class="px-6 py-16 text-center"
    >
      <p class="text-sm font-medium text-gray-900">No payments found</p>
      <p class="mt-1 text-sm text-gray-500">
        Try adjusting your filters or record a new payment.
      </p>
    </div>

    <div v-else class="overflow-x-auto">
      <table class="w-full text-left text-sm">
        <thead>
          <tr
            class="border-b border-gray-200 text-xs uppercase tracking-wide text-gray-500"
          >
            <th class="px-6 py-3 font-medium">Invoice #</th>
            <th class="px-6 py-3 font-medium">Customer</th>
            <th class="px-6 py-3 font-medium">Booking #</th>
            <th class="px-6 py-3 font-medium">Amount</th>
            <th class="px-6 py-3 font-medium">Method</th>
            <th class="px-6 py-3 font-medium">Status</th>
            <th class="px-6 py-3 font-medium">Date</th>
            <th class="px-6 py-3 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-navy-100">
          <tr
            v-for="payment in pagedPayments"
            :key="payment.id"
            class="hover:bg-gray-100"
          >
            <td class="px-6 py-4 font-mono text-xs text-gray-600">
              {{ payment.invoiceNumber }}
            </td>
            <td class="px-6 py-4 font-medium text-gray-900">
              {{ payment.customerName }}
            </td>
            <td class="px-6 py-4 font-mono text-xs text-gray-500">
              {{ payment.bookingNumber || "—" }}
            </td>
            <td class="px-6 py-4 font-semibold text-gray-900">
              {{ formatMoney(payment.amount, payment.currency) }}
            </td>
            <td class="px-6 py-4 text-gray-600">
              {{ labelForMethod(payment.method) }}
            </td>
            <td class="px-6 py-4">
              <span
                :class="[
                  'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
                  statusStyles[payment.status],
                ]"
              >
                {{ statusLabels[payment.status] }}
              </span>
            </td>
            <td class="px-6 py-4 text-gray-600">
              {{ formatDate(payment.createdAt) }}
            </td>
            <td class="px-6 py-4 text-right whitespace-nowrap">
              <button
                v-if="payment.status === 'pending'"
                type="button"
                class="rounded-lg px-2 py-1 text-sm font-medium text-green-600 hover:bg-green-50"
                @click="markAsPaid(payment)"
              >
                Mark as paid
              </button>
              <span v-else class="px-2 py-1 text-sm text-gray-400">—</span>
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
          class="rounded-lg border border-gray-200 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-100 disabled:opacity-40 disabled:hover:bg-transparent"
          @click="page--"
        >
          Previous
        </button>
        <button
          type="button"
          :disabled="page >= totalPages"
          class="rounded-lg border border-gray-200 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-100 disabled:opacity-40 disabled:hover:bg-transparent"
          @click="page++"
        >
          Next
        </button>
      </div>
    </div>
  </div>

  <PaymentForm v-if="showForm" @save="handleSave" @cancel="closeForm" />
</template>
