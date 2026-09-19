<script setup lang="ts">
import { computed, onMounted, reactive, ref, useId } from "vue";
import { useModalA11y } from "../ui/useModalA11y";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";
import { showToast } from "../../../lib/toast";
import {
  addCleanerPhone,
  getCleanerPhones,
  removeCleanerPhone,
  type Cleaner,
  type CleanerStatus,
  type CreateCleanerInput,
  type PhoneNumber,
} from "../../../lib/cleaners";

const props = defineProps<{
  cleaner?: Cleaner;
}>();

const emit = defineEmits<{
  save: [input: CreateCleanerInput, extraPhones: { label: string; phone: string }[]];
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

const phoneLabelOptions = ["Mobile", "Home", "Emergency", "Office", "Other"];

interface ExtraPhoneRow {
  key: number;
  id?: number;
  label: string;
  phone: string;
}

const form = reactive<CreateCleanerInput>({
  firstName: props.cleaner?.firstName ?? "",
  lastName: props.cleaner?.lastName ?? "",
  phone: props.cleaner?.phone ?? "",
  email: props.cleaner?.email ?? "",
  lineId: props.cleaner?.lineId ?? "",
  skills: props.cleaner?.skills ?? "",
  status: props.cleaner?.status ?? "available",
});

let rowKey = 0;
const extraPhones = ref<ExtraPhoneRow[]>([]);
const originalPhones = ref<PhoneNumber[]>([]);
const phonesLoading = ref(false);
const phonesError = ref<string | null>(null);
const syncingPhones = ref(false);

const canEditPhones = computed(() =>
  hasPermission(getSessionUser()?.role, "cleaners.update"),
);

async function loadPhones() {
  if (!props.cleaner) return;
  phonesLoading.value = true;
  phonesError.value = null;
  try {
    const rows = await getCleanerPhones(props.cleaner.id);
    originalPhones.value = rows;
    extraPhones.value = rows.map((p) => ({
      key: ++rowKey,
      id: p.id,
      label: p.label,
      phone: p.phone,
    }));
  } catch (err) {
    phonesError.value =
      err instanceof Error && err.message
        ? err.message
        : "Failed to load phone numbers";
  } finally {
    phonesLoading.value = false;
  }
}

onMounted(loadPhones);

function addPhoneRow() {
  if (!canEditPhones.value) return;
  extraPhones.value.push({ key: ++rowKey, label: "Mobile", phone: "" });
}

function removePhoneRow(index: number) {
  if (!canEditPhones.value) return;
  extraPhones.value.splice(index, 1);
}

async function syncPhones(cleanerId: number): Promise<void> {
  const current = extraPhones.value;
  const original = originalPhones.value;
  const removed = original.filter(
    (p) => !current.some((r) => r.id === p.id),
  );
  for (const p of removed) {
    await removeCleanerPhone(cleanerId, p.id);
  }
  for (const row of current) {
    const phone = row.phone.trim();
    if (!phone) continue;
    const prev = original.find((p) => p.id === row.id);
    if (row.id === undefined || !prev) {
      const created = await addCleanerPhone(cleanerId, {
        label: row.label,
        phone,
      });
      row.id = created.id;
    } else if (prev.label !== row.label || prev.phone !== phone) {
      await removeCleanerPhone(cleanerId, row.id);
      const created = await addCleanerPhone(cleanerId, {
        label: row.label,
        phone,
      });
      row.id = created.id;
    }
  }
  originalPhones.value = await getCleanerPhones(cleanerId);
}

const submitted = ref(false);

const errors = computed(() => {
  const e: Partial<Record<keyof CreateCleanerInput, string>> = {};
  if (!form.firstName.trim()) e.firstName = "First name is required";
  if (!form.phone.trim()) e.phone = "Phone is required";
  if (form.email.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email))
    e.email = "Email is invalid";
  return e;
});

const isValid = computed(() => Object.keys(errors.value).length === 0);

async function handleSubmit() {
  submitted.value = true;
  if (!isValid.value) return;
  phonesError.value = null;
  const blankRow = extraPhones.value.some((r) => !r.phone.trim());
  if (blankRow) {
    phonesError.value = "Additional phone numbers cannot be empty";
    return;
  }
  if (props.cleaner && canEditPhones.value) {
    syncingPhones.value = true;
    try {
      await syncPhones(props.cleaner.id);
    } catch (err) {
      const message =
        err instanceof Error && err.message
          ? err.message
          : "Failed to save phone numbers";
      phonesError.value = message;
      showToast(message, "error");
      return;
    } finally {
      syncingPhones.value = false;
    }
  }
  emit(
    "save",
    { ...form },
    extraPhones.value.map((r) => ({ label: r.label, phone: r.phone.trim() })),
  );
}

const inputClass =
  "w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500";

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
    <div class="absolute inset-0 bg-black/50" @click="emit('cancel')"></div>

    <div class="relative w-full max-w-lg rounded-xl bg-white shadow-xl">
      <div class="border-b border-gray-200 px-6 py-4">
        <h2 :id="titleId" class="text-base font-semibold text-gray-900">
          {{ cleaner ? "Edit Cleaner" : "New Cleaner" }}
        </h2>
      </div>

      <form @submit.prevent="handleSubmit">
        <div class="grid grid-cols-1 gap-4 px-6 py-5 sm:grid-cols-2">
          <div>
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
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
              class="mb-1 block text-sm font-medium text-gray-700"
              for="cl-lastName"
              >Last name <span class="font-normal text-gray-400">(optional)</span></label
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
              class="mb-1 block text-sm font-medium text-gray-700"
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
              class="mb-1 block text-sm font-medium text-gray-700"
              for="cl-email"
              >Email <span class="font-normal text-gray-400">(optional)</span></label
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

          <div>
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="cl-line-id"
              >LINE ID <span class="font-normal text-gray-400">(optional)</span></label
            >
            <input
              id="cl-line-id"
              v-model="form.lineId"
              :class="inputClass"
              placeholder="@smileclean or nok_clean"
            />
          </div>

          <div class="sm:col-span-2">
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
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
              class="mb-1 block text-sm font-medium text-gray-700"
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
            {{ cleaner ? "Save changes" : "Create cleaner" }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
