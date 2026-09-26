<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { getSessionUser } from "../../../lib/auth";
import { getBookings, type Booking } from "../../../lib/bookings";
import { hasPermission } from "../../../lib/roles";
import { getSites, type Site } from "../../../lib/sites";
import {
  UNITS,
  createSupply,
  getMovements,
  getSiteCosts,
  getSupplies,
  recordMovement,
  updateSupply,
  type MovementKind,
  type SiteSupplyCost,
  type Supply,
  type SupplyMovement,
} from "../../../lib/supplies";
import { showToast } from "../../../lib/toast";

const canManage = hasPermission(getSessionUser()?.role, "supplies.manage");
type Tab = "stock" | "log" | "sites";
const tab = ref<Tab>("stock");

function ymd(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}
const now = new Date();
const from = ref(ymd(new Date(now.getFullYear(), now.getMonth(), 1)));
const to = ref(ymd(now));

const supplies = ref<Supply[]>([]);
const movements = ref<SupplyMovement[]>([]);
const siteCosts = ref<SiteSupplyCost[]>([]);
const loading = ref(true);
const lowCount = computed(() => supplies.value.filter((s) => s.lowStock).length);

function money(v: number): string {
  return v.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}
function qty(v: number): string {
  return Number.isInteger(v) ? String(v) : v.toFixed(2);
}
function errText(e: unknown, fallback: string) {
  return e instanceof Error && e.message ? e.message : fallback;
}

async function loadStock() {
  supplies.value = await getSupplies();
}
async function loadLog() {
  movements.value = await getMovements({ from: from.value, to: to.value });
}
async function loadSites() {
  siteCosts.value = await getSiteCosts(from.value, to.value);
}

async function load() {
  loading.value = true;
  try {
    if (tab.value === "stock") await loadStock();
    else if (tab.value === "log") await loadLog();
    else await loadSites();
  } catch (e) {
    showToast(errText(e, "Failed to load supplies"), "error");
  } finally {
    loading.value = false;
  }
}

function setTab(t: Tab) {
  tab.value = t;
  void load();
}

// ---- New item ----
const showNew = ref(false);
const newForm = ref({ name: "", unit: "bottle", unitCost: 0, stockQty: 0, reorderLevel: 0 });
async function saveNew() {
  if (!newForm.value.name.trim()) {
    showToast("Name is required", "error");
    return;
  }
  try {
    const s = await createSupply({ ...newForm.value, name: newForm.value.name.trim() });
    supplies.value = [...supplies.value, s].sort((a, b) => a.name.localeCompare(b.name));
    showNew.value = false;
    newForm.value = { name: "", unit: "bottle", unitCost: 0, stockQty: 0, reorderLevel: 0 };
    showToast(`${s.name} added`, "success");
  } catch (e) {
    showToast(errText(e, "Failed to add supply"), "error");
  }
}

// ---- Record movement ----
const moveFor = ref<Supply | null>(null);
const moveForm = ref({ kind: "usage" as MovementKind, quantity: 1, unitCost: 0, target: "site" as "site" | "job", siteId: "" as number | "", bookingId: "" as number | "", note: "" });
const sites = ref<Site[]>([]);
const jobs = ref<Booking[]>([]);
const recording = ref(false);

async function openMove(s: Supply, kind: MovementKind) {
  moveFor.value = s;
  moveForm.value = { kind, quantity: 1, unitCost: s.unitCost, target: "job", siteId: "", bookingId: "", note: "" };
  if (sites.value.length === 0) {
    try {
      const since = new Date(Date.now() - 14 * 86_400_000).toISOString();
      [sites.value, jobs.value] = await Promise.all([getSites({ limit: 200 }), getBookings({ from: since, limit: 200 })]);
    } catch (e) {
      showToast(errText(e, "Failed to load sites and jobs"), "error");
    }
  }
}

async function saveMove() {
  const s = moveFor.value;
  if (!s) return;
  const f = moveForm.value;
  recording.value = true;
  try {
    const res = await recordMovement(s.id, {
      kind: f.kind,
      quantity: Number(f.quantity),
      unitCost: f.kind === "purchase" ? Number(f.unitCost) : undefined,
      siteId: f.kind === "usage" && f.target === "site" && f.siteId !== "" ? Number(f.siteId) : undefined,
      bookingId: f.kind === "usage" && f.target === "job" && f.bookingId !== "" ? Number(f.bookingId) : undefined,
      note: f.note.trim() || undefined,
    });
    supplies.value = supplies.value.map((x) => (x.id === res.supply.id ? res.supply : x));
    showToast(`${s.name}: stock now ${qty(res.supply.stockQty)} ${s.unit}`, "success");
    moveFor.value = null;
  } catch (e) {
    showToast(errText(e, "Could not record"), "error");
  } finally {
    recording.value = false;
  }
}

