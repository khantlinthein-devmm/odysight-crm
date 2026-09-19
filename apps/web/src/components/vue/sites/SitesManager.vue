<script setup lang="ts">
import { computed, reactive, ref, useId, watch } from "vue";
import { ApiError } from "../../../lib/api";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";
import {
  createSite,
  deleteSite,
  getSites,
  updateSite,
  type CreateSiteInput,
  type Site,
  type SitePropertyType,
} from "../../../lib/sites";
import { showToast } from "../../../lib/toast";
import ConfirmDialog from "../ui/ConfirmDialog.vue";
import { useModalA11y } from "../ui/useModalA11y";

const props = defineProps<{
  customerId: number;
  customerName: string;
}>();

const emit = defineEmits<{
  close: [];
}>();

const titleId = useId();
const { container } = useModalA11y(() => emit("close"));

const role = getSessionUser()?.role;
const canCreate = computed(() => hasPermission(role, "sites.create"));
const canUpdate = computed(() => hasPermission(role, "sites.update"));
const canDelete = computed(() => hasPermission(role, "sites.delete"));

const PROPERTY_TYPES: { value: SitePropertyType; label: string }[] = [
  { value: "house", label: "House" },
  { value: "condo", label: "Condo" },
  { value: "apartment", label: "Apartment" },
  { value: "office", label: "Office" },
  { value: "restaurant", label: "Restaurant" },
  { value: "factory", label: "Factory" },
  { value: "mall", label: "Mall" },
  { value: "retail", label: "Retail" },
  { value: "other", label: "Other" },
];

const sites = ref<Site[]>([]);
const loading = ref(true);
const saving = ref(false);
const pendingDelete = ref<Site | null>(null);
const deleting = ref(false);

async function load(): Promise<void> {
  loading.value = true;
  try {
    sites.value = await getSites({ customer: props.customerId, limit: 200 });
  } catch {
    showToast("Failed to load sites", "error");
  } finally {
    loading.value = false;
  }
}
void load();

const editingId = ref<number | null>(null);
const showForm = ref(false);

function emptyForm(): Omit<CreateSiteInput, "customerId"> {
  return {
    name: "",
    address: "",
    area: "",
    propertyType: "other",
    contactName: "",
    contactPhone: "",
    contactEmail: "",
    notes: "",
    status: "active",
  };
}

const form = reactive(emptyForm());

function openCreate(): void {
  editingId.value = null;
  Object.assign(form, emptyForm());
  showForm.value = true;
}

function openEdit(site: Site): void {
  editingId.value = site.id;
  Object.assign(form, {
    name: site.name,
    address: site.address,
    area: site.area,
    propertyType: site.propertyType,
    contactName: site.contactName,
    contactPhone: site.contactPhone,
    contactEmail: site.contactEmail,
    notes: site.notes,
    status: site.status,
  });
  showForm.value = true;
}

function closeForm(): void {
  showForm.value = false;
  editingId.value = null;
}

const submitted = ref(false);
const errors = computed(() => {
  const e: Partial<Record<"name", string>> = {};
  if (!form.name.trim()) e.name = "Name is required";
  return e;
});
const isValid = computed(() => Object.keys(errors.value).length === 0);

function errorMessage(err: unknown, fallback: string): string {
  if (err instanceof ApiError) return err.message || fallback;
  if (err instanceof Error && err.message) return err.message;
  return fallback;
}

async function submitForm(): Promise<void> {
  submitted.value = true;
  if (!isValid.value) return;
  saving.value = true;
  try {
    if (editingId.value != null) {
      await updateSite(editingId.value, { ...form });
      showToast("Site updated", "success");
    } else {
      await createSite({ customerId: props.customerId, ...form });
      showToast("Site added", "success");
    }
    closeForm();
    await load();
  } catch (err) {
    showToast(errorMessage(err, "Failed to save site"), "error");
  } finally {
    saving.value = false;
  }
}

async function handleDelete(): Promise<void> {
  if (!pendingDelete.value) return;
  deleting.value = true;
  try {
    await deleteSite(pendingDelete.value.id);
    showToast("Site removed", "success");
    pendingDelete.value = null;
    await load();
  } catch (err) {
    showToast(errorMessage(err, "Failed to remove site"), "error");
  } finally {
    deleting.value = false;
  }
}

watch(
  () => props.customerId,
  () => void load(),
);
</script>

