<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { getWorkspaceSettings } from "../../../lib/settings";

/**
 * Shows how a price becomes the invoice total, the same way the API does:
 * VAT at the company's rate (Settings → Company → Charge VAT, rate under
 * Payments) is added on top, or, when prices include VAT (ราคารวม VAT),
 * taken out of the price.
 */
const props = defineProps<{ price: number | null | undefined; currency: string }>();

const vatOn = ref(false);
const inclusive = ref(false);
const rate = ref(7);

onMounted(async () => {
  try {
    const s = await getWorkspaceSettings();
    vatOn.value = !!s.company.vatRegistered;
    inclusive.value = !!s.company.pricesIncludeVat;
    rate.value = s.payments.taxRatePercent ?? 7;
  } catch {
    /* settings unavailable: show the price only */
  }
});

const round2 = (v: number) => Math.round(v * 100) / 100;
const price = computed(() => (props.price && props.price > 0 ? round2(props.price) : 0));
const inside = computed(() => vatOn.value && inclusive.value && rate.value > 0);
// Value before VAT, VAT and total.
const base = computed(() => (inside.value ? round2((price.value * 100) / (100 + rate.value)) : price.value));
const vat = computed(() =>
  !vatOn.value ? 0 : inside.value ? round2(price.value - base.value) : round2((base.value * rate.value) / 100),
);
const total = computed(() => (inside.value ? price.value : round2(base.value + vat.value)));

function money(v: number): string {
  return `${props.currency} ${v.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}
</script>

<template>
  <div v-if="base > 0" class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm">
    <div class="flex justify-between text-gray-600">
      <span>Value before VAT</span><span class="tabular-nums">{{ money(base) }}</span>
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
      <template v-if="inside">Price includes VAT (ราคารวม VAT) — the VAT is taken out of it. </template>
      Withholding tax, if the customer deducts it, comes off the total.
    </p>
  </div>
</template>
