<script setup lang="ts">
import { computed, onMounted, reactive, ref, useId, watch } from "vue";
import {
  createExpense,
  deleteExpense,
  getExpenses,
  updateExpense,
  type Expense,
} from "../../../lib/expenses";
import { getPayments, type Payment } from "../../../lib/payments";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";
import {
  currencyCode,
  formatMoney as formatMoneyFull,
  getWorkspaceSettings,
} from "../../../lib/settings";
import { showToast } from "../../../lib/toast";
import DonutChart from "../reports/DonutChart.vue";
import TrendChart from "../reports/TrendChart.vue";
import ConfirmDialog from "../ui/ConfirmDialog.vue";
import { useModalA11y } from "../ui/useModalA11y";

const role = getSessionUser()?.role;
const canRead = computed(() => hasPermission(role, "expenses.read"));
const canManage = computed(() => hasPermission(role, "expenses.manage"));

type Preset = "month" | "quarter" | "year" | "custom";

const CATEGORIES = [
  "Supplies",
  "Transport",
  "Equipment",
  "Salary",
  "Rent",
  "Utilities",
  "Food",
  "Other",
];

const CATEGORY_COLORS = [
  "#f43f5e",
  "#f59e0b",
  "#8b5cf6",
  "#0ea5e9",
  "#10b981",
  "#2563eb",
  "#ec4899",
  "#64748b",
];

function toISODate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

function monthStart(offsetMonths: number): string {
  const now = new Date();
  const d = new Date(now.getFullYear(), now.getMonth() - offsetMonths, 1);
  return toISODate(d);
}

function yearStart(): string {
  return `${new Date().getFullYear()}-01-01`;
}

const preset = ref<Preset>("month");
const from = ref(monthStart(0));
const to = ref(toISODate(new Date()));

function setPreset(p: Preset) {
  preset.value = p;
  const today = toISODate(new Date());
  if (p === "month") {
    from.value = monthStart(0);
    to.value = today;
  } else if (p === "quarter") {
    from.value = monthStart(2);
    to.value = today;
  } else if (p === "year") {
    from.value = yearStart();
    to.value = today;
  }
}

const expenses = ref<Expense[]>([]);
const income = ref<Payment[]>([]);
const loading = ref(true);
const search = ref("");
const categoryFilter = ref("");

function monthBuckets(fromISO: string, toISO: string): string[] {
  const out: string[] = [];
  if (!/^\d{4}-\d{2}-\d{2}$/.test(fromISO) || !/^\d{4}-\d{2}-\d{2}$/.test(toISO)) return out;
  let [y, m] = [Number(fromISO.slice(0, 4)), Number(fromISO.slice(5, 7))];
  const [ey, em] = [Number(toISO.slice(0, 4)), Number(toISO.slice(5, 7))];
  let guard = 0;
  while ((y < ey || (y === ey && m <= em)) && guard < 36) {
    out.push(`${y}-${String(m).padStart(2, "0")}`);
    m += 1;
    if (m > 12) {
      m = 1;
      y += 1;
    }
    guard += 1;
  }
  return out;
}

const buckets = computed(() => monthBuckets(from.value, to.value));

const incomeByMonth = computed(() =>
  buckets.value.map((b) =>
    income.value
      .filter((p) => p.createdAt.slice(0, 7) === b)
      .reduce((s, p) => s + p.amount, 0),
  ),
);

const expensesByMonth = computed(() =>
  buckets.value.map((b) =>
    expenses.value
      .filter((e) => e.spentOn.slice(0, 7) === b)
      .reduce((s, e) => s + e.amount, 0),
  ),
);

const totalIncome = computed(() => income.value.reduce((s, p) => s + p.amount, 0));
const totalExpenses = computed(() => expenses.value.reduce((s, e) => s + e.amount, 0));
const profit = computed(() => totalIncome.value - totalExpenses.value);

const categorySegments = computed(() => {
  const sums = new Map<string, number>();
  for (const e of expenses.value) {
    sums.set(e.category, (sums.get(e.category) ?? 0) + e.amount);
  }
  return [...sums.entries()]
    .sort((a, b) => b[1] - a[1])
    .map(([label, value], i) => ({
      label,
      value,
      color: CATEGORY_COLORS[i % CATEGORY_COLORS.length]!,
    }));
});

