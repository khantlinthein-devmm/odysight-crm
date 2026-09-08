import { apiFetch, toQuery, unwrapPage, type Page } from "./api";
import type { Role } from "./auth";

export interface TeamUser {
  id: number;
  name: string;
  email: string;
  role: Role;
  createdAt: string;
}

export const TEAM_ROLES: Role[] = [
  "SUPER_ADMIN",
  "ADMIN",
  "MANAGER",
  "DISPATCH",
  "ACCOUNTANT",
  "CLEANER",
];

export interface CreateUserInput {
  name: string;
  email: string;
  password: string;
  role: Role;
}

export interface UpdateUserInput {
  name?: string;
  role?: Role;
}

export async function getUsers(params: {
  search?: string;
  limit?: number;
  offset?: number;
} = {}): Promise<TeamUser[]> {
  const json = await apiFetch<TeamUser[] | Page<TeamUser>>(
    "/api/v1/users" +
      toQuery(params as Record<string, string | number | undefined>),
  );
  return unwrapPage(json);
}

export async function createUser(input: CreateUserInput): Promise<TeamUser> {
  return apiFetch<TeamUser>("/api/v1/users", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateUser(
  id: number,
  input: UpdateUserInput,
): Promise<TeamUser> {
  return apiFetch<TeamUser>(`/api/v1/users/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function resetUserPassword(
  id: number,
  newPassword: string,
): Promise<void> {
  await apiFetch<{ status: string }>(`/api/v1/users/${id}/reset-password`, {
    method: "POST",
    body: JSON.stringify({ newPassword }),
  });
}

export async function deleteUser(id: number): Promise<void> {
  await apiFetch<void>(`/api/v1/users/${id}`, { method: "DELETE" });
}
