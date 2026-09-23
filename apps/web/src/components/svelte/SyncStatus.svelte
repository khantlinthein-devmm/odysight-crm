<script lang="ts">
  import { onMount } from 'svelte';
  import {
    isOnline,
    onConnectivityChange,
    pendingCount,
    processOutbox,
  } from '../../lib/offline';

  let online = $state(true);
  let pending = $state(0);
  let syncing = $state(false);

  async function refresh() {
    try {
      pending = await pendingCount();
    } catch {
      pending = 0;
    }
  }

  async function syncNow() {
    if (syncing || !online) return;
    syncing = true;
    try {
      await processOutbox();
    } catch {
      /* errors stay queued for the next attempt */
    } finally {
      syncing = false;
      await refresh();
    }
  }

  onMount(() => {
    online = isOnline();
    void refresh();
    const off = onConnectivityChange((isUp) => {
      online = isUp;
      if (isUp) void syncNow();
      else void refresh();
    });
    const timer = setInterval(refresh, 15000);
    return () => {
      off();
      clearInterval(timer);
    };
  });
</script>

{#if !online || pending > 0}
  <button
    type="button"
    onclick={syncNow}
    title={online ? 'Sync pending changes now' : 'Offline — changes are queued on this device'}
    class="inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-xs font-semibold ring-1 transition
      {online
        ? 'bg-amber-50 text-amber-700 ring-amber-200 hover:bg-amber-100'
        : 'bg-gray-100 text-gray-500 ring-gray-200'}"
  >
    <span class="h-2 w-2 rounded-full {online ? 'bg-amber-500' : 'bg-gray-400 animate-pulse'}"></span>
    {#if !online}
      Offline
    {:else if syncing}
      Syncing…
    {:else}
      {pending} pending
    {/if}
  </button>
{/if}
