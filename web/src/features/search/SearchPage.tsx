import { useState } from "react";
import { Camera, ChevronLeft, ChevronRight, ScanBarcode, Search, X } from "lucide-react";
import { OcrModal } from "@/features/ocr/components/OcrModal";
import { PushNotificationBanner } from "@/features/push/components/PushNotificationBanner";
import { RecentUpdatesSection } from "@/features/recent/components/RecentUpdatesSection";
import { BarcodeScannerModal } from "@/features/scan/components/BarcodeScannerModal";
import { Badge } from "@/shared/components/ui/badge";
import { Button } from "@/shared/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/components/ui/card";
import { Input } from "@/shared/components/ui/input";
import { MedicineCard } from "./components/MedicineCard";
import { useMedicineSearch } from "./hooks/useMedicineSearch";

export function SearchPage() {
  const [scanOpen, setScanOpen] = useState(false);
  const [ocrOpen, setOcrOpen] = useState(false);

  const {
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
  } = useMedicineSearch();

  return (
    <div className="space-y-5">
      <PushNotificationBanner />

      <Card>
        <CardHeader className="pb-3">
          <CardTitle>Cari Harga Obat</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex gap-2">
            <div className="relative flex-1">
              <Search className="pointer-events-none absolute top-3 left-3 h-4 w-4 text-muted-foreground" />
              <Input
                value={query}
                onChange={(e) => updateQuery(e.target.value)}
                placeholder="Ketik nama obat, SKU, atau barcode..."
                className="pr-9 pl-9"
                aria-label="Cari obat"
              />
              {query && (
                <button
                  type="button"
                  onClick={clear}
                  aria-label="Bersihkan pencarian"
                  className="absolute top-2.5 right-2.5 rounded p-0.5 text-muted-foreground hover:text-foreground"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>

            <Button
              type="button"
              variant="outline"
              onClick={() => setScanOpen(true)}
              title="Scan Barcode"
              className="shrink-0"
            >
              <ScanBarcode className="h-4 w-4" />
              <span className="hidden sm:inline">Barcode</span>
            </Button>

            <Button
              type="button"
              variant="outline"
              onClick={() => setOcrOpen(true)}
              title="Foto Nama Obat (OCR)"
              className="shrink-0"
            >
              <Camera className="h-4 w-4" />
              <span className="hidden sm:inline">OCR</span>
            </Button>
          </div>

          {barcodeFilter && (
            <div className="flex items-center gap-2 text-xs">
              <Badge variant="outline">Filter Barcode: {barcodeFilter}</Badge>
              <button
                type="button"
                onClick={clear}
                className="text-primary hover:underline"
              >
                Reset
              </button>
            </div>
          )}
        </CardContent>
      </Card>

      {isActive ? (
        <section className="space-y-3">
          <div className="flex items-center justify-between text-sm">
            <span className="font-medium text-foreground">
              {loading ? "Mencari..." : `${total} hasil ditemukan`}
            </span>
            {totalPages > 1 && (
              <span className="text-xs text-muted-foreground">
                Halaman {page} dari {totalPages}
              </span>
            )}
          </div>

          {error && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
              {error}
            </div>
          )}

          {!loading && !error && items.length === 0 && (
            <div className="rounded-lg border border-dashed border-border bg-card p-6 text-center text-sm text-muted-foreground">
              Obat tidak ditemukan. Coba kata kunci lebih singkat atau periksa ejaan.
            </div>
          )}

          <div className="space-y-2.5">
            {items.map((med) => (
              <MedicineCard key={med.id} medicine={med} />
            ))}
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-between pt-2">
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
        </section>
      ) : (
        <RecentUpdatesSection />
      )}

      <BarcodeScannerModal
        open={scanOpen}
        onClose={() => setScanOpen(false)}
        onScanSuccess={applyBarcode}
      />

      <OcrModal
        open={ocrOpen}
        onClose={() => setOcrOpen(false)}
        onSearchText={applyOcrText}
      />
    </div>
  );
}
