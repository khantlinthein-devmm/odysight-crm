<script setup lang="ts">
import { computed, reactive, ref, useId } from "vue";
import { useModalA11y } from "../ui/useModalA11y";
import type {
  CreateVisaCaseInput,
  VisaCase,
  VisaCaseStatus,
} from "../../../lib/visa-cases";

const props = defineProps<{
  visaCase?: VisaCase;
}>();

const emit = defineEmits<{
  save: [input: CreateVisaCaseInput];
  cancel: [];
}>();

const titleId = useId();
const { container } = useModalA11y(() => emit("cancel"));

const statusOptions: { value: VisaCaseStatus; label: string }[] = [
  { value: "draft", label: "Draft" },
  { value: "in_review", label: "In Review" },
  { value: "submitted", label: "Submitted" },
  { value: "additional_docs_required", label: "Additional Docs Required" },
  { value: "approved", label: "Approved" },
  { value: "rejected", label: "Rejected" },
  { value: "closed", label: "Closed" },
];

const visaTypes = [
  "Student Visa",
  "Work Permit",
  "Tourist Visa",
  "Business Visa",
  "Family Visa",
];

const form = reactive<CreateVisaCaseInput>({
  applicantName: props.visaCase?.applicantName ?? "",
  visaType: props.visaCase?.visaType ?? visaTypes[0],
  destination: props.visaCase?.destination ?? "",
  assignedTo: props.visaCase?.assignedTo ?? "",
  status: props.visaCase?.status ?? "draft",
});

const submitted = ref(false);

const errors = computed(() => {
  const e: Partial<Record<keyof CreateVisaCaseInput, string>> = {};
  if (!form.applicantName.trim())
    e.applicantName = "Applicant name is required";
  if (!form.destination.trim()) e.destination = "Destination is required";
  if (!form.assignedTo.trim()) e.assignedTo = "Assignee is required";
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

function inputClassFor(field: keyof CreateVisaCaseInput) {
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
          {{ visaCase ? `Edit Case ${visaCase.caseNumber}` : "New Visa Case" }}
        </h2>
      </div>

      <form @submit.prevent="handleSubmit">
        <div class="grid grid-cols-1 gap-4 px-6 py-5 sm:grid-cols-2">
          <div class="sm:col-span-2">
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="v-applicant"
              >Applicant name</label
            >
            <input
              id="v-applicant"
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

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="v-visaType"
              >Visa type</label
            >
            <select id="v-visaType" v-model="form.visaType" :class="inputClass">
              <option v-for="vt in visaTypes" :key="vt" :value="vt">
                {{ vt }}
              </option>
            </select>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="v-destination"
              >Destination</label
            >
            <input
              id="v-destination"
              v-model="form.destination"
              :class="inputClassFor('destination')"
              placeholder="Canada"
            />
            <p
              v-if="submitted && errors.destination"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.destination }}
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="v-assignedTo"
              >Assigned to</label
            >
            <input
              id="v-assignedTo"
              v-model="form.assignedTo"
              :class="inputClassFor('assignedTo')"
              placeholder="Sarah Miller"
            />
            <p
              v-if="submitted && errors.assignedTo"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.assignedTo }}
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="v-status"
              >Status</label
            >
            <select id="v-status" v-model="form.status" :class="inputClass">
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
            {{ visaCase ? "Save changes" : "Create case" }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
