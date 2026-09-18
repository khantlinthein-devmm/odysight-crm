<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { serviceLabel } from "../../../lib/settings";
import {
  clearPortalCustomer,
  getPortalCustomer,
  portalBookings,
  portalChangePassword,
  portalLogin,
  portalLogout,
  portalMe,
  portalSubmitFeedback,
  type PortalBooking,
  type PortalCustomer,
} from "../../../lib/portal";

const customer = ref<PortalCustomer | null>(null);
const bookings = ref<PortalBooking[]>([]);
const loading = ref(true);
const busy = ref(false);
const error = ref("");
const activeTab = ref<"bookings" | "profile">("bookings");
const isDev = import.meta.env.DEV;

const email = ref("");
const password = ref("");

const currentPw = ref("");
const newPw = ref("");
const pwMessage = ref("");

const feedbackFor = ref<PortalBooking | null>(null);
const feedbackRating = ref(5);
const feedbackComment = ref("");
const feedbackSaving = ref(false);
const feedbackError = ref("");

const statusStyles: Record<string, string> = {
  pending: "bg-amber-50 text-amber-700",
  confirmed: "bg-blue-50 text-blue-700",
  in_progress: "bg-indigo-50 text-indigo-700",
  completed: "bg-green-50 text-green-700",
  cancelled: "bg-red-50 text-red-700",
  no_show: "bg-gray-100 text-gray-600",
  rescheduled: "bg-purple-50 text-purple-700",
};

const init = async () => {
  const stored = getPortalCustomer();
  if (!stored) {
    loading.value = false;
    return;
  }
  try {
    customer.value = await portalMe();
    bookings.value = await portalBookings();
  } catch {
    clearPortalCustomer();
    customer.value = null;
  } finally {
    loading.value = false;
  }
};

onMounted(init);

async function submitLogin() {
  error.value = "";
  busy.value = true;
  try {
    customer.value = await portalLogin(email.value, password.value);
    bookings.value = await portalBookings();
    try {
      localStorage.setItem("odysight_portal_customer", JSON.stringify(customer.value));
    } catch { /* ignore */ }
    password.value = "";
    activeTab.value = "bookings";
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Sign in failed";
  } finally {
    busy.value = false;
  }
}

async function logout() {
  await portalLogout().catch(() => { /* ignore */ });
  clearPortalCustomer();
  customer.value = null;
  bookings.value = [];
}

async function changePassword() {
  pwMessage.value = "";
  error.value = "";
  try {
    await portalChangePassword(currentPw.value, newPw.value);
    pwMessage.value = "Password updated";
    currentPw.value = "";
    newPw.value = "";
  } catch (e) {
    pwMessage.value = "";
    error.value = e instanceof Error ? e.message : "Failed to change password";
  }
}

const completedUnsubmitted = computed(() =>
  bookings.value.filter((b) => b.status === "completed"),
);

