<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import {
  createBooking,
  deleteBooking,
  getBookings,
  updateBooking,
  type Booking,
  type BookingStatus,
  type CreateBookingInput,
} from "../../../lib/bookings";
import { showToast } from "../../../lib/toast";
import { isOfflineQueued } from "../../../lib/offline";
import { activeServices, currencyCode, getWorkspaceSettings, serviceLabel } from "../../../lib/settings";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";
import { createInvoice, getInvoices, isBillableBooking, type PaymentResult } from "../../../lib/invoices";
import { ApiError } from "../../../lib/api";
import RecordPaymentDialog from "../invoices/RecordPaymentDialog.vue";
import VatBreakdown from "../invoices/VatBreakdown.vue";
import { formatDuration } from "../../../lib/duration";
import BookingForm from "./BookingForm.vue";
import { getQuote, updateQuote, type Quote } from "../../../lib/quotes";
import { getCustomer } from "../../../lib/customers";
import ConfirmDialog from "../ui/ConfirmDialog.vue";
import MyJobs from "./MyJobs.vue";
import BookingDetailPanel from "./BookingDetailPanel.vue";
import BookingCalendar from "./BookingCalendar.vue";

const role = getSessionUser()?.role;
const canCreate = computed(() => hasPermission(role, "bookings.create"));
// Cleaners hold bookings.update only to progress their own jobs (My Jobs);
// the full edit form is office-only and the API rejects it for them.
const canEdit = computed(() => role !== "CLEANER" && hasPermission(role, "bookings.update"));
const canDelete = computed(() => hasPermission(role, "bookings.delete"));
const canInvoice = computed(() => hasPermission(role, "invoices.create"));
const canCollect = computed(() => canInvoice.value && hasPermission(role, "payments.create"));
const collecting = ref<Booking | null>(null);

function onCollected(result: PaymentResult) {
  collecting.value = null;
  showToast(`Paid — receipt ${result.receipt.receiptNumber} issued`, "success");
}

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

const statusAccent: Record<BookingStatus, string> = {
  pending: "bg-amber-400",
  confirmed: "bg-blue-500",
  in_progress: "bg-indigo-500",
  completed: "bg-green-500",
  cancelled: "bg-gray-300",
  no_show: "bg-red-500",
  rescheduled: "bg-purple-500",
};

const recurrenceLabels: Record<string, string> = {
  weekly: "Weekly",
  biweekly: "Bi-weekly",
  monthly: "Monthly",
};

const serviceTypeLabels: Record<string, string> = {
  house_cleaning: "House Cleaning",
  condo_cleaning: "Condo Cleaning",
  deep_cleaning: "Deep Cleaning",
  move_in_out: "Move In/Out",
  after_renovation: "After Renovation",
  office_cleaning: "Office Cleaning",
  junk_removal: "Junk Removal",
  aircon_service: "Aircon Service",
};

function labelForService(id: string): string {
  // Prefer the live catalog name; fall back to built-ins, then the raw id.
  const live = serviceLabel(id);
  if (live !== id) return live;
  return serviceTypeLabels[id] ?? id;
}

const bookings = ref<Booking[]>([]);
const loading = ref(true);
const search = ref("");
const statusFilter = ref<BookingStatus | "">("");
const page = ref(1);
const pageSize = 9;

const showForm = ref(false);
const editingBooking = ref<Booking | undefined>(undefined);
const saving = ref(false);
const pendingDelete = ref<Booking | null>(null);
const deleting = ref(false);
const invoicingId = ref<number | null>(null);

// Clicking a booking opens a side panel with the booking, its customer and
// the customer's other bookings.
const viewing = ref<Booking | null>(null);

function openDetails(booking: Booking, event?: MouseEvent) {
  // Buttons and links on the card keep their own action.
  if (event && (event.target as HTMLElement).closest("button, a")) return;
  viewing.value = booking;
}

function editFromPanel(booking: Booking) {
  viewing.value = null;
  openEdit(booking);
}

function collectFromPanel(booking: Booking) {
  viewing.value = null;
  collecting.value = booking;
}

