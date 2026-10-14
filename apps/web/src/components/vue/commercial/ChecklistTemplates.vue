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

const typeLabels: Record<ItemType, string> = { check: "Tick", score: "Score 1–5", pass_fail: "Pass / Fail" };

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
}

function edit(t: ChecklistTemplate) {
  draft.value = {
    id: t.id, name: t.name, serviceType: t.serviceType, isActive: t.isActive,
    items: t.items.map((i) => ({ label: i.label, itemType: i.itemType ?? "check", requirePhoto: !!i.requirePhoto })),
  };
  bulk.value = "";
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
  <div class="px-6 py-4">
    <div v-if="!draft">
      <div class="flex items-center justify-between">
        <p class="text-sm text-gray-500">The template matching a booking's service is sent to the crew when they open the job.</p>
        <button type="button" class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white" @click="startNew">+ New template</button>
      </div>
      <div v-if="loading" class="py-8 text-center text-sm text-gray-500">Loading…</div>
      <ul v-else class="mt-4 divide-y divide-gray-100 rounded-lg border border-gray-200">
        <li v-for="t in templates" :key="t.id" class="flex flex-wrap items-center gap-3 px-4 py-3">
          <div class="min-w-0 flex-1">
            <p class="text-sm font-medium text-gray-900">{{ t.name }}
              <span v-if="!t.isActive" class="ml-1 rounded-full bg-gray-100 px-2 py-0.5 text-[11px] font-semibold text-gray-500">inactive</span>
            </p>
            <p class="text-xs text-gray-500">
              {{ t.serviceType ? serviceLabel(t.serviceType) : "Any service (general)" }} · {{ t.items.length }} items
              <template v-if="t.items.some((i) => i.itemType === 'score')"> · {{ t.items.filter((i) => i.itemType === "score").length }} scored</template>
              <template v-if="t.items.some((i) => i.requirePhoto)"> · {{ t.items.filter((i) => i.requirePhoto).length }} need photo</template>
            </p>
          </div>
          <button type="button" class="text-sm font-medium text-navy-600 hover:underline" @click="edit(t)">Edit</button>
        </li>
        <li v-if="templates.length === 0" class="px-4 py-6 text-center text-sm text-gray-500">No templates yet.</li>
      </ul>
    </div>

    <div v-else class="space-y-4">
      <div class="grid gap-3 sm:grid-cols-3">
        <label class="block">
          <span class="mb-1 block text-xs font-medium text-gray-600">Template name</span>
          <input v-model="draft.name" type="text" placeholder="e.g. Restaurant kitchen night clean" class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm" />
        </label>
        <label class="block">
          <span class="mb-1 block text-xs font-medium text-gray-600">Service (auto-selected for these bookings)</span>
          <select v-model="draft.serviceType" class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm">
            <option value="">Any service (general)</option>
            <option v-for="sv in services" :key="sv.id" :value="sv.id">{{ sv.name }}</option>
            <option v-if="draft.serviceType && !services.some((s) => s.id === draft!.serviceType)" :value="draft.serviceType">{{ serviceLabel(draft.serviceType) }}</option>
          </select>
        </label>
        <label class="flex items-end gap-2 pb-2 text-sm text-gray-700">
          <input v-model="draft.isActive" type="checkbox" class="h-4 w-4 accent-navy-600" /> Active
        </label>
      </div>

      <div class="overflow-x-auto rounded-lg border border-gray-200">
        <table class="w-full text-sm">
          <thead class="bg-gray-50 text-left text-xs font-medium text-gray-500">
            <tr><th class="px-3 py-2">Item</th><th class="px-3 py-2">Answer</th><th class="px-3 py-2">After photo</th><th class="px-3 py-2"></th></tr>
          </thead>
          <tbody class="divide-y divide-gray-100">
            <tr v-for="(it, i) in draft.items" :key="i">
              <td class="px-3 py-2"><input v-model="it.label" type="text" class="w-full min-w-48 rounded border border-gray-200 px-2 py-1.5" placeholder="กวาดและถูพื้น / Floors swept and mopped" /></td>
              <td class="px-3 py-2">
                <select v-model="it.itemType" class="rounded border border-gray-200 px-2 py-1.5">
                  <option v-for="(lbl, val) in typeLabels" :key="val" :value="val">{{ lbl }}</option>
                </select>
              </td>
              <td class="px-3 py-2"><label class="inline-flex items-center gap-1.5"><input v-model="it.requirePhoto" type="checkbox" class="h-4 w-4 accent-navy-600" /> Required</label></td>
              <td class="whitespace-nowrap px-3 py-2 text-right">
                <button type="button" class="px-1 text-gray-500 disabled:opacity-30" :disabled="i === 0" aria-label="Move up" @click="move(i, -1)">↑</button>
                <button type="button" class="px-1 text-gray-500 disabled:opacity-30" :disabled="i === draft.items.length - 1" aria-label="Move down" @click="move(i, 1)">↓</button>
                <button type="button" class="px-1 text-red-500" aria-label="Remove" @click="draft.items.splice(i, 1)">✕</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="flex flex-col gap-2 sm:flex-row sm:items-start">
        <button type="button" class="text-sm font-medium text-navy-600 hover:underline" @click="addItem">+ Add item</button>
        <details class="text-sm sm:ml-4">
          <summary class="cursor-pointer text-gray-500">Paste many items</summary>
          <textarea v-model="bulk" rows="4" placeholder="One item per line" class="mt-2 w-full rounded-lg border border-gray-200 px-3 py-2 text-sm sm:w-96"></textarea>
          <button type="button" class="mt-1 text-sm font-medium text-navy-600 hover:underline" @click="addBulk">Add lines</button>
        </details>
      </div>
      <p v-if="draft.id != null" class="text-xs text-gray-500">Changes apply to checklists created from now on. Checklists already sent keep their items.</p>
      <div class="flex gap-2">
        <button type="button" :disabled="saving" class="rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" @click="save">{{ saving ? "Saving…" : "Save template" }}</button>
        <button type="button" class="rounded-lg border border-gray-200 px-4 py-2 text-sm text-gray-700" @click="draft = null">Cancel</button>
      </div>
    </div>
  </div>
</template>
