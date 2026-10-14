<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  checklistPhotoUrl,
  checklistState,
  getChecklistByBooking,
  type Checklist,
  type ChecklistItem,
  type ChecklistState,
} from "../../../lib/checklists";
import { getBooking, type Booking } from "../../../lib/bookings";
import { getWorkspaceSettings, serviceLabel } from "../../../lib/settings";

// Checklist report for the customer or the file, printed to PDF by the
// browser. Headings come in English, Myanmar, or both; item labels and notes
// stay as the template and the crew wrote them.

type PdfLang = "en" | "my" | "both";

const S = {
  title: ["Cleaning checklist report", "သန့်ရှင်းရေး စစ်ဆေးစာရင်း အစီရင်ခံစာ"],
  booking: ["Booking", "အလုပ်အမှတ်"],
  customer: ["Customer", "ဖောက်သည်"],
  address: ["Address", "လိပ်စာ"],
  date: ["Date", "ရက်စွဲ"],
  service: ["Service", "ဝန်ဆောင်မှု"],
  cleaners: ["Cleaners", "သန့်ရှင်းရေးဝန်ထမ်း"],
  summary: ["Summary", "အကျဉ်းချုပ်"],
  done: ["Items done", "ပြီးစီးသည့် အချက်"],
  avgScore: ["Average score", "ပျမ်းမျှ အမှတ်"],
  fails: ["Failed items", "ကျသည့် အချက်"],
  state: ["Status", "အခြေအနေ"],
  submitted: ["Submitted by", "တင်သွင်းသူ"],
  reviewed: ["Reviewed by", "စစ်ဆေးသူ"],
  item: ["Item", "အချက်"],
  result: ["Result", "ရလဒ်"],
  notes: ["Notes", "မှတ်ချက်"],
  by: ["Done by", "လုပ်ဆောင်သူ"],
  photos: ["Photos", "ဓာတ်ပုံများ"],
  before: ["Before", "မလုပ်မီ"],
  after: ["After", "လုပ်ပြီး"],
  pass: ["Pass", "အောင်"],
  fail: ["Fail", "ကျ"],
  na: ["N/A", "မသက်ဆိုင်"],
  notDone: ["Not done", "မပြီးသေး"],
  doneTick: ["Done", "ပြီး"],
  reviewNote: ["Office note", "ရုံး မှတ်ချက်"],
  client: ["Client confirmation", "ဖောက်သည် အတည်ပြုချက်"],
  signedBy: ["Signed by", "လက်မှတ်ထိုးသူ"],
  notSigned: ["Not signed yet", "လက်မှတ် မထိုးရသေး"],
  generated: ["Generated", "ထုတ်သည့်ရက်"],
  print: ["Save as PDF / Print", "PDF သိမ်းမည် / ပရင့်ထုတ်မည်"],
  pdfHint: ["In the print dialog choose “Save as PDF”.", "ပရင့် window တွင် “Save as PDF” ကို ရွေးပါ။"],
  loading: ["Loading…", "ဖွင့်နေသည်…"],
  missing: ["No checklist for this booking.", "ဒီအလုပ်အတွက် စစ်ဆေးစာရင်း မရှိပါ။"],
  pending: ["Not started", "မစရသေး"],
  in_progress: ["In progress", "လုပ်ဆောင်နေသည်"],
  st_submitted: ["Waiting for office review", "ရုံးစစ်ဆေးရန် စောင့်နေသည်"],
  approved: ["Approved", "အတည်ပြုပြီး"],
  rework: ["Sent back for rework", "ပြန်ပြင်ရန် ပို့ထားသည်"],
} as const;
type Key = keyof typeof S;

const params = new URLSearchParams(window.location.search);
const bookingId = Number(params.get("booking")) || 0;
const initial = params.get("lang");
const lang = ref<PdfLang>(initial === "my" || initial === "both" ? initial : "en");

function T(k: Key): string {
  const [en, my] = S[k];
  return lang.value === "en" ? en : lang.value === "my" ? my : `${en} / ${my}`;
}

const checklist = ref<Checklist | null>(null);
const booking = ref<Booking | null>(null);
const loading = ref(true);
const error = ref("");
const photos = ref<Record<string, string>>({});

// One date format in every language: browsers differ in Myanmar date support.
function when(iso?: string | null, withTime = true): string {
  if (!iso) return "";
  return new Date(iso).toLocaleString("en-GB", {
    day: "numeric", month: "short", year: "numeric",
    ...(withTime ? { hour: "2-digit", minute: "2-digit" } : {}),
  });
}

const stateKey: Record<ChecklistState, Key> = {
  pending: "pending", in_progress: "in_progress", submitted: "st_submitted", approved: "approved", rework: "rework",
};
const state = computed(() => (checklist.value ? checklistState(checklist.value) : null));
const doneCount = computed(() => checklist.value?.items.filter((i) => i.isCompleted).length ?? 0);
const failCount = computed(() => checklist.value?.items.filter((i) => i.result === "fail").length ?? 0);
const photoItems = computed(() => checklist.value?.items.filter((i) => i.beforePhotoUrl || i.afterPhotoUrl) ?? []);

