<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import {
  enablePush,
  getChatContacts,
  getConversations,
  getMessages,
  markRead,
  openConversation,
  pushState,
  sendFile,
  sendText,
  sendTyping,
  type ChatContact,
  type ChatConversation,
  type ChatMessage,
  type PushState,
} from "../../../lib/chat";
import { getSessionUser } from "../../../lib/auth";
import { hasPermission } from "../../../lib/roles";
import { dateLocale, getLang, t, type Lang, type MessageKey } from "../../../lib/i18n";
import { showToast } from "../../../lib/toast";
import { chatLive, onChatEvent, type ChatEventData, type ChatEventType } from "../../../lib/chatEvents";
import ChatMedia from "./ChatMedia.vue";
import ChatGroups from "./ChatGroups.vue";
import ChatAvatar from "./ChatAvatar.vue";
import { usePresence } from "./usePresence";

const user = getSessionUser();
const meId = Number(user?.id ?? 0);
const canManageGroups = hasPermission(user?.role ?? "viewer", "users.manage");

// Cleaners use the app in their own language; office staff in English,
// matching the rest of the field screens.
const lang = ref<Lang>("en");
function L(key: MessageKey): string {
  return t(key, lang.value);
}

const tab = ref<"chats" | "groups">("chats");
const conversations = ref<ChatConversation[]>([]);
const active = ref<ChatConversation | null>(null);
const messages = ref<ChatMessage[]>([]);
const loadingThread = ref(false);
const hasOlder = ref(false);
const text = ref("");
const sending = ref(false);
const push = ref<PushState>("unsupported");

const showPicker = ref(false);
const contacts = ref<ChatContact[]>([]);
const contactFilter = ref("");

const scroller = ref<HTMLElement | null>(null);
const root = ref<HTMLElement | null>(null);
const { isOnline } = usePresence();

let convTimer: ReturnType<typeof setInterval> | undefined;
let threadTimer: ReturnType<typeof setInterval> | undefined;

const filteredContacts = computed(() => {
  const q = contactFilter.value.trim().toLowerCase();
  return contacts.value.filter((c) => !q || `${c.name} ${c.role}`.toLowerCase().includes(q));
});

const lastMineSeen = computed(() => {
  const conv = active.value;
  if (!conv) return 0;
  const mine = messages.value.filter((m) => m.senderId === meId);
  const last = mine[mine.length - 1];
  return last && conv.otherLastRead >= last.id ? last.id : 0;
});

function roleLabel(role: string): string {
  return role.charAt(0) + role.slice(1).toLowerCase().replace("_", " ");
}

function preview(c: ChatConversation): string {
  const who = c.lastFromMe ? `${L("chat.you")}: ` : c.group && c.lastSender ? `${c.lastSender}: ` : "";
  if (c.lastKind === "image") return `${who}📷 ${c.lastBody || L("chat.photoMessage")}`;
  if (c.lastKind === "voice") return `${who}🎤 ${L("chat.voiceMessage")}`;
  return who + c.lastBody;
}

function shortTime(iso: string | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  const now = new Date();
  if (d.toDateString() === now.toDateString()) {
    return d.toLocaleTimeString(dateLocale(lang.value), { hour: "2-digit", minute: "2-digit" });
  }
  return d.toLocaleDateString(dateLocale(lang.value), { day: "numeric", month: "short" });
}

function dayLabel(iso: string): string {
  const d = new Date(iso);
  const now = new Date();
  const y = new Date(now);
  y.setDate(now.getDate() - 1);
  if (d.toDateString() === now.toDateString()) return L("chat.today");
  if (d.toDateString() === y.toDateString()) return L("chat.yesterday");
  return d.toLocaleDateString(dateLocale(lang.value), { weekday: "short", day: "numeric", month: "short" });
}

// In a group chat, name the sender above the first of their messages in a row.
function showSender(i: number): boolean {
  const m = messages.value[i]!;
  if (!active.value?.group || m.senderId === meId) return false;
  const prev = messages.value[i - 1];
  return !prev || prev.senderId !== m.senderId || newDay(i);
}

