<script setup lang="ts">
import { computed, onMounted, reactive, ref, useId } from "vue";
import { useModalA11y } from "../ui/useModalA11y";
import { getBookings, type Booking } from "../../../lib/bookings";
import { getCleaners, type Cleaner } from "../../../lib/cleaners";
import {
  serviceLabel,
} from "../../../lib/settings";
import type {
  ServiceRecord,
  ServiceRecordStatus,
  CreateServiceRecordInput,
} from "../../../lib/service-records";

const props = defineProps<{
  record?: ServiceRecord;
}>();

const emit = defineEmits<{
  save: [input: CreateServiceRecordInput];
  cancel: [];
}>();

const titleId = useId();
const { container } = useModalA11y(() => emit("cancel"));

const statusOptions: { value: ServiceRecordStatus; label: string }[] = [
  { value: "pending", label: "Pending" },
  { value: "completed", label: "Completed" },
  { value: "rescheduled", label: "Rescheduled" },
  { value: "cancelled", label: "Cancelled" },
];

const serviceTypes = [
  "House Cleaning",
  "Condo Cleaning",
  "Deep Cleaning",
  "Move In/Out",
  "After Renovation",
  "Office Cleaning",
  "Junk Removal",
  "Aircon Service",
];

const form = reactive<CreateServiceRecordInput>({
  bookingNumber: props.record?.bookingNumber ?? "",
  cleanerName: props.record?.cleanerName ?? "",
  serviceType: props.record?.serviceType ?? serviceTypes[0],
  rating: props.record?.rating ?? null,
  status: props.record?.status ?? "pending",
  notes: props.record?.notes ?? "",
});

const submitted = ref(false);

const bookings = ref<Booking[]>([]);
const cleaners = ref<Cleaner[]>([]);
const bookingChoice = ref("");
const cleanerChoice = ref("");

function fullName(c: Cleaner): string {
  return `${c.firstName} ${c.lastName}`.trim();
}

function applyBookingChoice(val: string) {
  if (val === "" || val === "manual") return;
  const b = bookings.value.find((x) => x.bookingNumber === val);
  if (b) form.bookingNumber = b.bookingNumber;
}

function onBookingChoice(ev: Event) {
  const val = (ev.target as HTMLSelectElement).value;
  bookingChoice.value = val;
  applyBookingChoice(val);
}

function onCleanerChoice(ev: Event) {
  const val = (ev.target as HTMLSelectElement).value;
  cleanerChoice.value = val;
  if (val !== "" && val !== "manual") form.cleanerName = val;
}

onMounted(async () => {
  try {
    bookings.value = await getBookings({ limit: 200 });
  } catch {
    /* keep empty list; manual entry still works */
  }
  try {
    cleaners.value = await getCleaners();
  } catch {
    /* keep empty list; manual entry still works */
  }
  if (props.record) {
    const rec = props.record;
    const b = bookings.value.find((x) => x.bookingNumber === rec.bookingNumber);
    bookingChoice.value = b ? b.bookingNumber : "manual";
    const c = cleaners.value.find((x) => fullName(x) === rec.cleanerName);
    cleanerChoice.value = c ? fullName(c) : "manual";
  }
});

const errors = computed(() => {
  const e: Partial<Record<keyof CreateServiceRecordInput, string>> = {};
  if (!form.bookingNumber.trim())
    e.bookingNumber = "Booking number is required";
  if (!form.cleanerName.trim()) e.cleanerName = "Cleaner name is required";
  return e;
});

const isValid = computed(() => Object.keys(errors.value).length === 0);

function handleSubmit() {
  submitted.value = true;
  if (!isValid.value) return;
  emit("save", { ...form, rating: form.rating ? Number(form.rating) : null });
}

const inputClass =
  "w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500";

