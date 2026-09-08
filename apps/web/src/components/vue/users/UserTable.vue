<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ApiError } from "../../../lib/api";
import { clearSession, getSessionUser } from "../../../lib/auth";
import {
  createUser,
  deleteUser,
  getUsers,
  resetUserPassword,
  updateUser,
  TEAM_ROLES,
  type TeamUser,
} from "../../../lib/users";
import { showToast } from "../../../lib/toast";
import ConfirmDialog from "../ui/ConfirmDialog.vue";

const users = ref<TeamUser[]>([]);
const loading = ref(true);
const search = ref("");
const forbidden = ref(false);
const authExpired = ref(false);

const showForm = ref(false);
const editing = ref<TeamUser | null>(null);
const formName = ref("");
const formEmail = ref("");
const formPassword = ref("");
const formRole = ref<TeamUser["role"]>("DISPATCH");
const saving = ref(false);
const formError = ref<string | null>(null);

const pendingDelete = ref<TeamUser | null>(null);
const deleting = ref(false);

const resetTarget = ref<TeamUser | null>(null);
const resetPassword = ref("");
const resetting = ref(false);
const resetError = ref<string | null>(null);

const currentUserId = computed(() => getSessionUser()?.id ?? null);

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase();
  if (!q) return users.value;
  return users.value.filter((u) =>
    `${u.name} ${u.email} ${u.role}`.toLowerCase().includes(q),
  );
});

function errMessage(e: unknown, fallback: string): string {
  if (e instanceof ApiError) return `API error ${e.status}: ${e.message}`;
  return e instanceof Error ? e.message : fallback;
}

