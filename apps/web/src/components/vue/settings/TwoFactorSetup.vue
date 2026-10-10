<script setup lang="ts">
import { onMounted, ref } from "vue";
import { beginTwoFactorSetup, enableTwoFactor, type TwoFactorSetup } from "../../../lib/twoFactor";
import BackupCodes from "./BackupCodes.vue";

// Enrolment: scan the QR code with an authenticator app, confirm with a code,
// then save the backup codes. Emits "done" after the codes are saved.
const emit = defineEmits<{ done: []; cancel: [] }>();
defineProps<{ cancellable?: boolean }>();

const setup = ref<TwoFactorSetup | null>(null);
const code = ref("");
const busy = ref(false);
const error = ref("");
const showKey = ref(false);
const backupCodes = ref<string[] | null>(null);

async function start() {
  error.value = "";
  try {
    setup.value = await beginTwoFactorSetup();
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Could not start setup";
  }
}

async function confirm() {
  if (!code.value.trim()) return;
  busy.value = true;
  error.value = "";
  try {
    backupCodes.value = await enableTwoFactor(code.value);
  } catch (e) {
    error.value = e instanceof Error ? e.message : "Could not verify the code";
  } finally {
    busy.value = false;
  }
}

function groupedKey(k: string): string {
  return k.replace(/(.{4})/g, "$1 ").trim();
}

onMounted(start);
</script>

<template>
  <div class="space-y-4">
    <template v-if="!backupCodes">
      <ol class="list-decimal space-y-1 pl-5 text-sm text-gray-600">
        <li>
          Install an authenticator app on your phone:
          <span class="font-medium text-gray-800">Google Authenticator</span> or
          <span class="font-medium text-gray-800">Microsoft Authenticator</span> (free).
        </li>
        <li>In the app, tap <span class="font-medium text-gray-800">+</span> and scan this QR code.</li>
        <li>Type the 6-digit code the app shows.</li>
      </ol>

      <div v-if="setup" class="flex flex-col items-center gap-3 rounded-lg border border-gray-200 bg-gray-50 p-4 sm:flex-row sm:items-start">
        <img :src="setup.qr" alt="QR code for your authenticator app" width="180" height="180" class="h-44 w-44 rounded bg-white p-1" />
        <div class="min-w-0 text-sm text-gray-600">
          <p>Can’t scan? On the phone, choose “Enter a setup key” and type:</p>
          <button
            v-if="!showKey"
            type="button"
            class="mt-1 font-medium text-navy-700 hover:underline"
            @click="showKey = true"
          >Show setup key</button>
          <p v-else class="mt-1 break-all font-mono text-sm font-semibold text-gray-900 select-all">{{ groupedKey(setup.secret) }}</p>
          <p class="mt-2 text-xs text-gray-500">Account: Smile Clean · Time-based</p>
        </div>
      </div>
      <p v-else-if="!error" class="text-sm text-gray-400">Preparing…</p>

      <form class="flex flex-wrap items-center gap-2" @submit.prevent="confirm">
        <input
          v-model="code"
          type="text"
          inputmode="numeric"
          autocomplete="one-time-code"
          maxlength="7"
          placeholder="123 456"
          class="w-40 rounded-lg border border-gray-200 px-3 py-2 text-center text-lg tracking-widest focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
        />
        <button
          type="submit"
          :disabled="busy || !setup"
          class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50"
        >{{ busy ? "Checking…" : "Turn on" }}</button>
        <button
          v-if="cancellable"
          type="button"
          class="rounded-lg border border-gray-200 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
          @click="emit('cancel')"
        >Cancel</button>
      </form>
      <p v-if="error" class="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{{ error }}</p>
    </template>

    <BackupCodes v-else :codes="backupCodes" @done="emit('done')" />
  </div>
</template>
