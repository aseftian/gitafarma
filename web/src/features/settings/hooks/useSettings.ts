import { useCallback, useEffect, useState } from "react";
import type { Setting } from "@/shared/types/medicine";
import { fetchSettings, updateSetting } from "../api";

export function useSettings() {
  const [settings, setSettings] = useState<Setting[]>([]);
  const [loading, setLoading] = useState(true);
  const [savingKey, setSavingKey] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const items = await fetchSettings();
      setSettings(items);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Gagal memuat pengaturan");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const save = useCallback(async (key: string, value: string) => {
    setSavingKey(key);
    setError(null);
    setNotice(null);
    try {
      const updated = await updateSetting(key, value);
      setSettings((prev) => prev.map((s) => (s.key === key ? updated : s)));
      setNotice(`Pengaturan "${key}" berhasil disimpan.`);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Gagal menyimpan pengaturan");
    } finally {
      setSavingKey(null);
    }
  }, []);

  return { settings, loading, savingKey, error, notice, save, reload: load };
}
