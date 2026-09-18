<script setup lang="ts">
import { computed, useId } from "vue";
import { donutSlices, type DonutSegment } from "../../../lib/charts";

const props = withDefaults(
  defineProps<{
    segments: DonutSegment[];
    size?: number;
    thickness?: number;
    centerTop?: string;
    centerBottom?: string;
    /** Force vertical stacking (for narrow columns). */
    compact?: boolean;
  }>(),
  { size: 200, thickness: 30, centerTop: "", centerBottom: "", compact: false },
);

const gid = useId().replace(/[^a-zA-Z0-9_-]/g, "");

const slices = computed(() => donutSlices(props.segments));
const total = computed(() =>
  props.segments.reduce(
    (s, g) => s + (Number.isFinite(g.value) && g.value > 0 ? g.value : 0),
    0,
  ),
);

const radius = computed(() => (props.size - props.thickness) / 2);
const center = computed(() => props.size / 2);
</script>

<template>
  <div
    class="flex flex-col items-center gap-5"
    :class="compact ? '' : 'sm:flex-row sm:gap-7'"
  >
    <div class="relative shrink-0" :style="{ width: size + 'px', height: size + 'px' }">
      <svg
        :viewBox="`0 0 ${size} ${size}`"
        :width="size"
        :height="size"
        role="img"
        :aria-label="centerTop || 'Distribution chart'"
      >
        <circle
          :cx="center"
          :cy="center"
          :r="radius"
          fill="none"
          class="stroke-gray-100"
          :stroke-width="thickness"
        />
        <circle
          v-for="s in slices"
          :key="s.label"
          :cx="center"
          :cy="center"
          :r="radius"
          fill="none"
          :stroke="s.color"
          :stroke-width="thickness"
          pathLength="100"
          :stroke-dasharray="`${s.length} ${100 - s.length}`"
          :stroke-dashoffset="s.offset"
          stroke-linecap="butt"
          class="transition-all duration-700"
        >
          <title>{{ s.label }}: {{ s.value }} ({{ s.percent.toFixed(1) }}%)</title>
        </circle>
      </svg>
      <div
        class="pointer-events-none absolute inset-0 flex flex-col items-center justify-center text-center"
      >
        <p class="text-2xl font-semibold tracking-tight text-gray-900">
          {{ centerTop || total }}
        </p>
        <p v-if="centerBottom" class="mt-0.5 max-w-[110px] text-[11px] leading-tight text-gray-500">
          {{ centerBottom }}
        </p>
      </div>
    </div>

    <ul
      class="grid w-full flex-1 grid-cols-1 gap-2"
      :class="compact ? '' : 'sm:grid-cols-2 lg:grid-cols-1 xl:grid-cols-2'"
    >
      <li
        v-for="s in slices"
        :key="s.label"
        class="flex items-center gap-2.5 rounded-xl bg-gray-50/70 px-3 py-2 ring-1 ring-gray-100"
      >
        <span
          class="h-3 w-3 shrink-0 rounded-full shadow-sm"
          :style="{ backgroundColor: s.color }"
        ></span>
        <span class="min-w-0 flex-1 truncate text-[13px] text-gray-600">{{ s.label }}</span>
        <span class="text-[13px] font-semibold text-gray-900">{{ s.value }}</span>
        <span class="w-12 shrink-0 text-right text-xs tabular-nums text-gray-400">
          {{ s.percent.toFixed(0) }}%
        </span>
      </li>
    </ul>
  </div>
</template>
