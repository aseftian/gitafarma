import { apiFetch } from "@/shared/api/client";
import type { Setting } from "@/shared/types/medicine";

export async function fetchSettings(): Promise<Setting[]> {
  const res = await apiFetch<{ items: Setting[] }>("/api/v1/admin/settings");
  return res.items ?? [];
}

export function updateSetting(key: string, value: string): Promise<Setting> {
  return apiFetch<Setting>(`/api/v1/admin/settings/${encodeURIComponent(key)}`, {
    method: "PUT",
    body: JSON.stringify({ value }),
  });
}
