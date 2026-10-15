<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  acceptBooking,
  getAvailableBookings,
  getBookings,
  updateBooking,
  type Booking,
} from "../../../lib/bookings";
import { checkIn, checkOut, getAttendance } from "../../../lib/attendance";
import { getMyCleanerProfile } from "../../../lib/cleaners";
import { checklistState, getChecklistStatuses, type ChecklistStatus } from "../../../lib/checklists";
import { getSessionUser } from "../../../lib/auth";
import { isOfflineQueued } from "../../../lib/offline";
import { ApiError } from "../../../lib/api";
import { showToast } from "../../../lib/toast";
import { dateLocale, t, tStatus, type MessageKey } from "../../../lib/i18n";
import { getPosition } from "../../../lib/geo";
import { getWorkspaceSettings, serviceLabel } from "../../../lib/settings";
import AppIcon from "../ui/AppIcon.vue";
import LanguageSwitcher from "../ui/LanguageSwitcher.vue";

// The cleaners' field app: their jobs, where to go and what to do. It never
// shows money; the API also strips prices for this role.

type Tab = "today" | "upcoming" | "open" | "done";

const user = getSessionUser();
const firstName = (user?.name ?? "").split(" ")[0] ?? "";

const mine = ref<Booking[]>([]);
const pool = ref<Booking[]>([]);
const loading = ref(true);
const tab = ref<Tab>("today");
const selected = ref<Booking | null>(null);
const busyId = ref<number | null>(null);

// Checklist state per booking, for the badge on each job card.
const checklists = ref<Record<number, ChecklistStatus>>({});

const profileId = ref<number | null>(null);
const checkedIn = ref(false);
const checkedOut = ref(false);
const dayBusy = ref(false);

function dayKey(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}
const todayKey = dayKey(new Date());

function errMsg(err: unknown, fallback: string): string {
  return err instanceof Error && err.message ? err.message : fallback;
}

async function load() {
  loading.value = true;
  try {
    await getWorkspaceSettings().catch(() => null);
    // The server only returns this cleaner's own jobs for this role.
    const [own, open] = await Promise.all([
      getBookings({ limit: 200 }),
      getAvailableBookings().catch(() => [] as Booking[]),
    ]);
    mine.value = own;
    pool.value = open;
    void loadChecklists(own);
  } catch (err) {
    showToast(errMsg(err, t("cj.loadFailed")), "error");
  } finally {
    loading.value = false;
  }
}

async function loadChecklists(jobs: Booking[]) {
  try {
    const rows = await getChecklistStatuses(jobs.filter((b) => b.status !== "cancelled").map((b) => b.id));
    checklists.value = Object.fromEntries(rows.map((r) => [r.bookingId, r]));
  } catch {
    /* badges are a hint; the checklist page still works */
  }
}

const checklistBadge: Record<string, { label: MessageKey; tone: string }> = {
  pending: { label: "cl.badgeTodo", tone: "bg-gray-100 text-gray-700" },
  in_progress: { label: "cl.badgeTodo", tone: "bg-gray-100 text-gray-700" },
  submitted: { label: "cl.badgeSubmitted", tone: "bg-amber-50 text-amber-800" },
  approved: { label: "cl.badgeApproved", tone: "bg-green-50 text-green-700" },
  rework: { label: "cl.badgeRework", tone: "bg-red-100 text-red-700" },
};

function badgeFor(b: Booking) {
  const c = checklists.value[b.id];
  return c ? { ...checklistBadge[checklistState(c)]!, c } : null;
}

function isOpenJob(b: Booking): boolean {
  return b.status === "pending" && (b.cleaners ?? []).length === 0 && !b.assignedCleaner;
}

async function loadDay() {
  try {
    const me = await getMyCleanerProfile().catch(() => null);
    profileId.value = me?.id ?? null;
    if (profileId.value == null) return;
    const rows = await getAttendance({ type: "cleaner", person: profileId.value });
    const today = rows.find((r) => r.workDate === todayKey);
    checkedIn.value = !!today?.checkInAt;
    checkedOut.value = !!today?.checkOutAt;
  } catch {
    /* offline: buttons stay available; the server dedupes on sync */
  }
}

