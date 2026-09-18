<script setup lang="ts">
import { computed, useId } from "vue";
import {
  areaPath,
  gridlines,
  smoothPath,
  trendLayout,
} from "../../../lib/charts";

const props = withDefaults(
  defineProps<{
    values: number[];
    labels: string[];
    formatValue?: (n: number) => string;
    lineColor?: string;
    width?: number;
    height?: number;
  }>(),
  {
    formatValue: (n: number) => String(n),
    lineColor: "#2563eb",
    width: 640,
    height: 250,
  },
);

const gid = useId().replace(/[^a-zA-Z0-9_-]/g, "");

const layout = computed(() =>
  trendLayout(props.values, props.width, props.height),
);
const line = computed(() => smoothPath(layout.value.points));
const area = computed(() => {
  const pts = layout.value.points;
  if (pts.length === 0) return "";
  return areaPath(line.value, pts[0]!.x, pts[pts.length - 1]!.x, props.height - 30);
});
const lines = computed(() =>
  gridlines(layout.value.min, layout.value.max, 4).map((v) => ({
    value: v,
    y: 18 + (1 - (v - layout.value.min) / ((layout.value.max - layout.value.min) || 1)) * (props.height - 18 - 30),
  })),
);
</script>

<template>
  <svg
    :viewBox="`0 0 ${width} ${height}`"
    class="h-auto w-full"
    role="img"
    aria-label="Revenue trend chart"
  >
    <defs>
      <linearGradient :id="`${gid}-fill`" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" :stop-color="lineColor" stop-opacity="0.28" />
        <stop offset="100%" :stop-color="lineColor" stop-opacity="0.02" />
      </linearGradient>
    </defs>

    <g v-for="g in lines" :key="g.value">
      <line
        x1="28"
        :x2="width - 8"
        :y1="g.y"
        :y2="g.y"
        class="stroke-gray-100"
        stroke-width="1"
        stroke-dasharray="3 4"
      />
      <text x="0" :y="g.y + 3.5" font-size="10" class="fill-gray-400">
        {{ formatValue(Math.round(g.value)) }}
      </text>
    </g>

    <path :d="area" :fill="`url(#${gid}-fill)`" />
    <path
      :d="line"
      fill="none"
      :stroke="lineColor"
      stroke-width="2.5"
      stroke-linecap="round"
      stroke-linejoin="round"
      class="drop-shadow-sm"
    />

    <g v-for="(p, i) in layout.points" :key="i">
      <circle
        :cx="p.x"
        :cy="p.y"
        r="4.5"
        :stroke="lineColor"
        stroke-width="2.5"
        class="fill-white"
      >
        <title>{{ labels[i] }}: {{ formatValue(values[i] ?? 0) }}</title>
      </circle>
      <text
        :x="p.x"
        :y="height - 12"
        text-anchor="middle"
        font-size="11"
        class="fill-gray-400"
      >
        {{ labels[i] }}
      </text>
    </g>
  </svg>
</template>
