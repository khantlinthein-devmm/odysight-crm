<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import {
  approveQuote,
  createQuote,
  downloadQuotePdf,
  getQuotes,
  updateQuote,
  type Quote,
  type QuoteItem,
  type QuoteStatus,
} from "../../../lib/quotes";
import { getCustomers, type Customer } from "../../../lib/customers";
import { getSites, type Site } from "../../../lib/sites";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";
import { showToast } from "../../../lib/toast";
import { getWorkspaceSettings } from "../../../lib/settings";
import ConfirmDialog from "../ui/ConfirmDialog.vue";

// Quotations: build a quote with several service lines, download the PDF to
// send to the customer (LINE, email, print), record their answer, and turn an
// accepted quote into a booking.

const role = getSessionUser()?.role ?? "viewer";
const canCreate = hasPermission(role, "quotes.create");
const canUpdate = hasPermission(role, "quotes.update");
const canApprove = hasPermission(role, "quotes.approve");
const canBook = hasPermission(role, "bookings.create");

const quotes = ref<Quote[]>([]);
const customers = ref<Customer[]>([]);
const loading = ref(true);
const filter = ref<QuoteStatus | "">("");
const busyId = ref<number | null>(null);

const STATUS: Record<QuoteStatus, { label: string; cls: string }> = {
  draft: { label: "Draft", cls: "bg-gray-100 text-gray-700" },
  sent: { label: "Sent", cls: "bg-sky-50 text-sky-700" },
  accepted: { label: "Accepted", cls: "bg-green-50 text-green-700" },
  rejected: { label: "Declined", cls: "bg-red-50 text-red-700" },
  expired: { label: "Expired", cls: "bg-amber-50 text-amber-800" },
};

const customerNames = computed(() => {
  const map = new Map<number, string>();
  for (const c of customers.value) map.set(c.id, `${c.firstName} ${c.lastName}`.trim());
  return map;
});

const visible = computed(() => quotes.value.filter((q) => !filter.value || q.status === filter.value));