async function fetchUsers() {
  loading.value = true;
  forbidden.value = false;
  authExpired.value = false;
  try {
    users.value = await getUsers({ limit: 100 });
  } catch (e) {
    if (e instanceof ApiError && e.status === 403) {
      forbidden.value = true;
    } else if (e instanceof ApiError && e.status === 401) {
      clearSession();
      authExpired.value = true;
    } else {
      showToast(errMessage(e, "Failed to load team"), "error");
    }
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  editing.value = null;
  formName.value = "";
  formEmail.value = "";
  formPassword.value = "";
  formRole.value = "DISPATCH";
  formError.value = null;
  showForm.value = true;
}

function openEdit(user: TeamUser) {
  editing.value = user;
  formName.value = user.name;
  formEmail.value = user.email;
  formRole.value = user.role;
  formPassword.value = "";
  formError.value = null;
  showForm.value = true;
}

async function handleSave() {
  formError.value = null;
  if (!formName.value.trim()) {
    formError.value = "Name is required.";
    return;
  }
  if (!editing.value && formPassword.value.length < 8) {
    formError.value = "Password must be at least 8 characters.";
    return;
  }
  saving.value = true;
  try {
    if (editing.value) {
      const updated = await updateUser(editing.value.id, {
        name: formName.value.trim(),
        role: formRole.value,
      });
      users.value = users.value.map((u) =>
        u.id === updated.id ? updated : u,
      );
      showToast("Team member updated", "success");
    } else {
      const created = await createUser({
        name: formName.value.trim(),
        email: formEmail.value.trim(),
        password: formPassword.value,
        role: formRole.value,
      });
      users.value = [created, ...users.value];
      showToast(`Account created for ${created.email}`, "success");
    }
    showForm.value = false;
  } catch (e) {
    formError.value = errMessage(e, "Failed to save user");
  } finally {
    saving.value = false;
  }
}

async function handleDelete() {
  if (!pendingDelete.value) return;
  deleting.value = true;
  try {
    await deleteUser(pendingDelete.value.id);
    users.value = users.value.filter((u) => u.id !== pendingDelete.value!.id);
    showToast("Team member removed", "success");
  } catch (e) {
    showToast(errMessage(e, "Failed to delete user"), "error");
  } finally {
    deleting.value = false;
    pendingDelete.value = null;
  }
}

function openReset(user: TeamUser) {
  resetTarget.value = user;
  resetPassword.value = "";
  resetError.value = null;
}

async function handleReset() {
  if (!resetTarget.value) return;
  resetError.value = null;
  if (resetPassword.value.length < 8) {
    resetError.value = "New password must be at least 8 characters.";
    return;
  }
  resetting.value = true;
  try {
    await resetUserPassword(resetTarget.value.id, resetPassword.value);
    showToast(`Password reset for ${resetTarget.value.email}`, "success");
    resetTarget.value = null;
  } catch (e) {
    resetError.value = errMessage(e, "Failed to reset password");
  } finally {
    resetting.value = false;
  }
}

function signInAgain() {
  clearSession();
  window.location.href = "/login?next=/settings";
}

onMounted(fetchUsers);
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-white">
    <div
      class="flex flex-col gap-3 border-b border-gray-200 px-6 py-4 sm:flex-row sm:items-center"
    >
      <h2 class="text-base font-semibold text-gray-900">Team members</h2>
      <span
        class="rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-600"
      >
        {{ users.length }}
      </span>

      <div class="sm:ml-auto flex flex-col gap-2 sm:flex-row">
        <input
          v-model="search"
          type="search"
          placeholder="Search name, email, role..."
          class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500 sm:w-56"
        />
        <button
          type="button"
          class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700"
          @click="openCreate"
        >
          New member
        </button>
      </div>
    </div>

    <div v-if="loading" class="px-6 py-16 text-center">
      <p class="text-sm text-gray-500">Loading team...</p>
    </div>

    <div
      v-else-if="forbidden"
      class="px-6 py-16 text-center"
    >
      <p class="text-sm font-medium text-gray-900">Restricted to admins</p>
      <p class="mx-auto mt-1 max-w-md text-sm text-gray-500">
        Only ADMIN and SUPER_ADMIN roles can manage team accounts. Ask an
        administrator to create the account or change your role.
      </p>
    </div>

    <div
      v-else-if="authExpired"
      class="px-6 py-16 text-center"
    >
      <p class="text-sm font-medium text-gray-900">Session expired</p>
      <p class="mt-1 text-sm text-gray-500">
        Please sign in again to manage the team.
      </p>
      <button
        type="button"
        class="mt-4 rounded-lg bg-navy-600 px-4 py-2 text-sm font-semibold text-white hover:bg-navy-700"
        @click="signInAgain"
      >
        Sign in again
      </button>
    </div>

    <div v-else-if="filtered.length === 0" class="px-6 py-16 text-center">
      <p class="text-sm font-medium text-gray-900">No team members found</p>
      <p class="mt-1 text-sm text-gray-500">
        Invite your first team member with the New member button.
      </p>
    </div>

    <div v-else class="overflow-x-auto">
      <table class="w-full min-w-[640px] text-left text-sm">
        <thead>
          <tr class="border-b border-gray-100 text-xs uppercase tracking-wide text-gray-400">
            <th class="px-6 py-3 font-medium">Name</th>
            <th class="px-6 py-3 font-medium">Email</th>
            <th class="px-6 py-3 font-medium">Role</th>
            <th class="px-6 py-3 text-right font-medium">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-navy-100">
          <tr v-for="u in filtered" :key="u.id" class="hover:bg-gray-100">
            <td class="px-6 py-3 font-medium text-gray-900">
              {{ u.name }}
              <span
                v-if="u.id === currentUserId"
                class="ml-2 rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-700"
                >you</span
              >
            </td>
            <td class="px-6 py-3 text-gray-600">{{ u.email }}</td>
            <td class="px-6 py-3">
              <span
                class="rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-600"
                >{{ u.role }}</span
              >
            </td>
            <td class="px-6 py-3">
              <div class="flex justify-end gap-2">
                <button
                  type="button"
                  class="rounded-lg border border-gray-200 px-3 py-1.5 text-xs font-medium text-gray-700 hover:bg-gray-100"
                  @click="openEdit(u)"
                >
                  Edit
                </button>
                <button
                  type="button"
                  class="rounded-lg border border-gray-200 px-3 py-1.5 text-xs font-medium text-gray-700 hover:bg-gray-100"
                  @click="openReset(u)"
                >
                  Reset password
                </button>
                <button
                  type="button"
                  :disabled="u.id === currentUserId"
                  title="You cannot delete your own account"
                  class="rounded-lg border border-red-200 px-3 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50 disabled:opacity-40"
                  @click="pendingDelete = u"
                >
                  Remove
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create / edit form -->
    <div
      v-if="showForm"
      class="border-t border-gray-200 bg-gray-50 px-6 py-5"
    >
      <h3 class="text-sm font-semibold text-gray-900">
        {{ editing ? `Edit ${editing.email}` : "Invite team member" }}
      </h3>
      <div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
        <label class="block">
          <span class="mb-1 block text-xs font-medium text-gray-600">Full name</span>
          <input
            v-model="formName"
            type="text"
            placeholder="Jane Doe"
            class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
          />
        </label>
        <label v-if="!editing" class="block">
          <span class="mb-1 block text-xs font-medium text-gray-600">Email</span>
          <input
            v-model="formEmail"
            type="email"
            placeholder="teammate@company.com"
            class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
          />
        </label>
        <label v-if="!editing" class="block">
          <span class="mb-1 block text-xs font-medium text-gray-600">Temporary password (min 8 chars)</span>
          <input
            v-model="formPassword"
            type="password"
            autocomplete="new-password"
            placeholder="••••••••"
            class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
          />
        </label>
        <label class="block">
          <span class="mb-1 block text-xs font-medium text-gray-600">Role</span>
          <select
            v-model="formRole"
            class="w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
          >
            <option v-for="r in TEAM_ROLES" :key="r" :value="r">{{ r }}</option>
          </select>
        </label>
      </div>
      <p v-if="formError" class="mt-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
        {{ formError }}
      </p>
      <div class="mt-4 flex justify-end gap-2">
        <button
          type="button"
          class="rounded-lg border border-gray-200 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100"
          @click="showForm = false"
        >
          Cancel
        </button>
        <button
          type="button"
          :disabled="saving"
          class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50"
          @click="handleSave"
        >
          {{ saving ? "Saving..." : editing ? "Save changes" : "Create account" }}
        </button>
      </div>
      <p v-if="!editing" class="mt-3 text-xs text-gray-500">
        The new member signs in at <code>/login</code> with this email +
        temporary password, then changes it via <code>/settings</code> or your
        admin resets it here.
      </p>
    </div>

    <!-- Reset password panel -->
    <div
      v-if="resetTarget"
      class="border-t border-gray-200 bg-gray-50 px-6 py-5"
    >
      <h3 class="text-sm font-semibold text-gray-900">
        Reset password for {{ resetTarget.email }}
      </h3>
      <div class="mt-3 flex max-w-md gap-2">
        <input
          v-model="resetPassword"
          type="password"
          autocomplete="new-password"
          placeholder="New temporary password (min 8 chars)"
          class="flex-1 rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-navy-500"
        />
        <button
          type="button"
          :disabled="resetting"
          class="rounded-lg bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50"
          @click="handleReset"
        >
          {{ resetting ? "Saving..." : "Set password" }}
        </button>
        <button
          type="button"
          class="rounded-lg border border-gray-200 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100"
          @click="resetTarget = null"
        >
          Cancel
        </button>
      </div>
      <p v-if="resetError" class="mt-3 max-w-md rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
        {{ resetError }}
      </p>
    </div>

    <ConfirmDialog
      v-if="pendingDelete"
      title="Remove team member?"
      :message="`This will permanently delete ${pendingDelete.email}. They will no longer be able to sign in.`"
      confirm-label="Remove member"
      :busy="deleting"
      @confirm="handleDelete"
      @cancel="pendingDelete = null"
    />
  </div>
</template>
