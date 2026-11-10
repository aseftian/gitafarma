import { useCallback, useEffect, useState } from "react";
import { fetchVapidPublicKey, subscribePush, unsubscribePush } from "../api";

function urlBase64ToUint8Array(base64String: string): Uint8Array<ArrayBuffer> {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");
  const raw = window.atob(base64);
  const output = new Uint8Array(new ArrayBuffer(raw.length));
  for (let i = 0; i < raw.length; i++) {
    output[i] = raw.charCodeAt(i);
  }
  return output;
}

export function usePushSubscription() {
  const supported =
    typeof window !== "undefined" &&
    "Notification" in window &&
    "serviceWorker" in navigator &&
    "PushManager" in window;

  const [enabledOnServer, setEnabledOnServer] = useState(false);
  const [publicKey, setPublicKey] = useState("");
  const [subscribed, setSubscribed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!supported) return;
    let active = true;

    const init = async () => {
      try {
        const [vapid, reg] = await Promise.all([
          fetchVapidPublicKey(),
          navigator.serviceWorker.ready,
        ]);
        if (!active) return;
        setEnabledOnServer(vapid.enabled);
        setPublicKey(vapid.public_key);

        const existing = await reg.pushManager.getSubscription();
        if (active && existing) {
          setSubscribed(true);
        }
      } catch {
        // silent when offline or VAPID disabled
      }
    };

    void init();
    return () => {
      active = false;
    };
  }, [supported]);

  const subscribe = useCallback(async () => {
    if (!supported || !publicKey) return;
    setBusy(true);
    setError(null);
    try {
      const perm = await Notification.requestPermission();
      if (perm !== "granted") {
        setError("Izin notifikasi ditolak di browser.");
        return;
      }
      const reg = await navigator.serviceWorker.ready;
      const sub = await reg.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(publicKey),
      });
      await subscribePush(sub.toJSON());
      setSubscribed(true);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Gagal mengaktifkan notifikasi.");
    } finally {
      setBusy(false);
    }
  }, [supported, publicKey]);

  const unsubscribe = useCallback(async () => {
    if (!supported) return;
    setBusy(true);
    setError(null);
    try {
      const reg = await navigator.serviceWorker.ready;
      const existing = await reg.pushManager.getSubscription();
      if (existing) {
        await unsubscribePush(existing.endpoint);
        await existing.unsubscribe();
      }
      setSubscribed(false);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Gagal menonaktifkan notifikasi.");
    } finally {
      setBusy(false);
    }
  }, [supported]);

  return {
    supported,
    enabledOnServer,
    subscribed,
    busy,
    error,
    subscribe,
    unsubscribe,
  };
}
