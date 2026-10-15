<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  CHECKLIST_STATE_BADGE,
  answerChecklistItem,
  checklistState,
  confirmChecklist,
  createChecklist,
  getChecklistByBooking,
  getChecklistTemplates,
  reviewChecklist,
  submitChecklist,
  submitProblems,
  uploadItemPhoto,
  type Checklist,
  type ChecklistItem,
  type ChecklistTemplate,
  type ItemAnswer,
  type ItemResult,
} from "../../../lib/checklists";
import { serviceLabel } from "../../../lib/settings";
import { hasPermission } from "../../../lib/roles";
import { getBookings, type Booking } from "../../../lib/bookings";
import { isOfflineQueued } from "../../../lib/offline";
import { ApiError } from "../../../lib/api";
import { showToast } from "../../../lib/toast";
import { getSessionUser } from "../../../lib/auth";
import { t, type MessageKey } from "../../../lib/i18n";
import AppIcon, { type IconName } from "../ui/AppIcon.vue";
import LanguageSwitcher from "../ui/LanguageSwitcher.vue";
import ChecklistPhoto from "./ChecklistPhoto.vue";
import ChecklistReviewList from "./ChecklistReviewList.vue";
import ChecklistTemplates from "./ChecklistTemplates.vue";

// Cleaners see this page in their chosen language; office staff in English.
const isCleaner = getSessionUser()?.role === "CLEANER";
function L(key: MessageKey, english: string): string {
  return isCleaner ? t(key) : english;
}
const canManage =
  !isCleaner && hasPermission(getSessionUser()?.role ?? "viewer", "checklists.manage");

// Office tabs: the review list, one booking's checklist, and templates.
type View = "review" | "checklist" | "templates";
// A ?booking= link opens straight on that checklist.
const deepBooking =
  typeof window === "undefined"
    ? null
    : Number(new URLSearchParams(window.location.search).get("booking")) || null;
const view = ref<View>(deepBooking ? "checklist" : "review");

const bookingId = ref<number | "">(deepBooking ?? "");
const bookings = ref<Booking[]>([]);
const bookingsLoading = ref(true);
const checklist = ref<Checklist | null>(null);
const loading = ref(false);
const missing = ref(false);
// The booking's service has no checklist template (and there is no general one).
const noTemplate = ref(false);
const signature = ref("");
const confirming = ref(false);
const submitting = ref(false);
const reviewNote = ref("");
const reviewing = ref(false);
const noteOpen = ref<Record<number, string>>({});

// Templates: "" = pick automatically from the booking's service type.
const templates = ref<ChecklistTemplate[]>([]);
const templateId = ref<number | "">("");

const selectedBooking = computed(() =>
  bookings.value.find((b) => b.id === bookingId.value),
);
const autoTemplate = computed(() => {
  const st = selectedBooking.value?.serviceType;
  return (
    templates.value.find((tp) => tp.isActive && st && tp.serviceType === st) ??
    templates.value.find((tp) => tp.isActive && !tp.serviceType)
  );
});
const state = computed(() => (checklist.value ? checklistState(checklist.value) : null));
const problems = computed(() => (checklist.value ? submitProblems(checklist.value) : []));
const locked = computed(() => !!checklist.value?.locked);
const doneCount = computed(() => checklist.value?.items.filter((i) => i.isCompleted).length ?? 0);
const pct = computed(() =>
  checklist.value?.items.length ? Math.round((doneCount.value / checklist.value.items.length) * 100) : 0,
);
// Progress ring geometry (r = 26).
const RING = 2 * Math.PI * 26;

const tabs: { id: View; label: string; icon: IconName }[] = [
  { id: "review", label: "Review", icon: "inbox" },
  { id: "checklist", label: "Checklist", icon: "clipboard" },
  ...(canManage ? [{ id: "templates" as View, label: "Templates", icon: "stack" as IconName }] : []),
];

const stateBanner: Record<string, { icon: IconName; box: string; chip: string }> = {
  rework: { icon: "undo", box: "bg-red-50 ring-red-100 text-red-900", chip: "bg-red-100 text-red-600" },
  submitted: { icon: "clock", box: "bg-amber-50 ring-amber-100 text-amber-900", chip: "bg-amber-100 text-amber-600" },
  approved: { icon: "badge", box: "bg-emerald-50 ring-emerald-100 text-emerald-900", chip: "bg-emerald-100 text-emerald-600" },
};

