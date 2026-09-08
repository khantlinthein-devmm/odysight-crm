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
  active: "bg-green-50 text-green-700",
  inactive: "bg-gray-100 text-gray-600",
  blocked: "bg-red-50 text-red-700",
};

const customers = ref<Customer[]>([]);
const loading = ref(true);
const search = ref("");
const statusFilter = ref<CustomerStatus | "">("");
const page = ref(1);
const pageSize = 5;

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
  } catch {
    showToast("Failed to delete customer", "error");
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
  <div class="rounded-xl border border-gray-200 bg-white">
    <div
      class="flex flex-col gap-3 border-b border-gray-200 px-6 py-4 sm:flex-row sm:items-center"
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

    <div v-else class="overflow-x-auto">
      <table class="w-full text-left text-sm">
        <thead>
          <tr
            class="border-b border-gray-200 text-xs uppercase tracking-wide text-gray-500"
          >
            <th class="px-6 py-3 font-medium">Name</th>
            <th class="px-6 py-3 font-medium">Email</th>
            <th class="px-6 py-3 font-medium">Phone</th>
            <th class="px-6 py-3 font-medium">Property</th>
            <th class="px-6 py-3 font-medium">Area</th>
            <th class="px-6 py-3 font-medium">Status</th>
            <th class="px-6 py-3 font-medium">Portal</th>
            <th class="px-6 py-3 font-medium">Created</th>
            <th class="px-6 py-3 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-navy-100">
          <tr
            v-for="customer in pagedCustomers"
            :key="customer.id"
            class="hover:bg-gray-100"
          >
            <td class="px-6 py-4 font-medium text-gray-900">
              {{ customer.firstName }} {{ customer.lastName }}
            </td>
            <td class="px-6 py-4 text-gray-600">{{ customer.email }}</td>
            <td class="px-6 py-4 text-gray-600">{{ customer.phone }}</td>
            <td class="px-6 py-4 text-gray-600">
              {{ customer.propertyType }}
            </td>
            <td class="px-6 py-4 text-gray-600">{{ customer.area }}</td>
            <td class="px-6 py-4">
              <span
                :class="[
                  'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
                  statusStyles[customer.status],
                ]"
              >
                {{ statusLabels[customer.status] }}
              </span>
            </td>
            <td class="px-6 py-4">
              <button
                type="button"
                class="inline-flex items-center gap-1.5 rounded-lg border border-gray-200 px-2.5 py-1 text-xs font-medium transition-colors"
                :class="
                  customer.portalEnabled
                    ? 'border-green-200 bg-green-50 text-green-700 hover:bg-green-100'
                    : 'text-gray-500 hover:bg-gray-100 hover:text-gray-700'
                "
                @click="openPortalModal(customer)"
              >
                <span
                  class="h-1.5 w-1.5 rounded-full"
                  :class="
                    customer.portalEnabled ? 'bg-green-500' : 'bg-gray-300'
                  "
                />
                {{ customer.portalEnabled ? "Enabled" : "Off" }}
              </button>
            </td>
            <td class="px-6 py-4 text-gray-600">
              {{ formatDate(customer.createdAt) }}
            </td>
            <td class="px-6 py-4 text-right whitespace-nowrap">
              <button
                type="button"
                class="rounded-lg px-2 py-1 text-sm font-medium text-gray-600 hover:bg-gray-100 hover:text-gray-900"
                @click="openEdit(customer)"
              >
                Edit
              </button>
              <button
                type="button"
                class="rounded-lg px-2 py-1 text-sm font-medium text-red-600 hover:bg-red-50"
                @click="pendingDelete = customer"
              >
                Delete
              </button>
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

  <CustomerForm
    v-if="showForm"
    :key="editingCustomer?.id ?? 'new'"
    :customer="editingCustomer"
    @save="handleSave"
    @cancel="closeForm"
  />

  <ConfirmDialog
    v-if="pendingDelete"
    title="Delete customer"
    :message="`Are you sure you want to delete ${pendingDelete.firstName} ${pendingDelete.lastName}? This action cannot be undone.`"
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
