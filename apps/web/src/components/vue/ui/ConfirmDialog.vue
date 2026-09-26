<script setup lang="ts">
import { useId } from "vue";
import { useModalA11y } from "./useModalA11y";

// `open` defaults to true so callers can mount the dialog with v-if alone;
// callers that keep it mounted pass :open explicitly.
const props = withDefaults(
  defineProps<{
    open?: boolean;
    title: string;
    message?: string;
    confirmLabel?: string;
    busy?: boolean;
  }>(),
  { open: true },
);

const emit = defineEmits<{
  confirm: [];
  cancel: [];
}>();

const titleId = useId();
const { container } = useModalA11y(() => emit("cancel"));
</script>

<template>
  <div
    v-if="props.open"
    ref="container"
    role="alertdialog"
    aria-modal="true"
    :aria-labelledby="titleId"
    class="fixed inset-0 z-50 flex items-center justify-center p-4"
  >
    <div class="absolute inset-0 bg-black/50" @click="emit('cancel')"></div>

    <div class="relative w-full max-w-sm rounded-xl bg-white p-6 shadow-xl">
      <h2 :id="titleId" class="text-base font-semibold text-gray-900">
        {{ title }}
      </h2>
      <p v-if="message" class="mt-2 text-sm text-gray-600">{{ message }}</p>
      <slot />

      <div class="mt-5 flex justify-end gap-3">
        <button
          type="button"
          class="rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100"
          @click="emit('cancel')"
        >
          Cancel
        </button>
        <button
          type="button"
          :disabled="busy"
          class="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
          @click="emit('confirm')"
        >
          {{ confirmLabel ?? "Delete" }}
        </button>
      </div>
    </div>
  </div>
</template>
