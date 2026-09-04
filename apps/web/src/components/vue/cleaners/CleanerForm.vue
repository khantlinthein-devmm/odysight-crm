<script setup lang="ts">
import { computed, reactive, ref, useId } from "vue";
import { useModalA11y } from "../ui/useModalA11y";
import type {
  Cleaner,
  CleanerStatus,
  CreateCleanerInput,
} from "../../../lib/cleaners";

const props = defineProps<{
  cleaner?: Cleaner;
}>();

const emit = defineEmits<{
  save: [input: CreateCleanerInput];
  cancel: [];
}>();

const titleId = useId();
const { container } = useModalA11y(() => emit("cancel"));

const statusOptions: { value: CleanerStatus; label: string }[] = [
  { value: "available", label: "Available" },
  { value: "assigned", label: "Assigned" },
  { value: "on_leave", label: "On Leave" },
  { value: "inactive", label: "Inactive" },
];

const form = reactive<CreateCleanerInput>({
  firstName: props.cleaner?.firstName ?? "",
  lastName: props.cleaner?.lastName ?? "",
  phone: props.cleaner?.phone ?? "",
  email: props.cleaner?.email ?? "",
  skills: props.cleaner?.skills ?? "",
  status: props.cleaner?.status ?? "available",
});

const submitted = ref(false);

const errors = computed(() => {
  const e: Partial<Record<keyof CreateCleanerInput, string>> = {};
  if (!form.firstName.trim()) e.firstName = "First name is required";
  if (!form.lastName.trim()) e.lastName = "Last name is required";
  if (!form.phone.trim()) e.phone = "Phone is required";
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
  "w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 placeholder-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500";

function inputClassFor(field: keyof CreateCleanerInput) {
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
          {{ cleaner ? "Edit Cleaner" : "New Cleaner" }}
        </h2>
      </div>

      <form @submit.prevent="handleSubmit">
        <div class="grid grid-cols-1 gap-4 px-6 py-5 sm:grid-cols-2">
          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="cl-firstName"
              >First name</label
            >
            <input
              id="cl-firstName"
              v-model="form.firstName"
              :class="inputClassFor('firstName')"
              placeholder="Nok"
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
              for="cl-lastName"
              >Last name</label
            >
            <input
              id="cl-lastName"
              v-model="form.lastName"
              :class="inputClassFor('lastName')"
              placeholder="Srisuwan"
            />
            <p
              v-if="submitted && errors.lastName"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.lastName }}
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="cl-phone"
              >Phone</label
            >
            <input
              id="cl-phone"
              v-model="form.phone"
              :class="inputClassFor('phone')"
              placeholder="+66 81 111 2233"
            />
            <p
              v-if="submitted && errors.phone"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.phone }}
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="cl-email"
              >Email</label
            >
            <input
              id="cl-email"
              v-model="form.email"
              type="email"
              :class="inputClassFor('email')"
              placeholder="nok@smileclean.com"
            />
            <p
              v-if="submitted && errors.email"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.email }}
            </p>
          </div>

          <div class="sm:col-span-2">
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="cl-skills"
              >Skills</label
            >
            <input
              id="cl-skills"
              v-model="form.skills"
              :class="inputClass"
              placeholder="Deep Cleaning, Condo"
            />
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-slate-700"
              for="cl-status"
              >Status</label
            >
            <select id="cl-status" v-model="form.status" :class="inputClass">
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
            {{ cleaner ? "Save changes" : "Create cleaner" }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
