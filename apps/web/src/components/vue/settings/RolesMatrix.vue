<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { PERMISSION_MATRIX } from "../../../lib/roles";
import { hasPermission } from "../../../lib/roles";
import { getSessionUser } from "../../../lib/auth";
import { apiFetch } from "../../../lib/api";
import { showToast } from "../../../lib/toast";
import ConfirmDialog from "../ui/ConfirmDialog.vue";

// Edit what each role may do. Super admin always keeps everything (so the
// owner can never be locked out); the API enforces the result on every
// request.

interface RoleView {
  role: string;
  editable: boolean;
  customized: boolean;
  permissions: string[];
  defaults: string[];
}
interface RolesResponse {
  allPermissions: string[];
  roles: RoleView[];
}

const canEdit = hasPermission(getSessionUser()?.role, "users.manage");

const data = ref<RolesResponse | null>(null);
const draft = ref<Record<string, Set<string>>>({});
const loading = ref(true);
const saving = ref(false);
const pendingReset = ref<string | null>(null);

// Sensitive permissions get a warning marker.
const SENSITIVE = new Set(["users.manage", "settings.manage", "audit.read", "payments.update", "invoices.update"]);

const groups = computed(() => {
  const all = data.value?.allPermissions ?? [];
  const labelled = new Set<string>();
  const out = PERMISSION_MATRIX.map((g) => ({
    resource: g.resource,
    permissions: g.permissions
      .filter((p) => all.includes(p.key))
      .map((p) => {
        labelled.add(p.key);
        return { key: p.key, label: p.label };
      }),
  })).filter((g) => g.permissions.length > 0);
  const rest = all.filter((k) => !labelled.has(k));
  if (rest.length) out.push({ resource: "Other", permissions: rest.map((k) => ({ key: k, label: k })) });
  return out;
});

const changedRoles = computed(() =>
  (data.value?.roles ?? []).filter((r) => r.editable && !sameSet(draft.value[r.role], r.permissions)),
);

function sameSet(a: Set<string> | undefined, b: string[]): boolean {
  if (!a) return true;
  return a.size === b.length && b.every((x) => a.has(x));
}

function apply(res: RolesResponse) {
  data.value = res;
  const d: Record<string, Set<string>> = {};
  for (const r of res.roles) d[r.role] = new Set(r.permissions);
  draft.value = d;
}

async function load() {
  loading.value = true;
  try {
    apply(await apiFetch<RolesResponse>("/api/v1/roles"));
  } catch (e) {
    showToast(e instanceof Error ? e.message : "Failed to load roles", "error");
  } finally {
    loading.value = false;
  }
}

function has(role: string, key: string): boolean {
  return draft.value[role]?.has(key) ?? false;
}

function isDefault(role: RoleView, key: string): boolean {
  return role.defaults.includes(key);
}

function toggle(role: string, key: string) {
  const s = new Set(draft.value[role]);
  if (s.has(key)) s.delete(key);
  else s.add(key);
  // Editing or deleting without being able to see makes no sense: keep
  // ".read" on when anything else in the same area is granted.
  const area = key.split(".")[0];
  if (s.has(key) && !key.endsWith(".read") && data.value?.allPermissions.includes(`${area}.read`)) {
    s.add(`${area}.read`);
  }
  draft.value = { ...draft.value, [role]: s };
}

async function save() {
  saving.value = true;
  try {
    let res: RolesResponse | null = null;
    for (const r of changedRoles.value) {
      res = await apiFetch<RolesResponse>(`/api/v1/roles/${r.role}`, {
        method: "PATCH",
        body: JSON.stringify({ permissions: [...draft.value[r.role]!] }),
      });
    }
    if (res) apply(res);
    showToast("Permissions saved — they apply on each person's next page load", "success");
  } catch (e) {
    showToast(e instanceof Error ? e.message : "Failed to save permissions", "error");
  } finally {
    saving.value = false;
  }
}

function discard() {
  if (data.value) apply(data.value);
}

async function confirmReset() {
  const role = pendingReset.value;
  if (!role) return;
  try {
    apply(await apiFetch<RolesResponse>(`/api/v1/roles/${role}`, { method: "DELETE" }));
    showToast(`${role} is back to the default permissions`, "success");
  } catch (e) {
    showToast(e instanceof Error ? e.message : "Failed to reset", "error");
  } finally {
    pendingReset.value = null;
  }
}

