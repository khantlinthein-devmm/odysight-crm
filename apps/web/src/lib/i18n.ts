// Field-app translations for the screens cleaners use on their phones
// (My Jobs, check-in, checklists, bottom navigation). Office screens stay in
// English. The choice is per device and switching reloads the page, which
// keeps Vue and Svelte islands in sync without shared reactive state.

export type Lang = "en" | "th" | "my";

export const LANGS: { code: Lang; label: string }[] = [
  { code: "en", label: "English" },
  { code: "th", label: "ไทย" },
  { code: "my", label: "မြန်မာ" },
];

const STORAGE_KEY = "smileclean.lang";

const en = {
  "nav.jobs": "Jobs",
  "nav.checkin": "Check-in",
  "nav.calculator": "Calculator",
  "nav.home": "Home",
  "jobs.loading": "Loading jobs…",
  "jobs.empty": "No upcoming jobs",
  "jobs.emptyHint": "Open jobs you can accept will appear here.",
  "jobs.accept": "Accept job",
  "jobs.checkIn": "Check in",
  "jobs.checkOut": "Check out",
  "jobs.complete": "Mark completed",
  "jobs.checklist": "Checklist",
  "jobs.navigate": "Navigate",
  "jobs.working": "Working…",
  "jobs.accepted": "Job accepted",
  "jobs.acceptFailed": "Accept failed",
  "jobs.checkedIn": "Checked in",
  "jobs.checkInFailed": "Check-in failed",
  "jobs.checkedOut": "Checked out",
  "jobs.checkOutFailed": "Check-out failed",
  "jobs.completed": "Job completed",
  "jobs.completeFailed": "Complete failed",
  "jobs.noProfile": "No cleaner profile linked to this login — ask the office",
  "jobs.offlineAccept": "Saved offline — accept will sync when online",
  "jobs.offlineCheckIn": "Saved offline — check-in will sync",
  "jobs.offlineCheckOut": "Saved offline — check-out will sync",
  "jobs.offlineComplete": "Saved offline — completion will sync",
  "status.pending": "pending",
  "status.confirmed": "confirmed",
  "status.in_progress": "in progress",
  "status.completed": "completed",
  "status.cancelled": "cancelled",
  "att.title": "Daily check-in",
  "att.checkedIn": "Checked-in",
  "att.checkedOut": "Checked-out",
  "att.notIn": "Not in",
  "att.in": "In",
  "att.out": "Out",
  "att.history": "History",
  "att.noProfile": "Your login is not linked to a cleaner profile — ask the office.",
  "att.locating": "Getting your location…",
  "att.needLocation": "Allow location access to check in at the site.",
  "cl.title": "Checklist & proof of work",
  "cl.subtitle": "Tick each item and add before/after photos.",
  "cl.selectBooking": "Select booking",
  "cl.loadingBookings": "Loading bookings…",
  "cl.newFromTemplate": "New from template",
  "cl.loading": "Loading…",
  "cl.noItems": "No items yet.",
  "cl.before": "Before",
  "cl.after": "After",
  "cl.clientConfirm": "Client confirmation",
  "cl.signedBy": "Signed by",
  "cl.signaturePlaceholder": "Client name / signature",
  "cl.confirm": "Confirm",
  "cl.confirming": "Confirming…",
  "cl.empty": "Load a booking checklist, or create one from the default template.",
  "cl.photoAttached": "Photo attached",
  "cl.confirmed": "Checklist confirmed by client",
  "cl.needSignature": "Enter client name/signature to confirm",
  "lang.label": "Language",
};

export type MessageKey = keyof typeof en;

