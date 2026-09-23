<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { serviceLabel } from "../../../lib/settings";
import {
  clearPortalCustomer,
  getPortalCustomer,
  portalBookings,
  portalCancelBooking,
  portalChangePassword,
  portalCreateBooking,
  portalLogin,
  portalLogout,
  portalMe,
  portalServices,
  portalSites,
  portalSubmitFeedback,
  type PortalBooking,
  type PortalCustomer,
  type PortalService,
  type PortalSite,
} from "../../../lib/portal";

type Lang = "th" | "en";

const LANG_KEY = "odysight_portal_lang";
const lang = ref<Lang>("th");

function setLang(l: Lang) {
  lang.value = l;
  try {
    localStorage.setItem(LANG_KEY, l);
  } catch { /* ignore */ }
}

try {
  const saved = typeof window !== "undefined" ? localStorage.getItem(LANG_KEY) : null;
  if (saved === "en" || saved === "th") lang.value = saved;
} catch { /* ignore */ }

const STR = {
  th: {
    brandSub: "สไมล์คลีน",
    portal: "พอร์ทัลลูกค้า",
    signOut: "ออกจากระบบ",
    backHome: "← กลับสู่ Smile Clean",
    tagline: "ทำความสะอาดมืออาชีพ",
    heroTitleA: "พื้นที่สะอาดกว่าเดิม,",
    heroTitleB: "จองได้ในหนึ่งนาที",
    b1: "จองบริการบ้าน คอนโด และสำนักงานออนไลน์",
    b2: "ติดตามสถานะแบบเรียลไทม์ ตั้งแต่รอยืนยันถึงเสร็จงาน",
    b3: "ให้คะแนนบริการและจัดการสถานที่ของคุณ",
    trusted: "ได้รับความไว้วางใจจากบ้านและธุรกิจทั่วกรุงเทพฯ",
    welcomeBack: "ยินดีต้อนรับกลับ",
    signInDesc: "เข้าสู่ระบบเพื่อจัดการงานทำความสะอาดและนัดหมายของคุณ",
    email: "อีเมล",
    password: "รหัสผ่าน",
    emailPh: "you@example.com",
    passwordPh: "กรอกรหัสผ่าน",
    signIn: "เข้าสู่ระบบ",
    signingIn: "กำลังเข้าสู่ระบบ…",
    noAccount: "ยังไม่มีบัญชี? ติดต่อสำนักงานเพื่อให้เราเปิดสิทธิ์พอร์ทัลให้คุณ",
    goodDay: "สวัสดี",
    newBookingBtn: "จองใหม่",
    upcoming: "งานที่กำลังจะมาถึง",
    upcomingSub: "งานที่รอและยืนยันแล้ว",
    completedL: "เสร็จสิ้น",
    completedSub: "งานที่ทำเสร็จแล้ว",
    nextVisit: "นัดครั้งถัดไป",
    nothingScheduled: "ยังไม่มีนัด",
    bookNextHint: "จองงานทำความสะอาดครั้งถัดไปด้านบน",
    tabBookings: "การจองของฉัน",
    tabBook: "จองบริการ",
    tabProfile: "โปรไฟล์และความปลอดภัย",
    noBookings: "ยังไม่มีการจอง",
    noBookingsSub: "เมื่อคุณจองบริการทำความสะอาด จะแสดงที่นี่พร้อมสถานะแบบเรียลไทม์",
    bookFirst: "จองบริการครั้งแรก",
    serviceAddress: "ที่อยู่หน้างาน",
    cleaner: "พนักงาน",
    cleanerUnassigned: "รอสำนักงานมอบหมาย",
    minUnit: "นาที",
    leaveReview: "★ เขียนรีวิว",
    cancelBookingBtn: "ยกเลิกการจอง",
    cancelling: "กำลังยกเลิก…",
    requestTitle: "ขอจองบริการทำความสะอาด",
    requestSub: "เลือกบริการ สถานที่ และเวลา — สำนักงานจะยืนยันและมอบหมายพนักงานให้",
    serviceL: "บริการ",
    selectService: "เลือกบริการ",
    locationL: "สถานที่",
    useDefaultAddress: "ใช้ที่อยู่เริ่มต้นของฉัน",
    prefDate: "วันที่ต้องการ",
    prefTime: "เวลาที่ต้องการ",
    notesL: "หมายเหตุ",
    optional: "(ไม่บังคับ)",
    notesPh: "รหัสประตู สัตว์เลี้ยง ที่จอดรถ สิ่งที่อยากเน้น…",
    requestBtn: "ส่งคำขอจอง",
    sendingBtn: "กำลังส่งคำขอ…",
    yourSelection: "รายการที่เลือก",
    durationL: "ระยะเวลา",
    locationRow: "สถานที่",
    scheduleL: "กำหนดการ",
    defaultAddr: "ที่อยู่เริ่มต้น",
    freeCancel: "ยกเลิกฟรีก่อนถึงเวลานัด 24 ชม. ทีมงานยืนยันทุกคำขอทางโทรศัพท์หรือข้อความ",
    howItWorks: "ขั้นตอนการจอง",
    step1: "ขอเวลาที่สะดวกสำหรับคุณ",
    step2: "เรายืนยันและมอบหมายพนักงาน",
    step3: "ผ่อนคลายได้เลย — แล้วให้คะแนนบริการ",
    phone: "โทรศัพท์",
    addressL: "ที่อยู่",
    areaL: "พื้นที่/โซน",
    myLocations: "สถานที่ของฉัน",
    defaultBadge: "ค่าเริ่มต้น",
    security: "ความปลอดภัย",
    securitySub: "ตั้งรหัสผ่านที่แข็งแรงเพื่อปกป้องบัญชีพอร์ทัลของคุณ",
    currentPw: "รหัสผ่านปัจจุบัน",
    newPw: "รหัสผ่านใหม่ (อย่างน้อย 8 ตัวอักษร)",
    updatePw: "อัปเดตรหัสผ่าน",
    pwUpdated: "อัปเดตรหัสผ่านเรียบร้อยแล้ว",
    reviewTitle: "รีวิว",
    howWas: "บริการของคุณเป็นอย่างไรบ้าง?",
    yourRating: "คะแนนของคุณ",
    commentL: "ความคิดเห็น",
    commentPh: "เล่าให้เราฟังหน่อยว่าเป็นอย่างไร…",
    submitReview: "ส่งรีวิว",
    submitting: "กำลังส่ง…",
    cancel: "ยกเลิก",
    errChooseService: "กรุณาเลือกบริการ",
    errChooseDate: "กรุณาเลือกวันที่",
    errInvalidDate: "วัน/เวลาไม่ถูกต้อง",
    errSignIn: "เข้าสู่ระบบไม่สำเร็จ",
    errBooking: "ส่งคำขอจองไม่สำเร็จ",
    errCancel: "ยกเลิกการจองไม่สำเร็จ",
    errPassword: "เปลี่ยนรหัสผ่านไม่สำเร็จ",
    errFeedback: "ส่งรีวิวไม่สำเร็จ",
  },
  en: {
    brandSub: "Smile Clean",
    portal: "Customer Portal",
    signOut: "Sign out",
    backHome: "← Back to Smile Clean",
    tagline: "Professional cleaning",
    heroTitleA: "A cleaner space,",
    heroTitleB: "booked in a minute.",
    b1: "Book home, condo & office cleaning online",
    b2: "Live status from pending to completed",
    b3: "Rate your service and manage locations",
    trusted: "Trusted by homes & businesses across Bangkok.",
    welcomeBack: "Welcome back",
    signInDesc: "Sign in to manage your cleanings and appointments.",
    email: "Email address",
    password: "Password",
    emailPh: "you@example.com",
    passwordPh: "Enter your password",
    signIn: "Sign in to portal",
    signingIn: "Signing in…",
    noAccount: "No account yet? Contact our office and we'll enable portal access for you.",
    goodDay: "Good day,",
    newBookingBtn: "New booking",
    upcoming: "Upcoming",
    upcomingSub: "Pending & confirmed visits",
    completedL: "Completed",
    completedSub: "Finished cleanings",
    nextVisit: "Next visit",
    nothingScheduled: "Nothing scheduled",
    bookNextHint: "Book your next cleaning above",
    tabBookings: "My bookings",
    tabBook: "New booking",
    tabProfile: "Profile & security",
    noBookings: "No bookings yet",
    noBookingsSub: "When you request a cleaning it will appear here with live status updates.",
    bookFirst: "Book your first cleaning",
    serviceAddress: "Service address",
    cleaner: "Cleaner",
    cleanerUnassigned: "To be assigned by our office",
    minUnit: "min",
    leaveReview: "★ Leave a review",
    cancelBookingBtn: "Cancel booking",
    cancelling: "Cancelling…",
    requestTitle: "Request a cleaning",
    requestSub: "Choose a service and time — our office confirms and assigns your cleaner.",
    serviceL: "Service",
    selectService: "Select a service",
    locationL: "Service location",
    useDefaultAddress: "Use my default address",
    prefDate: "Preferred date",
    prefTime: "Preferred time",
    notesL: "Notes",
    optional: "(optional)",
    notesPh: "Gate code, pets, parking, priorities…",
    requestBtn: "Request booking",
    sendingBtn: "Sending request…",
    yourSelection: "Your selection",
    durationL: "Duration",
    locationRow: "Location",
    scheduleL: "Schedule",
    defaultAddr: "Default address",
    freeCancel: "Free cancellation up to 24h before your visit. Our team confirms every request by phone or message.",
    howItWorks: "How it works",
    step1: "Request a time that suits you.",
    step2: "We confirm and assign your cleaner.",
    step3: "Relax — then rate your service.",
    phone: "Phone",
    addressL: "Address",
    areaL: "Area",
    myLocations: "My locations",
    defaultBadge: "DEFAULT",
    security: "Security",
    securitySub: "Keep your portal account protected with a strong password.",
    currentPw: "Current password",
    newPw: "New password (min 8 characters)",
    updatePw: "Update password",
    pwUpdated: "Password updated successfully.",
    reviewTitle: "Review",
    howWas: "How was your service?",
    yourRating: "Your rating",
    commentL: "Comment",
    commentPh: "Tell us how it went…",
    submitReview: "Submit review",
    submitting: "Submitting…",
    cancel: "Cancel",
    errChooseService: "Choose a service",
    errChooseDate: "Choose a date",
    errInvalidDate: "Invalid date/time",
    errSignIn: "Sign in failed",
    errBooking: "Failed to create booking",
    errCancel: "Failed to cancel booking",
    errPassword: "Failed to change password",
    errFeedback: "Failed to submit feedback",
  },
} as const;

