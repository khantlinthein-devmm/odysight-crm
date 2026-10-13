<script setup lang="ts">
import { computed, ref } from "vue";
import { formatMoney } from "../../../lib/settings";
import { previewProfit, type CostDraft } from "../../../lib/booking-profit";

// "Job costs & profit" block of the booking form. The cleaners fee is either
// one amount for the whole team on this job (lead + everyone who came, from
// any team) or a fee per cleaner; plus transportation and other costs. The
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

function num(v: number | "" | undefined): number {
  return typeof v === "number" && Number.isFinite(v) ? v : 0;
}

const leadName = computed(() => props.team.find((m) => m.role === "primary")?.name ?? "");
const crewCount = computed(() => props.team.filter((m) => m.role === "crew").length);

const preview = computed(() =>
  previewProfit(props.price, {
    teamFee: draft.value.mode === "team" ? num(draft.value.teamFee) : null,
    cleanerFees: Object.fromEntries(props.team.map((m) => [m.id, num(draft.value.fees[m.id])])),
    transport: num(draft.value.transport),
    other: num(draft.value.other),
    otherNote: draft.value.otherNote,
  }),
);

function setMode(mode: CostDraft["mode"]) {
  draft.value.mode = mode;
  emit("touched");
}

// Quick fill: fee per hour × job hours (whole team, or each cleaner).
function applyHourly() {
  const rate = num(hourlyFee.value);
  if (rate <= 0) return;
  const fee = Math.round(rate * props.hours * 100) / 100;
  if (draft.value.mode === "team") draft.value.teamFee = fee;
  else for (const m of props.team) draft.value.fees[m.id] = fee;
  emit("touched");
}
</script>

<template>
  <div class="rounded-lg border border-gray-200 bg-gray-50 p-4">
    <div class="mb-3 flex flex-wrap items-baseline justify-between gap-2">
      <span class="text-sm font-semibold text-gray-900">Job costs &amp; profit</span>
      <span class="text-xs text-gray-500">Only office staff see this</span>
    </div>

    <div class="mb-3 inline-flex rounded-lg bg-white p-0.5 ring-1 ring-gray-200" role="radiogroup" aria-label="How cleaners are paid">
      <button
        type="button"
        role="radio"
        :aria-checked="draft.mode === 'team'"
        :class="['rounded-md px-3 py-1 text-xs font-medium', draft.mode === 'team' ? 'bg-navy-600 text-white' : 'text-gray-600 hover:bg-gray-100']"
        @click="setMode('team')"
      >
        One fee for the team
      </button>
      <button
        type="button"
        role="radio"
        :aria-checked="draft.mode === 'each'"
        :class="['rounded-md px-3 py-1 text-xs font-medium', draft.mode === 'each' ? 'bg-navy-600 text-white' : 'text-gray-600 hover:bg-gray-100']"
        @click="setMode('each')"
      >
        Fee per cleaner
      </button>
    </div>

    <template v-if="draft.mode === 'team'">
      <label class="block text-xs text-gray-600">
        Cleaners fee (whole team)
        <input
          v-model.number="draft.teamFee"
          type="number"
          min="0"
          step="0.01"
          placeholder="e.g. 2000"
          :class="[inputClass, 'mt-1 sm:w-48']"
          @input="emit('touched')"
        />
      </label>
      <p class="mt-1 text-xs text-gray-500">
        <template v-if="leadName">Lead {{ leadName }}<template v-if="crewCount"> + {{ crewCount }} crew</template> — the lead shares it with the team.</template>
        <template v-else>Pick the lead cleaner above.</template>
      </p>
    </template>

    <template v-else>
      <div v-if="team.length" class="space-y-2">
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
    </template>

    <div class="mt-2 flex flex-wrap items-end gap-2">
      <label class="text-xs text-gray-600">
        Or: fee per hour
        <input
          v-model.number="hourlyFee"
          type="number"
          min="0"
          step="1"
          placeholder="e.g. 500"
          class="mt-1 block w-28 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-sm"
        />
      </label>
      <button
        type="button"
        class="rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-xs font-medium text-gray-700 hover:bg-gray-100"
        @click="applyHourly"
      >
        × {{ hours }}h = fill {{ draft.mode === "team" ? "team fee" : "each cleaner" }}
      </button>
    </div>

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
