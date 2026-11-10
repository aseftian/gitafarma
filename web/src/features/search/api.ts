import { apiFetch } from "@/shared/api/client";
import type { MedicinePage } from "@/shared/types/medicine";

export interface SearchMedicinesParams {
  q?: string;
  barcode?: string;
  sku?: string;
  page?: number;
  limit?: number;
}

export function searchMedicines(
  params: SearchMedicinesParams,
  signal?: AbortSignal,
): Promise<MedicinePage> {
  const sp = new URLSearchParams();
  if (params.q) sp.set("q", params.q);
  if (params.barcode) sp.set("barcode", params.barcode);
  if (params.sku) sp.set("sku", params.sku);
  if (params.page) sp.set("page", String(params.page));
  if (params.limit) sp.set("limit", String(params.limit));

  return apiFetch<MedicinePage>(`/api/v1/medicines?${sp.toString()}`, { signal });
}
