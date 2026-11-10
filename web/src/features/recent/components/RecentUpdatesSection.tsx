import { RefreshCw, Sparkles } from "lucide-react";
import { MedicineCard } from "@/features/search/components/MedicineCard";
import { Button } from "@/shared/components/ui/button";
import { useRecentMedicines } from "../hooks/useRecentMedicines";

export function RecentUpdatesSection() {
  const { items, total, recentDays, loading, error, refresh } = useRecentMedicines();

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Sparkles className="h-4 w-4 text-primary" />
          <h2 className="text-sm font-semibold text-foreground">
            Update Harga Terbaru ({recentDays} Hari Terakhir)
          </h2>
          {!loading && total > 0 && (
            <span className="text-xs text-muted-foreground">({total})</span>
          )}
        </div>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => void refresh()}
          disabled={loading}
          title="Muat ulang"
        >
          <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
          <span className="sr-only sm:not-sr-only">Segarkan</span>
        </Button>
      </div>

      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          {error}
        </div>
      )}

      {loading && items.length === 0 && (
        <div className="space-y-2">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-20 animate-pulse rounded-lg border border-border bg-card" />
          ))}
        </div>
      )}

      {!loading && !error && items.length === 0 && (
        <div className="rounded-lg border border-dashed border-border bg-card p-6 text-center text-sm text-muted-foreground">
          Belum ada perubahan harga obat dalam {recentDays} hari terakhir.
        </div>
      )}

      <div className="space-y-2.5">
        {items.map((med) => (
          <MedicineCard key={med.id} medicine={med} highlightRecent />
        ))}
      </div>
    </section>
  );
}
