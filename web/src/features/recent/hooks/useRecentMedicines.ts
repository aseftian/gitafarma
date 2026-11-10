import { useCallback, useEffect, useState } from "react";
import type { Medicine } from "@/shared/types/medicine";
import { fetchRecentMedicines } from "../api";

export function useRecentMedicines() {
  const [items, setItems] = useState<Medicine[]>([]);
  const [total, setTotal] = useState(0);
  const [recentDays, setRecentDays] = useState(7);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetchRecentMedicines(1, 20);
      setItems(res.items ?? []);
      setTotal(res.total ?? 0);
      if (res.recent_days > 0) {
        setRecentDays(res.recent_days);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Gagal memuat daftar harga terbaru");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return {
    items,
    total,
    recentDays,
    loading,
    error,
    refresh: load,
  };
}
