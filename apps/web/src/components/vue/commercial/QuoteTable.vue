<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { approveQuote, createQuote, getQuotes, updateQuote, type Quote } from "../../../lib/quotes";
import { getCustomers, type Customer } from "../../../lib/customers";
import { getSites, type Site } from "../../../lib/sites";
import { showToast } from "../../../lib/toast";

const quotes = ref<Quote[]>([]);
const customers = ref<Customer[]>([]);
const sites = ref<Site[]>([]);
const loading = ref(true);
const customerId = ref<number | "">("");
const siteId = ref<number | "">("");
const serviceName = ref("");
const quantity = ref(1);
const unitPrice = ref(0);
const saving = ref(false);

const customerNames = computed(() => {
  const map = new Map<number, string>();
  for (const c of customers.value) map.set(c.id, `${c.firstName} ${c.lastName}`.trim());
  return map;
});

async function fetchQuotes() {
  loading.value = true;
  try {
    const [rows, customerRows] = await Promise.all([
      getQuotes(),
      customers.value.length > 0 ? Promise.resolve(customers.value) : getCustomers({ limit: 200 }),
    ]);
    quotes.value = rows;
    customers.value = customerRows;
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to load quotes", "error");
  } finally {
    loading.value = false;
  }
}

watch(customerId, async (cid) => {
  siteId.value = "";
  sites.value = [];
  if (cid === "") return;
  try {
    sites.value = await getSites({ customerId: Number(cid), limit: 100 });
  } catch {
    /* site is optional */
  }
});

async function handleCreate() {
  if (customerId.value === "" || !serviceName.value.trim()) {
    showToast("Customer and a service line are required", "error");
    return;
  }
  saving.value = true;
  try {
    const created = await createQuote({
      customerId: Number(customerId.value),
      siteId: siteId.value === "" ? null : Number(siteId.value),
      status: "draft", currency: "THB", taxRate: 7,
      items: [{ serviceName: serviceName.value.trim(), quantity: quantity.value, unitPrice: unitPrice.value }],
    });
    quotes.value = [created, ...quotes.value];
    serviceName.value = "";
    showToast("Quote created", "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to create quote", "error");
  } finally {
    saving.value = false;
  }
}

async function handleApprove(q: Quote) {
  try {
    const updated = await approveQuote(q.id);
    quotes.value = quotes.value.map((x) => (x.id === q.id ? updated : x));
    showToast("Quote accepted", "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to approve quote", "error");
  }
}

async function handleSend(q: Quote) {
  try {
    const updated = await updateQuote(q.id, { status: "sent" });
    quotes.value = quotes.value.map((x) => (x.id === q.id ? updated : x));
    showToast("Quote sent to customer", "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to send quote", "error");
  }
}

onMounted(fetchQuotes);
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-white">
    <div class="border-b border-gray-200 px-6 py-4">
      <h2 class="text-base font-semibold text-gray-900">Quotes</h2>
      <p class="mt-1 text-sm text-gray-500">Customer → Site survey → Quote → Accepted → optional Contract → Booking → Invoice.</p>
    </div>
    <div class="grid grid-cols-1 gap-2 border-b border-gray-200 px-6 py-4 sm:grid-cols-6">
      <select v-model="customerId" class="rounded-lg border border-gray-200 px-3 py-2 text-sm">
        <option value="" disabled>Select customer</option>
        <option v-for="c in customers" :key="c.id" :value="c.id">
          {{ c.firstName }} {{ c.lastName }}
        </option>
      </select>
      <select v-model="siteId" :disabled="customerId === ''" class="rounded-lg border border-gray-200 px-3 py-2 text-sm disabled:opacity-50">
        <option value="">No specific site</option>
        <option v-for="s in sites" :key="s.id" :value="s.id">{{ s.name }}</option>
      </select>
      <input v-model="serviceName" placeholder="Service (e.g. Weekly lobby)" class="rounded-lg border border-gray-200 px-3 py-2 text-sm" />
      <input v-model.number="quantity" type="number" min="1" placeholder="Qty" class="rounded-lg border border-gray-200 px-3 py-2 text-sm" />
      <input v-model.number="unitPrice" type="number" min="0" placeholder="Unit price" class="rounded-lg border border-gray-200 px-3 py-2 text-sm" />
      <button type="button" :disabled="saving" class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" @click="handleCreate">Add quote</button>
    </div>
    <div v-if="loading" class="px-6 py-10 text-center text-sm text-gray-500">Loading quotes...</div>
    <div v-else-if="quotes.length === 0" class="px-6 py-10 text-center text-sm text-gray-500">No quotes yet.</div>
    <table v-else class="w-full text-left text-sm">
      <thead><tr class="border-b border-gray-200 text-xs uppercase text-gray-500">
        <th class="px-6 py-3">Quote</th><th class="px-6 py-3">Customer</th><th class="px-6 py-3">Total</th><th class="px-6 py-3">Status</th><th class="px-6 py-3 text-right">Actions</th>
      </tr></thead>
      <tbody>
        <tr v-for="q in quotes" :key="q.id" class="border-b border-gray-100">
          <td class="px-6 py-3 font-medium">{{ q.quoteNumber }}</td>
          <td class="px-6 py-3 text-gray-600">{{ customerNames.get(q.customerId) ?? `#${q.customerId}` }}</td>
          <td class="px-6 py-3 text-gray-600">{{ q.currency }} {{ q.total.toFixed(2) }}</td>
          <td class="px-6 py-3"><span class="rounded-full bg-gray-100 px-2.5 py-0.5 text-xs">{{ q.status }}</span></td>
          <td class="px-6 py-3 text-right whitespace-nowrap">
            <button v-if="q.status === 'draft'" type="button" class="text-sm font-medium text-navy-600" @click="handleSend(q)">Send</button>
            <button v-if="q.status === 'sent'" type="button" class="ml-2 text-sm font-medium text-navy-600" @click="handleApprove(q)">Approve</button>
            <span v-if="q.status === 'accepted'" class="text-xs text-gray-400">accepted — ready for booking/contract</span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
