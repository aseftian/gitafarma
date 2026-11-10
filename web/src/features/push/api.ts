import { apiFetch } from "@/shared/api/client";

export interface VapidKeyResponse {
  enabled: boolean;
  public_key: string;
}

export function fetchVapidPublicKey(): Promise<VapidKeyResponse> {
  return apiFetch<VapidKeyResponse>("/api/v1/push/vapid-public-key");
}

export function subscribePush(subscription: PushSubscriptionJSON): Promise<void> {
  return apiFetch<void>("/api/v1/push/subscribe", {
    method: "POST",
    body: JSON.stringify(subscription),
  });
}

export function unsubscribePush(endpoint: string): Promise<void> {
  return apiFetch<void>("/api/v1/push/unsubscribe", {
    method: "POST",
    body: JSON.stringify({ endpoint }),
  });
}
