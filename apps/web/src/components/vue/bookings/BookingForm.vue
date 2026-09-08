<script setup lang="ts">
import { computed, onMounted, reactive, ref, useId } from "vue";
import { useModalA11y } from "../ui/useModalA11y";
import { getCustomers, type Customer } from "../../../lib/customers";
import type { Cleaner } from "../../../lib/cleaners";
import { getCleaners } from "../../../lib/cleaners";
import type {
  Booking,
  BookingStatus,
  CreateBookingInput,
  RecurrenceFreq,
} from "../../../lib/bookings";
import {
  activeServices,
  getWorkspaceSettings,
} from "../../../lib/settings";

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

const recurrenceOptions: { value: RecurrenceFreq | ""; label: string }[] = [
  { value: "", label: "One-time" },
  { value: "weekly", label: "Every week" },
  { value: "biweekly", label: "Every 2 weeks" },
  { value: "monthly", label: "Every month" },
];

const serviceTypeOptions = computed(() => {
  const list = activeServices();
  if (list.length > 0) return list.map((s) => ({ value: s.id, label: s.name }));
  return [{ value: "house_cleaning", label: "House Cleaning" }];
});

function fullName(c: Cleaner): string {
  return `${c.firstName} ${c.lastName}`;
}

function applyServiceDuration(fallback?: number) {
  // New bookings only: prefill the catalog's default duration.
  if (props.booking) return;
  const found = activeServices().find((s) => s.id === form.serviceType);
  form.durationMinutes = found?.durationMinutes ?? fallback ?? form.durationMinutes;
}

const cleaners = ref<Cleaner[]>([]);
const cleanersLoading = ref(true);
const customers = ref<Customer[]>([]);
const customersLoading = ref(false);
const selectedCustomerId = ref<number | null>(props.booking?.customerId ?? null);

function customerFullName(c: Customer): string {
  return `${c.firstName} ${c.lastName}`;
}

function applyCustomer(customerId: number | string) {
  const raw = typeof customerId === "string" ? Number(customerId) : customerId;
  selectedCustomerId.value = raw || null;
  form.customerId = selectedCustomerId.value;
  const customer = customers.value.find((c) => c.id === raw);
  if (!customer) return;
  form.customerName = customerFullName(customer);
  form.customerEmail = customer.email;
  form.address = form.address || customer.address || "";
}

onMounted(async () => {
  try {
    const ws = await getWorkspaceSettings();
    if (!props.booking) {
      if (!form.serviceType && ws.services.length > 0) {
        form.serviceType = ws.services[0]!.id;
      }
      applyServiceDuration(ws.booking.defaultDurationMinutes);
    }
  } catch {
    /* fall back to built-in defaults */
  }
  customersLoading.value = true;
  try {
    customers.value = await getCustomers({ limit: 200 });
  } catch {
    /* keep empty list so the input falls back to free-text behavior */
  } finally {
    customersLoading.value = false;
  }
  try {
    cleaners.value = await getCleaners({ limit: 100 });
  } catch {
    /* keep empty list so the selects fall back to free-text behavior */
  } finally {
    cleanersLoading.value = false;
  }
});

// Pickers derive cleanerIds: first = primary, rest = crew.
const primaryCleanerId = ref<number | "">(
  props.booking?.cleaners.find((c) => c.role === "primary")?.id ??
    props.booking?.cleaners[0]?.id ??
    "",
);
const crewIds = ref<number[]>(
  props.booking?.cleaners
    .filter((c) => c.role === "crew")
    .map((c) => c.id) ?? [],
);

const availableCrew = computed(() =>
  cleaners.value.filter((c) => c.id !== primaryCleanerId.value),
);

function builderCleanerIds(): number[] {
  const primary =
    typeof primaryCleanerId.value === "number"
      ? [primaryCleanerId.value]
      : [];
  return [...primary, ...crewIds.value];
}

function primaryCleanerName(): string {
  if (typeof primaryCleanerId.value !== "number") return "";
  const c = cleaners.value.find((x) => x.id === primaryCleanerId.value);
  return c ? `${c.firstName} ${c.lastName}` : "";
}

