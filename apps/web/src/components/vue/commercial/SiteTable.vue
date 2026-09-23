<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { createSite, deleteSite, getSites, type Site } from "../../../lib/sites";
import { getCustomers, type Customer } from "../../../lib/customers";
import { showToast } from "../../../lib/toast";

const sites = ref<Site[]>([]);
const customers = ref<Customer[]>([]);
const loading = ref(true);
const customerFilter = ref<number | "">("");
const search = ref("");
const selectedId = ref<number | null>(null);
const name = ref("");
const address = ref("");
const customerId = ref<number | "">("");
const saving = ref(false);
const deletingId = ref<number | null>(null);

const customerNames = computed(() => {
  const map = new Map<number, string>();
  for (const c of customers.value) map.set(c.id, `${c.firstName} ${c.lastName}`.trim());
  return map;
});

function customerLabel(id: number): string {
  return customerNames.value.get(id) ?? `Customer #${id}`;
}

const filteredSites = computed(() => {
  const q = search.value.trim().toLowerCase();
  return sites.value.filter((s) => {
    if (!q) return true;
    return `${s.name} ${s.address} ${customerLabel(s.customerId)}`.toLowerCase().includes(q);
  });
});

const selected = computed(() => {
  if (selectedId.value == null) return filteredSites.value[0] ?? sites.value[0] ?? null;
  return sites.value.find((s) => s.id === selectedId.value) ?? filteredSites.value[0] ?? null;
});

function select(id: number) {
  selectedId.value = id;
}

// Actual Google Maps embed — no API key needed.
// Uses lat/lng when the site has coordinates, otherwise the address query.
const mapQuery = computed(() => {
  const s = selected.value;
  if (!s) return "";
  if (s.latitude != null && s.longitude != null) return `${s.latitude},${s.longitude}`;
  return s.address || s.name;
});

const mapEmbedUrl = computed(() => {
  if (!mapQuery.value) return "";
  return `https://maps.google.com/maps?q=${encodeURIComponent(mapQuery.value)}&z=16&output=embed`;
});

const directionsUrl = computed(() => {
  if (!mapQuery.value) return "";
  return `https://www.google.com/maps/dir/?api=1&destination=${encodeURIComponent(mapQuery.value)}`;
});