const t = computed(() => STR[lang.value]);

const SERVICE_TH: Record<string, string> = {
  house_cleaning: "ทำความสะอาดบ้าน",
  condo_cleaning: "ทำความสะอาดคอนโด",
  deep_cleaning: "ทำความสะอาดแบบล้ำลึก (Deep Cleaning)",
  move_in_out: "ทำความสะอาดย้ายเข้า/ออก",
  after_renovation: "ทำความสะอาดหลังรีโนเวท",
  office_cleaning: "ทำความสะอาดสำนักงาน",
  junk_removal: "ขนย้ายขยะ/ของไม่ใช้แล้ว",
  aircon_service: "ล้างแอร์",
  sofa_cleaning: "ทำความสะอาดโซฟา",
  matress_cleaning: "ทำความสะอาดที่นอน",
};

function serviceName(id: string): string {
  if (lang.value === "th") return SERVICE_TH[id] ?? serviceLabel(id);
  return serviceLabel(id);
}

const STATUS_TH: Record<string, string> = {
  pending: "รอยืนยัน",
  confirmed: "ยืนยันแล้ว",
  in_progress: "กำลังดำเนินการ",
  completed: "เสร็จสิ้น",
  cancelled: "ยกเลิกแล้ว",
  no_show: "ไม่มาตามนัด",
  rescheduled: "เลื่อนนัด",
};

