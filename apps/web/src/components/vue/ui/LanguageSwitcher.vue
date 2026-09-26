<script setup lang="ts">
import { LANGS, getLang, setLang, t, type Lang } from "../../../lib/i18n";

const current = getLang();

function choose(code: Lang) {
  if (code !== current) setLang(code);
}
</script>

<template>
  <div class="flex items-center justify-end gap-1" role="group" :aria-label="t('lang.label')">
    <button
      v-for="l in LANGS"
      :key="l.code"
      type="button"
      class="field-tap-sm rounded-full px-3 py-1 text-xs font-semibold"
      :class="l.code === current ? 'bg-navy-600 text-white' : 'bg-gray-100 text-gray-700 active:bg-gray-200'"
      :aria-pressed="l.code === current"
      @click="choose(l.code)"
    >
      {{ l.label }}
    </button>
  </div>
</template>
