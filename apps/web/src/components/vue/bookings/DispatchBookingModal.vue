<script setup lang="ts">
import { computed, ref, useId } from "vue";
import { useModalA11y } from "../ui/useModalA11y";
import type { Booking, BookingStatus } from "../../../lib/bookings";
import { serviceLabel } from "../../../lib/settings";

const props = defineProps<{
  bookings: Booking[];
}>();

const emit = defineEmits<{
  close: [];
}>();

const titleId = useId();
const { container } = useModalA11y(() => emit("close"));

const activeIndex = ref(0);

const current = computed<Booking>(
  () => props.bookings[activeIndex.value] ?? props.bookings[0],
);

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
  confirmed: "bg-navy-50 text-navy-700",
  in_progress: "bg-green-50 text-green-700",
  completed: "bg-gray-100 text-gray-600",
  cancelled: "bg-gray-100 text-gray-400",
  no_show: "bg-red-50 text-red-700",
  rescheduled: "bg-navy-50 text-navy-700",
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
  const live = serviceLabel(id);
  return live !== id ? live : (serviceTypeLabels[id] ?? id);
}

function fullDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

function timeRange(): string {
  const start = new Date(current.value.scheduledFor);
  const end = new Date(start.getTime() + current.value.durationMinutes * 60_000);
  const fmt = (x: Date) =>
    x.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
  return `${fmt(start)} – ${fmt(end)}`;
}

function chipTime(iso: string, minutes: number): string {
  const start = new Date(iso);
  const end = new Date(start.getTime() + minutes * 60_000);
  const fmt = (x: Date) =>
    x.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
  return `${fmt(start)}–${fmt(end)}`;
}

function durationHuman(minutes: number): string {
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  if (h === 0) return `${m} min`;
  if (m === 0) return `${h} hr`;
  return `${h} hr ${m} min`;
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
    <div class="absolute inset-0 bg-black/50" @click="emit('close')"></div>

    <div class="relative w-full max-w-lg rounded-2xl bg-white shadow-xl ring-1 ring-gray-100">
      <div class="flex items-start justify-between gap-4 border-b border-gray-200 px-6 py-4">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <h2 :id="titleId" class="text-base font-semibold text-gray-900">
              {{ current.bookingNumber }}
            </h2>
            <span
              :class="[
                'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
                statusStyles[current.status],
              ]"
            >
              {{ statusLabels[current.status] }}
            </span>
          </div>
          <p class="mt-1 text-sm text-gray-500">Booking details</p>
        </div>
        <button
          type="button"
          aria-label="Close"
          class="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100 hover:text-gray-900"
          @click="emit('close')"
        >
          <svg
            class="h-5 w-5"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        </button>
      </div>

      <div
        v-if="bookings.length > 1"
        class="flex flex-wrap gap-2 border-b border-gray-200 px-6 py-3"
      >
        <button
          v-for="(b, i) in bookings"
          :key="b.id"
          type="button"
          :class="[
            'rounded-lg border px-3 py-1.5 text-xs font-medium',
            i === activeIndex
              ? 'border-navy-600 bg-navy-600 text-white'
              : 'border-gray-200 text-gray-600 hover:bg-gray-50',
          ]"
          @click="activeIndex = i"
        >
          {{ b.bookingNumber }} · {{ chipTime(b.scheduledFor, b.durationMinutes) }}
        </button>
      </div>

      <div class="grid grid-cols-1 gap-4 px-6 py-5 sm:grid-cols-2">
        <div>
          <p class="text-xs font-medium uppercase tracking-wide text-gray-500">Customer</p>
          <p class="mt-1 text-sm font-semibold text-gray-900">{{ current.customerName }}</p>
        </div>
        <div>
          <p class="text-xs font-medium uppercase tracking-wide text-gray-500">Service type</p>
          <p class="mt-1 text-sm font-semibold text-gray-900">{{ labelForService(current.serviceType) }}</p>
        </div>
        <div class="sm:col-span-2">
          <p class="text-xs font-medium uppercase tracking-wide text-gray-500">Schedule</p>
          <p class="mt-1 text-sm text-gray-900">{{ fullDate(current.scheduledFor) }}</p>
          <p class="text-sm text-gray-600">
            {{ timeRange() }} · {{ durationHuman(current.durationMinutes) }}
          </p>
        </div>
        <div class="sm:col-span-2">
          <p class="text-xs font-medium uppercase tracking-wide text-gray-500">Location</p>
          <p class="mt-1 text-sm text-gray-900">{{ current.address }}</p>
        </div>
        <div class="sm:col-span-2">
          <p class="text-xs font-medium uppercase tracking-wide text-gray-500">
            Assigned team
          </p>
          <ul
            v-if="current.cleaners && current.cleaners.length > 0"
            class="mt-1 space-y-1.5"
          >
            <li
              v-for="c in current.cleaners"
              :key="c.id"
              class="flex items-center justify-between rounded-lg border border-gray-200 px-3 py-2"
            >
              <span class="text-sm font-medium text-gray-900">{{ c.name }}</span>
              <span
                :class="[
                  'inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium',
                  c.role === 'primary'
                    ? 'bg-navy-50 text-navy-700'
                    : 'bg-gray-100 text-gray-600',
                ]"
              >
                {{ c.role === "primary" ? "Lead cleaner" : "Crew" }}
              </span>
            </li>
          </ul>
          <p v-else-if="current.assignedCleaner" class="mt-1 text-sm text-gray-900">
            {{ current.assignedCleaner }}
          </p>
          <p v-else class="mt-1 text-sm text-gray-400">Unassigned</p>
        </div>
        <div v-if="current.notes" class="sm:col-span-2">
          <p class="text-xs font-medium uppercase tracking-wide text-gray-500">Notes</p>
          <p class="mt-1 whitespace-pre-wrap text-sm text-gray-900">{{ current.notes }}</p>
        </div>
      </div>

      <div class="flex justify-end gap-3 border-t border-gray-200 px-6 py-4">
        <button
          type="button"
          class="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100"
          @click="emit('close')"
        >
          Close
        </button>
        <a
          href="/bookings"
          class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700"
        >
          Open in Bookings
        </a>
      </div>
    </div>
  </div>
</template>