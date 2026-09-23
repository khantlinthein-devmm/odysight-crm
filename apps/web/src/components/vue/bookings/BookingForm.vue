<script setup lang="ts">
import { computed, onMounted, reactive, ref, useId } from "vue";
import { useModalA11y } from "../ui/useModalA11y";
import { getCustomers, type Customer } from "../../../lib/customers";
import type { Cleaner } from "../../../lib/cleaners";
import { getCleaners } from "../../../lib/cleaners";
import { getSites, type Site } from "../../../lib/sites";
import { getContracts, type Contract } from "../../../lib/contracts";
import type {
  Booking,
  BookingStatus,
  CreateBookingInput,
  RecurrenceFreq,
} from "../../../lib/bookings";
import {
  activeServices,
  formatMoney,
  getWorkspaceSettings,
} from "../../../lib/settings";
import { lineTotal } from "../../../lib/pricing";

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
  return `${c.firstName} ${c.lastName}`.trim();
}

function applyServiceDuration(fallback?: number) {
  // New bookings only: prefill the catalog's default duration.
  if (props.booking) return;
  const found = activeServices().find((s) => s.id === form.serviceType);
  form.durationMinutes = found?.durationMinutes ?? fallback ?? form.durationMinutes;
  if (found && !rateTouched.value) {
    ratePerSqm.value = found.pricePerSqm ?? 0;
    basePrice.value = found.basePrice ?? 0;
  }
}

// --- Price estimate (informational; booking API stores no price) ---
const areaSqm = ref(0);
const ratePerSqm = ref(0);
const basePrice = ref(0);
const rateTouched = ref(false);
const equipmentFee = ref(0);
const transportFee = ref(0);
const labourFee = ref(0);
const showEstimate = ref(true);

const estimateTotal = computed(
  () =>
    lineTotal({
      basePrice: Number(basePrice.value) || 0,
      areaSqm: Number(areaSqm.value) || 0,
      pricePerSqm: Number(ratePerSqm.value) || 0,
    }) +
    (Number(equipmentFee.value) || 0) +
    (Number(transportFee.value) || 0) +
    (Number(labourFee.value) || 0),
);

function insertEstimateIntoNotes() {
  const area = Number(areaSqm.value) || 0;
  const summary =
    `Estimate: ${formatMoney(estimateTotal.value)}` +
    (area > 0 ? ` (${area}m² @ ${formatMoney(Number(ratePerSqm.value) || 0)}/m²)` : "");
  form.notes = form.notes ? `${form.notes}\n${summary}` : summary;
}

const cleaners = ref<Cleaner[]>([]);
const cleanersLoading = ref(true);
const customers = ref<Customer[]>([]);
const customersLoading = ref(false);
const selectedCustomerId = ref<number | null>(props.booking?.customerId ?? null);
const sites = ref<Site[]>([]);
const sitesLoading = ref(false);
const selectedSiteId = ref<number | null>(props.booking?.siteId ?? null);
const contracts = ref<Contract[]>([]);
const selectedContractId = ref<number | null>(props.booking?.contractId ?? null);

async function loadCommercial(customerId: number | null) {
  sites.value = [];
  contracts.value = [];
  selectedSiteId.value = null;
  selectedContractId.value = null;
  if (!customerId) return;
  sitesLoading.value = true;
  try {
    sites.value = await getSites({ customerId, limit: 100 });
    if (props.booking?.siteId && sites.value.some((s) => s.id === props.booking!.siteId)) {
      selectedSiteId.value = props.booking.siteId;
      form.siteId = props.booking.siteId;
    } else if (!props.booking) {
      const def = sites.value.find((s) => s.isDefault) ?? sites.value[0];
      if (def) {
        selectedSiteId.value = def.id;
        form.siteId = def.id;
        if (!form.address) form.address = def.address;
      }
    }
  } catch {
    /* site picker is optional: bookings work with free-text address */
  } finally {
    sitesLoading.value = false;
  }
  try {
    contracts.value = await getContracts({ customerId, limit: 100 });
    if (props.booking?.contractId && contracts.value.some((c) => c.id === props.booking!.contractId)) {
      selectedContractId.value = props.booking.contractId;
      form.contractId = props.booking.contractId;
    }
  } catch {
    /* contracts are optional */
  }
}

function applySite(siteId: number | string) {
  const raw = typeof siteId === "string" ? Number(siteId) : siteId;
  selectedSiteId.value = raw || null;
  form.siteId = selectedSiteId.value;
  const site = sites.value.find((s) => s.id === raw);
  if (site && !form.address) form.address = site.address;
}

function applyContract(contractId: number | string) {
  const raw = typeof contractId === "string" ? Number(contractId) : contractId;
  selectedContractId.value = raw || null;
  form.contractId = selectedContractId.value;
}

function customerFullName(c: Customer): string {
  return `${c.firstName} ${c.lastName}`.trim();
}

function applyCustomer(customerId: number | string) {
  const raw = typeof customerId === "string" ? Number(customerId) : customerId;
  selectedCustomerId.value = raw || null;
  form.customerId = selectedCustomerId.value;
  const customer = customers.value.find((c) => c.id === raw);
  if (!customer) {
    void loadCommercial(selectedCustomerId.value);
    return;
  }
  form.customerName = customerFullName(customer);
  form.customerEmail = customer.email;
  form.address = form.address || customer.address || "";
  void loadCommercial(selectedCustomerId.value);
}

