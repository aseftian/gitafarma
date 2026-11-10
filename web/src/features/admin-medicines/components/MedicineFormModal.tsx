import { useEffect, useState } from "react";
import { Button } from "@/shared/components/ui/button";
import { Input } from "@/shared/components/ui/input";
import { Label } from "@/shared/components/ui/label";
import { Modal } from "@/shared/components/ui/modal";
import type { Medicine } from "@/shared/types/medicine";
import type { MedicinePayload } from "../api";

interface MedicineFormModalProps {
  open: boolean;
  initial: Medicine | null;
  onClose: () => void;
  onSubmit: (id: string | null, payload: MedicinePayload) => Promise<void>;
}

export function MedicineFormModal({
  open,
  initial,
  onClose,
  onSubmit,
}: MedicineFormModalProps) {
  const [name, setName] = useState("");
  const [barcode, setBarcode] = useState("");
  const [sku, setSku] = useState("");
  const [price, setPrice] = useState("");
  const [uom, setUom] = useState("Strip");
  const [altUom, setAltUom] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setError(null);
    if (initial) {
      setName(initial.name);
      setBarcode(initial.barcode);
      setSku(initial.sku);
      setPrice(String(initial.price));
      setUom(initial.uom);
      setAltUom(initial.alt_uom);
    } else {
      setName("");
      setBarcode("");
      setSku("");
      setPrice("");
      setUom("Strip");
      setAltUom("");
    }
  }, [open, initial]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const parsedPrice = Number(price);
    if (!name.trim() || !uom.trim() || Number.isNaN(parsedPrice) || parsedPrice < 0) {
      setError("Nama obat, UOM, dan harga yang valid wajib diisi.");
      return;
    }

    setSaving(true);
    setError(null);
    try {
      await onSubmit(initial?.id ?? null, {
        name: name.trim(),
        barcode: barcode.trim(),
        sku: sku.trim(),
        price: Math.round(parsedPrice),
        uom: uom.trim(),
        alt_uom: altUom.trim(),
      });
      onClose();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Gagal menyimpan data obat");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal open={open} onClose={onClose} title={initial ? "Edit Obat" : "Tambah Obat Baru"}>
      <form onSubmit={(e) => void handleSubmit(e)} className="space-y-3">
        {error && (
          <div className="rounded-md border border-red-200 bg-red-50 p-2.5 text-xs text-red-700">
            {error}
          </div>
        )}

        <div className="space-y-1">
          <Label htmlFor="med-name">Nama Obat *</Label>
          <Input
            id="med-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Paracetamol 500mg Tablet"
            required
          />
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <Label htmlFor="med-sku">SKU</Label>
            <Input
              id="med-sku"
              value={sku}
              onChange={(e) => setSku(e.target.value)}
              placeholder="GF-001"
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="med-barcode">Barcode</Label>
            <Input
              id="med-barcode"
              value={barcode}
              onChange={(e) => setBarcode(e.target.value)}
              placeholder="8991234000001"
            />
          </div>
        </div>

        <div className="space-y-1">
          <Label htmlFor="med-price">Harga (IDR) *</Label>
          <Input
            id="med-price"
            type="number"
            min={0}
            value={price}
            onChange={(e) => setPrice(e.target.value)}
            placeholder="5000"
            required
          />
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <Label htmlFor="med-uom">Satuan Utama (UOM) *</Label>
            <Input
              id="med-uom"
              value={uom}
              onChange={(e) => setUom(e.target.value)}
              placeholder="Strip / Botol / Box"
              required
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="med-alt-uom">Satuan Alternatif</Label>
            <Input
              id="med-alt-uom"
              value={altUom}
              onChange={(e) => setAltUom(e.target.value)}
              placeholder="Tablet / Kapsul"
            />
          </div>
        </div>

        <div className="flex justify-end gap-2 pt-2">
          <Button type="button" variant="outline" onClick={onClose} disabled={saving}>
            Batal
          </Button>
          <Button type="submit" disabled={saving}>
            {saving ? "Menyimpan..." : "Simpan ke GSheet"}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
