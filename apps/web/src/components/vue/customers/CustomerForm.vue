<script setup lang="ts">
import { computed, onMounted, reactive, ref, useId } from "vue";
import { useModalA11y } from "../ui/useModalA11y";
import type {
  Customer,
  CustomerStatus,
  CreateCustomerInput,
  PropertyType,
} from "../../../lib/customers";
import { getLeads, type Lead } from "../../../lib/leads";

const props = defineProps<{
  customer?: Customer;
}>();

const emit = defineEmits<{
  save: [input: CreateCustomerInput];
  cancel: [];
}>();

const titleId = useId();
const { container } = useModalA11y(() => emit("cancel"));

const statusOptions: { value: CustomerStatus; label: string }[] = [
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
  { value: "blocked", label: "Blocked" },
];

const propertyTypes: { value: PropertyType; label: string }[] = [
  { value: "house", label: "House" },
  { value: "condo", label: "Condo" },
  { value: "office", label: "Office" },
  { value: "apartment", label: "Apartment" },
  { value: "other", label: "Other" },
];

const form = reactive<CreateCustomerInput>({
  firstName: props.customer?.firstName ?? "",
  lastName: props.customer?.lastName ?? "",
  email: props.customer?.email ?? "",
  phone: props.customer?.phone ?? "",
  address: props.customer?.address ?? "",
  propertyType: props.customer?.propertyType ?? "house",
  area: props.customer?.area ?? "",
  status: props.customer?.status ?? "active",
  leadId: props.customer?.leadId ?? null,
});

const leads = ref<Lead[]>([]);
const leadsLoading = ref(false);
const selectedLeadId = ref<number | null>(props.customer?.leadId ?? null);

const isCreating = computed(() => !props.customer);

function leadLabel(lead: Lead): string {
  return [`${lead.firstName} ${lead.lastName}`.trim(), lead.email].filter(Boolean).join(" — ");
}

async function loadLeads() {
  if (!isCreating.value) return;
  leadsLoading.value = true;
  try {
    leads.value = await getLeads();
  } catch {
    leads.value = [];
  } finally {
    leadsLoading.value = false;
  }
}

function applyLead(leadId: number | string) {
  const raw = typeof leadId === "string" ? Number(leadId) : leadId;
  selectedLeadId.value = raw || null;
  form.leadId = selectedLeadId.value;
  const lead = leads.value.find((l) => l.id === raw);
  if (!lead) return;
  form.firstName = lead.firstName;
  form.lastName = lead.lastName;
  form.email = lead.email;
  form.phone = lead.phone;
}

const submitted = ref(false);

const errors = computed(() => {
  const e: Partial<Record<keyof CreateCustomerInput, string>> = {};
  if (!form.firstName.trim()) e.firstName = "First name is required";
  if (form.email.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email))
    e.email = "Email is invalid";
  if (!form.phone.trim()) e.phone = "Phone is required";
  if (!form.address.trim()) e.address = "Address is required";
  if (!form.area.trim()) e.area = "Area is required";
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

function inputClassFor(field: keyof CreateCustomerInput) {
  return [
    inputClass,
    submitted.value && errors.value[field]
      ? "border-red-300 focus:border-red-500 focus:ring-red-500"
      : "",
  ];
}

onMounted(loadLeads);
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
          {{ customer ? "Edit Customer" : "New Customer" }}
        </h2>
      </div>

      <form @submit.prevent="handleSubmit">
        <div class="grid grid-cols-1 gap-4 px-6 py-5 sm:grid-cols-2">
          <div v-if="isCreating" class="sm:col-span-2">
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="c-leadId"
              >Create from lead (optional)</label
            >
            <select
              id="c-leadId"
              :value="selectedLeadId ?? ''"
              :disabled="leadsLoading"
              @change="applyLead(($event.target as HTMLSelectElement).value)"
              class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
            >
              <option value="">Manual entry</option>
              <option
                v-for="lead in leads"
                :key="lead.id"
                :value="lead.id"
              >
                {{ leadLabel(lead) }}
              </option>
            </select>
            <p class="mt-1 text-xs text-gray-500">
              Pick a lead to pre-fill customer details, or leave empty to enter
              manually.
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="c-firstName"
              >First name</label
            >
            <input
              id="c-firstName"
              v-model="form.firstName"
              :class="inputClassFor('firstName')"
              placeholder="Somchai"
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
              for="c-lastName"
              >Last name <span class="font-normal text-gray-400">(optional)</span></label
            >
            <input
              id="c-lastName"
              v-model="form.lastName"
              :class="inputClassFor('lastName')"
              placeholder="Prasert"
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
              for="c-email"
              >Email <span class="font-normal text-gray-400">(optional)</span></label
            >
            <input
              id="c-email"
              v-model="form.email"
              type="email"
              :class="inputClassFor('email')"
              placeholder="somchai@example.com"
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
              for="c-phone"
              >Phone</label
            >
            <input
              id="c-phone"
              v-model="form.phone"
              :class="inputClassFor('phone')"
              placeholder="+66 91 234 5678"
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
              for="c-area"
              >Area</label
            >
            <input
              id="c-area"
              v-model="form.area"
              :class="inputClassFor('area')"
              placeholder="Sukhumvit"
            />
            <p
              v-if="submitted && errors.area"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.area }}
            </p>
          </div>

          <div class="sm:col-span-2">
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="c-address"
              >Address</label
            >
            <input
              id="c-address"
              v-model="form.address"
              :class="inputClassFor('address')"
              placeholder="Sukhumvit 38, Klongton, Bangkok"
            />
            <p
              v-if="submitted && errors.address"
              class="mt-1 text-xs text-red-600"
            >
              {{ errors.address }}
            </p>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="c-propertyType"
              >Property type</label
            >
            <select
              id="c-propertyType"
              v-model="form.propertyType"
              :class="inputClass"
            >
              <option
                v-for="pt in propertyTypes"
                :key="pt.value"
                :value="pt.value"
              >
                {{ pt.label }}
              </option>
            </select>
          </div>

          <div>
            <label
              class="mb-1 block text-sm font-medium text-gray-700"
              for="c-status"
              >Status</label
            >
            <select id="c-status" v-model="form.status" :class="inputClass">
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
            {{ customer ? "Save changes" : "Create customer" }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
