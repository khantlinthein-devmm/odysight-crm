<script setup lang="ts">
import { onMounted, ref } from "vue";
import { getCommercialReport, type CommercialReport } from "../../../lib/reports";
import { showToast } from "../../../lib/toast";

const report = ref<CommercialReport | null>(null);
const loading = ref(true);

async function fetchReport() {
  loading.value = true;
  try {
    report.value = await getCommercialReport();
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to load commercial report", "error");
  } finally {
    loading.value = false;
  }
}

function pct(v: number | null): string {
  return v === null ? "—" : `${v.toFixed(1)}%`;
}

onMounted(fetchReport);
</script>

<template>
  <div class="mt-6 rounded-xl border border-gray-200 bg-white">
    <div class="border-b border-gray-200 px-6 py-4">
      <h2 class="text-base font-semibold text-gray-900">Commercial performance</h2>
      <p class="mt-1 text-sm text-gray-500">Revenue by site and contract, quote win rate, checklist completion.</p>
    </div>
    <div v-if="loading" class="px-6 py-10 text-center text-sm text-gray-500">Loading commercial report...</div>
    <template v-else-if="report">
      <div class="grid grid-cols-1 gap-4 px-6 py-4 sm:grid-cols-2">
        <div class="rounded-lg bg-gray-50 px-4 py-3">
          <p class="text-xs uppercase tracking-wide text-gray-500">Quote win rate</p>
          <p class="mt-1 text-lg font-semibold">{{ pct(report.quoteWinRate.rate) }}</p>
          <p class="text-xs text-gray-500">{{ report.quoteWinRate.accepted }} accepted of {{ report.quoteWinRate.total }}</p>
        </div>
        <div class="rounded-lg bg-gray-50 px-4 py-3">
          <p class="text-xs uppercase tracking-wide text-gray-500">Checklist completion</p>
          <p class="mt-1 text-lg font-semibold">{{ pct(report.checklists.completionRate) }}</p>
          <p class="text-xs text-gray-500">{{ report.checklists.itemsDone }} of {{ report.checklists.itemsTotal }} items · {{ report.checklists.completed }}/{{ report.checklists.total }} lists done</p>
        </div>
      </div>
      <div class="px-6 pb-2">
        <h3 class="text-sm font-semibold text-gray-900">Revenue by site</h3>
      </div>
      <table v-if="report.revenueBySite.length > 0" class="w-full text-left text-sm">
        <thead><tr class="border-b border-gray-200 text-xs uppercase text-gray-500">
          <th class="px-6 py-2">Site</th><th class="px-6 py-2">Bookings</th><th class="px-6 py-2">Billed</th>
        </tr></thead>
        <tbody>
          <tr v-for="s in report.revenueBySite" :key="s.siteId" class="border-b border-gray-100">
            <td class="px-6 py-2 font-medium">{{ s.siteName }} <span class="font-normal text-gray-400">#{{ s.customerId }}</span></td>
            <td class="px-6 py-2 text-gray-600">{{ s.completed }}/{{ s.bookings }} done</td>
            <td class="px-6 py-2 text-gray-600">{{ s.billed.toFixed(2) }}</td>
          </tr>
        </tbody>
      </table>
      <p v-else class="px-6 py-2 text-sm text-gray-500">No sites yet.</p>
      <div class="px-6 pb-2 pt-4">
        <h3 class="text-sm font-semibold text-gray-900">Revenue by contract</h3>
      </div>
      <table v-if="report.revenueByContract.length > 0" class="w-full text-left text-sm">
        <thead><tr class="border-b border-gray-200 text-xs uppercase text-gray-500">
          <th class="px-6 py-2">Contract</th><th class="px-6 py-2">Bookings</th><th class="px-6 py-2">Billed / value</th>
        </tr></thead>
        <tbody>
          <tr v-for="c in report.revenueByContract" :key="c.contractId" class="border-b border-gray-100">
            <td class="px-6 py-2 font-medium">{{ c.contractNumber }} <span class="font-normal text-gray-400">{{ c.title }}</span></td>
            <td class="px-6 py-2 text-gray-600">{{ c.bookings }}</td>
            <td class="px-6 py-2 text-gray-600">{{ c.billed.toFixed(2) }} / {{ c.contractValue.toFixed(2) }}</td>
          </tr>
        </tbody>
      </table>
      <p v-else class="px-6 py-2 text-sm text-gray-500">No contracts yet — one-time bookings need none.</p>
      <div class="h-4"></div>
    </template>
  </div>
</template>