// List of cards, or the day / month / year calendar with its summary.
type PageView = "list" | "calendar";
const pageView = ref<PageView>("list");
function setPageView(v: PageView) {
  pageView.value = v;
  const url = new URL(window.location.href);
  if (v === "calendar") url.searchParams.set("view", "calendar");
  else url.searchParams.delete("view");
  history.replaceState(null, "", url);
}

const filteredBookings = computed(() => {
  const query = search.value.trim().toLowerCase();
  return bookings.value.filter((booking) => {
    const matchesStatus =
      !statusFilter.value || booking.status === statusFilter.value;
    const matchesQuery =
      !query ||
      booking.customerName.toLowerCase().includes(query) ||
      booking.bookingNumber.toLowerCase().includes(query);
    return matchesStatus && matchesQuery;
  });
});

const totalPages = computed(() =>
  Math.max(1, Math.ceil(filteredBookings.value.length / pageSize)),
);

const pagedBookings = computed(() => {
  const start = (page.value - 1) * pageSize;
  return filteredBookings.value.slice(start, start + pageSize);
});

// --- Date filter: asks the server for the chosen days ---
type DatePreset = "all" | "today" | "tomorrow" | "week" | "month" | "custom";
const datePresets: { value: DatePreset; label: string }[] = [
  { value: "all", label: "All dates" },
  { value: "today", label: "Today" },
  { value: "tomorrow", label: "Tomorrow" },
  { value: "week", label: "This week" },
  { value: "month", label: "This month" },
  { value: "custom", label: "Pick dates" },
];
const datePreset = ref<DatePreset>("all");
const customFrom = ref("");
const customTo = ref("");

function dayStart(d: Date, addDays = 0): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate() + addDays);
}

function parseDay(value: string): Date | null {
  const [y, m, d] = value.split("-").map(Number);
  return y && m && d ? new Date(y, m - 1, d) : null;
}

const dateRange = computed<{ from: Date; to: Date } | null>(() => {
  const today = dayStart(new Date());
  switch (datePreset.value) {
    case "today":
      return { from: today, to: dayStart(today, 1) };
    case "tomorrow":
      return { from: dayStart(today, 1), to: dayStart(today, 2) };
    case "week": {
      const monday = dayStart(today, -((today.getDay() + 6) % 7));
      return { from: monday, to: dayStart(monday, 7) };
    }
    case "month":
      return {
        from: new Date(today.getFullYear(), today.getMonth(), 1),
        to: new Date(today.getFullYear(), today.getMonth() + 1, 1),
      };
    case "custom": {
      const from = parseDay(customFrom.value);
      const to = parseDay(customTo.value || customFrom.value);
      if (!from || !to) return null;
      // "To" is inclusive for the admin: search up to the end of that day.
      return from <= to ? { from, to: dayStart(to, 1) } : { from: to, to: dayStart(from, 1) };
    }
    default:
      return null;
  }
});

const PAGE = 200;
const MAX_PAGES = 10;
const truncated = ref(false);

let fetchSeq = 0;
async function fetchBookings() {
  const seq = ++fetchSeq;
  loading.value = true;
  truncated.value = false;
  try {
    // Warm the workspace settings cache so service names render from the catalog.
    await getWorkspaceSettings().catch(() => null);
    const range = dateRange.value;
    const rows: Booking[] = [];
    for (let i = 0; i < MAX_PAGES; i++) {
      const batch = await getBookings({
        limit: PAGE,
        offset: i * PAGE,
        ...(range ? { from: range.from.toISOString(), to: range.to.toISOString() } : {}),
      });
      rows.push(...batch);
      if (batch.length < PAGE) break;
      if (i === MAX_PAGES - 1) truncated.value = true;
    }
    if (seq !== fetchSeq) return;
    // A date range reads best in time order; "All dates" keeps newest first.
    if (range) rows.sort((a, b) => a.scheduledFor.localeCompare(b.scheduledFor));
    bookings.value = rows;
  } catch {
    if (seq === fetchSeq) showToast("Failed to load bookings", "error");
  } finally {
    if (seq === fetchSeq) loading.value = false;
  }
}

