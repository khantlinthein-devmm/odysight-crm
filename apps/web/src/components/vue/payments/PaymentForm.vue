<script setup lang="ts">
import { computed, reactive, ref, useId } from "vue";
import { useModalA11y } from "../ui/useModalA11y";
import type { CreatePaymentInput } from "../../../lib/payments";

const emit = defineEmits<{
  save: [input: CreatePaymentInput];
  cancel: [];
}>();

const titleId = useId();
const { container } = useModalA11y(() => emit("cancel"));

const paymentMethods = [
  "Credit Card",
  "Bank Transfer",
  "Cash",
  "Online Wallet",
];

const form = reactive<CreatePaymentInput>({
  payerName: "",
  amount: 0,
  currency: "USD",
  method: paymentMethods[0],
  status: "pending",
});

const submitted = ref(false);

const errors = computed(() => {
  const e: Partial<Record<keyof CreatePaymentInput, string>> = {};
  if (!form.payerName.trim()) e.payerName = "Payer name is required";
  if (!form.amount || form.amount <= 0)
    e.amount = "Amount must be greater than zero";
  return e;
});

const isValid = computed(() => Object.keys(errors.value).length === 0);

function handleSubmit() {
  submitted.value = true;
  if (!isValid.value) return;
  emit("save", { ...form, amount: Number(form.amount) });
}

const inputClass =
  "w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 placeholder-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500";

function inputClassFor(field: keyof CreatePaymentInput) {
  return [
    inputClass,
    submitted.value && errors.value[field]
      ? "border-red-300 focus:border-red-500 focus:ring-red-500"
      : "",
  ];
}
</script>

<template>
  <div
    ref="container"
    role="dialog"
    aria-modal="true"
    :aria-labelledby="titleId"
    class="fixed inset-0 z-50 flex items-center justify-center p-4"
  >
    <div class="absolute inset-0 bg-slate-900/50" @click="emit('cancel')"></div>

    <div class="relative w-full max-w-lg rounded-xl bg-white shadow-xl">
      <div class="border-b border-slate-200 px-6 py-4">
        <h2 :id="titleId" class="text-base font-semibold text-slate-900">
          New Payment
        </h2>
      </div>

      <form @submit.prevent="handleSubmit">
        <div class="grid grid-cols-1 gap-4 px-6 py-5 sm:grid-cols-2">
          <div class="sm:col-span-2">
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="p-payer"
              >Payer name</label
            >
            <input
              id="p-payer"
              v-model="form.payerName"
              :class="inputClassFor('payerName')"
              placeholder="John Doe"
            />
            <p
              v-if="submitted && errors.payerName"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.payerName }}
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="p-amount"
              >Amount</label
            >
            <input
              id="p-amount"
              v-model.number="form.amount"
              type="number"
              min="0"
              step="0.01"
              :class="inputClassFor('amount')"
              placeholder="0.00"
            />
            <p
              v-if="submitted && errors.amount"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.amount }}
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="p-currency"
              >Currency</label
            >
            <select id="p-currency" v-model="form.currency" :class="inputClass">
              <option>USD</option>
              <option>EUR</option>
              <option>GBP</option>
            </select>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="p-method"
              >Method</label
            >
            <select id="p-method" v-model="form.method" :class="inputClass">
              <option v-for="m in paymentMethods" :key="m" :value="m">
                {{ m }}
              </option>
            </select>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="p-status"
              >Status</label
            >
            <select id="p-status" v-model="form.status" :class="inputClass">
              <option value="pending">Pending</option>
              <option value="paid">Paid</option>
            </select>
          </div>
        </div>

        <div class="flex justify-end gap-3 border-t border-slate-200 px-6 py-4">
          <button
            type="button"
            class="rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
            @click="emit('cancel')"
          >
            Cancel
          </button>
          <button
            type="submit"
            class="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
          >
            Create payment
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
