<script setup lang="ts">
import { ref, watch } from "vue";
import { getBookingProfit, type BookingProfit } from "../../../lib/booking-profit";
import { formatMoney } from "../../../lib/settings";

// Profit section of the booking side panel: price − team pay − supplies.
// Re-fetches whenever the booking (price, duration, team) changes.
const props = defineProps<{ bookingId: number; version?: string }>();

const profit = ref<BookingProfit | null>(null);
const loading = ref(false);
const failed = ref(false);

const payTypeLabels: Record<string, string> = {
  hourly: "/h",
  daily: "/day",
  per_job: "/job",
  monthly: "/month",
};

let seq = 0;
async function load() {
  const mine = ++seq;
  loading.value = true;
  failed.value = false;
  try {
    const p = await getBookingProfit(props.bookingId);
    if (mine === seq) profit.value = p;
  } catch {
    if (mine === seq) failed.value = true;
  } finally {
    if (mine === seq) loading.value = false;
  }
}

watch(() => [props.bookingId, props.version], load, { immediate: true });

function hours(h: number): string {
  return `${Number.isInteger(h) ? h : h.toFixed(2).replace(/0+$/, "")}h`;
}
</script>

<template>
  <section>
    <h3 class="mb-2 text-xs font-semibold uppercase tracking-wider text-gray-400">Profit (estimate)</h3>
    <p v-if="loading && !profit" class="text-sm text-gray-500">Calculating…</p>
    <p v-else-if="failed" class="text-sm text-gray-500">Could not calculate profit.</p>
    <div v-else-if="profit" class="rounded-xl text-sm ring-1 ring-gray-200">
      <div class="flex justify-between px-4 py-2.5">
        <span class="text-gray-500">Price (before VAT)</span>
        <span class="font-medium text-gray-900">{{ profit.price != null ? formatMoney(profit.price) : "—" }}</span>
      </div>

      <div class="border-t border-gray-100 px-4 py-2.5">
        <div class="flex justify-between">
          <span class="text-gray-500">− Team pay</span>
          <span class="text-gray-900">{{ formatMoney(profit.labor) }}</span>
        </div>
        <ul v-if="profit.team.length" class="mt-1 space-y-0.5 text-xs text-gray-500">
          <li v-for="m in profit.team" :key="m.cleanerId" class="flex justify-between gap-2">
            <span class="truncate">
              {{ m.name }}<span v-if="m.role === 'primary'"> (lead)</span>
              <template v-if="m.hasRate && m.rate != null && m.payType">
                · {{ hours(m.hours) }} × {{ formatMoney(m.rate) }}{{ payTypeLabels[m.payType] }}
              </template>
              <a v-else-if="!m.hasRate" href="/payroll" class="font-medium text-amber-700 hover:underline">· no pay rate set</a>
            </span>
            <span v-if="m.hasRate && m.rate != null" class="shrink-0">{{ formatMoney(m.cost) }}</span>
          </li>
        </ul>
        <p v-else class="mt-1 text-xs text-gray-400">No team assigned yet.</p>
      </div>

      <div class="flex justify-between border-t border-gray-100 px-4 py-2.5">
        <span class="text-gray-500">− Supplies used</span>
        <span class="text-gray-900">{{ formatMoney(profit.supplies) }}</span>
      </div>

      <div class="flex items-baseline justify-between border-t border-gray-200 bg-gray-50 px-4 py-3">
        <span class="font-semibold text-gray-900">Profit</span>
        <span v-if="profit.profit != null" :class="['text-base font-bold', profit.profit < 0 ? 'text-red-600' : 'text-green-700']">
          {{ formatMoney(profit.profit) }}
          <span v-if="profit.margin != null" class="ml-1 text-xs font-semibold">({{ profit.margin }}%)</span>
        </span>
        <span v-else class="text-xs text-gray-500">Set a booking price to see profit</span>
      </div>
    </div>
    <p v-if="profit && profit.missingRates > 0" class="mt-1.5 text-xs text-amber-700">
      {{ profit.missingRates }} cleaner{{ profit.missingRates === 1 ? " has" : "s have" }} no pay rate, so their pay is counted as {{ formatMoney(0) }} —
      <a href="/payroll" class="font-medium underline">set pay rates</a>.
    </p>
    <p v-if="profit" class="mt-1 text-xs text-gray-400">Planned hours × pay rate. Overtime and travel are not included.</p>
  </section>
</template>