watch(dateRange, (next, prev) => {
  if (next?.from.getTime() === prev?.from.getTime() && next?.to.getTime() === prev?.to.getTime()) return;
  page.value = 1;
  void fetchBookings();
});

const rangeLabel = computed(() => {
  const r = dateRange.value;
  if (!r) return "";
  const last = dayStart(r.to, -1);
  const fmt = (d: Date) => d.toLocaleDateString(undefined, { day: "numeric", month: "short", year: "numeric" });
  return r.from.getTime() === last.getTime() ? fmt(r.from) : `${fmt(r.from)} – ${fmt(last)}`;
});

function openCreate() {
  editingBooking.value = undefined;
  showForm.value = true;
}

function openEdit(booking: Booking) {
  editingBooking.value = booking;
  showForm.value = true;
}

function closeForm() {
  showForm.value = false;
  editingBooking.value = undefined;
  prefill.value = undefined;
  fromQuote.value = null;
}

// "Create booking" on an accepted quote lands here with ?fromQuote=<id>:
// open a new booking filled in from the quote, then link the two.
const prefill = ref<Partial<CreateBookingInput> | undefined>(undefined);
const fromQuote = ref<Quote | null>(null);

async function openFromQuote() {
  const id = Number(new URLSearchParams(window.location.search).get("fromQuote"));
  if (!id) return;
  history.replaceState(null, "", window.location.pathname);
  try {
    const q = await getQuote(id);
    const c = await getCustomer(q.customerId).catch(() => null);
    const inclusive = await getWorkspaceSettings()
      .then((st) => !!st.company.pricesIncludeVat)
      .catch(() => false);
    const lines = q.items.map((i) => `${i.serviceName} × ${i.quantity}`).join(", ");
    prefill.value = {
      customerId: q.customerId,
      customerName: c ? `${c.firstName} ${c.lastName}`.trim() : "",
      customerEmail: c?.email ?? "",
      siteId: q.siteId ?? null,
      // The booking price is read the way Settings says prices are entered:
      // with VAT inside (ราคารวม VAT) it is what the customer pays.
      price: inclusive ? q.total : q.subtotal,
      notes: `Quote ${q.quoteNumber}: ${lines}`.slice(0, 500),
    };
    fromQuote.value = q;
    editingBooking.value = undefined;
    showForm.value = true;
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Could not load the quote", "error");
  }
}

// The calculator ("Create a booking with this price") and customer cards
// ("New booking") land here with ?newBooking=1: open a pre-filled booking.
async function openNewFromLink() {
  const params = new URLSearchParams(window.location.search);
  if (params.get("newBooking") !== "1") return;
  history.replaceState(null, "", window.location.pathname);
  const inclusive = await getWorkspaceSettings()
    .then((st) => !!st.company.pricesIncludeVat)
    .catch(() => false);
  // Same rule as quotes: with VAT-inclusive pricing the price is the total.
  const price = Number(params.get(inclusive ? "total" : "price"));
  const minutes = Number(params.get("minutes"));
  const service = params.get("service");
  // "New booking" on a customer card passes ?customer=<id>.
  const customerId = Number(params.get("customer"));
  const c = customerId ? await getCustomer(customerId).catch(() => null) : null;
  prefill.value = {
    ...(c
      ? {
          customerId: c.id,
          customerName: `${c.firstName} ${c.lastName}`.trim(),
          customerEmail: c.email,
          address: c.address,
        }
      : {}),
    ...(Number.isFinite(price) && price > 0 ? { price } : {}),
    ...(service ? { serviceType: service } : {}),
    ...(minutes > 0 ? { durationMinutes: minutes } : {}),
  };
  editingBooking.value = undefined;
  showForm.value = true;
}

