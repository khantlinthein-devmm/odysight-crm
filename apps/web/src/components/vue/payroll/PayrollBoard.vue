<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ApiError } from "../../../lib/api";
import { getSessionUser } from "../../../lib/auth";
import {
  PAY_TYPE_LABELS,
  downloadPayrollCsv,
  getPayRates,
  getPayroll,
  setPayRate,
  type PayRate,
  type PayType,
  type PayrollReport,
} from "../../../lib/payroll";
import { hasPermission } from "../../../lib/roles";
import { showToast } from "../../../lib/toast";

const canManage = hasPermission(getSessionUser()?.role, "payroll.manage");

function ymd(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}
const today = new Date();
const from = ref(ymd(new Date(today.getFullYear(), today.getMonth(), 1)));
const to = ref(ymd(today));

const report = ref<PayrollReport | null>(null);
const rates = ref<Map<number, PayRate>>(new Map());
const loading = ref(true);
const exporting = ref(false);

function money(v: number): string {
  return v.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

function errText(e: unknown, fallback: string): string {
  return e instanceof ApiError ? e.message : fallback;
}

async function load() {
  loading.value = true;
  try {
    const [rep, rs] = await Promise.all([getPayroll(from.value, to.value), getPayRates()]);
    report.value = rep;
    rates.value = new Map(rs.map((r) => [r.cleanerId, r]));
  } catch (e) {
    showToast(errText(e, "Failed to load payroll"), "error");
  } finally {
    loading.value = false;
  }
}

function presetThisMonth() {
  from.value = ymd(new Date(today.getFullYear(), today.getMonth(), 1));
  to.value = ymd(today);
  void load();
}
function presetLastMonth() {
  from.value = ymd(new Date(today.getFullYear(), today.getMonth() - 1, 1));
  to.value = ymd(new Date(today.getFullYear(), today.getMonth(), 0));
  void load();
}
function presetFirstHalf() {
  from.value = ymd(new Date(today.getFullYear(), today.getMonth(), 1));
  to.value = ymd(new Date(today.getFullYear(), today.getMonth(), 15));
  void load();
}

async function exportCsv() {
  exporting.value = true;
  try {
    await downloadPayrollCsv(from.value, to.value);
  } catch (e) {
    showToast(errText(e, "Export failed"), "error");
  } finally {
    exporting.value = false;
  }
}

const missingRates = computed(() => report.value?.lines.filter((l) => !l.hasRate).length ?? 0);
const openDays = computed(() => report.value?.lines.reduce((n, l) => n + l.openDays, 0) ?? 0);

// Rate editor
const editing = ref<{ cleanerId: number; name: string } | null>(null);
const form = ref<{ payType: PayType; rate: number; otMultiplier: number; standardHours: number }>({
  payType: "daily",
  rate: 0,
  otMultiplier: 1.5,
  standardHours: 8,
});
const saving = ref(false);

function openRate(cleanerId: number, name: string) {
  const r = rates.value.get(cleanerId);
  form.value = r
    ? { payType: r.payType, rate: r.rate, otMultiplier: r.otMultiplier, standardHours: r.standardHours }
    : { payType: "daily", rate: 0, otMultiplier: 1.5, standardHours: 8 };
  editing.value = { cleanerId, name };
}

async function saveRate() {
  if (!editing.value) return;
  saving.value = true;
  try {
    await setPayRate(editing.value.cleanerId, {
      payType: form.value.payType,
      rate: Number(form.value.rate),
      otMultiplier: Number(form.value.otMultiplier),
      standardHours: Number(form.value.standardHours),
    });
    showToast("Pay rate saved", "success");
    editing.value = null;
    await load();
  } catch (e) {
    showToast(errText(e, "Failed to save pay rate"), "error");
  } finally {
    saving.value = false;
  }
}

function rateLabel(r: { payType: PayType; rate: number }): string {
  return `${money(r.rate)} ${PAY_TYPE_LABELS[r.payType].toLowerCase()}`;
}

onMounted(load);

const input =
  "w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500";
</script>

<template>
  <div class="space-y-4">
    <section class="rounded-xl border border-gray-200 bg-white p-5">
      <div class="flex flex-wrap items-end gap-3">
        <div>
          <h2 class="text-base font-semibold text-gray-900">Cleaner payroll</h2>
          <p class="text-xs text-gray-500">From attendance check-in/out and completed jobs. Days without a check-out are not paid until fixed.</p>
        </div>
        <div class="ml-auto flex flex-wrap items-end gap-2">
          <label class="text-xs text-gray-500">From<input v-model="from" type="date" :class="input" /></label>
          <label class="text-xs text-gray-500">To<input v-model="to" type="date" :class="input" /></label>
          <button type="button" class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700" @click="load">Show</button>
          <button type="button" :disabled="exporting" class="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50" @click="exportCsv">
            {{ exporting ? "Exporting…" : "Export CSV" }}
          </button>
        </div>
      </div>
      <div class="mt-3 flex flex-wrap gap-2 text-xs">
        <button type="button" class="rounded-full bg-gray-100 px-3 py-1 font-medium text-gray-700 hover:bg-gray-200" @click="presetThisMonth">This month</button>
        <button type="button" class="rounded-full bg-gray-100 px-3 py-1 font-medium text-gray-700 hover:bg-gray-200" @click="presetFirstHalf">1st–15th</button>
        <button type="button" class="rounded-full bg-gray-100 px-3 py-1 font-medium text-gray-700 hover:bg-gray-200" @click="presetLastMonth">Last month</button>
      </div>
      <div v-if="report && (missingRates || openDays)" class="mt-3 space-y-1">
        <p v-if="missingRates" class="rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-800">
          {{ missingRates }} cleaner(s) have no pay rate — their pay shows 0. Set a rate with “Set rate”.
        </p>
        <p v-if="openDays" class="rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-800">
          {{ openDays }} day(s) have a check-in without a check-out and are not paid. Fix them on the Attendance page.
        </p>
      </div>
    </section>

    <section class="overflow-x-auto rounded-xl border border-gray-200 bg-white">
      <div v-if="loading" class="px-6 py-12 text-center text-sm text-gray-500">Loading payroll…</div>
      <table v-else-if="report" class="min-w-full text-sm">
        <thead class="bg-gray-50 text-left text-xs font-semibold uppercase tracking-wide text-gray-500">
          <tr>
            <th class="px-4 py-3">Cleaner</th>
            <th class="px-4 py-3">Rate</th>
            <th class="px-4 py-3 text-right">Days</th>
            <th class="px-4 py-3 text-right">Hours</th>
            <th class="px-4 py-3 text-right">OT h</th>
            <th class="px-4 py-3 text-right">Jobs</th>
            <th class="px-4 py-3 text-right">Base</th>
            <th class="px-4 py-3 text-right">OT pay</th>
            <th class="px-4 py-3 text-right">Total ฿</th>
            <th class="px-4 py-3"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-for="l in report.lines" :key="l.cleanerId">
            <td class="px-4 py-3 font-medium text-gray-900">{{ l.name }}</td>
            <td class="px-4 py-3 text-gray-600">
              <span v-if="l.hasRate">{{ rateLabel(l) }}</span>
              <span v-else class="text-amber-700">not set</span>
            </td>
            <td class="px-4 py-3 text-right tabular-nums">
              {{ l.daysWorked }}<span v-if="l.openDays" class="ml-1 text-xs text-amber-700" :title="`${l.openDays} day(s) without check-out`">+{{ l.openDays }}?</span>
            </td>
            <td class="px-4 py-3 text-right tabular-nums">{{ l.hours.toFixed(2) }}</td>
            <td class="px-4 py-3 text-right tabular-nums">{{ l.otHours.toFixed(2) }}</td>
            <td class="px-4 py-3 text-right tabular-nums">{{ l.jobs }}</td>
            <td class="px-4 py-3 text-right tabular-nums">{{ money(l.basePay) }}</td>
            <td class="px-4 py-3 text-right tabular-nums">{{ money(l.otPay) }}</td>
            <td class="px-4 py-3 text-right font-semibold tabular-nums text-gray-900">{{ money(l.total) }}</td>
            <td class="px-4 py-3 text-right">
              <button v-if="canManage" type="button" class="text-xs font-semibold text-navy-700 hover:underline" @click="openRate(l.cleanerId, l.name)">Set rate</button>
            </td>
          </tr>
          <tr v-if="report.lines.length === 0">
            <td colspan="10" class="px-4 py-10 text-center text-gray-500">No cleaners yet.</td>
          </tr>
        </tbody>
        <tfoot class="bg-gray-50">
          <tr>
            <td colspan="8" class="px-4 py-3 text-right text-sm font-semibold text-gray-700">Total {{ report.from }} – {{ report.to }}</td>
            <td class="px-4 py-3 text-right text-base font-bold tabular-nums text-gray-900">{{ money(report.total) }}</td>
            <td></td>
          </tr>
        </tfoot>
      </table>
    </section>

    <div v-if="editing" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="editing = null">
      <div class="w-full max-w-md rounded-xl bg-white p-6 shadow-xl">
        <h3 class="text-base font-semibold text-gray-900">Pay rate — {{ editing.name }}</h3>
        <div class="mt-4 grid grid-cols-2 gap-3">
          <label class="col-span-2 text-xs font-medium text-gray-600">Paid
            <select v-model="form.payType" :class="input">
              <option v-for="(label, key) in PAY_TYPE_LABELS" :key="key" :value="key">{{ label }}</option>
            </select>
          </label>
          <label class="col-span-2 text-xs font-medium text-gray-600">Rate (฿)
            <input v-model.number="form.rate" type="number" min="0" step="0.01" :class="input" />
          </label>
          <label class="text-xs font-medium text-gray-600">Standard hours / day
            <input v-model.number="form.standardHours" type="number" min="1" max="24" step="0.5" :class="input" />
          </label>
          <label class="text-xs font-medium text-gray-600">Overtime ×
            <input v-model.number="form.otMultiplier" type="number" min="1" max="5" step="0.25" :class="input" :disabled="form.payType === 'per_job'" />
          </label>
        </div>
        <p class="mt-2 text-xs text-gray-500">
          Hours beyond the standard day are overtime (not for per-job pay). Monthly salaries are prorated over 30-day months.
        </p>
        <div class="mt-5 flex justify-end gap-2">
          <button type="button" class="rounded-lg border border-gray-200 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50" @click="editing = null">Cancel</button>
          <button type="button" :disabled="saving" class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50" @click="saveRate">
            {{ saving ? "Saving…" : "Save" }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
