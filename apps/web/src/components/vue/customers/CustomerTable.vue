<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  createCustomer,
  deleteCustomer,
  getCustomers,
  setCustomerPortal,
  updateCustomer,
  type Customer,
  type CustomerStatus,
  type CreateCustomerInput,
} from "../../../lib/customers";
import { showToast } from "../../../lib/toast";
import { updateLead } from "../../../lib/leads";
import CustomerForm from "./CustomerForm.vue";
import ConfirmDialog from "../ui/ConfirmDialog.vue";

const statusLabels: Record<CustomerStatus, string> = {
  active: "Active",
  inactive: "Inactive",
  blocked: "Blocked",
};

const statusStyles: Record<CustomerStatus, string> = {
  active: "bg-green-100 text-green-800 ring-1 ring-green-300",
  inactive: "bg-gray-200 text-gray-700 ring-1 ring-gray-300",
  blocked: "bg-red-100 text-red-800 ring-1 ring-red-300",
};

const statusAccent: Record<CustomerStatus, string> = {
  active: "bg-green-500",
  inactive: "bg-gray-300",
  blocked: "bg-red-500",
};

const customers = ref<Customer[]>([]);
const loading = ref(true);
const search = ref("");
const statusFilter = ref<CustomerStatus | "">("");
const page = ref(1);
const pageSize = 9;

const showForm = ref(false);
const editingCustomer = ref<Customer | undefined>(undefined);
const saving = ref(false);
const pendingDelete = ref<Customer | null>(null);
const deleting = ref(false);

const portalTarget = ref<Customer | null>(null);
const portalEnabling = ref(false);
const portalPassword = ref("");
const portalBusy = ref(false);
const portalError = ref("");

function openPortalModal(customer: Customer) {
  portalTarget.value = customer;
  portalEnabling.value = !customer.portalEnabled;
  portalPassword.value = "";
  portalError.value = "";
}

async function submitPortal() {
  if (!portalTarget.value) return;
  if (portalEnabling.value && portalPassword.value.trim().length < 8) {
    portalError.value = "Password must be at least 8 characters";
    return;
  }
  portalBusy.value = true;
  portalError.value = "";
  try {
    const updated = await setCustomerPortal(
      portalTarget.value.id,
      portalEnabling.value,
      portalPassword.value,
    );
    customers.value = customers.value.map((c) =>
      c.id === updated.id ? updated : c,
    );
    showToast(
      portalEnabling.value
        ? "Portal access enabled"
        : "Portal access disabled",
      "success",
    );
    portalTarget.value = null;
  } catch (e) {
    portalError.value =
      e instanceof Error ? e.message : "Failed to update portal access";
  } finally {
    portalBusy.value = false;
  }
}

const filteredCustomers = computed(() => {
  const query = search.value.trim().toLowerCase();
  return customers.value.filter((customer) => {
    const matchesStatus =
      !statusFilter.value || customer.status === statusFilter.value;
    const matchesQuery =
      !query ||
      `${customer.firstName} ${customer.lastName}`
        .toLowerCase()
        .includes(query) ||
      customer.email.toLowerCase().includes(query);
    return matchesStatus && matchesQuery;
  });
});

const totalPages = computed(() =>
  Math.max(1, Math.ceil(filteredCustomers.value.length / pageSize)),
);

const pagedCustomers = computed(() => {
  const start = (page.value - 1) * pageSize;
  return filteredCustomers.value.slice(start, start + pageSize);
});

