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
import ChatAvatar from "./ChatAvatar.vue";
import { usePresence } from "./usePresence";

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

type RoleTab = "all" | "office" | "cleaners" | "others";
const roleTab = ref<RoleTab>("all");
const { isOnline } = usePresence();

function tabOf(role: string): Exclude<RoleTab, "all"> {
  if (OFFICE.has(role)) return "office";
  if (role === "CLEANER") return "cleaners";
  return "others";
}

const tabs = computed(() => {
  const count = (t: RoleTab) => users.value.filter((u) => t === "all" || tabOf(u.role) === t).length;
  return (
    [
      { key: "all", label: "All" },
      { key: "cleaners", label: "Cleaners" },
      { key: "office", label: "Office" },
      { key: "others", label: "Others" },
    ] as { key: RoleTab; label: string }[]
  )
    .map((t) => ({ ...t, count: count(t.key) }))
    .filter((t) => t.key === "all" || t.count > 0);
});

const visibleUsers = computed(() => {
  const q = filter.value.trim().toLowerCase();
  return users.value.filter(
    (u) =>
      (roleTab.value === "all" || tabOf(u.role) === roleTab.value) &&
      (!q || `${u.name} ${u.email} ${u.role}`.toLowerCase().includes(q)),
  );
});

const selectedUsers = computed(() => {
  const e = editing.value;
  if (!e) return [];
  return users.value.filter((u) => e.members.has(u.id));
});

function roleLabel(role: string): string {
  const r = role.toLowerCase().replace("_", " ");
  return r.charAt(0).toUpperCase() + r.slice(1);
}

function roleBadge(role: string): string {
  if (OFFICE.has(role)) return "bg-navy-50 text-navy-700";
  if (role === "CLEANER") return "bg-emerald-50 text-emerald-700";
  return "bg-gray-100 text-gray-600";
}

function onlineCount(g: ChatGroup): number {
  return g.members.filter((m) => isOnline(m.userId)).length;
}

function selectShown() {
  if (!editing.value) return;
  const s = new Set(editing.value.members);
  for (const u of visibleUsers.value) s.add(u.id);
  editing.value.members = s;
}

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
  roleTab.value = "all";
  editing.value = { id: null, name: "", members: new Set() };
}