function formatCompact(value: number): string {
  const code = currencyCode();
  if (value >= 1000) return `${code} ${(value / 1000).toFixed(0)}k`;
  return `${code} ${value}`;
}

function formatDate(iso: string): string {
  return new Date(iso.length <= 10 ? `${iso}T00:00:00` : iso).toLocaleDateString(
    undefined,
    { day: "2-digit", month: "short", year: "numeric" },
  );
}

async function fetchAll() {
  if (!canRead.value) {
    loading.value = false;
    return;
  }
  loading.value = true;
  try {
    const [exp, inc] = await Promise.all([
      getExpenses({
        from: from.value || undefined,
        to: to.value || undefined,
        category: categoryFilter.value || undefined,
        search: search.value || undefined,
        limit: 1000,
      }),
      getPayments({ status: "paid", from: from.value || undefined, to: to.value || undefined, limit: 1000 }),
    ]);
    expenses.value = exp;
    income.value = inc;
  } catch {
    showToast("Failed to load finance data", "error");
  } finally {
    loading.value = false;
  }
}

let searchTimer: ReturnType<typeof setTimeout> | null = null;
watch([search, categoryFilter], () => {
  if (searchTimer) clearTimeout(searchTimer);
  searchTimer = setTimeout(fetchAll, 300);
});
watch([from, to], () => {
  if (preset.value !== "custom" || (from.value && to.value)) void fetchAll();
});

onMounted(async () => {
  try {
    await getWorkspaceSettings();
  } catch {
    /* fall back to defaults */
  }
  await fetchAll();
});

// ---- Expense modal form ----
const titleId = useId();
const showForm = ref(false);
const editing = ref<Expense | null>(null);
const saving = ref(false);
const submitted = ref(false);
const form = reactive({ spentOn: "", category: "", amount: 0, note: "" });
const { container: modalContainer } = useModalA11y(() => closeForm());

function openCreate() {
  editing.value = null;
  submitted.value = false;
  form.spentOn = toISODate(new Date());
  form.category = "";
  form.amount = 0;
  form.note = "";
  showForm.value = true;
}

function openEdit(e: Expense) {
  editing.value = e;
  submitted.value = false;
  form.spentOn = e.spentOn;
  form.category = e.category;
  form.amount = e.amount;
  form.note = e.note;
  showForm.value = true;
}

function closeForm() {
  showForm.value = false;
  editing.value = null;
}

const formErrors = computed(() => {
  const e: Partial<Record<"spentOn" | "category" | "amount", string>> = {};
  if (!form.spentOn) e.spentOn = "Date is required";
  if (!form.category.trim()) e.category = "Category is required";
  if (!form.amount || Number(form.amount) <= 0) e.amount = "Amount must be greater than zero";
  return e;
});

const formValid = computed(() => Object.keys(formErrors.value).length === 0);

async function handleSave() {
  submitted.value = true;
  if (!formValid.value) return;
  saving.value = true;
  try {
    const payload = {
      spentOn: form.spentOn,
      category: form.category.trim(),
      amount: Number(form.amount),
      note: form.note.trim(),
    };
    if (editing.value) {
      const updated = await updateExpense(editing.value.id, payload);
      expenses.value = expenses.value.map((e) => (e.id === updated.id ? updated : e));
      showToast("Expense updated");
    } else {
      const created = await createExpense(payload);
      expenses.value = [created, ...expenses.value];
      showToast("Expense added");
    }
    closeForm();
  } catch {
    showToast("Failed to save expense", "error");
  } finally {
    saving.value = false;
  }
}

// ---- Delete ----
const pendingDelete = ref<Expense | null>(null);
const deleting = ref(false);

async function handleDelete(e: Expense) {
  deleting.value = true;
  try {
    await deleteExpense(e.id);
    expenses.value = expenses.value.filter((x) => x.id !== e.id);
    showToast("Expense deleted");
  } catch {
    showToast("Failed to delete expense", "error");
  } finally {
    deleting.value = false;
    pendingDelete.value = null;
  }
}

const inputClass =
  "w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500";

const presetTabs: { key: Preset; label: string }[] = [
  { key: "month", label: "This month" },
  { key: "quarter", label: "Last quarter" },
  { key: "year", label: "This year" },
  { key: "custom", label: "Custom" },
];
</script>

