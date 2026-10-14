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
  setUrl(Number(bookingId.value));
  try {
    checklist.value = await getChecklistByBooking(Number(bookingId.value));
    reviewNote.value = "";
  } catch (err) {
    // 404 just means no checklist yet; the empty state says what to do.
    if (err instanceof ApiError && err.status === 404) {
      missing.value = true;
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

async function saveNote(it: ChecklistItem) {
  const notes = (noteOpen.value[it.id] ?? "").trim();
  if (await answer(it, { isCompleted: it.isCompleted, notes }, { notes })) {
    const next = { ...noteOpen.value };
    delete next[it.id];
    noteOpen.value = next;
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
  <div class="rounded-xl border border-gray-200 bg-white">
    <div class="border-b border-gray-200 px-6 py-4">
      <LanguageSwitcher v-if="isCleaner" class="mb-2" />
      <h2 class="text-base font-semibold text-gray-900">{{ L("cl.title", "Checklists & proof of work") }}</h2>
      <p class="mt-1 text-sm text-gray-500">{{ L("cl.subtitle", "Crews fill these in on the job; review what they scored and the photos they took.") }}</p>
      <div v-if="!isCleaner" class="mt-3 flex gap-1 border-b border-transparent">
        <button
          v-for="tab in ([['review', 'Review'], ['checklist', 'Checklist'], ...(canManage ? [['templates', 'Templates']] : [])] as [View, string][])"
          :key="tab[0]"
          type="button"
          :class="['rounded-lg px-3 py-1.5 text-sm font-medium', view === tab[0] ? 'bg-navy-50 text-navy-700' : 'text-gray-600 hover:bg-gray-50']"
          @click="tab[0] === 'review' ? backToList() : (view = tab[0])"
        >{{ tab[1] }}</button>
      </div>
    </div>

    <ChecklistReviewList v-if="!isCleaner && view === 'review'" @open="openBooking" />
    <ChecklistTemplates v-else-if="!isCleaner && view === 'templates'" />

    <template v-else>
      <div class="flex flex-col gap-2 border-b border-gray-100 px-6 py-3 sm:flex-row">
        <select v-model="bookingId" :disabled="bookingsLoading" class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm sm:w-80" @change="handleLoad">
          <option value="" disabled>{{ bookingsLoading ? L("cl.loadingBookings", "Loading bookings…") : L("cl.selectBooking", "Select booking") }}</option>
          <option v-if="bookingId !== '' && !selectedBooking" :value="bookingId">#{{ bookingId }}</option>
          <option v-for="b in bookings" :key="b.id" :value="b.id">
            {{ b.bookingNumber }} — {{ b.customerName }}
          </option>
        </select>
        <template v-if="canManage && missing">
          <select v-model="templateId" class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm sm:w-72" aria-label="Template">
            <option value="">Auto{{ autoTemplate ? ` — ${autoTemplate.name}` : "" }}</option>
            <option v-for="tp in templates.filter((x) => x.isActive)" :key="tp.id" :value="tp.id">
              {{ tp.name }} ({{ tp.items.length }})
            </option>
          </select>
          <button type="button" class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white" @click="handleCreate">New from template</button>
        </template>
      </div>

      <div v-if="loading" class="px-6 py-10 text-center text-sm text-gray-500">{{ L("cl.loading", "Loading…") }}</div>
      <div v-else-if="checklist" class="px-4 py-4 sm:px-6">
        <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-gray-500">
          <span>{{ selectedBooking?.bookingNumber ?? `Booking #${checklist.bookingId}` }}<span v-if="selectedBooking"> · {{ serviceLabel(selectedBooking.serviceType) }}</span></span>
          <span class="font-medium text-gray-900">{{ doneCount }}/{{ checklist.items.length }} {{ L("cl.done", "done") }}</span>
          <span v-if="checklist.avgScore != null" class="font-medium text-gray-900">★ {{ checklist.avgScore.toFixed(1) }} / 5</span>
          <span v-if="!isCleaner && state" :class="['rounded-full px-2 py-0.5 text-[11px] font-semibold', CHECKLIST_STATE_BADGE[state].tone]">{{ CHECKLIST_STATE_BADGE[state].label }}</span>
          <a
            :href="`/checklists/print?booking=${checklist.bookingId}&lang=${isCleaner ? 'my' : 'en'}`"
            target="_blank"
            rel="noopener"
            class="field-tap-sm ml-auto inline-flex items-center rounded-lg bg-gray-50 px-2.5 py-1.5 text-sm font-medium text-navy-600 ring-1 ring-gray-200"
          >📄 PDF</a>
        </div>

        <!-- Where the checklist stands -->
        <div v-if="state === 'rework'" class="mt-3 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800">
          <p class="font-semibold">{{ L("cl.rework", "Sent back to the cleaner for rework") }}</p>
          <p class="mt-1 whitespace-pre-line">{{ checklist.reviewNote }}</p>
          <p v-if="!isCleaner && checklist.reviewedBy" class="mt-1 text-xs text-red-700">{{ checklist.reviewedBy }} · {{ fmt(checklist.reviewedAt) }}</p>
        </div>
        <div v-else-if="state === 'submitted'" class="mt-3 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
          <p class="font-semibold">{{ L("cl.waitingReview", "Submitted — waiting for review") }}</p>
          <p v-if="checklist.submittedBy" class="mt-1 text-xs">{{ checklist.submittedBy }} · {{ fmt(checklist.submittedAt) }}</p>
        </div>
        <div v-else-if="state === 'approved'" class="mt-3 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-800">
          <p class="font-semibold">{{ L("cl.approved", "Approved") }}</p>
          <p v-if="checklist.reviewNote" class="mt-1 whitespace-pre-line">{{ checklist.reviewNote }}</p>
          <p class="mt-1 text-xs">
            <template v-if="checklist.submittedBy">{{ checklist.submittedBy }} · {{ fmt(checklist.submittedAt) }}</template>
            <template v-if="!isCleaner && checklist.reviewedBy"> — approved by {{ checklist.reviewedBy }} · {{ fmt(checklist.reviewedAt) }}</template>
          </p>
        </div>

        <ul class="mt-3 space-y-2">
          <li v-for="it in checklist.items" :key="it.id" class="rounded-lg border border-gray-100 px-3 py-3">
            <div class="flex flex-wrap items-center gap-3">
              <input
                v-if="it.itemType === 'check'"
                type="checkbox"
                :checked="it.isCompleted"
                :disabled="locked"
                class="field-tap-sm h-6 w-6 accent-navy-600 sm:h-4 sm:w-4"
                :aria-label="it.label"
                @change="toggle(it)"
              />
              <span v-else :class="['flex h-6 w-6 items-center justify-center rounded-full text-xs sm:h-5 sm:w-5', it.isCompleted ? 'bg-navy-600 text-white' : 'bg-gray-100 text-gray-400']">✓</span>
              <span class="min-w-0 flex-1 text-base sm:text-sm" :class="it.isCompleted && it.itemType === 'check' ? 'text-gray-400 line-through' : 'text-gray-900'">
                {{ it.label }}
                <span v-if="it.requirePhoto" class="ml-1 whitespace-nowrap rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-semibold text-gray-600">📷 {{ L("cl.photoRequired", "After photo required") }}</span>
              </span>
            </div>

            <!-- 1-5 score -->
            <div v-if="it.itemType === 'score'" class="mt-2 flex items-center gap-1.5" role="group" :aria-label="L('cl.score', 'Score')">
              <span class="mr-1 text-xs text-gray-500">{{ L("cl.score", "Score") }}</span>
              <button
                v-for="n in 5"
                :key="n"
                type="button"
                :disabled="locked"
                :class="['field-tap-sm h-9 w-9 rounded-lg text-sm font-bold ring-1 disabled:cursor-not-allowed', (it.score ?? 0) >= n ? 'bg-amber-400 text-white ring-amber-400' : 'bg-white text-gray-500 ring-gray-200']"
                :aria-pressed="it.score === n"
                @click="setScore(it, n)"
              >{{ n }}</button>
            </div>

            <!-- pass / fail / n/a -->
            <div v-else-if="it.itemType === 'pass_fail'" class="mt-2 flex gap-1.5" role="group">
              <button
                v-for="opt in ([['pass', L('cl.pass', 'Pass'), 'bg-green-600 ring-green-600'], ['fail', L('cl.fail', 'Fail'), 'bg-red-600 ring-red-600'], ['na', L('cl.na', 'N/A'), 'bg-gray-500 ring-gray-500']] as [ItemResult, string, string][])"
                :key="opt[0]"
                type="button"
                :disabled="locked"
                :class="['field-tap-sm rounded-lg px-4 py-2 text-sm font-semibold ring-1 disabled:cursor-not-allowed', it.result === opt[0] ? `${opt[2]} text-white` : 'bg-white text-gray-600 ring-gray-200']"
                :aria-pressed="it.result === opt[0]"
                @click="setResult(it, opt[0])"
              >{{ opt[1] }}</button>
            </div>

            <!-- note -->
            <p v-if="it.notes && noteOpen[it.id] === undefined" class="mt-2 whitespace-pre-line rounded bg-gray-50 px-2 py-1 text-sm text-gray-700">📝 {{ it.notes }}</p>
            <p v-if="it.result === 'fail' && !it.notes.trim() && noteOpen[it.id] === undefined" class="mt-2 text-sm font-medium text-red-600">{{ L("cl.failNeedsNote", "Failed — add a note saying why") }}</p>
            <div v-if="noteOpen[it.id] !== undefined" class="mt-2 flex flex-col gap-2 sm:flex-row">
              <textarea v-model="noteOpen[it.id]" rows="2" class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm" :placeholder="L('cl.note', 'Note')"></textarea>
              <button type="button" class="rounded-lg bg-navy-600 px-3 py-2 text-sm font-medium text-white" @click="saveNote(it)">{{ L("cl.saveNote", "Save note") }}</button>
            </div>

            <!-- photos and who did it -->
            <div class="mt-2 flex flex-wrap items-center gap-2 text-sm sm:text-xs">
              <ChecklistPhoto v-if="it.beforePhotoUrl" :url="it.beforePhotoUrl" :label="L('cl.before', 'Before')" />
              <ChecklistPhoto v-if="it.afterPhotoUrl" :url="it.afterPhotoUrl" :label="L('cl.after', 'After')" />
              <template v-if="!locked">
                <label class="field-tap-sm inline-flex cursor-pointer items-center rounded-lg bg-gray-50 px-2.5 py-1.5 font-medium text-navy-600">📷 {{ L("cl.before", "Before") }}<input type="file" accept="image/jpeg,image/png,image/webp" class="hidden" @change="attach(it.id, 'before', $event)" /></label>
                <label class="field-tap-sm inline-flex cursor-pointer items-center rounded-lg bg-gray-50 px-2.5 py-1.5 font-medium text-navy-600">📷 {{ L("cl.after", "After") }}<input type="file" accept="image/jpeg,image/png,image/webp" class="hidden" @change="attach(it.id, 'after', $event)" /></label>
                <button v-if="noteOpen[it.id] === undefined" type="button" class="field-tap-sm rounded-lg bg-gray-50 px-2.5 py-1.5 font-medium text-navy-600" @click="noteOpen = { ...noteOpen, [it.id]: it.notes }">📝 {{ L("cl.addNote", "Add note") }}</button>
              </template>
              <span v-if="it.isCompleted && (it.completedBy || it.completedAt)" class="ml-auto text-xs text-gray-400">{{ it.completedBy }}<template v-if="it.completedBy && it.completedAt"> · </template>{{ fmt(it.completedAt) }}</span>
            </div>
          </li>
        </ul>
        <p v-if="checklist.items.length === 0" class="mt-2 text-sm text-gray-500">{{ L("cl.noItems", "No items yet.") }}</p>

        <!-- Submit to office -->
        <div v-if="!locked && state !== 'approved' && checklist.items.length" class="mt-4 rounded-lg border border-navy-100 bg-navy-50/40 p-4">
          <template v-if="problems.length">
            <p class="text-sm font-semibold text-gray-900">{{ L("cl.finishFirst", "Finish these before submitting:") }}</p>
            <ul class="mt-1 list-disc pl-5 text-sm text-gray-700">
              <li v-for="it in problems.slice(0, 8)" :key="it.id">{{ problemText(it) }}</li>
              <li v-if="problems.length > 8">+{{ problems.length - 8 }}</li>
            </ul>
          </template>
          <button
            type="button"
            :disabled="submitting || problems.length > 0"
            class="field-tap mt-3 w-full rounded-xl bg-navy-600 py-3 text-base font-bold text-white disabled:opacity-40 sm:w-auto sm:px-6 sm:py-2 sm:text-sm"
            @click="handleSubmit"
          >{{ submitting ? L("cl.submitting", "Submitting…") : L("cl.submit", "Submit for review") }}</button>
        </div>

        <!-- Office review -->
        <div v-if="canManage && state === 'submitted'" class="mt-4 rounded-lg border border-amber-200 bg-white p-4">
          <h3 class="text-sm font-semibold text-gray-900">Review</h3>
          <textarea v-model="reviewNote" rows="2" class="mt-2 w-full rounded-lg border border-gray-200 px-3 py-2 text-sm" placeholder="Note to the cleaner (required to send back)"></textarea>
          <div class="mt-2 flex flex-wrap gap-2">
            <button type="button" :disabled="reviewing" class="rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" @click="handleReview('approved')">Approve</button>
            <button type="button" :disabled="reviewing" class="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" @click="handleReview('rework')">Send back for rework</button>
          </div>
        </div>

        <div class="mt-4 rounded-lg border border-gray-100 bg-gray-50 p-4">
          <h3 class="text-sm font-semibold text-gray-900">{{ L("cl.clientConfirm", "Client confirmation") }}</h3>
          <p v-if="checklist.clientSignature" class="mt-1 text-sm text-green-700">{{ L("cl.signedBy", "Signed by") }} {{ checklist.clientSignature }}<span v-if="checklist.clientConfirmedAt" class="text-gray-500"> · {{ fmt(checklist.clientConfirmedAt) }}</span></p>
          <div v-else class="mt-2 flex flex-col gap-2 sm:flex-row">
            <input v-model="signature" type="text" :placeholder="L('cl.signaturePlaceholder', 'Client name / signature')" class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm sm:w-80" />
            <button type="button" :disabled="confirming" class="rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" @click="handleConfirm">{{ confirming ? L("cl.confirming", "Confirming…") : L("cl.confirm", "Confirm") }}</button>
          </div>
        </div>
      </div>
      <div v-else-if="missing" class="px-6 py-10 text-center text-sm text-gray-500">
        {{ isCleaner ? L("cl.noChecklist", "") : "No checklist yet. It is created from the service's template when the cleaner opens the job, or create it now." }}
      </div>
      <div v-else class="px-6 py-10 text-center text-sm text-gray-500">{{ L("cl.empty", "Pick a booking to open its checklist.") }}</div>
    </template>
  </div>
</template>