function resultText(it: ChecklistItem): string {
  if (it.itemType === "score") return it.score ? `${it.score} / 5` : T("notDone");
  if (it.itemType === "pass_fail") return it.result ? T(it.result) : T("notDone");
  return it.isCompleted ? `✓ ${T("doneTick")}` : T("notDone");
}
function resultTone(it: ChecklistItem): string {
  if (!it.isCompleted) return "text-gray-400";
  if (it.result === "fail" || (it.score != null && it.score <= 2)) return "text-red-700";
  return "text-green-700";
}

function setTitle() {
  // The browser uses the title as the PDF file name.
  const no = booking.value?.bookingNumber ?? `booking-${bookingId}`;
  document.title = `Checklist-${no}-${lang.value.toUpperCase()}`;
}

function setLang(l: PdfLang) {
  lang.value = l;
  const url = new URL(window.location.href);
  url.searchParams.set("lang", l);
  window.history.replaceState(null, "", url);
  setTitle();
}

function print() {
  setTitle();
  window.print();
}

onMounted(async () => {
  try {
    if (!bookingId) throw new Error(T("missing"));
    await getWorkspaceSettings().catch(() => null);
    const [c, b] = await Promise.all([
      getChecklistByBooking(bookingId),
      getBooking(bookingId).catch(() => null),
    ]);
    checklist.value = c;
    booking.value = b;
    setTitle();
    // Load every photo before printing so none are blank in the PDF.
    const urls = c.items.flatMap((i) => [i.beforePhotoUrl, i.afterPhotoUrl]).filter((u): u is string => !!u);
    const loaded = await Promise.all(urls.map((u) => checklistPhotoUrl(u).then((src) => [u, src] as const).catch(() => null)));
    photos.value = Object.fromEntries(loaded.filter((x): x is readonly [string, string] => !!x));
  } catch (err) {
    error.value = err instanceof Error ? err.message : T("missing");
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <div class="mx-auto max-w-[210mm] px-3 py-4 print:max-w-none print:p-0">
    <!-- toolbar (screen only) -->
    <div class="mb-4 flex flex-wrap items-center gap-2 rounded-xl bg-white p-3 shadow-sm print:hidden">
      <div class="flex overflow-hidden rounded-lg ring-1 ring-gray-200" role="group" aria-label="Language">
        <button
          v-for="opt in ([['en', 'English'], ['my', 'မြန်မာ'], ['both', 'English + မြန်မာ']] as [PdfLang, string][])"
          :key="opt[0]"
          type="button"
          :class="['px-3 py-1.5 text-sm font-medium', lang === opt[0] ? 'bg-navy-600 text-white' : 'bg-white text-gray-700']"
          :aria-pressed="lang === opt[0]"
          @click="setLang(opt[0])"
        >{{ opt[1] }}</button>
      </div>
      <button type="button" :disabled="!checklist" class="ml-auto rounded-lg bg-navy-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-40" @click="print">⬇ {{ T("print") }}</button>
      <p class="w-full text-xs text-gray-500">{{ T("pdfHint") }}</p>
    </div>

    <div v-if="loading" class="py-16 text-center text-sm text-gray-500">{{ T("loading") }}</div>
    <div v-else-if="error || !checklist" class="py-16 text-center text-sm text-gray-500">{{ error || T("missing") }}</div>

    <article v-else class="rounded-xl bg-white p-8 text-[13px] leading-relaxed text-gray-900 shadow-sm print:rounded-none print:p-0 print:shadow-none">
      <header class="flex items-start justify-between gap-4 border-b-2 border-navy-600 pb-4">
        <div class="flex items-center gap-3">
          <img src="/logo/logo.png" alt="Smile Clean" class="h-12 w-12 object-contain" />
          <div>
            <p class="text-lg font-bold text-navy-800">Smile Clean</p>
            <h1 class="text-base font-semibold">{{ T("title") }}</h1>
          </div>
        </div>
        <div class="text-right text-xs text-gray-500">
          <p class="text-sm font-semibold text-gray-900">{{ booking?.bookingNumber ?? `#${checklist.bookingId}` }}</p>
          <p>{{ T("generated") }}: {{ when(new Date().toISOString()) }}</p>
        </div>
      </header>

      <!-- booking -->
      <section class="mt-4 grid grid-cols-2 gap-x-6 gap-y-1.5">
        <p v-if="booking"><span class="text-gray-500">{{ T("customer") }}:</span> <strong>{{ booking.customerName }}</strong></p>
        <p v-if="booking"><span class="text-gray-500">{{ T("date") }}:</span> {{ when(booking.scheduledFor) }}</p>
        <p v-if="booking?.address" class="col-span-2"><span class="text-gray-500">{{ T("address") }}:</span> {{ booking.address }}</p>
        <p v-if="booking"><span class="text-gray-500">{{ T("service") }}:</span> {{ serviceLabel(booking.serviceType) }}</p>
        <p v-if="booking?.cleaners?.length"><span class="text-gray-500">{{ T("cleaners") }}:</span> {{ booking.cleaners.map((c) => c.name).join(", ") }}</p>
      </section>

      <!-- summary -->
      <section class="mt-4 grid grid-cols-4 gap-2 break-inside-avoid">
        <div class="rounded-lg bg-gray-50 p-3">
          <p class="text-[11px] text-gray-500">{{ T("done") }}</p>
          <p class="text-lg font-bold">{{ doneCount }} / {{ checklist.items.length }}</p>
        </div>
        <div class="rounded-lg bg-gray-50 p-3">
          <p class="text-[11px] text-gray-500">{{ T("avgScore") }}</p>
          <p class="text-lg font-bold">{{ checklist.avgScore != null ? `${checklist.avgScore.toFixed(1)} / 5` : "—" }}</p>
        </div>
        <div class="rounded-lg bg-gray-50 p-3">
          <p class="text-[11px] text-gray-500">{{ T("fails") }}</p>
          <p :class="['text-lg font-bold', failCount ? 'text-red-700' : '']">{{ failCount }}</p>
        </div>
        <div class="rounded-lg bg-gray-50 p-3">
          <p class="text-[11px] text-gray-500">{{ T("state") }}</p>
          <p class="text-sm font-bold">{{ state ? T(stateKey[state]) : "" }}</p>
        </div>
      </section>
      <p v-if="checklist.submittedBy || checklist.reviewedBy" class="mt-2 text-xs text-gray-600">
        <template v-if="checklist.submittedBy">{{ T("submitted") }}: {{ checklist.submittedBy }} · {{ when(checklist.submittedAt) }}</template>
        <template v-if="checklist.submittedBy && checklist.reviewedBy"> — </template>
        <template v-if="checklist.reviewedBy">{{ T("reviewed") }}: {{ checklist.reviewedBy }} · {{ when(checklist.reviewedAt) }}</template>
      </p>
      <p v-if="checklist.reviewNote" class="mt-2 rounded-lg border border-gray-200 p-2 text-xs"><strong>{{ T("reviewNote") }}:</strong> {{ checklist.reviewNote }}</p>

      <!-- items -->
      <table class="mt-4 w-full border-collapse text-left">
        <thead>
          <tr class="border-b border-gray-300 text-[11px] text-gray-500">
            <th class="w-6 py-1.5 pr-2">#</th>
            <th class="py-1.5 pr-2">{{ T("item") }}</th>
            <th class="w-28 py-1.5 pr-2">{{ T("result") }}</th>
            <th class="py-1.5 pr-2">{{ T("notes") }}</th>
            <th class="w-32 py-1.5">{{ T("by") }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(it, i) in checklist.items" :key="it.id" class="break-inside-avoid border-b border-gray-100 align-top">
            <td class="py-1.5 pr-2 text-gray-500">{{ i + 1 }}</td>
            <td class="py-1.5 pr-2">{{ it.label }}</td>
            <td :class="['py-1.5 pr-2 font-semibold', resultTone(it)]">{{ resultText(it) }}</td>
            <td class="whitespace-pre-line py-1.5 pr-2 text-gray-700">{{ it.notes }}</td>
            <td class="py-1.5 text-[11px] text-gray-500">
              <template v-if="it.isCompleted">{{ it.completedBy }}<br v-if="it.completedBy" />{{ when(it.completedAt) }}</template>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- photos -->
      <section v-if="photoItems.length" class="mt-6">
        <h2 class="mb-2 text-sm font-semibold">{{ T("photos") }}</h2>
        <div class="grid grid-cols-2 gap-3">
          <figure v-for="it in photoItems" :key="it.id" class="break-inside-avoid rounded-lg border border-gray-200 p-2">
            <figcaption class="mb-1.5 text-xs font-medium">{{ it.label }}</figcaption>
            <div class="grid grid-cols-2 gap-2">
              <div v-for="kind in (['before', 'after'] as const)" :key="kind">
                <template v-if="kind === 'before' ? it.beforePhotoUrl : it.afterPhotoUrl">
                  <img
                    v-if="photos[(kind === 'before' ? it.beforePhotoUrl : it.afterPhotoUrl)!]"
                    :src="photos[(kind === 'before' ? it.beforePhotoUrl : it.afterPhotoUrl)!]"
                    :alt="`${it.label} — ${T(kind)}`"
                    class="aspect-[4/3] w-full rounded object-cover"
                  />
                  <div v-else class="flex aspect-[4/3] items-center justify-center rounded bg-gray-100 text-xs text-gray-400">—</div>
                  <p class="mt-0.5 text-center text-[11px] text-gray-500">{{ T(kind) }}</p>
                </template>
              </div>
            </div>
          </figure>
        </div>
      </section>

      <!-- client sign-off -->
      <section class="mt-6 break-inside-avoid rounded-lg border border-gray-200 p-3">
        <h2 class="text-sm font-semibold">{{ T("client") }}</h2>
        <p v-if="checklist.clientSignature" class="mt-1">{{ T("signedBy") }}: <strong>{{ checklist.clientSignature }}</strong> · {{ when(checklist.clientConfirmedAt) }}</p>
        <p v-else class="mt-1 text-gray-500">{{ T("notSigned") }}</p>
      </section>
    </article>
  </div>
</template>
