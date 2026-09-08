<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { getBookings, type Booking, type BookingStatus } from "../../../lib/bookings";
import { getCleaner, updateCleaner, type Cleaner, type CleanerStatus, type UpdateCleanerInput } from "../../../lib/cleaners";
import { showToast } from "../../../lib/toast";
import { ApiError } from "../../../lib/api";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";
import { serviceLabel } from "../../../lib/settings";
import ConfirmDialog from "../ui/ConfirmDialog.vue";

const props = defineProps<{ cleanerId: number }>();

const canEdit = computed(() =>
  hasPermission(getSessionUser()?.role, "cleaners.update"),
);

const statusLabels: Record<CleanerStatus, string> = {
  available: "Available",
  assigned: "Assigned",
  on_leave: "On Leave",
  inactive: "Inactive",
};

const statusStyles: Record<CleanerStatus, string> = {
  available: "bg-green-50 text-green-700",
  assigned: "bg-navy-50 text-navy-700",
  on_leave: "bg-amber-50 text-amber-700",
  inactive: "bg-gray-100 text-gray-600",
};

const bookingStatusStyles: Record<BookingStatus, string> = {
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

const cleaner = ref<Cleaner | null>(null);
const bookings = ref<Booking[]>([]);
const loading = ref(true);
const notFound = ref(false);
const saving = ref(false);
const edited = ref<Partial<Cleaner>>({});
const confirmingInactive = ref(false);

const upcoming = computed(() =>
  bookings.value
    .filter((b) => !["completed", "cancelled", "no_show"].includes(b.status))
    .sort((a, b) => a.scheduledFor.localeCompare(b.scheduledFor)),
);

const past = computed(() =>
  bookings.value
    .filter((b) => ["completed", "cancelled", "no_show"].includes(b.status))
    .sort((a, b) => b.scheduledFor.localeCompare(a.scheduledFor)),
);

async function load() {
  loading.value = true;
  try {
    cleaner.value = await getCleaner(props.cleanerId);
    bookings.value = await getBookings({ cleaner: props.cleanerId, limit: 200 });
    edited.value = { ...cleaner.value };
  } catch (err) {
    notFound.value = err instanceof ApiError && err.status === 404;
    if (!notFound.value) showToast((err as Error).message ?? "Failed to load cleaner", "error");
  } finally {
    loading.value = false;
  }
}

async function save() {
  if (!cleaner.value) return;
  saving.value = true;
  try {
    const updates: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(edited.value)) {
      if (k === "id" || k === "createdAt") continue;
      if (cleaner.value[k as keyof Cleaner] !== v) updates[k] = v;
    }
    if (Object.keys(updates).length > 0) {
      const updated = await updateCleaner(cleaner.value.id, updates as UpdateCleanerInput);
      cleaner.value = updated;
      edited.value = { ...updated };
      showToast("Cleaner updated", "success");
    }
  } catch (err) {
    showToast(err instanceof Error && err.message ? err.message : "Failed to update cleaner", "error");
  } finally {
    saving.value = false;
  }
}

function statusCanActivate(target: CleanerStatus): boolean {
  return target !== "inactive";
}

function requestStatus(target: CleanerStatus) {
  if (target === "inactive") {
    confirmingInactive.value = true;
    return;
  }
  applyStatus(target);
}

function applyStatus(target: CleanerStatus) {
  edited.value.status = target;
  save().then(() => {
    confirmingInactive.value = false;
  });
}

onMounted(load);
</script>