const ACTIVE = ["pending", "confirmed", "in_progress", "rescheduled"];

const lists = computed<Record<Tab, Booking[]>>(() => {
  const byTime = (a: Booking, b: Booking) => a.scheduledFor.localeCompare(b.scheduledFor);
  const today = mine.value.filter((b) => dayKey(new Date(b.scheduledFor)) === todayKey && b.status !== "cancelled");
  const upcoming = mine.value.filter(
    (b) => dayKey(new Date(b.scheduledFor)) > todayKey && ACTIVE.includes(b.status),
  );
  const done = mine.value
    .filter((b) => b.status === "completed")
    .sort((a, b) => b.scheduledFor.localeCompare(a.scheduledFor))
    .slice(0, 30);
  return {
    today: today.sort(byTime),
    upcoming: upcoming.sort(byTime),
    open: [...pool.value].sort(byTime),
    done,
  };
});

const nextJob = computed(() =>
  lists.value.today.find((b) => b.status !== "completed") ?? lists.value.upcoming[0] ?? null,
);

const tabs: { value: Tab; label: MessageKey; empty: MessageKey }[] = [
  { value: "today", label: "cj.today", empty: "cj.emptyToday" },
  { value: "upcoming", label: "cj.upcoming", empty: "cj.emptyUpcoming" },
  { value: "open", label: "cj.open", empty: "cj.emptyOpen" },
  { value: "done", label: "cj.done", empty: "cj.emptyDone" },
];

// --- formatting ---
function time(iso: string): string {
  return new Date(iso).toLocaleTimeString(dateLocale(), { hour: "2-digit", minute: "2-digit" });
}

function dayLabel(iso: string): string {
  return new Date(iso).toLocaleDateString(dateLocale(), { weekday: "short", day: "numeric", month: "short" });
}

function duration(min: number): string {
  if (!min) return "—";
  if (min < 60) return `${min} ${t("cj.min")}`;
  return `${Math.round((min / 60) * 100) / 100} ${t("cj.hours")}`;
}

function endTime(b: Booking): string {
  return time(new Date(new Date(b.scheduledFor).getTime() + b.durationMinutes * 60_000).toISOString());
}

function initials(name: string): string {
  return name.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w.charAt(0).toUpperCase()).join("");
}

