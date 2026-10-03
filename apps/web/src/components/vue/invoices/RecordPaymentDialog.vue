<script setup lang="ts">
import { computed, onMounted, reactive, ref, useId } from "vue";
import { useModalA11y } from "../ui/useModalA11y";
import {
  collectPayment,
  recordInvoicePayment,
  type PaymentResult,
} from "../../../lib/invoices";
import {
  activePaymentMethods,
  getWorkspaceSettings,
  paymentMethodLabel,
} from "../../../lib/settings";
import { ApiError } from "../../../lib/api";
import VatBreakdown from "./VatBreakdown.vue";

/**
 * Records money received. With invoiceId it pays that invoice (full or
 * partial); with bookingId it bills the completed booking and records the
 * payment in one go (pay on completion). Either way a receipt is issued.
 */
const props = defineProps<{
  title: string;
  subtitle?: string;
  invoiceId?: number;
  bookingId?: number;
  /** Balance due, prefilled as the amount. */
  balance?: number;
  /** Total the customer pays (net of withholding tax), for the deposit
   * shortcut. */
  payable?: number;
  /** Booking price, when known (collect mode). */
  price?: number | null;
  currency: string;
}>();

const emit = defineEmits<{ done: [result: PaymentResult]; cancel: [] }>();

const titleId = useId();
const { container } = useModalA11y(() => emit("cancel"));

const fallbackMethods = ["cash", "bank_transfer", "promptpay", "credit_card"];
const methods = ref<string[]>(fallbackMethods);

