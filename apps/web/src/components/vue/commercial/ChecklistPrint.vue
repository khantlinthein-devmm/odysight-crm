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
import AppIcon, { type IconName } from "../ui/AppIcon.vue";

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
  print: ["Save as PDF", "PDF သိမ်းမည်"],
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
const stateTone: Record<ChecklistState, string> = {
  pending: "bg-gray-100 text-gray-700",
  in_progress: "bg-blue-100 text-blue-800",
  submitted: "bg-amber-100 text-amber-800",
  approved: "bg-emerald-100 text-emerald-800",
  rework: "bg-red-100 text-red-800",
};
const state = computed(() => (checklist.value ? checklistState(checklist.value) : null));
const doneCount = computed(() => checklist.value?.items.filter((i) => i.isCompleted).length ?? 0);
const failCount = computed(() => checklist.value?.items.filter((i) => i.result === "fail").length ?? 0);
const pct = computed(() =>
  checklist.value?.items.length ? Math.round((doneCount.value / checklist.value.items.length) * 100) : 0,
);
const photoItems = computed(() => checklist.value?.items.filter((i) => i.beforePhotoUrl || i.afterPhotoUrl) ?? []);

/** Result pill for non-score items. */
function pill(it: ChecklistItem): { text: string; icon: IconName; tone: string } {
  if (!it.isCompleted) return { text: T("notDone"), icon: "minus", tone: "bg-gray-100 text-gray-500" };
  if (it.result === "fail") return { text: T("fail"), icon: "x", tone: "bg-red-100 text-red-700" };
  if (it.result === "na") return { text: T("na"), icon: "minus", tone: "bg-gray-100 text-gray-600" };
  if (it.result === "pass") return { text: T("pass"), icon: "check", tone: "bg-emerald-100 text-emerald-700" };
  return { text: T("doneTick"), icon: "check", tone: "bg-emerald-100 text-emerald-700" };
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
  <div class="mx-auto max-w-[210mm] px-3 py-5 print:max-w-none print:p-0">
    <!-- Toolbar (screen only) -->
    <div class="mb-5 flex flex-wrap items-center gap-3 rounded-3xl bg-white p-3 shadow-sm ring-1 ring-gray-100 print:hidden">
      <div class="inline-flex rounded-2xl bg-gray-50 p-1 ring-1 ring-gray-200" role="radiogroup" aria-label="Language">
        <button
          v-for="opt in ([['en', 'English'], ['my', 'မြန်မာ'], ['both', 'English + မြန်မာ']] as [PdfLang, string][])"
          :key="opt[0]"
          type="button"
          role="radio"
          :aria-checked="lang === opt[0]"
          :class="['rounded-xl px-3.5 py-1.5 text-sm font-semibold transition', lang === opt[0] ? 'bg-white text-navy-700 shadow-sm ring-1 ring-gray-200' : 'text-gray-500 hover:text-gray-700']"
          @click="setLang(opt[0])"
        >{{ opt[1] }}</button>
      </div>
      <p class="hidden text-xs text-gray-400 sm:block">{{ T("pdfHint") }}</p>
      <button
        type="button"
        :disabled="!checklist"
        class="ml-auto inline-flex items-center gap-2 rounded-2xl bg-gradient-to-r from-navy-700 to-navy-500 px-5 py-2.5 text-sm font-semibold text-white shadow-lg shadow-navy-600/25 disabled:opacity-40"
        @click="print"
      >
        <AppIcon name="download" :size="18" />
        {{ T("print") }}
      </button>
    </div>

    <div v-if="loading" class="h-[60vh] animate-pulse rounded-3xl bg-white"></div>
    <div v-else-if="error || !checklist" class="flex flex-col items-center rounded-3xl bg-white px-6 py-16 text-center text-sm text-gray-500">
      <AppIcon name="clipboard" :size="32" class="text-gray-300" />
      <p class="mt-3">{{ error || T("missing") }}</p>
    </div>

    <article v-else class="overflow-hidden rounded-3xl bg-white text-[12.5px] leading-relaxed text-gray-900 shadow-sm print:rounded-none print:shadow-none">
      <!-- Header band -->
      <header class="relative overflow-hidden bg-gradient-to-br from-navy-800 via-navy-600 to-blue-500 px-8 py-6 text-white">
        <div class="pointer-events-none absolute -right-10 -top-16 h-48 w-48 rounded-full bg-white/10"></div>
        <div class="relative flex items-center justify-between gap-4">
          <div class="flex items-center gap-3">
            <span class="flex h-12 w-12 items-center justify-center rounded-2xl bg-white p-1.5">
              <img src="/logo/logo.png" alt="Smile Clean" class="h-full w-full object-contain" />
            </span>
            <div>
              <p class="text-[11px] font-semibold uppercase tracking-widest text-blue-100">Smile Clean</p>
              <h1 class="text-lg font-semibold leading-tight">{{ T("title") }}</h1>
            </div>
          </div>
          <div class="text-right">
            <p class="text-xl font-bold tracking-tight">{{ booking?.bookingNumber ?? `#${checklist.bookingId}` }}</p>
            <p class="text-[11px] text-blue-100">{{ T("generated") }}: {{ when(new Date().toISOString()) }}</p>
          </div>
        </div>
      </header>

      <div class="space-y-5 px-8 py-6">
        <!-- Booking details -->
        <section v-if="booking" class="grid grid-cols-2 gap-x-8 gap-y-2.5 rounded-2xl bg-gray-50 p-4">
          <div class="flex items-start gap-2.5">
            <AppIcon name="user" :size="16" class="mt-0.5 text-navy-500" />
            <div><p class="text-[10.5px] font-semibold uppercase tracking-wide text-gray-400">{{ T("customer") }}</p><p class="font-semibold">{{ booking.customerName }}</p></div>
          </div>
          <div class="flex items-start gap-2.5">
            <AppIcon name="calendar" :size="16" class="mt-0.5 text-navy-500" />
            <div><p class="text-[10.5px] font-semibold uppercase tracking-wide text-gray-400">{{ T("date") }}</p><p class="font-semibold">{{ when(booking.scheduledFor) }}</p></div>
          </div>
          <div v-if="booking.address" class="col-span-2 flex items-start gap-2.5">
            <AppIcon name="pin" :size="16" class="mt-0.5 text-navy-500" />
            <div><p class="text-[10.5px] font-semibold uppercase tracking-wide text-gray-400">{{ T("address") }}</p><p>{{ booking.address }}</p></div>
          </div>
          <div class="flex items-start gap-2.5">
            <AppIcon name="stack" :size="16" class="mt-0.5 text-navy-500" />
            <div><p class="text-[10.5px] font-semibold uppercase tracking-wide text-gray-400">{{ T("service") }}</p><p>{{ serviceLabel(booking.serviceType) }}</p></div>
          </div>
          <div v-if="booking.cleaners?.length" class="flex items-start gap-2.5">
            <AppIcon name="user" :size="16" class="mt-0.5 text-navy-500" />
            <div><p class="text-[10.5px] font-semibold uppercase tracking-wide text-gray-400">{{ T("cleaners") }}</p><p>{{ booking.cleaners.map((c) => c.name).join(", ") }}</p></div>
          </div>
        </section>

        <!-- Stat tiles -->
        <section class="grid grid-cols-4 gap-3 break-inside-avoid">
          <div class="rounded-2xl p-3.5 ring-1 ring-gray-200">
            <p class="text-[10.5px] font-semibold uppercase tracking-wide text-gray-400">{{ T("done") }}</p>
            <p class="mt-1 text-xl font-bold">{{ doneCount }}<span class="text-sm text-gray-400"> / {{ checklist.items.length }}</span></p>
            <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-gray-100"><div class="h-full rounded-full bg-gradient-to-r from-navy-500 to-emerald-500" :style="{ width: `${pct}%` }"></div></div>
          </div>
          <div class="rounded-2xl p-3.5 ring-1 ring-gray-200">
            <p class="text-[10.5px] font-semibold uppercase tracking-wide text-gray-400">{{ T("avgScore") }}</p>
            <p class="mt-1 text-xl font-bold">{{ checklist.avgScore != null ? checklist.avgScore.toFixed(1) : "—" }}<span v-if="checklist.avgScore != null" class="text-sm text-gray-400"> / 5</span></p>
            <div class="mt-1.5 flex gap-0.5">
              <AppIcon v-for="n in 5" :key="n" name="star" :size="13" :class="(checklist.avgScore ?? 0) >= n - 0.5 ? 'text-amber-400' : 'text-gray-200'" />
            </div>
          </div>
          <div class="rounded-2xl p-3.5 ring-1" :class="failCount ? 'bg-red-50 ring-red-200' : 'ring-gray-200'">
            <p class="text-[10.5px] font-semibold uppercase tracking-wide text-gray-400">{{ T("fails") }}</p>
            <p :class="['mt-1 text-xl font-bold', failCount ? 'text-red-700' : '']">{{ failCount }}</p>
          </div>
          <div class="rounded-2xl p-3.5 ring-1 ring-gray-200">
            <p class="text-[10.5px] font-semibold uppercase tracking-wide text-gray-400">{{ T("state") }}</p>
            <span v-if="state" :class="['mt-1.5 inline-block rounded-full px-2.5 py-1 text-[11px] font-bold', stateTone[state]]">{{ T(stateKey[state]) }}</span>
          </div>
        </section>

        <p v-if="checklist.submittedBy || checklist.reviewedBy" class="flex flex-wrap gap-x-5 gap-y-1 text-[11px] text-gray-500">
          <span v-if="checklist.submittedBy" class="inline-flex items-center gap-1.5"><AppIcon name="send" :size="13" /> {{ T("submitted") }}: <strong class="text-gray-700">{{ checklist.submittedBy }}</strong> · {{ when(checklist.submittedAt) }}</span>
          <span v-if="checklist.reviewedBy" class="inline-flex items-center gap-1.5"><AppIcon name="badge" :size="13" /> {{ T("reviewed") }}: <strong class="text-gray-700">{{ checklist.reviewedBy }}</strong> · {{ when(checklist.reviewedAt) }}</span>
        </p>
        <div v-if="checklist.reviewNote" class="rounded-2xl border-l-4 border-amber-400 bg-amber-50 px-4 py-2.5 text-[12px]">
          <span class="font-semibold">{{ T("reviewNote") }}:</span> {{ checklist.reviewNote }}
        </div>

        <!-- Items -->
        <table class="w-full border-separate border-spacing-0 text-left">
          <thead>
            <tr class="text-[10.5px] font-semibold uppercase tracking-wide text-gray-400">
              <th class="w-8 border-b-2 border-gray-200 pb-2">#</th>
              <th class="border-b-2 border-gray-200 pb-2 pr-3">{{ T("item") }}</th>
              <th class="w-32 border-b-2 border-gray-200 pb-2 pr-3">{{ T("result") }}</th>
              <th class="border-b-2 border-gray-200 pb-2 pr-3">{{ T("notes") }}</th>
              <th class="w-32 border-b-2 border-gray-200 pb-2">{{ T("by") }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(it, i) in checklist.items" :key="it.id" :class="['break-inside-avoid align-top', i % 2 ? 'bg-gray-50/60' : '']">
              <td class="border-b border-gray-100 py-2.5 pl-1 text-gray-400">{{ i + 1 }}</td>
              <td class="border-b border-gray-100 py-2.5 pr-3 font-medium">{{ it.label }}</td>
              <td class="border-b border-gray-100 py-2.5 pr-3">
                <span v-if="it.itemType === 'score' && it.score" class="inline-flex items-center gap-0.5">
                  <AppIcon v-for="n in 5" :key="n" name="star" :size="13" :class="it.score >= n ? 'text-amber-400' : 'text-gray-200'" />
                  <span class="ml-1 text-[11px] font-bold text-gray-700">{{ it.score }}/5</span>
                </span>
                <span v-else :class="['inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-bold', pill(it).tone]">
                  <AppIcon :name="pill(it).icon" :size="12" :stroke="2.75" /> {{ pill(it).text }}
                </span>
              </td>
              <td class="whitespace-pre-line border-b border-gray-100 py-2.5 pr-3 text-gray-600">{{ it.notes }}</td>
              <td class="border-b border-gray-100 py-2.5 text-[11px] text-gray-500">
                <template v-if="it.isCompleted"><span class="font-medium text-gray-700">{{ it.completedBy }}</span><br v-if="it.completedBy" />{{ when(it.completedAt) }}</template>
              </td>
            </tr>
          </tbody>
        </table>

        <!-- Photos -->
        <section v-if="photoItems.length">
          <h2 class="mb-3 flex items-center gap-2 text-sm font-semibold"><AppIcon name="photo" :size="17" class="text-navy-500" /> {{ T("photos") }}</h2>
          <div class="grid grid-cols-2 gap-3">
            <figure v-for="it in photoItems" :key="it.id" class="break-inside-avoid rounded-2xl p-3 ring-1 ring-gray-200">
              <figcaption class="mb-2 text-[11.5px] font-semibold">{{ it.label }}</figcaption>
              <div class="grid grid-cols-2 gap-2">
                <div v-for="kind in (['before', 'after'] as const)" :key="kind">
                  <div class="relative aspect-[4/3] overflow-hidden rounded-xl bg-gray-100">
                    <img
                      v-if="photos[(kind === 'before' ? it.beforePhotoUrl : it.afterPhotoUrl) ?? '']"
                      :src="photos[(kind === 'before' ? it.beforePhotoUrl : it.afterPhotoUrl)!]"
                      :alt="`${it.label} — ${T(kind)}`"
                      class="h-full w-full object-cover"
                    />
                    <span v-else class="flex h-full items-center justify-center text-gray-300"><AppIcon name="photo" :size="22" /></span>
                    <span :class="['absolute left-1.5 top-1.5 rounded-full px-2 py-0.5 text-[9.5px] font-bold uppercase', kind === 'after' ? 'bg-emerald-600 text-white' : 'bg-white/90 text-gray-700']">{{ T(kind) }}</span>
                  </div>
                </div>
              </div>
            </figure>
          </div>
        </section>

        <!-- Client sign-off -->
        <section class="grid grid-cols-2 gap-6 break-inside-avoid rounded-2xl p-4 ring-1 ring-gray-200">
          <div>
            <h2 class="flex items-center gap-2 text-sm font-semibold"><AppIcon name="pencil" :size="16" class="text-navy-500" /> {{ T("client") }}</h2>
            <p v-if="checklist.clientSignature" class="mt-2 text-[12px] text-gray-600">{{ T("signedBy") }}: <strong class="text-gray-900">{{ checklist.clientSignature }}</strong><br />{{ when(checklist.clientConfirmedAt) }}</p>
            <p v-else class="mt-2 text-[12px] text-gray-400">{{ T("notSigned") }}</p>
          </div>
          <div class="flex flex-col justify-end">
            <p class="min-h-[28px] text-center font-serif text-lg italic text-gray-800">{{ checklist.clientSignature }}</p>
            <div class="border-t border-gray-300 pt-1 text-center text-[10.5px] uppercase tracking-wide text-gray-400">{{ T("signedBy") }}</div>
          </div>
        </section>
      </div>

      <footer class="border-t border-gray-100 px-8 py-3 text-center text-[10.5px] text-gray-400">
        Smile Clean · {{ T("title") }} · {{ booking?.bookingNumber ?? `#${checklist.bookingId}` }}
      </footer>
    </article>
  </div>
</template>
