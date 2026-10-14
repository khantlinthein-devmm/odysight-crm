<script setup lang="ts">
import { ref, watch } from "vue";
import { checklistPhotoUrl } from "../../../lib/checklists";

// A private proof-of-work photo: thumbnail that opens full size.
const props = defineProps<{ url: string; label: string }>();
const src = ref<string | null>(null);
const failed = ref(false);
const open = ref(false);

watch(
  () => props.url,
  (url) => {
    src.value = null;
    failed.value = false;
    checklistPhotoUrl(url)
      .then((u) => (src.value = u))
      .catch(() => (failed.value = true));
  },
  { immediate: true },
);
</script>

<template>
  <button
    type="button"
    class="relative h-14 w-14 shrink-0 overflow-hidden rounded-lg bg-gray-100 ring-1 ring-gray-200"
    :aria-label="label"
    @click="src && (open = true)"
  >
    <img v-if="src" :src="src" :alt="label" class="h-full w-full object-cover" />
    <span v-else class="flex h-full items-center justify-center text-[10px] text-gray-400">{{ failed ? "✕" : "…" }}</span>
    <span class="absolute inset-x-0 bottom-0 bg-black/50 text-center text-[9px] font-semibold uppercase text-white">{{ label }}</span>
  </button>
  <Teleport to="body">
    <div v-if="open && src" class="fixed inset-0 z-[60] flex items-center justify-center bg-black/80 p-4" role="dialog" aria-modal="true" @click="open = false">
      <img :src="src" :alt="label" class="max-h-full max-w-full rounded-lg" />
    </div>
  </Teleport>
</template>
