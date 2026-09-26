<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
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
import { getWorkspaceSettings, serviceLabel } from "../../../lib/settings";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";
import { createInvoice } from "../../../lib/invoices";
import BookingForm from "./BookingForm.vue";
import ConfirmDialog from "../ui/ConfirmDialog.vue";
import MyJobs from "./MyJobs.vue";

const role = getSessionUser()?.role;
const canCreate = computed(() => hasPermission(role, "bookings.create"));
// Cleaners hold bookings.update only to progress their own jobs (My Jobs);
// the full edit form is office-only and the API rejects it for them.
const canEdit = computed(() => role !== "CLEANER" && hasPermission(role, "bookings.update"));
const canDelete = computed(() => hasPermission(role, "bookings.delete"));
const canInvoice = computed(() => hasPermission(role, "invoices.create"));

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

async function fetchBookings() {
  loading.value = true;
  try {
    // Warm the workspace settings cache so service names render from the catalog.
    await getWorkspaceSettings().catch(() => null);
    bookings.value = await getBookings();
  } catch {
    showToast("Failed to load bookings", "error");
  } finally {
    loading.value = false;
  }
}

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
      showToast("Booking created", "success");
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

async function handleInvoice(booking: Booking) {
  invoicingId.value = booking.id;
  try {
    const created = await createInvoice({ bookingId: booking.id });
    showToast("Invoice created", "success");
    window.location.href = `/invoices/${created.id}`;
  } catch (err) {
    const message =
      err instanceof Error && err.message
        ? err.message
        : "Failed to create invoice";
    showToast(message, "error");
  } finally {
    invoicingId.value = null;
  }
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

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString();
}