onMounted(async () => {
  try {
    const ws = await getWorkspaceSettings();
    if (!props.booking) {
      if (!form.serviceType && ws.services.length > 0) {
        form.serviceType = ws.services[0]!.id;
      }
      applyServiceDuration(ws.booking.defaultDurationMinutes);
      const found = ws.services.find((s) => s.id === form.serviceType);
      if (found) {
        basePrice.value = found.basePrice ?? 0;
        ratePerSqm.value = found.pricePerSqm ?? 0;
      }
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
  if (selectedCustomerId.value) await loadCommercial(selectedCustomerId.value);
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
  return c ? `${c.firstName} ${c.lastName}`.trim() : "";
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
  siteId: props.booking?.siteId ?? null,
  contractId: props.booking?.contractId ?? null,
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
    class="fixed inset-0 z-50 overflow-y-auto"
  >
    <div class="fixed inset-0 bg-black/50" @click="emit('cancel')"></div>

    <div class="relative flex min-h-full items-center justify-center p-4 sm:p-6">
      <div class="relative w-full max-w-2xl rounded-2xl bg-white shadow-xl">
      <div class="sticky top-0 z-10 rounded-t-2xl border-b border-gray-200 bg-white px-6 py-4">
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

          <div v-if="selectedCustomerId">
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="b-siteId"
              >Site <span class="font-normal text-gray-400">(optional)</span></label
            >
            <select
              id="b-siteId"
              :value="selectedSiteId ?? ''"
              :disabled="sitesLoading"
              @change="applySite(($event.target as HTMLSelectElement).value)"
              :class="inputClass"
            >
              <option value="">No specific site</option>
              <option v-for="s in sites" :key="s.id" :value="s.id">
                {{ s.name }}{{ s.isDefault ? " (default)" : "" }}
              </option>
            </select>
          </div>

          <div v-if="selectedCustomerId && contracts.length > 0">
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="b-contractId"
              >Contract <span class="font-normal text-gray-400">(optional)</span></label
            >
            <select
              id="b-contractId"
              :value="selectedContractId ?? ''"
              @change="applyContract(($event.target as HTMLSelectElement).value)"
              :class="inputClass"
            >
              <option value="">No contract</option>
              <option v-for="c in contracts" :key="c.id" :value="c.id">
                {{ c.contractNumber }} — {{ c.status }}
              </option>
            </select>
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

          <div class="sm:col-span-2 rounded-2xl bg-gray-50 p-4 ring-1 ring-gray-100">
            <button
              type="button"
              class="flex w-full items-center justify-between text-sm font-medium text-gray-700"
              @click="showEstimate = !showEstimate"
            >
              <span>Price estimate · {{ formatMoney(estimateTotal) }}</span>
              <span class="text-xs text-gray-400">{{ showEstimate ? "Hide" : "Show" }}</span>
            </button>
            <div v-if="showEstimate" class="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-3">
              <div>
                <label class="mb-1 block text-xs font-medium text-gray-600" for="b-area">Area (m²)</label>
                <input
                  id="b-area"
                  v-model.number="areaSqm"
                  type="number"
                  min="0"
                  :class="inputClass"
                  placeholder="e.g. 80"
                />
              </div>
              <div>
                <label class="mb-1 block text-xs font-medium text-gray-600" for="b-rate">Rate / m²</label>
                <input
                  id="b-rate"
                  v-model.number="ratePerSqm"
                  type="number"
                  min="0"
                  step="0.01"
                  :class="inputClass"
                  @input="rateTouched = true"
                />
              </div>
              <div>
                <label class="mb-1 block text-xs font-medium text-gray-600" for="b-base">Base price</label>
                <input
                  id="b-base"
                  v-model.number="basePrice"
                  type="number"
                  min="0"
                  step="0.01"
                  :class="inputClass"
                  @input="rateTouched = true"
                />
              </div>
              <div>
                <label class="mb-1 block text-xs font-medium text-gray-600" for="b-equip">Equipment</label>
                <input id="b-equip" v-model.number="equipmentFee" type="number" min="0" step="0.01" :class="inputClass" />
              </div>
              <div>
                <label class="mb-1 block text-xs font-medium text-gray-600" for="b-transport">Transport</label>
                <input id="b-transport" v-model.number="transportFee" type="number" min="0" step="0.01" :class="inputClass" />
              </div>
              <div>
                <label class="mb-1 block text-xs font-medium text-gray-600" for="b-labour">Labour</label>
                <input id="b-labour" v-model.number="labourFee" type="number" min="0" step="0.01" :class="inputClass" />
              </div>
            </div>
            <div v-if="showEstimate" class="mt-3 flex flex-wrap items-center justify-between gap-2">
              <p class="text-sm text-gray-600">
                Total: <span class="font-semibold text-gray-900">{{ formatMoney(estimateTotal) }}</span>
                <a href="/calculator" target="_blank" class="ml-2 text-xs text-navy-600 hover:underline">Full calculator →</a>
              </p>
              <button
                type="button"
                class="rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-xs font-medium text-gray-700 hover:bg-gray-100"
                @click="insertEstimateIntoNotes"
              >
                Insert into notes
              </button>
            </div>
          </div>
        </div>

        <div class="sticky bottom-0 flex justify-end gap-3 rounded-b-2xl border-t border-gray-200 bg-white px-6 py-4">
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
  </div>
</template>
