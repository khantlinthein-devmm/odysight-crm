<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  deleteDocument,
  getDocuments,
  updateDocument,
  uploadDocument,
  type DocumentRecord,
  type DocumentStatus,
  type UpdateDocumentInput,
  type UploadDocumentInput,
} from "../../../lib/documents";
import { showToast } from "../../../lib/toast";
import DocumentForm from "./DocumentForm.vue";
import ConfirmDialog from "../ui/ConfirmDialog.vue";

const statusLabels: Record<DocumentStatus, string> = {
  pending: "Pending",
  verified: "Verified",
  rejected: "Rejected",
  expired: "Expired",
};

const statusStyles: Record<DocumentStatus, string> = {
  pending: "bg-amber-50 text-amber-700",
  verified: "bg-emerald-50 text-emerald-700",
  rejected: "bg-red-50 text-red-700",
  expired: "bg-slate-100 text-slate-600",
};

const documents = ref<DocumentRecord[]>([]);
const loading = ref(true);
const search = ref("");
const statusFilter = ref<DocumentStatus | "">("");
const page = ref(1);
const pageSize = 5;

const showForm = ref(false);
const editingDoc = ref<DocumentRecord | undefined>(undefined);
const saving = ref(false);
const pendingDelete = ref<DocumentRecord | null>(null);
const deleting = ref(false);

const filteredDocuments = computed(() => {
  const query = search.value.trim().toLowerCase();
  return documents.value.filter((doc) => {
    const matchesStatus =
      !statusFilter.value || doc.status === statusFilter.value;
    const matchesQuery =
      !query ||
      doc.name.toLowerCase().includes(query) ||
      doc.applicantName.toLowerCase().includes(query);
    return matchesStatus && matchesQuery;
  });
});

const totalPages = computed(() =>
  Math.max(1, Math.ceil(filteredDocuments.value.length / pageSize)),
);

const pagedDocuments = computed(() => {
  const start = (page.value - 1) * pageSize;
  return filteredDocuments.value.slice(start, start + pageSize);
});

async function fetchDocuments() {
  loading.value = true;
  try {
    documents.value = await getDocuments();
  } catch {
    showToast("Failed to load documents", "error");
  } finally {
    loading.value = false;
  }
}

function openUpload() {
  editingDoc.value = undefined;
  showForm.value = true;
}

function openEdit(doc: DocumentRecord) {
  editingDoc.value = doc;
  showForm.value = true;
}

function closeForm() {
  showForm.value = false;
  editingDoc.value = undefined;
}

async function handleSave(input: UploadDocumentInput) {
  saving.value = true;
  try {
    if (editingDoc.value) {
      const updated = await updateDocument(
        editingDoc.value.id,
        input as UpdateDocumentInput,
      );
      documents.value = documents.value.map((d) =>
        d.id === updated.id ? updated : d,
      );
      showToast("Document updated", "success");
    } else {
      const created = await uploadDocument(input);
      documents.value = [created, ...documents.value];
      showToast("Document added", "success");
    }
    closeForm();
  } catch {
    showToast("Failed to save document", "error");
  } finally {
    saving.value = false;
  }
}

async function handleDelete() {
  if (!pendingDelete.value) return;
  deleting.value = true;
  try {
    await deleteDocument(pendingDelete.value.id);
    documents.value = documents.value.filter(
      (d) => d.id !== pendingDelete.value!.id,
    );
    showToast("Document deleted", "success");
  } catch {
    showToast("Failed to delete document", "error");
  } finally {
    deleting.value = false;
    pendingDelete.value = null;
  }
}

