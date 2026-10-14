<script setup lang="ts">
import { ref, watch } from "vue";
import { checklistPhotoUrl } from "../../../lib/checklists";
import AppIcon from "../ui/AppIcon.vue";

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
    class="group relative h-16 w-16 shrink-0 overflow-hidden rounded-2xl bg-gray-100 shadow-sm ring-1 ring-gray-200"
    :aria-label="label"
    @click="src && (open = true)"
  >
    <img v-if="src" :src="src" :alt="label" class="h-full w-full object-cover transition group-hover:scale-105" />
    <span v-else class="flex h-full items-center justify-center text-gray-300">
      <AppIcon :name="failed ? 'warning' : 'photo'" :size="20" />
    </span>
    <span class="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/70 to-transparent pb-0.5 pt-3 text-center text-[9px] font-bold uppercase tracking-wide text-white">{{ label }}</span>
  </button>
  <Teleport to="body">
    <div
      v-if="open && src"
      class="fixed inset-0 z-[60] flex flex-col items-center justify-center bg-gray-950/90 p-4 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      :aria-label="label"
      @click="open = false"
    >
      <button type="button" class="absolute right-4 top-4 flex h-10 w-10 items-center justify-center rounded-full bg-white/10 text-white ring-1 ring-white/20 hover:bg-white/20" aria-label="Close">
        <AppIcon name="x" :size="20" />
      </button>
      <img :src="src" :alt="label" class="max-h-[85vh] max-w-full rounded-2xl shadow-2xl" @click.stop />
      <p class="mt-3 rounded-full bg-white/10 px-3 py-1 text-xs font-semibold uppercase tracking-wide text-white">{{ label }}</p>
    </div>
  </Teleport>
</template>
