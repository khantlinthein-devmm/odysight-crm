<script setup lang="ts">
import { ref } from "vue";
import { showToast } from "../../../lib/toast";

// One-time display of backup codes, with copy/download and an explicit
// "I saved them" before continuing.
const props = defineProps<{ codes: string[] }>();
const emit = defineEmits<{ done: [] }>();
const saved = ref(false);

function text(): string {
  return [
    "Smile Clean — two-factor backup codes",
    "Each code works once. Keep them somewhere safe (not on the same phone).",
    "",
    ...props.codes,
  ].join("\n");
}

async function copy() {
  try {
    await navigator.clipboard.writeText(text());
    showToast("Backup codes copied", "success");
  } catch {
    showToast("Copy failed — write the codes down", "error");
  }
}

function download() {
  const url = URL.createObjectURL(new Blob([text()], { type: "text/plain" }));
  const a = document.createElement("a");
  a.href = url;
  a.download = "smile-clean-backup-codes.txt";
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}
</script>

<template>
  <div class="space-y-3">
    <p class="rounded-lg bg-green-50 px-3 py-2 text-sm font-medium text-green-800">
      ✓ Two-factor authentication is on.
    </p>
    <p class="text-sm text-gray-600">
      Save these <span class="font-semibold">backup codes</span>. If you lose your phone, each code
      signs you in once. They are shown only now.
    </p>
    <ul class="grid grid-cols-2 gap-2 rounded-lg border border-gray-200 bg-gray-50 p-3 font-mono text-sm text-gray-900 sm:grid-cols-2">
      <li v-for="c in codes" :key="c" class="select-all">{{ c }}</li>
    </ul>
    <div class="flex flex-wrap gap-2">
      <button type="button" class="rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-700 hover:bg-gray-100" @click="copy">Copy</button>
      <button type="button" class="rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-700 hover:bg-gray-100" @click="download">Download .txt</button>
    </div>
    <label class="flex items-center gap-2 text-sm text-gray-700">
      <input v-model="saved" type="checkbox" class="h-4 w-4" />
      I have saved my backup codes
    </label>
    <button
      type="button"
      :disabled="!saved"
      class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50"
      @click="emit('done')"
    >Continue</button>
  </div>
</template>
