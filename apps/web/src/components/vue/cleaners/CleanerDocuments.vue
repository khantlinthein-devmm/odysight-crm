<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from "vue";
import {
  DOCUMENT_TYPE_LABELS,
  FILE_ONLY_TYPES,
  createDocument,
  deleteDocument,
  fetchDocumentFile,
  getCleanerDocuments,
  getDocumentStatuses,
  revealDocument,
  updateDocument,
  type CleanerDocument,
  type DocumentStatus,
  type DocumentType,
  type ExpiryStatus,
} from "../../../lib/cleaner-documents";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";
import { showToast } from "../../../lib/toast";
import ConfirmDialog from "../ui/ConfirmDialog.vue";
import DocumentPreview from "./DocumentPreview.vue";

const props = defineProps<{ cleanerId: number }>();

const role = getSessionUser()?.role;
const canRead = hasPermission(role, "cleaner_documents.read");
const canManage = hasPermission(role, "cleaner_documents.manage");

const docs = ref<CleanerDocument[]>([]);
const statuses = ref<DocumentStatus[]>([]);
const loading = ref(true);
const unavailable = ref("");
const revealed = ref<Record<number, string>>({});
const formOpen = ref(false);
const editingId = ref<number | null>(null);
const saving = ref(false);
const deletingId = ref<number | null>(null);
const fileInput = ref<HTMLInputElement | null>(null);
const previewingId = ref<number | null>(null);
const preview = ref<{ url: string; contentType: string; title: string; fileName: string } | null>(null);

const emptyForm = () => ({
  type: "passport" as DocumentType,
  number: "",
  issueDate: "",
  expiryDate: "",
  notes: "",
  file: null as File | null,
});
const form = reactive(emptyForm());

const statusStyles: Record<ExpiryStatus, string> = {
  none: "bg-gray-100 text-gray-600",
  valid: "bg-green-50 text-green-700",
  expiring: "bg-amber-50 text-amber-700",
  expired: "bg-red-50 text-red-700",
};

function expiryLabel(status: ExpiryStatus, days: number | null): string {
  if (status === "none") return "No expiry";
  if (status === "expired") return `Expired ${Math.abs(days ?? 0)}d ago`;
  if (status === "expiring") return days === 0 ? "Expires today" : `Expires in ${days}d`;
  return "Valid";
}

function formatDate(value: string | null): string {
  if (!value) return "—";
  const [y, m, d] = value.split("-");
  return `${d}/${m}/${y}`;
}

const fileOnly = computed(() => FILE_ONLY_TYPES.includes(form.type));

function isFileOnly(type: DocumentType): boolean {
  return FILE_ONLY_TYPES.includes(type);
}

function formatSize(bytes: number): string {
  if (bytes >= 1 << 20) return `${(bytes / (1 << 20)).toFixed(1)} MB`;
  return `${Math.max(1, Math.round(bytes / 1024))} KB`;
}

const rows = computed(() => (canRead ? docs.value : statuses.value));