function mapsUrl(b: Booking): string {
  return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(b.address || b.customerName)}`;
}

function isMe(id: number): boolean {
  return profileId.value != null && id === profileId.value;
}

const statusTone: Record<string, string> = {
  pending: "bg-amber-100 text-amber-800",
  confirmed: "bg-blue-100 text-blue-800",
  in_progress: "bg-indigo-100 text-indigo-800",
  completed: "bg-green-100 text-green-800",
  cancelled: "bg-gray-200 text-gray-600",
  no_show: "bg-red-100 text-red-700",
  rescheduled: "bg-purple-100 text-purple-800",
};

// --- actions ---
function replace(updated: Booking) {
  const i = mine.value.findIndex((b) => b.id === updated.id);
  if (i !== -1) mine.value[i] = updated;
  else mine.value.push(updated);
  if (selected.value?.id === updated.id) selected.value = updated;
}

async function setStatus(b: Booking, status: "in_progress" | "completed") {
  // Proof of work first: a job with an unsubmitted checklist opens it instead.
  const cl = checklists.value[b.id];
  if (status === "completed" && cl && !cl.submitted && cl.reviewStatus !== "approved") {
    showToast(t("cj.checklistFirst"), "error");
    window.location.href = `/checklists?booking=${b.id}`;
    return;
  }
  busyId.value = b.id;
  try {
    replace(await updateBooking(b.id, { status }));
    showToast(t(status === "completed" ? "jobs.completed" : "cj.started"), "success");
  } catch (err) {
    if (isOfflineQueued(err)) {
      replace({ ...b, status });
      showToast(t("jobs.offlineComplete"), "success");
    } else if (status === "completed" && err instanceof ApiError && err.status === 409) {
      // The server requires the checklist (e.g. none was opened yet).
      showToast(t("cj.checklistFirst"), "error");
      window.location.href = `/checklists?booking=${b.id}`;
    } else {
      showToast(errMsg(err, t(status === "completed" ? "jobs.completeFailed" : "cj.startFailed")), "error");
    }
  } finally {
    busyId.value = null;
  }
}

async function accept(b: Booking) {
  busyId.value = b.id;
  try {
    const taken = await acceptBooking(b.id);
    pool.value = pool.value.filter((x) => x.id !== b.id);
    replace(taken);
    showToast(t("jobs.accepted"), "success");
    selected.value = null;
    tab.value = dayKey(new Date(taken.scheduledFor)) === todayKey ? "today" : "upcoming";
  } catch (err) {
    if (isOfflineQueued(err)) showToast(t("jobs.offlineAccept"), "success");
    else showToast(errMsg(err, t("jobs.acceptFailed")), "error");
  } finally {
    busyId.value = null;
  }
}

async function dayCheckIn() {
  if (profileId.value == null) return showToast(t("jobs.noProfile"), "error");
  dayBusy.value = true;
  try {
    showToast(t("att.locating"), "info");
    await checkIn("cleaner", profileId.value, await getPosition());
    checkedIn.value = true;
    showToast(t("jobs.checkedIn"), "success");
  } catch (err) {
    if (isOfflineQueued(err)) {
      checkedIn.value = true;
      showToast(t("jobs.offlineCheckIn"), "success");
    } else showToast(errMsg(err, t("jobs.checkInFailed")), "error");
  } finally {
    dayBusy.value = false;
  }
}

async function dayCheckOut() {
  if (profileId.value == null) return showToast(t("jobs.noProfile"), "error");
  dayBusy.value = true;
  try {
    await checkOut("cleaner", profileId.value, await getPosition(8000));
    checkedOut.value = true;
    showToast(t("jobs.checkedOut"), "success");
  } catch (err) {
    if (isOfflineQueued(err)) {
      checkedOut.value = true;
      showToast(t("jobs.offlineCheckOut"), "success");
    } else showToast(errMsg(err, t("jobs.checkOutFailed")), "error");
  } finally {
    dayBusy.value = false;
  }
}

onMounted(async () => {
  await loadDay();
  await load();
});
</script>

<template>
  <div class="mx-auto w-full max-w-xl pb-28">
    <div class="mb-3 flex justify-end">
      <LanguageSwitcher />
    </div>

    <!-- Greeting + day status -->
    <section class="relative overflow-hidden rounded-3xl bg-gradient-to-br from-navy-600 to-indigo-700 p-5 text-white shadow-lg">
      <div class="absolute -right-10 -top-10 h-40 w-40 rounded-full bg-white/10"></div>
      <div class="absolute -bottom-12 right-10 h-28 w-28 rounded-full bg-white/5"></div>
      <div class="relative flex items-start justify-between gap-3">
        <div>
          <p class="text-sm text-white/70">{{ new Date().toLocaleDateString(dateLocale(), { weekday: "long", day: "numeric", month: "long" }) }}</p>
          <h1 class="mt-0.5 text-2xl font-bold">{{ t("cj.hello") }}{{ firstName ? `, ${firstName}` : "" }} 👋</h1>
          <p class="mt-1 text-sm text-white/80">
            <strong class="text-white">{{ lists.today.length }}</strong> {{ t("cj.jobsToday") }}
          </p>
        </div>
      </div>

      <div class="relative mt-4 flex items-center justify-between gap-3 rounded-2xl bg-white/10 p-3 backdrop-blur">
        <p class="flex items-center gap-2 text-sm font-medium">
          <span
            :class="['h-2.5 w-2.5 rounded-full', checkedOut ? 'bg-gray-300' : checkedIn ? 'bg-emerald-400' : 'bg-amber-300 animate-pulse']"
          ></span>
          {{ checkedOut ? t("cj.dayOut") : checkedIn ? t("cj.dayIn") : t("cj.dayNotIn") }}
        </p>
        <button
          v-if="!checkedIn"
          type="button"
          :disabled="dayBusy"
          class="field-tap shrink-0 rounded-xl bg-white px-4 py-2 text-sm font-bold text-navy-700 shadow active:scale-95 disabled:opacity-60"
          @click="dayCheckIn"
        >
          {{ dayBusy ? t("jobs.working") : t("jobs.checkIn") }}
        </button>
        <button
          v-else-if="!checkedOut"
          type="button"
          :disabled="dayBusy"
          class="field-tap shrink-0 rounded-xl bg-white/20 px-4 py-2 text-sm font-bold text-white ring-1 ring-white/30 active:scale-95 disabled:opacity-60"
          @click="dayCheckOut"
        >
          {{ dayBusy ? t("jobs.working") : t("jobs.checkOut") }}
        </button>
      </div>
    </section>

    <!-- Next job, big and simple -->
    <section
      v-if="nextJob && !loading"
      class="mt-4 rounded-3xl bg-white p-4 shadow-sm ring-1 ring-gray-200"
    >
      <p class="text-xs font-semibold uppercase tracking-wider text-navy-600">{{ t("cj.next") }}</p>
      <button type="button" class="mt-2 flex w-full items-center gap-4 text-left" @click="selected = nextJob">
        <span class="flex h-16 w-16 shrink-0 flex-col items-center justify-center rounded-2xl bg-navy-50 text-navy-700">
          <span class="text-lg font-bold leading-none">{{ time(nextJob.scheduledFor) }}</span>
          <span class="mt-1 text-[10px] font-medium">{{ dayLabel(nextJob.scheduledFor) }}</span>
        </span>
        <span class="min-w-0 flex-1">
          <span class="block truncate text-base font-semibold text-gray-900">{{ serviceLabel(nextJob.serviceType) }}</span>
          <span class="block truncate text-sm text-gray-500">{{ nextJob.address || nextJob.customerName }}</span>
        </span>
      </button>
      <div class="mt-3 grid grid-cols-2 gap-2">
        <a
          :href="mapsUrl(nextJob)"
          target="_blank"
          rel="noopener"
          class="field-tap flex items-center justify-center gap-2 rounded-xl bg-gray-900 py-3 text-sm font-bold text-white active:scale-95"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7" /></svg>
          {{ t("jobs.navigate") }}
        </a>
        <button
          v-if="nextJob.status !== 'in_progress'"
          type="button"
          :disabled="busyId === nextJob.id"
          class="field-tap rounded-xl bg-emerald-600 py-3 text-sm font-bold text-white active:scale-95 disabled:opacity-60"
          @click="setStatus(nextJob, 'in_progress')"
        >
          {{ busyId === nextJob.id ? t("jobs.working") : t("cj.start") }}
        </button>
        <button
          v-else
          type="button"
          :disabled="busyId === nextJob.id"
          class="field-tap rounded-xl bg-navy-600 py-3 text-sm font-bold text-white active:scale-95 disabled:opacity-60"
          @click="setStatus(nextJob, 'completed')"
        >
          {{ busyId === nextJob.id ? t("jobs.working") : t("jobs.complete") }}
        </button>
      </div>
    </section>

    <!-- Tabs -->
    <nav class="sticky top-0 z-10 -mx-1 mt-5 flex gap-1 overflow-x-auto bg-gray-50/90 px-1 py-2 backdrop-blur [scrollbar-width:none]" aria-label="Jobs">
      <button
        v-for="tb in tabs"
        :key="tb.value"
        type="button"
        :aria-pressed="tab === tb.value"
        :class="[
          'field-tap flex shrink-0 items-center gap-1.5 rounded-full px-4 py-2 text-sm font-semibold transition',
          tab === tb.value ? 'bg-gray-900 text-white shadow' : 'bg-white text-gray-600 ring-1 ring-gray-200',
        ]"
        @click="tab = tb.value"
      >
        {{ t(tb.label) }}
        <span
          v-if="lists[tb.value].length"
          :class="['rounded-full px-1.5 text-[11px]', tab === tb.value ? 'bg-white/20' : tb.value === 'open' ? 'bg-amber-100 text-amber-800' : 'bg-gray-100']"
        >{{ lists[tb.value].length }}</span>
      </button>
      <button
        type="button"
        class="field-tap ml-auto shrink-0 rounded-full bg-white p-2 text-gray-500 ring-1 ring-gray-200"
        :aria-label="t('cj.refresh')"
        @click="load"
      >
        <svg :class="['h-4 w-4', loading ? 'animate-spin' : '']" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
      </button>
    </nav>

    <p v-if="tab === 'open' && lists.open.length" class="mt-1 px-1 text-xs text-gray-500">{{ t("cj.openHint") }}</p>

    <!-- Job list -->
    <div class="mt-2 space-y-3">
      <div v-if="loading" class="space-y-3">
        <div v-for="i in 3" :key="i" class="h-28 animate-pulse rounded-2xl bg-white ring-1 ring-gray-100"></div>
      </div>
      <div
        v-else-if="lists[tab].length === 0"
        class="rounded-2xl bg-white px-6 py-10 text-center ring-1 ring-gray-100"
      >
        <p class="text-3xl">🧹</p>
        <p class="mt-2 text-sm font-medium text-gray-700">{{ t(tabs.find((x) => x.value === tab)!.empty) }}</p>
      </div>

      <article
        v-for="b in lists[tab]"
        v-else
        :key="b.id"
        class="overflow-hidden rounded-2xl bg-white shadow-sm ring-1 ring-gray-200 active:scale-[0.99]"
      >
        <button type="button" class="flex w-full gap-3 p-4 text-left" @click="selected = b">
          <span class="flex w-14 shrink-0 flex-col items-center rounded-xl bg-gray-50 py-2 text-center">
            <span class="text-[10px] font-semibold uppercase text-gray-500">{{ dayLabel(b.scheduledFor).split(" ")[0] }}</span>
            <span class="text-base font-bold leading-tight text-gray-900">{{ time(b.scheduledFor) }}</span>
          </span>
          <span class="min-w-0 flex-1">
            <span class="flex items-start justify-between gap-2">
              <span class="truncate text-[15px] font-semibold text-gray-900">{{ serviceLabel(b.serviceType) }}</span>
              <span :class="['shrink-0 rounded-full px-2 py-0.5 text-[11px] font-semibold capitalize', statusTone[b.status] ?? 'bg-gray-100 text-gray-700']">
                {{ tStatus(b.status) }}
              </span>
            </span>
            <span class="mt-0.5 block text-xs text-gray-500">{{ dayLabel(b.scheduledFor) }} · {{ duration(b.durationMinutes) }}</span>
            <span
              v-if="badgeFor(b)"
              :class="['mt-1.5 inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-semibold', badgeFor(b)!.tone]"
            ><AppIcon name="clipboard" :size="13" /> {{ t(badgeFor(b)!.label) }} · {{ badgeFor(b)!.c.itemsDone }}/{{ badgeFor(b)!.c.itemsTotal }}</span>
            <span class="mt-1.5 flex items-center gap-1.5 text-sm text-gray-700">
              <svg class="h-4 w-4 shrink-0 text-rose-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M17.657 16.657L13.414 20.9a2 2 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
              <span class="truncate">{{ b.address || b.customerName }}</span>
            </span>
            <span v-if="b.cleaners?.length" class="mt-2 flex -space-x-1.5">
              <span
                v-for="c in b.cleaners.slice(0, 4)"
                :key="c.id"
                :title="c.name"
                :class="[
                  'flex h-6 w-6 items-center justify-center rounded-full text-[10px] font-bold text-white ring-2 ring-white',
                  isMe(c.id) ? 'bg-emerald-500' : c.role === 'primary' ? 'bg-navy-600' : 'bg-slate-400',
                ]"
              >{{ initials(c.name) }}</span>
            </span>
          </span>
        </button>
        <div v-if="tab === 'open'" class="border-t border-gray-100 p-3">
          <button
            type="button"
            :disabled="busyId === b.id"
            class="field-tap w-full rounded-xl bg-amber-500 py-3 text-sm font-bold text-white active:scale-95 disabled:opacity-60"
            @click="accept(b)"
          >
            {{ busyId === b.id ? t("jobs.working") : t("jobs.accept") }}
          </button>
        </div>
      </article>
    </div>

    <!-- Job details: bottom sheet -->
    <Transition
      enter-active-class="transition duration-200"
      enter-from-class="opacity-0"
      leave-active-class="transition duration-150"
      leave-to-class="opacity-0"
    >
      <div v-if="selected" class="fixed inset-0 z-50" role="dialog" aria-modal="true" :aria-label="t('cj.details')">
        <div class="absolute inset-0 bg-black/40" @click="selected = null"></div>
        <div class="absolute inset-x-0 bottom-0 mx-auto max-h-[88vh] w-full max-w-xl overflow-y-auto rounded-t-3xl bg-white pb-[calc(env(safe-area-inset-bottom,0px)+1rem)] shadow-2xl">
          <div class="sticky top-0 bg-white px-5 pb-3 pt-3">
            <div class="mx-auto h-1.5 w-10 rounded-full bg-gray-200"></div>
            <div class="mt-3 flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="text-xs font-semibold uppercase tracking-wider text-gray-400">{{ selected.bookingNumber }}</p>
                <h2 class="text-xl font-bold text-gray-900">{{ serviceLabel(selected.serviceType) }}</h2>
              </div>
              <button
                type="button"
                class="field-tap rounded-full bg-gray-100 p-2 text-gray-500"
                :aria-label="t('cj.close')"
                @click="selected = null"
              >
                <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
              </button>
            </div>
          </div>

          <dl class="space-y-3 px-5">
            <div class="flex gap-3 rounded-2xl bg-gray-50 p-3">
              <dt class="sr-only">{{ t("cj.when") }}</dt>
              <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-navy-100 text-navy-700">🕒</span>
              <dd>
                <p class="font-semibold text-gray-900">{{ dayLabel(selected.scheduledFor) }}</p>
                <p class="text-sm text-gray-600">{{ time(selected.scheduledFor) }} – {{ endTime(selected) }} · {{ duration(selected.durationMinutes) }}</p>
              </dd>
            </div>
            <div class="flex gap-3 rounded-2xl bg-gray-50 p-3">
              <dt class="sr-only">{{ t("cj.where") }}</dt>
              <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-rose-100 text-rose-600">📍</span>
              <dd class="min-w-0">
                <p class="font-semibold text-gray-900">{{ selected.customerName }}</p>
                <p class="text-sm text-gray-600">{{ selected.address || "—" }}</p>
              </dd>
            </div>
            <div class="flex gap-3 rounded-2xl bg-gray-50 p-3">
              <dt class="sr-only">{{ t("cj.team") }}</dt>
              <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-emerald-100 text-emerald-700">👥</span>
              <dd class="min-w-0 flex-1">
                <p class="text-xs font-medium text-gray-500">{{ t("cj.team") }}</p>
                <ul class="mt-1 space-y-1">
                  <li v-for="c in selected.cleaners ?? []" :key="c.id" class="flex items-center gap-2 text-sm text-gray-900">
                    <span
                      :class="['flex h-6 w-6 items-center justify-center rounded-full text-[10px] font-bold text-white', isMe(c.id) ? 'bg-emerald-500' : 'bg-navy-600']"
                    >{{ initials(c.name) }}</span>
                    {{ c.name }}
                    <span v-if="isMe(c.id)" class="text-xs text-emerald-700">({{ t("cj.you") }})</span>
                    <span v-if="c.role === 'primary'" class="rounded bg-navy-50 px-1.5 text-[10px] font-semibold text-navy-700">{{ t("cj.lead") }}</span>
                  </li>
                  <li v-if="!(selected.cleaners ?? []).length" class="text-sm text-gray-500">—</li>
                </ul>
              </dd>
            </div>
            <div class="rounded-2xl bg-amber-50 p-3 ring-1 ring-amber-100">
              <dt class="text-xs font-semibold text-amber-800">{{ t("cj.notes") }}</dt>
              <dd class="mt-1 whitespace-pre-line text-sm text-gray-800">{{ selected.notes || t("cj.noNotes") }}</dd>
            </div>
          </dl>

          <div class="mt-5 grid grid-cols-2 gap-2 px-5">
            <template v-if="isOpenJob(selected)">
              <button
                type="button"
                :disabled="busyId === selected.id"
                class="field-tap col-span-2 rounded-2xl bg-amber-500 py-4 text-base font-bold text-white active:scale-95 disabled:opacity-60"
                @click="accept(selected)"
              >
                {{ busyId === selected.id ? t("jobs.working") : t("jobs.accept") }}
              </button>
            </template>
            <template v-else-if="['confirmed', 'pending', 'rescheduled'].includes(selected.status)">
              <button
                type="button"
                :disabled="busyId === selected.id"
                class="field-tap col-span-2 rounded-2xl bg-emerald-600 py-4 text-base font-bold text-white active:scale-95 disabled:opacity-60"
                @click="setStatus(selected, 'in_progress')"
              >
                {{ busyId === selected.id ? t("jobs.working") : t("cj.start") }}
              </button>
            </template>
            <template v-else-if="selected.status === 'in_progress'">
              <button
                type="button"
                :disabled="busyId === selected.id"
                class="field-tap col-span-2 rounded-2xl bg-navy-600 py-4 text-base font-bold text-white active:scale-95 disabled:opacity-60"
                @click="setStatus(selected, 'completed')"
              >
                {{ busyId === selected.id ? t("jobs.working") : t("jobs.complete") }}
              </button>
            </template>
            <a
              :href="mapsUrl(selected)"
              target="_blank"
              rel="noopener"
              class="field-tap flex items-center justify-center rounded-2xl bg-gray-900 py-3.5 text-sm font-bold text-white active:scale-95"
            >{{ t("jobs.navigate") }}</a>
            <p
              v-if="!isOpenJob(selected) && checklists[selected.id]?.reviewStatus === 'rework'"
              class="col-span-2 whitespace-pre-line rounded-2xl bg-red-50 px-4 py-3 text-sm text-red-800 ring-1 ring-red-200"
            ><span class="flex items-center gap-1.5 font-bold"><AppIcon name="undo" :size="16" />{{ t("cl.rework") }}</span>{{ checklists[selected.id]!.reviewNote }}</p>
            <a
              v-if="!isOpenJob(selected)"
              :href="`/checklists?booking=${selected.id}`"
              :class="[
                'field-tap flex items-center justify-center gap-2 rounded-2xl py-3.5 text-sm font-bold active:scale-95',
                checklists[selected.id]?.reviewStatus === 'rework' ? 'bg-red-600 text-white' : 'bg-gray-100 text-gray-800',
              ]"
            ><AppIcon name="clipboard" :size="18" /> {{ t("jobs.checklist") }}</a>
            <a
              href="/chat"
              :class="[
                'field-tap flex items-center justify-center gap-2 rounded-2xl bg-emerald-50 py-3.5 text-sm font-bold text-emerald-700 ring-1 ring-emerald-200 active:scale-95',
                isOpenJob(selected) ? '' : 'col-span-2',
              ]"
            ><AppIcon name="chat" :size="18" /> {{ t("cj.messageTeam") }}</a>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>
