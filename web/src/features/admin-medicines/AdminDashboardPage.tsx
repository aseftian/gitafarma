import { useState } from "react";
import {
  ChevronLeft,
  ChevronRight,
  LogOut,
  Pencil,
  Plus,
  RefreshCw,
  Search,
  Trash2,
} from "lucide-react";
import { useAuth } from "@/features/auth/auth-context";
import { SettingsPanel } from "@/features/settings/components/SettingsPanel";
import { Badge } from "@/shared/components/ui/badge";
import { Button } from "@/shared/components/ui/button";
import { Card, CardContent } from "@/shared/components/ui/card";
import { Input } from "@/shared/components/ui/input";
import { formatDateID, formatIDR } from "@/shared/lib/utils";
import type { Medicine } from "@/shared/types/medicine";
import { AuditLogsPanel } from "./components/AuditLogsPanel";
import { MedicineFormModal } from "./components/MedicineFormModal";
import { useAdminMedicines } from "./hooks/useAdminMedicines";

type AdminTab = "medicines" | "settings" | "audit";

export function AdminDashboardPage() {
  const { admin, logout } = useAuth();
  const [tab, setTab] = useState<AdminTab>("medicines");
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Medicine | null>(null);

  const {
    query,
    setQuery,
    page,
    setPage,
    totalPages,
    items,
    total,
    loading,
    syncing,
    error,
    notice,
    saveMedicine,
    removeMedicine,
    syncNow,
  } = useAdminMedicines();

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 rounded-lg border border-border bg-card p-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-lg font-semibold text-foreground">Panel Pengelola</h1>
          <p className="text-xs text-muted-foreground">
            Masuk sebagai <span className="font-medium text-foreground">{admin?.email}</span>{" "}
            <Badge variant="outline" className="ml-1">
              {admin?.role ?? "admin"}
            </Badge>
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={syncing}
            onClick={() => void syncNow()}
          >
            <RefreshCw className={`h-4 w-4 ${syncing ? "animate-spin" : ""}`} />
            Sync GSheet
          </Button>
          <Button type="button" variant="ghost" size="sm" onClick={() => void logout()}>
            <LogOut className="h-4 w-4" />
            Keluar
          </Button>
        </div>
      </div>

      <div className="flex gap-2 border-b border-border pb-2">
        {(
          [
            ["medicines", "Daftar Obat"],
            ["settings", "Pengaturan"],
            ["audit", "Log Audit"],
          ] as const
        ).map(([key, label]) => (
          <Button
            key={key}
            type="button"
            size="sm"
            variant={tab === key ? "default" : "ghost"}
            onClick={() => setTab(key)}
          >
            {label}
          </Button>
        ))}
      </div>

      {tab === "medicines" && (
        <div className="space-y-4">
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

          <div className="flex flex-col gap-2 sm:flex-row">
            <div className="relative flex-1">
              <Search className="pointer-events-none absolute top-3 left-3 h-4 w-4 text-muted-foreground" />
              <Input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Filter obat berdasarkan nama, SKU, atau barcode..."
                className="pl-9"
              />
            </div>
            <Button
              type="button"
              onClick={() => {
                setEditing(null);
                setModalOpen(true);
              }}
            >
              <Plus className="h-4 w-4" />
              Tambah Obat
            </Button>
          </div>

          <div className="text-xs text-muted-foreground">
            {loading ? "Memuat..." : `${total} obat tercatat`}
          </div>

          <div className="space-y-2">
            {items.map((med) => (
              <Card key={med.id}>
                <CardContent className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
                  <div className="space-y-1">
                    <p className="font-semibold text-foreground">{med.name}</p>
                    <div className="flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
                      <span>
                        {formatIDR(med.price)} / {med.uom}
                        {med.alt_uom ? ` (Alt: ${med.alt_uom})` : ""}
                      </span>
                      {med.sku && <span>SKU: {med.sku}</span>}
                      {med.barcode && <span className="font-mono">{med.barcode}</span>}
                      <span>Update: {formatDateID(med.updated_at)}</span>
                    </div>
                  </div>
                  <div className="flex shrink-0 gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setEditing(med);
                        setModalOpen(true);
                      }}
                    >
                      <Pencil className="h-3.5 w-3.5" />
                      Edit
                    </Button>
                    <Button
                      type="button"
                      variant="destructive"
                      size="sm"
                      onClick={() => {
                        if (window.confirm(`Hapus "${med.name}" dari Google Sheet?`)) {
                          void removeMedicine(med);
                        }
                      }}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                      Hapus
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-between pt-1">
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
              >
                <ChevronLeft className="h-4 w-4" />
                Sebelumnya
              </Button>
              <span className="text-xs text-muted-foreground">
                Halaman {page} / {totalPages}
              </span>
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={page >= totalPages}
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              >
                Berikutnya
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          )}

          <MedicineFormModal
            open={modalOpen}
            initial={editing}
            onClose={() => setModalOpen(false)}
            onSubmit={saveMedicine}
          />
        </div>
      )}

      {tab === "settings" && <SettingsPanel />}
      {tab === "audit" && <AuditLogsPanel />}
    </div>
  );
}