function toDatetimeLocal(iso: string): string {
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

const form = reactive<CreateBookingInput>({
  customerName: props.booking?.customerName ?? "",
  customerEmail: props.booking?.customerEmail ?? "",
  customerId: props.booking?.customerId ?? null,
  serviceType: props.booking?.serviceType ?? "house_cleaning",
  scheduledFor: props.booking?.scheduledFor
    ? toDatetimeLocal(props.booking.scheduledFor)
    : "",
  durationMinutes: props.booking?.durationMinutes ?? 180,
  address: props.booking?.address ?? "",
  assignedCleaner: props.booking?.assignedCleaner ?? "",
  status: props.booking?.status ?? "pending",
  notes: props.booking?.notes ?? "",
  isRecurring: props.booking?.isRecurring ?? false,
  recurrence:
    props.booking?.isRecurring && props.booking?.recurrence
      ? props.booking.recurrence
      : "",
});

const submitted = ref(false);

const errors = computed(() => {
  const e: Partial<Record<keyof CreateBookingInput, string>> & {
    crewIds?: string;
  } = {};
  if (!form.customerName.trim())
    e.customerName = "Customer name is required";
  if (!form.scheduledFor.trim()) e.scheduledFor = "Scheduled date is required";
  if (!form.durationMinutes || form.durationMinutes <= 0)
    e.durationMinutes = "Duration must be greater than zero";
  if (!form.address.trim()) e.address = "Address is required";
  if (typeof primaryCleanerId.value !== "number") {
    e.assignedCleaner = "Primary cleaner is required";
  } else if (
    crewIds.value.length > 0 &&
    cleaners.value.length > 0 &&
    availableCrew.value.length === 0
  ) {
    e.crewIds = "The selected crew member is not available";
  }
  return e;
});

const isValid = computed(() => Object.keys(errors.value).length === 0);

function handleSubmit() {
  submitted.value = true;
  if (!isValid.value) return;
  const scheduledFor = form.scheduledFor
    ? new Date(form.scheduledFor).toISOString()
    : "";
  const cleanerIds = builderCleanerIds();
  const isRecurring = form.recurrence !== "";
  emit("save", {
    ...form,
    scheduledFor,
    durationMinutes: Number(form.durationMinutes),
    assignedCleaner: primaryCleanerName() || form.assignedCleaner,
    cleanerIds,
    isRecurring,
    recurrence: isRecurring ? form.recurrence : "",
  });
}

const inputClass =
  "w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500";

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
    <div class="absolute inset-0 bg-black/50" @click="emit('cancel')"></div>

    <div class="relative w-full max-w-lg rounded-xl bg-white shadow-xl">
      <div class="border-b border-gray-200 px-6 py-4">
        <h2 :id="titleId" class="text-base font-semibold text-gray-900">
          {{ booking ? `Edit Booking ${booking.bookingNumber}` : "New Booking" }}
        </h2>
      </div>

      <form @submit.prevent="handleSubmit">
        <div class="grid grid-cols-1 gap-4 px-6 py-5 sm:grid-cols-2">
          <div class="sm:col-span-2">
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="b-customerId"
              >Customer</label
            >
            <select
              id="b-customerId"
              :value="selectedCustomerId ?? ''"
              :disabled="customersLoading"
              @change="applyCustomer(($event.target as HTMLSelectElement).value)"
              class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
            >
              <option value="">
                {{ customersLoading ? "Loading customers…" : "Manual entry" }}
              </option>
              <option v-for="c in customers" :key="c.id" :value="c.id">
                {{ customerFullName(c) }} — {{ c.phone }}
              </option>
            </select>
            <p class="mt-1 text-xs text-gray-500">
              Pick a customer to pre-fill the name, or leave empty to enter
              manually.
            </p>
          </div>

          <div class="sm:col-span-2">
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
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
              class="mb-1 block text-sm font-medium text-gray-700"
              for="b-customerEmail"
              >Customer email</label
            >
            <input
              id="b-customerEmail"
              v-model="form.customerEmail"
              type="email"
              :class="inputClass"
              placeholder="somchai@example.com"
            />
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="b-serviceType"
              >Service type</label
            >
            <select
              id="b-serviceType"
              v-model="form.serviceType"
              :class="inputClass"
              @change="applyServiceDuration"
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
              class="mb-1 block text-sm font-medium text-gray-700"
              for="b-recurrence"
              >Repeat</label
            >
            <select id="b-recurrence" v-model="form.recurrence" :class="inputClass">
              <option
                v-for="opt in recurrenceOptions"
                :key="opt.value"
                :value="opt.value"
              >
                {{ opt.label }}
              </option>
            </select>
            <p class="mt-1 text-xs text-gray-500">
              Recurring bookings auto-create the next appointment.
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
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
              class="mb-1 block text-sm font-medium text-gray-700"
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
              class="mb-1 block text-sm font-medium text-gray-700"
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
              class="mb-1 block text-sm font-medium text-gray-700"
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
              class="mb-1 block text-sm font-medium text-gray-700"
              for="b-cleaner"
              >Primary cleaner</label
            >
            <select
              id="b-cleaner"
              v-model="primaryCleanerId"
              :class="inputClassFor('assignedCleaner')"
              :disabled="cleanersLoading"
            >
              <option :value="''" disabled>
                {{ cleanersLoading ? "Loading cleaners…" : "Select a cleaner" }}
              </option>
              <option
                v-for="c in cleaners"
                :key="c.id"
                :value="c.id"
                :disabled="c.status === 'inactive' || c.status === 'on_leave'"
              >
                {{ fullName(c) }}
                <template v-if="c.status === 'on_leave'"> (on leave)</template>
              </option>
            </select>
            <p
              v-if="submitted && errors.assignedCleaner"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.assignedCleaner }}
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="b-crew"
              >Crew (optional)</label
            >
            <select
              id="b-crew"
              v-model="crewIds"
              multiple
              size="4"
              :class="[
                inputClass,
                submitted && errors.crewIds
                  ? 'border-red-300 focus:border-red-500 focus:ring-red-500'
                  : '',
              ]"
              :disabled="cleanersLoading"
            >
              <option
                v-for="c in availableCrew"
                :key="c.id"
                :value="c.id"
                :disabled="c.status === 'inactive' || c.status === 'on_leave'"
              >
                {{ fullName(c) }}
                <template v-if="c.status === 'on_leave'"> (on leave)</template>
              </option>
            </select>
            <p
              v-if="submitted && errors.crewIds"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.crewIds }}
            </p>
          </div>

          <div class="sm:col-span-2">
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
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
            {{ booking ? "Save changes" : "Create booking" }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
