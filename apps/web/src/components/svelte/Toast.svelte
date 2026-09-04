<script lang="ts">
  import type { ToastPayload, ToastType } from '../../lib/toast';

  interface Toast extends ToastPayload {
    id: number;
    type: ToastType;
  }

  let toasts = $state<Toast[]>([]);
  let nextId = 0;

  const styles: Record<ToastType, string> = {
    success: 'border-emerald-200 bg-emerald-50 text-emerald-800',
    error: 'border-red-200 bg-red-50 text-red-800',
    info: 'border-sky-200 bg-sky-50 text-sky-800',
  };

  function dismiss(id: number) {
    toasts = toasts.filter((t) => t.id !== id);
  }

  if (typeof window !== 'undefined') {
    window.addEventListener('odysight:toast', (event) => {
      const detail = (event as CustomEvent<ToastPayload>).detail;
      const type = detail.type ?? 'info';
      const id = nextId++;
      toasts.push({ ...detail, id, type });
      setTimeout(() => dismiss(id), 4000);
    });
  }
</script>

<div class="pointer-events-none fixed bottom-4 right-4 z-50 flex w-80 flex-col gap-2">
  {#each toasts as toast (toast.id)}
    <div
      class:list={[
        'pointer-events-auto rounded-lg border px-4 py-3 text-sm font-medium shadow-sm',
        styles[toast.type],
      ]}
      role="status"
    >
      <div class="flex items-start justify-between gap-2">
        <p>{toast.message}</p>
        <button
          type="button"
          onclick={() => dismiss(toast.id)}
          class="shrink-0 opacity-50 hover:opacity-100"
          aria-label="Dismiss"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>
  {/each}
</div>
