<template>
  <div class="rounded-xl border border-red-200 bg-white p-6">
    <h2 class="text-base font-semibold text-red-900">Danger zone</h2>
    <div class="mt-2 space-y-2 text-sm text-gray-600">
      <p>
        <span class="font-medium text-gray-900">Deletes are permanent.</span>
        Removing a lead, customer, booking, payment, or team member deletes the
        row immediately — there is no archive or trash state today.
      </p>
      <p>
        Every create, update, and delete is recorded in the
        <a href="#audit" class="font-medium text-navy-600">audit log</a>
        above (who, what, when), so destructive actions stay traceable.
      </p>
      <p>
        Before bulk cleanup, take a snapshot from
        <a href="#export" class="font-medium text-navy-600">Export data</a>.
        There is no automatic data-retention purge; nothing expires on its own.
      </p>
    </div>
  </div>
</template>