function newDay(i: number): boolean {
  if (i === 0) return true;
  return new Date(messages.value[i - 1]!.createdAt).toDateString() !== new Date(messages.value[i]!.createdAt).toDateString();
}

function timeOf(iso: string): string {
  return new Date(iso).toLocaleTimeString(dateLocale(lang.value), { hour: "2-digit", minute: "2-digit" });
}

function nearBottom(): boolean {
  const el = scroller.value;
  return !el || el.scrollHeight - el.scrollTop - el.clientHeight < 120;
}

async function scrollToBottom() {
  await nextTick();
  const el = scroller.value;
  if (el) el.scrollTop = el.scrollHeight;
}

async function loadConversations() {
  try {
    conversations.value = await getConversations();
    if (active.value) {
      const fresh = conversations.value.find((c) => c.id === active.value!.id);
      if (fresh) active.value = { ...active.value, otherLastRead: fresh.otherLastRead };
    }
  } catch {
    /* keep the last list; the next poll retries */
  }
}

async function select(conv: ChatConversation) {
  active.value = conv;
  messages.value = [];
  loadingThread.value = true;
  history.replaceState(null, "", `/chat?c=${conv.id}`);
  try {
    const page = await getMessages(conv.id, { limit: 50 });
    messages.value = page;
    hasOlder.value = page.length === 50;
    await scrollToBottom();
    await markSeen();
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to load messages", "error");
  } finally {
    loadingThread.value = false;
  }
}

function closeThread() {
  active.value = null;
  messages.value = [];
  history.replaceState(null, "", "/chat");
}

async function loadOlder() {
  const conv = active.value;
  const first = messages.value[0];
  if (!conv || !first) return;
  const el = scroller.value;
  const before = el ? el.scrollHeight : 0;
  const page = await getMessages(conv.id, { before: first.id, limit: 50 });
  hasOlder.value = page.length === 50;
  messages.value = [...page, ...messages.value];
  await nextTick();
  if (el) el.scrollTop = el.scrollHeight - before;
}

async function markSeen() {
  const conv = active.value;
  const last = messages.value[messages.value.length - 1];
  if (!conv || !last || document.visibilityState !== "visible") return;
  await markRead(conv.id, last.id).catch(() => {});
  const listed = conversations.value.find((c) => c.id === conv.id);
  if (listed) listed.unread = 0;
}

async function pollThread() {
  const conv = active.value;
  if (!conv || loadingThread.value || document.visibilityState !== "visible") return;
  const last = messages.value[messages.value.length - 1];
  try {
    const fresh = await getMessages(conv.id, { after: last?.id ?? 0, limit: 100 });
    if (active.value?.id !== conv.id || fresh.length === 0) return;
    const stick = nearBottom();
    const known = new Set(messages.value.map((m) => m.id));
    messages.value = [...messages.value, ...fresh.filter((m) => !known.has(m.id))];
    if (stick) await scrollToBottom();
    await markSeen();
  } catch {
    /* transient; next poll */
  }
}

function appendMine(msg: ChatMessage) {
  if (!messages.value.some((m) => m.id === msg.id)) messages.value = [...messages.value, msg];
  void scrollToBottom();
  void loadConversations();
}

async function submitText() {
  const conv = active.value;
  const body = text.value.trim();
  if (!conv || !body || sending.value) return;
  sending.value = true;
  try {
    appendMine(await sendText(conv.id, body));
    text.value = "";
  } catch (err) {
    showToast(err instanceof Error ? err.message : L("chat.sendFailed"), "error");
  } finally {
    sending.value = false;
  }
}

function onKey(e: KeyboardEvent) {
  // Enter sends on desktop; phones keep Enter for new lines.
  if (e.key === "Enter" && !e.shiftKey && !e.isComposing && window.matchMedia("(pointer: fine)").matches) {
    e.preventDefault();
    void submitText();
  }
}

