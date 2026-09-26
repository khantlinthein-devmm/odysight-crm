<script setup lang="ts">
import { onMounted, ref } from "vue";
import { completeChecklistItem, confirmChecklist, createChecklist, getChecklistByBooking, uploadItemPhoto, type Checklist } from "../../../lib/checklists";
import { getBookings, type Booking } from "../../../lib/bookings";
import { isOfflineQueued } from "../../../lib/offline";
import { showToast } from "../../../lib/toast";
import { getSessionUser } from "../../../lib/auth";
import { t, type MessageKey } from "../../../lib/i18n";
import LanguageSwitcher from "../ui/LanguageSwitcher.vue";

// Cleaners see this page in their chosen language; office staff in English.
const isCleaner = getSessionUser()?.role === "CLEANER";
function L(key: MessageKey, english: string): string {
  return isCleaner ? t(key) : english;
}

const bookingId = ref<number | "">("");
const bookings = ref<Booking[]>([]);
const bookingsLoading = ref(true);
const checklist = ref<Checklist | null>(null);
const loading = ref(false);
const signature = ref("");
const confirming = ref(false);

async function handleLoad() {
  if (bookingId.value === "") {
    showToast("Select a booking", "error");
    return;
  }
  loading.value = true;
  try {
    checklist.value = await getChecklistByBooking(Number(bookingId.value));
  } catch (err) {
    showToast(err instanceof Error ? err.message : "No checklist for this booking", "error");
    checklist.value = null;
  } finally {
    loading.value = false;
  }
}

async function handleCreate() {
  if (bookingId.value === "") return;
  try {
    checklist.value = await createChecklist(Number(bookingId.value), 1);
    signature.value = "";
    showToast("Checklist created from template", "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to create checklist", "error");
  }
}

async function handleConfirm() {
  if (!checklist.value || !signature.value.trim()) {
    showToast(L("cl.needSignature", "Enter client name/signature to confirm"), "error");
    return;
  }
  confirming.value = true;
  try {
    checklist.value = await confirmChecklist(checklist.value.bookingId, signature.value.trim());
    showToast(L("cl.confirmed", "Checklist confirmed by client"), "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to confirm checklist", "error");
  } finally {
    confirming.value = false;
  }
}

async function toggle(itemId: number, done: boolean) {
  // Optimistic tick: the queued PATCH replays the same set-semantics value.
  const prev = checklist.value;
  if (prev) {
    checklist.value = {
      ...prev,
      items: prev.items.map((it) =>
        it.id === itemId ? { ...it, isCompleted: !done } : it,
      ),
    };
  }
  try {
    checklist.value = await completeChecklistItem(itemId, !done);
  } catch (err) {
    if (isOfflineQueued(err)) {
      showToast("Saved offline — tick will sync when online", "success");
    } else {
      checklist.value = prev;
      showToast(err instanceof Error ? err.message : "Failed to update item", "error");
    }
  }
}

async function attach(itemId: number, kind: "before" | "after", ev: Event) {
  const input = ev.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (!file) return;
  try {
    checklist.value = await uploadItemPhoto(itemId, kind, file);
    showToast(L("cl.photoAttached", "Photo attached"), "success");
  } catch (err) {
    if (isOfflineQueued(err)) {
      // Offline preview from the on-device file; the queued upload replaces
      // it with the server URL on sync.
      const url = URL.createObjectURL(file);
      if (checklist.value) {
        checklist.value = {
          ...checklist.value,
          items: checklist.value.items.map((it) =>
            it.id === itemId
              ? { ...it, ...(kind === "before" ? { beforePhotoUrl: url } : { afterPhotoUrl: url }) }
              : it,
          ),
        };
      }
      showToast("Photo saved offline — will upload when online", "success");
    } else {
      showToast(err instanceof Error ? err.message : "Failed to upload photo", "error");
    }
  }
}

