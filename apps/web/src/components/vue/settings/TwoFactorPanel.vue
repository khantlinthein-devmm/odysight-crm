<script setup lang="ts">
import { onMounted, ref } from "vue";
import {
  disableTwoFactor,
  forgetTrustedDevices,
  getTwoFactorStatus,
  regenerateBackupCodes,
  type TwoFactorStatus,
} from "../../../lib/twoFactor";
import { showToast } from "../../../lib/toast";
import TwoFactorSetup from "./TwoFactorSetup.vue";
import BackupCodes from "./BackupCodes.vue";

const status = ref<TwoFactorStatus | null>(null);
const mode = ref<"idle" | "setup" | "regenerate" | "disable">("idle");
const code = ref("");
const busy = ref(false);
const newCodes = ref<string[] | null>(null);

async function load() {
  try {
    status.value = await getTwoFactorStatus();
  } catch {
    status.value = null;
  }
}

function start(m: typeof mode.value) {
  code.value = "";
  newCodes.value = null;
  mode.value = m;
}

async function submitCode() {
  if (!code.value.trim()) return;
  busy.value = true;
  try {
    if (mode.value === "regenerate") {
      newCodes.value = await regenerateBackupCodes(code.value);
    } else if (mode.value === "disable") {
      await disableTwoFactor(code.value);
      showToast("Two-factor authentication turned off", "success");
      mode.value = "idle";
      await load();
    }
  } catch (e) {
    showToast(e instanceof Error ? e.message : "Code not accepted", "error");
  } finally {
    busy.value = false;
  }
}

async function forget() {
  try {
    await forgetTrustedDevices();
    showToast("Remembered devices cleared — they will ask for a code again", "success");
    await load();
  } catch (e) {
    showToast(e instanceof Error ? e.message : "Failed", "error");
  }
}

async function finished() {
  mode.value = "idle";
  newCodes.value = null;
  await load();
}

onMounted(load);
</script>

<template>
  <div class="mt-8 border-t border-gray-100 pt-6">
    <h3 class="text-sm font-semibold text-gray-900">Two-factor authentication</h3>
    <p class="mt-1 max-w-lg text-sm text-gray-500">
      Sign-in asks for a code from an authenticator app on your phone (Google Authenticator or
      Microsoft Authenticator) as well as your password.
    </p>

    <p v-if="!status" class="mt-3 text-sm text-gray-400">Loading…</p>

    <template v-else-if="mode === 'setup'">
      <div class="mt-4 max-w-lg">
        <TwoFactorSetup cancellable @done="finished" @cancel="mode = 'idle'" />
      </div>
    </template>

    <template v-else-if="!status.enabled">
      <p class="mt-3 inline-flex items-center gap-2 rounded-full bg-amber-50 px-3 py-1 text-xs font-medium text-amber-800">Off</p>
      <div class="mt-3">
        <button type="button" class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700" @click="start('setup')">
          Set up two-factor authentication
        </button>
      </div>
    </template>

    <template v-else>
      <div class="mt-3 flex flex-wrap items-center gap-2 text-sm">
        <span class="inline-flex items-center gap-1 rounded-full bg-green-50 px-3 py-1 text-xs font-medium text-green-800">✓ On</span>
        <span v-if="status.required" class="text-xs text-gray-500">Required for your role</span>
      </div>
      <p class="mt-2 text-sm text-gray-600">
        Backup codes left: <span class="font-semibold" :class="status.backupCodesLeft <= 2 ? 'text-red-600' : 'text-gray-900'">{{ status.backupCodesLeft }}</span>
        · Remembered devices: <span class="font-semibold text-gray-900">{{ status.trustedDevices }}</span>
      </p>

      <div v-if="mode === 'regenerate' && newCodes" class="mt-4 max-w-lg">
        <BackupCodes :codes="newCodes" @done="finished" />
      </div>
      <form v-else-if="mode === 'regenerate' || mode === 'disable'" class="mt-4 flex max-w-lg flex-wrap items-center gap-2" @submit.prevent="submitCode">
        <span class="w-full text-sm text-gray-600">
          {{ mode === "disable" ? "Enter a current code to turn two-factor off:" : "Enter a current code to make new backup codes (the old ones stop working):" }}
        </span>
        <input v-model="code" type="text" inputmode="numeric" autocomplete="one-time-code" maxlength="11" placeholder="123 456"
          class="w-40 rounded-lg border border-gray-200 px-3 py-2 text-center tracking-widest focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500" />
        <button type="submit" :disabled="busy"
          class="rounded-lg px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
          :class="mode === 'disable' ? 'bg-red-600 hover:bg-red-700' : 'bg-navy-600 hover:bg-navy-700'">
          {{ mode === "disable" ? "Turn off" : "Make new codes" }}
        </button>
        <button type="button" class="rounded-lg border border-gray-200 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100" @click="mode = 'idle'">Cancel</button>
      </form>
      <div v-else class="mt-4 flex flex-wrap gap-2">
        <button type="button" class="rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-700 hover:bg-gray-100" @click="start('regenerate')">New backup codes</button>
        <button v-if="status.trustedDevices > 0" type="button" class="rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-700 hover:bg-gray-100" @click="forget">Forget remembered devices</button>
        <button v-if="!status.required" type="button" class="rounded-lg border border-red-200 px-3 py-2 text-sm text-red-600 hover:bg-red-50" @click="start('disable')">Turn off</button>
      </div>
    </template>
  </div>
</template>
