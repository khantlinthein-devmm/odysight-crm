<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  activeServices,
  formatMoney,
  getWorkspaceSettings,
  type ServiceItem,
} from "../../../lib/settings";
import {
  buildQuoteSummary,
  calculateQuote,
  newFeeId,
  type QuoteExtraFee,
  type QuoteServiceLine,
} from "../../../lib/pricing";
import { showToast } from "../../../lib/toast";

const props = withDefaults(
  defineProps<{ initialServiceId?: string; showHeader?: boolean }>(),
  { initialServiceId: "", showHeader: true },
);

const services = ref<ServiceItem[]>([]);
const loading = ref(true);
const copied = ref(false);

const lines = ref<QuoteServiceLine[]>([]);
const equipmentFee = ref(0);
const transportFee = ref(0);
const labourFee = ref(0);
const extraFees = ref<QuoteExtraFee[]>([]);
const discount = ref(0);
const taxRate = ref(7);

const newFeeLabel = ref("");
const newFeeAmount = ref(0);

function catalogById(id: string): ServiceItem | undefined {
  return services.value.find((s) => s.id === id);
}

function blankLine(serviceId?: string): QuoteServiceLine {
  const found = serviceId ? catalogById(serviceId) : services.value[0];
  return {
    serviceId: found?.id ?? "",
    serviceName: found?.name ?? "",
    basePrice: found?.basePrice ?? 0,
    areaSqm: 0,
    pricePerSqm: found?.pricePerSqm ?? 0,
  };
}

function applyCatalogDefaults(index: number) {
  const found = catalogById(lines.value[index]!.serviceId);
  if (!found) return;
  lines.value[index]!.serviceName = found.name;
  lines.value[index]!.basePrice = found.basePrice;
  lines.value[index]!.pricePerSqm = found.pricePerSqm;
}

function addLine() {
  lines.value.push(blankLine());
}

function removeLine(index: number) {
  lines.value.splice(index, 1);
}

function addExtraFee() {
  extraFees.value.push({
    id: newFeeId(),
    label: newFeeLabel.value.trim() || "Extra fee",
    amount: Number(newFeeAmount.value) || 0,
  });
  newFeeLabel.value = "";
  newFeeAmount.value = 0;
}

function removeExtraFee(id: string) {
  extraFees.value = extraFees.value.filter((f) => f.id !== id);
}

function reset() {
  lines.value = services.value.length > 0 ? [blankLine()] : [];
  equipmentFee.value = 0;
  transportFee.value = 0;
  labourFee.value = 0;
  extraFees.value = [];
  discount.value = 0;
}

onMounted(async () => {
  try {
    const ws = await getWorkspaceSettings();
    services.value = ws.services.filter((s) => s.active);
    taxRate.value = ws.payments.taxRatePercent ?? 7;
  } catch {
    services.value = activeServices();
  } finally {
    const first = props.initialServiceId || services.value[0]?.id;
    lines.value = [blankLine(first)];
    loading.value = false;
  }
});

const result = computed(() =>
  calculateQuote(
    {
      lines: lines.value,
      equipmentFee: Number(equipmentFee.value) || 0,
      transportFee: Number(transportFee.value) || 0,
      labourFee: Number(labourFee.value) || 0,
      extraFees: extraFees.value,
      discount: Number(discount.value) || 0,
      taxRatePercent: Number(taxRate.value) || 0,
    },
    (id) => catalogById(id)?.durationMinutes ?? 0,
  ),
);

async function copySummary() {
  const text = buildQuoteSummary(result.value, (n) => formatMoney(n));
  try {
    await navigator.clipboard.writeText(text);
    copied.value = true;
    showToast("Quote copied to clipboard", "success");
    setTimeout(() => (copied.value = false), 2000);
  } catch {
    showToast("Copy failed — select the total manually", "error");
  }
}

const input =
  "w-full rounded-xl border-0 bg-white px-3.5 py-2.5 text-sm text-gray-900 shadow-sm ring-1 ring-gray-200 transition focus:outline-none focus:ring-2 focus:ring-navy-500";
