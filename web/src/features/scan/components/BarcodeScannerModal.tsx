import { useState } from "react";
import { ScanBarcode } from "lucide-react";
import { Button } from "@/shared/components/ui/button";
import { Input } from "@/shared/components/ui/input";
import { Modal } from "@/shared/components/ui/modal";
import { useBarcodeScanner } from "../hooks/useBarcodeScanner";

interface BarcodeScannerModalProps {
  open: boolean;
  onClose: () => void;
  onScanSuccess: (barcode: string) => void;
}

export function BarcodeScannerModal({
  open,
  onClose,
  onScanSuccess,
}: BarcodeScannerModalProps) {
  const [manualBarcode, setManualBarcode] = useState("");
  const { videoRef, error, engine } = useBarcodeScanner(open, (code) => {
    onScanSuccess(code);
    onClose();
  });

  const handleManualSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!manualBarcode.trim()) return;
    onScanSuccess(manualBarcode.trim());
    setManualBarcode("");
    onClose();
  };

  return (
    <Modal open={open} onClose={onClose} title="Scan Barcode Obat">
      <div className="space-y-4">
        <div className="relative overflow-hidden rounded-lg bg-slate-900">
          <video
            ref={videoRef}
            playsInline
            muted
            className="aspect-video w-full object-cover"
          />
          <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
            <div className="h-28 w-64 rounded-lg border-2 border-teal-400/90 shadow-[0_0_0_9999px_rgba(0,0,0,0.35)]" />
          </div>
          {engine && (
            <span className="absolute right-2 bottom-2 rounded bg-slate-950/75 px-2 py-0.5 text-[11px] text-slate-200">
              {engine === "native" ? "Scanner Native" : "Scanner ZXing"}
            </span>
          )}
        </div>

        {error && (
          <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800">
            {error}
          </div>
        )}

        <form onSubmit={handleManualSubmit} className="space-y-2 border-t border-border pt-3">
          <p className="text-xs text-muted-foreground">
            Atau masukkan angka barcode secara manual:
          </p>
          <div className="flex gap-2">
            <Input
              value={manualBarcode}
              onChange={(e) => setManualBarcode(e.target.value)}
              placeholder="Contoh: 8991234000001"
              inputMode="numeric"
            />
            <Button type="submit" variant="secondary">
              <ScanBarcode className="h-4 w-4" />
              Cari
            </Button>
          </div>
        </form>
      </div>
    </Modal>
  );
}
