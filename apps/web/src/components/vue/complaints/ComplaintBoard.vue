<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { getSessionUser } from "../../../lib/auth";
import { getBookings, type Booking } from "../../../lib/bookings";
import {
  CATEGORY_LABELS,
  CHANNEL_LABELS,
  SLA_HOURS,
  bookReclean,
  createComplaint,
  getComplaints,
  updateComplaint,
  type Complaint,
  type Severity,
} from "../../../lib/complaints";
import { getCustomers, type Customer } from "../../../lib/customers";
import { hasPermission } from "../../../lib/roles";
import { showToast } from "../../../lib/toast";

const role = getSessionUser()?.role;
const canManage = hasPermission(role, "complaints.manage");
const canBook = canManage && hasPermission(role, "bookings.create");

type Tab = "active" | "overdue" | "resolved" | "all";
const tab = ref<Tab>("active");
const search = ref("");
const complaints = ref<Complaint[]>([]);
const loading = ref(true);
const selectedId = ref<number | null>(null);
const selected = computed(() => complaints.value.find((c) => c.id === selectedId.value) ?? null);

function errText(e: unknown, fallback: string) {
  return e instanceof Error && e.message ? e.message : fallback;
}

async function load() {
  loading.value = true;
  try {
    complaints.value = await getComplaints({
      status: tab.value === "active" || tab.value === "overdue" ? "active" : tab.value === "resolved" ? "resolved" : "",
      overdue: tab.value === "overdue",
      search: search.value.trim() || undefined,
    });
    if (selectedId.value && !complaints.value.some((c) => c.id === selectedId.value)) selectedId.value = null;
  } catch (e) {
    showToast(errText(e, "Failed to load complaints"), "error");
  } finally {
    loading.value = false;
  }
}

function setTab(t: Tab) {
  tab.value = t;
  void load();
}

function replace(c: Complaint) {
  complaints.value = complaints.value.map((x) => (x.id === c.id ? c : x));
}

const severityPill: Record<Severity, string> = {
  high: "bg-rose-100 text-rose-700",
  medium: "bg-amber-100 text-amber-700",
  low: "bg-gray-100 text-gray-600",
};
const statusPill: Record<string, string> = {
  open: "bg-blue-100 text-blue-700",
  in_progress: "bg-sky-100 text-sky-700",
  resolved: "bg-emerald-100 text-emerald-700",
  closed: "bg-gray-100 text-gray-500",
};

function dueLabel(c: Complaint): string {
  if (c.status === "resolved" || c.status === "closed") return c.resolvedAt ? `Resolved ${new Date(c.resolvedAt).toLocaleDateString()}` : "Resolved";
  const ms = new Date(c.dueAt).getTime() - Date.now();
  const h = Math.round(Math.abs(ms) / 3_600_000);
  const span = h >= 48 ? `${Math.round(h / 24)}d` : `${h}h`;
  return ms < 0 ? `Overdue by ${span}` : `Due in ${span}`;
}

// ---- New complaint ----
const showNew = ref(false);
const customers = ref<Customer[]>([]);
const recentBookings = ref<Booking[]>([]);
const form = ref({ customerId: "" as number | "", bookingId: "" as number | "", category: "quality", severity: "medium" as Severity, channel: "phone", description: "" });
const saving = ref(false);

async function openNew() {
  showNew.value = true;
  if (customers.value.length === 0) {
    try {
      [customers.value, recentBookings.value] = await Promise.all([getCustomers({ limit: 200 }), getBookings({ limit: 200 })]);
    } catch (e) {
      showToast(errText(e, "Failed to load customers"), "error");
    }
  }
}

const customerBookings = computed(() =>
  form.value.customerId === ""
    ? []
    : recentBookings.value.filter((b) => b.customerId === form.value.customerId).slice(0, 30),
);

