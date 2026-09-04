<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  createCustomer,
  deleteCustomer,
  getCustomers,
  updateCustomer,
  type Customer,
  type CustomerStatus,
  type CreateCustomerInput,
} from "../../../lib/customers";
import { showToast } from "../../../lib/toast";
import CustomerForm from "./CustomerForm.vue";
import ConfirmDialog from "../ui/ConfirmDialog.vue";

const statusLabels: Record<CustomerStatus, string> = {
  active: "Active",
  inactive: "Inactive",
  blocked: "Blocked",
};

const statusStyles: Record<CustomerStatus, string> = {
  active: "bg-emerald-50 text-emerald-700",
  inactive: "bg-slate-100 text-slate-600",
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
      const updated = await updateCustomer(editingCustomer.value.id, input);
      customers.value = customers.value.map((c) =>
        c.id === updated.id ? updated : c,
      );
      showToast("Customer updated", "success");
    } else {
      const created = await createCustomer(input);
      customers.value = [created, ...customers.value];
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
  <div class="rounded-xl border border-slate-200 bg-white">
    <div
      class="flex flex-col gap-3 border-b border-slate-200 px-6 py-4 sm:flex-row sm:items-center"
    >
      <h2 class="text-base font-semibold text-slate-900">Customers</h2>
      <span
        class="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-600"
      >
        {{ filteredCustomers.length }}
      </span>

      <div class="sm:ml-auto flex flex-col gap-2 sm:flex-row">
        <input
          v-model="search"
          @input="page = 1"
          type="search"
          placeholder="Search name or email..."
          class="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 sm:w-56"
        />
        <select
          v-model="statusFilter"
          @change="page = 1"
          class="rounded-lg border border-slate-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
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
          class="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
          @click="openCreate"
        >
          New Customer
        </button>
      </div>
    </div>

    <div v-if="loading" class="px-6 py-16 text-center">
      <p class="text-sm text-slate-500">Loading customers...</p>
    </div>

    <div
      v-else-if="filteredCustomers.length === 0"
      class="px-6 py-16 text-center"
    >
      <p class="text-sm font-medium text-slate-900">No customers found</p>
      <p class="mt-1 text-sm text-slate-500">
        Try adjusting your filters or create a new customer.
      </p>
    </div>

    <div v-else class="overflow-x-auto">
      <table class="w-full text-left text-sm">
        <thead>
          <tr
            class="border-b border-slate-200 text-xs uppercase tracking-wide text-slate-500"
          >
            <th class="px-6 py-3 font-medium">Name</th>
            <th class="px-6 py-3 font-medium">Email</th>
            <th class="px-6 py-3 font-medium">Phone</th>
            <th class="px-6 py-3 font-medium">Property</th>
            <th class="px-6 py-3 font-medium">Area</th>
            <th class="px-6 py-3 font-medium">Status</th>
            <th class="px-6 py-3 font-medium">Created</th>
            <th class="px-6 py-3 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          <tr
            v-for="customer in pagedCustomers"
            :key="customer.id"
            class="hover:bg-slate-50"
          >
            <td class="px-6 py-4 font-medium text-slate-900">
              {{ customer.firstName }} {{ customer.lastName }}
            </td>
            <td class="px-6 py-4 text-slate-600">{{ customer.email }}</td>
            <td class="px-6 py-4 text-slate-600">{{ customer.phone }}</td>
            <td class="px-6 py-4 text-slate-600">
              {{ customer.propertyType }}
            </td>
            <td class="px-6 py-4 text-slate-600">{{ customer.area }}</td>
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
            <td class="px-6 py-4 text-slate-600">
              {{ formatDate(customer.createdAt) }}
            </td>
            <td class="px-6 py-4 text-right whitespace-nowrap">
              <button
                type="button"
                class="rounded-lg px-2 py-1 text-sm font-medium text-slate-600 hover:bg-slate-100 hover:text-slate-900"
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
      class="flex items-center justify-between border-t border-slate-200 px-6 py-3"
    >
      <p class="text-sm text-slate-500">Page {{ page }} of {{ totalPages }}</p>
      <div class="flex gap-2">
        <button
          type="button"
          :disabled="page <= 1"
          class="rounded-lg border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-40 disabled:hover:bg-transparent"
          @click="page--"
        >
          Previous
        </button>
        <button
          type="button"
          :disabled="page >= totalPages"
          class="rounded-lg border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-40 disabled:hover:bg-transparent"
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
</template>
