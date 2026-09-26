import { ApiError, apiFetch, getApiBaseUrl } from "./api";

// Phase 1 offline engine: connectivity helpers, service-worker registration,
// persistent workspace-settings snapshot (powers the offline calculator), and
// an IndexedDB outbox that queues field mutations while offline and replays
// them with auto-retry when the connection returns.
//
// Server endpoints used here are retry-safe by design:
// - attendance check-in/out: duplicate POSTs return 409 "already checked …"
// - checklist item PATCH: set-semantics (same value twice is a no-op)
// - checklist photo POST: single before/after slot per item (overwrite)
// - booking PATCH: conflicts surface as 409 and become dead-letters, never
//   silent retries.

export type OutboxType =
  | "attendance.check-in"
  | "attendance.check-out"
  | "checklist.item"
  | "checklist.photo"
  | "booking.patch"
  | "booking.accept";

export interface OutboxEntry {
  id: string;
  type: OutboxType;
  payload: Record<string, unknown>;
  createdAt: string;
  attempts: number;
  lastError: string | null;
}

export interface DeadEntry extends OutboxEntry {
  reason: string;
  failedAt: string;
}

export interface ProcessResult {
  applied: number;
  dead: number;
  remaining: number;
}

/** Thrown by lib helpers when a mutation was queued instead of sent. */
export class OfflineQueued extends Error {
  readonly entryId: string;
  readonly outboxType: OutboxType;

  constructor(type: OutboxType, entryId: string) {
    super("Saved offline — will sync when the connection returns");
    this.name = "OfflineQueued";
    this.entryId = entryId;
    this.outboxType = type;
  }
}

export function isOfflineQueued(e: unknown): e is OfflineQueued {
  return e instanceof OfflineQueued;
}

// ---------------------------------------------------------------------------
// Connectivity
// ---------------------------------------------------------------------------

export function isOnline(): boolean {
  if (typeof navigator === "undefined") return true;
  return navigator.onLine !== false;
}

export function onConnectivityChange(cb: (online: boolean) => void): () => void {
  if (typeof window === "undefined") return () => {};
  const handler = () => cb(isOnline());
  window.addEventListener("online", handler);
  window.addEventListener("offline", handler);
  return () => {
    window.removeEventListener("online", handler);
    window.removeEventListener("offline", handler);
  };
}

export async function registerServiceWorker(): Promise<boolean> {
  try {
    if (typeof window === "undefined" || !("serviceWorker" in navigator)) return false;
    await navigator.serviceWorker.register("/sw.js");
    return true;
  } catch {
    return false;
  }
}

let initialized = false;

/** Call once per page load: registers the SW, flushes the outbox now and on reconnect. */
export function initOffline(): void {
  if (initialized || typeof window === "undefined") return;
  initialized = true;
  void registerServiceWorker();
  void processOutbox().catch(() => {});
  onConnectivityChange((online) => {
    if (online) void processOutbox().catch(() => {});
  });
}

// ---------------------------------------------------------------------------
// Workspace settings snapshot (offline calculator catalog + tax rate)
// ---------------------------------------------------------------------------

const SETTINGS_KEY = "odysight_settings_v1";

export function persistSettingsSnapshot(ws: unknown): void {
  try {
    if (typeof localStorage === "undefined") return;
    localStorage.setItem(
      SETTINGS_KEY,
      JSON.stringify({ savedAt: new Date().toISOString(), settings: ws }),
    );
  } catch {
    /* storage full or unavailable — calculator falls back to defaults */
  }
}

export function readSettingsSnapshot<T>(): Partial<T> | null {
  try {
    if (typeof localStorage === "undefined") return null;
    const raw = localStorage.getItem(SETTINGS_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as { settings?: Partial<T> };
    if (!parsed || typeof parsed.settings !== "object" || parsed.settings === null) return null;
    return parsed.settings;
  } catch {
    return null;
  }
}

// ---------------------------------------------------------------------------
// Outbox storage (IndexedDB with in-memory fallback for SSR/tests)
// ---------------------------------------------------------------------------

const DB_NAME = "odysight-offline";
const DB_VERSION = 1;

interface MemoryStores {
  outbox: Map<string, OutboxEntry>;
  dead: Map<string, DeadEntry>;
}

const memory: MemoryStores = { outbox: new Map(), dead: new Map() };

function idbSupported(): boolean {
  return typeof indexedDB !== "undefined";
}

function openDb(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains("outbox")) {
        db.createObjectStore("outbox", { keyPath: "id" });
      }
      if (!db.objectStoreNames.contains("dead")) {
        db.createObjectStore("dead", { keyPath: "id" });
      }
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error ?? new Error("IndexedDB open failed"));
  });
}

