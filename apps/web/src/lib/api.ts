const API_URL: string | undefined = import.meta.env.PUBLIC_API_URL;

export const USE_MOCKS = !API_URL && import.meta.env.DEV;

if (!API_URL && import.meta.env.PROD) {
  console.error(
    "PUBLIC_API_URL is not set. The application requires a configured API in production.",
  );
}

export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

export async function apiFetch<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  if (!API_URL) {
    throw new ApiError(0, "PUBLIC_API_URL is not configured");
  }

  const response = await fetch(`${API_URL}${path}`, {
    headers: { "Content-Type": "application/json", ...init?.headers },
    ...init,
  });

  if (!response.ok) {
    throw new ApiError(
      response.status,
      `Request failed: ${response.statusText}`,
    );
  }

  return response.json() as Promise<T>;
}

export function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
