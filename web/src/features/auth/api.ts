import { apiFetch } from "@/shared/api/client";
import type { AuthSessionResponse } from "@/shared/types/medicine";

export function loginWithGoogle(idToken: string): Promise<AuthSessionResponse> {
  return apiFetch<AuthSessionResponse>("/api/v1/auth/google", {
    method: "POST",
    body: JSON.stringify({ id_token: idToken }),
  });
}

export function logoutAdmin(): Promise<void> {
  return apiFetch<void>("/api/v1/auth/logout", {
    method: "POST",
  });
}

export function fetchCurrentAdmin(): Promise<AuthSessionResponse> {
  return apiFetch<AuthSessionResponse>("/api/v1/auth/me");
}
