import { USE_MOCKS, apiFetch, delay } from "./api";

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

const mockUsers: { password: string; user: AuthUser }[] = [
  {
    password: "admin123",
    user: {
      id: 1,
      name: "Admin User",
      email: "admin@example.com",
      role: "SUPER_ADMIN",
    },
  },
  {
    password: "dispatch123",
    user: {
      id: 2,
      name: "Dana Dispatch",
      email: "dispatch@example.com",
      role: "DISPATCH",
    },
  },
];

function setSessionCookie(token: string) {
  document.cookie = `${SESSION_COOKIE}=${token}; path=/; max-age=${60 * 60 * 8}; SameSite=Lax`;
}

function clearSessionCookie() {
  document.cookie = `${SESSION_COOKIE}=; path=/; max-age=0; SameSite=Lax`;
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
): Promise<AuthUser> {
  if (USE_MOCKS) {
    await delay(400);
    const match = mockUsers.find(
      (m) =>
        m.user.email.toLowerCase() === email.trim().toLowerCase() &&
        m.password === password,
    );
    if (!match) throw new Error("Invalid email or password");
    setSessionCookie(`mock-token-${match.user.id}`);
    localStorage.setItem(USER_KEY, JSON.stringify(match.user));
    return match.user;
  }

  const response = await apiFetch<LoginResponse>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
  setSessionCookie(response.token);
  localStorage.setItem(USER_KEY, JSON.stringify(response.user));
  return response.user;
}

export function logout(): void {
  clearSessionCookie();
  localStorage.removeItem(USER_KEY);
  window.location.href = "/login";
}

export function getInitials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]!.toUpperCase())
    .join("");
}