function formatSize(kb: number): string {
  return kb >= 1024 ? `${(kb / 1024).toFixed(1)} MB` : `${kb} KB`;
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

onMounted(fetchDocuments);
</script>

<template>
  <div class="rounded-xl border border-slate-200 bg-white">
    <div
      class="flex flex-col gap-3 border-b border-slate-200 px-6 py-4 sm:flex-row sm:items-center"
    >
      <h2 class="text-base font-semibold text-slate-900">Documents</h2>
      <span
        class="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-600"
      >
        {{ filteredDocuments.length }}
      </span>

      <div class="sm:ml-auto flex flex-col gap-2 sm:flex-row">
        <input
          v-model="search"
          @input="page = 1"
          type="search"
          placeholder="Search file or applicant..."
          class="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 sm:w-56"
        />
        <select
          v-model="statusFilter"
          @change="page = 1"
          class="rounded-lg border border-slate-200 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
        >
          <option value="">All statuses</option>
          <option
            v-for="(label, value) in statusLabels"
            :key="value"
            :value="value"
          >
            {{ label }}
          </option>
        </select>
        <button
          type="button"
          class="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
          @click="openUpload"
        >
          Upload
        </button>
      </div>
    </div>

    <div v-if="loading" class="px-6 py-16 text-center">
      <p class="text-sm text-slate-500">Loading documents...</p>
    </div>

    <div
      v-else-if="filteredDocuments.length === 0"
      class="px-6 py-16 text-center"
    >
      <p class="text-sm font-medium text-slate-900">No documents found</p>
      <p class="mt-1 text-sm text-slate-500">
        Try adjusting your filters or upload a new document.
      </p>
    </div>

    <div v-else class="overflow-x-auto">
      <table class="w-full text-left text-sm">
        <thead>
          <tr
            class="border-b border-slate-200 text-xs uppercase tracking-wide text-slate-500"
          >
            <th class="px-6 py-3 font-medium">File Name</th>
            <th class="px-6 py-3 font-medium">Type</th>
            <th class="px-6 py-3 font-medium">Applicant</th>
            <th class="px-6 py-3 font-medium">Size</th>
            <th class="px-6 py-3 font-medium">Status</th>
            <th class="px-6 py-3 font-medium">Uploaded</th>
            <th class="px-6 py-3 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          <tr
            v-for="doc in pagedDocuments"
            :key="doc.id"
            class="hover:bg-slate-50"
          >
            <td class="px-6 py-4">
              <div class="flex items-center gap-2">
                <svg
                  class="h-4 w-4 shrink-0 text-slate-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                  ></path>
                </svg>
                <span class="font-medium text-slate-900">{{ doc.name }}</span>
              </div>
            </td>
            <td class="px-6 py-4 text-slate-600">{{ doc.type }}</td>
            <td class="px-6 py-4 text-slate-600">{{ doc.applicantName }}</td>
            <td class="px-6 py-4 text-slate-500">
              {{ formatSize(doc.fileSizeKb) }}
            </td>
            <td class="px-6 py-4">
              <span
                :class="[
                  'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
                  statusStyles[doc.status],
                ]"
              >
                {{ statusLabels[doc.status] }}
              </span>
            </td>
            <td class="px-6 py-4 text-slate-600">
              {{ formatDate(doc.uploadedAt) }}
            </td>
            <td class="px-6 py-4 text-right whitespace-nowrap">
              <button
                type="button"
                class="rounded-lg px-2 py-1 text-sm font-medium text-slate-600 hover:bg-slate-100 hover:text-slate-900"
                @click="openEdit(doc)"
              >
                Edit
              </button>
              <button
                type="button"
                class="rounded-lg px-2 py-1 text-sm font-medium text-red-600 hover:bg-red-50"
                @click="pendingDelete = doc"
              >
                Delete
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div
      v-if="!loading && totalPages > 1"
      class="flex items-center justify-between border-t border-slate-200 px-6 py-3"
    >
      <p class="text-sm text-slate-500">Page {{ page }} of {{ totalPages }}</p>
      <div class="flex gap-2">
        <button
          type="button"
          :disabled="page <= 1"
          class="rounded-lg border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-40 disabled:hover:bg-transparent"
          @click="page--"
        >
          Previous
        </button>
        <button
          type="button"
          :disabled="page >= totalPages"
          class="rounded-lg border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-40 disabled:hover:bg-transparent"
          @click="page++"
        >
          Next
        </button>
      </div>
    </div>
  </div>

  <DocumentForm
    v-if="showForm"
    :key="editingDoc?.id ?? 'new'"
    :document="editingDoc"
    @save="handleSave"
    @cancel="closeForm"
  />

  <ConfirmDialog
    v-if="pendingDelete"
    title="Delete document"
    :message="`Are you sure you want to delete ${pendingDelete.name}? This action cannot be undone.`"
    confirm-label="Delete document"
    :busy="deleting"
    @confirm="handleDelete"
    @cancel="pendingDelete = null"
  />
</template>
