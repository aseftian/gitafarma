import { apiFetch } from "@/shared/api/client";
import type { AuditLog, Medicine, SyncResult } from "@/shared/types/medicine";

export interface MedicinePayload {
  name: string;
  barcode: string;
  sku: string;
  price: number;
  uom: string;
  alt_uom: string;
}

export function createMedicine(payload: MedicinePayload): Promise<Medicine> {
  return apiFetch<Medicine>("/api/v1/admin/medicines", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function updateMedicine(id: string, payload: MedicinePayload): Promise<Medicine> {
  return apiFetch<Medicine>(`/api/v1/admin/medicines/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export function deleteMedicine(id: string): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/medicines/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

export function triggerManualSync(): Promise<SyncResult> {
  return apiFetch<SyncResult>("/api/v1/admin/sync", {
    method: "POST",
  });
}

export async function fetchAuditLogs(limit = 50): Promise<AuditLog[]> {
  const res = await apiFetch<{ items: AuditLog[] }>(`/api/v1/admin/audit-logs?limit=${limit}`);
  return res.items ?? [];
}
