import { USE_MOCKS, getApiBaseUrl } from "./api";

// One Server-Sent Events connection per page, shared by the chat screen, the
// sidebar badge and the bottom bar. Events are hints ("conversation 12 has
// message 345"); listeners re-fetch through the normal API. EventSource
// reconnects on its own, and "open" fires again after every reconnect so
// listeners can catch up on anything missed while offline.

export type ChatEventType = "message" | "read" | "typing" | "open";

export interface ChatEventData {
  conversationId?: number;
  messageId?: number;
  senderId?: number;
  userId?: number;
  /** Who is typing (group chats show the name). */
  name?: string;
}

type Listener = (type: ChatEventType, data: ChatEventData) => void;

const listeners = new Set<Listener>();
let source: EventSource | null = null;
let connected = false;
let retryDelay = 1000;

function emit(type: ChatEventType, data: ChatEventData) {
  for (const l of listeners) {
    try {
      l(type, data);
    } catch {
      /* one bad listener must not break the others */
    }
  }
}

function connect() {
  if (source || USE_MOCKS || typeof EventSource === "undefined") return;
  const base = getApiBaseUrl();
  if (!base) return;
  source = new EventSource(`${base}/api/v1/chat/events`, { withCredentials: true });
  source.onopen = () => {
    connected = true;
    retryDelay = 1000;
    emit("open", {});
  };
  source.onerror = () => {
    connected = false;
    // The browser retries network drops itself, but gives up for good
    // (CLOSED) on an HTTP error, e.g. a 502 while the API restarts. Reconnect
    // ourselves with backoff; a signed-out user just backs off to a minute.
    if (source?.readyState === EventSource.CLOSED) {
      source.close();
      source = null;
      if (listeners.size > 0) {
        setTimeout(connect, retryDelay);
        retryDelay = Math.min(retryDelay * 2, 60_000);
      }
    }
  };
  for (const type of ["message", "read", "typing"] as const) {
    source.addEventListener(type, (e) => {
      let data: ChatEventData = {};
      try {
        data = JSON.parse((e as MessageEvent<string>).data) as ChatEventData;
      } catch {
        /* ignore malformed */
      }
      emit(type, data);
    });
  }
}

/** Subscribe to chat events; returns an unsubscribe function. */
export function onChatEvent(listener: Listener): () => void {
  listeners.add(listener);
  connect();
  return () => listeners.delete(listener);
}

/** True while the live connection is up (polling can slow down). */
export function chatLive(): boolean {
  return connected;
}
