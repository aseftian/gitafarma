import { useCallback, useEffect, useState } from "react";
import { searchMedicines } from "@/features/search/api";
import { useDebouncedValue } from "@/shared/hooks/useDebouncedValue";
import type { Medicine, SyncResult } from "@/shared/types/medicine";
import {
  createMedicine,
  deleteMedicine,
  triggerManualSync,
  updateMedicine,
  type MedicinePayload,
} from "../api";

export function useAdminMedicines() {
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<Medicine[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [syncing, setSyncing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const debouncedQuery = useDebouncedValue(query.trim(), 250);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await searchMedicines({ q: debouncedQuery || undefined, page, limit: 20 });
      setItems(res.items ?? []);
      setTotal(res.total ?? 0);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Gagal memuat daftar obat");
    } finally {
      setLoading(false);
    }
  }, [debouncedQuery, page]);

  useEffect(() => {
    void load();
  }, [load]);

  const saveMedicine = useCallback(
    async (id: string | null, payload: MedicinePayload) => {
      setError(null);
      setNotice(null);
      if (id) {
        await updateMedicine(id, payload);
        setNotice(`Obat "${payload.name}" berhasil diperbarui ke Google Sheet.`);
      } else {
        await createMedicine(payload);
        setNotice(`Obat "${payload.name}" berhasil ditambahkan ke Google Sheet.`);
      }
      await load();
    },
    [load],
  );

  const removeMedicine = useCallback(
    async (med: Medicine) => {
      setError(null);
      setNotice(null);
      try {
        await deleteMedicine(med.id);
        setNotice(`Obat "${med.name}" dihapus dari Google Sheet dan cache.`);
        await load();
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : "Gagal menghapus obat");
      }
    },
    [load],
  );

  const syncNow = useCallback(async (): Promise<SyncResult | null> => {
    setSyncing(true);
    setError(null);
    setNotice(null);
    try {
      const res = await triggerManualSync();
      setNotice(
        `Sinkronisasi selesai: ${res.total_fetched} data diambil, ${res.upserted} diperbarui, ${res.deleted} dihapus.`,
      );
      await load();
      return res;
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Sinkronisasi Google Sheet gagal");
      return null;
    } finally {
      setSyncing(false);
    }
  }, [load]);

  return {
    query,
    setQuery: (next: string) => {
      setQuery(next);
      setPage(1);
    },
    page,
    setPage,
    totalPages: Math.max(1, Math.ceil(total / 20)),
    items,
    total,
    loading,
    syncing,
    error,
    notice,
    saveMedicine,
    removeMedicine,
    syncNow,
  };
}
