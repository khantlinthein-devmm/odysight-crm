import { ApiError, USE_MOCKS, apiFetch, getApiBaseUrl } from "./api";

export interface ChatContact {
  userId: number;
  name: string;
  role: string;
}

export interface ChatConversation {
  id: number;
  /** The other person; for a group chat, the group's name (userId 0). */
  other: ChatContact;
  /** Set for a group chat: every member of the chat group talks here. */
  group?: { id: number; members: number; memberIds: number[] };
  lastMessageAt: string | null;
  lastKind: "" | "text" | "image" | "voice";
  lastBody: string;
  lastFromMe: boolean;
  /** Who sent the last message (group chats). */
  lastSender?: string;
  unread: number;
  /** Id of the last message the other person has read ("seen"). */
  otherLastRead: number;
}

export interface ChatMessage {
  id: number;
  conversationId: number;
  senderId: number | null;
  senderName: string;
  kind: "text" | "image" | "voice";
  body: string;
  fileUrl?: string;
  mimeType?: string;
  durationMs?: number;
  /** The voice note was deleted by the 30-day cleanup. */
  expired?: boolean;
  createdAt: string;
}

export interface ChatGroup {
  id: number;
  name: string;
  members: ChatContact[];
  createdAt: string;
}

type Wrapped<T> = { data: T };

export async function getChatUnread(): Promise<number> {
  if (USE_MOCKS || !getApiBaseUrl()) return 0;
  const r = await apiFetch<{ unread: number }>("/api/v1/chat/unread");
  return r.unread;
}

export async function getChatContacts(): Promise<ChatContact[]> {
  return (await apiFetch<Wrapped<ChatContact[]>>("/api/v1/chat/contacts")).data;
}

export async function getConversations(): Promise<ChatConversation[]> {
  return (await apiFetch<Wrapped<ChatConversation[]>>("/api/v1/chat/conversations")).data;
}

export async function openConversation(userId: number): Promise<ChatConversation> {
  return apiFetch<ChatConversation>("/api/v1/chat/conversations", {
    method: "POST",
    body: JSON.stringify({ userId }),
  });
}

export async function getMessages(
  conversationId: number,
  opts: { after?: number; before?: number; limit?: number } = {},
): Promise<ChatMessage[]> {
  const q = new URLSearchParams();
  if (opts.after) q.set("after", String(opts.after));
  if (opts.before) q.set("before", String(opts.before));
  if (opts.limit) q.set("limit", String(opts.limit));
  const qs = q.toString();
  return (
    await apiFetch<Wrapped<ChatMessage[]>>(
      `/api/v1/chat/conversations/${conversationId}/messages${qs ? `?${qs}` : ""}`,
    )
  ).data;
}

export async function sendText(conversationId: number, body: string): Promise<ChatMessage> {
  return apiFetch<ChatMessage>(`/api/v1/chat/conversations/${conversationId}/messages`, {
    method: "POST",
    body: JSON.stringify({ body }),
  });
}

export async function sendFile(
  conversationId: number,
  kind: "image" | "voice",
  file: Blob,
  opts: { caption?: string; durationMs?: number; fileName?: string } = {},
): Promise<ChatMessage> {
  const base = getApiBaseUrl();
  if (!base) throw new ApiError(0, "PUBLIC_API_URL is not configured");
  const form = new FormData();
  form.append("kind", kind);
  if (opts.caption) form.append("caption", opts.caption);
  if (opts.durationMs) form.append("durationMs", String(Math.round(opts.durationMs)));
  form.append("file", file, opts.fileName ?? (kind === "image" ? "photo.jpg" : "voice"));
  const res = await fetch(`${base}/api/v1/chat/conversations/${conversationId}/messages`, {
    method: "POST",
    body: form,
    credentials: "include",
  });
  if (!res.ok) {
    let message = `Upload failed (${res.status})`;
    try {
      const b = (await res.json()) as { error?: string };
      if (b.error) message = b.error;
    } catch {
      /* keep default */
    }
    throw new ApiError(res.status, message);
  }
  return (await res.json()) as ChatMessage;
}

/** Tells the other person "… is typing" (callers throttle this). */
export async function sendTyping(conversationId: number): Promise<void> {
  await apiFetch<void>(`/api/v1/chat/conversations/${conversationId}/typing`, { method: "POST" });
}