async function toggleActive(s: Supply) {
  try {
    const u = await updateSupply(s.id, { active: !s.active });
    supplies.value = u.active ? supplies.value.map((x) => (x.id === u.id ? u : x)) : supplies.value.filter((x) => x.id !== u.id);
  } catch (e) {
    showToast(errText(e, "Update failed"), "error");
  }
}

const kindLabel: Record<MovementKind, string> = { purchase: "Purchase", usage: "Used", adjustment: "Adjustment" };
const totalSiteCost = computed(() => siteCosts.value.reduce((n, s) => n + s.supplyCost, 0));

onMounted(load);
const input = "w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500";
</script>

<template>
  <div class="space-y-4">
    <section class="flex flex-wrap items-center gap-3 rounded-xl border border-gray-200 bg-white p-4">
      <div>
        <h2 class="text-base font-semibold text-gray-900">Supplies</h2>
        <p class="text-xs text-gray-500">Stock on hand, what each site uses, and supply cost against site revenue.</p>
      </div>
      <div class="flex overflow-hidden rounded-lg ring-1 ring-gray-200">
        <button type="button" class="px-3 py-1.5 text-xs font-semibold" :class="tab === 'stock' ? 'bg-navy-600 text-white' : 'bg-white text-gray-600'" @click="setTab('stock')">
          Stock<span v-if="lowCount" class="ml-1 rounded-full bg-rose-500 px-1.5 text-[10px] text-white">{{ lowCount }}</span>
        </button>
        <button type="button" class="px-3 py-1.5 text-xs font-semibold" :class="tab === 'log' ? 'bg-navy-600 text-white' : 'bg-white text-gray-600'" @click="setTab('log')">Usage log</button>
        <button type="button" class="px-3 py-1.5 text-xs font-semibold" :class="tab === 'sites' ? 'bg-navy-600 text-white' : 'bg-white text-gray-600'" @click="setTab('sites')">Cost by site</button>
      </div>
      <template v-if="tab !== 'stock'">
        <input v-model="from" type="date" class="rounded-lg border border-gray-200 px-2 py-1.5 text-sm" />
        <input v-model="to" type="date" class="rounded-lg border border-gray-200 px-2 py-1.5 text-sm" />
        <button type="button" class="rounded-lg bg-gray-800 px-3 py-1.5 text-sm text-white" @click="load">Show</button>
      </template>
      <button v-if="canManage && tab === 'stock'" type="button" class="ml-auto rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white" @click="showNew = true">+ Add supply</button>
    </section>

    <section class="overflow-x-auto rounded-xl border border-gray-200 bg-white">
      <div v-if="loading" class="p-10 text-center text-sm text-gray-500">Loading…</div>

      <table v-else-if="tab === 'stock'" class="min-w-full text-sm">
        <thead class="bg-gray-50 text-left text-xs font-semibold uppercase text-gray-500">
          <tr><th class="px-4 py-3">Item</th><th class="px-4 py-3 text-right">In stock</th><th class="px-4 py-3 text-right">Reorder at</th><th class="px-4 py-3 text-right">Unit cost ฿</th><th class="px-4 py-3 text-right">Stock value ฿</th><th class="px-4 py-3"></th></tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-for="s in supplies" :key="s.id" :class="s.lowStock ? 'bg-rose-50/60' : ''">
            <td class="px-4 py-3 font-medium text-gray-900">{{ s.name }} <span v-if="s.lowStock" class="ml-1 rounded-full bg-rose-100 px-2 py-0.5 text-[11px] font-semibold text-rose-700">reorder</span></td>
            <td class="px-4 py-3 text-right tabular-nums">{{ qty(s.stockQty) }} {{ s.unit }}</td>
            <td class="px-4 py-3 text-right tabular-nums text-gray-500">{{ s.reorderLevel ? qty(s.reorderLevel) : "—" }}</td>
            <td class="px-4 py-3 text-right tabular-nums">{{ money(s.unitCost) }}</td>
            <td class="px-4 py-3 text-right tabular-nums">{{ money(s.unitCost * Math.max(s.stockQty, 0)) }}</td>
            <td class="whitespace-nowrap px-4 py-3 text-right">
              <template v-if="canManage">
                <button type="button" class="mr-2 text-xs font-semibold text-navy-700 hover:underline" @click="openMove(s, 'usage')">Use</button>
                <button type="button" class="mr-2 text-xs font-semibold text-emerald-700 hover:underline" @click="openMove(s, 'purchase')">Buy</button>
                <button type="button" class="mr-2 text-xs font-semibold text-gray-600 hover:underline" @click="openMove(s, 'adjustment')">Adjust</button>
                <button type="button" class="text-xs text-gray-400 hover:underline" @click="toggleActive(s)">Archive</button>
              </template>
            </td>
          </tr>
          <tr v-if="supplies.length === 0"><td colspan="6" class="px-4 py-10 text-center text-gray-500">No supplies yet — add floor cleaner, glass spray, mops…</td></tr>
        </tbody>
      </table>

      <table v-else-if="tab === 'log'" class="min-w-full text-sm">
        <thead class="bg-gray-50 text-left text-xs font-semibold uppercase text-gray-500">
          <tr><th class="px-4 py-3">When</th><th class="px-4 py-3">Item</th><th class="px-4 py-3">Type</th><th class="px-4 py-3 text-right">Qty</th><th class="px-4 py-3 text-right">Cost ฿</th><th class="px-4 py-3">Site / job</th><th class="px-4 py-3">Note</th></tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-for="m in movements" :key="m.id">
            <td class="px-4 py-2 text-gray-500">{{ new Date(m.createdAt).toLocaleString() }}</td>
            <td class="px-4 py-2 font-medium">{{ m.supplyName }}</td>
            <td class="px-4 py-2">{{ kindLabel[m.kind] }}</td>
            <td class="px-4 py-2 text-right tabular-nums" :class="m.quantity < 0 ? 'text-rose-600' : 'text-emerald-700'">{{ m.quantity > 0 ? "+" : "" }}{{ qty(m.quantity) }} {{ m.unit }}</td>
            <td class="px-4 py-2 text-right tabular-nums">{{ money(m.cost) }}</td>
            <td class="px-4 py-2 text-gray-600">{{ m.siteName || "—" }}<span v-if="m.bookingNumber" class="text-gray-400"> · {{ m.bookingNumber }}</span></td>
            <td class="px-4 py-2 text-gray-500">{{ m.note }}</td>
          </tr>
          <tr v-if="movements.length === 0"><td colspan="7" class="px-4 py-10 text-center text-gray-500">Nothing recorded in this period.</td></tr>
        </tbody>
      </table>

      <table v-else class="min-w-full text-sm">
        <thead class="bg-gray-50 text-left text-xs font-semibold uppercase text-gray-500">
          <tr><th class="px-4 py-3">Site</th><th class="px-4 py-3">Customer</th><th class="px-4 py-3 text-right">Supply cost ฿</th><th class="px-4 py-3 text-right">Revenue ฿ (ex VAT)</th><th class="px-4 py-3 text-right">Supplies % of revenue</th></tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-for="s in siteCosts" :key="s.siteId">
            <td class="px-4 py-3 font-medium">{{ s.siteName }}</td>
            <td class="px-4 py-3 text-gray-600">{{ s.customerName }}</td>
            <td class="px-4 py-3 text-right tabular-nums">{{ money(s.supplyCost) }}</td>
            <td class="px-4 py-3 text-right tabular-nums">{{ money(s.revenue) }}</td>
            <td class="px-4 py-3 text-right tabular-nums" :class="s.costShare > 15 ? 'font-semibold text-rose-600' : ''">{{ s.revenue > 0 ? s.costShare.toFixed(1) + "%" : "—" }}</td>
          </tr>
          <tr v-if="siteCosts.length === 0"><td colspan="5" class="px-4 py-10 text-center text-gray-500">No site usage or invoices in this period.</td></tr>
        </tbody>
        <tfoot v-if="siteCosts.length" class="bg-gray-50"><tr><td colspan="2" class="px-4 py-3 text-right font-semibold">Total</td><td class="px-4 py-3 text-right font-bold tabular-nums">{{ money(totalSiteCost) }}</td><td colspan="2"></td></tr></tfoot>
      </table>
    </section>

    <div v-if="showNew" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="showNew = false">
      <div class="w-full max-w-md rounded-xl bg-white p-6 shadow-xl">
        <h3 class="text-base font-semibold">Add supply</h3>
        <div class="mt-4 grid grid-cols-2 gap-3">
          <label class="col-span-2 text-xs font-medium text-gray-600">Name<input v-model="newForm.name" :class="input" placeholder="Floor cleaner 5L" /></label>
          <label class="text-xs font-medium text-gray-600">Unit<select v-model="newForm.unit" :class="input"><option v-for="u in UNITS" :key="u" :value="u">{{ u }}</option></select></label>
          <label class="text-xs font-medium text-gray-600">Unit cost ฿<input v-model.number="newForm.unitCost" type="number" min="0" step="0.01" :class="input" /></label>
          <label class="text-xs font-medium text-gray-600">Opening stock<input v-model.number="newForm.stockQty" type="number" min="0" step="1" :class="input" /></label>
          <label class="text-xs font-medium text-gray-600">Reorder when at or below<input v-model.number="newForm.reorderLevel" type="number" min="0" step="1" :class="input" /></label>
        </div>
        <div class="mt-5 flex justify-end gap-2">
          <button type="button" class="rounded-lg border border-gray-200 px-4 py-2 text-sm" @click="showNew = false">Cancel</button>
          <button type="button" class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white" @click="saveNew">Add</button>
        </div>
      </div>
    </div>

    <div v-if="moveFor" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="moveFor = null">
      <div class="w-full max-w-md rounded-xl bg-white p-6 shadow-xl">
        <h3 class="text-base font-semibold">{{ kindLabel[moveForm.kind] }} — {{ moveFor.name }}</h3>
        <p class="text-xs text-gray-500">In stock: {{ qty(moveFor.stockQty) }} {{ moveFor.unit }}</p>
        <div class="mt-4 grid grid-cols-2 gap-3">
          <label class="text-xs font-medium text-gray-600">{{ moveForm.kind === "adjustment" ? "Change (+/−)" : "Quantity" }} ({{ moveFor.unit }})
            <input v-model.number="moveForm.quantity" type="number" step="0.5" :min="moveForm.kind === 'adjustment' ? undefined : 0.5" :class="input" />
          </label>
          <label v-if="moveForm.kind === 'purchase'" class="text-xs font-medium text-gray-600">Unit cost ฿<input v-model.number="moveForm.unitCost" type="number" min="0" step="0.01" :class="input" /></label>
          <template v-if="moveForm.kind === 'usage'">
            <div class="col-span-2 flex gap-4 text-xs">
              <label class="flex items-center gap-1"><input v-model="moveForm.target" type="radio" value="job" /> For a job</label>
              <label class="flex items-center gap-1"><input v-model="moveForm.target" type="radio" value="site" /> For a site</label>
            </div>
            <label v-if="moveForm.target === 'job'" class="col-span-2 text-xs font-medium text-gray-600">Job (last 14 days and upcoming)
              <select v-model="moveForm.bookingId" :class="input">
                <option value="" disabled>Select job</option>
                <option v-for="b in jobs" :key="b.id" :value="b.id">{{ b.bookingNumber }} · {{ b.customerName }} · {{ new Date(b.scheduledFor).toLocaleDateString() }}</option>
              </select>
            </label>
            <label v-else class="col-span-2 text-xs font-medium text-gray-600">Site
              <select v-model="moveForm.siteId" :class="input">
                <option value="" disabled>Select site</option>
                <option v-for="s in sites" :key="s.id" :value="s.id">{{ s.name }} — {{ s.address }}</option>
              </select>
            </label>
          </template>
          <label class="col-span-2 text-xs font-medium text-gray-600">Note {{ moveForm.kind === "adjustment" ? "*" : "" }}<input v-model="moveForm.note" :class="input" :placeholder="moveForm.kind === 'adjustment' ? 'Why? e.g. stock count, broken bottle' : ''" /></label>
        </div>
        <div class="mt-5 flex justify-end gap-2">
          <button type="button" class="rounded-lg border border-gray-200 px-4 py-2 text-sm" @click="moveFor = null">Cancel</button>
          <button type="button" :disabled="recording" class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" @click="saveMove">Save</button>
        </div>
      </div>
    </div>
  </div>
</template>