const today = new Date();
const todayStr = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, "0")}-${String(today.getDate()).padStart(2, "0")}`;

const collecting = computed(() => !props.invoiceId && !!props.bookingId);

const form = reactive({
  amount: props.balance && props.balance > 0 ? props.balance : undefined as number | undefined,
  price: props.price ?? (undefined as number | undefined),
  method: "cash",
  reference: "",
  paidAt: todayStr,
});

// Deposit shortcut: half of what the customer pays, while more than that is
// still owed (i.e. before the deposit is in).
const DEPOSIT_PERCENT = 50;
const depositAmount = computed(() => {
  if (!props.payable || !props.balance) return 0;
  const d = Math.round(props.payable * DEPOSIT_PERCENT) / 100;
  return d < props.balance - 0.005 ? d : 0;
});

const busy = ref(false);
const error = ref("");

onMounted(async () => {
  try {
    await getWorkspaceSettings();
    const live = activePaymentMethods();
    if (live.length > 0) {
      methods.value = live;
      if (!live.includes(form.method)) form.method = live[0]!;
    }
  } catch {
    /* built-in methods */
  }
});

const partial = computed(
  () =>
    !!props.balance &&
    form.amount !== undefined &&
    form.amount > 0 &&
    form.amount < props.balance - 0.005,
);

async function submit() {
  error.value = "";
  // An emptied number input yields "": that means "the full amount".
  if ((form.amount as unknown) === "") form.amount = undefined;
  if ((form.price as unknown) === "") form.price = undefined;
  if (form.amount !== undefined && !(form.amount > 0)) {
    error.value = "Amount must be greater than zero";
    return;
  }
  if (props.balance && form.amount && form.amount > props.balance + 0.005) {
    error.value = `Amount is more than the balance due (${props.currency} ${props.balance.toFixed(2)})`;
    return;
  }
  if (collecting.value && !props.price && !(form.price && form.price > 0)) {
    error.value = "Enter the job price";
    return;
  }
  busy.value = true;
  try {
    const payment = {
      amount: form.amount || undefined,
      method: form.method,
      reference: form.reference.trim() || undefined,
      paidAt: form.paidAt || undefined,
    };
    const result = props.invoiceId
      ? await recordInvoicePayment(props.invoiceId, payment)
      : await collectPayment({
          ...payment,
          bookingId: props.bookingId!,
          subtotal: !props.price && form.price ? Number(form.price) : undefined,
        });
    emit("done", result);
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : "Failed to record payment";
  } finally {
    busy.value = false;
  }
}

const inputClass =
  "w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500";
</script>

<template>
  <div
    ref="container"
    role="dialog"
    aria-modal="true"
    :aria-labelledby="titleId"
    class="fixed inset-0 z-50 flex items-center justify-center p-4"
  >
    <div class="absolute inset-0 bg-black/50" @click="emit('cancel')"></div>

    <div class="relative flex max-h-[calc(100dvh-2rem)] w-full max-w-md flex-col rounded-xl bg-white shadow-xl">
      <div class="border-b border-gray-200 px-6 py-4">
        <h2 :id="titleId" class="text-base font-semibold text-gray-900">{{ title }}</h2>
        <p v-if="subtitle" class="mt-0.5 text-sm text-gray-500">{{ subtitle }}</p>
      </div>

      <form class="flex min-h-0 flex-1 flex-col" @submit.prevent="submit">
        <div class="grid min-h-0 flex-1 grid-cols-1 gap-4 overflow-y-auto px-6 py-5 sm:grid-cols-2">
          <div v-if="collecting && !price" class="sm:col-span-2">
            <label class="mb-1 block text-sm font-medium text-gray-700" for="rp-price">
              Job price before VAT ({{ currency }})
            </label>
            <input
              id="rp-price"
              v-model.number="form.price"
              type="number"
              min="0"
              step="0.01"
              :class="inputClass"
              placeholder="Leave blank to use the catalog price"
            />
            <VatBreakdown class="mt-2" :price="form.price" :currency="currency" />
          </div>

          <div class="sm:col-span-2">
            <label class="mb-1 block text-sm font-medium text-gray-700" for="rp-amount">
              Amount received ({{ currency }})
            </label>
            <input
              id="rp-amount"
              v-model.number="form.amount"
              type="number"
              min="0"
              step="0.01"
              :class="inputClass"
              :placeholder="collecting ? 'Full amount' : ''"
            />
            <div v-if="balance && depositAmount" class="mt-1.5 flex flex-wrap gap-2">
              <button
                type="button"
                class="rounded-full border px-3 py-1 text-xs font-medium"
                :class="form.amount === depositAmount ? 'border-amber-400 bg-amber-50 text-amber-800' : 'border-gray-300 text-gray-700 hover:bg-gray-50'"
                @click="form.amount = depositAmount"
              >Deposit {{ DEPOSIT_PERCENT }}% · {{ depositAmount.toFixed(2) }}</button>
              <button
                type="button"
                class="rounded-full border px-3 py-1 text-xs font-medium"
                :class="form.amount === balance ? 'border-green-400 bg-green-50 text-green-800' : 'border-gray-300 text-gray-700 hover:bg-gray-50'"
                @click="form.amount = balance"
              >Full balance · {{ balance.toFixed(2) }}</button>
            </div>
            <p v-if="balance" class="mt-1 text-xs text-gray-500">
              Balance due {{ currency }} {{ balance.toFixed(2) }}
              <span v-if="partial" class="font-medium text-amber-700">· partial payment — the rest stays open</span>
            </p>
            <p v-else-if="collecting" class="mt-1 text-xs text-gray-500">
              Blank = full amount (price + VAT, less withholding tax if any).
            </p>
            <VatBreakdown v-if="collecting && price" class="mt-2" :price="price" :currency="currency" />
          </div>

          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700" for="rp-method">Method</label>
            <select id="rp-method" v-model="form.method" :class="inputClass">
              <option v-for="m in methods" :key="m" :value="m">{{ paymentMethodLabel(m) }}</option>
            </select>
          </div>

          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700" for="rp-date">Date received</label>
            <input id="rp-date" v-model="form.paidAt" type="date" :max="todayStr" :class="inputClass" />
          </div>

          <div class="sm:col-span-2">
            <label class="mb-1 block text-sm font-medium text-gray-700" for="rp-ref">Reference (optional)</label>
            <input
              id="rp-ref"
              v-model="form.reference"
              :class="inputClass"
              placeholder="Transfer slip no., cheque no.…"
            />
          </div>

          <p v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700 sm:col-span-2">{{ error }}</p>
        </div>

        <div class="flex justify-end gap-3 border-t border-gray-200 px-6 py-4">
          <button
            type="button"
            class="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100"
            @click="emit('cancel')"
          >Cancel</button>
          <button
            type="submit"
            :disabled="busy"
            class="rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50"
          >{{ busy ? "Saving…" : "Record payment & issue receipt" }}</button>
        </div>
      </form>
    </div>
  </div>
</template>