function statusLabel(s: string): string {
  if (lang.value === "th") return STATUS_TH[s] ?? s;
  return s.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

function msgReceived(bookingNumber: string): string {
  return lang.value === "th"
    ? `ได้รับคำขอแล้ว — ${bookingNumber} รอการยืนยันจากสำนักงาน`
    : `Request received — ${bookingNumber} is pending confirmation.`;
}

const customer = ref<PortalCustomer | null>(null);
const bookings = ref<PortalBooking[]>([]);
const sites = ref<PortalSite[]>([]);
const services = ref<PortalService[]>([]);
const loading = ref(true);
const busy = ref(false);
const error = ref("");
const activeTab = ref<"bookings" | "book" | "profile">("bookings");
const isDev = import.meta.env.DEV;

const email = ref("");
const password = ref("");

const currentPw = ref("");
const newPw = ref("");
const pwMessage = ref("");

const feedbackFor = ref<PortalBooking | null>(null);
const feedbackRating = ref(5);
const feedbackComment = ref("");
const feedbackSaving = ref(false);
const feedbackError = ref("");

// New-booking form state
const bookService = ref("");
const bookSite = ref<number | "">("");
const bookDate = ref("");
const bookTime = ref("09:00");
const bookSaving = ref(false);
const bookMessage = ref("");
const cancellingId = ref<number | null>(null);

const statusStyles: Record<string, string> = {
  pending: "bg-amber-50 text-amber-800 ring-amber-200",
  confirmed: "bg-blue-50 text-blue-800 ring-blue-200",
  in_progress: "bg-indigo-50 text-indigo-800 ring-indigo-200",
  completed: "bg-emerald-50 text-emerald-800 ring-emerald-200",
  cancelled: "bg-rose-50 text-rose-700 ring-rose-200",
  no_show: "bg-gray-100 text-gray-600 ring-gray-200",
  rescheduled: "bg-purple-50 text-purple-800 ring-purple-200",
};

const statusDot: Record<string, string> = {
  pending: "bg-amber-500",
  confirmed: "bg-blue-500",
  in_progress: "bg-indigo-500",
  completed: "bg-emerald-500",
  cancelled: "bg-rose-400",
  no_show: "bg-gray-400",
  rescheduled: "bg-purple-500",
};

const init = async () => {
  const stored = getPortalCustomer();
  if (!stored) {
    loading.value = false;
    return;
  }
  try {
    customer.value = await portalMe();
    bookings.value = await portalBookings();
    try {
      const [s, sv] = await Promise.all([portalSites(), portalServices()]);
      sites.value = s;
      services.value = sv;
      const def = s.find((x) => x.isDefault);
      if (def) bookSite.value = def.id;
      const first = sv[0];
      if (first) bookService.value = first.id;
    } catch {
      /* booking extras are optional: list still works */
    }
  } catch {
    clearPortalCustomer();
    customer.value = null;
  } finally {
    loading.value = false;
  }
};

onMounted(init);

async function loadCatalog() {
  try {
    const [s, sv] = await Promise.all([portalSites(), portalServices()]);
    sites.value = s;
    services.value = sv;
    const def = s.find((x) => x.isDefault);
    if (def && bookSite.value === "") bookSite.value = def.id;
    const first = sv[0];
    if (first && !bookService.value) bookService.value = first.id;
  } catch {
    /* ignore */
  }
}

async function submitLogin() {
  error.value = "";
  busy.value = true;
  try {
    customer.value = await portalLogin(email.value, password.value);
    bookings.value = await portalBookings();
    await loadCatalog();
    try {
      localStorage.setItem("odysight_portal_customer", JSON.stringify(customer.value));
    } catch { /* ignore */ }
    password.value = "";
    activeTab.value = "bookings";
  } catch (e) {
    error.value = e instanceof Error ? e.message : t.value.errSignIn;
  } finally {
    busy.value = false;
  }
}

async function logout() {
  await portalLogout().catch(() => { /* ignore */ });
  clearPortalCustomer();
  customer.value = null;
  bookings.value = [];
  sites.value = [];
  services.value = [];
}

async function changePassword() {
  pwMessage.value = "";
  error.value = "";
  try {
    await portalChangePassword(currentPw.value, newPw.value);
    pwMessage.value = t.value.pwUpdated;
    currentPw.value = "";
    newPw.value = "";
  } catch (e) {
    pwMessage.value = "";
    error.value = e instanceof Error ? e.message : t.value.errPassword;
  }
}

const upcomingCount = computed(() =>
  bookings.value.filter((b) => b.status === "pending" || b.status === "confirmed" || b.status === "in_progress").length,
);
const completedCount = computed(() =>
  bookings.value.filter((b) => b.status === "completed").length,
);
const nextBooking = computed(() => {
  const now = Date.now();
  const upcoming = bookings.value
    .filter((b) => new Date(b.scheduledFor).getTime() >= now - 1000 * 60 * 60 * 24 && b.status !== "cancelled")
    .sort((a, b) => +new Date(a.scheduledFor) - +new Date(b.scheduledFor));
  return upcoming[0] ?? null;
});
const selectedService = computed(() => services.value.find((s) => s.id === bookService.value) ?? null);
const selectedSite = computed(() => sites.value.find((s) => s.id === Number(bookSite.value)) ?? null);

function initials(name: string): string {
  return name
    .split(" ")
    .filter(Boolean)
    .slice(0, 2)
    .map((p) => p[0]?.toUpperCase())
    .join("");
}

function dateLocale(): string {
  // th-TH defaults to the Buddhist calendar (year 2569) — pin Gregorian
  // so dates match the office/staff side.
  return lang.value === "th" ? "th-TH-u-ca-gregory" : "en-US";
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString(dateLocale(), {
    weekday: "short",
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function dateBlock(iso: string): { day: string; mon: string } {
  const d = new Date(iso);
  return {
    day: d.toLocaleString(dateLocale(), { day: "2-digit" }),
    mon: d.toLocaleString(dateLocale(), { month: "short" }).toUpperCase(),
  };
}

function openFeedback(b: PortalBooking) {
  feedbackFor.value = b;
  feedbackRating.value = 5;
  feedbackComment.value = "";
  feedbackError.value = "";
}

async function submitFeedback() {
  if (!feedbackFor.value) return;
  feedbackSaving.value = true;
  feedbackError.value = "";
  try {
    await portalSubmitFeedback({
      bookingId: feedbackFor.value.id,
      rating: feedbackRating.value,
      comment: feedbackComment.value,
    });
    feedbackFor.value = null;
  } catch (e) {
    feedbackError.value = e instanceof Error ? e.message : t.value.errFeedback;
  } finally {
    feedbackSaving.value = false;
  }
}

const bookNotes = ref("");

async function submitBooking() {
  error.value = "";
  bookMessage.value = "";
  if (!bookService.value) {
    error.value = t.value.errChooseService;
    return;
  }
  if (!bookDate.value) {
    error.value = t.value.errChooseDate;
    return;
  }
  const iso = new Date(`${bookDate.value}T${bookTime.value || "09:00"}:00`).toISOString();
  if (Number.isNaN(Date.parse(iso))) {
    error.value = t.value.errInvalidDate;
    return;
  }
  bookSaving.value = true;
  try {
    const created = await portalCreateBooking({
      serviceType: bookService.value,
      scheduledFor: iso,
      siteId: bookSite.value === "" ? null : Number(bookSite.value),
      notes: bookNotes.value,
    });
    bookings.value = [created, ...bookings.value];
    bookMessage.value = msgReceived(created.bookingNumber);
    bookNotes.value = "";
    activeTab.value = "bookings";
  } catch (e) {
    error.value = e instanceof Error ? e.message : t.value.errBooking;
  } finally {
    bookSaving.value = false;
  }
}

async function cancelBooking(id: number) {
  cancellingId.value = id;
  try {
    await portalCancelBooking(id);
    bookings.value = bookings.value.map((b) =>
      b.id === id ? { ...b, status: "cancelled" } : b,
    );
  } catch (e) {
    error.value = e instanceof Error ? e.message : t.value.errCancel;
  } finally {
    cancellingId.value = null;
  }
}

const inputClass =
  "w-full rounded-xl border border-slate-200 bg-white px-3.5 py-2.5 text-sm text-slate-900 shadow-sm placeholder:text-slate-400 focus:border-navy-500 focus:outline-none focus:ring-2 focus:ring-navy-500/20";
const labelClass = "mb-1.5 block text-[13px] font-semibold text-slate-700";
</script>

<template>
  <div class="min-h-screen w-full bg-gradient-to-b from-navy-950 via-navy-900 to-slate-100">
    <!-- Top brand bar -->
    <header class="border-b border-white/10">
      <div class="mx-auto flex w-full max-w-5xl items-center justify-between gap-3 px-4 py-4 sm:px-6">
        <div class="flex items-center gap-3">
          <img
            src="/logo/logo.png"
            alt="Smile Clean logo"
            class="h-11 w-11 rounded-xl bg-white object-contain shadow-md ring-1 ring-white/40"
          />
          <div>
            <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-navy-200">{{ t.brandSub }}</p>
            <h1 class="text-lg font-semibold leading-tight text-white">{{ t.portal }}</h1>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <div class="flex rounded-xl bg-white/10 p-0.5 text-[12px] font-bold ring-1 ring-white/20" role="group" aria-label="Language">
            <button type="button" class="rounded-[10px] px-2.5 py-1.5 transition" :class="lang === 'th' ? 'bg-white text-navy-900 shadow' : 'text-white/70 hover:text-white'" @click="setLang('th')">ไทย</button>
            <button type="button" class="rounded-[10px] px-2.5 py-1.5 transition" :class="lang === 'en' ? 'bg-white text-navy-900 shadow' : 'text-white/70 hover:text-white'" @click="setLang('en')">EN</button>
          </div>
          <div v-if="customer" class="flex items-center gap-3">
            <div class="hidden text-right sm:block">
              <p class="text-sm font-semibold text-white">{{ customer.name }}</p>
              <p class="text-xs text-navy-200">{{ customer.email }}</p>
            </div>
            <span class="flex h-10 w-10 items-center justify-center rounded-full bg-white/15 text-sm font-bold text-white ring-1 ring-white/30">
              {{ initials(customer.name) }}
            </span>
            <button
              type="button"
              class="rounded-xl border border-white/20 px-3 py-2 text-[13px] font-semibold text-white/90 transition hover:bg-white/10"
              @click="logout"
            >
              {{ t.signOut }}
            </button>
          </div>
          <a v-else href="/" class="hidden rounded-xl border border-white/20 px-3 py-2 text-[13px] font-semibold text-white/90 transition hover:bg-white/10 sm:block">
            {{ t.backHome }}
          </a>
        </div>
      </div>
    </header>

    <main class="mx-auto w-full max-w-5xl px-4 pb-16 sm:px-6">
      <!-- Loading -->
      <div v-if="loading" class="mt-8 rounded-2xl bg-white p-8 shadow-xl ring-1 ring-slate-200">
        <div class="h-6 w-1/3 animate-pulse rounded bg-slate-100" />
        <div class="mt-4 h-24 animate-pulse rounded-xl bg-slate-100" />
        <div class="mt-3 h-24 animate-pulse rounded-xl bg-slate-100" />
      </div>

      <!-- Login -->
      <div v-else-if="!customer" class="mx-auto mt-10 grid max-w-4xl overflow-hidden rounded-2xl bg-white shadow-2xl ring-1 ring-slate-200 md:grid-cols-2">
        <div class="relative hidden flex-col justify-between overflow-hidden bg-navy-950 p-8 text-white md:flex">
          <div class="pointer-events-none absolute -right-20 -top-20 h-64 w-64 rounded-full bg-navy-600/40 blur-3xl" />
          <div class="pointer-events-none absolute -bottom-24 -left-16 h-72 w-72 rounded-full bg-blue-400/20 blur-3xl" />
          <div class="relative">
            <p class="text-[11px] font-semibold uppercase tracking-[0.2em] text-navy-200">{{ t.tagline }}</p>
            <h2 class="mt-2 text-2xl font-bold leading-snug">{{ t.heroTitleA }}<br />{{ t.heroTitleB }}</h2>
            <ul class="mt-6 space-y-3 text-sm text-navy-100">
              <li class="flex items-start gap-2.5"><span class="mt-0.5 flex h-5 w-5 items-center justify-center rounded-full bg-emerald-400/20 text-xs text-emerald-300">✓</span> {{ t.b1 }}</li>
              <li class="flex items-start gap-2.5"><span class="mt-0.5 flex h-5 w-5 items-center justify-center rounded-full bg-emerald-400/20 text-xs text-emerald-300">✓</span> {{ t.b2 }}</li>
              <li class="flex items-start gap-2.5"><span class="mt-0.5 flex h-5 w-5 items-center justify-center rounded-full bg-emerald-400/20 text-xs text-emerald-300">✓</span> {{ t.b3 }}</li>
            </ul>
          </div>
          <p class="relative text-xs text-navy-300">{{ t.trusted }}</p>
        </div>
        <form class="p-8" @submit.prevent="submitLogin">
          <h2 class="text-xl font-bold text-slate-900">{{ t.welcomeBack }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ t.signInDesc }}</p>
          <div class="mt-6 space-y-4">
            <div>
              <label :class="labelClass" for="portal-email">{{ t.email }}</label>
              <input
                id="portal-email"
                v-model="email"
                type="email"
                required
                autocomplete="email"
                :placeholder="t.emailPh"
                :class="inputClass"
              />
            </div>
            <div>
              <label :class="labelClass" for="portal-password">{{ t.password }}</label>
              <input
                id="portal-password"
                v-model="password"
                type="password"
                required
                autocomplete="current-password"
                :placeholder="t.passwordPh"
                :class="inputClass"
              />
            </div>
            <p v-if="error" class="rounded-xl border border-rose-200 bg-rose-50 px-3.5 py-2.5 text-sm text-rose-700">{{ error }}</p>
            <button
              type="submit"
              :disabled="busy"
              class="w-full rounded-xl bg-navy-600 py-3 text-sm font-bold text-white shadow-lg shadow-navy-600/25 transition hover:bg-navy-700 disabled:opacity-50"
            >
              {{ busy ? t.signingIn : t.signIn }}
            </button>
            <p class="text-center text-xs text-slate-500">{{ t.noAccount }}</p>
          </div>
          <p v-if="isDev" class="mt-4 rounded-xl bg-slate-50 px-3 py-2 text-center text-xs text-slate-400 ring-1 ring-slate-200">
            Demo: somchai@smileclean.com / portal123
          </p>
        </form>
      </div>

      <!-- Dashboard -->
      <div v-else class="mt-6 space-y-5">
        <!-- Welcome + stats -->
        <section class="overflow-hidden rounded-2xl bg-white shadow-xl ring-1 ring-slate-200">
          <div class="flex flex-col gap-4 bg-gradient-to-r from-navy-950 to-navy-800 px-6 py-6 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-navy-300">{{ t.goodDay }}</p>
              <h2 class="mt-0.5 text-2xl font-bold text-white">{{ customer.name }}</h2>
              <p class="mt-1 text-sm text-navy-200">{{ customer.address }} · {{ customer.phone }}</p>
            </div>
            <button
              type="button"
              class="inline-flex items-center justify-center gap-2 self-start rounded-xl bg-white px-5 py-2.5 text-sm font-bold text-navy-800 shadow transition hover:bg-navy-50 sm:self-center"
              @click="activeTab = 'book'"
            >
              <span class="text-base leading-none">+</span> {{ t.newBookingBtn }}
            </button>
          </div>
          <div class="grid grid-cols-1 divide-y divide-slate-100 sm:grid-cols-3 sm:divide-x sm:divide-y-0">
            <div class="px-6 py-4">
              <p class="text-[11px] font-semibold uppercase tracking-wider text-slate-400">{{ t.upcoming }}</p>
              <p class="mt-1 text-2xl font-bold text-slate-900">{{ upcomingCount }}</p>
              <p class="text-xs text-slate-500">{{ t.upcomingSub }}</p>
            </div>
            <div class="px-6 py-4">
              <p class="text-[11px] font-semibold uppercase tracking-wider text-slate-400">{{ t.completedL }}</p>
              <p class="mt-1 text-2xl font-bold text-slate-900">{{ completedCount }}</p>
              <p class="text-xs text-slate-500">{{ t.completedSub }}</p>
            </div>
            <div class="px-6 py-4">
              <p class="text-[11px] font-semibold uppercase tracking-wider text-slate-400">{{ t.nextVisit }}</p>
              <p class="mt-1 truncate text-base font-bold text-navy-700">{{ nextBooking ? formatDate(nextBooking.scheduledFor) : t.nothingScheduled }}</p>
              <p class="truncate text-xs text-slate-500">{{ nextBooking ? `${serviceName(nextBooking.serviceType)} · ${nextBooking.bookingNumber}` : t.bookNextHint }}</p>
            </div>
          </div>
          <nav class="flex gap-1 overflow-x-auto border-t border-slate-100 bg-slate-50/60 px-3 py-2">
            <button
              v-for="tb in [{ id: 'bookings', label: t.tabBookings }, { id: 'book', label: t.tabBook }, { id: 'profile', label: t.tabProfile }]"
              :key="tb.id"
              type="button"
              class="whitespace-nowrap rounded-xl px-4 py-2 text-[13px] font-semibold transition"
              :class="activeTab === tb.id ? 'bg-navy-950 text-white shadow' : 'text-slate-600 hover:bg-white hover:text-slate-900'"
              @click="activeTab = tb.id as typeof activeTab"
            >
              {{ tb.label }}
            </button>
          </nav>
        </section>

        <p v-if="error" class="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700 shadow-sm">{{ error }}</p>

        <!-- Bookings -->
        <section v-if="activeTab === 'bookings'" class="space-y-4">
          <div v-if="bookings.length === 0" class="rounded-2xl bg-white p-12 text-center shadow-xl ring-1 ring-slate-200">
            <div class="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-navy-50 text-2xl">✦</div>
            <p class="mt-4 text-base font-bold text-slate-900">{{ t.noBookings }}</p>
            <p class="mx-auto mt-1 max-w-sm text-sm text-slate-500">{{ t.noBookingsSub }}</p>
            <button type="button" class="mt-5 rounded-xl bg-navy-600 px-5 py-2.5 text-sm font-bold text-white shadow-lg shadow-navy-600/25 hover:bg-navy-700" @click="activeTab = 'book'">
              {{ t.bookFirst }}
            </button>
          </div>

          <article
            v-for="b in bookings"
            :key="b.id"
            class="overflow-hidden rounded-2xl bg-white shadow-lg ring-1 ring-slate-200 transition hover:shadow-xl"
          >
            <div class="flex flex-col gap-4 p-5 sm:flex-row sm:items-start">
              <div class="flex items-center gap-4 sm:w-64 sm:shrink-0">
                <div class="flex h-16 w-16 flex-col items-center justify-center rounded-2xl bg-navy-950 text-white shadow">
                  <span class="text-xl font-bold leading-none">{{ dateBlock(b.scheduledFor).day }}</span>
                  <span class="mt-1 text-[10px] font-bold tracking-widest text-navy-200">{{ dateBlock(b.scheduledFor).mon }}</span>
                </div>
                <div class="min-w-0">
                  <h3 class="truncate text-[15px] font-bold text-slate-900">{{ serviceName(b.serviceType) }}</h3>
                  <p class="text-xs font-medium text-slate-500">{{ b.bookingNumber }}</p>
                  <span :class="['mt-1.5 inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-bold ring-1', statusStyles[b.status] ?? 'bg-gray-100 text-gray-600 ring-gray-200']">
                    <span :class="['h-1.5 w-1.5 rounded-full', statusDot[b.status] ?? 'bg-gray-400']" />
                    {{ statusLabel(b.status) }}
                  </span>
                </div>
              </div>
              <div class="min-w-0 flex-1">
                <p class="text-[13px] font-semibold text-navy-700">{{ formatDate(b.scheduledFor) }} · {{ b.durationMinutes }} {{ t.minUnit }}</p>
                <dl class="mt-3 grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
                  <div class="rounded-xl bg-slate-50 px-3.5 py-2.5 ring-1 ring-slate-100">
                    <dt class="text-[10px] font-bold uppercase tracking-wider text-slate-400">{{ t.serviceAddress }}</dt>
                    <dd class="mt-0.5 text-[13px] font-medium text-slate-700">{{ b.address }}</dd>
                  </div>
                  <div class="rounded-xl bg-slate-50 px-3.5 py-2.5 ring-1 ring-slate-100">
                    <dt class="text-[10px] font-bold uppercase tracking-wider text-slate-400">{{ t.cleaner }}</dt>
                    <dd class="mt-0.5 text-[13px] font-medium text-slate-700">{{ b.assignee || t.cleanerUnassigned }}</dd>
                  </div>
                </dl>
                <p v-if="b.notes" class="mt-2.5 border-l-2 border-navy-200 pl-3 text-[13px] italic text-slate-500">“{{ b.notes }}”</p>
              </div>
            </div>
            <div v-if="b.status === 'completed' || b.status === 'pending' || b.status === 'confirmed'" class="flex flex-wrap items-center justify-end gap-2 border-t border-slate-100 bg-slate-50/70 px-5 py-3">
              <button
                v-if="b.status === 'completed'"
                type="button"
                class="rounded-xl bg-navy-600 px-4 py-2 text-[13px] font-bold text-white shadow hover:bg-navy-700"
                @click="openFeedback(b)"
              >
                {{ t.leaveReview }}
              </button>
              <button
                v-if="b.status === 'pending' || b.status === 'confirmed'"
                type="button"
                :disabled="cancellingId === b.id"
                class="rounded-xl border border-rose-200 bg-white px-4 py-2 text-[13px] font-semibold text-rose-600 transition hover:bg-rose-50 disabled:opacity-50"
                @click="cancelBooking(b.id)"
              >
                {{ cancellingId === b.id ? t.cancelling : t.cancelBookingBtn }}
              </button>
            </div>
          </article>
        </section>

        <!-- New booking -->
        <section v-else-if="activeTab === 'book'" class="grid gap-5 lg:grid-cols-5">
          <form class="rounded-2xl bg-white p-6 shadow-xl ring-1 ring-slate-200 lg:col-span-3" @submit.prevent="submitBooking">
            <h3 class="text-lg font-bold text-slate-900">{{ t.requestTitle }}</h3>
            <p class="mt-1 text-sm text-slate-500">{{ t.requestSub }}</p>
            <div class="mt-5 space-y-4">
              <div>
                <label :class="labelClass" for="book-service">{{ t.serviceL }}</label>
                <select id="book-service" v-model="bookService" required :class="inputClass">
                  <option value="" disabled>{{ t.selectService }}</option>
                  <option v-for="s in services" :key="s.id" :value="s.id">{{ serviceName(s.id) }} · {{ s.durationMinutes }} {{ t.minUnit }}</option>
                </select>
              </div>
              <div v-if="sites.length > 0">
                <label :class="labelClass" for="book-site">{{ t.locationL }}</label>
                <select id="book-site" v-model="bookSite" :class="inputClass">
                  <option value="">{{ t.useDefaultAddress }}</option>
                  <option v-for="s in sites" :key="s.id" :value="s.id">{{ s.name }} — {{ s.address }}</option>
                </select>
              </div>
              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label :class="labelClass" for="book-date">{{ t.prefDate }}</label>
                  <input id="book-date" v-model="bookDate" type="date" required :class="inputClass" />
                </div>
                <div>
                  <label :class="labelClass" for="book-time">{{ t.prefTime }}</label>
                  <input id="book-time" v-model="bookTime" type="time" required :class="inputClass" />
                </div>
              </div>
              <div>
                <label :class="labelClass" for="book-notes">{{ t.notesL }} <span class="font-normal text-slate-400">{{ t.optional }}</span></label>
                <textarea id="book-notes" v-model="bookNotes" rows="3" :placeholder="t.notesPh" :class="inputClass" />
              </div>
              <p v-if="bookMessage" class="rounded-xl border border-emerald-200 bg-emerald-50 px-3.5 py-2.5 text-sm font-medium text-emerald-800">{{ bookMessage }}</p>
              <button type="submit" :disabled="bookSaving" class="w-full rounded-xl bg-navy-600 py-3 text-sm font-bold text-white shadow-lg shadow-navy-600/25 transition hover:bg-navy-700 disabled:opacity-50">
                {{ bookSaving ? t.sendingBtn : t.requestBtn }}
              </button>
            </div>
          </form>
          <aside class="space-y-4 lg:col-span-2">
            <div class="rounded-2xl bg-navy-950 p-6 text-white shadow-xl">
              <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-navy-300">{{ t.yourSelection }}</p>
              <h4 class="mt-1 text-lg font-bold">{{ selectedService ? serviceName(selectedService.id) : t.selectService }}</h4>
              <dl class="mt-4 space-y-2.5 text-sm">
                <div class="flex justify-between gap-3"><dt class="text-navy-200">{{ t.durationL }}</dt><dd class="font-semibold">{{ selectedService ? `${selectedService.durationMinutes} ${t.minUnit}` : "—" }}</dd></div>
                <div class="flex justify-between gap-3"><dt class="text-navy-200">{{ t.locationRow }}</dt><dd class="text-right font-semibold">{{ selectedSite ? selectedSite.name : t.defaultAddr }}</dd></div>
                <div class="flex justify-between gap-3"><dt class="text-navy-200">{{ t.scheduleL }}</dt><dd class="font-semibold">{{ bookDate || "—" }} {{ bookTime }}</dd></div>
              </dl>
              <div class="mt-4 rounded-xl bg-white/10 p-3 text-xs leading-relaxed text-navy-100 ring-1 ring-white/15">
                {{ t.freeCancel }}
              </div>
            </div>
            <div class="rounded-2xl bg-white p-6 shadow-lg ring-1 ring-slate-200">
              <h4 class="text-sm font-bold text-slate-900">{{ t.howItWorks }}</h4>
              <ol class="mt-3 space-y-2.5 text-[13px] text-slate-600">
                <li class="flex gap-2.5"><span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-navy-950 text-[11px] font-bold text-white">1</span> {{ t.step1 }}</li>
                <li class="flex gap-2.5"><span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-navy-950 text-[11px] font-bold text-white">2</span> {{ t.step2 }}</li>
                <li class="flex gap-2.5"><span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-navy-950 text-[11px] font-bold text-white">3</span> {{ t.step3 }}</li>
              </ol>
            </div>
          </aside>
        </section>

        <!-- Profile -->
        <section v-else class="grid gap-5 md:grid-cols-2">
          <div class="rounded-2xl bg-white p-6 shadow-xl ring-1 ring-slate-200">
            <div class="flex items-center gap-4">
              <span class="flex h-14 w-14 items-center justify-center rounded-2xl bg-navy-950 text-lg font-bold text-white">{{ initials(customer.name) }}</span>
              <div>
                <h3 class="text-base font-bold text-slate-900">{{ customer.name }}</h3>
                <p class="text-sm text-slate-500">{{ customer.email }}</p>
              </div>
            </div>
            <dl class="mt-5 space-y-3 text-sm">
              <div class="flex justify-between gap-4 border-b border-slate-100 pb-3"><dt class="text-slate-400">{{ t.phone }}</dt><dd class="font-semibold text-slate-700">{{ customer.phone }}</dd></div>
              <div class="flex justify-between gap-4 border-b border-slate-100 pb-3"><dt class="text-slate-400">{{ t.addressL }}</dt><dd class="text-right font-semibold text-slate-700">{{ customer.address }}</dd></div>
              <div class="flex justify-between gap-4"><dt class="text-slate-400">{{ t.areaL }}</dt><dd class="font-semibold text-slate-700">{{ customer.area }}</dd></div>
            </dl>
            <div v-if="sites.length > 0" class="mt-4 rounded-xl bg-slate-50 p-4 ring-1 ring-slate-100">
              <p class="text-[11px] font-bold uppercase tracking-wider text-slate-400">{{ t.myLocations }} ({{ sites.length }})</p>
              <ul class="mt-2 space-y-1.5 text-[13px] text-slate-600">
                <li v-for="s in sites" :key="s.id" class="flex items-center justify-between gap-2">
                  <span class="truncate font-medium">{{ s.name }}</span>
                  <span v-if="s.isDefault" class="shrink-0 rounded-full bg-navy-100 px-2 py-0.5 text-[10px] font-bold text-navy-700">{{ t.defaultBadge }}</span>
                </li>
              </ul>
            </div>
          </div>
          <div class="rounded-2xl bg-white p-6 shadow-xl ring-1 ring-slate-200">
            <h3 class="text-base font-bold text-slate-900">{{ t.security }}</h3>
            <p class="mt-1 text-[13px] text-slate-500">{{ t.securitySub }}</p>
            <form class="mt-4 space-y-3" @submit.prevent="changePassword">
              <input v-model="currentPw" type="password" required :placeholder="t.currentPw" :class="inputClass" />
              <input v-model="newPw" type="password" required minlength="8" :placeholder="t.newPw" :class="inputClass" />
              <p v-if="pwMessage" class="rounded-xl border border-emerald-200 bg-emerald-50 px-3.5 py-2.5 text-sm font-medium text-emerald-800">{{ pwMessage }}</p>
              <button type="submit" class="rounded-xl bg-navy-950 px-5 py-2.5 text-sm font-bold text-white shadow transition hover:bg-navy-800">
                {{ t.updatePw }}
              </button>
            </form>
          </div>
        </section>
      </div>
    </main>

    <!-- Feedback modal -->
    <div
      v-if="feedbackFor"
      class="fixed inset-0 z-50 flex items-center justify-center bg-navy-950/60 p-4 backdrop-blur-sm"
      @click.self="feedbackFor = null"
    >
      <div class="w-full max-w-md overflow-hidden rounded-2xl bg-white shadow-2xl">
        <div class="bg-gradient-to-r from-navy-950 to-navy-800 px-6 py-5 text-white">
          <h3 class="text-base font-bold">{{ t.reviewTitle }} {{ feedbackFor.bookingNumber }}</h3>
          <p class="mt-0.5 text-[13px] text-navy-200">{{ t.howWas }} ({{ serviceName(feedbackFor.serviceType) }})</p>
        </div>
        <form class="space-y-4 px-6 py-5" @submit.prevent="submitFeedback">
          <div>
            <label :class="labelClass">{{ t.yourRating }}</label>
            <div class="flex items-center gap-1.5">
              <button
                v-for="n in 5"
                :key="n"
                type="button"
                class="flex h-11 w-11 items-center justify-center rounded-xl text-2xl transition focus:outline-none"
                :class="n <= feedbackRating ? 'bg-amber-50 text-amber-400 ring-1 ring-amber-200' : 'bg-slate-50 text-slate-300 ring-1 ring-slate-200 hover:text-amber-300'"
                @click="feedbackRating = n"
                :aria-label="`${n} star${n > 1 ? 's' : ''}`"
              >
                ★
              </button>
            </div>
          </div>
          <div>
            <label :class="labelClass" for="fb-comment">{{ t.commentL }}</label>
            <textarea id="fb-comment" v-model="feedbackComment" rows="3" :placeholder="t.commentPh" :class="inputClass" />
          </div>
          <p v-if="feedbackError" class="rounded-xl border border-rose-200 bg-rose-50 px-3.5 py-2.5 text-sm text-rose-700">{{ feedbackError }}</p>
          <div class="flex justify-end gap-2">
            <button type="button" class="rounded-xl border border-slate-200 px-4 py-2.5 text-sm font-semibold text-slate-600 hover:bg-slate-50" @click="feedbackFor = null">
              {{ t.cancel }}
            </button>
            <button type="submit" :disabled="feedbackSaving" class="rounded-xl bg-navy-600 px-5 py-2.5 text-sm font-bold text-white shadow hover:bg-navy-700 disabled:opacity-50">
              {{ feedbackSaving ? t.submitting : t.submitReview }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
