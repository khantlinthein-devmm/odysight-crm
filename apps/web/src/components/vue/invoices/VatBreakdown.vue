<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { getWorkspaceSettings } from "../../../lib/settings";

/**
 * Shows how a price before VAT becomes the invoice total: price, VAT at the
 * company's rate (Settings → Company → Charge VAT, rate under Payments) and
 * the total, the same way the API calculates it.
 */
const props = defineProps<{ price: number | null | undefined; currency: string }>();

const vatOn = ref(false);
const rate = ref(7);

onMounted(async () => {
  try {
    const s = await getWorkspaceSettings();
    vatOn.value = !!s.company.vatRegistered;
    rate.value = s.payments.taxRatePercent ?? 7;
  } catch {
    /* settings unavailable: show the price only */
  }
});

const round2 = (v: number) => Math.round(v * 100) / 100;
const base = computed(() => (props.price && props.price > 0 ? round2(props.price) : 0));
const vat = computed(() => (vatOn.value ? round2((base.value * rate.value) / 100) : 0));
const total = computed(() => round2(base.value + vat.value));

function money(v: number): string {
  return `${props.currency} ${v.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}
</script>

<template>
  <div v-if="base > 0" class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm">
    <div class="flex justify-between text-gray-600">
      <span>Price before VAT</span><span class="tabular-nums">{{ money(base) }}</span>
    </div>
    <div v-if="vatOn" class="flex justify-between text-gray-600">
      <span>VAT {{ rate }}%</span><span class="tabular-nums">{{ money(vat) }}</span>
    </div>
    <div class="mt-1 flex justify-between border-t border-gray-200 pt-1 font-semibold text-gray-900">
      <span>Total</span><span class="tabular-nums">{{ money(total) }}</span>
    </div>
    <p v-if="!vatOn" class="mt-1 text-xs text-gray-500">
      No VAT: Charge VAT is off in Settings → Company.
    </p>
    <p v-else class="mt-1 text-xs text-gray-500">
      Withholding tax, if the customer deducts it, comes off the total.
    </p>
  </div>
</template>
