import { onMounted, onUnmounted, ref, type Ref } from "vue";

const FOCUSABLE_SELECTOR = [
  "a[href]",
  "button:not([disabled])",
  "input:not([disabled])",
  "select:not([disabled])",
  "textarea:not([disabled])",
  '[tabindex]:not([tabindex="-1"])',
].join(", ");

export function useModalA11y(cancel: () => void): {
  container: Ref<HTMLElement | null>;
} {
  const container = ref<HTMLElement | null>(null);
  let previouslyFocused: HTMLElement | null = null;

  function getFocusable(): HTMLElement[] {
    if (!container.value) return [];
    return Array.from(
      container.value.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR),
    ).filter((el) => el.offsetParent !== null || el === document.activeElement);
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key === "Escape") {
      event.preventDefault();
      cancel();
      return;
    }

    if (event.key !== "Tab") return;

    const focusable = getFocusable();
    if (focusable.length === 0) return;

    const first = focusable[0]!;
    const last = focusable[focusable.length - 1]!;
    const active = document.activeElement as HTMLElement | null;

    if (event.shiftKey && active === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && active === last) {
      event.preventDefault();
      first.focus();
    }
  }

  onMounted(() => {
    previouslyFocused = document.activeElement as HTMLElement | null;
    document.addEventListener("keydown", onKeydown, true);

    const firstInput = container.value?.querySelector<HTMLElement>(
      "input, select, textarea",
    );
    const target =
      firstInput ??
      container.value?.querySelector<HTMLElement>(FOCUSABLE_SELECTOR) ??
      null;
    target?.focus();
  });

  onUnmounted(() => {
    document.removeEventListener("keydown", onKeydown, true);
    previouslyFocused?.focus();
  });

  return { container };
}
