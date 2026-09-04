<script setup lang="ts">
import { computed, reactive, ref, useId } from "vue";
import { useModalA11y } from "../ui/useModalA11y";
import type {
  DocumentRecord,
  DocumentStatus,
  UploadDocumentInput,
} from "../../../lib/documents";

const props = defineProps<{
  document?: DocumentRecord;
}>();

const emit = defineEmits<{
  save: [input: UploadDocumentInput];
  cancel: [];
}>();

const titleId = useId();
const { container } = useModalA11y(() => emit("cancel"));

const statusOptions: { value: DocumentStatus; label: string }[] = [
  { value: "pending", label: "Pending" },
  { value: "verified", label: "Verified" },
  { value: "rejected", label: "Rejected" },
  { value: "expired", label: "Expired" },
];

const docTypes = [
  "Passport",
  "Photo",
  "Education Certificate",
  "Financial Proof",
  "Medical Report",
  "Police Clearance",
  "Other",
];

const form = reactive<UploadDocumentInput>({
  name: props.document?.name ?? "",
  type: props.document?.type ?? docTypes[0],
  applicantName: props.document?.applicantName ?? "",
  status: props.document?.status ?? "pending",
});

const submitted = ref(false);

const errors = computed(() => {
  const e: Partial<Record<keyof UploadDocumentInput, string>> = {};
  if (!form.name.trim()) e.name = "File name is required";
  if (!form.applicantName.trim())
    e.applicantName = "Applicant name is required";
  return e;
});

const isValid = computed(() => Object.keys(errors.value).length === 0);

function handleSubmit() {
  submitted.value = true;
  if (!isValid.value) return;
  emit("save", { ...form });
}

const inputClass =
  "w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 placeholder-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500";

function inputClassFor(field: keyof UploadDocumentInput) {
  return [
    inputClass,
    submitted.value && errors.value[field]
      ? "border-red-300 focus:border-red-500 focus:ring-red-500"
      : "",
  ];
}
</script>

<template>
  <div
    ref="container"
    role="dialog"
    aria-modal="true"
    :aria-labelledby="titleId"
    class="fixed inset-0 z-50 flex items-center justify-center p-4"
  >
    <div class="absolute inset-0 bg-slate-900/50" @click="emit('cancel')"></div>

    <div class="relative w-full max-w-lg rounded-xl bg-white shadow-xl">
      <div class="border-b border-slate-200 px-6 py-4">
        <h2 :id="titleId" class="text-base font-semibold text-slate-900">
          {{ document ? "Edit Document" : "Upload Document" }}
        </h2>
      </div>

      <form @submit.prevent="handleSubmit">
        <div class="grid grid-cols-1 gap-4 px-6 py-5 sm:grid-cols-2">
          <div class="sm:col-span-2">
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="d-name"
              >File name</label
            >
            <input
              id="d-name"
              v-model="form.name"
              :class="inputClassFor('name')"
              placeholder="passport.pdf"
            />
            <p
              v-if="submitted && errors.name"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.name }}
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="d-type"
              >Document type</label
            >
            <select id="d-type" v-model="form.type" :class="inputClass">
              <option v-for="dt in docTypes" :key="dt" :value="dt">
                {{ dt }}
              </option>
            </select>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="d-applicant"
              >Applicant</label
            >
            <input
              id="d-applicant"
              v-model="form.applicantName"
              :class="inputClassFor('applicantName')"
              placeholder="John Doe"
            />
            <p
              v-if="submitted && errors.applicantName"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.applicantName }}
            </p>
          </div>

          <div class="sm:col-span-2">
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="d-status"
              >Status</label
            >
            <select id="d-status" v-model="form.status" :class="inputClass">
              <option
                v-for="opt in statusOptions"
                :key="opt.value"
                :value="opt.value"
              >
                {{ opt.label }}
              </option>
            </select>
          </div>
        </div>

        <div class="flex justify-end gap-3 border-t border-slate-200 px-6 py-4">
          <button
            type="button"
            class="rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
            @click="emit('cancel')"
          >
            Cancel
          </button>
          <button
            type="submit"
            class="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
          >
            {{ document ? "Save changes" : "Add document" }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
