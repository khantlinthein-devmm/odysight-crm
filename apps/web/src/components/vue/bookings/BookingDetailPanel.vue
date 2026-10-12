<script setup lang="ts">
import { computed, ref, useId, watch } from "vue";
import { useModalA11y } from "../ui/useModalA11y";
import { getBookings, type Booking, type BookingStatus } from "../../../lib/bookings";
import { getCustomer, type Customer } from "../../../lib/customers";
import { formatMoney, serviceLabel } from "../../../lib/settings";
import { isBillableBooking } from "../../../lib/invoices";
import { formatDuration } from "../../../lib/duration";

// Side panel on the Bookings page: the booking, the customer behind it and
// that customer's other bookings, without leaving the page.

const props = defineProps<{
  booking: Booking;
  canEdit?: boolean;
  canInvoice?: boolean;
  canCollect?: boolean;
  invoicing?: boolean;
}>();

const emit = defineEmits<{
  close: [];
  open: [booking: Booking];
  edit: [booking: Booking];
  invoice: [booking: Booking];
  collect: [booking: Booking];
}>();

const titleId = useId();
const { container } = useModalA11y(() => emit("close"));

const statusLabels: Record<BookingStatus, string> = {
  pending: "Pending",
  confirmed: "Confirmed",
  in_progress: "In Progress",
  completed: "Completed",
  cancelled: "Cancelled",
  no_show: "No Show",
  rescheduled: "Rescheduled",
};

const statusStyles: Record<BookingStatus, string> = {
  pending: "bg-amber-100 text-amber-800 ring-1 ring-amber-300",
  confirmed: "bg-blue-100 text-blue-800 ring-1 ring-blue-300",
  in_progress: "bg-indigo-100 text-indigo-800 ring-1 ring-indigo-300",
  completed: "bg-green-100 text-green-800 ring-1 ring-green-300",
  cancelled: "bg-gray-200 text-gray-700 ring-1 ring-gray-300",
  no_show: "bg-red-100 text-red-800 ring-1 ring-red-300",
  rescheduled: "bg-purple-100 text-purple-800 ring-1 ring-purple-300",
};

const recurrenceLabels: Record<string, string> = {
  weekly: "Every week",
  biweekly: "Every 2 weeks",
  monthly: "Every month",
};

const customer = ref<Customer | null>(null);
const customerLoading = ref(false);
const history = ref<Booking[]>([]);
const historyLoading = ref(false);

let seq = 0;
async function load(b: Booking) {
  const mine = ++seq;
  customer.value = null;
  history.value = [];
  if (b.customerId) {
    customerLoading.value = true;
    getCustomer(b.customerId)
      .then((c) => {
        if (mine === seq) customer.value = c;
      })
      .catch(() => {
        /* booking details still show */
      })
      .finally(() => {
        if (mine === seq) customerLoading.value = false;
      });
  }
  historyLoading.value = true;
  try {
    // The list endpoint searches by name; narrow to this customer's record.
    const found = await getBookings({ search: b.customerName, limit: 100 });
    if (mine !== seq) return;
    const name = b.customerName.trim().toLowerCase();
    history.value = found
      .filter((x) =>
        b.customerId ? x.customerId === b.customerId : x.customerName.trim().toLowerCase() === name,
      )
      .sort((x, y) => y.scheduledFor.localeCompare(x.scheduledFor));
  } catch {
    /* history is a bonus; the panel works without it */
  } finally {
    if (mine === seq) historyLoading.value = false;
  }
}

watch(() => props.booking.id, () => void load(props.booking), { immediate: true });

const now = Date.now();
const upcoming = computed(() =>
  history.value
    .filter((x) => new Date(x.scheduledFor).getTime() >= now && x.status !== "cancelled")
    .reverse(),
);
const past = computed(() => history.value.filter((x) => new Date(x.scheduledFor).getTime() < now));
const completedCount = computed(() => history.value.filter((x) => x.status === "completed").length);

function dateTime(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    weekday: "short",
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function shortDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, { day: "numeric", month: "short", year: "numeric" });
}

const team = computed(() => {
  const list = props.booking.cleaners ?? [];
  const lead = list.find((c) => c.role === "primary") ?? list[0];
  return { lead: lead?.name ?? props.booking.assignedCleaner ?? "", crew: list.filter((c) => c !== lead).map((c) => c.name) };
});
</script>

