import { useCallback, useEffect, useState } from "react";
import { useDebouncedValue } from "@/shared/hooks/useDebouncedValue";
import type { Medicine } from "@/shared/types/medicine";
import { searchMedicines } from "../api";

const PAGE_LIMIT = 20;

export function useMedicineSearch() {
  const [query, setQuery] = useState("");
  const [barcodeFilter, setBarcodeFilter] = useState("");
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<Medicine[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const debouncedQuery = useDebouncedValue(query.trim(), 280);
  const isActive = debouncedQuery.length > 0 || barcodeFilter.length > 0;

  useEffect(() => {
    if (!isActive) {
      setItems([]);
      setTotal(0);
      setError(null);
      return;
    }

    const controller = new AbortController();
    setLoading(true);
    setError(null);

    searchMedicines(
      {
        q: barcodeFilter ? undefined : debouncedQuery,
        barcode: barcodeFilter || undefined,
        page,
        limit: PAGE_LIMIT,
      },
      controller.signal,
    )
      .then((res) => {
        setItems(res.items ?? []);
        setTotal(res.total ?? 0);
      })
      .catch((err: unknown) => {
        if (err instanceof DOMException && err.name === "AbortError") return;
        setError(err instanceof Error ? err.message : "Gagal memuat data obat");
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
      });

    return () => controller.abort();
  }, [isActive, debouncedQuery, barcodeFilter, page]);

  const updateQuery = useCallback((next: string) => {
    setBarcodeFilter("");
    setQuery(next);
    setPage(1);
  }, []);

  const applyBarcode = useCallback((barcode: string) => {
    const clean = barcode.trim();
    setQuery(clean);
    setBarcodeFilter(clean);
    setPage(1);
  }, []);

  const applyOcrText = useCallback((text: string) => {
    setBarcodeFilter("");
    setQuery(text.trim());
    setPage(1);
  }, []);

  const clear = useCallback(() => {
    setBarcodeFilter("");
    setQuery("");
    setPage(1);
  }, []);

  const totalPages = Math.max(1, Math.ceil(total / PAGE_LIMIT));

  return {
    query,
    barcodeFilter,
    isActive,
    items,
    total,
    page,
    totalPages,
    loading,
    error,
    setPage,
    updateQuery,
    applyBarcode,
    applyOcrText,
    clear,
  };
}