onMounted(load);
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-white p-6">
    <h2 class="text-base font-semibold text-gray-900">Roles & permissions</h2>
    <p class="mt-1 text-sm text-gray-500">
      <template v-if="canEdit">
        Tick what each role may do and save. Super admin always has everything. Changes are enforced
        by the API straight away; menus update on each person's next page load.
      </template>
      <template v-else>What each role can do. Only a Super admin can change this.</template>
    </p>

    <p v-if="loading" class="mt-4 text-sm text-gray-400">Loading…</p>

    <template v-else-if="data">
      <div
        v-if="canEdit && changedRoles.length"
        class="sticky top-16 z-10 mt-4 flex flex-wrap items-center justify-between gap-2 rounded-lg border border-amber-200 bg-amber-50 px-4 py-2 text-sm text-amber-900"
      >
        <span>Unsaved changes for {{ changedRoles.map((r) => r.role).join(", ") }}</span>
        <span class="flex gap-2">
          <button type="button" class="rounded-lg border border-amber-300 bg-white px-3 py-1.5 text-sm" @click="discard">Discard</button>
          <button
            type="button"
            :disabled="saving"
            class="rounded-lg bg-navy-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50"
            @click="save"
          >{{ saving ? "Saving…" : "Save changes" }}</button>
        </span>
      </div>

      <div class="mt-4 overflow-x-auto">
        <table class="w-full min-w-[760px] text-left text-sm">
          <thead>
            <tr class="border-b border-gray-100 text-xs uppercase tracking-wide text-gray-400">
              <th class="py-2 pr-4 font-medium">Capability</th>
              <th v-for="r in data.roles" :key="r.role" class="px-2 py-2 text-center font-medium">
                <div>{{ r.role.replace("_", " ") }}</div>
                <div v-if="!r.editable" class="mt-0.5 text-[10px] normal-case text-gray-400">🔒 all</div>
                <div v-else-if="r.customized" class="mt-0.5 flex flex-col items-center gap-0.5 normal-case">
                  <span class="rounded-full bg-navy-50 px-1.5 text-[10px] font-medium text-navy-700">edited</span>
                  <button v-if="canEdit" type="button" class="text-[10px] text-gray-500 underline hover:text-gray-800" @click="pendingReset = r.role">
                    reset
                  </button>
                </div>
              </th>
            </tr>
          </thead>
          <tbody>
            <template v-for="group in groups" :key="group.resource">
              <tr>
                <td :colspan="data.roles.length + 1" class="bg-gray-50 px-2 py-1.5 text-xs font-semibold uppercase tracking-wide text-gray-500">
                  {{ group.resource }}
                </td>
              </tr>
              <tr v-for="perm in group.permissions" :key="perm.key" class="border-b border-gray-50">
                <td class="py-2 pr-4 text-gray-700">
                  {{ perm.label }}
                  <span v-if="SENSITIVE.has(perm.key)" title="Sensitive: grant with care" class="ml-1 text-amber-600">⚠</span>
                  <code class="ml-1 text-xs text-gray-400">{{ perm.key }}</code>
                </td>
                <td v-for="r in data.roles" :key="r.role" class="px-2 py-2 text-center">
                  <span
                    v-if="!r.editable"
                    class="inline-flex h-5 w-5 items-center justify-center rounded-full bg-green-100 text-xs font-bold text-green-700"
                    title="Always allowed"
                  >✓</span>
                  <input
                    v-else-if="canEdit"
                    type="checkbox"
                    class="h-4 w-4 cursor-pointer rounded border-gray-300 accent-navy-600"
                    :class="has(r.role, perm.key) !== isDefault(r, perm.key) ? 'ring-2 ring-amber-400 ring-offset-1' : ''"
                    :checked="has(r.role, perm.key)"
                    :title="has(r.role, perm.key) !== isDefault(r, perm.key) ? 'Differs from the default' : ''"
                    :aria-label="`${r.role} ${perm.key}`"
                    @change="toggle(r.role, perm.key)"
                  />
                  <span
                    v-else-if="has(r.role, perm.key)"
                    class="inline-flex h-5 w-5 items-center justify-center rounded-full bg-green-100 text-xs font-bold text-green-700"
                  >✓</span>
                  <span v-else class="text-gray-300">–</span>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>
      <p class="mt-3 text-xs text-gray-500">
        Outlined boxes differ from the built-in default. Cleaners always see and change only their
        own jobs and check-ins, whatever is ticked here.
      </p>
    </template>

    <ConfirmDialog
      v-if="pendingReset"
      title="Reset to default"
      :message="`Put ${pendingReset} back to the built-in permissions? Your changes for this role are removed.`"
      confirm-label="Reset"
      @confirm="confirmReset"
      @cancel="pendingReset = null"
    />
  </div>
</template>
