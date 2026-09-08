import { USE_MOCKS, apiFetch, delay, toQuery, unwrapPage, type Page } from "./api";

export type NotificationChannel = "email" | "sms" | "whatsapp";

export interface NotificationLogEntry {
  id: number;
  channel: NotificationChannel;
  eventType: string;
  recipient: string;
  subject: string;
  status: "sent" | "failed" | "skipped";
  error: string;
  createdAt: string;
}

export interface ListParams {
  search?: string;
  status?: string;
  limit?: number;
  offset?: number;
}

const mockEntries: NotificationLogEntry[] = [
  {
    id: 51,
    channel: "email",
    eventType: "invoice.issued",
    recipient: "somchai@example.com",
    subject: "Invoice BK-2026-0201 issued",
    status: "sent",
    error: "",
    createdAt: "2026-09-01T10:00:00Z",
  },
  {
    id: 52,
    channel: "whatsapp",
    eventType: "booking.reminder",
    recipient: "+66 812345678",
    subject: "Reminder: your cleaning is tomorrow at 09:00",
    status: "sent",
    error: "",
    createdAt: "2026-09-05T08:30:00Z",
  },
  {
    id: 53,
    channel: "sms",
    eventType: "invoice.overdue",
    recipient: "jane.smith@example.com",
    subject: "Reminder: invoice is overdue",
    status: "failed",
    error: "provider returned status 429",
    createdAt: "2026-09-06T09:00:00Z",
  },
];

function clone(e: NotificationLogEntry): NotificationLogEntry {
  return { ...e };
}

function filterMocks(params: ListParams): NotificationLogEntry[] {
  const q = params.search?.trim().toLowerCase() ?? "";
  const rows = mockEntries.filter((e) => {
    if (params.status && e.status !== params.status) return false;
    if (!q) return true;
    return (
      e.recipient.toLowerCase().includes(q) ||
      e.subject.toLowerCase().includes(q) ||
      e.eventType.toLowerCase().includes(q)
    );
  });
  const offset = params.offset ?? 0;
  const limit = params.limit ?? rows.length;
  return rows.slice(offset, offset + limit).map(clone);
}

export async function getNotificationLog(params: ListParams = {}): Promise<NotificationLogEntry[]> {
  if (USE_MOCKS) {
    await delay(300);
    return filterMocks(params);
  }
  const json = await apiFetch<NotificationLogEntry[] | Page<NotificationLogEntry>>(
    "/api/v1/notifications" + toQuery(params as Record<string, string | number | undefined>),
  );
  return unwrapPage(json);
}