async function handleSave(input: CreateBookingInput) {
  saving.value = true;
  try {
    if (editingBooking.value) {
      try {
        const updated = await updateBooking(editingBooking.value.id, input);
        bookings.value = bookings.value.map((b) =>
          b.id === updated.id ? updated : b,
        );
        showToast("Booking updated", "success");
      } catch (err) {
        if (isOfflineQueued(err)) {
          // Optimistic row update; the queued PATCH replays on reconnect.
          // A 409 there (reassigned / double-booked) becomes a reviewable
          // dead-letter instead of a silent retry.
          bookings.value = bookings.value.map((b) =>
            b.id === editingBooking.value!.id ? { ...b, ...input } : b,
          );
          showToast("Saved offline — edit will sync when online", "success");
        } else {
          throw err;
        }
      }
    } else {
      const created = await createBooking(input);
      bookings.value = [created, ...bookings.value];
      const q = fromQuote.value;
      if (q) {
        await updateQuote(q.id, { convertedBookingId: created.id }).catch(() => {
          showToast(`Booking created, but it could not be linked to quote ${q.quoteNumber}`, "info");
        });
      }
      showToast(q ? `Booking created from quote ${q.quoteNumber}` : "Booking created", "success");
    }
    closeForm();
  } catch (err) {
    const message =
      err instanceof Error && err.message
        ? err.message
        : "Failed to save booking";
    showToast(message, "error");
  } finally {
    saving.value = false;
  }
}

async function handleDelete() {
  if (!pendingDelete.value) return;
  deleting.value = true;
  try {
    await deleteBooking(pendingDelete.value.id);
    bookings.value = bookings.value.filter(
      (b) => b.id !== pendingDelete.value!.id,
    );
    showToast("Booking deleted", "success");
  } catch (err) {
    showToast(
      err instanceof Error ? `Failed to delete booking: ${err.message}` : "Failed to delete booking",
      "error",
    );
  } finally {
    deleting.value = false;
    pendingDelete.value = null;
  }
}

// "Invoice" first shows the price so it can be changed for this job, instead
// of editing the service catalog in Settings before every invoice.
const pendingInvoice = ref<Booking | null>(null);
const invoiceAmount = ref<string | number>("");
const savePriceOnBooking = ref(true);
const pricesIncludeVat = ref(false);

function catalogPrice(serviceType: string): number | null {
  const found = activeServices().find((s) => s.id === serviceType);
  return found && found.basePrice > 0 ? found.basePrice : null;
}

async function handleInvoice(booking: Booking) {
  const ws = await getWorkspaceSettings().catch(() => null);
  pricesIncludeVat.value = !!ws?.company.pricesIncludeVat;
  const start = booking.price ?? catalogPrice(booking.serviceType);
  invoiceAmount.value = start != null ? String(start) : "";
  savePriceOnBooking.value = true;
  pendingInvoice.value = booking;
}

const invoiceAmountValue = computed(() => {
  const n = Number(invoiceAmount.value);
  return String(invoiceAmount.value).trim() !== "" && Number.isFinite(n) && n > 0 ? n : null;
});

async function confirmInvoice() {
  const booking = pendingInvoice.value;
  const amount = invoiceAmountValue.value;
  if (!booking || amount == null) return;
  pendingInvoice.value = null;
  if (savePriceOnBooking.value && booking.price !== amount) {
    // Keep the booking (and reports) in step with what was billed.
    const updated = await updateBooking(booking.id, { price: amount }).catch(() => null);
    if (updated) {
      const i = bookings.value.findIndex((b) => b.id === updated.id);
      if (i !== -1) bookings.value[i] = updated;
      if (viewing.value?.id === updated.id) viewing.value = updated;
    } else {
      showToast("Invoice will use the new price, but the booking price was not updated", "info");
    }
  }
  await issueInvoice(booking, amount);
}

async function issueInvoice(booking: Booking, amount: number) {
  invoicingId.value = booking.id;
  try {
    const created = await createInvoice({ bookingId: booking.id, subtotal: amount });
    showToast("Invoice created", "success");
    window.location.href = `/invoices/${created.id}`;
  } catch (err) {
    // One active invoice per booking: open the existing one instead.
    if (err instanceof ApiError && err.status === 409) {
      const found = (await getInvoices({ search: booking.bookingNumber }).catch(() => []))
        .find((i) => i.bookingNumber === booking.bookingNumber && i.status !== "void");
      if (found) {
        window.location.href = `/invoices/${found.id}`;
        return;
      }
    }
    const message =
      err instanceof Error && err.message
        ? err.message
        : "Failed to create invoice";
    showToast(message, "error");
  } finally {
    invoicingId.value = null;
  }
}