const th: Record<MessageKey, string> = {
  "nav.jobs": "งาน",
  "nav.checkin": "ลงเวลา",
  "nav.calculator": "คำนวณราคา",
  "nav.home": "หน้าหลัก",
  "jobs.loading": "กำลังโหลดงาน…",
  "jobs.empty": "ยังไม่มีงานที่จะถึง",
  "jobs.emptyHint": "งานที่เปิดให้รับจะแสดงที่นี่",
  "jobs.accept": "รับงาน",
  "jobs.checkIn": "ลงเวลาเข้างาน",
  "jobs.checkOut": "ลงเวลาออกงาน",
  "jobs.complete": "ทำงานเสร็จแล้ว",
  "jobs.checklist": "เช็กลิสต์",
  "jobs.navigate": "นำทาง",
  "jobs.working": "กำลังดำเนินการ…",
  "jobs.accepted": "รับงานแล้ว",
  "jobs.acceptFailed": "รับงานไม่สำเร็จ",
  "jobs.checkedIn": "ลงเวลาเข้างานแล้ว",
  "jobs.checkInFailed": "ลงเวลาเข้างานไม่สำเร็จ",
  "jobs.checkedOut": "ลงเวลาออกงานแล้ว",
  "jobs.checkOutFailed": "ลงเวลาออกงานไม่สำเร็จ",
  "jobs.completed": "บันทึกงานเสร็จแล้ว",
  "jobs.completeFailed": "บันทึกงานเสร็จไม่สำเร็จ",
  "jobs.noProfile": "บัญชีนี้ยังไม่ได้ผูกกับโปรไฟล์แม่บ้าน กรุณาติดต่อออฟฟิศ",
  "jobs.offlineAccept": "บันทึกแบบออฟไลน์แล้ว จะส่งเมื่อมีอินเทอร์เน็ต",
  "jobs.offlineCheckIn": "บันทึกเวลาเข้าแบบออฟไลน์แล้ว จะส่งเมื่อมีอินเทอร์เน็ต",
  "jobs.offlineCheckOut": "บันทึกเวลาออกแบบออฟไลน์แล้ว จะส่งเมื่อมีอินเทอร์เน็ต",
  "jobs.offlineComplete": "บันทึกงานเสร็จแบบออฟไลน์แล้ว จะส่งเมื่อมีอินเทอร์เน็ต",
  "status.pending": "รอรับงาน",
  "status.confirmed": "ยืนยันแล้ว",
  "status.in_progress": "กำลังทำ",
  "status.completed": "เสร็จแล้ว",
  "status.cancelled": "ยกเลิก",
  "att.title": "ลงเวลาทำงานวันนี้",
  "att.checkedIn": "เข้างานแล้ว",
  "att.checkedOut": "ออกงานแล้ว",
  "att.notIn": "ยังไม่เข้างาน",
  "att.in": "เข้า",
  "att.out": "ออก",
  "att.history": "ประวัติ",
  "att.noProfile": "บัญชีนี้ยังไม่ได้ผูกกับโปรไฟล์แม่บ้าน กรุณาติดต่อออฟฟิศ",
  "att.locating": "กำลังหาตำแหน่งของคุณ…",
  "att.needLocation": "กรุณาอนุญาตให้เข้าถึงตำแหน่ง เพื่อลงเวลาที่หน้างาน",
  "cl.title": "เช็กลิสต์และหลักฐานการทำงาน",
  "cl.subtitle": "ติ๊กแต่ละรายการ และแนบรูปก่อน/หลังทำ",
  "cl.selectBooking": "เลือกงาน",
  "cl.loadingBookings": "กำลังโหลดงาน…",
  "cl.newFromTemplate": "สร้างจากแม่แบบ",
  "cl.loading": "กำลังโหลด…",
  "cl.noItems": "ยังไม่มีรายการ",
  "cl.before": "ก่อนทำ",
  "cl.after": "หลังทำ",
  "cl.clientConfirm": "ลูกค้ายืนยัน",
  "cl.signedBy": "ลงชื่อโดย",
  "cl.signaturePlaceholder": "ชื่อลูกค้า / ลายเซ็น",
  "cl.confirm": "ยืนยัน",
  "cl.confirming": "กำลังยืนยัน…",
  "cl.empty": "เลือกงานเพื่อดูเช็กลิสต์ หรือสร้างจากแม่แบบ",
  "cl.photoAttached": "แนบรูปแล้ว",
  "cl.confirmed": "ลูกค้ายืนยันเช็กลิสต์แล้ว",
  "cl.needSignature": "กรุณาใส่ชื่อ/ลายเซ็นลูกค้าเพื่อยืนยัน",
  "lang.label": "ภาษา",
};