const resultOptions = computed<{ value: ItemResult; label: string; icon: IconName; on: string }[]>(() => [
  { value: "pass", label: L("cl.pass", "Pass"), icon: "check", on: "bg-emerald-600 text-white ring-emerald-600 shadow-sm shadow-emerald-600/30" },
  { value: "fail", label: L("cl.fail", "Fail"), icon: "x", on: "bg-red-600 text-white ring-red-600 shadow-sm shadow-red-600/30" },
  { value: "na", label: L("cl.na", "N/A"), icon: "minus", on: "bg-gray-700 text-white ring-gray-700" },
]);

function initials(name?: string | null): string {
  return (name ?? "")
    .split(" ")
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w[0]!.toUpperCase())
    .join("");
}

function fmt(iso?: string | null): string {
  if (!iso) return "";
  return new Date(iso).toLocaleString(isCleaner ? undefined : "en-GB", {
    day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit",
  });
}

function problemText(it: ChecklistItem): string {
  if (!it.isCompleted) return it.label;
  if (it.requirePhoto && !it.afterPhotoUrl) return `${it.label} — ${L("cl.photoRequired", "After photo required")}`;
  return `${it.label} — ${L("cl.failNeedsNote", "Failed — add a note saying why")}`;
}

async function loadTemplates() {
  try {
    templates.value = await getChecklistTemplates();
  } catch {
    templates.value = [];
  }
}

function setUrl(id: number | null) {
  const url = new URL(window.location.href);
  if (id) url.searchParams.set("booking", String(id));
  else url.searchParams.delete("booking");
  window.history.replaceState(null, "", url);
}

async function handleLoad() {
  if (bookingId.value === "") return;
  loading.value = true;
  missing.value = false;
  noTemplate.value = false;
  setUrl(Number(bookingId.value));
  try {
    checklist.value = await getChecklistByBooking(Number(bookingId.value));
    reviewNote.value = "";
  } catch (err) {
    // 404: no checklist and none could be created. Opening a booking creates
    // its checklist from the template, so this means there is no template
    // for the service (or this role may only read checklists).
    if (err instanceof ApiError && err.status === 404) {
      missing.value = true;
      noTemplate.value = /template/i.test(err.message);
    } else {
      showToast(err instanceof Error ? err.message : "Failed to load checklist", "error");
    }
    checklist.value = null;
  } finally {
    loading.value = false;
  }
}

function openBooking(id: number) {
  bookingId.value = id;
  view.value = "checklist";
  void handleLoad();
}

function backToList() {
  view.value = "review";
  checklist.value = null;
  bookingId.value = "";
  setUrl(null);
}

async function handleCreate() {
  if (bookingId.value === "") {
    showToast("Select a booking", "error");
    return;
  }
  try {
    checklist.value = await createChecklist(
      Number(bookingId.value),
      templateId.value === "" ? undefined : templateId.value,
    );
    missing.value = false;
    signature.value = "";
    showToast("Checklist created from template", "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to create checklist", "error");
  }
}

async function handleConfirm() {
  if (!checklist.value || !signature.value.trim()) {
    showToast(L("cl.needSignature", "Enter client name/signature to confirm"), "error");
    return;
  }
  confirming.value = true;
  try {
    checklist.value = await confirmChecklist(checklist.value.bookingId, signature.value.trim());
    showToast(L("cl.confirmed", "Checklist confirmed by client"), "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to confirm checklist", "error");
  } finally {
    confirming.value = false;
  }
}

// Optimistic answer: the offline queue replays the same set-semantics PATCH.
async function answer(it: ChecklistItem, a: ItemAnswer, optimistic: Partial<ChecklistItem>) {
  const prev = checklist.value;
  if (prev) {
    checklist.value = {
      ...prev,
      items: prev.items.map((x) => (x.id === it.id ? { ...x, ...optimistic } : x)),
    };
  }
  try {
    checklist.value = await answerChecklistItem(it.id, a);
    return true;
  } catch (err) {
    if (isOfflineQueued(err)) {
      showToast("Saved offline — will sync when online", "success");
      return true;
    }
    checklist.value = prev;
    showToast(err instanceof Error ? err.message : "Failed to update item", "error");
    return false;
  }
}

function toggle(it: ChecklistItem) {
  void answer(it, { isCompleted: !it.isCompleted }, { isCompleted: !it.isCompleted });
}

function setScore(it: ChecklistItem, n: number) {
  const score = it.score === n ? 0 : n;
  void answer(it, { isCompleted: score > 0, score }, { score: score || null, isCompleted: score > 0 });
}

