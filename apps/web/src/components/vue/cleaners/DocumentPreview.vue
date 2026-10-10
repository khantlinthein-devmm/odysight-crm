<script setup lang="ts">
import { computed, useId } from "vue";
import { useModalA11y } from "../ui/useModalA11y";

// In-page viewer for a decrypted document (blob: URL owned by the parent).
// PDFs render in the browser's built-in viewer; images scale to fit. Mobile
// browsers often can't render PDFs inline, so "Open in new tab" and
// "Download" are always offered.
const props = defineProps<{
  url: string;
  contentType: string;
  title: string;
  fileName: string;
}>();

const emit = defineEmits<{ close: [] }>();

const titleId = useId();
const { container } = useModalA11y(() => emit("close"));

const isImage = computed(() => props.contentType.startsWith("image/"));
const isPdf = computed(() => props.contentType === "application/pdf");
</script>

<template>
  <div
    ref="container"
    role="dialog"
    aria-modal="true"
    :aria-labelledby="titleId"
    class="fixed inset-0 z-50 flex items-center justify-center p-2 sm:p-6"
  >
    <div class="absolute inset-0 bg-black/60" @click="emit('close')"></div>

    <div class="relative flex h-full max-h-[92vh] w-full max-w-4xl flex-col overflow-hidden rounded-xl bg-white shadow-xl">
      <div class="flex flex-wrap items-center justify-between gap-2 border-b border-gray-200 px-4 py-3">
        <h2 :id="titleId" class="min-w-0 truncate text-sm font-semibold text-gray-900">{{ title }}</h2>
        <div class="flex items-center gap-2">
          <a
            :href="url"
            target="_blank"
            rel="noopener"
            class="rounded-lg border border-gray-200 px-3 py-1.5 text-xs font-medium text-gray-700 hover:bg-gray-100"
          >
            Open in new tab
          </a>
          <a
            :href="url"
            :download="fileName"
            class="rounded-lg border border-gray-200 px-3 py-1.5 text-xs font-medium text-gray-700 hover:bg-gray-100"
          >
            Download
          </a>
          <button
            type="button"
            class="rounded-lg bg-navy-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-navy-700"
            @click="emit('close')"
          >
            Close
          </button>
        </div>
      </div>

      <div class="min-h-0 flex-1 bg-gray-100">
        <img v-if="isImage" :src="url" :alt="title" class="h-full w-full object-contain" />
        <iframe v-else-if="isPdf" :src="url" :title="title" class="h-full w-full border-0"></iframe>
        <p v-else class="p-8 text-center text-sm text-gray-500">Preview not available for this file type — use Download.</p>
      </div>
    </div>
  </div>
</template>