function startEdit(g: ChatGroup) {
  filter.value = "";
  roleTab.value = "all";
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
  <div class="p-4 sm:p-5">
    <!-- Group list -->
    <template v-if="!editing">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0 max-w-xl">
          <h2 class="text-base font-semibold text-gray-900">Chat groups</h2>
          <p class="mt-0.5 text-sm text-gray-500">
            Each group gets its own group chat, and its members can also message each other
            directly. Admin, manager and dispatch can message anyone.
          </p>
        </div>
        <button
          type="button"
          class="inline-flex items-center gap-1.5 rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-navy-700"
          @click="startNew"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" d="M12 5v14M5 12h14" /></svg>
          New group
        </button>
      </div>

      <p v-if="loading" class="py-10 text-center text-sm text-gray-400">Loading…</p>
      <div v-else-if="groups.length === 0" class="mt-5 rounded-xl border border-dashed border-gray-300 px-6 py-10 text-center">
        <ChatAvatar name="" group size="lg" />
        <p class="mt-3 text-sm font-medium text-gray-900">No chat groups yet</p>
        <p class="mt-1 text-sm text-gray-500">Create one, e.g. “Bang Na team”, and add its cleaners.</p>
      </div>

      <ul v-else class="mt-5 grid grid-cols-1 gap-3 md:grid-cols-2">
        <li v-for="g in groups" :key="g.id" class="flex flex-col rounded-xl border border-gray-200 bg-white p-4 shadow-sm transition hover:shadow-md">
          <div class="flex items-start gap-3">
            <ChatAvatar :name="g.name" group size="lg" />
            <div class="min-w-0 flex-1">
              <p class="truncate font-semibold text-gray-900">{{ g.name }}</p>
              <p class="mt-0.5 text-xs text-gray-500">
                {{ g.members.length }} {{ g.members.length === 1 ? "member" : "members" }}
                <span v-if="onlineCount(g)" class="text-green-600"> · {{ onlineCount(g) }} online</span>
              </p>
            </div>
            <div class="flex shrink-0 gap-1">
              <button type="button" class="rounded-lg p-2 text-gray-500 hover:bg-gray-100 hover:text-navy-700" :aria-label="`Edit ${g.name}`" title="Edit" @click="startEdit(g)">
                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.232 5.232l3.536 3.536M9 13l6.232-6.232a2.5 2.5 0 113.536 3.536L12.536 16.536 8 18l1.464-4.536z" /></svg>
              </button>
              <button type="button" class="rounded-lg p-2 text-gray-500 hover:bg-red-50 hover:text-red-600" :aria-label="`Delete ${g.name}`" title="Delete" @click="pendingDelete = g">
                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6M9 7V4h6v3M4 7h16" /></svg>
              </button>
            </div>
          </div>
          <div v-if="g.members.length" class="mt-3 flex items-center">
            <span class="flex -space-x-2">
              <span v-for="m in g.members.slice(0, 6)" :key="m.userId" class="rounded-full ring-2 ring-white" :title="m.name">
                <ChatAvatar :name="m.name" :online="isOnline(m.userId)" size="sm" />
              </span>
            </span>
            <span v-if="g.members.length > 6" class="ml-2 text-xs font-medium text-gray-500">+{{ g.members.length - 6 }}</span>
          </div>
          <p v-else class="mt-3 text-xs text-gray-400">No members yet — tap edit to add people.</p>
        </li>
      </ul>
    </template>

    <!-- Editor -->
    <div v-else class="mx-auto flex max-w-2xl flex-col">
      <div class="flex items-center gap-2">
        <button type="button" class="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100 hover:text-gray-900" aria-label="Back to groups" @click="editing = null">
          <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" /></svg>
        </button>
        <h2 class="text-base font-semibold text-gray-900">{{ editing.id ? "Edit group" : "New group" }}</h2>
      </div>

      <div class="mt-4 flex items-center gap-3">
        <ChatAvatar :name="editing.name" group size="lg" />
        <label class="min-w-0 flex-1">
          <span class="sr-only">Group name</span>
          <input
            v-model="editing.name"
            type="text"
            maxlength="80"
            placeholder="Group name, e.g. Bang Na team"
            class="w-full rounded-xl border border-gray-200 px-4 py-2.5 text-sm font-medium focus:border-navy-500 focus:outline-none focus:ring-2 focus:ring-navy-100"
          />
        </label>
      </div>

      <section class="mt-5">
        <div class="flex items-baseline justify-between">
          <h3 class="text-sm font-semibold text-gray-900">Members <span class="font-normal text-gray-500">({{ editing.members.size }})</span></h3>
          <button v-if="editing.members.size" type="button" class="text-xs font-medium text-gray-500 hover:text-red-600" @click="editing.members = new Set()">Remove all</button>
        </div>
        <div v-if="selectedUsers.length" class="mt-2 flex max-h-32 flex-wrap gap-2 overflow-y-auto">
          <span v-for="u in selectedUsers" :key="u.id" class="inline-flex items-center gap-1.5 rounded-full border border-gray-200 bg-white py-0.5 pl-0.5 pr-1 text-sm shadow-sm">
            <ChatAvatar :name="u.name" :online="isOnline(u.id)" size="sm" />
            <span class="max-w-[9rem] truncate text-gray-800">{{ u.name }}</span>
            <button type="button" class="rounded-full p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-700" :aria-label="`Remove ${u.name}`" @click="toggle(u.id)">
              <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" d="M6 6l12 12M18 6L6 18" /></svg>
            </button>
          </span>
        </div>
        <p v-else class="mt-2 rounded-lg bg-gray-50 px-3 py-2 text-sm text-gray-500">No one added yet — pick people below.</p>
      </section>

      <section class="mt-5">
        <h3 class="text-sm font-semibold text-gray-900">Add people</h3>
        <div class="relative mt-2">
          <svg class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" d="M21 21l-4.35-4.35M17 10.5a6.5 6.5 0 11-13 0 6.5 6.5 0 0113 0z" /></svg>
          <input v-model="filter" type="search" placeholder="Search by name, email or role" class="w-full rounded-xl border border-gray-200 py-2.5 pl-9 pr-3 text-sm focus:border-navy-500 focus:outline-none focus:ring-2 focus:ring-navy-100" />
        </div>
        <div class="mt-2 flex flex-wrap items-center gap-2">
          <button
            v-for="t in tabs"
            :key="t.key"
            type="button"
            class="rounded-full px-3 py-1 text-xs font-medium transition"
            :class="roleTab === t.key ? 'bg-navy-600 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'"
            @click="roleTab = t.key"
          >{{ t.label }} <span :class="roleTab === t.key ? 'text-white/70' : 'text-gray-400'">{{ t.count }}</span></button>
          <button v-if="visibleUsers.length" type="button" class="ml-auto text-xs font-medium text-navy-700 hover:underline" @click="selectShown">Add all shown</button>
        </div>

        <ul class="mt-3 max-h-[22rem] divide-y divide-gray-100 overflow-y-auto rounded-xl border border-gray-200 bg-white">
          <li v-if="visibleUsers.length === 0" class="px-4 py-6 text-center text-sm text-gray-500">No one matches.</li>
          <li v-for="u in visibleUsers" :key="u.id">
            <button
              type="button"
              class="flex w-full items-center gap-3 px-3 py-2.5 text-left transition hover:bg-gray-50"
              :class="editing.members.has(u.id) ? 'bg-navy-50/60' : ''"
              :aria-pressed="editing.members.has(u.id)"
              @click="toggle(u.id)"
            >
              <ChatAvatar :name="u.name" :online="isOnline(u.id)" />
              <span class="min-w-0 flex-1">
                <span class="block truncate text-sm font-medium text-gray-900">{{ u.name }}</span>
                <span class="block truncate text-xs text-gray-500">
                  <span v-if="isOnline(u.id)" class="text-green-600">Online · </span>{{ u.email }}
                </span>
              </span>
              <span class="hidden shrink-0 rounded-full px-2 py-0.5 text-[11px] font-medium sm:inline" :class="roleBadge(u.role)">{{ roleLabel(u.role) }}</span>
              <span
                class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full border-2 transition"
                :class="editing.members.has(u.id) ? 'border-navy-600 bg-navy-600 text-white' : 'border-gray-300 text-transparent'"
                aria-hidden="true"
              >
                <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" /></svg>
              </span>
            </button>
          </li>
        </ul>
      </section>

      <div class="sticky bottom-0 -mx-4 mt-4 flex items-center justify-between gap-2 border-t border-gray-200 bg-white/95 px-4 py-3 backdrop-blur sm:-mx-5 sm:px-5">
        <span class="text-sm text-gray-600">{{ editing.members.size }} selected</span>
        <span class="flex gap-2">
          <button type="button" class="rounded-lg border border-gray-200 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50" @click="editing = null">Cancel</button>
          <button type="button" :disabled="saving || !editing.name.trim()" class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50" @click="save">
            {{ saving ? "Saving…" : editing.id ? "Save changes" : "Create group" }}
          </button>
        </span>
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
