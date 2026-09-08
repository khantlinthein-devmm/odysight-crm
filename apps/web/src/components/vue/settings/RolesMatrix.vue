<script setup lang="ts">
import { PERMISSION_MATRIX, ROLE_ORDER } from "../../../lib/roles";

function allowed(roles: string[], role: string): boolean {
  return roles.includes(role);
}
</script>

<template>
  <div class="rounded-xl border border-gray-200 bg-white p-6">
    <h2 class="text-base font-semibold text-gray-900">Roles & permissions</h2>
    <p class="mt-1 text-sm text-gray-500">
      Read-only view of what each role can do. Enforced by the API on every
      request.
    </p>

    <div class="mt-4 overflow-x-auto">
      <table class="w-full min-w-[720px] text-left text-sm">
        <thead>
          <tr class="border-b border-gray-100 text-xs uppercase tracking-wide text-gray-400">
            <th class="py-2 pr-4 font-medium">Capability</th>
            <th
              v-for="role in ROLE_ORDER"
              :key="role"
              class="px-2 py-2 text-center font-medium"
            >
              {{ role }}
            </th>
          </tr>
        </thead>
        <tbody>
          <template v-for="group in PERMISSION_MATRIX" :key="group.resource">
            <tr>
              <td
                :colspan="ROLE_ORDER.length + 1"
                class="bg-gray-50 px-2 py-1.5 text-xs font-semibold uppercase tracking-wide text-gray-500"
              >
                {{ group.resource }}
              </td>
            </tr>
            <tr
              v-for="perm in group.permissions"
              :key="perm.key"
              class="border-b border-gray-50"
            >
              <td class="py-2 pr-4 text-gray-700">
                {{ perm.label }}
                <code class="ml-1 text-xs text-gray-400">{{ perm.key }}</code>
              </td>
              <td
                v-for="role in ROLE_ORDER"
                :key="role"
                class="px-2 py-2 text-center"
              >
                <span
                  v-if="allowed(perm.roles, role)"
                  class="inline-flex h-5 w-5 items-center justify-center rounded-full bg-green-100 text-xs font-bold text-green-700"
                  title="Allowed"
                  >✓</span
                >
                <span v-else class="text-gray-300" title="Not allowed">–</span>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>
  </div>
</template>