onMounted(fetchBookings);
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-gray-100/70">
    <div
      class="flex flex-col gap-3 rounded-t-xl border-b border-gray-200 bg-white px-6 py-4 sm:flex-row sm:items-center"
    >
      <h2 class="text-base font-semibold text-gray-900">Bookings</h2>
      <span
        class="rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-600"
      >
        {{ filteredBookings.length }}
      </span>

      <div class="sm:ml-auto flex flex-col gap-2 sm:flex-row">
        <input
          v-model="search"
          @input="page = 1"
          type="search"
          placeholder="Search booking or customer..."
          class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500 sm:w-56"
        />
        <select
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

    <!-- Cleaner field view (mobile cards with Accept / Check-in actions) -->
    <div v-if="role === 'CLEANER'" class="p-4 lg:hidden">
      <MyJobs :bookings="filteredBookings" :loading="loading" @changed="fetchBookings" />
    </div>

    <div>

    <div v-if="loading" class="px-6 py-16 text-center">
      <p class="text-sm text-gray-500">Loading bookings...</p>
    </div>

    <div
      v-else-if="filteredBookings.length === 0"
      class="px-6 py-16 text-center"
    >
      <p class="text-sm font-medium text-gray-900">No bookings found</p>
      <p class="mt-1 text-sm text-gray-500">
        Try adjusting your filters or create a new booking.
      </p>
    </div>

    <div v-else class="grid grid-cols-1 gap-4 p-4 sm:p-6 md:grid-cols-2 xl:grid-cols-3">
      <article
        v-for="booking in pagedBookings"
        :key="booking.id"
        class="flex flex-col overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm transition hover:shadow-md"
      >
        <!-- Status color bar -->
        <div :class="['h-1.5 w-full', statusAccent[booking.status]]"></div>
        <div class="flex flex-1 flex-col p-5">
        <div class="flex items-center justify-between gap-2">
          <span class="rounded bg-gray-900 px-2 py-0.5 font-mono text-[11px] font-semibold tracking-wide text-white">
            {{ booking.bookingNumber }}
          </span>
          <span
            :class="[
              'inline-flex shrink-0 items-center gap-1 rounded-full px-2.5 py-1 text-xs font-bold',
              statusStyles[booking.status],
            ]"
          >
            <span :class="['h-1.5 w-1.5 rounded-full', statusAccent[booking.status]]"></span>
            {{ statusLabels[booking.status] }}
          </span>
        </div>

        <div class="mt-3 flex items-center gap-3">
          <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-navy-600 text-base font-bold text-white">
            {{ (booking.customerName || "?").charAt(0).toUpperCase() }}
          </span>
          <div class="min-w-0">
            <h3 class="truncate text-base font-bold text-gray-900">
              {{ booking.customerName }}
            </h3>
            <p class="mt-0.5 flex flex-wrap items-center gap-1.5 text-sm">
              <span class="inline-flex rounded-md bg-slate-100 px-2 py-0.5 text-xs font-semibold text-slate-700 ring-1 ring-slate-200">
                {{ labelForService(booking.serviceType) }}
              </span>
              <span
                v-if="booking.isRecurring && booking.recurrence"
                class="inline-flex rounded-md bg-amber-100 px-2 py-0.5 text-xs font-semibold text-amber-800 ring-1 ring-amber-200"
                title="This booking repeats"
              >
                ↻ {{ recurrenceLabels[booking.recurrence] ?? booking.recurrence }}
              </span>
            </p>
          </div>
        </div>

        <dl class="mt-4 space-y-2.5 rounded-lg bg-gray-50 p-3 text-sm ring-1 ring-gray-100">
          <div class="flex items-center gap-2">
            <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-blue-100 text-blue-700"><svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg></span>
            <dd class="min-w-0 font-medium text-gray-800">{{ formatDate(booking.scheduledFor) }}</dd>
          </div>
          <div class="flex items-center gap-2">
            <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-green-100 text-green-700"><svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" /></svg></span>
            <dd class="min-w-0 truncate text-gray-700">{{ cleanerSummary(booking) || "Unassigned" }}</dd>
          </div>
          <div v-if="booking.address" class="flex items-center gap-2">
            <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-orange-100 text-orange-700"><svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M17.657 16.657L13.414 20.9a2 2 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" /></svg></span>
            <dd class="min-w-0 truncate text-gray-700">{{ booking.address }}</dd>
          </div>
          <div v-if="booking.notes" class="flex items-start gap-2">
            <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-yellow-100 text-yellow-700"><svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" /></svg></span>
            <dd class="line-clamp-2 min-w-0 text-gray-500">{{ booking.notes }}</dd>
          </div>
        </dl>

        <div class="mt-4 flex flex-wrap items-center gap-2 border-t border-gray-100 pt-3">
          <button
            v-if="canEdit"
            type="button"
            class="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-sm font-semibold text-gray-700 hover:bg-gray-100"
            @click="openEdit(booking)"
          >
            Edit
          </button>
          <button
            v-if="canInvoice && booking.status === 'completed'"
            type="button"
            :disabled="invoicingId === booking.id"
            class="rounded-lg bg-navy-600 px-3 py-1.5 text-sm font-semibold text-white hover:bg-navy-700 disabled:opacity-50"
            @click="handleInvoice(booking)"
          >
            {{ invoicingId === booking.id ? "Invoicing…" : "Invoice" }}
          </button>
          <a
            :href="`/checklists?booking=${booking.id}`"
            class="rounded-lg border border-navy-200 bg-navy-50 px-3 py-1.5 text-sm font-semibold text-navy-700 hover:bg-navy-100"
          >
            Checklist
          </a>
          <button
            v-if="canDelete"
            type="button"
            class="ml-auto rounded-lg px-2.5 py-1.5 text-sm font-medium text-red-600 hover:bg-red-50"
            @click="pendingDelete = booking"
          >
            Delete
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

  <BookingForm
    v-if="showForm"
    :key="editingBooking?.id ?? 'new'"
    :booking="editingBooking"
    @save="handleSave"
    @cancel="closeForm"
  />

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