function money(q: { currency: string }, v: number): string {
  return `${q.currency} ${v.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

function shortDate(d?: string | null): string {
  if (!d) return "—";
  return new Date(d.length === 10 ? d + "T00:00:00" : d).toLocaleDateString("en-GB", { day: "numeric", month: "short", year: "numeric" });
}

function isPastValid(q: Quote): boolean {
  return !!q.validUntil && (q.status === "sent" || q.status === "draft") && new Date(q.validUntil + "T23:59:59") < new Date();
}

async function fetchQuotes() {
  loading.value = true;
  try {
    const [rows, customerRows] = await Promise.all([
      getQuotes({ limit: 200 }),
      customers.value.length > 0 ? Promise.resolve(customers.value) : getCustomers({ limit: 200 }),
    ]);
    quotes.value = rows;
    customers.value = customerRows;
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to load quotes", "error");
  } finally {
    loading.value = false;
  }
}

function replace(q: Quote) {
  quotes.value = quotes.value.map((x) => (x.id === q.id ? q : x));
}

// ---- Editor -------------------------------------------------------------

interface Line { serviceName: string; description: string; quantity: number; unitPrice: number }
interface Draft {
  id: number | null;
  customerId: number | "";
  siteId: number | "";
  validUntil: string;
  vat: boolean;
  /** Line prices include VAT (ราคารวม VAT). */
  inclusive: boolean;
  notes: string;
  lines: Line[];
}

const editing = ref<Draft | null>(null);
const sites = ref<Site[]>([]);
const saving = ref(false);

function plusDays(n: number): string {
  const d = new Date();
  d.setDate(d.getDate() + n);
  return d.toISOString().slice(0, 10);
}

function blankLine(): Line {
  return { serviceName: "", description: "", quantity: 1, unitPrice: 0 };
}

// New quotes follow Settings → Company → Prices include VAT.
const companyInclusive = ref(false);
getWorkspaceSettings()
  .then((s) => (companyInclusive.value = !!s.company.pricesIncludeVat))
  .catch(() => {});

function startNew() {
  editing.value = { id: null, customerId: "", siteId: "", validUntil: plusDays(30), vat: true, inclusive: companyInclusive.value, notes: "", lines: [blankLine()] };
}

function startEdit(q: Quote) {
  editing.value = {
    id: q.id,
    customerId: q.customerId,
    siteId: q.siteId ?? "",
    validUntil: q.validUntil ?? "",
    vat: q.taxRate > 0,
    inclusive: !!q.pricesIncludeVat,
    notes: q.notes ?? "",
    lines: q.items.map((i) => ({ serviceName: i.serviceName, description: i.description ?? "", quantity: i.quantity, unitPrice: i.unitPrice })),
  };
}

watch(
  () => editing.value?.customerId,
  async (cid, old) => {
    if (!editing.value) return;
    if (old !== undefined && cid !== old) editing.value.siteId = "";
    sites.value = [];
    if (!cid) return;
    try {
      sites.value = await getSites({ customerId: Number(cid), limit: 100 });
    } catch {
      /* the site is optional */
    }
  },
);

const draftTotals = computed(() => {
  const e = editing.value;
  if (!e) return { subtotal: 0, vat: 0, total: 0 };
  const sum = e.lines.reduce((s, l) => s + Math.round((Number(l.quantity) || 0) * (Number(l.unitPrice) || 0) * 100) / 100, 0);
  if (e.vat && e.inclusive) {
    // ราคารวม VAT: the lines are what the customer pays; VAT is inside.
    const subtotal = Math.round((sum * 100 / 107) * 100) / 100;
    return { subtotal, vat: Math.round((sum - subtotal) * 100) / 100, total: sum };
  }
  const vat = e.vat ? Math.round(sum * 7) / 100 : 0;
  return { subtotal: sum, vat, total: sum + vat };
});

function lineTotal(l: Line): number {
  return (Number(l.quantity) || 0) * (Number(l.unitPrice) || 0);
}

function validateDraft(e: Draft): string | null {
  if (e.customerId === "") return "Choose a customer";
  const lines = e.lines.filter((l) => l.serviceName.trim());
  if (lines.length === 0) return "Add at least one service line";
  if (lines.some((l) => !(Number(l.quantity) > 0))) return "Each line needs a quantity above zero";
  if (lines.some((l) => Number(l.unitPrice) < 0)) return "Prices can't be negative";
  return null;
}

async function save(andDownload: boolean) {
  const e = editing.value;
  if (!e) return;
  const problem = validateDraft(e);
  if (problem) {
    showToast(problem, "error");
    return;
  }
  const items: QuoteItem[] = e.lines
    .filter((l) => l.serviceName.trim())
    .map((l) => ({ serviceName: l.serviceName.trim(), description: l.description.trim(), quantity: Number(l.quantity), unitPrice: Number(l.unitPrice) }));
  saving.value = true;
  try {
    let q: Quote;
    if (e.id === null) {
      q = await createQuote({
        customerId: Number(e.customerId),
        siteId: e.siteId === "" ? null : Number(e.siteId),
        status: "draft",
        currency: "THB",
        taxRate: e.vat ? 7 : 0,
        pricesIncludeVat: e.vat && e.inclusive,
        validUntil: e.validUntil || null,
        notes: e.notes.trim(),
        items,
      });
      quotes.value = [q, ...quotes.value];
    } else {
      q = await updateQuote(e.id, {
        ...(e.siteId === "" ? { clearSiteId: true } : { siteId: Number(e.siteId) }),
        ...(e.validUntil ? { validUntil: e.validUntil } : { clearValidUntil: true }),
        taxRate: e.vat ? 7 : 0,
        pricesIncludeVat: e.vat && e.inclusive,
        notes: e.notes.trim(),
        items,
      });
      replace(q);
    }
    editing.value = null;
    showToast(`Quote ${q.quoteNumber} saved`, "success");
    if (andDownload) await pdf(q);
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to save quote", "error");
  } finally {
    saving.value = false;
  }
}

// ---- Actions ------------------------------------------------------------

async function markSent(q: Quote) {
  busyId.value = q.id;
  try {
    replace(await updateQuote(q.id, { status: "sent" }));
    showToast(`Quote ${q.quoteNumber} marked as sent`, "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to update quote", "error");
  } finally {
    busyId.value = null;
  }
}

async function pdf(q: Quote) {
  busyId.value = q.id;
  try {
    await downloadQuotePdf(q);
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to download PDF", "error");
  } finally {
    busyId.value = null;
  }
}

async function accept(q: Quote) {
  busyId.value = q.id;
  try {
    replace(await approveQuote(q.id));
    showToast(`Quote ${q.quoteNumber} accepted`, "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to accept quote", "error");
  } finally {
    busyId.value = null;
  }
}

const pendingDecline = ref<Quote | null>(null);
async function confirmDecline() {
  const q = pendingDecline.value;
  pendingDecline.value = null;
  if (!q) return;
  try {
    replace(await updateQuote(q.id, { status: "rejected" }));
    showToast(`Quote ${q.quoteNumber} marked as declined`, "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to update quote", "error");
  }
}

onMounted(fetchQuotes);
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-white">
    <div class="flex flex-wrap items-start justify-between gap-3 border-b border-gray-200 px-4 py-4 sm:px-6">
      <div class="min-w-0">
        <h2 class="text-base font-semibold text-gray-900">Quotations</h2>
        <p class="mt-1 text-sm text-gray-500">Create a quote, download the PDF and send it to the customer, then record their answer.</p>
      </div>
      <button v-if="canCreate && !editing" type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700" @click="startNew">
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" d="M12 5v14M5 12h14" /></svg>
        New quote
      </button>
    </div>

    <!-- Editor -->
    <div v-if="editing" class="space-y-4 border-b border-gray-200 bg-gray-50/60 px-4 py-5 sm:px-6">
      <h3 class="text-sm font-semibold text-gray-900">{{ editing.id ? "Edit quote" : "New quote" }}</h3>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <label class="block">
          <span class="mb-1 block text-xs font-medium text-gray-600">Customer</span>
          <select id="q-customer" v-model="editing.customerId" :disabled="editing.id !== null" class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm disabled:opacity-60">
            <option value="" disabled>Select customer</option>
            <option v-for="c in customers" :key="c.id" :value="c.id">{{ c.firstName }} {{ c.lastName }}</option>
          </select>
        </label>
        <label class="block">
          <span class="mb-1 block text-xs font-medium text-gray-600">Site (optional)</span>
          <select id="q-site" v-model="editing.siteId" :disabled="editing.customerId === ''" class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm disabled:opacity-50">
            <option value="">Customer's address</option>
            <option v-for="s in sites" :key="s.id" :value="s.id">{{ s.name }}</option>
          </select>
        </label>
        <label class="block">
          <span class="mb-1 block text-xs font-medium text-gray-600">Valid until</span>
          <input id="q-valid" v-model="editing.validUntil" type="date" class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm" />
        </label>
      </div>

      <div class="overflow-x-auto rounded-lg border border-gray-200 bg-white">
        <table class="w-full min-w-[640px] text-sm">
          <thead>
            <tr class="border-b border-gray-200 text-left text-xs uppercase tracking-wide text-gray-500">
              <th class="px-3 py-2 font-medium">Service</th>
              <th class="w-24 px-3 py-2 font-medium">Qty</th>
              <th class="w-32 px-3 py-2 font-medium">Unit price</th>
              <th class="w-32 px-3 py-2 text-right font-medium">Amount</th>
              <th class="w-10 px-2 py-2"><span class="sr-only">Remove</span></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(l, i) in editing.lines" :key="i" class="border-b border-gray-100 align-top last:border-0">
              <td class="px-3 py-2">
                <input :id="`q-line-${i}`" v-model="l.serviceName" placeholder="e.g. Deep cleaning, 3-bedroom condo" class="w-full rounded-md border border-gray-200 px-2 py-1.5 text-sm" />
                <input :id="`q-line-desc-${i}`" v-model="l.description" placeholder="Details (optional)" class="mt-1 w-full rounded-md border border-gray-100 px-2 py-1 text-xs text-gray-600" />
              </td>
              <td class="px-3 py-2"><input :id="`q-qty-${i}`" v-model.number="l.quantity" type="number" min="0" step="any" class="w-full rounded-md border border-gray-200 px-2 py-1.5 text-sm" /></td>
              <td class="px-3 py-2"><input :id="`q-price-${i}`" v-model.number="l.unitPrice" type="number" min="0" step="any" class="w-full rounded-md border border-gray-200 px-2 py-1.5 text-sm" /></td>
              <td class="px-3 py-2 text-right tabular-nums text-gray-900">{{ lineTotal(l).toLocaleString("en-US", { minimumFractionDigits: 2 }) }}</td>
              <td class="px-2 py-2 text-center">
                <button v-if="editing.lines.length > 1" type="button" class="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600" :aria-label="`Remove line ${i + 1}`" @click="editing.lines.splice(i, 1)">✕</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <button type="button" class="text-sm font-medium text-navy-700 hover:underline" @click="editing.lines.push(blankLine())">+ Add line</button>

      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <label class="block">
          <span class="mb-1 block text-xs font-medium text-gray-600">Notes on the quotation (optional)</span>
          <textarea id="q-notes" v-model="editing.notes" rows="3" maxlength="2000" placeholder="e.g. Cleaning supplies included. Payment within 7 days of each visit." class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm" />
        </label>
        <div class="space-y-1.5 self-end rounded-lg border border-gray-200 bg-white p-3 text-sm">
          <label class="flex items-center gap-2 text-gray-700">
            <input id="q-vat" v-model="editing.vat" type="checkbox" class="h-4 w-4" />
            Add VAT 7%
          </label>
          <label v-if="editing.vat" class="flex items-center gap-2 text-gray-700">
            <input id="q-inclusive" v-model="editing.inclusive" type="checkbox" class="h-4 w-4" />
            Prices include VAT (ราคารวม VAT)
          </label>
          <div class="flex justify-between text-gray-600"><span>{{ editing.vat && editing.inclusive ? "Value before VAT" : "Subtotal" }}</span><span class="tabular-nums">{{ money({ currency: "THB" }, draftTotals.subtotal) }}</span></div>
          <div v-if="editing.vat" class="flex justify-between text-gray-600"><span>VAT 7%</span><span class="tabular-nums">{{ money({ currency: "THB" }, draftTotals.vat) }}</span></div>
          <div class="flex justify-between border-t border-gray-100 pt-1.5 font-semibold text-gray-900"><span>Total</span><span class="tabular-nums">{{ money({ currency: "THB" }, draftTotals.total) }}</span></div>
        </div>
      </div>

      <div class="flex flex-wrap justify-end gap-2">
        <button type="button" class="rounded-lg border border-gray-200 bg-white px-4 py-2 text-sm text-gray-700 hover:bg-gray-50" @click="editing = null">Cancel</button>
        <button type="button" :disabled="saving" class="rounded-lg border border-navy-200 bg-white px-4 py-2 text-sm font-medium text-navy-700 hover:bg-navy-50 disabled:opacity-50" @click="save(false)">Save draft</button>
        <button type="button" :disabled="saving" class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50" @click="save(true)">
          {{ saving ? "Saving…" : "Save & download PDF" }}
        </button>
      </div>
    </div>

    <!-- Filters -->
    <div class="flex flex-wrap gap-2 px-4 pt-4 sm:px-6">
      <button
        v-for="f in ([['', 'All'], ['draft', 'Draft'], ['sent', 'Sent'], ['accepted', 'Accepted'], ['rejected', 'Declined']] as const)"
        :key="f[0]"
        type="button"
        class="rounded-full px-3 py-1 text-xs font-medium"
        :class="filter === f[0] ? 'bg-navy-600 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'"
        @click="filter = f[0]"
      >{{ f[1] }}</button>
    </div>

    <div v-if="loading" class="px-6 py-10 text-center text-sm text-gray-500">Loading quotes…</div>
    <div v-else-if="visible.length === 0" class="px-6 py-10 text-center text-sm text-gray-500">
      {{ quotes.length === 0 ? "No quotes yet. Tap “New quote” to make the first one." : "No quotes with this status." }}
    </div>
    <div v-else class="overflow-x-auto px-4 pb-4 pt-3 sm:px-6">
      <table class="w-full min-w-[760px] text-left text-sm">
        <thead>
          <tr class="border-b border-gray-200 text-xs uppercase tracking-wide text-gray-500">
            <th class="py-2 pr-3 font-medium">Quote</th>
            <th class="py-2 pr-3 font-medium">Customer</th>
            <th class="py-2 pr-3 text-right font-medium">Total</th>
            <th class="py-2 pr-3 font-medium">Valid until</th>
            <th class="py-2 pr-3 font-medium">Status</th>
            <th class="py-2 text-right font-medium">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="q in visible" :key="q.id" class="border-b border-gray-100 align-middle">
            <td class="py-3 pr-3">
              <div class="font-medium text-gray-900">{{ q.quoteNumber }}</div>
              <div class="text-xs text-gray-500">{{ q.items.length }} {{ q.items.length === 1 ? "line" : "lines" }} · {{ shortDate(q.createdAt) }}</div>
            </td>
            <td class="py-3 pr-3 text-gray-700">{{ customerNames.get(q.customerId) ?? `Customer ${q.customerId}` }}</td>
            <td class="py-3 pr-3 text-right font-medium tabular-nums text-gray-900">{{ money(q, q.total) }}</td>
            <td class="py-3 pr-3" :class="isPastValid(q) ? 'text-amber-700' : 'text-gray-600'">
              {{ shortDate(q.validUntil) }}<span v-if="isPastValid(q)" class="block text-[11px]">past date</span>
            </td>
            <td class="py-3 pr-3"><span class="rounded-full px-2.5 py-0.5 text-xs font-medium" :class="STATUS[q.status].cls">{{ STATUS[q.status].label }}</span></td>
            <td class="py-3 text-right">
              <div class="flex flex-wrap justify-end gap-x-3 gap-y-1 text-sm font-medium">
                <button type="button" class="text-navy-700 hover:text-navy-900 disabled:opacity-50" :disabled="busyId === q.id" @click="pdf(q)">Download PDF</button>
                <button v-if="canUpdate && (q.status === 'draft' || q.status === 'sent')" type="button" class="text-gray-600 hover:text-gray-900" @click="startEdit(q)">Edit</button>
                <button v-if="canUpdate && q.status === 'draft'" type="button" class="text-navy-700 hover:text-navy-900 disabled:opacity-50" :disabled="busyId === q.id" title="Use after you have sent the PDF to the customer" @click="markSent(q)">Mark as sent</button>
                <button v-if="canApprove && (q.status === 'sent' || q.status === 'draft')" type="button" class="text-green-700 hover:text-green-900 disabled:opacity-50" :disabled="busyId === q.id" @click="accept(q)">Accepted</button>
                <button v-if="canUpdate && q.status === 'sent'" type="button" class="text-red-600 hover:text-red-800" @click="pendingDecline = q">Declined</button>
                <template v-if="q.status === 'accepted'">
                  <a v-if="q.convertedBookingId" href="/bookings" class="text-gray-500 hover:text-gray-800">Booking created</a>
                  <a v-else-if="canBook" :href="`/bookings?fromQuote=${q.id}`" class="text-navy-700 hover:text-navy-900">Create booking</a>
                </template>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <ConfirmDialog
      v-if="pendingDecline"
      title="Customer declined"
      :message="`Mark ${pendingDecline.quoteNumber} as declined? You can still make a new quote for this customer.`"
      confirm-label="Mark declined"
      @confirm="confirmDecline"
      @cancel="pendingDecline = null"
    />
  </div>
</template>
