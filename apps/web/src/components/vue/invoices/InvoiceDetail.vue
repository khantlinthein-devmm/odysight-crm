<script setup lang="ts">
import { onMounted, ref } from "vue";
import {
  getInvoice,
  updateInvoice,
  downloadInvoicePdf,
  sendInvoiceEmail,
  type Invoice,
} from "../../../lib/invoices";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";
import { ApiError } from "../../../lib/api";
import { showToast } from "../../../lib/toast";

const props = defineProps<{ invoiceId: number }>();
const canUpdate = hasPermission(
  getSessionUser()?.role ?? "viewer",
  "invoices.update",
);

const invoice = ref<Invoice | null>(null);
const loading = ref(true);
const error = ref("");
const actionError = ref("");
const busy = ref(false);
const emailing = ref(false);

async function load() {
  loading.value = true;
  error.value = "";
  try {
    invoice.value = await getInvoice(props.invoiceId);
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : "Invoice not found";
  } finally {
    loading.value = false;
  }
}

function money(v: number, currency: string): string {
  return `${currency} ${v.toFixed(2)}`;
}

function fmtDate(iso: string | null): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleString();
}

async function setStatus(next: "paid" | "void") {
  if (!invoice.value) return;
  busy.value = true;
  actionError.value = "";
  try {
    invoice.value = await updateInvoice(invoice.value.id, { status: next });
  } catch (e) {
    actionError.value =
      e instanceof ApiError ? e.message : "Failed to update invoice";
  } finally {
    busy.value = false;
  }
}

async function sendEmail() {
  if (!invoice.value) return;
  emailing.value = true;
  actionError.value = "";
  try {
    await sendInvoiceEmail(invoice.value.id);
    showToast(`Invoice emailed to ${invoice.value.customerEmail || "customer"}`, "success");
  } catch (e) {
    actionError.value =
      e instanceof ApiError ? e.message : "Failed to send invoice email";
  } finally {
    emailing.value = false;
  }
}

async function downloadPdf() {
  if (!invoice.value) return;
  try {
    await downloadInvoicePdf(invoice.value.id, `${invoice.value.invoiceNumber}.pdf`);
  } catch {
    showToast("Failed to download invoice PDF", "error");
  }
}

onMounted(load);
</script>

<template>
  <div class="space-y-4">
    <p v-if="error" class="rounded-lg bg-red-50 px-4 py-2 text-sm text-red-700">
      {{ error }}
    </p>
    <div v-else-if="loading" class="py-8 text-center text-gray-400">
      Loading…
    </div>

    <template v-else-if="invoice">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="text-lg font-semibold text-gray-900">
            {{ invoice.invoiceNumber }}
          </h2>
          <p class="text-sm text-gray-500">
            Booking {{ invoice.bookingNumber }} · {{ invoice.status.toUpperCase() }}
          </p>
        </div>
        <div class="flex items-center gap-2">
          <button
            type="button"
            class="rounded-lg border border-gray-300 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
            @click="downloadPdf"
          >Download PDF</button>
          <button
            v-if="canUpdate && invoice.status !== 'void'"
            :disabled="emailing"
            class="rounded-lg border border-gray-300 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 disabled:opacity-50"
            @click="sendEmail"
          >{{ emailing ? "Emailing…" : "Email" }}</button>
          <button
            class="rounded-lg border border-gray-300 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
            @click="window.print()"
          >Print</button>
          <button
            v-if="canUpdate && invoice.status === 'issued'"
            :disabled="busy"
            class="rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50"
            @click="setStatus('paid')"
          >Mark Paid</button>
          <button
            v-if="canUpdate && ['issued', 'draft', 'paid'].includes(invoice.status)"
            :disabled="busy"
            class="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
            @click="setStatus('void')"
          >Void</button>
        </div>
      </div>

      <p
        v-if="actionError"
        class="rounded-lg bg-red-50 px-4 py-2 text-sm text-red-700"
      >{{ actionError }}</p>

      <div
        class="mx-auto max-w-2xl rounded-xl border border-gray-200 bg-white p-8 shadow-sm"
      >
        <div class="flex items-start justify-between">
          <div>
            <h3 class="text-sm font-semibold uppercase tracking-wide text-gray-400">
              Invoice
            </h3>
            <p class="mt-1 text-2xl font-bold text-gray-900">Smile Clean</p>
          </div>
          <div class="text-right text-sm text-gray-500">
            <p>{{ invoice.invoiceNumber }}</p>
            <p>{{ invoice.bookingNumber }}</p>
            <p>{{ fmtDate(invoice.issuedAt) }}</p>
            <p class="mt-1 inline-block rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700">
              {{ invoice.status.toUpperCase() }}
            </p>
          </div>
        </div>

        <div class="mt-8 grid grid-cols-2 gap-6">
          <div>
            <h4 class="text-xs font-semibold uppercase text-gray-400">Bill To</h4>
            <p class="mt-1 text-sm font-medium text-gray-900">{{ invoice.customerName }}</p>
            <p class="mt-1 text-sm text-gray-500">{{ invoice.address }}</p>
            <p v-if="invoice.customerEmail" class="mt-1 text-sm text-gray-500">{{ invoice.customerEmail }}</p>
          </div>
          <div class="text-right">
            <h4 class="text-xs font-semibold uppercase text-gray-400">Service</h4>
            <p class="mt-1 text-sm font-medium text-gray-900">{{ invoice.serviceName }}</p>
            <p class="text-xs text-gray-500">{{ invoice.serviceType }}</p>
          </div>
        </div>

        <div class="mt-8 border-t border-gray-200 pt-4">
          <div class="flex justify-between py-1 text-sm">
            <span class="text-gray-500">{{ invoice.serviceName }}</span>
            <span class="font-medium text-gray-900">{{ money(invoice.subtotal, invoice.currency) }}</span>
          </div>
          <div class="flex justify-between py-1 text-sm text-gray-500">
            <span>Tax ({{ invoice.taxRate.toFixed(2) }}%)</span>
            <span>{{ money(invoice.taxAmount, invoice.currency) }}</span>
          </div>
          <div class="mt-2 flex justify-between border-t border-gray-200 pt-3 text-base font-semibold text-gray-900">
            <span>Total</span>
            <span>{{ money(invoice.total, invoice.currency) }}</span>
          </div>
        </div>

        <p class="mt-10 text-center text-xs text-gray-400">
          Thank you for your business!
        </p>
      </div>
    </template>
  </div>
</template>