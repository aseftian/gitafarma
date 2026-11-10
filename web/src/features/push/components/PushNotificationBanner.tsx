import { Bell, BellOff } from "lucide-react";
import { Button } from "@/shared/components/ui/button";
import { usePushSubscription } from "../hooks/usePushSubscription";

export function PushNotificationBanner() {
  const {
    supported,
    enabledOnServer,
    subscribed,
    busy,
    error,
    subscribe,
    unsubscribe,
  } = usePushSubscription();

  if (!supported || !enabledOnServer) {
    return null;
  }

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-border bg-card px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
      <div className="space-y-0.5">
        <p className="text-sm font-medium text-foreground">
          {subscribed ? "Notifikasi Update Harga Aktif" : "Aktifkan Notifikasi Perubahan Harga"}
        </p>
        <p className="text-xs text-muted-foreground">
          {subscribed
            ? "Perangkat ini akan menerima info langsung saat ada perubahan harga obat."
            : "Dapatkan pemberitahuan langsung di HP saat pengelola memperbarui harga."}
        </p>
        {error && <p className="text-xs text-destructive">{error}</p>}
      </div>

      {subscribed ? (
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={busy}
          onClick={() => void unsubscribe()}
        >
          <BellOff className="h-4 w-4" />
          Matikan
        </Button>
      ) : (
        <Button
          type="button"
          size="sm"
          disabled={busy}
          onClick={() => void subscribe()}
        >
          <Bell className="h-4 w-4" />
          Aktifkan
        </Button>
      )}
    </div>
  );
}
