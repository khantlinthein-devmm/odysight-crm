import { USE_MOCKS, apiFetch, delay, getApiBaseUrl } from "./api";

export type Role =
  | "SUPER_ADMIN"
  | "ADMIN"
  | "MANAGER"
  | "DISPATCH"
  | "ACCOUNTANT"
  | "CLEANER";

export interface AuthUser {
  id: number;
  name: string;
  email: string;
  role: Role;
}

interface LoginResponse {
  token: string;
  user: AuthUser;
}

const SESSION_COOKIE = "odysight_session";
const USER_KEY = "odysight_user";

// Dev-only mock users. Never used in production (USE_MOCKS is always false
// in prod builds). Credentials are env-overridable for local dev; defaults are
// local-dev-only and must not be treated as real secrets.
function getMockUsers(): { password: string; user: AuthUser }[] {
  if (import.meta.env.PROD) return [];
  return [
    {
      password:
        import.meta.env.PUBLIC_MOCK_ADMIN_PASSWORD ?? "admin123",
      user: {
        id: 1,
        name: "Admin User",
        email:
          import.meta.env.PUBLIC_MOCK_ADMIN_EMAIL ?? "admin@example.com",
        role: "SUPER_ADMIN",
      },
    },
    {
      password:
        import.meta.env.PUBLIC_MOCK_DISPATCH_PASSWORD ?? "dispatch123",
      user: {
        id: 2,
        name: "Dana Dispatch",
        email:
          import.meta.env.PUBLIC_MOCK_DISPATCH_EMAIL ?? "dispatch@example.com",
        role: "DISPATCH",
      },
    },
  ];
}

// In mock (dev) mode there is no API to set the HttpOnly cookie, so the mock
// session cookie is written from JS. This never runs in production builds.
function setMockSessionCookie(token: string) {
  document.cookie = `${SESSION_COOKIE}=${encodeURIComponent(token)}; path=/; max-age=${60 * 60 * 8}; SameSite=Lax`;
}

function clearMockSessionCookie() {
  document.cookie = `${SESSION_COOKIE}=; path=/; max-age=0; SameSite=Lax`;
}

function clearLocalUser() {
  try { localStorage.removeItem(USER_KEY); } catch { /* ignore */ }
}

export function getSessionUser(): AuthUser | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(USER_KEY);
    return raw ? (JSON.parse(raw) as AuthUser) : null;
  } catch {
    return null;
  }
}

export async function login(
  email: string,
  password: string,
  rememberMe = false,
): Promise<AuthUser> {
  if (USE_MOCKS) {
    await delay(400);
    const mockUsers = getMockUsers();
    const match = mockUsers.find(
      (m) =>
        m.user.email.toLowerCase() === email.trim().toLowerCase() &&
        m.password === password,
    );
    if (!match) throw new Error("Invalid email or password");
    setMockSessionCookie(`mock-token-${match.user.id}`);
    localStorage.setItem(USER_KEY, JSON.stringify(match.user));
    return match.user;
  }

  const response = await apiFetch<LoginResponse>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
  // The API sets the HttpOnly odysight_session cookie. The token is never
  // stored in localStorage or written to a JS-readable cookie.
  localStorage.setItem(USER_KEY, JSON.stringify(response.user));
  void rememberMe;
  return response.user;
}

export function clearSession(): void {
  if (USE_MOCKS) {
    clearMockSessionCookie();
  }
  clearLocalUser();
}

export function logout(): void {
  // Best-effort clear of the server-side HttpOnly cookie; never blocks logout.
  if (!USE_MOCKS && typeof window !== "undefined") {
    const api = getApiBaseUrl();
    if (api) {
      fetch(`${api}/api/v1/auth/logout`, {
        method: "POST",
        credentials: "include",
      }).catch(() => { /* ignore */ });
    }
  }
  clearSession();
  window.location.href = "/login";
}

export async function changePassword(
  currentPassword: string,
  newPassword: string,
): Promise<void> {
  if (USE_MOCKS) {
    await delay(400);
    if (!currentPassword || newPassword.length < 8) {
      throw new Error("New password must be at least 8 characters");
    }
    return;
  }
  await apiFetch<{ status: string }>("/api/v1/auth/change-password", {
    method: "POST",
    body: JSON.stringify({ currentPassword, newPassword }),
  });
}

export function getInitials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]!.toUpperCase())
    .join("");
}