async function submitNew() {
  if (form.value.customerId === "" || !form.value.description.trim()) {
    showToast("Pick the customer and describe the problem", "error");
    return;
  }
  saving.value = true;
  try {
    const c = await createComplaint({
      customerId: Number(form.value.customerId),
      bookingId: form.value.bookingId === "" ? undefined : Number(form.value.bookingId),
      category: form.value.category,
      severity: form.value.severity,
      channel: form.value.channel,
      description: form.value.description.trim(),
    });
    showToast(`${c.complaintNumber} logged — due ${new Date(c.dueAt).toLocaleString()}`, "success");
    showNew.value = false;
    form.value = { customerId: "", bookingId: "", category: "quality", severity: "medium", channel: "phone", description: "" };
    tab.value = "active";
    await load();
    select(complaints.value.find((x) => x.id === c.id) ?? c);
  } catch (e) {
    showToast(errText(e, "Failed to log complaint"), "error");
  } finally {
    saving.value = false;
  }
}

// ---- Detail actions ----
const resolutionText = ref("");
const recleanAt = ref("");
const sameCrew = ref(true);
const acting = ref(false);

function select(c: Complaint) {
  selectedId.value = c.id;
  resolutionText.value = c.resolution;
  const d = new Date(Date.now() + 24 * 3_600_000);
  d.setHours(9, 0, 0, 0);
  recleanAt.value = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}T09:00`;
}

async function setStatus(status: Complaint["status"]) {
  if (!selected.value) return;
  acting.value = true;
  try {
    replace(await updateComplaint(selected.value.id, { status, resolution: resolutionText.value }));
    showToast(`Marked ${status.replace("_", " ")}`, "success");
  } catch (e) {
    showToast(errText(e, "Update failed"), "error");
  } finally {
    acting.value = false;
  }
}

async function saveResolution() {
  if (!selected.value) return;
  acting.value = true;
  try {
    replace(await updateComplaint(selected.value.id, { resolution: resolutionText.value }));
    showToast("Notes saved", "success");
  } catch (e) {
    showToast(errText(e, "Save failed"), "error");
  } finally {
    acting.value = false;
  }
}

async function doReclean() {
  if (!selected.value) return;
  if (!recleanAt.value) {
    showToast("Pick the re-clean date and time", "error");
    return;
  }
  acting.value = true;
  try {
    const c = await bookReclean(selected.value.id, {
      scheduledFor: new Date(recleanAt.value).toISOString(),
      sameCrew: sameCrew.value,
    });
    replace(c);
    showToast(`Re-clean ${c.recleanBookingNumber} booked`, "success");
  } catch (e) {
    showToast(errText(e, "Could not book the re-clean"), "error");
  } finally {
    acting.value = false;
  }
}

onMounted(load);

const input = "w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500";
</script>

<template>
  <div class="space-y-4">
    <section class="flex flex-wrap items-center gap-3 rounded-xl border border-gray-200 bg-white p-4">
      <div>
        <h2 class="text-base font-semibold text-gray-900">Complaints</h2>
        <p class="text-xs text-gray-500">Resolve within {{ SLA_HOURS.high }}h (high) · {{ SLA_HOURS.medium }}h (medium) · {{ SLA_HOURS.low }}h (low).</p>
      </div>
      <div class="flex overflow-hidden rounded-lg ring-1 ring-gray-200">
        <button v-for="t in (['active', 'overdue', 'resolved', 'all'] as Tab[])" :key="t" type="button"
          class="px-3 py-1.5 text-xs font-semibold capitalize"
          :class="tab === t ? 'bg-navy-600 text-white' : 'bg-white text-gray-600 hover:bg-gray-50'"
          @click="setTab(t)">{{ t }}</button>
      </div>
      <input v-model="search" type="search" placeholder="Search…" class="w-44 rounded-lg border border-gray-200 px-3 py-1.5 text-sm" @keydown.enter="load" />
      <button v-if="canManage" type="button" class="ml-auto rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700" @click="openNew">+ Log complaint</button>
    </section>

    <div class="grid grid-cols-1 gap-4 lg:grid-cols-5">
      <section class="space-y-2 lg:col-span-2">
        <div v-if="loading" class="rounded-xl bg-white p-8 text-center text-sm text-gray-500 ring-1 ring-gray-200">Loading…</div>
        <div v-else-if="complaints.length === 0" class="rounded-xl bg-white p-8 text-center text-sm text-gray-500 ring-1 ring-gray-200">
          {{ tab === "overdue" ? "Nothing overdue 🎉" : "No complaints here." }}
        </div>
        <button v-for="c in complaints" :key="c.id" type="button"
          class="block w-full rounded-xl border bg-white p-3 text-left shadow-sm transition hover:shadow-md"
          :class="selectedId === c.id ? 'border-navy-400 ring-2 ring-navy-200' : c.overdue ? 'border-rose-300' : 'border-gray-200'"
          @click="select(c)">
          <div class="flex items-center gap-2">
            <span class="text-xs font-bold text-gray-500">{{ c.complaintNumber }}</span>
            <span class="rounded-full px-2 py-0.5 text-[11px] font-semibold capitalize" :class="severityPill[c.severity]">{{ c.severity }}</span>
            <span class="rounded-full px-2 py-0.5 text-[11px] font-semibold" :class="statusPill[c.status]">{{ c.status.replace("_", " ") }}</span>
            <span class="ml-auto text-[11px] font-semibold" :class="c.overdue ? 'text-rose-600' : 'text-gray-400'">{{ dueLabel(c) }}</span>
          </div>
          <p class="mt-1 text-sm font-semibold text-gray-900">{{ c.customerName }}<span v-if="c.siteName" class="font-normal text-gray-500"> · {{ c.siteName }}</span></p>
          <p class="text-xs text-gray-500">{{ CATEGORY_LABELS[c.category] ?? c.category }}<span v-if="c.bookingNumber"> · {{ c.bookingNumber }}</span></p>
          <p class="mt-1 line-clamp-2 text-xs text-gray-600">{{ c.description }}</p>
        </button>
      </section>

      <section class="lg:col-span-3">
        <div v-if="!selected" class="rounded-xl border border-dashed border-gray-300 bg-white p-10 text-center text-sm text-gray-400">Select a complaint to handle it.</div>
        <div v-else class="space-y-4 rounded-xl border border-gray-200 bg-white p-5">
          <div class="flex flex-wrap items-center gap-2">
            <h3 class="text-lg font-bold text-gray-900">{{ selected.complaintNumber }}</h3>
            <span class="rounded-full px-2 py-0.5 text-xs font-semibold capitalize" :class="severityPill[selected.severity]">{{ selected.severity }}</span>
            <span class="rounded-full px-2 py-0.5 text-xs font-semibold" :class="statusPill[selected.status]">{{ selected.status.replace("_", " ") }}</span>
            <span class="ml-auto text-xs font-semibold" :class="selected.overdue ? 'text-rose-600' : 'text-gray-500'">{{ dueLabel(selected) }}</span>
          </div>
          <dl class="grid grid-cols-2 gap-2 text-sm">
            <div><dt class="text-xs text-gray-400">Customer</dt><dd class="font-medium text-gray-900">{{ selected.customerName }}</dd></div>
            <div><dt class="text-xs text-gray-400">Site</dt><dd>{{ selected.siteName || "—" }}</dd></div>
            <div><dt class="text-xs text-gray-400">Job</dt><dd>{{ selected.bookingNumber || "—" }}</dd></div>
            <div><dt class="text-xs text-gray-400">Reported via</dt><dd>{{ CHANNEL_LABELS[selected.channel] ?? selected.channel }} · {{ new Date(selected.createdAt).toLocaleString() }}</dd></div>
            <div class="col-span-2"><dt class="text-xs text-gray-400">{{ CATEGORY_LABELS[selected.category] ?? selected.category }}</dt><dd class="whitespace-pre-line text-gray-800">{{ selected.description }}</dd></div>
          </dl>

          <div class="rounded-lg bg-gray-50 p-3 ring-1 ring-gray-200">
            <p class="text-sm font-semibold text-gray-900">Re-clean visit</p>
            <p v-if="selected.recleanBookingNumber" class="mt-1 text-sm text-emerald-700">
              Booked: <a :href="`/bookings?search=${selected.recleanBookingNumber}`" class="font-semibold underline">{{ selected.recleanBookingNumber }}</a> (free, noted on the job)
            </p>
            <div v-else-if="canBook && selected.status !== 'resolved' && selected.status !== 'closed'" class="mt-2 flex flex-wrap items-center gap-2">
              <input v-model="recleanAt" type="datetime-local" class="rounded-lg border border-gray-200 px-3 py-1.5 text-sm" />
              <label class="flex items-center gap-1.5 text-xs text-gray-600"><input v-model="sameCrew" type="checkbox" /> Same crew</label>
              <button type="button" :disabled="acting" class="rounded-lg bg-amber-500 px-3 py-1.5 text-sm font-semibold text-white hover:bg-amber-600 disabled:opacity-50" @click="doReclean">Book re-clean</button>
            </div>
            <p v-else class="mt-1 text-xs text-gray-400">No re-clean booked.</p>
          </div>

          <div>
            <label class="text-xs font-semibold text-gray-600">What was done (resolution)</label>
            <textarea v-model="resolutionText" rows="3" :class="input" :disabled="!canManage" placeholder="e.g. Re-cleaned kitchen on 5 Oct, apologised, 10% off next visit"></textarea>
          </div>
          <div v-if="canManage" class="flex flex-wrap gap-2">
            <button type="button" :disabled="acting" class="rounded-lg border border-gray-300 px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50" @click="saveResolution">Save notes</button>
            <button v-if="selected.status === 'open'" type="button" :disabled="acting" class="rounded-lg bg-sky-600 px-3 py-1.5 text-sm font-semibold text-white" @click="setStatus('in_progress')">Start handling</button>
            <button v-if="selected.status !== 'resolved' && selected.status !== 'closed'" type="button" :disabled="acting" class="rounded-lg bg-emerald-600 px-3 py-1.5 text-sm font-semibold text-white" @click="setStatus('resolved')">Mark resolved</button>
            <button v-if="selected.status === 'resolved'" type="button" :disabled="acting" class="rounded-lg bg-gray-700 px-3 py-1.5 text-sm font-semibold text-white" @click="setStatus('closed')">Close</button>
            <button v-if="selected.status === 'resolved' || selected.status === 'closed'" type="button" :disabled="acting" class="rounded-lg border border-gray-300 px-3 py-1.5 text-sm text-gray-700" @click="setStatus('open')">Reopen</button>
          </div>
        </div>
      </section>
    </div>

    <div v-if="showNew" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="showNew = false">
      <div class="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl">
        <h3 class="text-base font-semibold text-gray-900">Log a complaint</h3>
        <div class="mt-4 grid grid-cols-2 gap-3">
          <label class="col-span-2 text-xs font-medium text-gray-600">Customer *
            <select v-model="form.customerId" :class="input" @change="form.bookingId = ''">
              <option value="" disabled>Select customer</option>
              <option v-for="c in customers" :key="c.id" :value="c.id">{{ c.firstName }} {{ c.lastName }}</option>
            </select>
          </label>
          <label class="col-span-2 text-xs font-medium text-gray-600">Job (optional — links site and crew)
            <select v-model="form.bookingId" :class="input" :disabled="form.customerId === ''">
              <option value="">No specific job</option>
              <option v-for="b in customerBookings" :key="b.id" :value="b.id">{{ b.bookingNumber }} · {{ new Date(b.scheduledFor).toLocaleDateString() }} · {{ b.serviceType }}</option>
            </select>
          </label>
          <label class="text-xs font-medium text-gray-600">Problem
            <select v-model="form.category" :class="input">
              <option v-for="(label, key) in CATEGORY_LABELS" :key="key" :value="key">{{ label }}</option>
            </select>
          </label>
          <label class="text-xs font-medium text-gray-600">Severity
            <select v-model="form.severity" :class="input">
              <option value="high">High — {{ SLA_HOURS.high }}h</option>
              <option value="medium">Medium — {{ SLA_HOURS.medium }}h</option>
              <option value="low">Low — {{ SLA_HOURS.low }}h</option>
            </select>
          </label>
          <label class="col-span-2 text-xs font-medium text-gray-600">Reported via
            <select v-model="form.channel" :class="input">
              <option v-for="(label, key) in CHANNEL_LABELS" :key="key" :value="key">{{ label }}</option>
            </select>
          </label>
          <label class="col-span-2 text-xs font-medium text-gray-600">What happened *
            <textarea v-model="form.description" rows="3" :class="input"></textarea>
          </label>
        </div>
        <div class="mt-5 flex justify-end gap-2">
          <button type="button" class="rounded-lg border border-gray-200 px-4 py-2 text-sm text-gray-700" @click="showNew = false">Cancel</button>
          <button type="button" :disabled="saving" class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" @click="submitNew">{{ saving ? "Saving…" : "Log complaint" }}</button>
        </div>
      </div>
    </div>
  </div>
</template>
