<script setup lang="ts">
import { onMounted, ref } from "vue";
import {
  createChecklistTemplate,
  getChecklistTemplates,
  updateChecklistTemplate,
  type ChecklistTemplate,
  type ItemType,
} from "../../../lib/checklists";
import { activeServices, getWorkspaceSettings, serviceLabel } from "../../../lib/settings";
import { showToast } from "../../../lib/toast";
import AppIcon, { type IconName } from "../ui/AppIcon.vue";

// Office template editor: each item is a tick, a 1-5 score or pass/fail,
// optionally needing an after photo. Edits only affect checklists created
// afterwards; ones already sent keep their copy.

interface DraftItem { label: string; itemType: ItemType; requirePhoto: boolean }
interface Draft { id: number | null; name: string; serviceType: string; isActive: boolean; items: DraftItem[] }

const templates = ref<ChecklistTemplate[]>([]);
const loading = ref(true);
const services = ref<{ id: string; name: string }[]>([]);
const draft = ref<Draft | null>(null);
const saving = ref(false);
const bulk = ref("");
const showBulk = ref(false);

const types: { value: ItemType; label: string; icon: IconName }[] = [
  { value: "check", label: "Tick", icon: "check" },
  { value: "score", label: "Score 1–5", icon: "star" },
  { value: "pass_fail", label: "Pass / Fail", icon: "badge" },
];

function countOf(t: ChecklistTemplate, type: ItemType): number {
  return t.items.filter((i) => (i.itemType ?? "check") === type).length;
}

async function load() {
  loading.value = true;
  try {
    templates.value = await getChecklistTemplates();
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to load templates", "error");
  } finally {
    loading.value = false;
  }
}

function startNew() {
  draft.value = { id: null, name: "", serviceType: "", isActive: true, items: [{ label: "", itemType: "check", requirePhoto: false }] };
  bulk.value = "";
  showBulk.value = false;
}

function edit(t: ChecklistTemplate) {
  draft.value = {
    id: t.id, name: t.name, serviceType: t.serviceType, isActive: t.isActive,
    items: t.items.map((i) => ({ label: i.label, itemType: i.itemType ?? "check", requirePhoto: !!i.requirePhoto })),
  };
  bulk.value = "";
  showBulk.value = false;
}

function addItem() {
  draft.value?.items.push({ label: "", itemType: "check", requirePhoto: false });
}

function addBulk() {
  if (!draft.value) return;
  const lines = bulk.value.split("\n").map((l) => l.trim()).filter(Boolean);
  draft.value.items = [
    ...draft.value.items.filter((i) => i.label.trim()),
    ...lines.map((label) => ({ label, itemType: "check" as ItemType, requirePhoto: false })),
  ];
  bulk.value = "";
  showBulk.value = false;
}

function move(i: number, dir: -1 | 1) {
  const items = draft.value?.items;
  if (!items) return;
  const j = i + dir;
  if (j < 0 || j >= items.length) return;
  [items[i], items[j]] = [items[j]!, items[i]!];
}

async function save() {
  const d = draft.value;
  if (!d) return;
  const items = d.items.map((i) => ({ ...i, label: i.label.trim() })).filter((i) => i.label);
  if (!d.name.trim() || items.length === 0) {
    showToast("Enter a template name and at least one item", "error");
    return;
  }
  saving.value = true;
  try {
    const input = { name: d.name.trim(), serviceType: d.serviceType, isActive: d.isActive, items };
    const saved = d.id == null ? await createChecklistTemplate(input) : await updateChecklistTemplate(d.id, input);
    templates.value = d.id == null
      ? [saved, ...templates.value]
      : templates.value.map((t) => (t.id === saved.id ? saved : t));
    draft.value = null;
    showToast("Template saved", "success");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to save template", "error");
  } finally {
    saving.value = false;
  }
}

onMounted(() => {
  void load();
  getWorkspaceSettings()
    .then(() => (services.value = activeServices().map((s) => ({ id: s.id, name: s.name }))))
    .catch(() => {});
});
</script>