<template>
  <div
    ref="container"
    role="dialog"
    aria-modal="true"
    :aria-labelledby="titleId"
    class="fixed inset-0 z-50"
  >
    <div class="absolute inset-0 bg-black/40" @click="emit('close')"></div>

    <aside
      class="absolute inset-y-0 right-0 flex w-full max-w-lg flex-col bg-white shadow-2xl"
    >
      <!-- Header -->
      <div class="flex items-start justify-between gap-3 border-b border-gray-200 px-5 py-4">
        <div class="min-w-0">
          <p class="font-mono text-xs font-semibold text-gray-500">{{ booking.bookingNumber }}</p>
          <h2 :id="titleId" class="truncate text-lg font-semibold text-gray-900">{{ booking.customerName }}</h2>
          <span
            :class="['mt-1 inline-flex rounded-full px-2.5 py-0.5 text-xs font-semibold', statusStyles[booking.status]]"
          >{{ statusLabels[booking.status] }}</span>
        </div>
        <button
          type="button"
          class="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-700"
          aria-label="Close"
          @click="emit('close')"
        >
          <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
        </button>
      </div>

      <div class="flex-1 space-y-5 overflow-y-auto px-5 py-5">
        <!-- Booking -->
        <section>
          <h3 class="mb-2 text-xs font-semibold uppercase tracking-wider text-gray-400">Booking</h3>
          <dl class="divide-y divide-gray-100 rounded-xl text-sm ring-1 ring-gray-200">
            <div class="flex gap-3 px-4 py-2.5">
              <dt class="w-24 shrink-0 text-gray-500">When</dt>
              <dd class="font-medium text-gray-900">{{ dateTime(booking.scheduledFor) }} · {{ formatDuration(booking.durationMinutes) }}</dd>
            </div>
            <div class="flex gap-3 px-4 py-2.5">
              <dt class="w-24 shrink-0 text-gray-500">Service</dt>
              <dd class="text-gray-900">
                {{ serviceLabel(booking.serviceType) }}
                <span v-if="booking.isRecurring" class="ml-1 rounded-md bg-amber-100 px-1.5 py-0.5 text-xs font-semibold text-amber-800">
                  ↻ {{ recurrenceLabels[booking.recurrence ?? ""] ?? "Recurring" }}
                </span>
              </dd>
            </div>
            <div class="flex gap-3 px-4 py-2.5">
              <dt class="w-24 shrink-0 text-gray-500">Price</dt>
              <dd class="text-gray-900">{{ booking.price != null ? `${formatMoney(booking.price)} before VAT` : "Catalog price" }}</dd>
            </div>
            <div class="flex gap-3 px-4 py-2.5">
              <dt class="w-24 shrink-0 text-gray-500">Address</dt>
              <dd class="text-gray-900">{{ booking.address || "—" }}</dd>
            </div>
            <div class="flex gap-3 px-4 py-2.5">
              <dt class="w-24 shrink-0 text-gray-500">Team</dt>
              <dd>
                <span v-if="team.lead" class="text-gray-900">
                  {{ team.lead }}<span v-if="team.crew.length" class="text-gray-500"> + {{ team.crew.join(", ") }}</span>
                </span>
                <a v-else href="/dispatch" class="font-medium text-amber-700 hover:underline">Not assigned yet — assign in Dispatch</a>
              </dd>
            </div>
            <div v-if="booking.notes" class="flex gap-3 px-4 py-2.5">
              <dt class="w-24 shrink-0 text-gray-500">Notes</dt>
              <dd class="whitespace-pre-line text-gray-700">{{ booking.notes }}</dd>
            </div>
          </dl>

          <div class="mt-3 flex flex-wrap gap-2">
            <button
              v-if="canEdit"
              type="button"
              class="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-sm font-semibold text-gray-700 hover:bg-gray-100"
              @click="emit('edit', booking)"
            >Edit</button>
            <button
              v-if="canCollect && booking.status === 'completed'"
              type="button"
              class="rounded-lg bg-green-600 px-3 py-1.5 text-sm font-semibold text-white hover:bg-green-700"
              @click="emit('collect', booking)"
            >Collect payment</button>
            <button
              v-if="canInvoice && isBillableBooking(booking.status)"
              type="button"
              :disabled="invoicing"
              class="rounded-lg bg-navy-600 px-3 py-1.5 text-sm font-semibold text-white hover:bg-navy-700 disabled:opacity-50"
              @click="emit('invoice', booking)"
            >{{ invoicing ? "Invoicing…" : "Invoice" }}</button>
            <a
              :href="`/checklists?booking=${booking.id}`"
              class="rounded-lg border border-navy-200 bg-navy-50 px-3 py-1.5 text-sm font-semibold text-navy-700 hover:bg-navy-100"
            >Checklist</a>
          </div>
        </section>

        <!-- Customer -->
        <section>
          <h3 class="mb-2 text-xs font-semibold uppercase tracking-wider text-gray-400">Customer</h3>
          <div class="rounded-xl p-4 text-sm ring-1 ring-gray-200">
            <p v-if="customerLoading" class="text-gray-400">Loading customer…</p>
            <template v-else-if="customer">
              <div class="flex items-center gap-3">
                <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-navy-600 font-bold text-white">
                  {{ (customer.firstName || "?").charAt(0).toUpperCase() }}
                </span>
                <div class="min-w-0">
                  <p class="truncate font-semibold text-gray-900">{{ customer.firstName }} {{ customer.lastName }}</p>
                  <p class="text-xs capitalize text-gray-500">
                    {{ customer.propertyType }}<span v-if="customer.area"> · {{ customer.area }}</span>
                    <span v-if="customer.lineLinked" class="ml-1 rounded bg-green-100 px-1.5 py-0.5 font-semibold normal-case text-green-700">LINE</span>
                  </p>
                </div>
              </div>
              <dl class="mt-3 space-y-1.5">
                <div class="flex gap-3">
                  <dt class="w-20 shrink-0 text-gray-500">Phone</dt>
                  <dd><a :href="`tel:${customer.phone}`" class="font-medium text-navy-700 hover:underline">{{ customer.phone }}</a></dd>
                </div>
                <div class="flex gap-3">
                  <dt class="w-20 shrink-0 text-gray-500">Email</dt>
                  <dd class="min-w-0 truncate">
                    <a v-if="customer.email" :href="`mailto:${customer.email}`" class="text-navy-700 hover:underline">{{ customer.email }}</a>
                    <span v-else class="text-gray-400">—</span>
                  </dd>
                </div>
                <div class="flex gap-3">
                  <dt class="w-20 shrink-0 text-gray-500">Address</dt>
                  <dd class="text-gray-700">{{ customer.address }}</dd>
                </div>
                <div v-if="customer.taxId" class="flex gap-3">
                  <dt class="w-20 shrink-0 text-gray-500">Tax ID</dt>
                  <dd class="font-mono text-gray-700">{{ customer.taxId }}</dd>
                </div>
              </dl>
            </template>
            <template v-else>
              <p class="font-semibold text-gray-900">{{ booking.customerName }}</p>
              <p v-if="booking.customerEmail" class="text-gray-600">{{ booking.customerEmail }}</p>
              <p class="mt-1 text-xs text-gray-500">One-off job — no customer record is linked to this booking.</p>
            </template>
          </div>
        </section>

        <!-- Customer's bookings -->
        <section>
          <div class="mb-2 flex items-baseline justify-between">
            <h3 class="text-xs font-semibold uppercase tracking-wider text-gray-400">Bookings of this customer</h3>
            <p v-if="history.length > 1" class="text-xs text-gray-500">
              {{ history.length }} total · {{ completedCount }} completed
            </p>
          </div>
          <p v-if="historyLoading" class="text-sm text-gray-400">Loading…</p>
          <template v-else>
            <p v-if="history.length <= 1" class="text-sm text-gray-500">This is the customer's only booking.</p>
            <template v-else>
              <p v-if="upcoming.length" class="mb-1 text-xs font-medium text-gray-500">Upcoming</p>
              <ul v-if="upcoming.length" class="mb-3 divide-y divide-gray-100 rounded-xl ring-1 ring-gray-200">
                <li v-for="x in upcoming" :key="x.id">
                  <button
                    type="button"
                    :class="['flex w-full items-center gap-3 px-4 py-2 text-left text-sm hover:bg-gray-50', x.id === booking.id ? 'bg-navy-50' : '']"
                    @click="emit('open', x)"
                  >
                    <span class="w-28 shrink-0 text-gray-900">{{ shortDate(x.scheduledFor) }}</span>
                    <span class="min-w-0 flex-1 truncate text-gray-600">{{ serviceLabel(x.serviceType) }}</span>
                    <span :class="['shrink-0 rounded-full px-2 py-0.5 text-[11px] font-semibold', statusStyles[x.status]]">{{ statusLabels[x.status] }}</span>
                  </button>
                </li>
              </ul>
              <p v-if="past.length" class="mb-1 text-xs font-medium text-gray-500">Past</p>
              <ul v-if="past.length" class="divide-y divide-gray-100 rounded-xl ring-1 ring-gray-200">
                <li v-for="x in past.slice(0, 10)" :key="x.id">
                  <button
                    type="button"
                    :class="['flex w-full items-center gap-3 px-4 py-2 text-left text-sm hover:bg-gray-50', x.id === booking.id ? 'bg-navy-50' : '']"
                    @click="emit('open', x)"
                  >
                    <span class="w-28 shrink-0 text-gray-900">{{ shortDate(x.scheduledFor) }}</span>
                    <span class="min-w-0 flex-1 truncate text-gray-600">{{ serviceLabel(x.serviceType) }}</span>
                    <span :class="['shrink-0 rounded-full px-2 py-0.5 text-[11px] font-semibold', statusStyles[x.status]]">{{ statusLabels[x.status] }}</span>
                  </button>
                </li>
              </ul>
              <p v-if="past.length > 10" class="mt-1 text-xs text-gray-400">Showing the latest 10 past bookings.</p>
            </template>
          </template>
        </section>
      </div>
    </aside>
  </div>
</template>