function setResult(it: ChecklistItem, r: ItemResult) {
  const result = it.result === r ? "" : r;
  void answer(it, { isCompleted: !!result, result }, { result: result || null, isCompleted: !!result });
}

function cancelNote(id: number) {
  const next = { ...noteOpen.value };
  delete next[id];
  noteOpen.value = next;
}

async function saveNote(it: ChecklistItem) {
  const notes = (noteOpen.value[it.id] ?? "").trim();
  if (await answer(it, { isCompleted: it.isCompleted, notes }, { notes })) {
    cancelNote(it.id);
    showToast(L("cl.noteSaved", "Note saved"), "success");
  }
}

async function attach(itemId: number, kind: "before" | "after", ev: Event) {
  const input = ev.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (!file) return;
  try {
    checklist.value = await uploadItemPhoto(itemId, kind, file);
    showToast(L("cl.photoAttached", "Photo attached"), "success");
  } catch (err) {
    if (isOfflineQueued(err)) {
      // Offline preview from the on-device file; the queued upload replaces
      // it with the server URL on sync.
      const url = URL.createObjectURL(file);
      if (checklist.value) {
        checklist.value = {
          ...checklist.value,
          items: checklist.value.items.map((it) =>
            it.id === itemId
              ? { ...it, ...(kind === "before" ? { beforePhotoUrl: url } : { afterPhotoUrl: url }) }
              : it,
          ),
        };
      }
      showToast("Photo saved offline — will upload when online", "success");
    } else {
      showToast(err instanceof Error ? err.message : "Failed to upload photo", "error");
    }
  }
}

async function handleSubmit() {
  if (!checklist.value) return;
  submitting.value = true;
  try {
    checklist.value = await submitChecklist(checklist.value.bookingId);
    showToast(L("cl.submittedToast", "Checklist submitted"), "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to submit checklist", "error");
  } finally {
    submitting.value = false;
  }
}

async function handleReview(decision: "approved" | "rework") {
  if (!checklist.value) return;
  if (decision === "rework" && !reviewNote.value.trim()) {
    showToast("Say what needs redoing", "error");
    return;
  }
  reviewing.value = true;
  try {
    checklist.value = await reviewChecklist(checklist.value.bookingId, decision, reviewNote.value.trim());
    reviewNote.value = "";
    showToast(decision === "approved" ? "Checklist approved" : "Sent back to the cleaner", "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to save review", "error");
  } finally {
    reviewing.value = false;
  }
}

onMounted(async () => {
  if (!isCleaner) void loadTemplates();
  if (deepBooking) void handleLoad();
  try {
    bookings.value = await getBookings({ limit: 100 });
  } catch {
    /* empty select: API error surfaces on load attempt */
  } finally {
    bookingsLoading.value = false;
  }
});
</script>

