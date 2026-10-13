<script setup lang="ts">
import { computed, ref } from "vue";
import { formatMoney } from "../../../lib/settings";
import { previewProfit, type CostDraft } from "../../../lib/booking-profit";

// "Job costs & profit" block of the booking form. Fees are typed per cleaner
// (each can be adjusted), plus transportation and any other cost; the
// profit updates as you type. Supplies used are added after saving.

const props = defineProps<{
  team: { id: number; name: string; role: "primary" | "crew" }[];
  price: number | null;
  hours: number;
}>();

const draft = defineModel<CostDraft>({ required: true });
const emit = defineEmits<{ touched: [] }>();

const hourlyFee = ref<number | "">("");

const inputClass =
  "w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500";

function num(v: number | ""): number {
  return typeof v === "number" && Number.isFinite(v) ? v : 0;
}

const preview = computed(() =>
  previewProfit(props.price, {
    cleanerFees: Object.fromEntries(props.team.map((m) => [m.id, num(draft.value.fees[m.id] ?? "")])),
    transport: num(draft.value.transport),
    other: num(draft.value.other),
    otherNote: draft.value.otherNote,
  }),
);

// Quick fill: same hourly fee × job hours for everyone, then adjust per person.
function applyHourly() {
  const rate = num(hourlyFee.value);
  if (rate <= 0) return;
  const fee = Math.round(rate * props.hours * 100) / 100;
  for (const m of props.team) draft.value.fees[m.id] = fee;
  emit("touched");
}
</script>

<template>
  <div class="rounded-lg border border-gray-200 bg-gray-50 p-4">
    <div class="mb-3 flex flex-wrap items-baseline justify-between gap-2">
      <span class="text-sm font-semibold text-gray-900">Job costs &amp; profit</span>
      <span class="text-xs text-gray-500">Only office staff see this</span>
    </div>

    <div v-if="team.length" class="space-y-2">
      <div class="flex flex-wrap items-end gap-2">
        <label class="text-xs text-gray-600">
          Fill everyone: fee per hour
          <input
            v-model.number="hourlyFee"
            type="number"
            min="0"
            step="1"
            placeholder="e.g. 100"
            class="mt-1 block w-32 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-sm"
          />
        </label>
        <button
          type="button"
          class="rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-xs font-medium text-gray-700 hover:bg-gray-100"
          @click="applyHourly"
        >
          × {{ hours }}h = apply
        </button>
      </div>
      <div v-for="m in team" :key="m.id" class="flex items-center gap-3">
        <span class="min-w-0 flex-1 truncate text-sm text-gray-800">
          {{ m.name }}<span class="text-xs text-gray-500">{{ m.role === "primary" ? " (lead)" : " (crew)" }}</span>
        </span>
        <input
          v-model.number="draft.fees[m.id]"
          type="number"
          min="0"
          step="0.01"
          placeholder="Fee"
          :aria-label="`Fee for ${m.name}`"
          class="w-32 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-right text-sm"
          @input="emit('touched')"
        />
      </div>
    </div>
    <p v-else class="text-xs text-gray-500">Pick the cleaning team above to enter each cleaner's fee.</p>

    <div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-3">
      <label class="text-xs text-gray-600">
        Transportation
        <input v-model.number="draft.transport" type="number" min="0" step="0.01" placeholder="0" :class="[inputClass, 'mt-1']" @input="emit('touched')" />
      </label>
      <label class="text-xs text-gray-600">
        Other fees
        <input v-model.number="draft.other" type="number" min="0" step="0.01" placeholder="0" :class="[inputClass, 'mt-1']" @input="emit('touched')" />
      </label>
      <label class="text-xs text-gray-600">
        Other fees note
        <input v-model="draft.otherNote" maxlength="200" placeholder="e.g. parking, tools rent" :class="[inputClass, 'mt-1']" @input="emit('touched')" />
      </label>
    </div>

    <div class="mt-3 flex flex-wrap items-baseline justify-between gap-2 border-t border-gray-200 pt-3 text-sm">
      <span class="text-gray-600">Total cost <strong class="text-gray-900">{{ formatMoney(preview.cost) }}</strong></span>
      <span v-if="preview.profit != null" :class="['font-semibold', preview.profit < 0 ? 'text-red-600' : 'text-green-700']">
        Profit {{ formatMoney(preview.profit) }}<template v-if="preview.margin != null"> ({{ preview.margin }}%)</template>
      </span>
      <span v-else class="text-xs text-gray-500">Enter a price to see profit</span>
    </div>
    <p class="mt-1 text-[11px] text-gray-400">Supplies used on this job are added automatically after saving.</p>
  </div>
</template>
