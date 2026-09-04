export type ToastType = "success" | "error" | "info";

export interface ToastPayload {
  message: string;
  type?: ToastType;
}

export function showToast(message: string, type: ToastType = "info") {
  window.dispatchEvent(
    new CustomEvent<ToastPayload>("odysight:toast", {
      detail: { message, type },
    }),
  );
}
