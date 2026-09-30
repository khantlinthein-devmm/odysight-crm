<script setup lang="ts">
import { onMounted, ref } from "vue";
import { chatFileUrl, type ChatMessage } from "../../../lib/chat";

// Photos and voice notes are private files: load them with the session
// cookie, then show them from an object URL.
const props = defineProps<{ message: ChatMessage; mine: boolean }>();
const emit = defineEmits<{ loaded: [] }>();

const src = ref<string | null>(null);
const failed = ref(false);

onMounted(async () => {
  if (!props.message.fileUrl) return;
  try {
    src.value = await chatFileUrl(props.message.fileUrl);
  } catch {
    failed.value = true;
  }
});

function duration(ms?: number): string {
  if (!ms) return "";
  const s = Math.round(ms / 1000);
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
}
</script>

<template>
  <div v-if="message.kind === 'image'">
    <a v-if="src" :href="src" target="_blank" rel="noopener">
      <img
        :src="src"
        alt=""
        class="max-h-72 w-auto max-w-full rounded-lg object-contain"
        @load="emit('loaded')"
      />
    </a>
    <div v-else class="flex h-40 w-56 items-center justify-center rounded-lg bg-black/5 text-xs opacity-70">
      {{ failed ? "Photo unavailable" : "…" }}
    </div>
  </div>
  <div v-else-if="message.kind === 'voice'" class="flex items-center gap-2">
    <audio v-if="src" :src="src" controls preload="metadata" class="h-10 w-56 max-w-full" />
    <span v-else class="text-xs opacity-70">{{ failed ? "Voice unavailable" : "…" }}</span>
    <span v-if="message.durationMs" class="text-xs" :class="mine ? 'text-white/80' : 'text-gray-500'">
      {{ duration(message.durationMs) }}
    </span>
  </div>
</template>
