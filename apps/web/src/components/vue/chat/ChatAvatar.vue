<script setup lang="ts">
import { computed } from "vue";

// Round avatar with initials (or a group icon) and a green dot when online.
const props = withDefaults(
  defineProps<{ name: string; online?: boolean; group?: boolean; size?: "sm" | "md" | "lg" }>(),
  { online: false, group: false, size: "md" },
);

// A stable colour per name so people are easy to tell apart.
const PALETTE = [
  "bg-navy-100 text-navy-700",
  "bg-emerald-100 text-emerald-700",
  "bg-violet-100 text-violet-700",
  "bg-sky-100 text-sky-700",
  "bg-rose-100 text-rose-700",
  "bg-teal-100 text-teal-700",
];

const colour = computed(() => {
  if (props.group) return "bg-amber-100 text-amber-700";
  let h = 0;
  for (const ch of props.name) h = (h * 31 + ch.charCodeAt(0)) >>> 0;
  return PALETTE[h % PALETTE.length];
});

const initials = computed(() =>
  props.name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((p) => p[0]!.toUpperCase())
    .join(""),
);

const box = computed(() => ({ sm: "h-8 w-8 text-xs", md: "h-10 w-10 text-sm", lg: "h-11 w-11 text-sm" })[props.size]);
const dot = computed(() => ({ sm: "h-2.5 w-2.5", md: "h-3 w-3", lg: "h-3.5 w-3.5" })[props.size]);
</script>

<template>
  <span class="relative inline-flex shrink-0">
    <span class="flex items-center justify-center rounded-full font-semibold" :class="[box, colour]" aria-hidden="true">
      <svg v-if="group" class="h-1/2 w-1/2" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" /></svg>
      <template v-else>{{ initials }}</template>
    </span>
    <span v-if="online" class="absolute bottom-0 right-0 rounded-full bg-green-500 ring-2 ring-white" :class="dot" title="Online" />
  </span>
</template>
