<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { createContract, getContracts, renewContract, updateContract, type Contract } from "../../../lib/contracts";
import { getCustomers, type Customer } from "../../../lib/customers";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";
import { showToast } from "../../../lib/toast";

const role = getSessionUser()?.role;
const canEdit = computed(() => hasPermission(role, "contracts.update"));

const contracts = ref<Contract[]>([]);
const customers = ref<Customer[]>([]);
const loading = ref(true);
const customerId = ref<number | "">("");
const title = ref("");
const startDate = ref("");
const endDate = ref("");
const saving = ref(false);

const customerNames = computed(() => {
  const map = new Map<number, string>();
  for (const c of customers.value) map.set(c.id, `${c.firstName} ${c.lastName}`.trim());
  return map;
});

async function fetchContracts() {
  loading.value = true;
  try {
    const [rows, customerRows] = await Promise.all([
      getContracts(),
      customers.value.length > 0 ? Promise.resolve(customers.value) : getCustomers({ limit: 200 }),
    ]);
    contracts.value = rows;
    customers.value = customerRows;
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to load contracts", "error");
  } finally {
    loading.value = false;
  }
}

async function handleCreate() {
  if (customerId.value === "" || !startDate.value || !endDate.value) {
    showToast("Customer, start and end dates are required", "error");
    return;
  }
  saving.value = true;
  try {
    const created = await createContract({
      customerId: Number(customerId.value), title: title.value.trim(), status: "draft",
      startDate: startDate.value, endDate: endDate.value, contractValue: 0,
      billingFrequency: "monthly", slaTerms: "", notes: "", siteIds: [],
    });
    contracts.value = [created, ...contracts.value];
    title.value = "";
    showToast("Contract created", "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to create contract", "error");
  } finally {
    saving.value = false;
  }
}

onMounted(fetchContracts);

async function setStatus(c: Contract, status: Contract["status"]) {
  try {
    const updated = await updateContract(c.id, { status });
    contracts.value = contracts.value.map((x) => (x.id === c.id ? updated : x));
    showToast(`Contract ${status}`, "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to update contract", "error");
  }
}

async function handleRenew(c: Contract) {
  // Default: next 12-month period starting the day after the current end.
  const end = new Date(c.endDate + "T00:00:00");
  const start = new Date(end);
  start.setDate(start.getDate() + 1);
  const nextEnd = new Date(start);
  nextEnd.setFullYear(nextEnd.getFullYear() + 1);
  nextEnd.setDate(nextEnd.getDate() - 1);
  const fmt = (d: Date) => d.toISOString().slice(0, 10);
  try {
    const next = await renewContract(c.id, fmt(start), fmt(nextEnd));
    await fetchContracts();
    showToast(`Renewed as ${next.contractNumber}`, "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to renew contract", "error");
  }
}
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-white">
    <div class="border-b border-gray-200 px-6 py-4">
      <h2 class="text-base font-semibold text-gray-900">Contracts</h2>
      <p class="mt-1 text-sm text-gray-500">Optional. One-time and recurring bookings work without a contract.</p>
    </div>
    <div class="grid grid-cols-1 gap-2 border-b border-gray-200 px-6 py-4 sm:grid-cols-5">
      <select v-model="customerId" class="rounded-lg border border-gray-200 px-3 py-2 text-sm">
        <option value="" disabled>Select customer</option>
        <option v-for="c in customers" :key="c.id" :value="c.id">
          {{ c.firstName }} {{ c.lastName }}
        </option>
      </select>
      <input v-model="title" placeholder="Title" class="rounded-lg border border-gray-200 px-3 py-2 text-sm" />
      <input v-model="startDate" type="date" class="rounded-lg border border-gray-200 px-3 py-2 text-sm" />
      <input v-model="endDate" type="date" class="rounded-lg border border-gray-200 px-3 py-2 text-sm" />
      <button type="button" :disabled="saving" class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" @click="handleCreate">Add contract</button>
    </div>
    <div v-if="loading" class="px-6 py-10 text-center text-sm text-gray-500">Loading contracts...</div>
    <div v-else-if="contracts.length === 0" class="px-6 py-10 text-center text-sm text-gray-500">No contracts yet.</div>
    <table v-else class="w-full text-left text-sm">
      <thead><tr class="border-b border-gray-200 text-xs uppercase text-gray-500">
        <th class="px-6 py-3">Contract</th><th class="px-6 py-3">Customer</th><th class="px-6 py-3">Period</th><th class="px-6 py-3">Status</th><th class="px-6 py-3 text-right">Actions</th>
      </tr></thead>
      <tbody>
        <tr v-for="c in contracts" :key="c.id" class="border-b border-gray-100">
          <td class="px-6 py-3 font-medium">{{ c.contractNumber }}<div class="text-xs font-normal text-gray-500">{{ c.title }}</div></td>
          <td class="px-6 py-3 text-gray-600">{{ customerNames.get(c.customerId) ?? `#${c.customerId}` }}</td>
          <td class="px-6 py-3 text-gray-600">{{ c.startDate }} → {{ c.endDate }}</td>
          <td class="px-6 py-3"><span class="rounded-full bg-gray-100 px-2.5 py-0.5 text-xs">{{ c.status }}</span></td>
          <td class="px-6 py-3 text-right whitespace-nowrap">
            <template v-if="canEdit">
              <button v-if="c.status === 'draft'" type="button" class="px-2 py-1 text-sm font-medium text-navy-600" @click="setStatus(c, 'active')">Activate</button>
              <button v-if="c.status === 'active' || c.status === 'expiring'" type="button" class="px-2 py-1 text-sm font-medium text-navy-600" @click="setStatus(c, 'cancelled')">Cancel</button>
              <button v-if="c.status === 'active' || c.status === 'expiring' || c.status === 'expired'" type="button" class="px-2 py-1 text-sm font-medium text-navy-600" @click="handleRenew(c)">Renew</button>
            </template>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
