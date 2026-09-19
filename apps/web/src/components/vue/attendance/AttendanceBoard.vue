<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { ApiError } from "../../../lib/api";
import {
  checkIn,
  checkOut,
  getAttendance,
  type AttendanceRecord,
  type PersonType,
} from "../../../lib/attendance";
import { getSessionUser } from "../../../lib/auth";
import { getCleaners } from "../../../lib/cleaners";
import { hasPermission } from "../../../lib/roles";
import { showToast } from "../../../lib/toast";
import { getUsers } from "../../../lib/users";

function todayStr(): string {
  return new Date().toISOString().slice(0, 10);
}

function shiftDate(dateStr: string, days: number): string {
  const d = new Date(`${dateStr}T12:00:00`);
  d.setDate(d.getDate() + days);
  return d.toISOString().slice(0, 10);
}

function formatTime(iso: string | null): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function formatDate(dateStr: string): string {
  const d = new Date(`${dateStr}T12:00:00`);
  if (Number.isNaN(d.getTime())) return dateStr;
  return d.toLocaleDateString([], {
    weekday: "short",
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

function initials(name: string): string {
  return name
    .split(" ")
    .map((w) => w[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("")
    .toUpperCase();
}

type DayStatus = "checked-out" | "checked-in" | "not-in";

// One row per person on the board, whichever workforce is being shown.
interface Person {
  id: number;
  name: string;
  subtitle: string;
}

const role = getSessionUser()?.role;
const canManage = computed(() => hasPermission(role, "attendance.manage"));
// The staff roster comes from /users, so the tab needs that permission too.
const canSeeStaff = computed(() => hasPermission(role, "users.read"));

const activeTab = ref<PersonType>("cleaner");
const selectedDate = ref(todayStr());
const cleaners = ref<Person[]>([]);
const staff = ref<Person[]>([]);
const dayRecords = ref<AttendanceRecord[]>([]);
const loadingDay = ref(true);

const histPerson = ref<string>("");
const histFrom = ref<string>("");
const histTo = ref<string>("");
const history = ref<AttendanceRecord[]>([]);
const loadingHistory = ref(true);

const actingId = ref<number | null>(null);

const people = computed(() => (activeTab.value === "staff" ? staff.value : cleaners.value));

const recordByPerson = computed(() => {
  const map = new Map<number, AttendanceRecord>();
  for (const r of dayRecords.value) {
    if (r.personType !== activeTab.value) continue;
    if (!map.has(r.personId)) map.set(r.personId, r);
  }
  return map;
});

function statusFor(personId: number): DayStatus {
  const r = recordByPerson.value.get(personId);
  if (!r?.checkInAt) return "not-in";
  if (r.checkOutAt) return "checked-out";
  return "checked-in";
}

function statusLabel(s: DayStatus): string {
  if (s === "checked-in") return "Checked-in";
  if (s === "checked-out") return "Checked-out";
  return "Not in";
}

function statusPill(s: DayStatus): string {
  if (s === "checked-in") return "bg-emerald-100 text-emerald-700";
  if (s === "checked-out") return "bg-blue-100 text-navy-700";
  return "bg-amber-100 text-amber-700";
}

const checkedInCount = computed(
  () => people.value.filter((p) => statusFor(p.id) === "checked-in").length,
);
const checkedOutCount = computed(
  () => people.value.filter((p) => statusFor(p.id) === "checked-out").length,
);
const notInCount = computed(
  () => people.value.filter((p) => statusFor(p.id) === "not-in").length,
);

const isToday = computed(() => selectedDate.value === todayStr());

const emptyLabel = computed(() =>
  activeTab.value === "staff" ? "No team staff found." : "No cleaners found.",
);

async function loadPeople(): Promise<void> {
  const jobs: Promise<void>[] = [
    getCleaners({ limit: 200 }).then((rows) => {
      cleaners.value = rows.map((c) => ({
        id: c.id,
        name: `${c.firstName} ${c.lastName}`.trim(),
        subtitle: c.phone,
      }));
    }),
  ];
  if (canSeeStaff.value) {
    jobs.push(
      getUsers({ limit: 200 }).then((rows) => {
        // Field cleaners are tracked on the Cleaners tab from their cleaner
        // profile, so a CLEANER login would otherwise appear twice.
        staff.value = rows
          .filter((u) => u.role !== "CLEANER")
          .map((u) => ({ id: u.id, name: u.name, subtitle: u.role.replace("_", " ") }));
      }),
    );
  }
  await Promise.all(jobs);
}

async function loadDay(): Promise<void> {
  loadingDay.value = true;
  try {
    const [, records] = await Promise.all([
      loadPeople(),
      getAttendance({
        from: selectedDate.value,
        to: selectedDate.value,
        limit: 400,
      }),
    ]);
    dayRecords.value = records;
  } catch {
    showToast("Failed to load attendance", "error");
  } finally {
    loadingDay.value = false;
  }
}

async function loadHistory(): Promise<void> {
  loadingHistory.value = true;
  try {
    history.value = await getAttendance({
      type: activeTab.value,
      person: histPerson.value ? Number(histPerson.value) : undefined,
      from: histFrom.value || undefined,
      to: histTo.value || undefined,
      limit: 50,
    });
  } catch {
    showToast("Failed to load attendance history", "error");
  } finally {
    loadingHistory.value = false;
  }
}

function goPrev(): void {
  selectedDate.value = shiftDate(selectedDate.value, -1);
}

function goNext(): void {
  selectedDate.value = shiftDate(selectedDate.value, 1);
}

function goToday(): void {
  selectedDate.value = todayStr();
}

function switchTab(tab: PersonType): void {
  if (activeTab.value === tab) return;
  activeTab.value = tab;
  histPerson.value = "";
  void loadHistory();
}

function errorMessage(err: unknown, fallback: string): string {
  if (err instanceof ApiError) return err.message || fallback;
  if (err instanceof Error && err.message) return err.message;
  return fallback;
}

async function doCheckIn(personId: number): Promise<void> {
  actingId.value = personId;
  try {
    await checkIn(activeTab.value, personId);
    showToast("Checked in", "success");
    await Promise.all([loadDay(), loadHistory()]);
  } catch (err) {
    showToast(errorMessage(err, "Check-in failed"), "error");
  } finally {
    actingId.value = null;
  }
}

async function doCheckOut(personId: number): Promise<void> {
  actingId.value = personId;
  try {
    await checkOut(activeTab.value, personId);
    showToast("Checked out", "success");
    await Promise.all([loadDay(), loadHistory()]);
  } catch (err) {
    showToast(errorMessage(err, "Check-out failed"), "error");
  } finally {
    actingId.value = null;
  }
}

watch(selectedDate, () => {
  void loadDay();
});

onMounted(async () => {
  await Promise.all([loadDay(), loadHistory()]);
});
</script>

<template>
  <div class="space-y-4">
    <!-- Day navigator -->
    <section class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="font-semibold tracking-tight text-gray-900">Daily check-in</h2>
          <p class="text-xs text-gray-400">{{ formatDate(selectedDate) }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <button
            type="button"
            class="rounded-xl px-3 py-2 text-sm font-semibold text-gray-700 ring-1 ring-gray-200 transition hover:bg-gray-50"
            aria-label="Previous day"
            @click="goPrev"
          >
            ‹
          </button>
          <button
            type="button"
            class="rounded-xl px-3 py-2 text-sm font-semibold text-navy-600 ring-1 ring-gray-200 transition hover:bg-gray-50"
            @click="goToday"
          >
            Today
          </button>
          <button
            type="button"
            class="rounded-xl px-3 py-2 text-sm font-semibold text-gray-700 ring-1 ring-gray-200 transition hover:bg-gray-50"
            aria-label="Next day"
            @click="goNext"
          >
            ›
          </button>
          <input
            v-model="selectedDate"
            type="date"
            class="rounded-xl px-3 py-2 text-sm text-gray-700 ring-1 ring-gray-200"
          />
        </div>
      </div>

      <!-- Workforce tabs -->
      <div
        v-if="canSeeStaff"
        class="mt-4 inline-flex rounded-2xl bg-gray-100 p-1"
        role="tablist"
        aria-label="Workforce"
      >
        <button
          type="button"
          role="tab"
          :aria-selected="activeTab === 'cleaner'"
          :class="[
            'rounded-xl px-4 py-2 text-sm font-semibold transition',
            activeTab === 'cleaner'
              ? 'bg-white text-navy-700 shadow-sm'
              : 'text-gray-500 hover:text-gray-700',
          ]"
          @click="switchTab('cleaner')"
        >
          Cleaners
        </button>
        <button
          type="button"
          role="tab"
          :aria-selected="activeTab === 'staff'"
          :class="[
            'rounded-xl px-4 py-2 text-sm font-semibold transition',
            activeTab === 'staff'
              ? 'bg-white text-navy-700 shadow-sm'
              : 'text-gray-500 hover:text-gray-700',
          ]"
          @click="switchTab('staff')"
        >
          Team staff
        </button>
      </div>

      <div class="mt-4 flex flex-wrap gap-2 text-xs font-medium">
        <span class="rounded-full bg-emerald-100 px-3 py-1 text-emerald-700">
          {{ checkedInCount }} checked-in
        </span>
        <span class="rounded-full bg-blue-100 px-3 py-1 text-navy-700">
          {{ checkedOutCount }} checked-out
        </span>
        <span class="rounded-full bg-amber-100 px-3 py-1 text-amber-700">
          {{ notInCount }} not in
        </span>
        <span v-if="!isToday" class="rounded-full bg-gray-100 px-3 py-1 text-gray-500">
          Viewing past / future day
        </span>
      </div>
      <p v-if="!canManage" class="mt-3 text-xs text-gray-400">
        You need the attendance.manage permission to check staff in or out.
      </p>
    </section>

    <!-- Per-person day cards -->
    <div v-if="loadingDay" class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <div
        v-for="i in 6"
        :key="i"
        class="rounded-3xl bg-white p-5 shadow-sm ring-1 ring-gray-100"
      >
        <div class="h-10 w-10 animate-pulse rounded-xl bg-gray-200"></div>
        <div class="mt-4 h-5 w-32 animate-pulse rounded-lg bg-gray-200"></div>
        <div class="mt-2 h-4 w-24 animate-pulse rounded-lg bg-gray-100"></div>
      </div>
    </div>

    <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <article
        v-for="person in people"
        :key="person.id"
        class="rounded-3xl bg-white p-5 shadow-sm ring-1 ring-gray-100 transition hover:-translate-y-0.5 hover:shadow-md"
      >
        <div class="flex items-center gap-3">
          <span
            class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-gradient-to-br from-navy-500 to-blue-600 text-xs font-semibold text-white shadow-sm"
          >
            {{ initials(person.name) }}
          </span>
          <div class="min-w-0 flex-1">
            <p class="truncate text-sm font-medium text-gray-900">{{ person.name }}</p>
            <p class="truncate text-xs text-gray-400">{{ person.subtitle }}</p>
          </div>
          <span
            :class="[
              'shrink-0 rounded-full px-2.5 py-1 text-[11px] font-semibold',
              statusPill(statusFor(person.id)),
            ]"
          >
            {{ statusLabel(statusFor(person.id)) }}
          </span>
        </div>

        <div class="mt-4 flex items-center gap-4 text-sm tabular-nums">
          <div>
            <p class="text-[11px] font-medium uppercase tracking-wider text-gray-400">In</p>
            <p class="font-semibold text-gray-900">
              {{ formatTime(recordByPerson.get(person.id)?.checkInAt ?? null) }}
            </p>
          </div>
          <div>
            <p class="text-[11px] font-medium uppercase tracking-wider text-gray-400">Out</p>
            <p class="font-semibold text-gray-900">
              {{ formatTime(recordByPerson.get(person.id)?.checkOutAt ?? null) }}
            </p>
          </div>
        </div>

        <div class="mt-4 flex gap-2">
          <button
            type="button"
            :disabled="
              !canManage || actingId === person.id || statusFor(person.id) !== 'not-in'
            "
            class="flex-1 rounded-xl bg-gradient-to-r from-emerald-400 to-green-600 px-3 py-2 text-xs font-semibold text-white shadow-sm transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40"
            @click="doCheckIn(person.id)"
          >
            {{ actingId === person.id ? "Working…" : "Check in" }}
          </button>
          <button
            type="button"
            :disabled="
              !canManage ||
              actingId === person.id ||
              statusFor(person.id) !== 'checked-in'
            "
            class="flex-1 rounded-xl bg-gradient-to-r from-navy-500 to-blue-600 px-3 py-2 text-xs font-semibold text-white shadow-sm transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40"
            @click="doCheckOut(person.id)"
          >
            {{ actingId === person.id ? "Working…" : "Check out" }}
          </button>
        </div>
      </article>
    </div>

    <p v-if="!loadingDay && people.length === 0" class="text-sm text-gray-400">
      {{ emptyLabel }}
    </p>

    <!-- History -->
    <section class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 class="font-semibold tracking-tight text-gray-900">History</h2>
          <p class="text-xs text-gray-400">
            Recent check-in / check-out records ·
            {{ activeTab === "staff" ? "Team staff" : "Cleaners" }}
          </p>
        </div>
        <button
          type="button"
          class="rounded-xl px-3 py-2 text-xs font-semibold text-navy-600 ring-1 ring-gray-200 transition hover:bg-gray-50"
          @click="loadHistory"
        >
          Refresh
        </button>
      </div>

      <div class="mt-4 grid grid-cols-1 gap-2 sm:grid-cols-4">
        <select
          v-model="histPerson"
          class="rounded-xl px-3 py-2 text-sm text-gray-700 ring-1 ring-gray-200"
          :aria-label="activeTab === 'staff' ? 'Filter by staff member' : 'Filter by cleaner'"
          @change="loadHistory"
        >
          <option value="">
            {{ activeTab === "staff" ? "All team staff" : "All cleaners" }}
          </option>
          <option v-for="p in people" :key="p.id" :value="String(p.id)">
            {{ p.name }}
          </option>
        </select>
        <input
          v-model="histFrom"
          type="date"
          aria-label="From date"
          class="rounded-xl px-3 py-2 text-sm text-gray-700 ring-1 ring-gray-200"
          @change="loadHistory"
        />
        <input
          v-model="histTo"
          type="date"
          aria-label="To date"
          class="rounded-xl px-3 py-2 text-sm text-gray-700 ring-1 ring-gray-200"
          @change="loadHistory"
        />
        <button
          type="button"
          class="rounded-xl bg-gray-950 px-3 py-2 text-xs font-semibold text-white transition hover:bg-gray-800"
          @click="histPerson = ''; histFrom = ''; histTo = ''; loadHistory();"
        >
          Clear filters
        </button>
      </div>

      <div v-if="loadingHistory" class="mt-4 space-y-2">
        <div v-for="i in 4" :key="i" class="h-14 animate-pulse rounded-2xl bg-gray-50"></div>
      </div>

      <ul v-else class="mt-4 space-y-2">
        <li
          v-for="r in history"
          :key="r.id"
          class="flex flex-wrap items-center gap-3 rounded-2xl bg-gray-50/70 p-3 ring-1 ring-gray-100"
        >
          <span
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-navy-500 to-blue-600 text-xs font-semibold text-white"
          >
            {{ initials(r.personName) }}
          </span>
          <div class="min-w-0 flex-1">
            <p class="truncate text-sm font-medium text-gray-900">{{ r.personName }}</p>
            <p class="text-xs tabular-nums text-gray-500">
              {{ r.workDate }} · In {{ formatTime(r.checkInAt) }} · Out
              {{ formatTime(r.checkOutAt) }}
            </p>
          </div>
          <span
            :class="[
              'shrink-0 rounded-full px-2.5 py-1 text-[11px] font-semibold',
              r.checkOutAt
                ? 'bg-blue-100 text-navy-700'
                : r.checkInAt
                  ? 'bg-emerald-100 text-emerald-700'
                  : 'bg-amber-100 text-amber-700',
            ]"
          >
            {{ r.checkOutAt ? "Checked-out" : r.checkInAt ? "Checked-in" : "Not in" }}
          </span>
        </li>
      </ul>

      <p v-if="!loadingHistory && history.length === 0" class="mt-4 text-sm text-gray-400">
        No attendance records for these filters.
      </p>

      <p class="mt-4 text-xs text-gray-400">
        Live data from the Go API · GET /api/v1/attendance
      </p>
    </section>
  </div>
</template>