const openInMapsUrl = computed(() => {
  if (!mapQuery.value) return "";
  return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(mapQuery.value)}`;
});

async function fetchSites() {
  loading.value = true;
  try {
    const [siteRows, customerRows] = await Promise.all([
      getSites({ customerId: customerFilter.value === "" ? undefined : Number(customerFilter.value) }),
      customers.value.length > 0 ? Promise.resolve(customers.value) : getCustomers({ limit: 200 }),
    ]);
    sites.value = siteRows;
    customers.value = customerRows;
    if (selectedId.value == null && siteRows.length > 0) selectedId.value = siteRows[0]!.id;
    if (selectedId.value != null && !siteRows.some((s) => s.id === selectedId.value)) {
      selectedId.value = siteRows[0]?.id ?? null;
    }
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
    selectedId.value = created.id;
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
  deletingId.value = id;
  try {
    await deleteSite(id);
    sites.value = sites.value.filter((s) => s.id !== id);
    if (selectedId.value === id) selectedId.value = sites.value[0]?.id ?? null;
    showToast("Site deleted", "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to delete site", "error");
  } finally {
    deletingId.value = null;
  }
}

onMounted(fetchSites);
</script>

<template>
  <div class="overflow-hidden rounded-xl border border-gray-200 bg-gray-100/70">
    <!-- Header -->
    <div class="border-b border-gray-200 bg-white px-6 py-4">
      <div class="flex flex-wrap items-center gap-3">
        <span class="flex h-10 w-10 items-center justify-center rounded-xl bg-navy-600 text-white shadow-sm">
          <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M17.657 16.657L13.414 20.9a2 2 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
        </span>
        <div>
          <h2 class="text-base font-bold text-gray-900">Sites</h2>
          <p class="text-xs text-gray-500">Physical locations per customer · click a site to see it on Google Maps</p>
        </div>
        <span class="rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-600">{{ sites.length }}</span>
        <div class="ml-auto flex w-full flex-col gap-2 sm:w-auto sm:flex-row">
          <select v-model="customerFilter" class="rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-navy-500 sm:w-56" @change="fetchSites">
            <option value="">All customers</option>
            <option v-for="c in customers" :key="c.id" :value="c.id">{{ c.firstName }} {{ c.lastName }}</option>
          </select>
          <input
            v-model="search"
            type="search"
            placeholder="Search site or address…"
            class="rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-navy-500 sm:w-56"
          />
        </div>
      </div>
      <!-- Quick add -->
      <div class="mt-3 grid grid-cols-1 gap-2 rounded-xl bg-gray-50 p-3 ring-1 ring-gray-200 sm:grid-cols-4">
        <select v-model="customerId" class="rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-navy-500">
          <option value="" disabled>Select customer</option>
          <option v-for="c in customers" :key="c.id" :value="c.id">{{ c.firstName }} {{ c.lastName }}</option>
        </select>
        <input v-model="name" placeholder="Site name (e.g. Silom Branch)" class="rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-navy-500" />
        <input v-model="address" placeholder="Address" class="rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-navy-500" />
        <button type="button" :disabled="saving" class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition hover:bg-navy-700 disabled:opacity-50" @click="handleCreate">
          {{ saving ? "Adding…" : "+ Add site" }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="bg-white px-6 py-16 text-center text-sm text-gray-500">Loading sites…</div>
    <div v-else-if="sites.length === 0" class="bg-white px-6 py-16 text-center">
      <p class="text-sm font-semibold text-gray-900">No sites yet</p>
      <p class="mt-1 text-sm text-gray-500">Add your first branch above — one-time bookings work without a site.</p>
    </div>

    <!-- Split view: cards + live Google Map -->
    <div v-else class="grid grid-cols-1 gap-0 lg:grid-cols-5">
      <!-- Site cards -->
      <div class="max-h-[640px] space-y-3 overflow-y-auto bg-gray-50 p-4 lg:col-span-2">
        <button
          v-for="s in filteredSites"
          :key="s.id"
          type="button"
          class="block w-full rounded-xl border bg-white p-4 text-left shadow-sm transition hover:shadow-md"
          :class="selected?.id === s.id ? 'border-navy-400 ring-2 ring-navy-200' : 'border-gray-200'"
          @click="select(s.id)"
        >
          <div class="flex items-start justify-between gap-2">
            <p class="min-w-0 truncate text-sm font-bold text-gray-900">{{ s.name }}</p>
            <span v-if="s.isDefault" class="shrink-0 rounded-full bg-gray-100 px-2 py-0.5 text-[11px] font-medium text-gray-600 ring-1 ring-gray-200">default</span>
          </div>
          <p class="mt-0.5 text-xs font-medium text-navy-700">{{ customerLabel(s.customerId) }}</p>
          <p class="mt-1.5 flex items-start gap-1.5 text-xs text-gray-600">
            <svg class="mt-0.5 h-3.5 w-3.5 shrink-0 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M17.657 16.657L13.414 20.9a2 2 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
            <span class="line-clamp-2">{{ s.address }}</span>
          </p>
          <div class="mt-3 flex items-center gap-2">
            <span class="inline-flex items-center gap-1.5 rounded-full bg-emerald-100 px-2 py-0.5 text-[11px] font-semibold text-emerald-700 ring-1 ring-emerald-200"><span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>{{ s.status }}</span>
            <span class="ml-auto flex gap-1">
              <a :href="openInMapsUrl" v-if="selected?.id === s.id" target="_blank" rel="noopener" class="rounded-lg px-2 py-1 text-xs font-semibold text-navy-700 hover:bg-navy-50" @click.stop>View map</a>
              <span
                role="button"
                tabindex="0"
                class="rounded-lg px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-50"
                @click.stop="handleDelete(s.id)"
                @keydown.enter.prevent="handleDelete(s.id)"
              >
                {{ deletingId === s.id ? "Deleting…" : "Delete" }}
              </span>
            </span>
          </div>
        </button>
        <p v-if="filteredSites.length === 0" class="rounded-xl bg-white p-6 text-center text-sm text-gray-500">No sites match your search.</p>
      </div>

      <!-- Live Google Map -->
      <div class="relative min-h-[420px] bg-gray-100 lg:col-span-3">
        <iframe
          v-if="mapEmbedUrl"
          :key="mapEmbedUrl"
          :src="mapEmbedUrl"
          class="absolute inset-0 h-full w-full border-0"
          loading="lazy"
          referrerpolicy="no-referrer-when-downgrade"
          title="Site location on Google Maps"
          allowfullscreen
        ></iframe>
        <!-- Floating site card over the map -->
        <div v-if="selected" class="absolute left-4 right-4 top-4 rounded-xl bg-white p-4 shadow-lg ring-1 ring-gray-200 sm:right-auto sm:max-w-sm">
          <div class="flex items-start gap-3">
            <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-navy-600 text-sm font-bold text-white">
              {{ (selected.name || "?").charAt(0).toUpperCase() }}
            </span>
            <div class="min-w-0">
              <p class="truncate text-sm font-bold text-gray-900">{{ selected.name }}</p>
              <p class="text-xs text-gray-500">{{ customerLabel(selected.customerId) }}</p>
              <p class="mt-1 line-clamp-2 text-xs text-gray-600">{{ selected.address }}</p>
              <p v-if="selected.phone" class="mt-0.5 text-xs text-gray-500">{{ selected.phone }}</p>
            </div>
          </div>
          <div class="mt-3 grid grid-cols-2 gap-2">
            <a :href="directionsUrl" target="_blank" rel="noopener" class="rounded-lg bg-navy-600 px-3 py-2 text-center text-xs font-semibold text-white transition hover:bg-navy-700">
              Directions
            </a>
            <a :href="openInMapsUrl" target="_blank" rel="noopener" class="rounded-lg border border-gray-300 bg-white px-3 py-2 text-center text-xs font-semibold text-gray-700 transition hover:bg-gray-100">
              Open in Maps
            </a>
          </div>
        </div>
        <p class="absolute bottom-2 right-3 rounded bg-white/80 px-2 py-0.5 text-[10px] text-gray-400">Map data © Google</p>
      </div>
    </div>
  </div>
</template>