function idbPut(store: "outbox" | "dead", value: OutboxEntry | DeadEntry): Promise<void> {
  return openDb().then(
    (db) =>
      new Promise<void>((resolve, reject) => {
        const tx = db.transaction(store, "readwrite");
        tx.oncomplete = () => { db.close(); resolve(); };
        tx.onerror = () => { db.close(); reject(tx.error ?? new Error("IDB put failed")); };
        tx.objectStore(store).put(value);
      }),
  );
}

function idbDelete(store: "outbox" | "dead", id: string): Promise<void> {
  return openDb().then(
    (db) =>
      new Promise<void>((resolve, reject) => {
        const tx = db.transaction(store, "readwrite");
        tx.oncomplete = () => { db.close(); resolve(); };
        tx.onerror = () => { db.close(); reject(tx.error ?? new Error("IDB delete failed")); };
        tx.objectStore(store).delete(id);
      }),
  );
}

function idbAll<T>(store: "outbox" | "dead"): Promise<T[]> {
  return openDb().then(
    (db) =>
      new Promise<T[]>((resolve, reject) => {
        const tx = db.transaction(store, "readonly");
        const req = tx.objectStore(store).getAll();
        req.onsuccess = () => {
          db.close();
          resolve((req.result ?? []) as T[]);
        };
        req.onerror = () => { db.close(); reject(req.error ?? new Error("IDB read failed")); };
      }),
  );
}

export function newOutboxId(): string {
  try {
    if (typeof crypto !== "undefined" && "randomUUID" in crypto) return crypto.randomUUID();
  } catch {
    /* fall through */
  }
  return `q_${Date.now().toString(36)}_${Math.floor(Math.random() * 1e9).toString(36)}`;
}

export async function enqueue(type: OutboxType, payload: Record<string, unknown>): Promise<string> {
  const entry: OutboxEntry = {
    id: newOutboxId(),
    type,
    payload,
    createdAt: new Date().toISOString(),
    attempts: 0,
    lastError: null,
  };
  if (idbSupported()) {
    try {
      await idbPut("outbox", entry);
      return entry.id;
    } catch {
      /* fall through to memory store */
    }
  }
  memory.outbox.set(entry.id, entry);
  return entry.id;
}

export async function pendingCount(): Promise<number> {
  if (idbSupported()) {
    try {
      return (await idbAll<OutboxEntry>("outbox")).length;
    } catch {
      /* fall through */
    }
  }
  return memory.outbox.size;
}

export async function listDeadLetters(): Promise<DeadEntry[]> {
  if (idbSupported()) {
    try {
      const rows = await idbAll<DeadEntry>("dead");
      return rows.sort((a, b) => b.failedAt.localeCompare(a.failedAt));
    } catch {
      /* fall through */
    }
  }
  return [...memory.dead.values()].sort((a, b) => b.failedAt.localeCompare(a.failedAt));
}

export async function clearDeadLetter(id: string): Promise<void> {
  memory.dead.delete(id);
  if (idbSupported()) {
    try {
      await idbDelete("dead", id);
    } catch {
      /* already gone */
    }
  }
}

async function loadOutbox(): Promise<OutboxEntry[]> {
  if (idbSupported()) {
    try {
      const rows = await idbAll<OutboxEntry>("outbox");
      return rows.sort((a, b) => a.createdAt.localeCompare(b.createdAt));
    } catch {
      /* fall through */
    }
  }
  return [...memory.outbox.values()].sort((a, b) => a.createdAt.localeCompare(b.createdAt));
}

async function saveOutbox(entry: OutboxEntry): Promise<void> {
  memory.outbox.set(entry.id, entry);
  if (idbSupported()) {
    try {
      await idbPut("outbox", entry);
    } catch {
      /* memory copy retained */
    }
  }
}

async function removeOutbox(id: string): Promise<void> {
  memory.outbox.delete(id);
  if (idbSupported()) {
    try {
      await idbDelete("outbox", id);
    } catch {
      /* already gone */
    }
  }
}

async function saveDead(entry: OutboxEntry, reason: string): Promise<void> {
  const dead: DeadEntry = { ...entry, reason, failedAt: new Date().toISOString() };
  memory.dead.set(dead.id, dead);
  if (idbSupported()) {
    try {
      await idbPut("dead", dead);
    } catch {
      /* memory copy retained */
    }
  }
}

// ---------------------------------------------------------------------------
// Sync engine
// ---------------------------------------------------------------------------

const MAX_ATTEMPTS = 25;

