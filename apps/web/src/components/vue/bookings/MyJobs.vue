<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  acceptBooking,
  myJobs,
  updateBooking,
  type Booking,
} from "../../../lib/bookings";
import { checkIn, checkOut, getAttendance } from "../../../lib/attendance";
import { getCleaners, getMyCleanerProfile } from "../../../lib/cleaners";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";
import { isOfflineQueued } from "../../../lib/offline";
import { showToast } from "../../../lib/toast";
import { dateLocale, t, tStatus } from "../../../lib/i18n";
import { getPosition } from "../../../lib/geo";
import LanguageSwitcher from "../ui/LanguageSwitcher.vue";

const props = defineProps<{ bookings: Booking[]; loading: boolean }>();
const emit = defineEmits<{ changed: [] }>();

const user = getSessionUser();
const role = user?.role;
const canUpdate = computed(() => hasPermission(role, "bookings.update"));

const profileId = ref<number | null>(null);
const checkedInToday = ref(false);
const checkedOutToday = ref(false);
const busyId = ref<number | null>(null);

function todayStr(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

async function resolveProfile(): Promise<void> {
  try {
    // CLEANER logins cannot list all cleaners, so resolve the profile linked
    // to this login first; staff previewing the page fall back to email.
    const mine = await getMyCleanerProfile().catch(() => null);
    if (mine) {
      profileId.value = mine.id;
      return;
    }
    const email = (user?.email ?? "").trim().toLowerCase();
    if (!email) return;
    const cleaners = await getCleaners();
    const match = cleaners.find((c) => (c.email ?? "").trim().toLowerCase() === email);
    if (match) profileId.value = match.id;
  } catch {
    /* offline or API error: name fallback still applies in myJobs() */
  }
}

async function loadDayState(): Promise<void> {
  if (profileId.value == null) return;
  try {
    const rows = await getAttendance({ type: "cleaner", person: profileId.value });
    const today = rows.find((r) => r.workDate === todayStr());
    checkedInToday.value = !!today?.checkInAt;
    checkedOutToday.value = !!today?.checkOutAt;
  } catch {
    /* offline: action buttons stay available; server dedupes on sync */
  }
}

const jobs = computed(() => {
  const sorted = [...props.bookings].sort((a, b) =>
    a.scheduledFor.localeCompare(b.scheduledFor),
  );
  if (role !== "CLEANER") return sorted;
  return myJobs(props.bookings, {
    email: user?.email ?? "",
    name: user?.name ?? "",
    cleanerProfileId: profileId.value,
  });
});

function isMine(b: Booking): boolean {
  if (profileId.value != null && (b.cleaners ?? []).some((c) => c.id === profileId.value)) return true;
  const name = (user?.name ?? "").trim().toLowerCase();
  return !!name && b.assignedCleaner.trim().toLowerCase() === name;
}

function isOpen(b: Booking): boolean {
  return b.status === "pending" && (b.cleaners ?? []).length === 0 && !b.assignedCleaner;
}

function mapsUrl(b: Booking): string {
  return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(b.address || b.customerName)}`;
}

function formatWhen(iso: string): string {
  try {
    return new Date(iso).toLocaleString(dateLocale(), {
      weekday: "short",
      day: "numeric",
      month: "short",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return iso;
  }
}

function errMsg(err: unknown, fallback: string): string {
  return err instanceof Error && err.message ? err.message : fallback;
}

async function doAccept(b: Booking): Promise<void> {
  busyId.value = b.id;
  try {
    await acceptBooking(b.id);
    showToast(t("jobs.accepted"), "success");
    emit("changed");
  } catch (err) {
    if (isOfflineQueued(err)) showToast(t("jobs.offlineAccept"), "success");
    else showToast(errMsg(err, t("jobs.acceptFailed")), "error");
  } finally {
    busyId.value = null;
  }
}

async function doCheckIn(b: Booking): Promise<void> {
  if (profileId.value == null) {
    showToast(t("jobs.noProfile"), "error");
    return;
  }
  busyId.value = b.id;
  try {
    // The office can require check-in at the job site; send the GPS fix.
    showToast(t("att.locating"), "info");
    const fix = await getPosition();
    await checkIn("cleaner", profileId.value, fix);
    checkedInToday.value = true;
    showToast(t("jobs.checkedIn"), "success");
    emit("changed");
  } catch (err) {
    if (isOfflineQueued(err)) showToast(t("jobs.offlineCheckIn"), "success");
    else showToast(errMsg(err, t("jobs.checkInFailed")), "error");
  } finally {
    busyId.value = null;
  }
}

async function doCheckOut(b: Booking): Promise<void> {
  if (profileId.value == null) {
    showToast(t("jobs.noProfile"), "error");
    return;
  }
  busyId.value = b.id;
  try {
    await checkOut("cleaner", profileId.value, await getPosition(8000));
    checkedOutToday.value = true;
    showToast(t("jobs.checkedOut"), "success");
    emit("changed");
  } catch (err) {
    if (isOfflineQueued(err)) showToast(t("jobs.offlineCheckOut"), "success");
    else showToast(errMsg(err, t("jobs.checkOutFailed")), "error");
  } finally {
    busyId.value = null;
  }
}

async function doComplete(b: Booking): Promise<void> {
  busyId.value = b.id;
  try {
    await updateBooking(b.id, { status: "completed" });
    showToast(t("jobs.completed"), "success");
    emit("changed");
  } catch (err) {
    if (isOfflineQueued(err)) showToast(t("jobs.offlineComplete"), "success");
    else showToast(errMsg(err, t("jobs.completeFailed")), "error");
  } finally {
    busyId.value = null;
  }
}

onMounted(async () => {
  await resolveProfile();
  await loadDayState();
});
</script>

<template>
  <!-- Mobile-only field cards. Desktop keeps the table below untouched. -->
  <div class="space-y-3 lg:hidden">
    <LanguageSwitcher v-if="role === 'CLEANER'" />
    <div v-if="loading" class="rounded-2xl bg-white p-6 text-center text-sm text-gray-500 ring-1 ring-gray-100">
      {{ t("jobs.loading") }}
    </div>
    <div v-else-if="jobs.length === 0" class="rounded-2xl bg-white p-6 text-center ring-1 ring-gray-100">
      <p class="text-sm font-medium text-gray-900">{{ t("jobs.empty") }}</p>
      <p class="mt-1 text-xs text-gray-500">{{ t("jobs.emptyHint") }}</p>
    </div>
    <article
      v-for="b in jobs"
      :key="b.id"
      class="rounded-2xl bg-white p-4 shadow-sm ring-1 ring-gray-100"
    >
      <div class="flex items-start justify-between gap-2">
        <div class="min-w-0">
          <p class="truncate text-base font-semibold text-gray-900">{{ b.customerName }}</p>
          <p class="mt-0.5 text-sm text-gray-500">{{ formatWhen(b.scheduledFor) }}</p>
        </div>
        <span class="shrink-0 rounded-full bg-navy-50 px-2.5 py-1 text-xs font-medium text-navy-700">
          {{ tStatus(b.status) }}
        </span>
      </div>
      <p class="mt-2 flex items-center gap-1.5 text-sm text-gray-600"><svg class="h-4 w-4 shrink-0 text-orange-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M17.657 16.657L13.414 20.9a2 2 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" /></svg><span class="min-w-0 truncate">{{ b.address || "—" }}</span></p>

      <div class="mt-3 grid grid-cols-1 gap-2">
        <button
          v-if="isOpen(b) && canUpdate"
          type="button"
          :disabled="busyId === b.id"
          class="field-tap w-full rounded-xl bg-navy-600 px-4 py-3 text-base font-semibold text-white active:bg-navy-700 disabled:opacity-50"
          @click="doAccept(b)"
        >
          {{ busyId === b.id ? t("jobs.working") : t("jobs.accept") }}
        </button>
        <template v-if="isMine(b) && (b.status === 'confirmed' || b.status === 'in_progress')">
          <button
            v-if="!checkedInToday"
            type="button"
            :disabled="busyId === b.id"
            class="field-tap w-full rounded-xl bg-green-600 px-4 py-3 text-base font-semibold text-white active:bg-green-700 disabled:opacity-50"
            @click="doCheckIn(b)"
          >
            {{ busyId === b.id ? t("jobs.working") : t("jobs.checkIn") }}
          </button>
          <button
            v-else-if="!checkedOutToday"
            type="button"
            :disabled="busyId === b.id"
            class="field-tap w-full rounded-xl bg-gray-900 px-4 py-3 text-base font-semibold text-white active:bg-gray-700 disabled:opacity-50"
            @click="doCheckOut(b)"
          >
            {{ busyId === b.id ? t("jobs.working") : t("jobs.checkOut") }}
          </button>
          <button
            type="button"
            :disabled="busyId === b.id"
            class="field-tap w-full rounded-xl bg-white px-4 py-3 text-base font-semibold text-navy-700 ring-1 ring-navy-200 active:bg-navy-50 disabled:opacity-50"
            @click="doComplete(b)"
          >
            {{ busyId === b.id ? t("jobs.working") : t("jobs.complete") }}
          </button>
        </template>
        <div class="grid grid-cols-2 gap-2">
          <a
            :href="`/checklists?booking=${b.id}`"
            class="field-tap inline-flex items-center justify-center rounded-xl bg-gray-100 px-4 py-3 text-sm font-semibold text-gray-800 active:bg-gray-200"
          >
            {{ t("jobs.checklist") }}
          </a>
          <a
            :href="mapsUrl(b)"
            target="_blank"
            rel="noopener"
            class="field-tap inline-flex items-center justify-center rounded-xl bg-gray-100 px-4 py-3 text-sm font-semibold text-gray-800 active:bg-gray-200"
          >
            {{ t("jobs.navigate") }}
          </a>
        </div>
      </div>
    </article>
  </div>
</template>