function statusLabel(s: string): string {
  return s.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    weekday: "short",
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function openFeedback(b: PortalBooking) {
  feedbackFor.value = b;
  feedbackRating.value = 5;
  feedbackComment.value = "";
  feedbackError.value = "";
}

async function submitFeedback() {
  if (!feedbackFor.value) return;
  feedbackSaving.value = true;
  feedbackError.value = "";
  try {
    await portalSubmitFeedback({
      bookingId: feedbackFor.value.id,
      rating: feedbackRating.value,
      comment: feedbackComment.value,
    });
    feedbackFor.value = null;
  } catch (e) {
    feedbackError.value = e instanceof Error ? e.message : "Failed to submit feedback";
  } finally {
    feedbackSaving.value = false;
  }
}

const inputClass =
  "w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 placeholder-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500";
</script>

<template>
  <div class="w-full max-w-3xl px-4">
    <div class="mb-8 flex flex-col items-center gap-3">
      <img
        src="/logo/logo.png"
        alt="Odysight logo"
        class="h-16 w-16 rounded-2xl bg-white object-contain shadow-sm ring-1 ring-gray-200"
      />
      <div class="text-center">
        <h1 class="text-2xl font-semibold text-gray-900">Customer Portal</h1>
        <p class="mt-1 text-sm text-gray-500">
          Track your cleanings, payments and appointments with Smile Clean.
        </p>
      </div>
    </div>

    <div v-if="loading" class="rounded-xl border border-gray-200 bg-white p-8">
      <div class="h-24 animate-pulse rounded-lg bg-navy-50" />
    </div>

    <!-- Login -->
    <div v-else-if="!customer" class="mx-auto max-w-md">
      <form
        class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm"
        @submit.prevent="submitLogin"
      >
        <div class="space-y-4">
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700">Email</label>
            <input
              v-model="email"
              type="email"
              required
              autocomplete="email"
              placeholder="you@example.com"
              :class="inputClass"
            />
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700">Password</label>
            <input
              v-model="password"
              type="password"
              required
              autocomplete="current-password"
              placeholder="••••••••"
              :class="inputClass"
            />
          </div>
          <p v-if="error" class="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
            {{ error }}
          </p>
          <button
            type="submit"
            :disabled="busy"
            class="w-full rounded-lg bg-navy-600 py-2.5 text-sm font-semibold text-white hover:bg-navy-700 disabled:opacity-50"
          >
            {{ busy ? "Signing in…" : "Sign in" }}
          </button>
          <p class="text-center text-xs text-gray-500">
            <a href="/" class="font-medium text-gray-600 hover:text-gray-900"
              >← Back to Smile Clean</a
            >
          </p>
        </div>
      </form>
      <p v-if="isDev" class="mt-4 text-center text-xs text-gray-400">
        Demo: somchai@smileclean.com / portal123 (mocks only)
      </p>
    </div>

    <!-- Dashboard -->
    <div v-else class="space-y-4">
      <div class="rounded-xl border border-gray-200 bg-white p-6">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 class="text-base font-semibold text-gray-900">
              Hello, {{ customer.name }}
            </h2>
            <p class="mt-0.5 text-sm text-gray-500">{{ customer.email }}</p>
          </div>
          <nav class="flex flex-wrap gap-2 text-sm">
            <button
              type="button"
              class="rounded-lg border border-gray-200 px-3 py-1.5 font-medium text-gray-700 hover:bg-gray-100"
              :class="activeTab === 'bookings' ? 'bg-navy-50 text-navy-700' : ''"
              @click="activeTab = 'bookings'"
            >
              My Bookings
            </button>
            <button
              type="button"
              class="rounded-lg border border-gray-200 px-3 py-1.5 font-medium text-gray-700 hover:bg-gray-100"
              :class="activeTab === 'profile' ? 'bg-navy-50 text-navy-700' : ''"
              @click="activeTab = 'profile'"
            >
              Profile
            </button>
            <button
              type="button"
              class="rounded-lg border border-red-200 px-3 py-1.5 font-medium text-red-600 hover:bg-red-50"
              @click="logout"
            >
              Log out
            </button>
          </nav>
        </div>
      </div>

      <!-- Bookings tab -->
      <div v-if="activeTab === 'bookings'" class="space-y-4">
        <div v-if="bookings.length === 0" class="rounded-xl border border-gray-200 bg-white p-12 text-center">
          <p class="text-sm font-medium text-gray-900">No bookings yet</p>
          <p class="mt-1 text-sm text-gray-500">
            When you book a cleaning, it will appear here.
          </p>
        </div>

        <div
          v-for="b in bookings"
          :key="b.id"
          class="rounded-xl border border-gray-200 bg-white p-6"
        >
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <div class="flex flex-wrap items-center gap-2">
                <h3 class="text-sm font-semibold text-gray-900">{{ serviceLabel(b.serviceType) }}</h3>
                <span
                  :class="[
                    'inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium',
                    statusStyles[b.status] ?? 'bg-gray-100 text-gray-600',
                  ]"
                >
                  {{ statusLabel(b.status) }}
                </span>
              </div>
              <p class="mt-1 text-sm text-gray-600">{{ b.bookingNumber }}</p>
            </div>
            <p class="text-sm font-medium text-navy-700">{{ formatDate(b.scheduledFor) }}</p>
          </div>
          <dl class="mt-4 grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
            <div>
              <dt class="text-xs font-medium uppercase tracking-wide text-gray-400">Address</dt>
              <dd class="text-gray-700">{{ b.address }}</dd>
            </div>
            <div>
              <dt class="text-xs font-medium uppercase tracking-wide text-gray-400">Cleaner</dt>
              <dd class="text-gray-700">{{ b.assignee || "To be assigned" }}</dd>
            </div>
            <div>
              <dt class="text-xs font-medium uppercase tracking-wide text-gray-400">Duration</dt>
              <dd class="text-gray-700">{{ b.durationMinutes }} minutes</dd>
            </div>
            <div v-if="b.notes">
              <dt class="text-xs font-medium uppercase tracking-wide text-gray-400">Notes</dt>
              <dd class="text-gray-700">{{ b.notes }}</dd>
            </div>
          </dl>
          <div v-if="b.status === 'completed'" class="mt-4 flex justify-end">
            <button
              type="button"
              class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700"
              @click="openFeedback(b)"
            >
              Leave a review
            </button>
          </div>
        </div>
      </div>

      <!-- Profile tab -->
      <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div class="rounded-xl border border-gray-200 bg-white p-6">
          <h3 class="text-sm font-semibold text-gray-900">Profile</h3>
          <dl class="mt-4 space-y-3 text-sm">
            <div>
              <dt class="text-xs font-medium uppercase tracking-wide text-gray-400">Name</dt>
              <dd class="text-gray-700">{{ customer.name }}</dd>
            </div>
            <div>
              <dt class="text-xs font-medium uppercase tracking-wide text-gray-400">Email</dt>
              <dd class="text-gray-700">{{ customer.email }}</dd>
            </div>
            <div>
              <dt class="text-xs font-medium uppercase tracking-wide text-gray-400">Phone</dt>
              <dd class="text-gray-700">{{ customer.phone }}</dd>
            </div>
            <div>
              <dt class="text-xs font-medium uppercase tracking-wide text-gray-400">Address</dt>
              <dd class="text-gray-700">{{ customer.address }}</dd>
            </div>
          </dl>
        </div>

        <div class="rounded-xl border border-gray-200 bg-white p-6">
          <h3 class="text-sm font-semibold text-gray-900">Change password</h3>
          <form class="mt-4 space-y-3" @submit.prevent="changePassword">
            <input
              v-model="currentPw"
              type="password"
              required
              placeholder="Current password"
              :class="inputClass"
            />
            <input
              v-model="newPw"
              type="password"
              required
              minlength="8"
              placeholder="New password (min 8 chars)"
              :class="inputClass"
            />
            <p v-if="error" class="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
              {{ error }}
            </p>
            <p v-if="pwMessage" class="rounded-lg border border-green-200 bg-green-50 px-3 py-2 text-sm text-green-700">
              {{ pwMessage }}
            </p>
            <button
              type="submit"
              class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700"
            >
              Update password
            </button>
          </form>
        </div>
      </div>
    </div>

    <!-- Feedback modal -->
    <div
      v-if="feedbackFor"
      class="fixed inset-0 z-50 flex items-center justify-center bg-navy-900/40 p-4"
      @click.self="feedbackFor = null"
    >
      <div class="w-full max-w-md rounded-xl bg-white p-6 shadow-xl">
        <h3 class="text-base font-semibold text-gray-900">
          Review {{ feedbackFor.bookingNumber }}
        </h3>
        <p class="mt-1 text-sm text-gray-500">
          How was your {{ serviceLabel(feedbackFor.serviceType) }}?
        </p>
        <form class="mt-4 space-y-4" @submit.prevent="submitFeedback">
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700">Rating</label>
            <div class="flex items-center gap-1">
              <button
                v-for="n in 5"
                :key="n"
                type="button"
                class="text-2xl focus:outline-none"
                :class="n <= feedbackRating ? 'text-amber-400' : 'text-gray-200'"
                @click="feedbackRating = n"
                :aria-label="`${n} star${n > 1 ? 's' : ''}`"
              >
                ★
              </button>
            </div>
          </div>
          <div>
            <label class="mb-1 block text-sm font-medium text-gray-700">Comment</label>
            <textarea
              v-model="feedbackComment"
              rows="3"
              placeholder="Tell us how it went…"
              :class="inputClass"
            />
          </div>
          <p v-if="feedbackError" class="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
            {{ feedbackError }}
          </p>
          <div class="flex justify-end gap-2">
            <button
              type="button"
              class="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100"
              @click="feedbackFor = null"
            >
              Cancel
            </button>
            <button
              type="submit"
              :disabled="feedbackSaving"
              class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50"
            >
              {{ feedbackSaving ? "Submitting…" : "Submit review" }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>