onMounted(async () => {
  try {
    bookings.value = await getBookings({ limit: 100 });
  } catch {
    /* empty select: API error surfaces on load attempt */
  } finally {
    bookingsLoading.value = false;
  }
  const deep = new URLSearchParams(window.location.search).get("booking");
  if (deep) {
    bookingId.value = Number(deep) || "";
    if (bookingId.value !== "") await handleLoad();
  }
});
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-white">
    <div class="border-b border-gray-200 px-6 py-4">
      <LanguageSwitcher v-if="isCleaner" class="mb-2" />
      <h2 class="text-base font-semibold text-gray-900">{{ L("cl.title", "Checklists & proof of work") }}</h2>
      <p class="mt-1 text-sm text-gray-500">{{ L("cl.subtitle", "Optional per booking. Tick each item and attach before/after photos.") }}</p>
      <div class="mt-3 flex flex-col gap-2 sm:flex-row">
        <select v-model="bookingId" :disabled="bookingsLoading" class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm sm:w-80" @change="handleLoad">
          <option value="" disabled>{{ bookingsLoading ? L("cl.loadingBookings", "Loading bookings…") : L("cl.selectBooking", "Select booking") }}</option>
          <option v-for="b in bookings" :key="b.id" :value="b.id">
            {{ b.bookingNumber }} — {{ b.customerName }}
          </option>
        </select>
        <button type="button" class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white" @click="handleCreate">{{ L("cl.newFromTemplate", "New from template") }}</button>
      </div>
    </div>
    <div v-if="loading" class="px-6 py-10 text-center text-sm text-gray-500">{{ L("cl.loading", "Loading…") }}</div>
    <div v-else-if="checklist" class="px-6 py-4">
      <p class="text-sm text-gray-500">Booking #{{ checklist.bookingId }} · <span class="font-medium text-gray-900">{{ checklist.status }}</span></p>
      <ul class="mt-3 space-y-2">
        <li v-for="it in checklist.items" :key="it.id" class="flex flex-wrap items-center gap-3 rounded-lg border border-gray-100 px-3 py-3 sm:py-2">
          <input type="checkbox" :checked="it.isCompleted" class="field-tap-sm h-6 w-6 accent-navy-600 sm:h-4 sm:w-4" @change="toggle(it.id, it.isCompleted)" />
          <span class="min-w-0 flex-1 text-base sm:text-sm" :class="it.isCompleted ? 'text-gray-400 line-through' : 'text-gray-900'">{{ it.label }}</span>
          <span class="ml-auto flex items-center gap-3 text-sm sm:gap-2 sm:text-xs">
            <span v-if="it.beforePhotoUrl" class="text-gray-400">before ✓</span>
            <span v-if="it.afterPhotoUrl" class="text-gray-400">after ✓</span>
            <label class="field-tap-sm inline-flex cursor-pointer items-center font-medium text-navy-600">{{ L("cl.before", "Before") }}<input type="file" accept="image/jpeg,image/png,image/webp" class="hidden" @change="attach(it.id, 'before', $event)" /></label>
            <label class="field-tap-sm inline-flex cursor-pointer items-center font-medium text-navy-600">{{ L("cl.after", "After") }}<input type="file" accept="image/jpeg,image/png,image/webp" class="hidden" @change="attach(it.id, 'after', $event)" /></label>
          </span>
        </li>
      </ul>
      <p v-if="checklist.items.length === 0" class="mt-2 text-sm text-gray-500">{{ L("cl.noItems", "No items yet.") }}</p>
      <div class="mt-4 rounded-lg border border-gray-100 bg-gray-50 p-4">
        <h3 class="text-sm font-semibold text-gray-900">{{ L("cl.clientConfirm", "Client confirmation") }}</h3>
        <p v-if="checklist.clientSignature" class="mt-1 text-sm text-green-700">{{ L("cl.signedBy", "Signed by") }} {{ checklist.clientSignature }}</p>
        <div v-else class="mt-2 flex flex-col gap-2 sm:flex-row">
          <input v-model="signature" type="text" :placeholder="L('cl.signaturePlaceholder', 'Client name / signature')" class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm sm:w-80" />
          <button type="button" :disabled="confirming" class="rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" @click="handleConfirm">{{ confirming ? L("cl.confirming", "Confirming…") : L("cl.confirm", "Confirm") }}</button>
        </div>
      </div>
    </div>
    <div v-else class="px-6 py-10 text-center text-sm text-gray-500">{{ L("cl.empty", "Load a booking checklist, or create one from the default template.") }}</div>
  </div>
</template>
