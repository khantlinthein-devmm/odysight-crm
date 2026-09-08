import { apiFetch, toQuery, unwrapPage, type Page } from "./api";

export interface AuditEntry {
  id: number;
  userId: number | null;
  userName: string;
  userEmail: string;
  action: string;
  resource: string;
  resourceId: number | null;
  createdAt: string;
}

export async function getAuditLogs(params: {
  search?: string;
  limit?: number;
  offset?: number;
} = {}): Promise<AuditEntry[]> {
  const json = await apiFetch<AuditEntry[] | Page<AuditEntry>>(
    "/api/v1/audit-logs" +
      toQuery(params as Record<string, string | number | undefined>),
  );
  return unwrapPage(json);
}
