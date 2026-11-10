const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "";

let memoryCsrfToken = "";

export function setCsrfToken(token: string) {
  memoryCsrfToken = token;
}

function getCsrfToken(): string {
  if (memoryCsrfToken) return memoryCsrfToken;
  if (typeof document === "undefined") return "";
  const match = document.cookie
    .split("; ")
    .find((row) => row.startsWith("gf_csrf="));
  return match ? decodeURIComponent(match.split("=").slice(1).join("=")) : "";
}

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const method = (init?.method ?? "GET").toUpperCase();
  const isMutating = ["POST", "PUT", "PATCH", "DELETE"].includes(method);
  const csrf = isMutating ? getCsrfToken() : "";

  const res = await fetch(`${API_BASE_URL}${path}`, {
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(csrf ? { "X-CSRF-Token": csrf } : {}),
      ...(init?.headers ?? {}),
    },
    ...init,
  });

  if (!res.ok) {
    let message = res.statusText;
    try {
      const text = await res.text();
      if (text) {
        const parsed = JSON.parse(text) as { error?: string };
        message = parsed.error || text;
      }
    } catch {
      // keep fallback statusText
    }
    throw new ApiError(res.status, message);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}