<template>
  <div
    v-if="!canRead"
    class="rounded-3xl bg-white p-16 text-center shadow-sm ring-1 ring-gray-100"
  >
    <p class="text-sm font-medium text-gray-900">
      You do not have permission to view finance.
    </p>
  </div>

  <div v-else>
    <!-- Range presets -->
    <div class="flex flex-wrap items-center gap-2">
      <div class="flex rounded-2xl bg-white p-1 shadow-sm ring-1 ring-gray-100">
        <button
          v-for="t in presetTabs"
          :key="t.key"
          type="button"
          :class="[
            'rounded-xl px-4 py-2 text-sm font-medium transition',
            preset === t.key
              ? 'bg-navy-600 text-white shadow-sm'
              : 'text-gray-600 hover:bg-gray-50',
          ]"
          @click="setPreset(t.key)"
        >
          {{ t.label }}
        </button>
      </div>
      <div v-if="preset === 'custom'" class="flex items-center gap-2 rounded-2xl bg-white px-3 py-2 shadow-sm ring-1 ring-gray-100">
        <input v-model="from" type="date" :class="inputClass" class="!w-auto" aria-label="From date" />
        <span class="text-sm text-gray-400">→</span>
        <input v-model="to" type="date" :class="inputClass" class="!w-auto" aria-label="To date" />
      </div>
      <p class="ml-auto text-xs text-gray-400">{{ from }} → {{ to }}</p>
    </div>

    <div
      v-if="loading"
      class="mt-4 rounded-3xl bg-white p-8 shadow-sm ring-1 ring-gray-100"
    >
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div v-for="i in 3" :key="i" class="rounded-2xl bg-gray-50 p-6">
          <div class="h-10 w-10 animate-pulse rounded-xl bg-gray-200"></div>
          <div class="mt-4 h-7 w-24 animate-pulse rounded-lg bg-gray-200"></div>
        </div>
      </div>
    </div>

    <template v-else>
      <!-- Hero summary cards -->
      <div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div class="rounded-3xl bg-white p-5 shadow-sm ring-1 ring-gray-100">
          <div class="flex items-center justify-between">
            <span class="flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-emerald-400 to-green-600 text-white shadow-sm">
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
              </svg>
            </span>
          </div>
          <p class="mt-4 text-[26px] font-semibold leading-none tracking-tight text-gray-900">
            {{ formatMoneyFull(totalIncome) }}
          </p>
          <p class="mt-1.5 text-sm font-medium text-emerald-700">Income</p>
          <p class="text-xs text-gray-400">{{ income.length }} paid payments</p>
        </div>

        <div class="rounded-3xl bg-white p-5 shadow-sm ring-1 ring-gray-100">
          <div class="flex items-center justify-between">
            <span class="flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-rose-400 to-red-600 text-white shadow-sm">
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M17 9V7a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-6a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2z" />
              </svg>
            </span>
          </div>
          <p class="mt-4 text-[26px] font-semibold leading-none tracking-tight text-gray-900">
            {{ formatMoneyFull(totalExpenses) }}
          </p>
          <p class="mt-1.5 text-sm font-medium text-rose-700">Expenses</p>
          <p class="text-xs text-gray-400">{{ expenses.length }} expense records</p>
        </div>

        <div class="rounded-3xl bg-white p-5 shadow-sm ring-1 ring-gray-100">
          <div class="flex items-center justify-between">
            <span class="flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-navy-500 to-blue-700 text-white shadow-sm">
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
            </span>
          </div>
          <p
            class="mt-4 text-[26px] font-semibold leading-none tracking-tight"
            :class="profit < 0 ? 'text-red-600' : 'text-gray-900'"
          >
            {{ formatMoneyFull(profit) }}
          </p>
          <p class="mt-1.5 text-sm font-medium text-gray-700">Profit</p>
          <p class="text-xs text-gray-400">Income minus expenses</p>
        </div>
      </div>

      <!-- Monthly trends -->
      <div class="mt-4 grid grid-cols-1 gap-4 xl:grid-cols-2">
        <section class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7">
          <h2 class="font-semibold tracking-tight text-gray-900">Monthly income</h2>
          <p class="text-xs text-gray-400">Paid payments per month</p>
          <TrendChart
            class="mt-4"
            :values="incomeByMonth"
            :labels="buckets"
            :format-value="formatCompact"
            line-color="#10b981"
          />
        </section>
        <section class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7">
          <h2 class="font-semibold tracking-tight text-gray-900">Monthly expenses</h2>
          <p class="text-xs text-gray-400">Recorded spend per month</p>
          <TrendChart
            class="mt-4"
            :values="expensesByMonth"
            :labels="buckets"
            :format-value="formatCompact"
            line-color="#f43f5e"
          />
        </section>
      </div>

      <!-- Expenses by category -->
      <section class="mt-4 rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7">
        <h2 class="font-semibold tracking-tight text-gray-900">Expenses by category</h2>
        <p class="text-xs text-gray-400">Spend mix in selected range</p>
        <p v-if="categorySegments.length === 0" class="mt-5 text-sm text-gray-400">
          No expenses in this range yet.
        </p>
        <DonutChart
          v-else
          class="mt-5"
          :segments="categorySegments"
          :center-top="formatMoneyFull(totalExpenses)"
          center-bottom="total expenses"
        />
      </section>

      <!-- Expense table -->
      <section class="mt-4 rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 class="font-semibold tracking-tight text-gray-900">Expenses</h2>
            <p class="text-xs text-gray-400">Track and manage spend</p>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <select v-model="categoryFilter" :class="inputClass" class="!w-auto" aria-label="Filter by category">
              <option value="">All categories</option>
              <option v-for="c in CATEGORIES" :key="c" :value="c">{{ c }}</option>
            </select>
            <input
              v-model="search"
              type="text"
              placeholder="Search expenses…"
              :class="inputClass"
              class="!w-48"
            />
            <button
              v-if="canManage"
              type="button"
              class="rounded-xl bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700"
              @click="openCreate"
            >
              + Add expense
            </button>
          </div>
        </div>

        <div v-if="expenses.length === 0" class="py-12 text-center">
          <p class="text-sm font-medium text-gray-900">No expenses found</p>
          <p class="mt-1 text-sm text-gray-500">Add your first expense to start tracking spend.</p>
        </div>

        <div v-else class="mt-4 overflow-x-auto">
          <table class="w-full min-w-[640px] text-left text-sm">
            <thead>
              <tr class="text-xs uppercase tracking-wide text-gray-400">
                <th class="pb-2 pr-4 font-medium">Date</th>
                <th class="pb-2 pr-4 font-medium">Category</th>
                <th class="pb-2 pr-4 font-medium">Note</th>
                <th class="pb-2 pr-4 text-right font-medium">Amount</th>
                <th v-if="canManage" class="pb-2 text-right font-medium">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100">
              <tr v-for="e in expenses" :key="e.id" class="hover:bg-gray-50/60">
                <td class="whitespace-nowrap py-3 pr-4 text-gray-900">{{ formatDate(e.spentOn) }}</td>
                <td class="py-3 pr-4">
                  <span class="inline-block rounded-full bg-navy-50 px-2.5 py-1 text-xs font-medium text-navy-700 ring-1 ring-navy-100">
                    {{ e.category }}
                  </span>
                </td>
                <td class="max-w-[280px] truncate py-3 pr-4 text-gray-600">{{ e.note || "—" }}</td>
                <td class="whitespace-nowrap py-3 pr-4 text-right font-semibold tabular-nums text-gray-900">
                  {{ formatMoneyFull(e.amount) }}
                </td>
                <td v-if="canManage" class="whitespace-nowrap py-3 text-right">
                  <button
                    type="button"
                    class="rounded-lg px-2 py-1 text-sm font-medium text-navy-600 hover:bg-navy-50"
                    @click="openEdit(e)"
                  >
                    Edit
                  </button>
                  <button
                    type="button"
                    class="rounded-lg px-2 py-1 text-sm font-medium text-red-600 hover:bg-red-50"
                    @click="pendingDelete = e"
                  >
                    Delete
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <!-- Income table (read-only) -->
      <section class="mt-4 rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-100 sm:p-7">
        <h2 class="font-semibold tracking-tight text-gray-900">Income</h2>
        <p class="text-xs text-gray-400">Paid payments in selected range · read-only</p>
        <div v-if="income.length === 0" class="py-12 text-center">
          <p class="text-sm font-medium text-gray-900">No income in this range</p>
          <p class="mt-1 text-sm text-gray-500">Paid payments will appear here.</p>
        </div>
        <div v-else class="mt-4 overflow-x-auto">
          <table class="w-full min-w-[640px] text-left text-sm">
            <thead>
              <tr class="text-xs uppercase tracking-wide text-gray-400">
                <th class="pb-2 pr-4 font-medium">Date</th>
                <th class="pb-2 pr-4 font-medium">Invoice</th>
                <th class="pb-2 pr-4 font-medium">Customer</th>
                <th class="pb-2 text-right font-medium">Amount</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100">
              <tr v-for="p in income" :key="p.id" class="hover:bg-gray-50/60">
                <td class="whitespace-nowrap py-3 pr-4 text-gray-900">{{ formatDate(p.createdAt) }}</td>
                <td class="whitespace-nowrap py-3 pr-4 font-mono text-xs text-gray-500">{{ p.invoiceNumber }}</td>
                <td class="py-3 pr-4 text-gray-600">{{ p.customerName }}</td>
                <td class="whitespace-nowrap py-3 text-right font-semibold tabular-nums text-emerald-700">
                  {{ formatMoneyFull(p.amount) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>

    <!-- Expense modal -->
    <div
      v-if="showForm"
      ref="modalContainer"
      role="dialog"
      aria-modal="true"
      :aria-labelledby="titleId"
      class="fixed inset-0 z-50 flex items-center justify-center p-4"
    >
      <div class="absolute inset-0 bg-black/50" @click="closeForm"></div>
      <div class="relative w-full max-w-lg rounded-xl bg-white shadow-xl">
        <div class="border-b border-gray-200 px-6 py-4">
          <h2 :id="titleId" class="text-base font-semibold text-gray-900">
            {{ editing ? "Edit expense" : "New expense" }}
          </h2>
        </div>
        <form @submit.prevent="handleSave">
          <div class="grid grid-cols-1 gap-4 px-6 py-5 sm:grid-cols-2">
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700" for="e-spentOn">Date</label>
              <input
                id="e-spentOn"
                v-model="form.spentOn"
                type="date"
                :class="[inputClass, submitted && formErrors.spentOn ? 'border-red-300 focus:border-red-500 focus:ring-red-500' : '']"
              />
              <p v-if="submitted && formErrors.spentOn" class="mt-1 text-xs text-red-600">
                {{ formErrors.spentOn }}
              </p>
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700" for="e-category">Category</label>
              <input
                id="e-category"
                v-model="form.category"
                type="text"
                list="e-category-list"
                placeholder="e.g. Supplies"
                :class="[inputClass, submitted && formErrors.category ? 'border-red-300 focus:border-red-500 focus:ring-red-500' : '']"
              />
              <datalist id="e-category-list">
                <option v-for="c in CATEGORIES" :key="c" :value="c" />
              </datalist>
              <p v-if="submitted && formErrors.category" class="mt-1 text-xs text-red-600">
                {{ formErrors.category }}
              </p>
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700" for="e-amount">Amount</label>
              <input
                id="e-amount"
                v-model.number="form.amount"
                type="number"
                min="0"
                step="0.01"
                placeholder="0.00"
                :class="[inputClass, submitted && formErrors.amount ? 'border-red-300 focus:border-red-500 focus:ring-red-500' : '']"
              />
              <p v-if="submitted && formErrors.amount" class="mt-1 text-xs text-red-600">
                {{ formErrors.amount }}
              </p>
            </div>
            <div>
              <label class="mb-1 block text-sm font-medium text-gray-700" for="e-note">Note</label>
              <input
                id="e-note"
                v-model="form.note"
                type="text"
                placeholder="Optional note"
                :class="inputClass"
              />
            </div>
          </div>
          <div class="flex justify-end gap-3 border-t border-gray-200 px-6 py-4">
            <button
              type="button"
              class="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100"
              @click="closeForm"
            >
              Cancel
            </button>
            <button
              type="submit"
              :disabled="saving"
              class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50"
            >
              {{ editing ? "Save changes" : "Add expense" }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <ConfirmDialog
      :open="pendingDelete !== null"
      title="Delete expense?"
      :busy="deleting"
      @confirm="pendingDelete && handleDelete(pendingDelete)"
      @cancel="pendingDelete = null"
    >
      <p class="text-sm text-gray-500">This permanently removes this expense record.</p>
    </ConfirmDialog>

    <p class="mt-4 text-xs text-gray-400">
      Live data from the Go API · GET /api/v1/expenses · GET /api/v1/payments?status=paid
    </p>
  </div>
</template>