<template>
  <div
    ref="container"
    role="dialog"
    aria-modal="true"
    :aria-labelledby="titleId"
    class="fixed inset-0 z-50 overflow-y-auto"
  >
    <div class="fixed inset-0 bg-black/50" @click="emit('close')"></div>

    <div class="relative flex min-h-full items-center justify-center p-4 sm:p-6">
      <div class="relative w-full max-w-2xl rounded-2xl bg-white shadow-xl">
        <div
          class="sticky top-0 z-10 flex items-center justify-between rounded-t-2xl border-b border-gray-200 bg-white px-6 py-4"
        >
          <div>
            <h2 :id="titleId" class="text-base font-semibold text-gray-900">
              Sites — {{ customerName }}
            </h2>
            <p class="text-xs text-gray-500">
              Branches, buildings or areas this customer gets cleaned.
            </p>
          </div>
          <button
            type="button"
            class="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
            aria-label="Close"
            @click="emit('close')"
          >
            ✕
          </button>
        </div>

        <div class="max-h-[70vh] overflow-y-auto px-6 py-5">
          <div v-if="loading" class="space-y-2">
            <div v-for="i in 3" :key="i" class="h-16 animate-pulse rounded-xl bg-gray-50"></div>
          </div>

          <ul v-else-if="sites.length > 0" class="space-y-2">
            <li
              v-for="site in sites"
              :key="site.id"
              class="rounded-xl border border-gray-200 p-4"
            >
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <p class="truncate text-sm font-medium text-gray-900">{{ site.name }}</p>
                  <p class="truncate text-xs text-gray-500">
                    {{ site.address || "No address on file" }}
                  </p>
                  <p class="mt-1 flex flex-wrap gap-1.5 text-xs text-gray-400">
                    <span class="rounded-full bg-gray-100 px-2 py-0.5">{{ site.propertyType }}</span>
                    <span v-if="site.area" class="rounded-full bg-gray-100 px-2 py-0.5">{{ site.area }}</span>
                    <span
                      class="rounded-full px-2 py-0.5"
                      :class="site.status === 'active' ? 'bg-emerald-50 text-emerald-700' : 'bg-gray-100 text-gray-500'"
                    >
                      {{ site.status }}
                    </span>
                  </p>
                </div>
                <div class="flex shrink-0 gap-1">
                  <button
                    v-if="canUpdate"
                    type="button"
                    class="rounded-lg px-2 py-1 text-xs font-medium text-gray-600 hover:bg-gray-100"
                    @click="openEdit(site)"
                  >
                    Edit
                  </button>
                  <button
                    v-if="canDelete"
                    type="button"
                    class="rounded-lg px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-50"
                    @click="pendingDelete = site"
                  >
                    Delete
                  </button>
                </div>
              </div>
            </li>
          </ul>

          <p v-else class="py-8 text-center text-sm text-gray-400">
            No sites yet — every customer starts with none until you add one.
          </p>

          <button
            v-if="canCreate && !showForm"
            type="button"
            class="mt-4 w-full rounded-xl border-2 border-dashed border-gray-200 py-3 text-sm font-medium text-gray-500 hover:border-navy-300 hover:text-navy-600"
            @click="openCreate"
          >
            + Add site
          </button>

          <form
            v-if="showForm"
            class="mt-4 space-y-3 rounded-xl border border-gray-200 p-4"
            @submit.prevent="submitForm"
          >
            <h3 class="text-sm font-semibold text-gray-900">
              {{ editingId != null ? "Edit site" : "New site" }}
            </h3>

            <div>
              <label class="mb-1 block text-xs font-medium text-gray-700" for="site-name">Name</label>
              <input
                id="site-name"
                v-model="form.name"
                placeholder="Sukhumvit Branch"
                class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
              />
              <p v-if="submitted && errors.name" class="mt-1 text-xs text-red-600">{{ errors.name }}</p>
            </div>

            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div>
                <label class="mb-1 block text-xs font-medium text-gray-700" for="site-address">Address</label>
                <input
                  id="site-address"
                  v-model="form.address"
                  class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
                />
              </div>
              <div>
                <label class="mb-1 block text-xs font-medium text-gray-700" for="site-area">Area</label>
                <input
                  id="site-area"
                  v-model="form.area"
                  placeholder="Bang Na"
                  class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
                />
              </div>
            </div>

            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div>
                <label class="mb-1 block text-xs font-medium text-gray-700" for="site-type">Property type</label>
                <select
                  id="site-type"
                  v-model="form.propertyType"
                  class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
                >
                  <option v-for="t in PROPERTY_TYPES" :key="t.value" :value="t.value">{{ t.label }}</option>
                </select>
              </div>
              <div>
                <label class="mb-1 block text-xs font-medium text-gray-700" for="site-status">Status</label>
                <select
                  id="site-status"
                  v-model="form.status"
                  class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
                >
                  <option value="active">Active</option>
                  <option value="inactive">Inactive</option>
                </select>
              </div>
            </div>

            <details class="text-xs">
              <summary class="cursor-pointer font-medium text-gray-600">
                Site contact override (optional — leave blank to use the customer's)
              </summary>
              <div class="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-3">
                <input
                  v-model="form.contactName"
                  placeholder="Contact name"
                  class="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
                />
                <input
                  v-model="form.contactPhone"
                  placeholder="Contact phone"
                  class="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
                />
                <input
                  v-model="form.contactEmail"
                  placeholder="Contact email"
                  class="rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
                />
              </div>
            </details>

            <div>
              <label class="mb-1 block text-xs font-medium text-gray-700" for="site-notes">Notes</label>
              <textarea
                id="site-notes"
                v-model="form.notes"
                rows="2"
                class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
              ></textarea>
            </div>

            <div class="flex justify-end gap-2 pt-1">
              <button
                type="button"
                class="rounded-lg px-3 py-1.5 text-sm font-medium text-gray-600 hover:bg-gray-100"
                @click="closeForm"
              >
                Cancel
              </button>
              <button
                type="submit"
                :disabled="saving"
                class="rounded-lg bg-navy-600 px-4 py-1.5 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50"
              >
                {{ saving ? "Saving…" : editingId != null ? "Save changes" : "Add site" }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>

  <ConfirmDialog
    :open="pendingDelete !== null"
    title="Delete site"
    :message="pendingDelete ? `Are you sure you want to delete ${pendingDelete.name}? Bookings already linked to it are unaffected.` : ''"
    confirm-label="Delete site"
    :busy="deleting"
    @confirm="handleDelete"
    @cancel="pendingDelete = null"
  />
</template>