export async function markRead(conversationId: number, messageId: number): Promise<void> {
  await apiFetch<void>(`/api/v1/chat/conversations/${conversationId}/read`, {
    method: "POST",
    body: JSON.stringify({ messageId }),
  });
  window.dispatchEvent(new Event("chat:read"));
}

// Photos and voice notes are private: fetch them with the session cookie and
// hand the element an object URL. Cached per file for the page's lifetime.
const blobCache = new Map<string, Promise<string>>();

export function chatFileUrl(fileUrl: string): Promise<string> {
  let p = blobCache.get(fileUrl);
  if (!p) {
    const base = getApiBaseUrl();
    p = fetch(`${base}${fileUrl}`, { credentials: "include" }).then(async (r) => {
      if (!r.ok) throw new ApiError(r.status, "Failed to load file");
      return URL.createObjectURL(await r.blob());
    });
    p.catch(() => blobCache.delete(fileUrl));
    blobCache.set(fileUrl, p);
  }
  return p;
}

// Groups (admins)

export async function getChatGroups(): Promise<ChatGroup[]> {
  return (await apiFetch<Wrapped<ChatGroup[]>>("/api/v1/chat/groups")).data;
}

export async function saveChatGroup(
  id: number | null,
  input: { name: string; memberIds: number[] },
): Promise<void> {
  await apiFetch<void>(id ? `/api/v1/chat/groups/${id}` : "/api/v1/chat/groups", {
    method: id ? "PATCH" : "POST",
    body: JSON.stringify(input),
  });
}

export async function deleteChatGroup(id: number): Promise<void> {
  await apiFetch<void>(`/api/v1/chat/groups/${id}`, { method: "DELETE" });
}

// Push notifications

function urlBase64ToUint8Array(b64: string): Uint8Array {
  const padded = (b64 + "=".repeat((4 - (b64.length % 4)) % 4)).replace(/-/g, "+").replace(/_/g, "/");
  const raw = atob(padded);
  const out = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i);
  return out;
}

export type PushState = "unsupported" | "denied" | "off" | "on";

export async function pushState(): Promise<PushState> {
  if (typeof window === "undefined" || !("serviceWorker" in navigator) || !("PushManager" in window) || !("Notification" in window)) {
    return "unsupported";
  }
  if (Notification.permission === "denied") return "denied";
  const reg = await navigator.serviceWorker.getRegistration();
  const sub = await reg?.pushManager.getSubscription();
  return sub ? "on" : "off";
}

/** Asks for permission and registers this device for chat notifications. */
export async function enablePush(): Promise<PushState> {
  if ((await pushState()) === "unsupported") return "unsupported";
  const permission = await Notification.requestPermission();
  if (permission !== "granted") return permission === "denied" ? "denied" : "off";
  const reg = (await navigator.serviceWorker.getRegistration()) ?? (await navigator.serviceWorker.register("/sw.js"));
  await navigator.serviceWorker.ready;
  const { publicKey } = await apiFetch<{ publicKey: string }>("/api/v1/chat/push/key");
  const key = urlBase64ToUint8Array(publicKey);
  let sub = await reg.pushManager.getSubscription();
  // A subscription made with a different server key would be rejected by
  // the push service; replace it.
  if (sub && !sameKey(sub.options.applicationServerKey, key)) {
    await sub.unsubscribe();
    sub = null;
  }
  if (!sub) {
    sub = await reg.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: key as BufferSource });
  }
  await apiFetch<void>("/api/v1/chat/push/subscribe", {
    method: "POST",
    body: JSON.stringify(sub.toJSON()),
  });
  return "on";
}

function sameKey(a: ArrayBuffer | null, b: Uint8Array): boolean {
  if (!a) return false;
  const x = new Uint8Array(a);
  return x.length === b.length && x.every((v, i) => v === b[i]);
}

export async function disablePush(): Promise<PushState> {
  const reg = await navigator.serviceWorker.getRegistration();
  const sub = await reg?.pushManager.getSubscription();
  if (sub) {
    await apiFetch<void>("/api/v1/chat/push/unsubscribe", {
      method: "POST",
      body: JSON.stringify({ endpoint: sub.endpoint }),
    }).catch(() => {});
    await sub.unsubscribe();
  }
  return "off";
}

// Presence

/** Ids of the people you can message who have the app open right now. */
export async function getPresence(): Promise<number[]> {
  if (USE_MOCKS || !getApiBaseUrl()) return [];
  return (await apiFetch<{ online: number[] }>("/api/v1/chat/presence")).online;
}