<template>
  <div>
    <!-- Template gallery -->
    <div v-if="!draft" class="space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-3 rounded-3xl bg-white p-4 shadow-sm ring-1 ring-gray-100 sm:p-5">
        <p class="max-w-xl text-sm text-gray-500">The template that matches a booking's service is sent to the crew when they open the job.</p>
        <button type="button" class="inline-flex items-center gap-2 rounded-2xl bg-navy-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-navy-700" @click="startNew">
          <AppIcon name="plus" :size="17" /> New template
        </button>
      </div>

      <div v-if="loading" class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        <div v-for="i in 3" :key="i" class="h-40 animate-pulse rounded-3xl bg-white ring-1 ring-gray-100"></div>
      </div>
      <div v-else-if="templates.length === 0" class="flex flex-col items-center rounded-3xl bg-white px-6 py-14 text-center shadow-sm ring-1 ring-gray-100">
        <span class="flex h-14 w-14 items-center justify-center rounded-2xl bg-navy-50 text-navy-500"><AppIcon name="stack" :size="28" /></span>
        <p class="mt-3 text-sm text-gray-500">No templates yet.</p>
      </div>
      <div v-else class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        <button
          v-for="t in templates"
          :key="t.id"
          type="button"
          :class="['group flex flex-col rounded-3xl bg-white p-5 text-left shadow-sm ring-1 ring-gray-100 transition hover:-translate-y-0.5 hover:shadow-md', t.isActive ? '' : 'opacity-60']"
          @click="edit(t)"
        >
          <div class="flex items-start justify-between gap-3">
            <span class="flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-navy-600 to-blue-400 text-white shadow-sm">
              <AppIcon name="clipboard" :size="22" />
            </span>
            <span :class="['rounded-full px-2.5 py-1 text-[11px] font-bold', t.isActive ? 'bg-emerald-50 text-emerald-700' : 'bg-gray-100 text-gray-500']">{{ t.isActive ? "Active" : "Inactive" }}</span>
          </div>
          <p class="mt-4 font-semibold text-gray-900">{{ t.name }}</p>
          <p class="text-xs text-gray-500">{{ t.serviceType ? serviceLabel(t.serviceType) : "Any service (general)" }}</p>
          <div class="mt-4 flex flex-wrap gap-1.5">
            <span class="inline-flex items-center gap-1 rounded-full bg-gray-50 px-2.5 py-1 text-xs font-semibold text-gray-600 ring-1 ring-gray-100">{{ t.items.length }} items</span>
            <span v-if="countOf(t, 'score')" class="inline-flex items-center gap-1 rounded-full bg-amber-50 px-2.5 py-1 text-xs font-semibold text-amber-700"><AppIcon name="star" :size="12" /> {{ countOf(t, "score") }}</span>
            <span v-if="countOf(t, 'pass_fail')" class="inline-flex items-center gap-1 rounded-full bg-blue-50 px-2.5 py-1 text-xs font-semibold text-blue-700"><AppIcon name="badge" :size="12" /> {{ countOf(t, "pass_fail") }}</span>
            <span v-if="t.items.some((i) => i.requirePhoto)" class="inline-flex items-center gap-1 rounded-full bg-navy-50 px-2.5 py-1 text-xs font-semibold text-navy-700"><AppIcon name="camera" :size="12" /> {{ t.items.filter((i) => i.requirePhoto).length }}</span>
          </div>
          <span class="mt-4 inline-flex items-center gap-1 text-sm font-semibold text-navy-600">
            Edit <AppIcon name="chevronRight" :size="15" class="transition group-hover:translate-x-0.5" />
          </span>
        </button>
      </div>
    </div>

    <!-- Editor -->
    <div v-else class="space-y-4">
      <section class="rounded-3xl bg-white p-5 shadow-sm ring-1 ring-gray-100">
        <button type="button" class="inline-flex items-center gap-1.5 text-sm font-semibold text-gray-500 hover:text-navy-600" @click="draft = null">
          <AppIcon name="arrowLeft" :size="16" /> All templates
        </button>
        <div class="mt-4 grid gap-3 sm:grid-cols-[1fr_1fr_auto] sm:items-end">
          <label class="block">
            <span class="mb-1.5 block text-xs font-semibold uppercase tracking-wide text-gray-500">Template name</span>
            <input v-model="draft.name" type="text" placeholder="e.g. Restaurant kitchen night clean" class="w-full rounded-2xl border-0 bg-gray-50 px-3 py-2.5 text-sm ring-1 ring-gray-200 focus:ring-2 focus:ring-navy-500" />
          </label>
          <label class="block">
            <span class="mb-1.5 block text-xs font-semibold uppercase tracking-wide text-gray-500">Used for service</span>
            <select v-model="draft.serviceType" class="w-full text-sm">
              <option value="">Any service (general)</option>
              <option v-for="sv in services" :key="sv.id" :value="sv.id">{{ sv.name }}</option>
              <option v-if="draft.serviceType && !services.some((s) => s.id === draft!.serviceType)" :value="draft.serviceType">{{ serviceLabel(draft.serviceType) }}</option>
            </select>
          </label>
          <button
            type="button"
            role="switch"
            :aria-checked="draft.isActive"
            class="flex items-center gap-2.5 rounded-2xl px-3 py-2.5 text-sm font-semibold text-gray-700 ring-1 ring-gray-200"
            @click="draft.isActive = !draft.isActive"
          >
            <span :class="['relative h-6 w-11 rounded-full transition', draft.isActive ? 'bg-emerald-500' : 'bg-gray-300']">
              <span :class="['absolute top-0.5 h-5 w-5 rounded-full bg-white shadow transition-all', draft.isActive ? 'left-[22px]' : 'left-0.5']"></span>
            </span>
            {{ draft.isActive ? "Active" : "Inactive" }}
          </button>
        </div>
      </section>

      <ol class="space-y-2.5">
        <li v-for="(it, i) in draft.items" :key="i" class="rounded-3xl bg-white p-4 shadow-sm ring-1 ring-gray-100">
          <div class="flex items-start gap-3">
            <span class="mt-1.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-xl bg-navy-50 text-xs font-bold text-navy-700">{{ i + 1 }}</span>
            <div class="min-w-0 flex-1 space-y-3">
              <input v-model="it.label" type="text" class="w-full rounded-2xl border-0 bg-gray-50 px-3 py-2.5 text-sm ring-1 ring-gray-200 focus:ring-2 focus:ring-navy-500" placeholder="ကြမ်းပြင် တိုက်ပြီး / Floors swept and mopped" />
              <div class="flex flex-wrap items-center gap-2">
                <div class="inline-flex rounded-2xl bg-gray-50 p-1 ring-1 ring-gray-200" role="radiogroup" aria-label="Answer type">
                  <button
                    v-for="ty in types"
                    :key="ty.value"
                    type="button"
                    role="radio"
                    :aria-checked="it.itemType === ty.value"
                    :class="['inline-flex items-center gap-1.5 rounded-xl px-3 py-1.5 text-xs font-semibold transition', it.itemType === ty.value ? 'bg-white text-navy-700 shadow-sm ring-1 ring-gray-200' : 'text-gray-500 hover:text-gray-700']"
                    @click="it.itemType = ty.value"
                  >
                    <AppIcon :name="ty.icon" :size="14" :class="ty.value === 'score' && it.itemType === ty.value ? 'text-amber-400' : ''" />
                    {{ ty.label }}
                  </button>
                </div>
                <button
                  type="button"
                  :aria-pressed="it.requirePhoto"
                  :class="['inline-flex items-center gap-1.5 rounded-2xl px-3 py-2 text-xs font-semibold ring-1 transition', it.requirePhoto ? 'bg-navy-600 text-white ring-navy-600' : 'bg-white text-gray-500 ring-gray-200 hover:text-gray-700']"
                  @click="it.requirePhoto = !it.requirePhoto"
                >
                  <AppIcon name="camera" :size="14" /> After photo required
                </button>
              </div>
            </div>
            <div class="flex shrink-0 flex-col gap-1">
              <button type="button" class="flex h-8 w-8 items-center justify-center rounded-xl text-gray-400 hover:bg-gray-50 hover:text-gray-700 disabled:opacity-30" :disabled="i === 0" aria-label="Move up" @click="move(i, -1)"><AppIcon name="chevronUp" :size="16" /></button>
              <button type="button" class="flex h-8 w-8 items-center justify-center rounded-xl text-gray-400 hover:bg-gray-50 hover:text-gray-700 disabled:opacity-30" :disabled="i === draft.items.length - 1" aria-label="Move down" @click="move(i, 1)"><AppIcon name="chevronDown" :size="16" /></button>
              <button type="button" class="flex h-8 w-8 items-center justify-center rounded-xl text-gray-400 hover:bg-red-50 hover:text-red-600" aria-label="Remove" @click="draft.items.splice(i, 1)"><AppIcon name="trash" :size="16" /></button>
            </div>
          </div>
        </li>
      </ol>

      <div class="flex flex-wrap gap-2">
        <button type="button" class="inline-flex items-center gap-2 rounded-2xl border-2 border-dashed border-gray-200 px-4 py-2.5 text-sm font-semibold text-gray-600 hover:border-navy-300 hover:text-navy-600" @click="addItem">
          <AppIcon name="plus" :size="16" /> Add item
        </button>
        <button type="button" class="inline-flex items-center gap-2 rounded-2xl px-4 py-2.5 text-sm font-semibold text-gray-500 hover:text-navy-600" @click="showBulk = !showBulk">
          <AppIcon name="clipboard" :size="16" /> Paste many items
        </button>
      </div>
      <div v-if="showBulk" class="rounded-3xl bg-white p-4 shadow-sm ring-1 ring-gray-100">
        <textarea v-model="bulk" rows="4" placeholder="One item per line" class="w-full rounded-2xl border-0 bg-gray-50 px-3 py-2.5 text-sm ring-1 ring-gray-200 focus:ring-2 focus:ring-navy-500"></textarea>
        <button type="button" class="mt-2 rounded-xl bg-navy-600 px-3 py-1.5 text-sm font-semibold text-white" @click="addBulk">Add lines</button>
      </div>

      <div class="flex flex-wrap items-center gap-3 rounded-3xl bg-white p-4 shadow-sm ring-1 ring-gray-100">
        <button type="button" :disabled="saving" class="inline-flex items-center gap-2 rounded-2xl bg-emerald-600 px-5 py-2.5 text-sm font-semibold text-white shadow-sm shadow-emerald-600/30 hover:bg-emerald-700 disabled:opacity-50" @click="save">
          <AppIcon name="check" :size="17" :stroke="2.5" /> {{ saving ? "Saving…" : "Save template" }}
        </button>
        <button type="button" class="rounded-2xl px-4 py-2.5 text-sm font-semibold text-gray-600 ring-1 ring-gray-200 hover:bg-gray-50" @click="draft = null">Cancel</button>
        <p v-if="draft.id != null" class="text-xs text-gray-500">Changes apply to checklists created from now on. Checklists already sent keep their items.</p>
      </div>
    </div>
  </div>
</template>