// Card styling: the ticket stub takes the status colour.
const stubTone: Record<BookingStatus, { bg: string; text: string }> = {
  pending: { bg: "from-amber-50", text: "text-amber-600" },
  confirmed: { bg: "from-blue-50", text: "text-blue-600" },
  in_progress: { bg: "from-indigo-50", text: "text-indigo-600" },
  completed: { bg: "from-green-50", text: "text-green-600" },
  cancelled: { bg: "from-gray-100", text: "text-gray-400" },
  no_show: { bg: "from-red-50", text: "text-red-600" },
  rescheduled: { bg: "from-purple-50", text: "text-purple-600" },
};

function dayParts(iso: string) {
  const d = new Date(iso);
  return {
    month: d.toLocaleDateString(undefined, { month: "short" }),
    day: d.getDate(),
    weekday: d.toLocaleDateString(undefined, { weekday: "short" }),
    time: d.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" }),
  };
}

function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w.charAt(0).toUpperCase())
    .join("");
}

function cleanerSummary(booking: Booking): string {
  const primary =
    booking.cleaners?.find((c) => c.role === "primary")?.name ??
    booking.cleaners?.[0]?.name ??
    booking.assignedCleaner ??
    "";
  const crewCount = (booking.cleaners ?? []).filter((c) => c.role === "crew")
    .length;
  return crewCount > 0 ? `${primary} +${crewCount} crew` : primary;
}