// Voice notes: MediaRecorder picks webm/opus (Chrome, Android, Firefox) or
// mp4/aac (iPhone Safari), at 24 kbps — clear speech at ~180 KB a minute.
// Recording stops itself after one minute; the server deletes notes after
// 30 days.
const recording = ref(false);
const recordMs = ref(0);
let recorder: MediaRecorder | null = null;
let chunks: Blob[] = [];
let recordStart = 0;
let recordTick: ReturnType<typeof setInterval> | undefined;
let stream: MediaStream | null = null;
let sendAfterStop = false;
const MAX_RECORD_MS = 60_000;

function pickMime(): string {
  for (const m of ["audio/webm;codecs=opus", "audio/webm", "audio/mp4", "audio/ogg;codecs=opus"]) {
    if (typeof MediaRecorder !== "undefined" && MediaRecorder.isTypeSupported(m)) return m;
  }
  return "";
}

async function startRecording() {
  if (!active.value || recording.value) return;
  try {
    stream = await navigator.mediaDevices.getUserMedia({ audio: true });
  } catch {
    showToast(L("chat.micDenied"), "error");
    return;
  }
  const mime = pickMime();
  recorder = new MediaRecorder(stream, {
    ...(mime ? { mimeType: mime } : {}),
    audioBitsPerSecond: 24_000,
  });
  chunks = [];
  sendAfterStop = false;
  recorder.ondataavailable = (e) => {
    if (e.data.size > 0) chunks.push(e.data);
  };
  recorder.onstop = () => void finishRecording();
  recorder.start();
  recordStart = Date.now();
  recordMs.value = 0;
  recording.value = true;
  recordTick = setInterval(() => {
    recordMs.value = Date.now() - recordStart;
    if (recordMs.value >= MAX_RECORD_MS) stopRecording(true);
  }, 200);
}

function stopRecording(send: boolean) {
  sendAfterStop = send;
  clearInterval(recordTick);
  recordMs.value = Date.now() - recordStart;
  if (recorder && recorder.state !== "inactive") recorder.stop();
  else void finishRecording();
}

async function finishRecording() {
  stream?.getTracks().forEach((tr) => tr.stop());
  stream = null;
  recording.value = false;
  const conv = active.value;
  const type = recorder?.mimeType || "audio/webm";
  recorder = null;
  if (!sendAfterStop || !conv || chunks.length === 0 || recordMs.value < 700) return;
  const blob = new Blob(chunks, { type });
  sending.value = true;
  try {
    const ext = type.includes("mp4") ? "m4a" : type.includes("ogg") ? "ogg" : "webm";
    appendMine(await sendFile(conv.id, "voice", blob, { durationMs: recordMs.value, fileName: `voice.${ext}` }));
  } catch (err) {
    showToast(err instanceof Error ? err.message : L("chat.sendFailed"), "error");
  } finally {
    sending.value = false;
  }
}

function recordLabel(): string {
  const s = Math.floor(recordMs.value / 1000);
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
}

async function openPicker() {
  contactFilter.value = "";
  showPicker.value = true;
  try {
    contacts.value = await getChatContacts();
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to load people", "error");
  }
}

async function startWith(c: ChatContact) {
  showPicker.value = false;
  try {
    const conv = await openConversation(c.userId);
    await select(conv);
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to open chat", "error");
  }
}

async function turnOnPush() {
  try {
    push.value = await enablePush();
    if (push.value === "on") showToast(L("chat.notifOn"), "success");
    else if (push.value === "denied") showToast(L("chat.notifBlocked"), "error");
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to turn on notifications", "error");
  }
}

// Live updates. With the event stream up, polling drops to a 30 s safety net.
const typingUntil = ref(0);
const typingName = ref("");
const now = ref(Date.now());
let clock: ReturnType<typeof setInterval> | undefined;
let offEvents: (() => void) | undefined;
let lastTypingSent = 0;
let tick = 0;

const otherTyping = computed(() => typingUntil.value > now.value);
const groupOnline = computed(() => (active.value?.group?.memberIds ?? []).filter((id) => id !== meId && isOnline(id)).length);

