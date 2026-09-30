import { onBeforeUnmount, onMounted, ref } from "vue";
import { getPresence } from "../../../lib/chat";
import { onChatEvent } from "../../../lib/chatEvents";

// Who is online (has the app open). Loaded once, kept current by "presence"
// events, and re-synced after a reconnect and every minute as a safety net.
export function usePresence() {
  const online = ref<Set<number>>(new Set());
  let timer: ReturnType<typeof setInterval> | undefined;
  let off: (() => void) | undefined;

  async function refresh() {
    try {
      online.value = new Set(await getPresence());
    } catch {
      /* keep the last known state */
    }
  }

  onMounted(() => {
    void refresh();
    timer = setInterval(() => {
      if (document.visibilityState === "visible") void refresh();
    }, 60_000);
    off = onChatEvent((type, data) => {
      if (type === "open") void refresh();
      if (type === "presence" && data.userId) {
        const next = new Set(online.value);
        if (data.online) next.add(data.userId);
        else next.delete(data.userId);
        online.value = next;
      }
    });
  });
  onBeforeUnmount(() => {
    clearInterval(timer);
    off?.();
  });

  return { online, isOnline: (id: number) => online.value.has(id) };
}