const my: Record<MessageKey, string> = {
  "nav.jobs": "အလုပ်များ",
  "nav.checkin": "ချိန်စာရင်း",
  "nav.calculator": "ဈေးတွက်",
  "nav.home": "ပင်မ",
  "jobs.loading": "အလုပ်များ ဖွင့်နေသည်…",
  "jobs.empty": "လာမည့်အလုပ် မရှိသေးပါ",
  "jobs.emptyHint": "လက်ခံနိုင်သော အလုပ်များ ဤနေရာတွင် ပေါ်လာပါမည်။",
  "jobs.accept": "အလုပ်လက်ခံမည်",
  "jobs.checkIn": "အလုပ်ဝင်ချိန်မှတ်မည်",
  "jobs.checkOut": "အလုပ်ထွက်ချိန်မှတ်မည်",
  "jobs.complete": "အလုပ်ပြီးပြီ",
  "jobs.checklist": "စစ်ဆေးစာရင်း",
  "jobs.navigate": "လမ်းညွှန်",
  "jobs.working": "လုပ်ဆောင်နေသည်…",
  "jobs.accepted": "အလုပ်လက်ခံပြီးပါပြီ",
  "jobs.acceptFailed": "အလုပ်လက်ခံ၍ မရပါ",
  "jobs.checkedIn": "ဝင်ချိန် မှတ်ပြီးပါပြီ",
  "jobs.checkInFailed": "ဝင်ချိန် မှတ်၍ မရပါ",
  "jobs.checkedOut": "ထွက်ချိန် မှတ်ပြီးပါပြီ",
  "jobs.checkOutFailed": "ထွက်ချိန် မှတ်၍ မရပါ",
  "jobs.completed": "အလုပ်ပြီးကြောင်း မှတ်ပြီးပါပြီ",
  "jobs.completeFailed": "အလုပ်ပြီးကြောင်း မှတ်၍ မရပါ",
  "jobs.noProfile": "ဤအကောင့်ကို သန့်ရှင်းရေးဝန်ထမ်း profile နှင့် မချိတ်ရသေးပါ — ရုံးကို ဆက်သွယ်ပါ",
  "jobs.offlineAccept": "အင်တာနက်မရှိ၍ သိမ်းထားသည် — အင်တာနက်ရလျှင် ပို့ပါမည်",
  "jobs.offlineCheckIn": "ဝင်ချိန်ကို သိမ်းထားသည် — အင်တာနက်ရလျှင် ပို့ပါမည်",
  "jobs.offlineCheckOut": "ထွက်ချိန်ကို သိမ်းထားသည် — အင်တာနက်ရလျှင် ပို့ပါမည်",
  "jobs.offlineComplete": "အလုပ်ပြီးကြောင်း သိမ်းထားသည် — အင်တာနက်ရလျှင် ပို့ပါမည်",
  "status.pending": "စောင့်ဆိုင်းဆဲ",
  "status.confirmed": "အတည်ပြုပြီး",
  "status.in_progress": "လုပ်ဆောင်ဆဲ",
  "status.completed": "ပြီးဆုံး",
  "status.cancelled": "ပယ်ဖျက်",
  "att.title": "ယနေ့ အလုပ်ချိန်မှတ်တမ်း",
  "att.checkedIn": "ဝင်ပြီး",
  "att.checkedOut": "ထွက်ပြီး",
  "att.notIn": "မဝင်ရသေး",
  "att.in": "ဝင်",
  "att.out": "ထွက်",
  "att.history": "မှတ်တမ်း",
  "att.noProfile": "ဤအကောင့်ကို သန့်ရှင်းရေးဝန်ထမ်း profile နှင့် မချိတ်ရသေးပါ — ရုံးကို ဆက်သွယ်ပါ။",
  "att.locating": "သင့်တည်နေရာကို ရှာနေသည်…",
  "att.needLocation": "လုပ်ငန်းခွင်တွင် ချိန်မှတ်ရန် တည်နေရာ (location) ခွင့်ပြုပါ။",
  "cl.title": "စစ်ဆေးစာရင်းနှင့် အလုပ်ပြီးကြောင်း အထောက်အထား",
  "cl.subtitle": "အချက်တစ်ခုချင်း အမှန်ခြစ်ပြီး မလုပ်မီ/လုပ်ပြီး ဓာတ်ပုံ ထည့်ပါ။",
  "cl.selectBooking": "အလုပ်ရွေးပါ",
  "cl.loadingBookings": "အလုပ်များ ဖွင့်နေသည်…",
  "cl.newFromTemplate": "ပုံစံမှ အသစ်လုပ်မည်",
  "cl.loading": "ဖွင့်နေသည်…",
  "cl.noItems": "အချက်များ မရှိသေးပါ။",
  "cl.before": "မလုပ်မီ",
  "cl.after": "လုပ်ပြီး",
  "cl.clientConfirm": "ဖောက်သည် အတည်ပြုချက်",
  "cl.signedBy": "လက်မှတ်ထိုးသူ",
  "cl.signaturePlaceholder": "ဖောက်သည်အမည် / လက်မှတ်",
  "cl.confirm": "အတည်ပြုမည်",
  "cl.confirming": "အတည်ပြုနေသည်…",
  "cl.empty": "စစ်ဆေးစာရင်းကြည့်ရန် အလုပ်ရွေးပါ၊ သို့မဟုတ် ပုံစံမှ အသစ်လုပ်ပါ။",
  "cl.photoAttached": "ဓာတ်ပုံ ထည့်ပြီးပါပြီ",
  "cl.confirmed": "ဖောက်သည် အတည်ပြုပြီးပါပြီ",
  "cl.needSignature": "အတည်ပြုရန် ဖောက်သည်အမည်/လက်မှတ် ထည့်ပါ",
  "lang.label": "ဘာသာစကား",
};