// Phones: while typing, pin the chat full screen (like messaging apps) and
// end it just above the on-screen keyboard. Browsers report the keyboard
// differently: iPhone and Android Chrome tabs shrink the visible area
// (visualViewport), but an installed Android app can let the keyboard cover
// the page without shrinking anything. There Chrome's VirtualKeyboard API
// tells us the keyboard's height, so we opt in to it and use it.
interface VirtualKeyboardLike extends EventTarget {
  overlaysContent: boolean;
  boundingRect: DOMRect;
}
const virtualKeyboard = (navigator as Navigator & { virtualKeyboard?: VirtualKeyboardLike }).virtualKeyboard;
const keyboardStyle = ref<Record<string, string>>({});
function fitToKeyboard() {
  const vv = window.visualViewport;
  const el = document.activeElement;
  const typing = !!vv && !!el && el.tagName === "TEXTAREA" && !!root.value?.contains(el);
  // Pin whenever the text box has focus on a phone-sized screen.
  const open = typing && window.matchMedia("(max-width: 1023px)").matches;
  const was = Object.keys(keyboardStyle.value).length > 0;
  let height = vv?.height ?? window.innerHeight;
  const covered = virtualKeyboard?.boundingRect.height ?? 0;
  if (covered > 0) height = Math.min(height, window.innerHeight - covered - (vv?.offsetTop ?? 0));
  keyboardStyle.value = open
    ? {
        position: "fixed",
        top: `${vv!.offsetTop}px`,
        left: "0",
        right: "0",
        height: `${Math.max(200, height)}px`,
        minHeight: "0",
        zIndex: "60",
        borderRadius: "0",
      }
    : {};
  if (open && !was) void scrollToBottom();
}
function onFocusChange() {
  // Let the keyboard animation settle, then fit again.
  fitToKeyboard();
  setTimeout(fitToKeyboard, 300);
  setTimeout(fitToKeyboard, 800);
}

function onEvent(type: ChatEventType, data: ChatEventData) {
  const conv = active.value;
  const here = !!conv && data.conversationId === conv.id;
  switch (type) {
    case "open": // (re)connected: catch up on anything missed
      void loadConversations();
      void pollThread();
      break;
    case "message":
      if (here) {
        if (data.senderId !== meId) typingUntil.value = 0;
        void pollThread();
      }
      void loadConversations();
      break;
    case "read":
      if (here && data.userId !== meId && data.messageId && conv.otherLastRead < data.messageId) {
        active.value = { ...conv, otherLastRead: data.messageId };
      }
      if (data.userId === meId) void loadConversations();
      break;
    case "typing":
      if (here && data.userId !== meId) {
        typingUntil.value = Date.now() + 4000;
        typingName.value = data.name ?? "";
      }
      break;
  }
}

// Tell the other person we're typing, at most every 3 s.
watch(text, (v) => {
  const conv = active.value;
  if (!conv || !v.trim() || Date.now() - lastTypingSent < 3000) return;
  lastTypingSent = Date.now();
  void sendTyping(conv.id).catch(() => {});
});

function onVisible() {
  if (document.visibilityState === "visible") {
    void loadConversations();
    void pollThread();
  }
}

onMounted(async () => {
  lang.value = user?.role === "CLEANER" ? getLang() : "en";
  push.value = await pushState().catch(() => "unsupported" as PushState);
  await loadConversations();
  const deep = Number(new URLSearchParams(window.location.search).get("c"));
  if (deep) {
    const conv = conversations.value.find((c) => c.id === deep);
    if (conv) await select(conv);
  }
  offEvents = onChatEvent(onEvent);
  clock = setInterval(() => (now.value = Date.now()), 1000);
  // Fallback polling: every 5 s / 15 s only while the live stream is down,
  // otherwise every 30 s as a safety net.
  convTimer = setInterval(() => {
    if (document.visibilityState === "visible" && !chatLive()) void loadConversations();
  }, 15_000);
  threadTimer = setInterval(() => {
    tick++;
    if (!chatLive() || tick % 6 === 0) void pollThread();
  }, 5_000);
  document.addEventListener("visibilitychange", onVisible);
  window.visualViewport?.addEventListener("resize", fitToKeyboard);
  window.visualViewport?.addEventListener("scroll", fitToKeyboard);
  if (virtualKeyboard) {
    // Report the keyboard's size (geometrychange) instead of letting it
    // silently cover the page. Only on the chat screen; reset on leave.
    virtualKeyboard.overlaysContent = true;
    virtualKeyboard.addEventListener("geometrychange", fitToKeyboard);
  }
  document.addEventListener("focusin", onFocusChange);
  document.addEventListener("focusout", onFocusChange);
});

