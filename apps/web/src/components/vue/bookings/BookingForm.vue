<script setup lang="ts">
import { computed, reactive, ref, useId } from "vue";
import { useModalA11y } from "../ui/useModalA11y";
import type {
  Booking,
  BookingStatus,
  CreateBookingInput,
  ServiceType,
} from "../../../lib/bookings";

const props = defineProps<{
  booking?: Booking;
}>();

const emit = defineEmits<{
  save: [input: CreateBookingInput];
  cancel: [];
}>();

const titleId = useId();
const { container } = useModalA11y(() => emit("cancel"));

const statusOptions: { value: BookingStatus; label: string }[] = [
  { value: "pending", label: "Pending" },
  { value: "confirmed", label: "Confirmed" },
  { value: "in_progress", label: "In Progress" },
  { value: "completed", label: "Completed" },
  { value: "cancelled", label: "Cancelled" },
  { value: "no_show", label: "No Show" },
  { value: "rescheduled", label: "Rescheduled" },
];

const serviceTypeOptions: { value: ServiceType; label: string }[] = [
  { value: "house_cleaning", label: "House Cleaning" },
  { value: "condo_cleaning", label: "Condo Cleaning" },
  { value: "deep_cleaning", label: "Deep Cleaning" },
  { value: "move_in_out", label: "Move In/Out" },
  { value: "after_renovation", label: "After Renovation" },
  { value: "office_cleaning", label: "Office Cleaning" },
  { value: "junk_removal", label: "Junk Removal" },
  { value: "aircon_service", label: "Aircon Service" },
];

function toDatetimeLocal(iso: string): string {
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

const form = reactive<CreateBookingInput>({
  customerName: props.booking?.customerName ?? "",
  serviceType: props.booking?.serviceType ?? "house_cleaning",
  scheduledFor: props.booking?.scheduledFor
    ? toDatetimeLocal(props.booking.scheduledFor)
    : "",
  durationMinutes: props.booking?.durationMinutes ?? 180,
  address: props.booking?.address ?? "",
  assignedCleaner: props.booking?.assignedCleaner ?? "",
  status: props.booking?.status ?? "pending",
  notes: props.booking?.notes ?? "",
});

const submitted = ref(false);

const errors = computed(() => {
  const e: Partial<Record<keyof CreateBookingInput, string>> = {};
  if (!form.customerName.trim())
    e.customerName = "Customer name is required";
  if (!form.scheduledFor.trim()) e.scheduledFor = "Scheduled date is required";
  if (!form.durationMinutes || form.durationMinutes <= 0)
    e.durationMinutes = "Duration must be greater than zero";
  if (!form.address.trim()) e.address = "Address is required";
  if (!form.assignedCleaner.trim())
    e.assignedCleaner = "Assigned cleaner is required";
  return e;
});

const isValid = computed(() => Object.keys(errors.value).length === 0);

function handleSubmit() {
  submitted.value = true;
  if (!isValid.value) return;
  const scheduledFor = form.scheduledFor
    ? new Date(form.scheduledFor).toISOString()
    : "";
  emit("save", {
    ...form,
    scheduledFor,
    durationMinutes: Number(form.durationMinutes),
  });
}

const inputClass =
  "w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 placeholder-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500";

function inputClassFor(field: keyof CreateBookingInput) {
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
          {{ booking ? `Edit Booking ${booking.bookingNumber}` : "New Booking" }}
        </h2>
      </div>

      <form @submit.prevent="handleSubmit">
        <div class="grid grid-cols-1 gap-4 px-6 py-5 sm:grid-cols-2">
          <div class="sm:col-span-2">
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="b-customer"
              >Customer name</label
            >
            <input
              id="b-customer"
              v-model="form.customerName"
              :class="inputClassFor('customerName')"
              placeholder="Somchai Prasert"
            />
            <p
              v-if="submitted && errors.customerName"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.customerName }}
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="b-serviceType"
              >Service type</label
            >
            <select
              id="b-serviceType"
              v-model="form.serviceType"
              :class="inputClass"
            >
              <option
                v-for="opt in serviceTypeOptions"
                :key="opt.value"
                :value="opt.value"
              >
                {{ opt.label }}
              </option>
            </select>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="b-status"
              >Status</label
            >
            <select id="b-status" v-model="form.status" :class="inputClass">
              <option
                v-for="opt in statusOptions"
                :key="opt.value"
                :value="opt.value"
              >
                {{ opt.label }}
              </option>
            </select>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="b-scheduledFor"
              >Scheduled for</label
            >
            <input
              id="b-scheduledFor"
              v-model="form.scheduledFor"
              type="datetime-local"
              :class="inputClassFor('scheduledFor')"
            />
            <p
              v-if="submitted && errors.scheduledFor"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.scheduledFor }}
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="b-duration"
              >Duration (minutes)</label
            >
            <input
              id="b-duration"
              v-model.number="form.durationMinutes"
              type="number"
              min="1"
              :class="inputClassFor('durationMinutes')"
            />
            <p
              v-if="submitted && errors.durationMinutes"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.durationMinutes }}
            </p>
          </div>

          <div class="sm:col-span-2">
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="b-address"
              >Address</label
            >
            <input
              id="b-address"
              v-model="form.address"
              :class="inputClassFor('address')"
              placeholder="Sukhumvit 38, Klongton, Bangkok"
            />
            <p
              v-if="submitted && errors.address"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.address }}
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="b-cleaner"
              >Assigned cleaner</label
            >
            <input
              id="b-cleaner"
              v-model="form.assignedCleaner"
              :class="inputClassFor('assignedCleaner')"
              placeholder="Nok Srisuwan"
            />
            <p
              v-if="submitted && errors.assignedCleaner"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.assignedCleaner }}
            </p>
          </div>

          <div class="sm:col-span-2">
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="b-notes"
              >Notes</label
            >
            <textarea
              id="b-notes"
              v-model="form.notes"
              rows="3"
              :class="inputClass"
              placeholder="Any additional notes..."
            ></textarea>
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
            {{ booking ? "Save changes" : "Create booking" }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