function inputClassFor(field: keyof CreateServiceRecordInput) {
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
    <div class="absolute inset-0 bg-black/50" @click="emit('cancel')"></div>

    <div class="relative w-full max-w-lg rounded-xl bg-white shadow-xl">
      <div class="border-b border-gray-200 px-6 py-4">
        <h2 :id="titleId" class="text-base font-semibold text-gray-900">
          {{ record ? "Edit Service Record" : "Upload Service Record" }}
        </h2>
      </div>

      <form @submit.prevent="handleSubmit">
        <div class="grid grid-cols-1 gap-4 px-6 py-5 sm:grid-cols-2">
          <div class="sm:col-span-2">
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="sr-booking-choice"
              >Booking</label
            >
            <select
              id="sr-booking-choice"
              :value="bookingChoice"
              :class="inputClass"
              @change="onBookingChoice"
            >
              <option value="" disabled>Select a booking…</option>
              <option value="manual">Manual entry</option>
              <option
                v-for="b in bookings"
                :key="b.bookingNumber"
                :value="b.bookingNumber"
              >
                {{ b.bookingNumber }} — {{ b.customerName }} ·
                {{ serviceLabel(b.serviceType) }}
              </option>
            </select>
            <p class="mt-1 text-xs text-gray-400">
              Pick an existing booking to auto-fill its number, or choose
              “Manual entry” to type it.
            </p>
            <input
              id="sr-booking"
              v-model="form.bookingNumber"
              :class="[...inputClassFor('bookingNumber'), 'mt-2']"
              :disabled="bookingChoice !== '' && bookingChoice !== 'manual'"
              placeholder="BK-2026-0201"
            />
            <p
              v-if="submitted && errors.bookingNumber"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.bookingNumber }}
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="sr-cleaner-choice"
              >Cleaner</label
            >
            <select
              id="sr-cleaner-choice"
              :value="cleanerChoice"
              :class="inputClass"
              @change="onCleanerChoice"
            >
              <option value="" disabled>Choose a cleaner…</option>
              <option value="manual">Manual entry</option>
              <option v-for="c in cleaners" :key="c.id" :value="fullName(c)">
                {{ fullName(c) }}
              </option>
            </select>
            <p class="mt-1 text-xs text-gray-400">
              Pick a cleaner or choose “Manual entry”.
            </p>
            <input
              id="sr-cleaner"
              v-model="form.cleanerName"
              :class="[...inputClassFor('cleanerName'), 'mt-2']"
              :disabled="cleanerChoice !== '' && cleanerChoice !== 'manual'"
              placeholder="Nok Srisuwan"
            />
            <p
              v-if="submitted && errors.cleanerName"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.cleanerName }}
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="sr-serviceType"
              >Service type</label
            >
            <select
              id="sr-serviceType"
              v-model="form.serviceType"
              :class="inputClass"
            >
              <option v-for="st in serviceTypes" :key="st" :value="st">
                {{ st }}
              </option>
            </select>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="sr-rating"
              >Rating (1-5)</label
            >
            <input
              id="sr-rating"
              v-model.number="form.rating"
              type="number"
              min="1"
              max="5"
              :class="inputClass"
              placeholder="—"
            />
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="sr-status"
              >Status</label
            >
            <select id="sr-status" v-model="form.status" :class="inputClass">
              <option
                v-for="opt in statusOptions"
                :key="opt.value"
                :value="opt.value"
              >
                {{ opt.label }}
              </option>
            </select>
          </div>

          <div class="sm:col-span-2">
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="sr-notes"
              >Notes</label
            >
            <textarea
              id="sr-notes"
              v-model="form.notes"
              rows="3"
              :class="inputClass"
              placeholder="Any additional notes..."
            ></textarea>
          </div>
        </div>

        <div class="flex justify-end gap-3 border-t border-gray-200 px-6 py-4">
          <button
            type="button"
            class="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100"
            @click="emit('cancel')"
          >
            Cancel
          </button>
          <button
            type="submit"
            class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700"
          >
            {{ record ? "Save changes" : "Add record" }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
