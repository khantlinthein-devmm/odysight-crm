<script lang="ts">
  interface Notification {
    id: number;
    title: string;
    time: string;
    unread: boolean;
  }

  let open = $state(false);
  let notifications = $state<Notification[]>([
    { id: 1, title: 'New lead assigned: John Doe', time: '5m ago', unread: true },
    { id: 2, title: 'Booking BK-2026-0002 moved to in progress', time: '1h ago', unread: true },
    { id: 3, title: 'Payment received from Jane Smith', time: '3h ago', unread: false },
  ]);

  const unreadCount = $derived(notifications.filter((n) => n.unread).length);

  function toggle() {
    open = !open;
  }

  function markAllRead() {
    notifications = notifications.map((n) => ({ ...n, unread: false }));
  }
</script>

<div class="relative">
  <button
    type="button"
    onclick={toggle}
    class="relative rounded-lg p-2 text-slate-500 hover:bg-slate-100"
    aria-label="Notifications"
  >
    <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
      <path
        stroke-linecap="round"
        stroke-linejoin="round"
        d="M15 17h5l-1.4-1.4A2 2 0 0118 14.2V11a6 6 0 00-4-5.7V5a2 2 0 10-4 0v.3A6 6 0 006 11v3.2c0 .5-.2 1-.6 1.4L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
      />
    </svg>
    {#if unreadCount > 0}
      <span
        class="absolute -right-0.5 -top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-semibold text-white"
      >
        {unreadCount}
      </span>
    {/if}
  </button>

  {#if open}
    <div class="fixed inset-0 z-40" onclick={toggle} aria-hidden="true"></div>
    <div class="absolute right-0 z-50 mt-2 w-80 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-lg">
      <div class="flex items-center justify-between border-b border-slate-200 px-4 py-3">
        <p class="text-sm font-semibold text-slate-900">Notifications</p>
        <button type="button" onclick={markAllRead} class="text-xs font-medium text-indigo-600 hover:text-indigo-700">
          Mark all read
        </button>
      </div>
      <ul class="max-h-64 divide-y divide-slate-100 overflow-y-auto">
        {#each notifications as notification (notification.id)}
          <li class:list={['px-4 py-3 text-sm', notification.unread ? 'bg-indigo-50/50' : '']}>
            <p class="font-medium text-slate-900">{notification.title}</p>
            <p class="mt-0.5 text-xs text-slate-500">{notification.time}</p>
          </li>
        {:else}
          <li class="px-4 py-6 text-center text-sm text-slate-500">No notifications</li>
        {/each}
      </ul>
    </div>
  {/if}
</div>
