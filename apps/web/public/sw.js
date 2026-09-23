/* Odysight CRM service worker — Phase 1 offline (field + calculator).
 *
 * Strategy:
 * - Static app shell (/_astro/*, /logo/*, favicon, manifest): cache-first.
 * - Offline allowlist navigations (/dashboard, /calculator, /bookings,
 *   /attendance, /checklists): network-first, fall back to cache, then
 *   /offline.html. Successful navigations are cached for later offline use.
 * - Everything else (incl. /api/*): network-only. API reads fall back to
 *   on-device snapshots managed by the app (see src/lib/offline.ts).
 */

const VERSION = "odysight-v1";
const STATIC_CACHE = `${VERSION}-static`;
const PAGE_CACHE = `${VERSION}-pages`;

const OFFLINE_PAGES = new Set([
  "/dashboard",
  "/calculator",
  "/bookings",
  "/attendance",
  "/checklists",
]);

function isStaticAsset(url) {
  return (
    url.pathname.startsWith("/_astro/") ||
    url.pathname.startsWith("/logo/") ||
    url.pathname.startsWith("/assets/") ||
    url.pathname === "/favicon.svg" ||
    url.pathname === "/favicon.ico" ||
    url.pathname === "/manifest.webmanifest" ||
    url.pathname === "/offline.html"
  );
}

function pageKey(pathname) {
  if (OFFLINE_PAGES.has(pathname)) return pathname;
  if (pathname.length > 1 && pathname.endsWith("/")) {
    const trimmed = pathname.slice(0, -1);
    if (OFFLINE_PAGES.has(trimmed)) return trimmed;
  }
  return null;
}

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches
      .open(STATIC_CACHE)
      .then((cache) => cache.addAll(["/offline.html", "/manifest.webmanifest"]))
      .then(() => self.skipWaiting()),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) =>
        Promise.all(
          keys
            .filter((k) => k.startsWith("odysight-") && k !== STATIC_CACHE && k !== PAGE_CACHE)
            .map((k) => caches.delete(k)),
        ),
      )
      .then(() => self.clients.claim()),
  );
});

async function cacheFirst(request, cacheName) {
  const cache = await caches.open(cacheName);
  const hit = await cache.match(request);
  if (hit) return hit;
  const res = await fetch(request);
  if (res.ok) cache.put(request, res.clone());
  return res;
}

async function navigationFallback(request, key) {
  const cache = await caches.open(PAGE_CACHE);
  try {
    const res = await fetch(request);
    if (res.ok) cache.put(key, res.clone());
    return res;
  } catch {
    const hit = await cache.match(key);
    if (hit) return hit;
    const fallback = await cache.match("/offline.html");
    if (fallback) return fallback;
    return new Response("Offline", { status: 503 });
  }
}

self.addEventListener("fetch", (event) => {
  const { request } = event;
  if (request.method !== "GET") return;
  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;

  if (isStaticAsset(url)) {
    event.respondWith(cacheFirst(request, STATIC_CACHE));
    return;
  }

  if (request.mode === "navigate") {
    const key = pageKey(url.pathname);
    if (key) event.respondWith(navigationFallback(request, key));
  }
});