<template>
  <div class="space-y-4">
    <!-- Hero -->
    <section class="relative overflow-hidden rounded-3xl bg-gradient-to-br from-navy-800 via-navy-600 to-blue-500 p-5 text-white shadow-lg sm:p-7">
      <div class="pointer-events-none absolute -right-16 -top-16 h-56 w-56 rounded-full bg-white/10 blur-2xl"></div>
      <div class="pointer-events-none absolute -bottom-20 left-1/3 h-48 w-48 rounded-full bg-blue-300/20 blur-2xl"></div>
      <div class="relative">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="flex items-center gap-3">
            <span class="flex h-11 w-11 items-center justify-center rounded-2xl bg-white/15 ring-1 ring-white/25 backdrop-blur">
              <AppIcon name="clipboard" :size="22" />
            </span>
            <div>
              <p class="text-[11px] font-medium uppercase tracking-widest text-blue-100">{{ L("cl.title", "Checklists & proof of work") }}</p>
              <h1 class="text-xl font-semibold tracking-tight sm:text-2xl">
                <template v-if="checklist && view === 'checklist'">{{ selectedBooking?.bookingNumber ?? `Booking #${checklist.bookingId}` }}</template>
                <template v-else-if="view === 'templates'">Templates</template>
                <template v-else-if="view === 'review'">Review</template>
                <template v-else>{{ L("cl.selectBooking", "Select booking") }}</template>
              </h1>
            </div>
          </div>
          <LanguageSwitcher v-if="isCleaner" />
        </div>

        <!-- Office tabs -->
        <div v-if="!isCleaner" class="mt-5 inline-flex rounded-2xl bg-white/10 p-1 ring-1 ring-white/20 backdrop-blur">
          <button
            v-for="tab in tabs"
            :key="tab.id"
            type="button"
            :class="[
              'flex items-center gap-2 rounded-xl px-3.5 py-2 text-sm font-semibold transition',
              view === tab.id ? 'bg-white text-navy-700 shadow' : 'text-white/85 hover:bg-white/10',
            ]"
            @click="tab.id === 'review' ? backToList() : (view = tab.id)"
          >
            <AppIcon :name="tab.icon" :size="17" />
            {{ tab.label }}
          </button>
        </div>

        <!-- Checklist summary in the hero -->
        <div v-if="checklist && view === 'checklist' && !loading" class="mt-5 flex flex-wrap items-center gap-5">
          <div class="relative h-16 w-16">
            <svg viewBox="0 0 64 64" class="h-16 w-16 -rotate-90">
              <circle cx="32" cy="32" r="26" fill="none" stroke="currentColor" stroke-width="6" class="text-white/20" />
              <circle
                cx="32" cy="32" r="26" fill="none" stroke="currentColor" stroke-width="6" stroke-linecap="round"
                class="text-white transition-all duration-500"
                :stroke-dasharray="RING"
                :stroke-dashoffset="RING * (1 - pct / 100)"
              />
            </svg>
            <span class="absolute inset-0 flex items-center justify-center text-sm font-bold">{{ pct }}%</span>
          </div>
          <div class="min-w-0 flex-1">
            <p class="truncate text-sm text-blue-50/90">
              <template v-if="selectedBooking">{{ selectedBooking.customerName }} · {{ serviceLabel(selectedBooking.serviceType) }}</template>
            </p>
            <div class="mt-1.5 flex flex-wrap items-center gap-2 text-sm">
              <span class="inline-flex items-center gap-1.5 rounded-full bg-white/15 px-3 py-1 font-semibold ring-1 ring-white/20">
                <AppIcon name="check" :size="15" :stroke="2.5" />
                {{ doneCount }}/{{ checklist.items.length }} {{ L("cl.done", "done") }}
              </span>
              <span v-if="checklist.avgScore != null" class="inline-flex items-center gap-1.5 rounded-full bg-white/15 px-3 py-1 font-semibold ring-1 ring-white/20">
                <AppIcon name="star" :size="15" class="text-amber-300" />
                {{ checklist.avgScore.toFixed(1) }} / 5
              </span>
              <span v-if="!isCleaner && state" class="rounded-full bg-white px-3 py-1 text-xs font-bold text-navy-700">{{ CHECKLIST_STATE_BADGE[state].label }}</span>
            </div>
          </div>
          <a
            :href="`/checklists/print?booking=${checklist.bookingId}&lang=${isCleaner ? 'my' : 'en'}`"
            target="_blank"
            rel="noopener"
            class="inline-flex items-center gap-2 rounded-xl bg-white px-4 py-2.5 text-sm font-semibold text-navy-700 shadow transition hover:brightness-95"
          >
            <AppIcon name="document" :size="18" />
            PDF
          </a>
        </div>
      </div>
    </section>

    <ChecklistReviewList v-if="!isCleaner && view === 'review'" @open="openBooking" />
    <ChecklistTemplates v-else-if="!isCleaner && view === 'templates'" />

    <template v-else>
      <!-- Booking picker -->
      <section class="rounded-3xl bg-white p-4 shadow-sm ring-1 ring-gray-100 sm:p-5">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
          <select v-model="bookingId" :disabled="bookingsLoading" class="w-full text-sm font-medium sm:max-w-sm" @change="handleLoad">
              <option value="" disabled>{{ bookingsLoading ? L("cl.loadingBookings", "Loading bookings…") : L("cl.selectBooking", "Select booking") }}</option>
              <option v-if="bookingId !== '' && !selectedBooking" :value="bookingId">#{{ bookingId }}</option>
              <option v-for="b in bookings" :key="b.id" :value="b.id">{{ b.bookingNumber }} — {{ b.customerName }}</option>
            </select>
          <template v-if="canManage && missing && !noTemplate">
            <select v-model="templateId" class="w-full text-sm sm:w-72" aria-label="Template">
              <option value="">Auto{{ autoTemplate ? ` — ${autoTemplate.name}` : "" }}</option>
              <option v-for="tp in templates.filter((x) => x.isActive)" :key="tp.id" :value="tp.id">{{ tp.name }} ({{ tp.items.length }})</option>
            </select>
            <button type="button" class="inline-flex items-center justify-center gap-2 rounded-2xl bg-navy-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-navy-700" @click="handleCreate">
              <AppIcon name="plus" :size="17" /> New from template
            </button>
          </template>
        </div>
      </section>

      <!-- Loading skeleton -->
      <div v-if="loading" class="space-y-3">
        <div v-for="i in 3" :key="i" class="h-28 animate-pulse rounded-3xl bg-white ring-1 ring-gray-100"></div>
      </div>

      <template v-else-if="checklist">
        <!-- Where it stands -->
        <section v-if="state && stateBanner[state]" :class="['flex gap-3 rounded-3xl p-4 ring-1 sm:p-5', stateBanner[state]!.box]">
          <span :class="['flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl', stateBanner[state]!.chip]">
            <AppIcon :name="stateBanner[state]!.icon" :size="20" />
          </span>
          <div class="min-w-0 text-sm">
            <p class="font-semibold">
              <template v-if="state === 'rework'">{{ L("cl.rework", "Sent back to the cleaner for rework") }}</template>
              <template v-else-if="state === 'submitted'">{{ L("cl.waitingReview", "Submitted — waiting for review") }}</template>
              <template v-else>{{ L("cl.approved", "Approved") }}</template>
            </p>
            <p v-if="checklist.reviewNote" class="mt-1 whitespace-pre-line">{{ checklist.reviewNote }}</p>
            <p class="mt-1 text-xs opacity-75">
              <template v-if="state !== 'rework' && checklist.submittedBy">{{ checklist.submittedBy }} · {{ fmt(checklist.submittedAt) }}</template>
              <template v-if="!isCleaner && checklist.reviewedBy && state !== 'submitted'"><template v-if="state !== 'rework' && checklist.submittedBy"> — </template>Reviewed by {{ checklist.reviewedBy }} · {{ fmt(checklist.reviewedAt) }}</template>
            </p>
          </div>
        </section>

        <!-- Items -->
        <ol class="space-y-3">
          <li
            v-for="(it, idx) in checklist.items"
            :key="it.id"
            :class="[
              'rounded-3xl bg-white p-4 shadow-sm ring-1 transition sm:p-5',
              it.result === 'fail' ? 'ring-red-200' : it.isCompleted ? 'ring-emerald-100' : 'ring-gray-100',
            ]"
          >
            <div class="flex items-start gap-3">
              <!-- tick / status -->
              <button
                v-if="it.itemType === 'check'"
                type="button"
                :disabled="locked"
                :aria-pressed="it.isCompleted"
                :aria-label="it.label"
                :class="[
                  'field-tap-sm flex h-9 w-9 shrink-0 items-center justify-center rounded-xl ring-1 transition disabled:cursor-not-allowed',
                  it.isCompleted ? 'bg-emerald-500 text-white ring-emerald-500 shadow-sm shadow-emerald-500/30' : 'bg-white text-gray-300 ring-gray-300 hover:ring-navy-400',
                ]"
                @click="toggle(it)"
              >
                <AppIcon name="check" :size="18" :stroke="2.75" />
              </button>
              <span
                v-else
                :class="[
                  'flex h-9 w-9 shrink-0 items-center justify-center rounded-xl text-sm font-bold',
                  it.result === 'fail' ? 'bg-red-100 text-red-600' : it.isCompleted ? 'bg-emerald-100 text-emerald-600' : 'bg-gray-100 text-gray-500',
                ]"
              >
                <AppIcon v-if="it.result === 'fail'" name="x" :size="18" :stroke="2.5" />
                <AppIcon v-else-if="it.isCompleted" name="check" :size="18" :stroke="2.5" />
                <template v-else>{{ idx + 1 }}</template>
              </span>

              <div class="min-w-0 flex-1">
                <p :class="['text-[15px] font-medium leading-snug', it.isCompleted && it.itemType === 'check' ? 'text-gray-400' : 'text-gray-900']">{{ it.label }}</p>
                <div class="mt-1 flex flex-wrap items-center gap-1.5">
                  <span v-if="it.requirePhoto" :class="['inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-semibold', it.afterPhotoUrl ? 'bg-emerald-50 text-emerald-700' : 'bg-amber-50 text-amber-700']">
                    <AppIcon name="camera" :size="13" /> {{ L("cl.photoRequired", "After photo required") }}
                  </span>
                  <span v-if="it.isCompleted && (it.completedBy || it.completedAt)" class="inline-flex items-center gap-1.5 text-[11px] text-gray-400">
                    <span v-if="it.completedBy" class="flex h-5 w-5 items-center justify-center rounded-full bg-navy-50 text-[9px] font-bold text-navy-700">{{ initials(it.completedBy) }}</span>
                    {{ it.completedBy }}<template v-if="it.completedBy && it.completedAt"> · </template>{{ fmt(it.completedAt) }}
                  </span>
                </div>
              </div>
            </div>

            <!-- 1-5 stars -->
            <div v-if="it.itemType === 'score'" class="mt-4 flex items-center gap-1 sm:pl-12" role="group" :aria-label="L('cl.score', 'Score')">
              <button
                v-for="n in 5"
                :key="n"
                type="button"
                :disabled="locked"
                :aria-label="`${n} / 5`"
                :aria-pressed="it.score === n"
                :class="['field-tap-sm rounded-lg p-1 transition hover:scale-110 disabled:cursor-not-allowed disabled:hover:scale-100', (it.score ?? 0) >= n ? 'text-amber-400' : 'text-gray-200']"
                @click="setScore(it, n)"
              >
                <AppIcon name="star" :size="30" />
              </button>
              <span class="ml-2 text-sm font-semibold text-gray-700">{{ it.score ? `${it.score} / 5` : L("cl.score", "Score") }}</span>
            </div>

            <!-- pass / fail / n/a -->
            <div v-else-if="it.itemType === 'pass_fail'" class="mt-4 grid grid-cols-3 gap-2 sm:max-w-sm sm:pl-12" role="group">
              <button
                v-for="opt in resultOptions"
                :key="opt.value"
                type="button"
                :disabled="locked"
                :aria-pressed="it.result === opt.value"
                :class="[
                  'field-tap-sm inline-flex items-center justify-center gap-1.5 rounded-xl py-2.5 text-sm font-semibold ring-1 transition disabled:cursor-not-allowed',
                  it.result === opt.value ? opt.on : 'bg-white text-gray-600 ring-gray-200 hover:bg-gray-50',
                ]"
                @click="setResult(it, opt.value)"
              >
                <AppIcon :name="opt.icon" :size="16" :stroke="2.5" />
                {{ opt.label }}
              </button>
            </div>

            <!-- note -->
            <div class="sm:pl-12">
              <blockquote v-if="it.notes && noteOpen[it.id] === undefined" class="mt-3 rounded-2xl border-l-4 border-navy-300 bg-gray-50 px-3 py-2 text-sm text-gray-700 whitespace-pre-line">{{ it.notes }}</blockquote>
              <p v-if="it.result === 'fail' && !it.notes.trim() && noteOpen[it.id] === undefined" class="mt-3 inline-flex items-center gap-1.5 text-sm font-medium text-red-600">
                <AppIcon name="warning" :size="16" /> {{ L("cl.failNeedsNote", "Failed — add a note saying why") }}
              </p>
              <div v-if="noteOpen[it.id] !== undefined" class="mt-3 rounded-2xl bg-gray-50 p-2 ring-1 ring-gray-200">
                <textarea v-model="noteOpen[it.id]" rows="2" class="w-full resize-none border-0 bg-transparent px-2 py-1 text-sm focus:ring-0" :placeholder="L('cl.note', 'Note')"></textarea>
                <div class="flex justify-end gap-2">
                  <button type="button" class="rounded-xl px-3 py-1.5 text-sm font-medium text-gray-500 hover:bg-gray-100" @click="cancelNote(it.id)">
                    <AppIcon name="x" :size="16" />
                  </button>
                  <button type="button" class="rounded-xl bg-navy-600 px-3 py-1.5 text-sm font-semibold text-white" @click="saveNote(it)">{{ L("cl.saveNote", "Save note") }}</button>
                </div>
              </div>

              <!-- photos -->
              <div class="mt-3 flex flex-wrap items-center gap-2">
                <template v-for="kind in (['before', 'after'] as const)" :key="kind">
                  <div v-if="kind === 'before' ? it.beforePhotoUrl : it.afterPhotoUrl" class="relative">
                    <ChecklistPhoto :url="(kind === 'before' ? it.beforePhotoUrl : it.afterPhotoUrl)!" :label="kind === 'before' ? L('cl.before', 'Before') : L('cl.after', 'After')" />
                    <label v-if="!locked" class="absolute -right-1.5 -top-1.5 flex h-6 w-6 cursor-pointer items-center justify-center rounded-full bg-white text-gray-600 shadow ring-1 ring-gray-200" :aria-label="kind">
                      <AppIcon name="camera" :size="13" />
                      <input type="file" accept="image/jpeg,image/png,image/webp" class="hidden" @change="attach(it.id, kind, $event)" />
                    </label>
                  </div>
                  <label
                    v-else-if="!locked"
                    :class="[
                      'field-tap-sm flex h-16 w-16 cursor-pointer flex-col items-center justify-center gap-0.5 rounded-2xl border-2 border-dashed text-[10px] font-semibold uppercase tracking-wide transition hover:bg-navy-50',
                      kind === 'after' && it.requirePhoto ? 'border-amber-300 text-amber-600' : 'border-gray-200 text-gray-400',
                    ]"
                  >
                    <AppIcon name="camera" :size="20" />
                    {{ kind === "before" ? L("cl.before", "Before") : L("cl.after", "After") }}
                    <input type="file" accept="image/jpeg,image/png,image/webp" class="hidden" @change="attach(it.id, kind, $event)" />
                  </label>
                </template>
                <button
                  v-if="!locked && noteOpen[it.id] === undefined"
                  type="button"
                  class="field-tap-sm flex h-16 w-16 flex-col items-center justify-center gap-0.5 rounded-2xl border-2 border-dashed border-gray-200 text-[10px] font-semibold uppercase tracking-wide text-gray-400 transition hover:bg-navy-50"
                  @click="noteOpen = { ...noteOpen, [it.id]: it.notes }"
                >
                  <AppIcon name="pencil" :size="20" />
                  {{ L("cl.note", "Note") }}
                </button>
              </div>
            </div>
          </li>
        </ol>
        <p v-if="checklist.items.length === 0" class="rounded-3xl bg-white p-8 text-center text-sm text-gray-500 ring-1 ring-gray-100">{{ L("cl.noItems", "No items yet.") }}</p>

        <!-- Submit -->
        <section v-if="!locked && state !== 'approved' && checklist.items.length" class="rounded-3xl bg-white p-5 shadow-sm ring-1 ring-gray-100">
          <div class="flex items-center justify-between gap-3">
            <p class="text-sm font-semibold text-gray-900">{{ problems.length ? L("cl.finishFirst", "Finish these before submitting:") : L("cl.submit", "Submit for review") }}</p>
            <span class="text-xs font-semibold text-gray-500">{{ pct }}%</span>
          </div>
          <div class="mt-2 h-2 overflow-hidden rounded-full bg-gray-100">
            <div class="h-full rounded-full bg-gradient-to-r from-navy-500 to-emerald-500 transition-all duration-500" :style="{ width: `${pct}%` }"></div>
          </div>
          <ul v-if="problems.length" class="mt-3 space-y-1.5">
            <li v-for="it in problems.slice(0, 8)" :key="it.id" class="flex items-start gap-2 text-sm text-gray-700">
              <AppIcon name="warning" :size="16" class="mt-0.5 text-amber-500" />
              <span>{{ problemText(it) }}</span>
            </li>
            <li v-if="problems.length > 8" class="pl-6 text-sm text-gray-500">+{{ problems.length - 8 }}</li>
          </ul>
          <button
            type="button"
            :disabled="submitting || problems.length > 0"
            class="field-tap mt-4 inline-flex w-full items-center justify-center gap-2 rounded-2xl bg-gradient-to-r from-navy-700 to-navy-500 py-3.5 text-base font-bold text-white shadow-lg shadow-navy-600/25 transition active:scale-[0.99] disabled:from-gray-300 disabled:to-gray-300 disabled:shadow-none sm:w-auto sm:px-8 sm:text-sm"
            @click="handleSubmit"
          >
            <AppIcon name="send" :size="18" />
            {{ submitting ? L("cl.submitting", "Submitting…") : L("cl.submit", "Submit for review") }}
          </button>
        </section>

        <!-- Office review -->
        <section v-if="canManage && state === 'submitted'" class="rounded-3xl bg-white p-5 shadow-sm ring-1 ring-amber-200">
          <div class="flex items-center gap-2">
            <span class="flex h-9 w-9 items-center justify-center rounded-xl bg-amber-100 text-amber-600"><AppIcon name="inbox" :size="18" /></span>
            <h3 class="font-semibold text-gray-900">Review</h3>
          </div>
          <textarea v-model="reviewNote" rows="2" class="mt-3 w-full rounded-2xl border-0 bg-gray-50 px-3 py-2.5 text-sm ring-1 ring-gray-200 focus:ring-2 focus:ring-navy-500" placeholder="Note to the cleaner (required to send back)"></textarea>
          <div class="mt-3 flex flex-wrap gap-2">
            <button type="button" :disabled="reviewing" class="inline-flex items-center gap-2 rounded-2xl bg-emerald-600 px-5 py-2.5 text-sm font-semibold text-white shadow-sm shadow-emerald-600/30 hover:bg-emerald-700 disabled:opacity-50" @click="handleReview('approved')">
              <AppIcon name="badge" :size="18" /> Approve
            </button>
            <button type="button" :disabled="reviewing" class="inline-flex items-center gap-2 rounded-2xl bg-white px-5 py-2.5 text-sm font-semibold text-red-600 ring-1 ring-red-200 hover:bg-red-50 disabled:opacity-50" @click="handleReview('rework')">
              <AppIcon name="undo" :size="18" /> Send back for rework
            </button>
          </div>
        </section>

        <!-- Client sign-off -->
        <section class="rounded-3xl bg-white p-5 shadow-sm ring-1 ring-gray-100">
          <div class="flex items-center gap-2">
            <span class="flex h-9 w-9 items-center justify-center rounded-xl bg-navy-50 text-navy-600"><AppIcon name="pencil" :size="18" /></span>
            <h3 class="font-semibold text-gray-900">{{ L("cl.clientConfirm", "Client confirmation") }}</h3>
          </div>
          <div v-if="checklist.clientSignature" class="mt-3 flex items-center gap-3 rounded-2xl bg-emerald-50 p-3 text-sm text-emerald-800">
            <AppIcon name="badge" :size="20" class="text-emerald-600" />
            <span>{{ L("cl.signedBy", "Signed by") }} <strong>{{ checklist.clientSignature }}</strong><span v-if="checklist.clientConfirmedAt" class="text-emerald-700/70"> · {{ fmt(checklist.clientConfirmedAt) }}</span></span>
          </div>
          <div v-else class="mt-3 flex flex-col gap-2 sm:flex-row">
            <input v-model="signature" type="text" :placeholder="L('cl.signaturePlaceholder', 'Client name / signature')" class="w-full rounded-2xl border-0 bg-gray-50 px-3 py-2.5 text-sm ring-1 ring-gray-200 focus:ring-2 focus:ring-navy-500 sm:max-w-sm" />
            <button type="button" :disabled="confirming" class="inline-flex items-center justify-center gap-2 rounded-2xl bg-emerald-600 px-5 py-2.5 text-sm font-semibold text-white disabled:opacity-50" @click="handleConfirm">
              <AppIcon name="check" :size="17" :stroke="2.5" />
              {{ confirming ? L("cl.confirming", "Confirming…") : L("cl.confirm", "Confirm") }}
            </button>
          </div>
        </section>
      </template>

      <!-- Empty states -->
      <section v-else class="flex flex-col items-center rounded-3xl bg-white px-6 py-14 text-center shadow-sm ring-1 ring-gray-100">
        <span class="flex h-14 w-14 items-center justify-center rounded-2xl bg-navy-50 text-navy-500"><AppIcon name="clipboard" :size="28" /></span>
        <p class="mt-4 max-w-sm text-sm text-gray-500">
          <template v-if="missing && noTemplate">
            <span class="block font-semibold text-gray-800">{{ L("cl.noTemplateTitle", "No checklist template for this service yet") }}</span>
            <span class="mt-1 block">{{ isCleaner ? L("cl.noTemplateCleaner", "") : `Add a template for “${selectedBooking ? serviceLabel(selectedBooking.serviceType) : "this service"}” (or a general one for any service), then open the booking again.` }}</span>
          </template>
          <template v-else-if="missing">{{ isCleaner ? L("cl.noChecklist", "") : "No checklist for this booking yet." }}</template>
          <template v-else>{{ L("cl.empty", "Pick a booking to open its checklist.") }}</template>
        </p>
        <button
          v-if="missing && noTemplate && canManage"
          type="button"
          class="mt-5 inline-flex items-center gap-2 rounded-2xl bg-navy-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-navy-700"
          @click="view = 'templates'"
        >
          <AppIcon name="stack" :size="17" /> Go to Templates
        </button>
      </section>
    </template>
  </div>
</template>
