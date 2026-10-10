<script lang="ts">
  import { onDestroy } from 'svelte';
  import { getBookings } from '../../lib/bookings';
  import { getLeads } from '../../lib/leads';
  import { getPayments } from '../../lib/payments';
  import { getSessionUser } from '../../lib/auth';
  import { hasPermission } from '../../lib/roles';
  import { formatMoney } from '../../lib/settings';

  type Kind = 'lead' | 'booking' | 'payment';

  interface NotificationItem {
    id: string;
    kind: Kind;
    title: string;
    meta: string;
    createdAt: string;
    href: string;
    unread: boolean;
  }

  const STORAGE_KEY = 'odysight:notifications:lastRead';

  let open = $state(false);
  let loading = $state(true);
  let failed = $state(false);
  let items = $state<NotificationItem[]>([]);
  let lastRead = $state(0);

  const unreadCount = $derived(items.filter((n) => n.unread).length);

  function readLastRead(): number {
    try {
      return Number(localStorage.getItem(STORAGE_KEY) ?? '0') || 0;
    } catch {
      return 0;
    }
  }

  function writeLastRead(): void {
    lastRead = Date.now();
    try {
      localStorage.setItem(STORAGE_KEY, String(lastRead));
    } catch {
      /* storage unavailable */
    }
  }

  function timeAgo(iso: string): string {
    const delta = Date.now() - new Date(iso).getTime();
    if (delta < 60_000) return 'just now';
    if (delta < 3_600_000) return `${Math.floor(delta / 60_000)}m ago`;
    if (delta < 86_400_000) return `${Math.floor(delta / 3_600_000)}h ago`;
    if (delta < 604_800_000) return `${Math.floor(delta / 86_400_000)}d ago`;
    return new Date(iso).toLocaleDateString(undefined, {
      month: 'short',
      day: 'numeric',
    });
  }

  function markAllRead(): void {
    writeLastRead();
  }

  async function refresh(): Promise<void> {
    loading = true;
    failed = false;
    try {
      // Only ask for what this role may read, so e.g. a dispatcher without
      // payments.read still gets lead and booking notifications.
      const role = getSessionUser()?.role;
      const [leads, bookings, payments] = await Promise.all([
        hasPermission(role, 'leads.read') ? getLeads({ limit: 4 }) : Promise.resolve([]),
        hasPermission(role, 'bookings.read') ? getBookings({ limit: 4 }) : Promise.resolve([]),
        hasPermission(role, 'payments.read') ? getPayments({ limit: 4 }) : Promise.resolve([]),
      ]);

      const next: NotificationItem[] = [];

      for (const lead of leads) {
        next.push({
          id: `lead-${lead.id}`,
          kind: 'lead',
          title: lead.status === 'new' ? 'New lead received' : `Lead updated`,
          meta: `${lead.firstName} ${lead.lastName} · ${lead.status.replace(/_/g, ' ')}`,
          createdAt: lead.createdAt,
          href: '/leads',
          unread: new Date(lead.createdAt).getTime() > lastRead,
        });
      }

      for (const booking of bookings) {
        next.push({
          id: `booking-${booking.id}`,
          kind: 'booking',
          title: `Booking ${booking.bookingNumber}`,
          meta: `${booking.customerName} · ${booking.status.replace(/_/g, ' ')}`,
          createdAt: booking.createdAt,
          href: '/bookings',
          unread: new Date(booking.createdAt).getTime() > lastRead,
        });
      }

      for (const payment of payments) {
        next.push({
          id: `payment-${payment.id}`,
          kind: 'payment',
          title: payment.status === 'paid' ? 'Payment received' : `Payment ${payment.status}`,
          meta: `${payment.invoiceNumber} · ${formatMoney(payment.amount, payment.currency)}`,
          createdAt: payment.createdAt,
          href: '/payments',
          unread: new Date(payment.createdAt).getTime() > lastRead,
        });
      }

      next.sort(
        (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
      );
      items = next.slice(0, 10);
    } catch {
      failed = true;
    } finally {
      loading = false;
    }
  }

  function toggle(): void {
    open = !open;
    if (open) {
      writeLastRead();
      refresh();
    }
  }

  function go(href: string): void {
    writeLastRead();
    window.location.href = href;
  }

  const kindStyles: Record<Kind, string> = {
    lead: 'bg-navy-50 text-navy-600',
    booking: 'bg-green-50 text-green-600',
    payment: 'bg-gray-100 text-gray-600',
  };

  lastRead = readLastRead();
  refresh();
  const timer = setInterval(() => refresh(), 60_000);
  onDestroy(() => clearInterval(timer));
</script>

<div class="relative">
  <button
    type="button"
    onclick={toggle}
    class="relative flex h-9 w-9 items-center justify-center rounded-lg text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900"
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
        class="absolute -right-1 -top-1 flex h-5 min-w-5 items-center justify-center rounded-full bg-navy-600 px-1 text-[10px] font-semibold text-white ring-2 ring-white"
      >
        {unreadCount > 9 ? '9+' : unreadCount}
      </span>
    {/if}
  </button>

  {#if open}
    <div class="fixed inset-0 z-40" onclick={toggle} aria-hidden="true"></div>
    <div
      class="absolute right-0 z-50 mt-2 w-[22rem] origin-top-right overflow-hidden rounded-xl border border-gray-200 bg-white shadow-xl shadow-black/5"
      role="menu"
      aria-label="Notifications"
    >
      <div class="flex items-center justify-between gap-3 border-b border-gray-100 px-4 py-3">
        <div class="flex items-center gap-2">
          <p class="text-sm font-semibold text-gray-900">Notifications</p>
          {#if unreadCount > 0}
            <span
              class="flex h-5 min-w-5 items-center justify-center rounded-full bg-navy-50 px-1.5 text-[11px] font-semibold text-navy-700"
            >
              {unreadCount}
            </span>
          {/if}
        </div>
        {#if items.length > 0 && unreadCount > 0}
          <button
            type="button"
            onclick={markAllRead}
            class="shrink-0 text-xs font-medium text-navy-600 transition-colors hover:text-navy-700"
          >
            Mark all read
          </button>
        {/if}
      </div>

      {#if loading}
        <div class="space-y-4 px-4 py-4">
          {#each Array(4) as _, i (i)}
            <div class="flex animate-pulse items-start gap-3">
              <div class="h-8 w-8 shrink-0 rounded-lg bg-gray-100"></div>
              <div class="flex-1 space-y-2 pt-0.5">
                <div class="h-3.5 w-3/4 rounded bg-gray-100"></div>
                <div class="h-3 w-1/2 rounded bg-gray-100"></div>
              </div>
            </div>
          {/each}
        </div>
      {:else if failed}
        <div class="px-4 py-10 text-center">
          <p class="text-sm text-gray-500">Couldn't load notifications.</p>
          <button
            type="button"
            onclick={refresh}
            class="mt-2 text-xs font-medium text-navy-600 hover:text-navy-700"
          >
            Try again
          </button>
        </div>
      {:else if items.length === 0}
        <div class="px-4 py-10 text-center">
          <svg
            class="mx-auto h-8 w-8 text-gray-300"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="1.5"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M15 17h5l-1.4-1.4A2 2 0 0018 14.2V11a6 6 0 00-4-5.7V5a2 2 0 10-4 0v.3A6 6 0 006 11v3.2c0 .5-.2 1-.6 1.4L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
            />
          </svg>
          <p class="mt-2 text-sm font-medium text-gray-900">You're all caught up</p>
          <p class="mt-0.5 text-xs text-gray-500">
            New leads, bookings and payments will show up here.
          </p>
        </div>
      {:else}
        <ul class="max-h-[24rem] divide-y divide-gray-100 overflow-y-auto">
          {#each items as item (item.id)}
            <li>
              <button
                type="button"
                onclick={() => go(item.href)}
                class="group flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-gray-50"
              >
                <span
                  class="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg {kindStyles[item.kind]}"
                >
                  {#if item.kind === 'lead'}
                    <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        d="M18 9v3m0 0v3m0-3h3m-3 0h-3m-2-5a4 4 0 11-8 0 4 4 0 018 0zm-4 7a7 7 0 00-7 7h14a7 7 0 00-7-7z"
                      />
                    </svg>
                  {:else if item.kind === 'booking'}
                    <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        d="M8 7V3m8 4V3m-9 4h10a1 1 0 011 1v11a1 1 0 01-1 1H7a1 1 0 01-1-1V8a1 1 0 011-1zm3 5h.01M12 12h.01M15 12h.01M9 15h.01M12 15h.01M15 15h.01"
                      />
                    </svg>
                  {:else}
                    <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        d="M3 10h18M7 15h2m4 0h2M3 5h18a1 1 0 011 1v12a1 1 0 01-1 1H3a1 1 0 01-1-1V6a1 1 0 011-1z"
                      />
                    </svg>
                  {/if}
                </span>
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-sm font-medium text-gray-900">
                    {item.title}
                  </span>
                  <span class="mt-0.5 flex items-center gap-1.5 truncate text-xs text-gray-500">
                    {item.meta}
                    <span class="shrink-0">·</span>
                    <span class="shrink-0">{timeAgo(item.createdAt)}</span>
                  </span>
                </span>
                {#if item.unread}
                  <span class="mt-2 h-2 w-2 shrink-0 rounded-full bg-navy-600"></span>
                {/if}
              </button>
            </li>
          {/each}
        </ul>
        <div class="flex items-center gap-1 border-t border-gray-100 bg-gray-50/50 px-2 py-2">
          <a
            href="/leads"
            onclick={() => go('/leads')}
            class="flex-1 rounded-md px-2 py-1.5 text-center text-xs font-medium text-gray-500 transition-colors hover:bg-gray-100 hover:text-navy-600"
          >
            Leads
          </a>
          <a
            href="/bookings"
            onclick={() => go('/bookings')}
            class="flex-1 rounded-md px-2 py-1.5 text-center text-xs font-medium text-gray-500 transition-colors hover:bg-gray-100 hover:text-navy-600"
          >
            Bookings
          </a>
          <a
            href="/payments"
            onclick={() => go('/payments')}
            class="flex-1 rounded-md px-2 py-1.5 text-center text-xs font-medium text-gray-500 transition-colors hover:bg-gray-100 hover:text-navy-600"
          >
            Payments
          </a>
        </div>
      {/if}
    </div>
  {/if}
</div>