onBeforeUnmount(() => {
  clearInterval(convTimer);
  clearInterval(threadTimer);
  clearInterval(clock);
  offEvents?.();
  document.removeEventListener("visibilitychange", onVisible);
  window.visualViewport?.removeEventListener("resize", fitToKeyboard);
  window.visualViewport?.removeEventListener("scroll", fitToKeyboard);
  if (virtualKeyboard) {
    virtualKeyboard.removeEventListener("geometrychange", fitToKeyboard);
    virtualKeyboard.overlaysContent = false;
  }
  document.removeEventListener("focusin", onFocusChange);
  document.removeEventListener("focusout", onFocusChange);
  if (recording.value) stopRecording(false);
});
</script>

<template>
  <div
    ref="root"
    class="flex h-[calc(100dvh-9rem)] min-h-[420px] flex-col overflow-hidden rounded-xl border border-gray-200 bg-white lg:h-[calc(100dvh-8rem)]"
    :style="keyboardStyle"
  >
    <div v-if="canManageGroups" class="flex border-b border-gray-200 text-sm font-medium">
      <button
        type="button"
        class="px-4 py-2.5"
        :class="tab === 'chats' ? 'border-b-2 border-navy-600 text-navy-700' : 'text-gray-500 hover:text-gray-800'"
        @click="tab = 'chats'"
      >{{ L("nav.chat") }}</button>
      <button
        type="button"
        class="px-4 py-2.5"
        :class="tab === 'groups' ? 'border-b-2 border-navy-600 text-navy-700' : 'text-gray-500 hover:text-gray-800'"
        @click="tab = 'groups'"
      >Chat groups</button>
    </div>

    <div v-if="tab === 'groups'" class="min-h-0 flex-1 overflow-y-auto">
      <ChatGroups />
    </div>

    <div v-else class="flex min-h-0 flex-1">
      <!-- Conversation list -->
      <aside
        class="w-full flex-col border-r border-gray-200 md:flex md:w-80 md:shrink-0"
        :class="active ? 'hidden' : 'flex'"
      >
        <div class="flex items-center justify-between gap-2 border-b border-gray-100 px-4 py-3">
          <h2 class="text-base font-semibold text-gray-900">{{ L("chat.title") }}</h2>
          <button
            type="button"
            class="rounded-lg bg-navy-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-navy-700"
            @click="openPicker"
          >{{ L("chat.newMessage") }}</button>
        </div>
        <button
          v-if="push === 'off'"
          type="button"
          class="mx-3 mt-3 rounded-lg border border-navy-200 bg-navy-50 px-3 py-2 text-left text-xs font-medium text-navy-700 hover:bg-navy-100"
          @click="turnOnPush"
        >🔔 {{ L("chat.notifEnable") }}</button>
        <p v-else-if="push === 'denied'" class="mx-3 mt-3 rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-800">{{ L("chat.notifBlocked") }}</p>
        <p v-else-if="push === 'unsupported'" class="mx-3 mt-3 rounded-lg bg-gray-50 px-3 py-2 text-xs text-gray-500">{{ L("chat.notifUnsupported") }}</p>

        <p v-if="conversations.length === 0" class="px-4 py-8 text-center text-sm text-gray-500">{{ L("chat.empty") }}</p>
        <ul class="min-h-0 flex-1 overflow-y-auto">
          <li v-for="c in conversations" :key="c.id">
            <button
              type="button"
              class="flex w-full items-center gap-3 px-4 py-3 text-left hover:bg-gray-50"
              :class="active?.id === c.id ? 'bg-navy-50' : ''"
              @click="select(c)"
            >
              <ChatAvatar :name="c.other.name" :group="!!c.group" :online="!c.group && isOnline(c.other.userId)" />
              <span class="min-w-0 flex-1">
                <span class="flex items-baseline justify-between gap-2">
                  <span class="truncate text-sm font-medium text-gray-900">{{ c.other.name }}</span>
                  <span class="shrink-0 text-[11px] text-gray-400">{{ shortTime(c.lastMessageAt) }}</span>
                </span>
                <span class="flex items-center justify-between gap-2">
                  <span class="truncate text-xs" :class="c.unread ? 'font-semibold text-gray-900' : 'text-gray-500'">{{ preview(c) }}</span>
                  <span v-if="c.unread" class="min-w-5 shrink-0 rounded-full bg-red-500 px-1.5 text-center text-[11px] font-semibold leading-5 text-white">{{ c.unread }}</span>
                </span>
              </span>
            </button>
          </li>
        </ul>
      </aside>

      <!-- Thread -->
      <section class="min-w-0 flex-1 flex-col md:flex" :class="active ? 'flex' : 'hidden'">
        <template v-if="active">
          <header class="flex items-center gap-3 border-b border-gray-100 px-4 py-3">
            <button type="button" class="rounded-lg px-2 py-1 text-sm text-navy-700 hover:bg-gray-100 md:hidden" :aria-label="L('chat.back')" @click="closeThread">←</button>
            <ChatAvatar :name="active.other.name" :group="!!active.group" :online="!active.group && isOnline(active.other.userId)" />
            <div class="min-w-0">
              <p class="truncate text-sm font-semibold text-gray-900">{{ active.other.name }}</p>
              <p v-if="otherTyping" class="text-xs font-medium text-green-600">{{ active.group && typingName ? `${typingName} ${L("chat.isTyping")}` : L("chat.typing") }}</p>
              <p v-else-if="active.group" class="text-xs text-gray-500">
                {{ active.group.members }} {{ L("chat.members") }}<span v-if="groupOnline" class="text-green-600"> · {{ groupOnline }} {{ L("chat.online").toLowerCase() }}</span>
              </p>
              <p v-else-if="isOnline(active.other.userId)" class="text-xs font-medium text-green-600">{{ L("chat.online") }}</p>
              <p v-else class="text-xs text-gray-500">{{ roleLabel(active.other.role) }}</p>
            </div>
          </header>

          <div ref="scroller" class="min-h-0 flex-1 space-y-1 overflow-y-auto bg-gray-50 px-3 py-3">
            <div v-if="hasOlder" class="text-center">
              <button type="button" class="text-xs font-medium text-navy-700 hover:underline" @click="loadOlder">{{ L("chat.older") }}</button>
            </div>
            <p v-if="loadingThread" class="py-6 text-center text-sm text-gray-400">…</p>
            <template v-for="(m, i) in messages" :key="m.id">
              <p v-if="newDay(i)" class="py-2 text-center text-[11px] font-medium text-gray-400">{{ dayLabel(m.createdAt) }}</p>
              <div class="flex" :class="m.senderId === meId ? 'justify-end' : 'justify-start'">
                <div
                  class="max-w-[80%] rounded-2xl px-3 py-2 text-sm shadow-sm"
                  :class="m.senderId === meId ? 'rounded-br-md bg-navy-600 text-white' : 'rounded-bl-md bg-white text-gray-900'"
                >
                  <p v-if="showSender(i)" class="mb-0.5 text-xs font-semibold text-amber-700">{{ m.senderName || "—" }}</p>
                  <ChatMedia v-if="m.kind !== 'text'" :message="m" :mine="m.senderId === meId" :expired-text="L('chat.expired')" @loaded="nearBottom() && scrollToBottom()" />
                  <p v-if="m.body" class="whitespace-pre-wrap break-words" :class="m.kind !== 'text' ? 'mt-1' : ''">{{ m.body }}</p>
                  <p class="mt-0.5 text-right text-[10px]" :class="m.senderId === meId ? 'text-white/70' : 'text-gray-400'">
                    {{ timeOf(m.createdAt) }}<span v-if="m.id === lastMineSeen"> · {{ L("chat.seen") }}</span>
                  </p>
                </div>
              </div>
            </template>
          </div>

          <footer class="border-t border-gray-200 p-2">
            <div v-if="recording" class="flex items-center gap-2">
              <span class="h-2.5 w-2.5 animate-pulse rounded-full bg-red-500" />
              <span class="flex-1 text-sm text-gray-700">{{ L("chat.recording") }} {{ recordLabel() }}</span>
              <button type="button" class="rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-700" @click="stopRecording(false)">{{ L("chat.cancel") }}</button>
              <button type="button" class="rounded-lg bg-green-600 px-3 py-2 text-sm font-medium text-white" @click="stopRecording(true)">{{ L("chat.stopSend") }}</button>
            </div>
            <form v-else class="flex items-end gap-2" @submit.prevent="submitText">
              <textarea
                v-model="text"
                rows="1"
                :placeholder="L('chat.placeholder')"
                class="max-h-32 min-h-[42px] flex-1 resize-none rounded-xl border border-gray-200 px-3 py-2 text-base focus:border-navy-500 focus:outline-none lg:text-sm"
                @keydown="onKey"
              />
              <button
                v-if="!text.trim()"
                type="button"
                class="field-tap-sm shrink-0 rounded-lg p-2 text-gray-500 hover:bg-gray-100 hover:text-navy-700"
                :title="L('chat.record')"
                :aria-label="L('chat.record')"
                :disabled="sending"
                @click="startRecording"
              >
                <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z" /></svg>
              </button>
              <button
                v-else
                type="submit"
                :disabled="sending"
                @pointerdown.prevent
                class="shrink-0 rounded-xl bg-navy-600 px-4 py-2 text-sm font-medium text-white hover:bg-navy-700 disabled:opacity-50"
              >{{ L("chat.send") }}</button>
            </form>
          </footer>
        </template>
        <div v-else class="flex flex-1 items-center justify-center text-sm text-gray-400">{{ L("chat.pick") }}</div>
      </section>
    </div>

    <!-- New message: pick a person -->
    <div v-if="showPicker" class="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-0 sm:items-center sm:p-4" @click.self="showPicker = false">
      <div class="flex max-h-[80dvh] w-full max-w-md flex-col rounded-t-2xl bg-white shadow-xl sm:rounded-2xl">
        <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3">
          <h3 class="text-base font-semibold text-gray-900">{{ L("chat.newMessage") }}</h3>
          <button type="button" class="rounded-lg px-2 py-1 text-gray-500 hover:bg-gray-100" :aria-label="L('chat.cancel')" @click="showPicker = false">✕</button>
        </div>
        <div class="p-3">
          <input v-model="contactFilter" type="search" :placeholder="L('chat.search')" class="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm" />
        </div>
        <p v-if="contacts.length === 0" class="px-4 pb-6 text-center text-sm text-gray-500">{{ L("chat.noContacts") }}</p>
        <ul class="min-h-0 flex-1 overflow-y-auto pb-3">
          <li v-for="c in filteredContacts" :key="c.userId">
            <button type="button" class="flex w-full items-center gap-3 px-4 py-2.5 text-left hover:bg-gray-50" @click="startWith(c)">
              <ChatAvatar :name="c.name" :online="isOnline(c.userId)" size="sm" />
              <span class="min-w-0 flex-1">
                <span class="block truncate text-sm text-gray-900">{{ c.name }}</span>
                <span v-if="isOnline(c.userId)" class="block text-[11px] text-green-600">{{ L("chat.online") }}</span>
              </span>
              <span class="text-xs text-gray-500">{{ roleLabel(c.role) }}</span>
            </button>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>
