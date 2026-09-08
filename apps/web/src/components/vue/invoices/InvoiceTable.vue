<script setup lang="ts">
import { onMounted, ref, computed } from "vue";
import {
  getInvoices,
  createInvoice,
  updateInvoice,
  type Invoice,
  type InvoiceStatus,
  type CreateInvoiceInput,
} from "../../../lib/invoices";
import { getBookings, type Booking } from "../../../lib/bookings";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";
import { ApiError } from "../../../lib/api";
import ConfirmDialog from "../ui/ConfirmDialog.vue";

const canCreate = hasPermission(getSessionUser()?.role ?? "viewer", "invoices.create");
const canUpdate = hasPermission(getSessionUser()?.role ?? "viewer", "invoices.update");

const invoices = ref<Invoice[]>([]);
const loading = ref(true);
const error = ref("");
const search = ref("");
const status = ref("");

const showCreate = ref(false);
const createBusy = ref(false);
const createError = ref("");
const selectedBooking = ref<Booking | null>(null);
const completedBookings = ref<Booking[]>([]);
const subtotal = ref<string>("");

const statusMeta: Record<InvoiceStatus, { label: string; cls: string }> = {
  draft: { label: "Draft", cls: "bg-gray-100 text-gray-700" },
  issued: { label: "Issued", cls: "bg-blue-100 text-blue-700" },
  paid: { label: "Paid", cls: "bg-green-100 text-green-700" },
  void: { label: "Void", cls: "bg-red-100 text-red-700" },
};

async function load() {
  loading.value = true;
  error.value = "";
  try {
    invoices.value = await getInvoices({
      search: search.value || undefined,
      status: status.value || undefined,
    });
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : "Failed to load invoices";
  } finally {
    loading.value = false;
  }
}

function money(v: number, currency: string): string {
  return `${currency} ${v.toFixed(2)}`;
}

function fmtDate(iso: string): string {
  return new Date(iso).toLocaleDateString();
}

function openCreate() {
  createError.value = "";
  subtotal.value = "";
  selectedBooking.value = null;
  showCreate.value = true;
}

async function loadCompleted() {
  try {
    const all = await getBookings({ limit: 200 });
    completedBookings.value = all.filter((b) => b.status === "completed");
  } catch (e) {
    createError.value =
      e instanceof ApiError ? e.message : "Failed to load completed bookings";
  }
}

async function submitCreate() {
  if (!selectedBooking.value) {
    createError.value = "Select a completed booking";
    return;
  }
  createBusy.value = true;
  createError.value = "";
  try {
    const input: CreateInvoiceInput = { bookingId: selectedBooking.value.id };
    const parsed = parseFloat(subtotal.value);
    if (subtotal.value.trim() !== "" && !isNaN(parsed) && parsed >= 0) {
      input.subtotal = parsed;
    }
    const created = await createInvoice(input);
    showCreate.value = false;
    invoices.value = [created, ...invoices.value.filter((i) => i.id !== created.id)];
  } catch (e) {
    createError.value = e instanceof ApiError ? e.message : "Failed to create invoice";
  } finally {
    createBusy.value = false;
  }
}

async function setStatus(inv: Invoice, next: InvoiceStatus) {
  if (!canUpdate) return;
  error.value = "";
  try {
    const updated = await updateInvoice(inv.id, { status: next });
    const idx = invoices.value.findIndex((i) => i.id === updated.id);
    if (idx !== -1) invoices.value[idx] = updated;
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : "Failed to update invoice";
  }
}

