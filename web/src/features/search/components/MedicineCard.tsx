import { Barcode, Clock, Tag } from "lucide-react";
import { Badge } from "@/shared/components/ui/badge";
import { Card, CardContent } from "@/shared/components/ui/card";
import { formatDateID, formatIDR } from "@/shared/lib/utils";
import type { Medicine } from "@/shared/types/medicine";

interface MedicineCardProps {
  medicine: Medicine;
  highlightRecent?: boolean;
}

export function MedicineCard({ medicine, highlightRecent }: MedicineCardProps) {
  return (
    <Card className="transition-colors hover:border-primary/40">
      <CardContent className="flex flex-col gap-2 p-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0 space-y-1.5">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-base font-semibold text-foreground">{medicine.name}</h3>
            {highlightRecent && <Badge variant="success">Update</Badge>}
          </div>

          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
            {medicine.sku && (
              <span className="inline-flex items-center gap-1">
                <Tag className="h-3.5 w-3.5" />
                SKU: {medicine.sku}
              </span>
            )}
            {medicine.barcode && (
              <span className="inline-flex items-center gap-1 font-mono">
                <Barcode className="h-3.5 w-3.5" />
                {medicine.barcode}
              </span>
            )}
            <span className="inline-flex items-center gap-1">
              <Clock className="h-3.5 w-3.5" />
              {formatDateID(medicine.updated_at)}
            </span>
          </div>
        </div>

        <div className="flex shrink-0 items-baseline justify-between gap-2 border-t border-border pt-2 sm:flex-col sm:items-end sm:border-t-0 sm:pt-0">
          <div className="text-lg font-bold text-primary">
            {formatIDR(medicine.price)}
            <span className="ml-1 text-xs font-normal text-muted-foreground">
              / {medicine.uom}
            </span>
          </div>
          {medicine.alt_uom && (
            <Badge variant="secondary" className="text-[11px]">
              Alt: {medicine.alt_uom}
            </Badge>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
