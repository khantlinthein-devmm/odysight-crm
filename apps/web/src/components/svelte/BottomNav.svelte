<script lang="ts">
  import { onMount } from 'svelte';
  import { getLang, t, type Lang, type MessageKey } from '../../lib/i18n';

  interface Item {
    label: MessageKey;
    href: string;
    icon: string;
  }

  const items: Item[] = [
    {
      label: 'nav.jobs',
      href: '/bookings',
      icon: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z',
    },
    {
      label: 'nav.checkin',
      href: '/attendance',
      icon: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z',
    },
    {
      label: 'nav.calculator',
      href: '/calculator',
      icon: 'M9 7h6m-5 4h4m5-7H5a2 2 0 00-2 2v12a2 2 0 002 2h14a2 2 0 002-2V6a2 2 0 00-2-2zm-3 14h-2v-2h2v2z',
    },
    {
      label: 'nav.home',
      href: '/dashboard',
      icon: 'M3 12l9-9 9 9M5 10v10a1 1 0 001 1h4v-6h4v6h4a1 1 0 001-1V10',
    },
  ];

  let path = $state('');
  // English on the server; the device language is applied after mount so
  // hydration matches the server-rendered markup.
  let lang = $state<Lang>('en');

  onMount(() => {
    path = window.location.pathname;
    lang = getLang();
  });

  function isActive(href: string): boolean {
    return path === href || (href !== '/' && path.startsWith(href + '/'));
  }
</script>

<nav
  aria-label="Field navigation"
  class="fixed inset-x-0 bottom-0 z-30 border-t border-gray-200 bg-white/95 pb-[env(safe-area-inset-bottom,0px)] backdrop-blur lg:hidden"
>
  <div class="grid grid-cols-4">
    {#each items as item}
      <a
        href={item.href}
        aria-current={isActive(item.href) ? 'page' : undefined}
        class="field-tap flex flex-col items-center justify-center gap-0.5 py-2 text-[11px] font-medium {isActive(item.href) ? 'text-navy-700' : 'text-gray-500'}"
      >
        <svg
          class="h-6 w-6"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width={isActive(item.href) ? 2.25 : 2}
        >
          <path stroke-linecap="round" stroke-linejoin="round" d={item.icon} />
        </svg>
        {t(item.label, lang)}
      </a>
    {/each}
  </div>
</nav>
