<script lang="ts">
  interface Command {
    label: string;
    href: string;
    hint?: string;
  }

  const commands: Command[] = [
    { label: 'Dashboard', href: '/dashboard' },
    { label: 'Leads', href: '/leads', hint: 'View all leads' },
    { label: 'New Lead', href: '/leads', hint: 'Go to leads to create' },
    { label: 'Customers', href: '/customers' },
    { label: 'Bookings', href: '/bookings' },
    { label: 'Cleaners', href: '/cleaners' },
    { label: 'Service Records', href: '/service-records' },
    { label: 'Payments', href: '/payments' },
    { label: 'Reports', href: '/reports' },
    { label: 'Settings', href: '/settings' },
  ];

  let open = $state(false);
  let query = $state('');
  let activeIndex = $state(0);

  const results = $derived(
    commands.filter((c) => c.label.toLowerCase().includes(query.trim().toLowerCase()))
  );

  function show() {
    open = true;
    query = '';
    activeIndex = 0;
  }

  function hide() {
    open = false;
  }

  function navigate(command: Command) {
    window.location.href = command.href;
    open = false;
  }

  function onKeydown(event: KeyboardEvent) {
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
      event.preventDefault();
      open ? hide() : show();
      return;
    }
    if (!open) return;
    if (event.key === 'Escape') {
      hide();
    } else if (event.key === 'ArrowDown') {
      event.preventDefault();
      activeIndex = Math.min(activeIndex + 1, results.length - 1);
    } else if (event.key === 'ArrowUp') {
      event.preventDefault();
      activeIndex = Math.max(activeIndex - 1, 0);
    } else if (event.key === 'Enter' && results[activeIndex]) {
      navigate(results[activeIndex]);
    }
  }

  if (typeof window !== 'undefined') {
    window.addEventListener('keydown', onKeydown);
    window.addEventListener('odysight:open-palette', show);
  }
</script>

{#if open}
  <div class="fixed inset-0 z-50 flex items-start justify-center px-4 pt-24">
    <div class="absolute inset-0 bg-slate-900/50" onclick={hide} aria-hidden="true"></div>

    <div class="relative w-full max-w-lg overflow-hidden rounded-xl bg-white shadow-2xl">
      <input
        type="text"
        placeholder="Type a command or search..."
        bind:value={query}
        oninput={() => (activeIndex = 0)}
        class="w-full border-b border-slate-200 px-4 py-3 text-sm text-slate-900 placeholder-slate-400 focus:outline-none"
      />

      <ul class="max-h-64 overflow-y-auto py-2">
        {#each results as command, index (command.label)}
          <li>
            <button
              type="button"
              class:list={[
                'flex w-full items-center justify-between px-4 py-2 text-left text-sm',
                index === activeIndex ? 'bg-indigo-50 text-indigo-700' : 'text-slate-700 hover:bg-slate-50',
              ]}
              onmouseenter={() => (activeIndex = index)}
              onclick={() => navigate(command)}
            >
              <span class="font-medium">{command.label}</span>
              {#if command.hint}
                <span class="text-xs text-slate-400">{command.hint}</span>
              {/if}
            </button>
          </li>
        {:else}
          <li class="px-4 py-6 text-center text-sm text-slate-500">No matching commands</li>
        {/each}
      </ul>

      <div class="flex items-center gap-4 border-t border-slate-200 px-4 py-2 text-xs text-slate-400">
        <span>↑↓ Navigate</span>
        <span>↵ Select</span>
        <span>Esc Close</span>
      </div>
    </div>
  </div>
{/if}
