<script setup lang="ts">
import { ref } from "vue";
import { getLeads } from "../../../lib/leads";
import { getCustomers } from "../../../lib/customers";
import { getBookings } from "../../../lib/bookings";
import { getInvoices } from "../../../lib/invoices";
import { getPayments } from "../../../lib/payments";
import { showToast } from "../../../lib/toast";

const exporting = ref<string | null>(null);

function toCsv(rows: Record<string, unknown>[]): string {
  if (rows.length === 0) return "";
  const headers = Object.keys(rows[0]!);
  const esc = (v: unknown) => {
    const s = v === null || v === undefined ? "" : String(v);
    return /[",\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
  };
  return [headers.join(","), ...rows.map((r) => headers.map((h) => esc(r[h])).join(","))].join("\n");
}

function download(name: string, csv: string) {
  const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  a.click();
  URL.revokeObjectURL(url);
}

async function exportLeads() {
  exporting.value = "leads";
  try {
    const rows = await getLeads({ limit: 1000 });
    if (rows.length === 0) {
      showToast("No leads to export", "error");
      return;
    }
    download(`leads-${new Date().toISOString().slice(0, 10)}.csv`, toCsv(rows as unknown as Record<string, unknown>[]));
    showToast(`Exported ${rows.length} leads`, "success");
  } catch (e) {
    showToast(e instanceof Error ? e.message : "Export failed", "error");
  } finally {
    exporting.value = null;
  }
}

async function exportCustomers() {
  exporting.value = "customers";
  try {
    const rows = await getCustomers({ limit: 1000 });
    if (rows.length === 0) {
      showToast("No customers to export", "error");
      return;
    }
    download(`customers-${new Date().toISOString().slice(0, 10)}.csv`, toCsv(rows as unknown as Record<string, unknown>[]));
    showToast(`Exported ${rows.length} customers`, "success");
  } catch (e) {
    showToast(e instanceof Error ? e.message : "Export failed", "error");
  } finally {
    exporting.value = null;
  }
}

async function exportBookings() {
  exporting.value = "bookings";
  try {
    const rows = await getBookings({ limit: 1000 });
    if (rows.length === 0) {
      showToast("No bookings to export", "error");
      return;
    }
    download(`bookings-${new Date().toISOString().slice(0, 10)}.csv`, toCsv(rows as unknown as Record<string, unknown>[]));
    showToast(`Exported ${rows.length} bookings`, "success");
  } catch (e) {
    showToast(e instanceof Error ? e.message : "Export failed", "error");
  } finally {
    exporting.value = null;
  }
}

async function exportInvoices() {
  exporting.value = "invoices";
  try {
    const rows = await getInvoices({ limit: 1000 });
    if (rows.length === 0) {
      showToast("No invoices to export", "error");
      return;
    }
    download(`invoices-${new Date().toISOString().slice(0, 10)}.csv`, toCsv(rows as unknown as Record<string, unknown>[]));
    showToast(`Exported ${rows.length} invoices`, "success");
  } catch (e) {
    showToast(e instanceof Error ? e.message : "Export failed", "error");
  } finally {
    exporting.value = null;
  }
}

async function exportPayments() {
  exporting.value = "payments";
  try {
    const rows = await getPayments({ limit: 1000 });
    if (rows.length === 0) {
      showToast("No payments to export", "error");
      return;
    }
    download(`payments-${new Date().toISOString().slice(0, 10)}.csv`, toCsv(rows as unknown as Record<string, unknown>[]));
    showToast(`Exported ${rows.length} payments`, "success");
  } catch (e) {
    showToast(e instanceof Error ? e.message : "Export failed", "error");
  } finally {
    exporting.value = null;
  }
}

const btn =
  "rounded-lg border border-gray-200 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100 disabled:opacity-50";
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-white p-6">
    <h2 class="text-base font-semibold text-gray-900">Export data</h2>
    <p class="mt-1 text-sm text-gray-500">
      Download up to 1,000 rows per file as CSV — for backups or migration.
      Import is not available yet; add rows via each module's UI.
    </p>
    <div class="mt-4 flex flex-wrap gap-2">
      <button type="button" :class="btn" :disabled="exporting !== null" @click="exportLeads">
        {{ exporting === "leads" ? "Exporting…" : "Export leads" }}
      </button>
      <button type="button" :class="btn" :disabled="exporting !== null" @click="exportCustomers">
        {{ exporting === "customers" ? "Exporting…" : "Export customers" }}
      </button>
      <button type="button" :disabled="exporting !== null" :class="btn" @click="exportBookings">
        {{ exporting === "bookings" ? "Exporting…" : "Export bookings" }}
      </button>
      <button type="button" :disabled="exporting !== null" :class="btn" @click="exportInvoices">
        {{ exporting === "invoices" ? "Exporting…" : "Export invoices" }}
      </button>
      <button type="button" :disabled="exporting !== null" :class="btn" @click="exportPayments">
        {{ exporting === "payments" ? "Exporting…" : "Export payments" }}
      </button>
    </div>
  </div>
</template>
