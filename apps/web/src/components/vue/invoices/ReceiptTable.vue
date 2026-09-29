<script setup lang="ts">
import { onMounted, ref } from "vue";
import {
  downloadReceiptPdf,
  getReceipts,
  sendReceiptEmail,
  type Receipt,
} from "../../../lib/invoices";
import { paymentMethodLabel } from "../../../lib/settings";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";
import { ApiError } from "../../../lib/api";
import { showToast } from "../../../lib/toast";

const canEmail = hasPermission(getSessionUser()?.role ?? "viewer", "payments.update");

const receipts = ref<Receipt[]>([]);
const loading = ref(true);
const error = ref("");
const search = ref("");
const status = ref("");

async function load() {
  loading.value = true;
  error.value = "";
  try {
    receipts.value = await getReceipts({
      search: search.value || undefined,
      status: (status.value || undefined) as Receipt["status"] | undefined,
      limit: 100,
    });
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : "Failed to load receipts";
  } finally {
    loading.value = false;
  }
}

function money(v: number, currency: string): string {
  return `${currency} ${v.toFixed(2)}`;
}

async function pdf(rc: Receipt) {
  try {
    await downloadReceiptPdf(rc);
  } catch {
    showToast("Failed to download receipt PDF", "error");
  }
}

async function email(rc: Receipt) {
  try {
    await sendReceiptEmail(rc.id);
    showToast(`Receipt ${rc.receiptNumber} emailed`, "success");
  } catch (e) {
    showToast(e instanceof ApiError ? e.message : "Failed to email receipt", "error");
  }
}

onMounted(load);
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center gap-2">
      <input
        v-model="search"
        type="search"
        placeholder="Search receipt, invoice, customer…"
        class="rounded-lg border border-gray-300 px-3 py-2 text-sm"
        @keyup.enter="load"
      />
      <select
        v-model="status"
        class="rounded-lg border border-gray-300 px-3 py-2 text-sm"
        @change="load"
      >
        <option value="">All receipts</option>
        <option value="valid">Valid</option>
        <option value="cancelled">Cancelled</option>
      </select>
      <button
        class="rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50"
        @click="load"
      >Refresh</button>
    </div>
    <p class="text-xs text-gray-500">
      A receipt is issued automatically for every payment recorded against an invoice.
      Refunding the payment cancels its receipt.
    </p>

    <p v-if="error" class="rounded-lg bg-red-50 px-4 py-2 text-sm text-red-700">{{ error }}</p>

    <div class="overflow-x-auto rounded-xl border border-gray-200 bg-white shadow-sm">
      <table class="min-w-full divide-y divide-gray-200 text-sm">
        <thead class="bg-gray-50 text-left text-xs font-semibold uppercase text-gray-500">
          <tr>
            <th class="px-4 py-3">Receipt</th>
            <th class="px-4 py-3">Customer</th>
            <th class="px-4 py-3">Invoice</th>
            <th class="px-4 py-3">Date</th>
            <th class="px-4 py-3">Method</th>
            <th class="px-4 py-3 text-right">Amount</th>
            <th class="px-4 py-3"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-if="loading">
            <td colspan="7" class="px-4 py-8 text-center text-gray-400">Loading…</td>
          </tr>
          <tr v-else-if="receipts.length === 0">
            <td colspan="7" class="px-4 py-8 text-center text-gray-400">No receipts yet.</td>
          </tr>
          <tr v-for="rc in receipts" v-else :key="rc.id" class="hover:bg-gray-50">
            <td class="px-4 py-3 font-medium text-gray-900">
              {{ rc.receiptNumber }}
              <span
                v-if="rc.status === 'cancelled'"
                class="ml-1 rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700"
              >Cancelled</span>
              <span
                v-else-if="rc.vatRegistered"
                class="ml-1 rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600"
                title="Receipt / tax invoice"
              >Tax invoice</span>
            </td>
            <td class="px-4 py-3">{{ rc.customerName }}</td>
            <td class="px-4 py-3">
              <a :href="`/invoices/${rc.invoiceId}`" class="text-navy-700 hover:underline">{{ rc.invoiceNumber }}</a>
            </td>
            <td class="px-4 py-3 text-gray-500">{{ new Date(rc.paidAt).toLocaleDateString() }}</td>
            <td class="px-4 py-3 text-gray-500">
              {{ paymentMethodLabel(rc.method) }}<span v-if="rc.reference" class="block text-xs">{{ rc.reference }}</span>
            </td>
            <td
              class="px-4 py-3 text-right font-medium"
              :class="rc.status === 'cancelled' ? 'text-gray-400 line-through' : ''"
            >{{ money(rc.amount, rc.currency) }}</td>
            <td class="px-4 py-3 text-right">
              <div class="flex justify-end gap-3">
                <button class="text-navy-700 hover:underline" @click="pdf(rc)">PDF</button>
                <button
                  v-if="canEmail && rc.status === 'valid'"
                  class="text-navy-700 hover:underline"
                  @click="email(rc)"
                >Email</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