onMounted(() => {
  if (new URLSearchParams(window.location.search).get("view") === "calendar") pageView.value = "calendar";
  void fetchBookings();
  void openFromQuote();
  void openNewFromLink();
});
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-gray-100/70">
    <div
      class="flex flex-col gap-3 rounded-t-xl border-b border-gray-200 bg-white px-6 py-4 sm:flex-row sm:items-center"
    >
      <h2 class="text-base font-semibold text-gray-900">Bookings</h2>
      <span
        v-if="pageView === 'list'"
        class="self-start rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-600 sm:self-auto"
      >
        {{ filteredBookings.length }}
      </span>
      <div
        v-if="role !== 'CLEANER'"
        class="inline-flex self-start rounded-lg bg-gray-100 p-0.5"
        role="tablist"
        aria-label="Bookings view"
      >
        <button
          v-for="v in (['list', 'calendar'] as const)"
          :key="v"
          type="button"
          role="tab"
          :aria-selected="pageView === v"
          :class="[
            'rounded-md px-3 py-1 text-sm font-medium capitalize transition-colors',
            pageView === v ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-800',
          ]"
          @click="setPageView(v)"
        >
          {{ v }}
        </button>
      </div>

      <div class="sm:ml-auto flex flex-col gap-2 sm:flex-row">
        <input
          v-if="pageView === 'list'"
          v-model="search"
          @input="page = 1"
          type="search"
          placeholder="Search booking or customer..."
          class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500 sm:w-56"
        />
        <select
          v-if="pageView === 'list'"
          v-model="statusFilter"
          @change="page = 1"
          class="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
        >
          <option value="">All statuses</option>
          <option
            v-for="(label, value) in statusLabels"
            :key="value"
            :value="value"
          >
            {{ label }}
          </option>
        </select>
        <button
          v-if="canCreate"
          type="button"
          class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700"
          @click="openCreate"
        >
          New Booking
        </button>
      </div>
    </div>

    <div
      v-if="pageView === 'list'"
      class="flex flex-wrap items-center gap-2 border-b border-gray-200 bg-white px-6 py-3"
    >
      <span class="text-xs font-medium text-gray-500">Date</span>
      <button
        v-for="d in datePresets"
        :key="d.value"
        type="button"
        :aria-pressed="datePreset === d.value"
        :class="[
          'rounded-full px-3 py-1 text-xs font-medium ring-1 transition-colors',
          datePreset === d.value
            ? 'bg-navy-600 text-white ring-navy-600'
            : 'bg-white text-gray-600 ring-gray-200 hover:bg-gray-50',
        ]"
        @click="datePreset = d.value"
      >
        {{ d.label }}
      </button>
      <template v-if="datePreset === 'custom'">
        <label class="sr-only" for="bk-from">From</label>
        <input
          id="bk-from"
          v-model="customFrom"
          type="date"
          class="rounded-lg border border-gray-200 px-2 py-1 text-xs focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
        />
        <span class="text-xs text-gray-400">to</span>
        <label class="sr-only" for="bk-to">To</label>
        <input
          id="bk-to"
          v-model="customTo"
          type="date"
          :min="customFrom || undefined"
          class="rounded-lg border border-gray-200 px-2 py-1 text-xs focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
        />
      </template>
      <span v-if="rangeLabel" class="text-xs text-gray-500 sm:ml-auto">
        {{ rangeLabel }} · <strong class="text-gray-900">{{ filteredBookings.length }}</strong> bookings
      </span>
    </div>
    <p v-if="pageView === 'list' && truncated" class="bg-amber-50 px-6 py-2 text-xs text-amber-800">
      Showing the first {{ PAGE * MAX_PAGES }} bookings. Pick a shorter date range to see the rest.
    </p>

    <!-- Cleaner field view (mobile cards with Accept / Check-in actions) -->
    <div v-if="role === 'CLEANER'" class="p-4 lg:hidden">
      <MyJobs :bookings="filteredBookings" :loading="loading" @changed="fetchBookings" />
    </div>

    <div v-if="pageView === 'calendar'" class="p-4 sm:p-6">
      <BookingCalendar @open="viewing = $event" />
    </div>

    <div v-else>

    <div v-if="loading" class="px-6 py-16 text-center">
      <p class="text-sm text-gray-500">Loading bookings...</p>
    </div>

    <div
      v-else-if="filteredBookings.length === 0"
      class="px-6 py-16 text-center"
    >
      <p class="text-sm font-medium text-gray-900">No bookings found</p>
      <p class="mt-1 text-sm text-gray-500">
        {{ dateRange ? "No bookings on these dates." : "Try adjusting your filters or create a new booking." }}
      </p>
    </div>

    <div v-else class="grid grid-cols-1 gap-4 p-4 sm:p-6 md:grid-cols-2 xl:grid-cols-3">
      <article
        v-for="booking in pagedBookings"
        :key="booking.id"
        class="group relative flex cursor-pointer overflow-hidden rounded-2xl bg-white shadow-sm ring-1 ring-gray-200 transition duration-200 hover:-translate-y-0.5 hover:shadow-lg hover:ring-gray-300"
        @click="openDetails(booking, $event)"
      >
        <!-- Ticket stub: the date at a glance -->
        <div
          :class="[
            'relative flex w-20 shrink-0 flex-col items-center justify-center gap-0.5 border-r-2 border-dashed border-gray-200 bg-gradient-to-b to-white px-2 py-4 text-center',
            stubTone[booking.status].bg,
          ]"
        >
          <span :class="['text-[11px] font-bold uppercase tracking-widest', stubTone[booking.status].text]">
            {{ dayParts(booking.scheduledFor).month }}
          </span>
          <span class="text-3xl font-bold leading-none tracking-tight text-gray-900">
            {{ dayParts(booking.scheduledFor).day }}
          </span>
          <span class="text-[11px] font-medium text-gray-500">{{ dayParts(booking.scheduledFor).weekday }}</span>
          <span class="mt-2 rounded-md bg-white px-1.5 py-0.5 text-[11px] font-semibold tabular-nums text-gray-700 shadow-sm ring-1 ring-gray-200">
            {{ dayParts(booking.scheduledFor).time }}
          </span>
          <!-- ticket notches -->
          <span class="absolute -right-[9px] -top-2 h-4 w-4 rounded-full bg-gray-100"></span>
          <span class="absolute -bottom-2 -right-[9px] h-4 w-4 rounded-full bg-gray-100"></span>
        </div>

        <div class="flex min-w-0 flex-1 flex-col p-4">
          <div class="flex items-center justify-between gap-2">
            <span class="font-mono text-[11px] font-semibold tracking-wide text-gray-400">{{ booking.bookingNumber }}</span>
            <span
              :class="[
                'inline-flex shrink-0 items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-semibold',
                statusStyles[booking.status],
              ]"
            >
              <span :class="['h-1.5 w-1.5 rounded-full', statusAccent[booking.status]]"></span>
              {{ statusLabels[booking.status] }}
            </span>
          </div>

          <h3 class="mt-1.5 truncate text-base font-semibold text-gray-900">{{ booking.customerName }}</h3>
          <p class="mt-1 flex flex-wrap items-center gap-1.5">
            <span class="inline-flex rounded-full bg-slate-100 px-2 py-0.5 text-[11px] font-semibold text-slate-700">
              {{ labelForService(booking.serviceType) }}
            </span>
            <span class="inline-flex rounded-full bg-slate-100 px-2 py-0.5 text-[11px] font-semibold text-slate-700">
              {{ formatDuration(booking.durationMinutes) }}
            </span>
            <span
              v-if="booking.isRecurring && booking.recurrence"
              class="inline-flex rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-semibold text-amber-800"
              title="This booking repeats"
            >
              &#8635; {{ recurrenceLabels[booking.recurrence] ?? booking.recurrence }}
            </span>
          </p>

          <div class="mt-3 space-y-1.5 text-sm">
            <div class="flex items-center gap-2">
              <template v-if="booking.cleaners?.length">
                <span class="flex -space-x-1.5">
                  <span
                    v-for="c in booking.cleaners.slice(0, 3)"
                    :key="c.id"
                    :title="c.name"
                    :class="[
                      'flex h-6 w-6 items-center justify-center rounded-full text-[10px] font-bold text-white ring-2 ring-white',
                      c.role === 'primary' ? 'bg-navy-600' : 'bg-slate-400',
                    ]"
                  >{{ initials(c.name) }}</span>
                </span>
                <span class="min-w-0 truncate text-gray-700">{{ cleanerSummary(booking) }}</span>
              </template>
              <span
                v-else
                class="inline-flex items-center gap-1 rounded-full bg-amber-50 px-2 py-0.5 text-xs font-semibold text-amber-700 ring-1 ring-amber-200"
              >
                <span class="h-1.5 w-1.5 animate-pulse rounded-full bg-amber-500"></span>
                Needs a cleaner
              </span>
            </div>
            <div v-if="booking.address" class="flex items-center gap-2 text-gray-500">
              <svg class="h-4 w-4 shrink-0 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M17.657 16.657L13.414 20.9a2 2 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
              <span class="min-w-0 truncate">{{ booking.address }}</span>
            </div>
            <p v-if="booking.notes" class="line-clamp-1 text-xs italic text-gray-400">{{ booking.notes }}</p>
          </div>

          <div class="mt-auto flex flex-wrap items-center gap-1.5 pt-4">
            <button
              type="button"
              class="rounded-full bg-gray-100 px-3 py-1 text-xs font-semibold text-gray-700 hover:bg-gray-200"
              @click="viewing = booking"
            >
              Details
            </button>
            <button
              v-if="canEdit"
              type="button"
              class="rounded-full bg-gray-100 px-3 py-1 text-xs font-semibold text-gray-700 hover:bg-gray-200"
              @click="openEdit(booking)"
            >
              Edit
            </button>
            <button
              v-if="canCollect && booking.status === 'completed'"
              type="button"
              title="Customer pays now: bill the job and issue the receipt"
              class="rounded-full bg-green-600 px-3 py-1 text-xs font-semibold text-white hover:bg-green-700"
              @click="collecting = booking"
            >
              Collect payment
            </button>
            <button
              v-if="canInvoice && isBillableBooking(booking.status)"
              type="button"
              :title="booking.status === 'completed' ? 'Customer pays later: send an invoice' : 'Invoice before the job, e.g. for a 50% deposit'"
              :disabled="invoicingId === booking.id"
              class="rounded-full bg-navy-600 px-3 py-1 text-xs font-semibold text-white hover:bg-navy-700 disabled:opacity-50"
              @click="handleInvoice(booking)"
            >
              {{ invoicingId === booking.id ? "Invoicing..." : "Invoice" }}
            </button>
            <a
              :href="`/checklists?booking=${booking.id}`"
              class="rounded-full bg-navy-50 px-3 py-1 text-xs font-semibold text-navy-700 hover:bg-navy-100"
            >
              Checklist
            </a>
            <button
              v-if="canDelete"
              type="button"
              class="ml-auto rounded-full p-1.5 text-gray-400 transition hover:bg-red-50 hover:text-red-600"
              :aria-label="`Delete ${booking.bookingNumber}`"
              title="Delete"
              @click="pendingDelete = booking"
            >
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
            </button>
          </div>
        </div>
      </article>
    </div>

    <div
      v-if="!loading && totalPages > 1"
      class="flex items-center justify-between border-t border-gray-200 px-6 py-3"
    >
      <p class="text-sm text-gray-500">Page {{ page }} of {{ totalPages }}</p>
      <div class="flex gap-2">
        <button
          type="button"
          :disabled="page <= 1"
          class="rounded-lg border border-gray-200 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-100 disabled:opacity-40 disabled:hover:bg-transparent"
          @click="page--"
        >
          Previous
        </button>
        <button
          type="button"
          :disabled="page >= totalPages"
          class="rounded-lg border border-gray-200 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-100 disabled:opacity-40 disabled:hover:bg-transparent"
          @click="page++"
        >
          Next
        </button>
      </div>
    </div>
    </div>
  </div>

  <BookingDetailPanel
    v-if="viewing"
    :booking="viewing"
    :can-edit="canEdit"
    :can-invoice="canInvoice"
    :can-collect="canCollect"
    :invoicing="invoicingId === viewing.id"
    @close="viewing = null"
    @open="viewing = $event"
    @edit="editFromPanel"
    @invoice="handleInvoice"
    @collect="collectFromPanel"
  />

  <BookingForm
    v-if="showForm"
    :key="editingBooking?.id ?? 'new'"
    :booking="editingBooking"
    :prefill="prefill"
    @save="handleSave"
    @cancel="closeForm"
  />

  <RecordPaymentDialog
    v-if="collecting"
    :title="`Collect payment — ${collecting.bookingNumber}`"
    :subtitle="`${collecting.customerName}${collecting.price != null ? ` · price ${collecting.price.toFixed(2)} before VAT` : ''}`"
    :booking-id="collecting.id"
    :price="collecting.price ?? null"
    :currency="currencyCode()"
    @done="onCollected"
    @cancel="collecting = null"
  />

  <ConfirmDialog
    v-if="pendingInvoice"
    :title="`Invoice ${pendingInvoice.bookingNumber}`"
    :message="`${pendingInvoice.customerName} · ${labelForService(pendingInvoice.serviceType)}`"
    confirm-label="Create invoice"
    tone="primary"
    @confirm="confirmInvoice"
    @cancel="pendingInvoice = null"
  >
    <div class="mt-3 space-y-3">
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">
          {{ pricesIncludeVat ? "Price for this job (VAT included)" : "Price for this job (before VAT)" }}
        </span>
        <input
          v-model="invoiceAmount"
          type="number"
          min="0"
          step="0.01"
          inputmode="decimal"
          class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
          placeholder="Enter the price"
        />
      </label>
      <p class="text-xs text-gray-500">
        Pre-filled from {{ pendingInvoice.price != null ? "this booking" : "the service price in Settings" }}.
        Change it here for this job only — no need to edit Settings.
      </p>
      <label class="flex items-center gap-2 text-sm text-gray-700">
        <input v-model="savePriceOnBooking" type="checkbox" class="h-4 w-4 rounded border-gray-300" />
        Also save this price on the booking
      </label>
      <p v-if="invoiceAmountValue == null" class="text-xs text-red-600">Enter a price greater than zero.</p>
      <VatBreakdown :price="invoiceAmountValue" :currency="currencyCode()" />
    </div>
  </ConfirmDialog>

  <ConfirmDialog
    v-if="pendingDelete"
    title="Delete booking"
    :message="`Are you sure you want to delete ${pendingDelete.bookingNumber}? This action cannot be undone.`"
    confirm-label="Delete booking"
    :busy="deleting"
    @confirm="handleDelete"
    @cancel="pendingDelete = null"
  />
</template>
