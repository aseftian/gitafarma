import { useEffect, useState } from "react";
import { Save } from "lucide-react";
import { Button } from "@/shared/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/components/ui/card";
import { Input } from "@/shared/components/ui/input";
import { Label } from "@/shared/components/ui/label";
import { formatDateTimeID } from "@/shared/lib/utils";
import { useSettings } from "../hooks/useSettings";

export function SettingsPanel() {
  const { settings, loading, savingKey, error, notice, save } = useSettings();
  const [drafts, setDrafts] = useState<Record<string, string>>({});

  useEffect(() => {
    const next: Record<string, string> = {};
    for (const s of settings) {
      next[s.key] = s.value;
    }
    setDrafts(next);
  }, [settings]);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Pengaturan Aplikasi</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {error && (
          <div className="rounded-md border border-red-200 bg-red-50 p-3 text-xs text-red-700">
            {error}
          </div>
        )}
        {notice && (
          <div className="rounded-md border border-emerald-200 bg-emerald-50 p-3 text-xs text-emerald-700">
            {notice}
          </div>
        )}

        {loading ? (
          <p className="text-sm text-muted-foreground">Memuat pengaturan...</p>
        ) : (
          <div className="space-y-4">
            {settings.map((s) => (
              <form
                key={s.key}
                onSubmit={(e) => {
                  e.preventDefault();
                  void save(s.key, drafts[s.key] ?? "");
                }}
                className="space-y-2 rounded-lg border border-border p-3.5"
              >
                <div className="flex items-center justify-between gap-2">
                  <Label htmlFor={`setting-${s.key}`} className="font-mono text-xs uppercase">
                    {s.key}
                  </Label>
                  <span className="text-[11px] text-muted-foreground">
                    Diperbarui: {formatDateTimeID(s.updated_at)}
                  </span>
                </div>
                {s.description && (
                  <p className="text-xs text-muted-foreground">{s.description}</p>
                )}
                <div className="flex gap-2">
                  <Input
                    id={`setting-${s.key}`}
                    value={drafts[s.key] ?? ""}
                    onChange={(e) =>
                      setDrafts((prev) => ({ ...prev, [s.key]: e.target.value }))
                    }
                  />
                  <Button type="submit" size="sm" disabled={savingKey === s.key}>
                    <Save className="h-4 w-4" />
                    Simpan
                  </Button>
                </div>
              </form>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
