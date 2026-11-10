import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/components/ui/card";
import { formatDateTimeID } from "@/shared/lib/utils";
import type { AuditLog } from "@/shared/types/medicine";
import { fetchAuditLogs } from "../api";

export function AuditLogsPanel() {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchAuditLogs(50)
      .then(setLogs)
      .catch((err: unknown) =>
        setError(err instanceof Error ? err.message : "Gagal memuat log audit"),
      )
      .finally(() => setLoading(false));
  }, []);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Log Aktivitas Admin</CardTitle>
      </CardHeader>
      <CardContent>
        {error && <p className="text-xs text-destructive">{error}</p>}
        {loading ? (
          <p className="text-sm text-muted-foreground">Memuat log...</p>
        ) : logs.length === 0 ? (
          <p className="text-sm text-muted-foreground">Belum ada aktivitas tercatat.</p>
        ) : (
          <div className="divide-y divide-border">
            {logs.map((log) => (
              <div key={log.id} className="flex flex-col gap-0.5 py-2.5 text-xs">
                <div className="flex items-center justify-between gap-2">
                  <span className="font-semibold text-foreground">
                    {log.action} · {log.entity_type}
                  </span>
                  <span className="text-muted-foreground">{formatDateTimeID(log.created_at)}</span>
                </div>
                <span className="text-muted-foreground">{log.detail}</span>
                <span className="text-[11px] text-muted-foreground">oleh {log.admin_email}</span>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