async function fetchCustomers() {
  loading.value = true;
  try {
    customers.value = await getCustomers();
  } catch {
    showToast("Failed to load customers", "error");
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  editingCustomer.value = undefined;
  showForm.value = true;
}

function openEdit(customer: Customer) {
  editingCustomer.value = customer;
  showForm.value = true;
}

function closeForm() {
  showForm.value = false;
  editingCustomer.value = undefined;
}

async function handleSave(input: CreateCustomerInput) {
  saving.value = true;
  try {
    if (editingCustomer.value) {
      const { leadId: _leadId, ...updateInput } = input;
      const updated = await updateCustomer(editingCustomer.value.id, updateInput);
      customers.value = customers.value.map((c) =>
        c.id === updated.id ? updated : c,
      );
      showToast("Customer updated", "success");
    } else {
      const created = await createCustomer(input);
      customers.value = [created, ...customers.value];
      if (created.leadId) {
        await updateLead(created.leadId, { status: "won" });
      }
      showToast("Customer created", "success");
    }
    closeForm();
  } catch {
    showToast("Failed to save customer", "error");
  } finally {
    saving.value = false;
  }
}

async function handleDelete() {
  if (!pendingDelete.value) return;
  deleting.value = true;
  try {
    await deleteCustomer(pendingDelete.value.id);
    customers.value = customers.value.filter(
      (c) => c.id !== pendingDelete.value!.id,
    );
    showToast("Customer deleted", "success");
  } catch (err) {
    showToast(
      err instanceof Error ? `Failed to delete customer: ${err.message}` : "Failed to delete customer",
      "error",
    );
  } finally {
    deleting.value = false;
    pendingDelete.value = null;
  }
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

onMounted(fetchCustomers);
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-gray-100/70">
    <div
      class="flex flex-col gap-3 rounded-t-xl border-b border-gray-200 bg-white px-6 py-4 sm:flex-row sm:items-center"
    >
      <h2 class="text-base font-semibold text-gray-900">Customers</h2>
      <span
        class="rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-600"
      >
        {{ filteredCustomers.length }}
      </span>

      <div class="sm:ml-auto flex flex-col gap-2 sm:flex-row">
        <input
          v-model="search"
          @input="page = 1"
          type="search"
          placeholder="Search name or email..."
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
          class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700"
          @click="openCreate"
        >
          New Customer
        </button>
      </div>
    </div>

    <div v-if="loading" class="px-6 py-16 text-center">
      <p class="text-sm text-gray-500">Loading customers...</p>
    </div>

    <div
      v-else-if="filteredCustomers.length === 0"
      class="px-6 py-16 text-center"
    >
      <p class="text-sm font-medium text-gray-900">No customers found</p>
      <p class="mt-1 text-sm text-gray-500">
        Try adjusting your filters or create a new customer.
      </p>
    </div>

    <div v-else class="grid grid-cols-1 gap-4 p-4 sm:p-6 md:grid-cols-2 xl:grid-cols-3">
      <article
        v-for="customer in pagedCustomers"
        :key="customer.id"
        class="flex flex-col overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm transition hover:shadow-md"
      >
        <div :class="['h-1.5 w-full', statusAccent[customer.status]]"></div>
        <div class="flex flex-1 flex-col p-5">
          <div class="flex items-center justify-between gap-2">
            <span class="rounded bg-gray-900 px-2 py-0.5 font-mono text-[11px] font-semibold tracking-wide text-white">
              #{{ customer.id }}
            </span>
            <span
              :class="[
                'inline-flex shrink-0 items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-bold',
                statusStyles[customer.status],
              ]"
            >
              <span :class="['h-1.5 w-1.5 rounded-full', statusAccent[customer.status]]"></span>
              {{ statusLabels[customer.status] }}
            </span>
          </div>

          <div class="mt-3 flex items-center gap-3">
            <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-navy-600 text-base font-bold text-white shadow-sm">
              {{ (customer.firstName || "?").charAt(0).toUpperCase() }}{{ (customer.lastName || "").charAt(0).toUpperCase() }}
            </span>
            <div class="min-w-0">
              <h3 class="truncate text-base font-bold text-gray-900">
                {{ customer.firstName }} {{ customer.lastName }}
              </h3>
              <p class="mt-0.5 flex flex-wrap items-center gap-1.5 text-xs">
                <span v-if="customer.propertyType" class="inline-flex items-center gap-1 rounded-md bg-slate-100 px-2 py-0.5 font-semibold text-slate-700 ring-1 ring-slate-200">
                  <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" /></svg>
                  {{ customer.propertyType }}
                </span>
                <span v-if="customer.area" class="inline-flex items-center gap-1 rounded-md bg-slate-100 px-2 py-0.5 font-semibold text-slate-700 ring-1 ring-slate-200">
                  <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M17.657 16.657L13.414 20.9a2 2 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
                  {{ customer.area }}
                </span>
              </p>
            </div>
          </div>

          <dl class="mt-4 space-y-2 rounded-lg bg-gray-50 p-3 text-sm ring-1 ring-gray-100">
            <div class="flex items-center gap-2">
              <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-blue-100 text-blue-700"><svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" /></svg></span>
              <dd class="min-w-0 truncate text-gray-700">{{ customer.email || "—" }}</dd>
            </div>
            <div class="flex items-center gap-2">
              <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-green-100 text-green-700"><svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" /></svg></span>
              <dd class="min-w-0 truncate font-medium text-gray-800">{{ customer.phone || "—" }}</dd>
            </div>
            <div class="flex items-center gap-2">
              <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-gray-100 text-gray-500"><svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg></span>
              <dd class="min-w-0 text-gray-600">{{ formatDate(customer.createdAt) }}</dd>
            </div>
          </dl>

          <div class="mt-3">
            <button
              type="button"
              class="inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1 text-xs font-semibold transition-colors"
              :class="
                customer.portalEnabled
                  ? 'border-green-300 bg-green-100 text-green-800 hover:bg-green-200'
                  : 'border-gray-200 bg-gray-100 text-gray-600 hover:bg-gray-200'
              "
              @click="openPortalModal(customer)"
            >
              <span
                class="h-1.5 w-1.5 rounded-full"
                :class="customer.portalEnabled ? 'bg-green-500' : 'bg-gray-400'"
              />
              Portal: {{ customer.portalEnabled ? "Enabled" : "Off" }}
            </button>
          </div>

          <div class="mt-4 flex flex-wrap items-center gap-2 border-t border-gray-100 pt-3">
            <button
              type="button"
              class="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-sm font-semibold text-gray-700 hover:bg-gray-100"
              @click="openEdit(customer)"
            >
              Edit
            </button>
            <button
              type="button"
              class="ml-auto rounded-lg px-2.5 py-1.5 text-sm font-medium text-red-600 hover:bg-red-50"
              @click="pendingDelete = customer"
            >
              Delete
            </button>
          </div>
        </div>
      </article>
    </div>

    <div
      v-if="!loading && totalPages > 1"
      class="flex items-center justify-between rounded-b-xl border-t border-gray-200 bg-white px-6 py-3"
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

  <CustomerForm
    v-if="showForm"
    :key="editingCustomer?.id ?? 'new'"
    :customer="editingCustomer"
    @save="handleSave"
    @cancel="closeForm"
  />

  <ConfirmDialog
    v-if="pendingDelete"
    :title="`${pendingDelete.firstName} ${pendingDelete.lastName}`.trim()"
    message="Are you sure you want to delete this customer? This action cannot be undone."
    confirm-label="Delete customer"
    :busy="deleting"
    @confirm="handleDelete"
    @cancel="pendingDelete = null"
  />

  <div
    v-if="portalTarget"
    class="fixed inset-0 z-50 flex items-center justify-center bg-navy-900/40 p-4"
    @click.self="portalTarget = null"
  >
    <div class="w-full max-w-md rounded-xl bg-white p-6 shadow-xl">
      <h3 class="text-base font-semibold text-gray-900">
        {{
          portalEnabling
            ? `Enable portal for ${portalTarget.firstName} ${portalTarget.lastName}`
            : `Disable portal for ${portalTarget.firstName} ${portalTarget.lastName}`
        }}
      </h3>
      <p class="mt-1 text-sm text-gray-500">
        {{
          portalEnabling
            ? portalTarget.portalEnabled
              ? "Portal access is already enabled. Setting a new password will replace the current one."
              : "This lets the customer sign in at /portal to view bookings."
            : "The customer will no longer be able to sign in at /portal."
        }}
      </p>

      <div v-if="portalEnabling" class="mt-4">
        <label class="mb-1 block text-sm font-medium text-gray-700">
          Portal password
          <span class="text-gray-400">(min 8 characters)</span>
        </label>
        <input
          v-model="portalPassword"
          type="password"
          placeholder="••••••••"
          class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
        />
        <p v-if="portalTarget.portalEnabled" class="mt-2 text-xs text-gray-500">
          Leave blank to keep the current password.
        </p>
      </div>

      <p
        v-if="portalError"
        class="mt-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700"
      >
        {{ portalError }}
      </p>

      <div class="mt-6 flex justify-end gap-2">
        <button
          type="button"
          class="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100"
          @click="portalTarget = null"
        >
          Cancel
        </button>
        <button
          type="button"
          :disabled="portalBusy"
          :class="
            portalEnabling
              ? 'bg-navy-600 hover:bg-navy-700'
              : 'bg-red-600 hover:bg-red-700'
          "
          class="rounded-lg px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
          @click="submitPortal"
        >
          {{ portalBusy ? "Saving…" : portalEnabling ? "Enable portal" : "Disable portal" }}
        </button>
      </div>
    </div>
  </div>
</template>