const dictionaries: Record<Lang, Record<MessageKey, string>> = { en, th, my };

function isLang(v: unknown): v is Lang {
  return v === "en" || v === "th" || v === "my";
}

/** The device's chosen language, else a guess from the browser, else English. */
export function getLang(): Lang {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (isLang(saved)) return saved;
  } catch {
    /* storage blocked: fall through to the browser language */
  }
  if (typeof navigator !== "undefined") {
    const nav = navigator.language?.toLowerCase() ?? "";
    if (nav.startsWith("th")) return "th";
    if (nav.startsWith("my")) return "my";
  }
  return "en";
}

/** Saves the language and reloads so every island re-renders in it. */
export function setLang(lang: Lang): void {
  try {
    localStorage.setItem(STORAGE_KEY, lang);
  } catch {
    /* storage blocked: the choice lasts for this page only */
  }
  if (typeof window !== "undefined") window.location.reload();
}

/** Translates a key for the current device language (English fallback). */
export function t(key: MessageKey, lang: Lang = getLang()): string {
  return dictionaries[lang][key] ?? en[key];
}

/** Translated booking status, falling back to the raw value. */
export function tStatus(status: string, lang: Lang = getLang()): string {
  const key = `status.${status}` as MessageKey;
  return key in en ? t(key, lang) : status.replace(/_/g, " ");
}

/** Locale tag for dates/numbers in the chosen language. */
export function dateLocale(lang: Lang = getLang()): string {
  return lang === "th" ? "th-TH" : lang === "my" ? "my-MM" : "en-GB";
}