<template>
  <div class="p-6">
    <div v-if="loading" class="flex items-center justify-center py-20">
      <p class="text-sm text-gray-500">Loading cleaner…</p>
    </div>

    <div v-else-if="notFound" class="flex flex-col items-center py-20 text-center">
      <p class="text-sm font-medium text-gray-900">Cleaner not found</p>
      <p class="mt-1 text-sm text-gray-500">It may have been removed.</p>
      <a href="/cleaners" class="mt-4 rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700">
        Back to Cleaners
      </a>
    </div>

    <div v-else-if="cleaner" class="space-y-6">
      <div class="flex items-start justify-between gap-4">
        <div class="flex items-center gap-4">
          <div class="flex h-14 w-14 items-center justify-center rounded-full bg-navy-600 text-lg font-semibold text-white">
            {{ cleaner.firstName[0] }}{{ cleaner.lastName[0] }}
          </div>
          <div>
            <h1 class="text-xl font-semibold text-gray-900">
              {{ cleaner.firstName }} {{ cleaner.lastName }}
            </h1>
            <div class="mt-1 flex items-center gap-2">
              <span
                :class="[
                  'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
                  statusStyles[cleaner.status],
                ]"
              >
                {{ statusLabels[cleaner.status] }}
              </span>
              <span class="text-xs text-gray-500">Joined {{ new Date(cleaner.createdAt).toLocaleDateString() }}</span>
            </div>
          </div>
        </div>
        <a href="/cleaners" class="rounded-lg border border-gray-200 px-3 py-2 text-sm font-medium text-gray-600 hover:bg-gray-100">
          ← Back
        </a>
      </div>

      <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div class="rounded-lg border border-gray-200 bg-white lg:col-span-1">
          <div class="border-b border-gray-200 px-5 py-4 text-sm font-semibold text-gray-900">
            Profile
          </div>
          <div class="space-y-3 p-5 text-sm">
            <div>
              <label class="block text-xs font-medium uppercase tracking-wide text-gray-500">First name</label>
              <input
                v-model="edited.firstName"
                :disabled="!canEdit"
                class="mt-1 w-full rounded-lg border border-gray-200 px-3 py-2 text-sm disabled:bg-gray-50 disabled:text-gray-500 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
              />
            </div>
            <div>
              <label class="block text-xs font-medium uppercase tracking-wide text-gray-500">Last name</label>
              <input
                v-model="edited.lastName"
                :disabled="!canEdit"
                class="mt-1 w-full rounded-lg border border-gray-200 px-3 py-2 text-sm disabled:bg-gray-50 disabled:text-gray-500 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
              />
            </div>
            <div>
              <label class="block text-xs font-medium uppercase tracking-wide text-gray-500">Phone</label>
              <input
                v-model="edited.phone"
                :disabled="!canEdit"
                class="mt-1 w-full rounded-lg border border-gray-200 px-3 py-2 text-sm disabled:bg-gray-50 disabled:text-gray-500 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
              />
            </div>
            <div>
              <label class="block text-xs font-medium uppercase tracking-wide text-gray-500">Email</label>
              <input
                v-model="edited.email"
                type="email"
                :disabled="!canEdit"
                class="mt-1 w-full rounded-lg border border-gray-200 px-3 py-2 text-sm disabled:bg-gray-50 disabled:text-gray-500 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
              />
            </div>
            <div>
              <label class="block text-xs font-medium uppercase tracking-wide text-gray-500">Skills</label>
              <textarea
                v-model="edited.skills"
                rows="2"
                :disabled="!canEdit"
                class="mt-1 w-full rounded-lg border border-gray-200 px-3 py-2 text-sm disabled:bg-gray-50 disabled:text-gray-500 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
              ></textarea>
            </div>
            <div>
              <label class="block text-xs font-medium uppercase tracking-wide text-gray-500">Status</label>
              <select
                :value="edited.status"
                :disabled="!canEdit"
                @change="requestStatus(($event.target as HTMLSelectElement).value as CleanerStatus)"
                class="mt-1 w-full rounded-lg border border-gray-200 px-3 py-2 text-sm disabled:bg-gray-50 disabled:text-gray-500 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
              >
                <option v-for="(label, value) in statusLabels" :key="value" :value="value">
                  {{ label }}
                </option>
              </select>
            </div>
            <button
              v-if="canEdit"
              type="button"
              class="mt-2 w-full rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50"
              :disabled="saving"
              @click="save"
            >
              {{ saving ? "Saving…" : "Save Changes" }}
            </button>
          </div>
        </div>

        <div class="space-y-6 lg:col-span-2">
          <div class="rounded-lg border border-gray-200 bg-white">
            <div class="flex items-center justify-between border-b border-gray-200 px-5 py-4">
              <h2 class="text-sm font-semibold text-gray-900">Upcoming Assignments</h2>
              <span class="text-xs text-gray-500">{{ upcoming.length }}</span>
            </div>
            <div v-if="upcoming.length === 0" class="px-5 py-8 text-center text-sm text-gray-500">
              No upcoming assignments.
            </div>
            <ul v-else class="divide-y divide-gray-100">
              <li v-for="b in upcoming" :key="b.id" class="flex flex-wrap items-center justify-between gap-2 px-5 py-3">
                <div class="min-w-0">
                  <p class="truncate text-sm font-medium text-gray-900">
                    {{ b.customerName }}
                    <span class="ml-1 text-xs font-normal text-gray-500">{{ b.bookingNumber }}</span>
                  </p>
                  <p class="truncate text-xs text-gray-500">
                    {{ new Date(b.scheduledFor).toLocaleString() }}
                    <template v-if="(b.cleaners ?? []).some((c) => c.role === 'crew')">
                      · team member
                    </template>
                  </p>
                </div>
                <div class="flex items-center gap-2">
                  <span
                    :class="[
                      'inline-flex rounded-full px-2 py-0.5 text-xs font-medium',
                      bookingStatusStyles[b.status],
                    ]"
                  >
                    {{ b.status.replace("_", " ") }}
                  </span>
                  <a href="/bookings" class="text-xs font-medium text-navy-600 hover:text-navy-700">View</a>
                </div>
              </li>
            </ul>
          </div>

          <div class="rounded-lg border border-gray-200 bg-white">
            <div class="flex items-center justify-between border-b border-gray-200 px-5 py-4">
              <h2 class="text-sm font-semibold text-gray-900">Assignment History</h2>
              <span class="text-xs text-gray-500">{{ past.length }}</span>
            </div>
            <div v-if="past.length === 0" class="px-5 py-8 text-center text-sm text-gray-500">
              No past assignments.
            </div>
            <ul v-else class="divide-y divide-gray-100">
              <li v-for="b in past" :key="b.id" class="flex flex-wrap items-center justify-between gap-2 px-5 py-3">
                <div class="min-w-0">
                  <p class="truncate text-sm font-medium text-gray-900">
                    {{ b.customerName }}
                    <span class="ml-1 text-xs font-normal text-gray-500">{{ b.bookingNumber }}</span>
                  </p>
                  <p class="truncate text-xs text-gray-500">
                    {{ new Date(b.scheduledFor).toLocaleDateString() }} · {{ labelForService(b.serviceType) }}
                  </p>
                </div>
                <span
                  :class="[
                    'inline-flex rounded-full px-2 py-0.5 text-xs font-medium',
                    bookingStatusStyles[b.status],
                  ]"
                >
                  {{ b.status.replace("_", " ") }}
                </span>
              </li>
            </ul>
          </div>
        </div>
      </div>
    </div>

    <ConfirmDialog
      v-if="confirmingInactive"
      title="Deactivate cleaner?"
      message="Inactive cleaners are hidden from new-booking dropdowns but their history is kept."
      confirmLabel="Deactivate"
      @cancel="confirmingInactive = false"
      @confirm="applyStatus('inactive')"
    />
  </div>
</template>