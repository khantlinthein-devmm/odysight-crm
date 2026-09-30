<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  deleteChatGroup,
  getChatGroups,
  saveChatGroup,
  type ChatGroup,
} from "../../../lib/chat";
import { apiFetch, unwrapPage, type Page } from "../../../lib/api";
import { showToast } from "../../../lib/toast";
import ConfirmDialog from "../ui/ConfirmDialog.vue";

// Admin screen: each group has a group chat, and people in the same group
// may also message each other directly. Office
// roles (admin, manager, dispatch) can message anyone without a group.

interface UserRow {
  id: number;
  name: string;
  email: string;
  role: string;
}

const groups = ref<ChatGroup[]>([]);
const users = ref<UserRow[]>([]);
const loading = ref(true);
const editing = ref<{ id: number | null; name: string; members: Set<number> } | null>(null);
const saving = ref(false);
const filter = ref("");
const pendingDelete = ref<ChatGroup | null>(null);

const OFFICE = new Set(["SUPER_ADMIN", "ADMIN", "MANAGER", "DISPATCH"]);

const visibleUsers = computed(() => {
  const q = filter.value.trim().toLowerCase();
  return users.value.filter((u) => !q || `${u.name} ${u.email} ${u.role}`.toLowerCase().includes(q));
});

async function load() {
  loading.value = true;
  try {
    const [g, u] = await Promise.all([
      getChatGroups(),
      apiFetch<UserRow[] | Page<UserRow>>("/api/v1/users?limit=200"),
    ]);
    groups.value = g;
    users.value = unwrapPage(u).sort((a, b) => a.name.localeCompare(b.name));
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to load chat groups", "error");
  } finally {
    loading.value = false;
  }
}

function startNew() {
  filter.value = "";
  editing.value = { id: null, name: "", members: new Set() };
}

function startEdit(g: ChatGroup) {
  filter.value = "";
  editing.value = { id: g.id, name: g.name, members: new Set(g.members.map((m) => m.userId)) };
}

function toggle(id: number) {
  if (!editing.value) return;
  const s = new Set(editing.value.members);
  if (s.has(id)) s.delete(id);
  else s.add(id);
  editing.value.members = s;
}

async function save() {
  const e = editing.value;
  if (!e || !e.name.trim()) {
    showToast("Enter a group name", "error");
    return;
  }
  saving.value = true;
  try {
    await saveChatGroup(e.id, { name: e.name.trim(), memberIds: [...e.members] });
    showToast("Chat group saved", "success");
    editing.value = null;
    await load();
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to save group", "error");
  } finally {
    saving.value = false;
  }
}

async function confirmDelete() {
  const g = pendingDelete.value;
  if (!g) return;
  try {
    await deleteChatGroup(g.id);
    showToast(`Deleted ${g.name}`, "success");
    await load();
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to delete group", "error");
  } finally {
    pendingDelete.value = null;
  }
}

onMounted(load);
</script>

<template>
  <div class="space-y-4 p-4">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <p class="text-sm text-gray-500">
        Each group has its own group chat for all its members, and people in the
        same group can also message each other directly. Admin, manager and
        dispatch can message anyone.
      </p>
      <button
        type="button"
        class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700"
        @click="startNew"
      >New group</button>
    </div>

    <p v-if="loading" class="py-6 text-center text-sm text-gray-400">Loading…</p>
    <p v-else-if="groups.length === 0 && !editing" class="py-6 text-center text-sm text-gray-500">
      No chat groups yet. Create one, e.g. “Bang Na team”, and add its cleaners.
    </p>

    <ul v-if="!editing" class="divide-y divide-gray-100 rounded-lg border border-gray-200">
      <li v-for="g in groups" :key="g.id" class="flex flex-wrap items-center justify-between gap-2 px-4 py-3">
        <div class="min-w-0">
          <p class="font-medium text-gray-900">{{ g.name }}</p>
          <p class="truncate text-xs text-gray-500">
            {{ g.members.length }} members<span v-if="g.members.length">: {{ g.members.map((m) => m.name).join(", ") }}</span>
          </p>
        </div>
        <div class="flex gap-3 text-sm">
          <button type="button" class="text-navy-700 hover:underline" @click="startEdit(g)">Edit</button>
          <button type="button" class="text-red-600 hover:underline" @click="pendingDelete = g">Delete</button>
        </div>
      </li>
    </ul>

    <div v-if="editing" class="space-y-3 rounded-lg border border-gray-200 p-4">
      <label class="block">
        <span class="mb-1 block text-xs font-medium text-gray-600">Group name</span>
        <input v-model="editing.name" type="text" placeholder="Bang Na team" class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm" />
      </label>
      <div>
        <div class="mb-1 flex items-center justify-between gap-2">
          <span class="text-xs font-medium text-gray-600">Members ({{ editing.members.size }})</span>
          <input v-model="filter" type="search" placeholder="Filter…" class="w-40 rounded-lg border border-gray-200 px-2 py-1 text-xs" />
        </div>
        <ul class="max-h-72 divide-y divide-gray-100 overflow-y-auto rounded-lg border border-gray-200">
          <li v-for="u in visibleUsers" :key="u.id">
            <label class="flex cursor-pointer items-center gap-3 px-3 py-2 text-sm hover:bg-gray-50">
              <input type="checkbox" :checked="editing.members.has(u.id)" class="h-4 w-4" @change="toggle(u.id)" />
              <span class="flex-1 text-gray-900">{{ u.name }}</span>
              <span class="text-xs" :class="OFFICE.has(u.role) ? 'text-navy-600' : 'text-gray-500'">{{ u.role }}</span>
            </label>
          </li>
        </ul>
      </div>
      <div class="flex justify-end gap-2">
        <button type="button" class="rounded-lg border border-gray-200 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50" @click="editing = null">Cancel</button>
        <button type="button" :disabled="saving" class="rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50" @click="save">
          {{ saving ? "Saving…" : "Save group" }}
        </button>
      </div>
    </div>

    <ConfirmDialog
      v-if="pendingDelete"
      title="Delete chat group"
      :message="`Delete ${pendingDelete.name}? Its group chat and messages are deleted too. Members who share no other group will no longer be able to message each other directly.`"
      confirm-label="Delete group"
      @confirm="confirmDelete"
      @cancel="pendingDelete = null"
    />
  </div>
</template>
