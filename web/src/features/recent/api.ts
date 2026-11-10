import { apiFetch } from "@/shared/api/client";
import type { RecentMedicinesResponse } from "@/shared/types/medicine";

export function fetchRecentMedicines(
  page = 1,
  limit = 20,
): Promise<RecentMedicinesResponse> {
  return apiFetch<RecentMedicinesResponse>(
    `/api/v1/medicines/recent?page=${page}&limit=${limit}`,
  );
}
