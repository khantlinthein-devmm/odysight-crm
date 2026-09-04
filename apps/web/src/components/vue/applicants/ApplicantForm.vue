<script setup lang="ts">
import { computed, reactive, ref, useId } from "vue";
import { useModalA11y } from "../ui/useModalA11y";
import type {
  Applicant,
  ApplicantStatus,
  CreateApplicantInput,
} from "../../../lib/applicants";

const props = defineProps<{
  applicant?: Applicant;
}>();

const emit = defineEmits<{
  save: [input: CreateApplicantInput];
  cancel: [];
}>();

const titleId = useId();
const { container } = useModalA11y(() => emit("cancel"));

const statusOptions: { value: ApplicantStatus; label: string }[] = [
  { value: "screening", label: "Screening" },
  { value: "document_collection", label: "Document Collection" },
  { value: "submitted", label: "Submitted" },
  { value: "processing", label: "Processing" },
  { value: "approved", label: "Approved" },
  { value: "rejected", label: "Rejected" },
];

const visaTypes = [
  "Student Visa",
  "Work Permit",
  "Tourist Visa",
  "Business Visa",
  "Family Visa",
];

const form = reactive<CreateApplicantInput>({
  firstName: props.applicant?.firstName ?? "",
  lastName: props.applicant?.lastName ?? "",
  email: props.applicant?.email ?? "",
  phone: props.applicant?.phone ?? "",
  nationality: props.applicant?.nationality ?? "",
  visaType: props.applicant?.visaType ?? visaTypes[0],
  status: props.applicant?.status ?? "screening",
});

const submitted = ref(false);

const errors = computed(() => {
  const e: Partial<Record<keyof CreateApplicantInput, string>> = {};
  if (!form.firstName.trim()) e.firstName = "First name is required";
  if (!form.lastName.trim()) e.lastName = "Last name is required";
  if (!form.email.trim()) e.email = "Email is required";
  else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email))
    e.email = "Email is invalid";
  if (!form.nationality.trim()) e.nationality = "Nationality is required";
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

function inputClassFor(field: keyof CreateApplicantInput) {
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
          {{ applicant ? "Edit Applicant" : "New Applicant" }}
        </h2>
      </div>

      <form @submit.prevent="handleSubmit">
        <div class="grid grid-cols-1 gap-4 px-6 py-5 sm:grid-cols-2">
          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="a-firstName"
              >First name</label
            >
            <input
              id="a-firstName"
              v-model="form.firstName"
              :class="inputClassFor('firstName')"
              placeholder="John"
            />
            <p
              v-if="submitted && errors.firstName"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.firstName }}
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="a-lastName"
              >Last name</label
            >
            <input
              id="a-lastName"
              v-model="form.lastName"
              :class="inputClassFor('lastName')"
              placeholder="Doe"
            />
            <p
              v-if="submitted && errors.lastName"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.lastName }}
            </p>
          </div>

          <div class="sm:col-span-2">
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="a-email"
              >Email</label
            >
            <input
              id="a-email"
              v-model="form.email"
              type="email"
              :class="inputClassFor('email')"
              placeholder="john@example.com"
            />
            <p
              v-if="submitted && errors.email"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.email }}
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="a-phone"
              >Phone</label
            >
            <input
              id="a-phone"
              v-model="form.phone"
              :class="inputClass"
              placeholder="+1 555-0100"
            />
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="a-nationality"
              >Nationality</label
            >
            <input
              id="a-nationality"
              v-model="form.nationality"
              :class="inputClassFor('nationality')"
              placeholder="USA"
            />
            <p
              v-if="submitted && errors.nationality"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.nationality }}
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="a-visaType"
              >Visa type</label
            >
            <select id="a-visaType" v-model="form.visaType" :class="inputClass">
              <option v-for="vt in visaTypes" :key="vt" :value="vt">
                {{ vt }}
              </option>
            </select>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="a-status"
              >Status</label
            >
            <select id="a-status" v-model="form.status" :class="inputClass">
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
            {{ applicant ? "Save changes" : "Create applicant" }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