onMounted(() => {
  load();
  if (canCreate) loadCompleted();
});
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex flex-wrap items-center gap-2">
        <input
          v-model="search"
          type="search"
          placeholder="Search invoices…"
          class="rounded-lg border border-gray-300 px-3 py-2 text-sm"
          @keyup.enter="load"
        />
        <select
          v-model="status"
          class="rounded-lg border border-gray-300 px-3 py-2 text-sm"
          @change="load"
        >
          <option value="">All statuses</option>
          <option value="draft">Draft</option>
          <option value="issued">Issued</option>
          <option value="paid">Paid</option>
          <option value="void">Void</option>
        </select>
        <button
          class="rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50"
          @click="load"
        >
          Refresh
        </button>
      </div>
      <button
        v-if="canCreate"
        class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700"
        @click="openCreate"
      >
        New Invoice
      </button>
    </div>

    <p v-if="error" class="rounded-lg bg-red-50 px-4 py-2 text-sm text-red-700">
      {{ error }}
    </p>

    <div class="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
      <table class="min-w-full divide-y divide-gray-200 text-sm">
        <thead class="bg-gray-50 text-left text-xs font-medium uppercase text-gray-500">
          <tr>
            <th class="px-4 py-3">Invoice</th>
            <th class="px-4 py-3">Customer</th>
            <th class="px-4 py-3">Booking</th>
            <th class="px-4 py-3">Date</th>
            <th class="px-4 py-3 text-right">Total</th>
            <th class="px-4 py-3">Status</th>
            <th class="px-4 py-3 text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-if="loading">
            <td colspan="7" class="px-4 py-8 text-center text-gray-400">Loading…</td>
          </tr>
          <tr v-else-if="invoices.length === 0">
            <td colspan="7" class="px-4 py-8 text-center text-gray-400">
              No invoices yet.
            </td>
          </tr>
          <tr
            v-for="inv in invoices"
            :key="inv.id"
            class="hover:bg-gray-50"
          >
            <td class="px-4 py-3">
              <a
                :href="`/invoices/${inv.id}`"
                class="font-medium text-navy-700 hover:underline"
              >{{ inv.invoiceNumber }}</a>
            </td>
            <td class="px-4 py-3">{{ inv.customerName }}</td>
            <td class="px-4 py-3 text-gray-500">{{ inv.bookingNumber }}</td>
            <td class="px-4 py-3 text-gray-500">{{ fmtDate(inv.issuedAt) }}</td>
            <td class="px-4 py-3 text-right font-medium">
              {{ money(inv.total, inv.currency) }}
            </td>
            <td class="px-4 py-3">
              <span
                :class="[
                  'inline-flex rounded-full px-2 py-1 text-xs font-medium',
                  statusMeta[inv.status].cls,
                ]"
              >{{ statusMeta[inv.status].label }}</span>
            </td>
            <td class="px-4 py-3 text-right">
              <div class="flex justify-end gap-2">
                <a
                  :href="`/invoices/${inv.id}`"
                  class="text-navy-700 hover:underline"
                >View</a>
                <button
                  v-if="canUpdate && inv.status === 'issued'"
                  class="text-green-700 hover:underline"
                  @click="setStatus(inv, 'paid')"
                >Mark Paid</button>
                <button
                  v-if="canUpdate && ['issued', 'draft', 'paid'].includes(inv.status)"
                  class="text-red-700 hover:underline"
                  @click="setStatus(inv, 'void')"
                >Void</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <ConfirmDialog
      v-if="showCreate"
      :title="'Create Invoice'"
      :message="'Generate an invoice from a completed booking.'"
      :confirm-label="'Create'"
      :busy="createBusy"
      @confirm="submitCreate"
      @cancel="showCreate = false"
    >
      <div class="mt-3 space-y-3">
        <p v-if="createError" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700">
          {{ createError }}
        </p>
        <p
          v-else-if="completedBookings.length === 0"
          class="rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-700"
        >
          No completed bookings yet. Mark a booking as completed from the
          Bookings page, then return here to create its invoice.
        </p>
        <select
          v-model="selectedBooking"
          class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
        >
          <option :value="null" disabled>Select a completed booking</option>
          <option
            v-for="b in completedBookings"
            :key="b.id"
            :value="b"
          >{{ b.bookingNumber }} — {{ b.customerName }}</option>
        </select>
        <label
          class="block text-xs text-gray-500"
        >Subtotal (empty = catalog price)</label>
        <input
          v-model="subtotal"
          type="number"
          step="0.01"
          min="0"
          placeholder="Optional"
          class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
        />
        <p v-if="selectedBooking" class="text-xs text-gray-500">
          {{ selectedBooking.serviceType }} at {{ fmtDate(selectedBooking.scheduledFor) }}
        </p>
      </div>
    </ConfirmDialog>
  </div>
</template>