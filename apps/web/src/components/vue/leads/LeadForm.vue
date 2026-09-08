<script setup lang="ts">
import { computed, reactive, ref, useId } from "vue";
import type {
  CreateLeadInput,
  Lead,
  LeadSource,
  LeadStatus,
} from "./../../../lib/leads";
import { useModalA11y } from "../ui/useModalA11y";

const props = defineProps<{
  lead?: Lead;
}>();

const emit = defineEmits<{
  save: [input: CreateLeadInput];
  cancel: [];
}>();

const titleId = useId();
const { container } = useModalA11y(() => emit("cancel"));

const statusOptions: { value: LeadStatus; label: string }[] = [
  { value: "new", label: "New" },
  { value: "contacted", label: "Contacted" },
  { value: "qualified", label: "Qualified" },
  { value: "proposal", label: "Proposal" },
  { value: "won", label: "Won" },
  { value: "lost", label: "Lost" },
];

const sourceOptions: { value: LeadSource; label: string }[] = [
  { value: "website", label: "Website" },
  { value: "referral", label: "Referral" },
  { value: "social_media", label: "Social Media" },
  { value: "walk_in", label: "Walk-in" },
  { value: "campaign", label: "Campaign" },
];

const form = reactive<CreateLeadInput>({
  firstName: props.lead?.firstName ?? "",
  lastName: props.lead?.lastName ?? "",
  email: props.lead?.email ?? "",
  phone: props.lead?.phone ?? "",
  status: props.lead?.status ?? "new",
  source: props.lead?.source ?? "website",
});

const submitted = ref(false);

const errors = computed(() => {
  const e: Partial<Record<keyof CreateLeadInput, string>> = {};
  if (!form.firstName.trim()) e.firstName = "First name is required";
  if (!form.lastName.trim()) e.lastName = "Last name is required";
  if (!form.email.trim()) e.email = "Email is required";
  else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email))
    e.email = "Email is invalid";
  return e;
});

const isValid = computed(() => Object.keys(errors.value).length === 0);

function handleSubmit() {
  submitted.value = true;
  if (!isValid.value) return;
  emit("save", { ...form });
}

const inputClass =
  "w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500";

function inputClassFor(field: keyof CreateLeadInput) {
  return [
    inputClass,
    submitted && errors.value[field]
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
    <div class="absolute inset-0 bg-black/50" @click="emit('cancel')"></div>

    <div class="relative w-full max-w-lg rounded-xl bg-white shadow-xl">
      <div class="border-b border-gray-200 px-6 py-4">
        <h2 :id="titleId" class="text-base font-semibold text-gray-900">
          {{ lead ? "Edit Lead" : "New Lead" }}
        </h2>
      </div>

      <form @submit.prevent="handleSubmit">
        <div class="grid grid-cols-1 gap-4 px-6 py-5 sm:grid-cols-2">
          <div>
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="firstName"
              >First name</label
            >
            <input
              id="firstName"
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
              class="mb-1 block text-sm font-medium text-gray-700"
              for="lastName"
              >Last name</label
            >
            <input
              id="lastName"
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
              class="mb-1 block text-sm font-medium text-gray-700"
              for="email"
              >Email</label
            >
            <input
              id="email"
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
              class="mb-1 block text-sm font-medium text-gray-700"
              for="phone"
              >Phone</label
            >
            <input
              id="phone"
              v-model="form.phone"
              :class="inputClass"
              placeholder="+1 555-0100"
            />
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="source"
              >Source</label
            >
            <select id="source" v-model="form.source" :class="inputClass">
              <option
                v-for="opt in sourceOptions"
                :key="opt.value"
                :value="opt.value"
              >
                {{ opt.label }}
              </option>
            </select>
          </div>

          <div class="sm:col-span-2">
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="status"
              >Status</label
            >
            <select id="status" v-model="form.status" :class="inputClass">
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

        <div class="flex justify-end gap-3 border-t border-gray-200 px-6 py-4">
          <button
            type="button"
            class="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100"
            @click="emit('cancel')"
          >
            Cancel
          </button>
          <button
            type="submit"
            class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700"
          >
            {{ lead ? "Save changes" : "Create lead" }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
