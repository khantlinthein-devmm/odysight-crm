const builtApiUrl: string | undefined = import.meta.env.PUBLIC_API_URL;

function isLoopbackHost(hostname: string): boolean {
  return hostname === "localhost" || /^\d+\.\d+\.\d+\.\d+$/.test(hostname);
}

// Resolves the API base for the page the user is on. The session cookie set by
// the API is host-scoped, so in local development the API must be reached on
// the same host as the UI (localhost vs 127.0.0.1 vs a LAN IP) or the cookie
// never travels back to the SSR middleware. Non-loopback origins (production)
// are returned untouched.
export function getApiBaseUrl(): string | undefined {
  const built = builtApiUrl;
  if (!built || typeof window === "undefined") return built;
  try {
    const builtUrl = new URL(built);
    if (!isLoopbackHost(builtUrl.hostname)) return built;
    const hostname = window.location.hostname;
    if (builtUrl.hostname === hostname || !isLoopbackHost(hostname)) return built;
    const api = new URL(built);
    api.hostname = hostname;
    return api.origin + (api.pathname === "/" ? "" : api.pathname);
  } catch {
    return built;
  }
}

const API_URL = getApiBaseUrl();

// Mocks are dev-only and explicit opt-in. Never enabled in production builds,
// so mock fixtures and demo credentials are dead code in prod and API calls fail closed.
export const USE_MOCKS =
  import.meta.env.DEV === true &&
  (import.meta.env.PUBLIC_USE_MOCKS === "true" ||
    (!API_URL && import.meta.env.PUBLIC_USE_MOCKS !== "false"));

if (!API_URL && import.meta.env.PROD) {
  throw new Error(
    "PUBLIC_API_URL is not set. The application requires a configured API in production.",
  );
}

export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

export interface Page<T> {
  data: T[];
  total: number;
  limit: number;
  offset: number;
}

export function unwrapPage<T>(json: T[] | Page<T>): T[] {
  if (Array.isArray(json)) return json;
  if (json && Array.isArray((json as Page<T>).data)) return (json as Page<T>).data;
  return [];
}

export function toQuery(params: Record<string, string | number | undefined>): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== "") q.set(k, String(v));
  }
  const s = q.toString();
  return s ? `?${s}` : "";
}

export async function apiFetch<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  if (!API_URL) {
    throw new ApiError(0, "PUBLIC_API_URL is not configured");
  }

  // Auth is via the HttpOnly odysight_session cookie set by the API on login.
  const response = await fetch(`${API_URL}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...init?.headers,
    },
  });

  if (!response.ok) {
    let message = `Request failed: ${response.statusText || response.status}`;
    try {
      const body = await response.clone().json() as { error?: string; message?: string };
      if (body?.error) message = body.error;
      else if (body?.message) message = body.message;
    } catch { /* keep default */ }
    throw new ApiError(response.status, message);
  }

  if (response.status === 204) return undefined as T;
  const text = await response.text();
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}

export function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