const label = "mb-1.5 block text-xs font-medium text-gray-500";
const stepBadge =
  "flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-navy-600 to-blue-500 text-xs font-semibold text-white shadow-sm";
</script>

<template>
  <div class="grid grid-cols-1 items-start gap-5 xl:grid-cols-3">
    <div class="space-y-5 xl:col-span-2">
      <div
        v-if="showHeader"
        class="relative overflow-hidden rounded-3xl bg-gradient-to-br from-navy-800 via-navy-600 to-blue-500 p-7 text-white shadow-lg"
      >
        <div
          class="pointer-events-none absolute -right-16 -top-16 h-56 w-56 rounded-full bg-white/10 blur-2xl"
        ></div>
        <div
          class="pointer-events-none absolute -bottom-20 left-1/3 h-48 w-48 rounded-full bg-blue-300/20 blur-2xl"
        ></div>
        <p class="relative text-xs font-medium uppercase tracking-widest text-blue-100">
          Instant quote
        </p>
        <h2 class="relative mt-1 text-2xl font-semibold tracking-tight">
          Price calculator
        </h2>
        <p class="relative mt-2 max-w-xl text-sm leading-relaxed text-blue-50/90">
          Combine any services with room size (m²), then add equipment,
          transport, labour and any custom fees. Everything is editable per
          quote — catalog values are just starting points.
        </p>
        <div class="relative mt-4 inline-flex items-center gap-2 rounded-full bg-white/15 px-4 py-1.5 text-sm font-semibold backdrop-blur">
          <span class="h-2 w-2 animate-pulse rounded-full bg-emerald-300"></span>
          Total · {{ formatMoney(result.total) }}
        </div>
      </div>

      <div
        v-if="loading"
        class="rounded-3xl bg-white p-8 text-sm text-gray-500 shadow-sm ring-1 ring-gray-100"
      >
        <div class="h-4 w-40 animate-pulse rounded-full bg-gray-100"></div>
        <div class="mt-3 h-4 w-2/3 animate-pulse rounded-full bg-gray-100"></div>
      </div>

      <section
        v-else
        class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7"
      >
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex items-center gap-3">
            <span :class="stepBadge">1</span>
            <div>
              <h3 class="font-semibold tracking-tight text-gray-900">Services &amp; area</h3>
              <p class="text-xs text-gray-500">Pick services, set the m², fine-tune prices</p>
            </div>
          </div>
          <button
            type="button"
            class="rounded-xl bg-navy-50 px-4 py-2 text-xs font-semibold text-navy-700 ring-1 ring-navy-100 transition hover:bg-navy-100"
            @click="addLine"
          >
            + Add service
          </button>
        </div>

        <div
          v-for="(line, i) in lines"
          :key="i"
          class="mt-4 rounded-2xl bg-gradient-to-b from-gray-50 to-white p-4 ring-1 ring-gray-100 sm:p-5"
        >
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <div class="sm:col-span-2 lg:col-span-4">
              <label :class="label">Service {{ i + 1 }}</label>
              <div class="flex gap-2">
                <select v-model="line.serviceId" :class="input" @change="applyCatalogDefaults(i)">
                  <option v-for="s in services" :key="s.id" :value="s.id">
                    {{ s.name }} (base {{ formatMoney(s.basePrice) }})
                  </option>
                </select>
                <button
                  v-if="lines.length > 1"
                  type="button"
                  class="shrink-0 rounded-xl bg-red-50 px-3 py-2 text-xs font-semibold text-red-600 ring-1 ring-red-100 transition hover:bg-red-100"
                  @click="removeLine(i)"
                >
                  Remove
                </button>
              </div>
            </div>
            <div>
              <label :class="label">Area (m²)</label>
              <input v-model.number="line.areaSqm" type="number" min="0" step="1" :class="input" placeholder="e.g. 80" />
            </div>
            <div>
              <label :class="label">Base price</label>
              <input v-model.number="line.basePrice" type="number" min="0" step="0.01" :class="input" />
            </div>
            <div>
              <label :class="label">Rate per m²</label>
              <input v-model.number="line.pricePerSqm" type="number" min="0" step="0.01" :class="input" />
            </div>
            <div>
              <label :class="label">Line total</label>
              <p class="rounded-xl bg-navy-600/5 px-3.5 py-2.5 text-sm font-semibold text-navy-700 ring-1 ring-navy-100">
                {{ formatMoney(result.lines[i]?.lineTotal ?? 0) }}
              </p>
            </div>
          </div>
          <p class="mt-2.5 text-xs text-gray-400">
            {{ formatMoney(Number(line.basePrice) || 0) }} base + {{ Number(line.areaSqm) || 0 }} m² ×
            {{ formatMoney(Number(line.pricePerSqm) || 0) }}/m²
          </p>
        </div>
      </section>

      <section class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7">
        <div class="flex items-center gap-3">
          <span :class="stepBadge">2</span>
          <div>
            <h3 class="font-semibold tracking-tight text-gray-900">Fees &amp; adjustments</h3>
            <p class="text-xs text-gray-500">Fixed costs, extras, discount and tax</p>
          </div>
        </div>

        <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
          <div class="rounded-2xl bg-gray-50/70 p-3 ring-1 ring-gray-100">
            <label class="mb-1.5 block text-xs font-medium text-gray-500" for="q-equip">🔧 Equipment</label>
            <input id="q-equip" v-model.number="equipmentFee" type="number" min="0" step="0.01" :class="input" />
          </div>
          <div class="rounded-2xl bg-gray-50/70 p-3 ring-1 ring-gray-100">
            <label class="mb-1.5 block text-xs font-medium text-gray-500" for="q-transport">🚚 Transportation</label>
            <input id="q-transport" v-model.number="transportFee" type="number" min="0" step="0.01" :class="input" />
          </div>
          <div class="rounded-2xl bg-gray-50/70 p-3 ring-1 ring-gray-100">
            <label class="mb-1.5 block text-xs font-medium text-gray-500" for="q-labour">👷 Labour</label>
            <input id="q-labour" v-model.number="labourFee" type="number" min="0" step="0.01" :class="input" />
          </div>
        </div>

        <div class="mt-5">
          <p class="text-xs font-semibold uppercase tracking-widest text-gray-400">Other fees</p>
          <div v-if="extraFees.length > 0" class="mt-2.5 space-y-2">
            <div v-for="f in extraFees" :key="f.id" class="flex items-center gap-2">
              <input v-model="f.label" :class="input" placeholder="Fee name" />
              <input v-model.number="f.amount" type="number" min="0" step="0.01" :class="input + ' max-w-36'" />
              <button
                type="button"
                class="shrink-0 rounded-xl bg-red-50 px-3 py-2.5 text-xs font-semibold text-red-600 ring-1 ring-red-100 transition hover:bg-red-100"
                @click="removeExtraFee(f.id)"
              >
                ✕
              </button>
            </div>
          </div>
          <div class="mt-2.5 flex gap-2">
            <input v-model="newFeeLabel" :class="input" placeholder="e.g. Parking, stain removal, weekend surcharge" />
            <input v-model.number="newFeeAmount" type="number" min="0" step="0.01" :class="input + ' max-w-36'" placeholder="Amount" />
            <button
              type="button"
              class="shrink-0 rounded-xl bg-gray-900 px-4 py-2.5 text-xs font-semibold text-white transition hover:bg-gray-700"
              @click="addExtraFee"
            >
              Add
            </button>
          </div>
        </div>

        <div class="mt-5 grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <label class="mb-1.5 block text-xs font-medium text-gray-500" for="q-discount">Discount</label>
            <input id="q-discount" v-model.number="discount" type="number" min="0" step="0.01" :class="input" />
          </div>
          <div>
            <label class="mb-1.5 block text-xs font-medium text-gray-500" for="q-tax">Tax %</label>
            <input id="q-tax" v-model.number="taxRate" type="number" min="0" max="100" step="0.01" :class="input" />
          </div>
        </div>
      </section>
    </div>

    <div class="xl:col-span-1">
      <div
        class="relative overflow-hidden rounded-3xl bg-gray-950 p-6 text-white shadow-xl ring-1 ring-gray-900 xl:sticky xl:top-4 sm:p-7"
      >
        <div
          class="pointer-events-none absolute -right-14 -top-14 h-48 w-48 rounded-full bg-navy-500/40 blur-3xl"
        ></div>
        <div
          class="pointer-events-none absolute -bottom-16 -left-10 h-44 w-44 rounded-full bg-blue-500/20 blur-3xl"
        ></div>
        <p class="relative text-xs font-medium uppercase tracking-widest text-gray-400">
          Quote summary
        </p>
        <dl class="relative mt-4 space-y-2.5 text-sm">
          <div
            v-for="(l, i) in result.lines"
            :key="i"
            class="flex items-start justify-between gap-2"
          >
            <dt class="min-w-0 text-gray-300">
              {{ l.serviceName || "Service" }}
              <span v-if="l.areaSqm > 0" class="text-gray-500">· {{ l.areaSqm }} m²</span>
            </dt>
            <dd class="shrink-0 font-medium text-white">{{ formatMoney(l.lineTotal) }}</dd>
          </div>
          <div class="flex justify-between border-t border-white/10 pt-2.5 text-gray-300">
            <dt>Fees &amp; extras</dt>
            <dd class="font-medium text-white">{{ formatMoney(result.feesTotal) }}</dd>
          </div>
          <div class="flex justify-between text-gray-300">
            <dt>Subtotal</dt>
            <dd class="font-medium text-white">{{ formatMoney(result.subtotal) }}</dd>
          </div>
          <div v-if="result.discount > 0" class="flex justify-between text-gray-300">
            <dt>Discount</dt>
            <dd class="font-medium text-emerald-300">−{{ formatMoney(result.discount) }}</dd>
          </div>
          <div class="flex justify-between text-gray-300">
            <dt>Tax ({{ Number(taxRate) || 0 }}%)</dt>
            <dd class="font-medium text-white">{{ formatMoney(result.tax) }}</dd>
          </div>
        </dl>
        <div class="relative mt-4 rounded-2xl bg-white/10 p-4 backdrop-blur">
          <p class="text-xs uppercase tracking-widest text-gray-400">Total</p>
          <p class="mt-0.5 bg-gradient-to-r from-blue-200 via-white to-blue-200 bg-clip-text text-3xl font-semibold tracking-tight text-transparent">
            {{ formatMoney(result.total) }}
          </p>
          <p v-if="result.estimatedMinutes > 0" class="mt-1 text-xs text-gray-400">
            ⏱ Estimated {{ Math.floor(result.estimatedMinutes / 60) }}h
            {{ result.estimatedMinutes % 60 }}m
          </p>
        </div>
        <div class="relative mt-4 flex gap-2">
          <button
            type="button"
            class="flex-1 rounded-xl bg-gradient-to-r from-navy-500 to-blue-500 px-4 py-2.5 text-sm font-semibold text-white shadow-lg shadow-navy-950/40 transition hover:brightness-110 active:scale-[0.99]"
            @click="copySummary"
          >
            {{ copied ? "Copied ✓" : "Copy quote" }}
          </button>
          <button
            type="button"
            class="rounded-xl bg-white/10 px-4 py-2.5 text-sm font-medium text-gray-200 ring-1 ring-white/15 transition hover:bg-white/20"
            @click="reset"
          >
            Reset
          </button>
        </div>
        <a
          href="/bookings"
          class="relative mt-3 block text-center text-xs text-gray-400 hover:text-white hover:underline"
        >
          Use this total when creating the booking →
        </a>
      </div>
    </div>
  </div>
</template>
