<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { createSite, deleteSite, getSites, type Site } from "../../../lib/sites";
import { getCustomers, type Customer } from "../../../lib/customers";
import { showToast } from "../../../lib/toast";

const sites = ref<Site[]>([]);
const customers = ref<Customer[]>([]);
const loading = ref(true);
const customerFilter = ref<number | "">("");
const name = ref("");
const address = ref("");
const customerId = ref<number | "">("");
const saving = ref(false);

const customerNames = computed(() => {
  const map = new Map<number, string>();
  for (const c of customers.value) map.set(c.id, `${c.firstName} ${c.lastName}`.trim());
  return map;
});

function customerLabel(id: number): string {
  return customerNames.value.get(id) ?? `Customer #${id}`;
}

async function fetchSites() {
  loading.value = true;
  try {
    const [siteRows, customerRows] = await Promise.all([
      getSites({ customerId: customerFilter.value === "" ? undefined : Number(customerFilter.value) }),
      customers.value.length > 0 ? Promise.resolve(customers.value) : getCustomers({ limit: 200 }),
    ]);
    sites.value = siteRows;
    customers.value = customerRows;
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to load sites", "error");
  } finally {
    loading.value = false;
  }
}

async function handleCreate() {
  if (customerId.value === "" || !name.value.trim() || !address.value.trim()) {
    showToast("Customer, name and address are required", "error");
    return;
  }
  saving.value = true;
  try {
    const created = await createSite({
      customerId: Number(customerId.value), name: name.value.trim(), address: address.value.trim(),
      contactName: "", phone: "", email: "", notes: "", status: "active", isDefault: false,
    });
    sites.value = [created, ...sites.value];
    name.value = "";
    address.value = "";
    showToast("Site created", "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to create site", "error");
  } finally {
    saving.value = false;
  }
}

async function handleDelete(id: number) {
  try {
    await deleteSite(id);
    sites.value = sites.value.filter((s) => s.id !== id);
    showToast("Site deleted", "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to delete site", "error");
  }
}

onMounted(fetchSites);
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-white">
    <div class="border-b border-gray-200 px-6 py-4">
      <h2 class="text-base font-semibold text-gray-900">Sites</h2>
      <p class="mt-1 text-sm text-gray-500">Physical locations per customer. One-time bookings work without a site; commercial customers add branches here.</p>
      <div class="mt-3 flex flex-col gap-2 sm:flex-row">
        <select v-model="customerFilter" class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm sm:w-64" @change="fetchSites">
          <option value="">All customers</option>
          <option v-for="c in customers" :key="c.id" :value="c.id">
            {{ c.firstName }} {{ c.lastName }}
          </option>
        </select>
      </div>
    </div>
    <div class="grid grid-cols-1 gap-2 border-b border-gray-200 px-6 py-4 sm:grid-cols-4">
      <select v-model="customerId" class="rounded-lg border border-gray-200 px-3 py-2 text-sm">
        <option value="" disabled>Select customer</option>
        <option v-for="c in customers" :key="c.id" :value="c.id">
          {{ c.firstName }} {{ c.lastName }}
        </option>
      </select>
      <input v-model="name" placeholder="Site name (e.g. Silom Branch)" class="rounded-lg border border-gray-200 px-3 py-2 text-sm" />
      <input v-model="address" placeholder="Address" class="rounded-lg border border-gray-200 px-3 py-2 text-sm" />
      <button type="button" :disabled="saving" class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" @click="handleCreate">Add site</button>
    </div>
    <div v-if="loading" class="px-6 py-10 text-center text-sm text-gray-500">Loading sites...</div>
    <div v-else-if="sites.length === 0" class="px-6 py-10 text-center text-sm text-gray-500">No sites yet.</div>
    <table v-else class="w-full text-left text-sm">
      <thead><tr class="border-b border-gray-200 text-xs uppercase text-gray-500">
        <th class="px-6 py-3">Site</th><th class="px-6 py-3">Customer</th><th class="px-6 py-3">Address</th><th class="px-6 py-3 text-right">Actions</th>
      </tr></thead>
      <tbody>
        <tr v-for="s in sites" :key="s.id" class="border-b border-gray-100">
          <td class="px-6 py-3 font-medium">{{ s.name }} <span v-if="s.isDefault" class="ml-1 rounded bg-gray-100 px-1.5 text-xs">default</span></td>
          <td class="px-6 py-3 text-gray-600">{{ customerLabel(s.customerId) }}</td>
          <td class="px-6 py-3 text-gray-600">{{ s.address }}</td>
          <td class="px-6 py-3 text-right"><button type="button" class="text-sm text-red-600" @click="handleDelete(s.id)">Delete</button></td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
