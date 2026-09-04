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
import BookingForm from "./BookingForm.vue";
import ConfirmDialog from "../ui/ConfirmDialog.vue";

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
  pending: "bg-amber-50 text-amber-700",
  confirmed: "bg-sky-50 text-sky-700",
  in_progress: "bg-indigo-50 text-indigo-700",
  completed: "bg-emerald-50 text-emerald-700",
  cancelled: "bg-slate-100 text-slate-600",
  no_show: "bg-red-50 text-red-700",
  rescheduled: "bg-violet-50 text-violet-700",
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

const bookings = ref<Booking[]>([]);
const loading = ref(true);
const search = ref("");
const statusFilter = ref<BookingStatus | "">("");
const page = ref(1);
const pageSize = 5;

const showForm = ref(false);
const editingBooking = ref<Booking | undefined>(undefined);
const saving = ref(false);
const pendingDelete = ref<Booking | null>(null);
const deleting = ref(false);

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
      const updated = await updateBooking(editingBooking.value.id, input);
      bookings.value = bookings.value.map((b) =>
        b.id === updated.id ? updated : b,
      );
      showToast("Booking updated", "success");
    } else {
      const created = await createBooking(input);
      bookings.value = [created, ...bookings.value];
      showToast("Booking created", "success");
    }
    closeForm();
  } catch {
    showToast("Failed to save booking", "error");
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
  } catch {
    showToast("Failed to delete booking", "error");
  } finally {
    deleting.value = false;
    pendingDelete.value = null;
  }
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString();
}

onMounted(fetchBookings);
</script>

<template>
  <div class="rounded-xl border border-slate-200 bg-white">
    <div
      class="flex flex-col gap-3 border-b border-slate-200 px-6 py-4 sm:flex-row sm:items-center"
    >
      <h2 class="text-base font-semibold text-slate-900">Bookings</h2>
      <span
        class="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-600"
      >
        {{ filteredBookings.length }}
      </span>

      <div class="sm:ml-auto flex flex-col gap-2 sm:flex-row">
        <input
          v-model="search"
          @input="page = 1"
          type="search"
          placeholder="Search booking or customer..."
          class="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 sm:w-56"
        />
        <select
          v-model="statusFilter"
          @change="page = 1"
          class="rounded-lg border border-slate-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
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
          type="button"
          class="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
          @click="openCreate"
        >
          New Booking
        </button>
      </div>
    </div>

    <div v-if="loading" class="px-6 py-16 text-center">
      <p class="text-sm text-slate-500">Loading bookings...</p>
    </div>

    <div
      v-else-if="filteredBookings.length === 0"
      class="px-6 py-16 text-center"
    >
      <p class="text-sm font-medium text-slate-900">No bookings found</p>
      <p class="mt-1 text-sm text-slate-500">
        Try adjusting your filters or create a new booking.
      </p>
    </div>

    <div v-else class="overflow-x-auto">
      <table class="w-full text-left text-sm">
        <thead>
          <tr
            class="border-b border-slate-200 text-xs uppercase tracking-wide text-slate-500"
          >
            <th class="px-6 py-3 font-medium">Booking #</th>
            <th class="px-6 py-3 font-medium">Customer</th>
            <th class="px-6 py-3 font-medium">Service Type</th>
            <th class="px-6 py-3 font-medium">Scheduled</th>
            <th class="px-6 py-3 font-medium">Cleaner</th>
            <th class="px-6 py-3 font-medium">Status</th>
            <th class="px-6 py-3 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          <tr
            v-for="booking in pagedBookings"
            :key="booking.id"
            class="hover:bg-slate-50"
          >
            <td class="px-6 py-4 font-mono text-xs text-indigo-600">
              {{ booking.bookingNumber }}
            </td>
            <td class="px-6 py-4 font-medium text-slate-900">
              {{ booking.customerName }}
            </td>
            <td class="px-6 py-4 text-slate-600">
              {{ serviceTypeLabels[booking.serviceType] ?? booking.serviceType }}
            </td>
            <td class="px-6 py-4 text-slate-600">
              {{ formatDate(booking.scheduledFor) }}
            </td>
            <td class="px-6 py-4 text-slate-600">
              {{ booking.assignedCleaner }}
            </td>
            <td class="px-6 py-4">
              <span
                :class="[
                  'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
                  statusStyles[booking.status],
                ]"
              >
                {{ statusLabels[booking.status] }}
              </span>
            </td>
            <td class="px-6 py-4 text-right whitespace-nowrap">
              <button
                type="button"
                class="rounded-lg px-2 py-1 text-sm font-medium text-slate-600 hover:bg-slate-100 hover:text-slate-900"
                @click="openEdit(booking)"
              >
                Edit
              </button>
              <button
                type="button"
                class="rounded-lg px-2 py-1 text-sm font-medium text-red-600 hover:bg-red-50"
                @click="pendingDelete = booking"
              >
                Delete
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div
      v-if="!loading && totalPages > 1"
      class="flex items-center justify-between border-t border-slate-200 px-6 py-3"
    >
      <p class="text-sm text-slate-500">Page {{ page }} of {{ totalPages }}</p>
      <div class="flex gap-2">
        <button
          type="button"
          :disabled="page <= 1"
          class="rounded-lg border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-40 disabled:hover:bg-transparent"
          @click="page--"
        >
          Previous
        </button>
        <button
          type="button"
          :disabled="page >= totalPages"
          class="rounded-lg border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-40 disabled:hover:bg-transparent"
          @click="page++"
        >
          Next
        </button>
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