async function load() {
  loading.value = true;
  unavailable.value = "";
  try {
    if (canRead) docs.value = await getCleanerDocuments(props.cleanerId);
    else statuses.value = await getDocumentStatuses(props.cleanerId);
  } catch (err) {
    unavailable.value = err instanceof Error && err.message ? err.message : "Failed to load documents";
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  Object.assign(form, emptyForm());
  editingId.value = null;
  formOpen.value = true;
}

function openEdit(doc: CleanerDocument) {
  Object.assign(form, emptyForm(), {
    type: doc.type,
    number: revealed.value[doc.id] ?? "",
    issueDate: doc.issueDate ?? "",
    expiryDate: doc.expiryDate ?? "",
    notes: doc.notes,
  });
  editingId.value = doc.id;
  formOpen.value = true;
}

function closeForm() {
  formOpen.value = false;
  editingId.value = null;
  if (fileInput.value) fileInput.value.value = "";
}

function onFile(e: Event) {
  form.file = (e.target as HTMLInputElement).files?.[0] ?? null;
}

async function submit() {
  if (editingId.value === null && fileOnly.value && !form.file) {
    showToast(`Choose the ${DOCUMENT_TYPE_LABELS[form.type].toLowerCase()} file to upload`, "error");
    return;
  }
  saving.value = true;
  try {
    if (editingId.value === null) {
      await createDocument(
        props.cleanerId,
        fileOnly.value
          ? { type: form.type, notes: form.notes }
          : { type: form.type, number: form.number, issueDate: form.issueDate, expiryDate: form.expiryDate, notes: form.notes },
        form.file,
      );
      showToast("Document added", "success");
    } else {
      const id = editingId.value;
      const fields: Record<string, string> = {
        type: form.type,
        issueDate: fileOnly.value ? "" : form.issueDate,
        expiryDate: fileOnly.value ? "" : form.expiryDate,
        notes: form.notes,
      };
      // The number is only sent when the user typed one, so editing dates
      // never requires revealing (and auditing) the current number.
      if (form.number.trim() && !fileOnly.value) fields.number = form.number;
      await updateDocument(id, fields);
      delete revealed.value[id];
      showToast("Document updated", "success");
    }
    closeForm();
    await load();
  } catch (err) {
    showToast(err instanceof Error && err.message ? err.message : "Failed to save document", "error");
  } finally {
    saving.value = false;
  }
}

async function toggleReveal(doc: CleanerDocument) {
  if (revealed.value[doc.id] !== undefined) {
    delete revealed.value[doc.id];
    return;
  }
  try {
    const full = await revealDocument(doc.id);
    revealed.value[doc.id] = full.number;
  } catch (err) {
    showToast(err instanceof Error && err.message ? err.message : "Failed to show number", "error");
  }
}

async function previewFile(doc: CleanerDocument) {
  previewingId.value = doc.id;
  try {
    const blob = await fetchDocumentFile(doc.id);
    closePreview();
    const label = DOCUMENT_TYPE_LABELS[doc.type];
    preview.value = {
      url: URL.createObjectURL(blob),
      contentType: blob.type || doc.contentType,
      title: doc.originalName ? `${label} · ${doc.originalName}` : label,
      fileName: doc.originalName || `${doc.type}-${doc.id}`,
    };
  } catch (err) {
    showToast(err instanceof Error && err.message ? err.message : "Failed to open file", "error");
  } finally {
    previewingId.value = null;
  }
}

// Preview the picked file before it is uploaded (it never leaves the browser).
function previewLocal() {
  if (!form.file) return;
  closePreview();
  preview.value = {
    url: URL.createObjectURL(form.file),
    contentType: form.file.type,
    title: `${DOCUMENT_TYPE_LABELS[form.type]} · ${form.file.name} (not uploaded yet)`,
    fileName: form.file.name,
  };
}

function closePreview() {
  if (preview.value) URL.revokeObjectURL(preview.value.url);
  preview.value = null;
}

async function confirmDelete() {
  if (deletingId.value === null) return;
  try {
    await deleteDocument(deletingId.value);
    showToast("Document deleted", "success");
    await load();
  } catch (err) {
    showToast(err instanceof Error && err.message ? err.message : "Failed to delete document", "error");
  } finally {
    deletingId.value = null;
  }
}

onMounted(load);
onUnmounted(closePreview);
</script>

<template>
  <div class="rounded-lg border border-gray-200 bg-white">
    <div class="flex items-center justify-between border-b border-gray-200 px-5 py-4">
      <div>
        <h2 class="text-sm font-semibold text-gray-900">Documents</h2>
        <p class="text-xs text-gray-500">Passport, visa, work permit · encrypted, access is logged</p>
      </div>
      <button
        v-if="canManage && !formOpen && !unavailable"
        type="button"
        class="rounded-lg bg-navy-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-navy-700"
        @click="openCreate"
      >
        + Add document
      </button>
    </div>

    <form v-if="formOpen" class="grid grid-cols-1 gap-3 border-b border-gray-200 bg-gray-50 p-5 sm:grid-cols-2" @submit.prevent="submit">
      <div>
        <label class="block text-xs font-medium uppercase tracking-wide text-gray-500">Type</label>
        <select v-model="form.type" class="mt-1 w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500">
          <option v-for="(label, value) in DOCUMENT_TYPE_LABELS" :key="value" :value="value">{{ label }}</option>
        </select>
      </div>
      <div v-if="!fileOnly">
        <label class="block text-xs font-medium uppercase tracking-wide text-gray-500">Document number</label>
        <input
          v-model="form.number"
          autocomplete="off"
          :placeholder="editingId !== null ? 'Leave blank to keep current' : 'e.g. MD1234567'"
          class="mt-1 w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
        />
      </div>
      <div v-if="!fileOnly">
        <label class="block text-xs font-medium uppercase tracking-wide text-gray-500">Issue date</label>
        <input v-model="form.issueDate" type="date" class="mt-1 w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500" />
      </div>
      <div v-if="!fileOnly">
        <label class="block text-xs font-medium uppercase tracking-wide text-gray-500">Expiry date</label>
        <input v-model="form.expiryDate" type="date" class="mt-1 w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500" />
      </div>
      <div class="sm:col-span-2">
        <label class="block text-xs font-medium uppercase tracking-wide text-gray-500">Notes</label>
        <input v-model="form.notes" class="mt-1 w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500" />
      </div>
      <div v-if="editingId === null" class="sm:col-span-2">
        <label class="block text-xs font-medium uppercase tracking-wide text-gray-500">Scan / photo (JPG, PNG, WEBP or PDF)</label>
        <input
          ref="fileInput"
          type="file"
          accept="image/jpeg,image/png,image/webp,application/pdf"
          class="mt-1 block w-full text-sm text-gray-600 file:mr-3 file:rounded-lg file:border-0 file:bg-navy-50 file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-navy-700"
          @change="onFile"
        />
        <p v-if="form.file" class="mt-1 text-xs text-gray-500">
          {{ form.file.name }} · {{ formatSize(form.file.size) }}
          <button type="button" class="ml-1 font-medium text-navy-600 hover:text-navy-700" @click="previewLocal">Preview</button>
        </p>
      </div>
      <div class="flex justify-end gap-2 sm:col-span-2">
        <button type="button" class="rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm font-medium text-gray-600 hover:bg-gray-100" @click="closeForm">
          Cancel
        </button>
        <button type="submit" :disabled="saving" class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50">
          {{ saving ? "Saving…" : editingId === null ? "Add" : "Save" }}
        </button>
      </div>
    </form>

    <div v-if="loading" class="px-5 py-8 text-center text-sm text-gray-500">Loading documents…</div>
    <div v-else-if="unavailable" class="px-5 py-8 text-center text-sm text-gray-500">{{ unavailable }}</div>
    <div v-else-if="rows.length === 0" class="px-5 py-8 text-center text-sm text-gray-500">No documents on file.</div>

    <ul v-else-if="canRead" class="divide-y divide-gray-100">
      <li v-for="doc in docs" :key="doc.id" class="flex flex-wrap items-center justify-between gap-3 px-5 py-3">
        <div class="min-w-0">
          <p class="text-sm font-medium text-gray-900">
            {{ DOCUMENT_TYPE_LABELS[doc.type] }}
            <span v-if="doc.number" class="ml-1 font-mono text-xs font-normal text-gray-600">
              {{ revealed[doc.id] ?? doc.number }}
            </span>
            <button v-if="doc.number" type="button" class="ml-1 text-xs font-medium text-navy-600 hover:text-navy-700" @click="toggleReveal(doc)">
              {{ revealed[doc.id] !== undefined ? "Hide" : "Show" }}
            </button>
          </p>
          <p class="text-xs text-gray-500">
            <template v-if="isFileOnly(doc.type)">
              {{ doc.originalName || "File" }}<template v-if="doc.sizeBytes"> · {{ formatSize(doc.sizeBytes) }}</template>
              · uploaded {{ new Date(doc.createdAt).toLocaleDateString() }}
            </template>
            <template v-else>Issued {{ formatDate(doc.issueDate) }} · Expires {{ formatDate(doc.expiryDate) }}</template>
            <template v-if="doc.notes"> · {{ doc.notes }}</template>
          </p>
        </div>
        <div class="flex items-center gap-3">
          <span
            v-if="!(isFileOnly(doc.type) && doc.expiryStatus === 'none')"
            :class="['inline-flex rounded-full px-2 py-0.5 text-xs font-medium', statusStyles[doc.expiryStatus]]"
          >
            {{ expiryLabel(doc.expiryStatus, doc.daysToExpiry) }}
          </span>
          <button
            v-if="doc.hasFile"
            type="button"
            :disabled="previewingId === doc.id"
            class="text-xs font-medium text-navy-600 hover:text-navy-700 disabled:opacity-50"
            @click="previewFile(doc)"
          >
            {{ previewingId === doc.id ? "Opening…" : "Preview" }}
          </button>
          <button v-if="canManage" type="button" class="text-xs font-medium text-gray-600 hover:text-gray-900" @click="openEdit(doc)">Edit</button>
          <button v-if="canManage" type="button" class="text-xs font-medium text-red-600 hover:text-red-700" @click="deletingId = doc.id">Delete</button>
        </div>
      </li>
    </ul>

    <ul v-else class="divide-y divide-gray-100">
      <li v-for="s in statuses" :key="s.id" class="flex items-center justify-between gap-3 px-5 py-3">
        <p class="text-sm text-gray-900">
          {{ DOCUMENT_TYPE_LABELS[s.type] }}
          <span class="ml-1 text-xs text-gray-500">expires {{ formatDate(s.expiryDate) }}</span>
        </p>
        <span :class="['inline-flex rounded-full px-2 py-0.5 text-xs font-medium', statusStyles[s.expiryStatus]]">
          {{ expiryLabel(s.expiryStatus, s.daysToExpiry) }}
        </span>
      </li>
    </ul>

    <ConfirmDialog
      :open="deletingId !== null"
      title="Delete document?"
      message="The record and its encrypted scan are permanently removed."
      confirmLabel="Delete"
      @cancel="deletingId = null"
      @confirm="confirmDelete"
    />

    <DocumentPreview
      v-if="preview"
      :url="preview.url"
      :content-type="preview.contentType"
      :title="preview.title"
      :file-name="preview.fileName"
      @close="closePreview"
    />
  </div>
</template>
