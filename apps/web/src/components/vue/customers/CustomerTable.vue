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
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";

const canBook = hasPermission(getSessionUser()?.role, "bookings.create");

const statusLabels: Record<CustomerStatus, string> = {
  active: "Active",
  inactive: "Inactive",
  blocked: "Blocked",
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
      customer.email.toLowerCase().includes(query) ||
      // Phone matches ignore spaces/dashes: "0812345678" finds "081-234-5678".
      (query.replace(/\D/g, "").length >= 3 &&
        customer.phone.replace(/\D/g, "").includes(query.replace(/\D/g, "")));
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

// Card cover colour: fixed per customer (by id), so a card keeps its colour.
const COVERS = [
  "from-sky-400 to-indigo-500",
  "from-emerald-400 to-teal-600",
  "from-amber-400 to-orange-500",
  "from-fuchsia-400 to-purple-600",
  "from-rose-400 to-pink-600",
  "from-cyan-400 to-blue-600",
];

function coverFor(customer: Customer): string {
  if (customer.status === "blocked") return "from-gray-400 to-gray-600";
  return COVERS[customer.id % COVERS.length]!;
}

const customerStatusText: Record<CustomerStatus, string> = {
  active: "text-green-700",
  inactive: "text-gray-600",
  blocked: "text-red-700",
};

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
          placeholder="Search name, phone or email..."
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
        class="group relative flex flex-col overflow-hidden rounded-3xl bg-white shadow-sm ring-1 ring-gray-200 transition duration-200 hover:-translate-y-0.5 hover:shadow-xl"
      >
        <!-- Cover: a colour per customer so cards are easy to tell apart -->
        <div :class="['relative h-20 bg-gradient-to-br', coverFor(customer)]">
          <div class="absolute inset-0 bg-[radial-gradient(circle_at_1px_1px,rgba(255,255,255,0.35)_1px,transparent_0)] [background-size:14px_14px]"></div>
          <span class="absolute left-4 top-3 rounded-full bg-black/20 px-2 py-0.5 font-mono text-[11px] font-semibold text-white backdrop-blur-sm">
            #{{ customer.no || customer.id }}
          </span>
          <span
            :class="[
              'absolute right-4 top-3 inline-flex items-center gap-1.5 rounded-full bg-white/90 px-2 py-0.5 text-[11px] font-semibold shadow-sm backdrop-blur-sm',
              customerStatusText[customer.status],
            ]"
          >
            <span :class="['h-1.5 w-1.5 rounded-full', statusAccent[customer.status]]"></span>
            {{ statusLabels[customer.status] }}
          </span>
        </div>

        <div class="flex flex-1 flex-col px-5 pb-5">
          <div class="relative z-10 -mt-8 flex items-end justify-between gap-3">
            <span
              :class="[
                'flex h-16 w-16 shrink-0 items-center justify-center rounded-2xl bg-gradient-to-br text-xl font-bold text-white shadow-md ring-4 ring-white',
                coverFor(customer),
              ]"
            >
              {{ (customer.firstName || "?").charAt(0).toUpperCase() }}{{ (customer.lastName || "").charAt(0).toUpperCase() }}
            </span>
            <!-- Quick contact -->
            <div class="flex gap-1.5 pb-1">
              <a
                v-if="customer.phone"
                :href="`tel:${customer.phone}`"
                class="flex h-8 w-8 items-center justify-center rounded-full bg-gray-100 text-gray-600 transition hover:bg-navy-600 hover:text-white"
                :title="`Call ${customer.phone}`"
                :aria-label="`Call ${customer.firstName}`"
              >
                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" /></svg>
              </a>
              <a
                v-if="customer.email"
                :href="`mailto:${customer.email}`"
                class="flex h-8 w-8 items-center justify-center rounded-full bg-gray-100 text-gray-600 transition hover:bg-navy-600 hover:text-white"
                :title="`Email ${customer.email}`"
                :aria-label="`Email ${customer.firstName}`"
              >
                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" /></svg>
              </a>
              <span
                v-if="customer.lineLinked"
                class="flex h-8 items-center rounded-full bg-[#06C755] px-2.5 text-[11px] font-bold text-white"
                title="Booking confirmations, reminders and invoices are sent to this customer's LINE chat"
              >LINE</span>
            </div>
          </div>

          <h3 class="mt-3 truncate text-lg font-semibold text-gray-900">
            {{ customer.firstName }} {{ customer.lastName }}
          </h3>
          <p class="mt-0.5 flex flex-wrap items-center gap-x-2 text-sm text-gray-500">
            <span v-if="customer.propertyType" class="capitalize">{{ customer.propertyType }}</span>
            <span v-if="customer.propertyType && customer.area" class="text-gray-300">&bull;</span>
            <span v-if="customer.area">{{ customer.area }}</span>
          </p>

          <dl class="mt-4 grid grid-cols-1 gap-2 text-sm">
            <div class="flex items-center justify-between gap-3 rounded-xl bg-gray-50 px-3 py-2">
              <dt class="text-xs text-gray-500">Phone</dt>
              <dd class="min-w-0 truncate font-medium tabular-nums text-gray-900">{{ customer.phone || "—" }}</dd>
            </div>
            <div class="flex items-center justify-between gap-3 rounded-xl bg-gray-50 px-3 py-2">
              <dt class="text-xs text-gray-500">Email</dt>
              <dd class="min-w-0 truncate text-gray-700">{{ customer.email || "—" }}</dd>
            </div>
          </dl>

          <div class="mt-3 flex items-center justify-between text-xs text-gray-400">
            <span>Customer since {{ formatDate(customer.createdAt) }}</span>
            <button
              type="button"
              class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 font-semibold ring-1 transition-colors"
              :class="
                customer.portalEnabled
                  ? 'bg-green-50 text-green-700 ring-green-200 hover:bg-green-100'
                  : 'bg-white text-gray-500 ring-gray-200 hover:bg-gray-50'
              "
              @click="openPortalModal(customer)"
            >
              <span class="h-1.5 w-1.5 rounded-full" :class="customer.portalEnabled ? 'bg-green-500' : 'bg-gray-300'" />
              Portal {{ customer.portalEnabled ? "on" : "off" }}
            </button>
          </div>

          <div class="mt-auto flex items-center gap-2 pt-4">
            <a
              v-if="canBook"
              :href="`/bookings?newBooking=1&customer=${customer.id}`"
              class="flex-1 rounded-xl bg-navy-600 px-3 py-2 text-center text-sm font-semibold text-white shadow-sm hover:bg-navy-700"
            >
              New booking
            </a>
            <button
              type="button"
              class="rounded-xl bg-gray-100 px-3 py-2 text-sm font-semibold text-gray-700 hover:bg-gray-200"
              :class="canBook ? '' : 'flex-1'"
              @click="openEdit(customer)"
            >
              Edit
            </button>
            <button
              type="button"
              class="rounded-xl p-2 text-gray-400 transition hover:bg-red-50 hover:text-red-600"
              :aria-label="`Delete ${customer.firstName} ${customer.lastName}`.trim()"
              title="Delete"
              @click="pendingDelete = customer"
            >
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
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
