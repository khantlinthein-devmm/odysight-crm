<script setup lang="ts">
import { getSessionUser, logout } from "../../../lib/auth";
import TwoFactorSetup from "./TwoFactorSetup.vue";

// Full-page enrolment shown to office users who have not set up 2FA yet;
// nothing else in the app opens until this is done.
const user = getSessionUser();

function finish() {
  window.location.href = user?.role === "CLEANER" ? "/bookings" : "/dashboard";
}
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
    <h1 class="text-lg font-semibold text-gray-900">Set up two-factor authentication</h1>
    <p class="mt-1 text-sm text-gray-500">
      Your role <span class="font-medium text-gray-700">{{ user?.role ?? "" }}</span> handles customer data
      and money, so sign-in needs a code from your phone as well as your password. This takes about a minute.
    </p>
    <div class="mt-5">
      <TwoFactorSetup @done="finish" />
    </div>
    <button type="button" class="mt-6 text-sm text-gray-500 hover:text-gray-800" @click="logout()">Sign out</button>
  </div>
</template>