function duplicateMeansApplied(type: OutboxType, message: string): boolean {
  const msg = message.toLowerCase();
  if (type === "attendance.check-in") return msg.includes("already checked in");
  if (type === "attendance.check-out") return msg.includes("already checked out");
  return false;
}

function terminalFailure(type: OutboxType, status: number): boolean {
  // 409 on anything except "already applied" duplicates means the world moved
  // on (job reassigned, item deleted, double-booked): never silently retry.
  if (status === 404 || status === 422) return true;
  if (status === 409) return type !== "attendance.check-in" && type !== "attendance.check-out";
  return status === 400 || status === 401 || status === 403;
}

export type Sender = (entry: OutboxEntry) => Promise<unknown>;

async function defaultSender(entry: OutboxEntry): Promise<unknown> {
  switch (entry.type) {
    case "attendance.check-in":
      return apiFetch("/api/v1/attendance/check-in", {
        method: "POST",
        body: JSON.stringify({
          personType: entry.payload.personType,
          personId: entry.payload.personId,
          latitude: entry.payload.latitude,
          longitude: entry.payload.longitude,
          accuracy: entry.payload.accuracy,
        }),
      });
    case "attendance.check-out":
      return apiFetch("/api/v1/attendance/check-out", {
        method: "POST",
        body: JSON.stringify({
          personType: entry.payload.personType,
          personId: entry.payload.personId,
          latitude: entry.payload.latitude,
          longitude: entry.payload.longitude,
          accuracy: entry.payload.accuracy,
        }),
      });
    case "checklist.item":
      return apiFetch(`/api/v1/checklists/items/${String(entry.payload.itemId)}`, {
        method: "PATCH",
        body: JSON.stringify({
          isCompleted: entry.payload.isCompleted,
          completedBy: entry.payload.completedBy,
        }),
      });
    case "checklist.photo": {
      const base = getApiBaseUrl();
      if (!base) throw new ApiError(0, "PUBLIC_API_URL is not configured");
      const form = new FormData();
      form.append("kind", String(entry.payload.kind));
      form.append(
        "file",
        entry.payload.blob as Blob,
        String(entry.payload.fileName ?? "photo.jpg"),
      );
      const res = await fetch(`${base}/api/v1/checklists/items/${String(entry.payload.itemId)}/photo`, {
        method: "POST",
        body: form,
        credentials: "include",
      });
      if (!res.ok) {
        let message = `Upload failed: ${res.statusText || res.status}`;
        try {
          const body = (await res.clone().json()) as { error?: string; message?: string };
          if (body?.error) message = body.error;
          else if (body?.message) message = body.message;
        } catch {
          /* keep default */
        }
        throw new ApiError(res.status, message);
      }
      return res.json() as Promise<unknown>;
    }
    case "booking.patch":
      return apiFetch(`/api/v1/bookings/${String(entry.payload.id)}`, {
        method: "PATCH",
        body: JSON.stringify(entry.payload.patch),
      });
    case "booking.accept":
      // First tap wins server-side; a 409 means someone else took the job
      // while this device was offline — terminal, never retried.
      return apiFetch(`/api/v1/bookings/${String(entry.payload.id)}/accept`, {
        method: "POST",
        body: JSON.stringify({}),
      });
  }
}

function senderErrorInfo(e: unknown): { status: number; message: string } {
  if (e instanceof ApiError) return { status: e.status, message: e.message };
  if (e instanceof TypeError) return { status: 0, message: "Network unreachable" };
  return { status: 0, message: e instanceof Error ? e.message : "Sync failed" };
}

export async function processOutboxWith(sender: Sender): Promise<ProcessResult> {
  const entries = await loadOutbox();
  let applied = 0;
  let dead = 0;
  for (const entry of entries) {
    if (!isOnline()) break;
    try {
      await sender(entry);
      await removeOutbox(entry.id);
      applied += 1;
    } catch (e) {
      const { status, message } = senderErrorInfo(e);
      if (duplicateMeansApplied(entry.type, message)) {
        await removeOutbox(entry.id);
        applied += 1;
        continue;
      }
      entry.attempts += 1;
      entry.lastError = message;
      if (terminalFailure(entry.type, status) || entry.attempts >= MAX_ATTEMPTS) {
        await removeOutbox(entry.id);
        await saveDead(entry, message);
        dead += 1;
      } else {
        await saveOutbox(entry);
      }
    }
  }
  return { applied, dead, remaining: (await loadOutbox()).length };
}

export function processOutbox(): Promise<ProcessResult> {
  return processOutboxWith(defaultSender);
}
