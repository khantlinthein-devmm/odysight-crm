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

function ring(pctValue: number | null): string {
  const v = pctValue ?? 0;
  return `${v} ${100 - v}`;
}

onMounted(fetchReport);
</script>

<template>
  <div class="mt-4 overflow-hidden rounded-3xl bg-white shadow-sm ring-1 ring-gray-100">
    <div class="flex flex-wrap items-start justify-between gap-3 border-b border-amber-100 bg-gradient-to-r from-amber-50 to-white px-6 py-5">
      <div>
        <p class="text-[11px] font-bold uppercase tracking-[0.18em] text-amber-600">Wealth management</p>
        <h2 class="mt-1 text-lg font-bold tracking-tight text-gray-900">Commercial performance</h2>
        <p class="mt-0.5 text-sm text-gray-500">Revenue by site and contract, quote win rate, checklist completion.</p>
      </div>
      <span class="rounded-full bg-slate-900 px-3 py-1 text-[11px] font-bold text-amber-300 ring-1 ring-slate-700">◆ Premium report</span>
    </div>

    <div v-if="loading" class="px-6 py-10 text-center text-sm text-gray-500">Loading commercial report...</div>

    <template v-else-if="report">
      <!-- Banking stat duo: win rate + completion as account cards -->
      <div class="grid grid-cols-1 gap-4 px-6 py-5 sm:grid-cols-2">
        <div class="relative overflow-hidden rounded-2xl bg-gradient-to-br from-slate-950 to-slate-800 p-5 text-white shadow-md">
          <div class="pointer-events-none absolute -right-10 -top-10 h-32 w-32 rounded-full bg-amber-400/25 blur-2xl"></div>
          <p class="text-[11px] font-bold uppercase tracking-widest text-amber-300">Quote win rate</p>
          <div class="mt-3 flex items-center gap-4">
            <svg viewBox="0 0 80 80" class="h-16 w-16 shrink-0">
              <circle cx="40" cy="40" r="34" fill="none" class="stroke-white/10" stroke-width="9" />
              <circle cx="40" cy="40" r="34" fill="none" stroke="#D4AF37" stroke-width="9" stroke-linecap="round" pathLength="100" :stroke-dasharray="ring(report.quoteWinRate.rate)" stroke-dashoffset="25" />
              <text x="40" y="45" text-anchor="middle" font-size="15" font-weight="700" class="fill-white">{{ pct(report.quoteWinRate.rate) }}</text>
            </svg>
            <div>
              <p class="text-2xl font-bold">{{ pct(report.quoteWinRate.rate) }}</p>
              <p class="text-xs text-slate-300">{{ report.quoteWinRate.accepted }} accepted of {{ report.quoteWinRate.total }}</p>
            </div>
          </div>
        </div>
        <div class="relative overflow-hidden rounded-2xl bg-gradient-to-br from-amber-50 to-orange-50 p-5 ring-1 ring-amber-200">
          <p class="text-[11px] font-bold uppercase tracking-widest text-amber-700">Checklist completion</p>
          <div class="mt-3 flex items-center gap-4">
            <svg viewBox="0 0 80 80" class="h-16 w-16 shrink-0">
              <circle cx="40" cy="40" r="34" fill="none" class="stroke-amber-200" stroke-width="9" />
              <circle cx="40" cy="40" r="34" fill="none" stroke="#059669" stroke-width="9" stroke-linecap="round" pathLength="100" :stroke-dasharray="ring(report.checklists.completionRate)" stroke-dashoffset="25" />
              <text x="40" y="45" text-anchor="middle" font-size="15" font-weight="700" fill="#065f46">{{ pct(report.checklists.completionRate) }}</text>
            </svg>
            <div>
              <p class="text-2xl font-bold text-gray-900">{{ pct(report.checklists.completionRate) }}</p>
              <p class="text-xs text-gray-500">{{ report.checklists.itemsDone }} of {{ report.checklists.itemsTotal }} items · {{ report.checklists.completed }}/{{ report.checklists.total }} lists done</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Transaction-style lists -->
      <div class="grid grid-cols-1 gap-4 px-6 pb-6 xl:grid-cols-2">
        <section class="overflow-hidden rounded-2xl ring-1 ring-gray-100">
          <h3 class="bg-slate-50 px-4 py-2.5 text-xs font-bold uppercase tracking-widest text-slate-500">Recent activity · by site</h3>
          <ul v-if="report.revenueBySite.length > 0" class="divide-y divide-gray-100 bg-white">
            <li v-for="s in report.revenueBySite" :key="s.siteId" class="flex items-center gap-3 px-4 py-3 transition hover:bg-amber-50/50">
              <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-amber-400 to-orange-500 text-xs font-bold text-white">⌂</span>
              <div class="min-w-0 flex-1">
                <p class="truncate text-sm font-semibold text-gray-900">{{ s.siteName }}</p>
                <p class="text-xs text-gray-400">Customer #{{ s.customerId }} · {{ s.completed }}/{{ s.bookings }} done</p>
              </div>
              <p class="shrink-0 text-sm font-bold tabular-nums text-gray-900">+{{ s.billed.toFixed(2) }}</p>
            </li>
          </ul>
          <p v-else class="bg-white px-4 py-4 text-sm text-gray-500">No sites yet.</p>
        </section>

        <section class="overflow-hidden rounded-2xl ring-1 ring-gray-100">
          <h3 class="bg-slate-50 px-4 py-2.5 text-xs font-bold uppercase tracking-widest text-slate-500">Contracts portfolio</h3>
          <ul v-if="report.revenueByContract.length > 0" class="divide-y divide-gray-100 bg-white">
            <li v-for="c in report.revenueByContract" :key="c.contractId" class="flex items-center gap-3 px-4 py-3 transition hover:bg-amber-50/50">
              <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-slate-800 to-slate-950 text-xs font-bold text-amber-300">◆</span>
              <div class="min-w-0 flex-1">
                <p class="truncate text-sm font-semibold text-gray-900">{{ c.contractNumber }}</p>
                <p class="truncate text-xs text-gray-400">{{ c.title }} · {{ c.bookings }} bookings</p>
              </div>
              <div class="shrink-0 text-right">
                <p class="text-sm font-bold tabular-nums text-gray-900">{{ c.billed.toFixed(2) }}</p>
                <p class="text-[11px] tabular-nums text-gray-400">/ {{ c.contractValue.toFixed(2) }}</p>
              </div>
            </li>
          </ul>
          <p v-else class="bg-white px-4 py-4 text-sm text-gray-500">No contracts yet — one-time bookings need none.</p>
        </section>
      </div>
    </template>
  </div>
</template>
