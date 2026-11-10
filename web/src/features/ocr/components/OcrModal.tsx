import { useEffect, useRef, useState } from "react";
import { Camera, ImagePlus, Search } from "lucide-react";
import { Button } from "@/shared/components/ui/button";
import { Input } from "@/shared/components/ui/input";
import { Label } from "@/shared/components/ui/label";
import { Modal } from "@/shared/components/ui/modal";
import { useOcr } from "../hooks/useOcr";

interface OcrModalProps {
  open: boolean;
  onClose: () => void;
  onSearchText: (text: string) => void;
}

export function OcrModal({ open, onClose, onSearchText }: OcrModalProps) {
  const cameraInputRef = useRef<HTMLInputElement | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);

  const {
    processing,
    progress,
    editedText,
    setEditedText,
    candidates,
    error,
    recognizeFile,
    reset,
  } = useOcr();

  useEffect(() => {
    return () => {
      if (previewUrl) URL.revokeObjectURL(previewUrl);
    };
  }, [previewUrl]);

  const handleFile = (file?: File) => {
    if (!file) return;
    if (previewUrl) URL.revokeObjectURL(previewUrl);
    setPreviewUrl(URL.createObjectURL(file));
    void recognizeFile(file);
  };

  const handleClose = () => {
    reset();
    if (previewUrl) {
      URL.revokeObjectURL(previewUrl);
      setPreviewUrl(null);
    }
    onClose();
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!editedText.trim()) return;
    onSearchText(editedText.trim());
    handleClose();
  };

  return (
    <Modal open={open} onClose={handleClose} title="Baca Nama Obat dari Foto (OCR)">
      <div className="space-y-4">
        <p className="text-xs text-muted-foreground">
          Ambil foto kemasan obat yang terlihat jelas nama dan dosisnya, lalu periksa teks sebelum mencari.
        </p>

        <div className="grid grid-cols-2 gap-2">
          <Button
            type="button"
            variant="outline"
            onClick={() => cameraInputRef.current?.click()}
            disabled={processing}
          >
            <Camera className="h-4 w-4" />
            Ambil Foto
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={() => fileInputRef.current?.click()}
            disabled={processing}
          >
            <ImagePlus className="h-4 w-4" />
            Pilih Gambar
          </Button>
        </div>

        <input
          ref={cameraInputRef}
          type="file"
          accept="image/*"
          capture="environment"
          className="hidden"
          onChange={(e) => handleFile(e.target.files?.[0])}
        />
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          className="hidden"
          onChange={(e) => handleFile(e.target.files?.[0])}
        />

        {previewUrl && (
          <img
            src={previewUrl}
            alt="Preview kemasan obat"
            className="max-h-48 w-full rounded-lg border border-border object-contain bg-slate-100"
          />
        )}

        {processing && (
          <div className="space-y-1.5">
            <div className="flex justify-between text-xs text-muted-foreground">
              <span>Membaca teks kemasan...</span>
              <span>{progress}%</span>
            </div>
            <div className="h-2 w-full overflow-hidden rounded-full bg-secondary">
              <div
                className="h-full bg-primary transition-all"
                style={{ width: `${progress}%` }}
              />
            </div>
          </div>
        )}

        {error && (
          <div className="rounded-md border border-red-200 bg-red-50 p-3 text-xs text-red-700">
            {error}
          </div>
        )}

        {candidates.length > 0 && (
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground">Baris terdeteksi (ketuk untuk pilih):</Label>
            <div className="flex flex-wrap gap-1.5">
              {candidates.map((cand) => (
                <button
                  key={cand}
                  type="button"
                  onClick={() => setEditedText(cand)}
                  className="rounded-full border border-border bg-secondary px-2.5 py-1 text-xs font-medium text-foreground hover:border-primary"
                >
                  {cand}
                </button>
              ))}
            </div>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-3 border-t border-border pt-3">
          <div className="space-y-1.5">
            <Label htmlFor="ocr-query-input">Teks pencarian (dapat diedit)</Label>
            <Input
              id="ocr-query-input"
              value={editedText}
              onChange={(e) => setEditedText(e.target.value)}
              placeholder="Hasil baca OCR akan muncul di sini..."
            />
          </div>
          <Button type="submit" className="w-full" disabled={processing || !editedText.trim()}>
            <Search className="h-4 w-4" />
            Cari Obat Ini
          </Button>
        </form>
      </div>
    </Modal>
  );
}
