import { useCallback, useEffect, useRef, useState } from "react";
import type { IScannerControls } from "@zxing/browser";

interface BarcodeDetectorResult {
  rawValue?: string;
}

interface BarcodeDetectorInstance {
  detect(source: HTMLVideoElement): Promise<BarcodeDetectorResult[]>;
}

type BarcodeDetectorCtor = new (options?: { formats?: string[] }) => BarcodeDetectorInstance;

function getNativeBarcodeDetector(): BarcodeDetectorCtor | undefined {
  return (window as unknown as { BarcodeDetector?: BarcodeDetectorCtor }).BarcodeDetector;
}

export function useBarcodeScanner(
  active: boolean,
  onDetected: (barcode: string) => void,
) {
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [engine, setEngine] = useState<"native" | "zxing" | null>(null);
  const onDetectedRef = useRef(onDetected);
  onDetectedRef.current = onDetected;

  const stopTracks = useCallback((stream: MediaStream | null) => {
    stream?.getTracks().forEach((t) => t.stop());
  }, []);

  useEffect(() => {
    if (!active) {
      setError(null);
      return;
    }

    if (!navigator.mediaDevices?.getUserMedia) {
      setError("Kamera tidak didukung di browser ini atau halaman belum memakai HTTPS.");
      return;
    }

    const videoEl = videoRef.current;
    if (!videoEl) return;

    let cancelled = false;
    let rafId = 0;
    let stream: MediaStream | null = null;
    let zxingControls: IScannerControls | null = null;
    let detected = false;

    const handleCode = (raw: string) => {
      if (detected || cancelled) return;
      const clean = raw.trim();
      if (!clean) return;
      detected = true;
      onDetectedRef.current(clean);
    };

    const start = async () => {
      setError(null);

      const NativeDetector = getNativeBarcodeDetector();
      if (NativeDetector) {
        try {
          setEngine("native");
          stream = await navigator.mediaDevices.getUserMedia({
            video: { facingMode: { ideal: "environment" } },
            audio: false,
          });
          if (cancelled) {
            stopTracks(stream);
            return;
          }
          videoEl.srcObject = stream;
          await videoEl.play();

          const detector = new NativeDetector({
            formats: ["ean_13", "ean_8", "code_128", "code_39", "upc_a", "upc_e", "qr_code"],
          });

          const tick = async () => {
            if (cancelled || detected) return;
            try {
              if (videoEl.readyState >= 2) {
                const hits = await detector.detect(videoEl);
                const first = hits.find((h) => h.rawValue && h.rawValue.trim() !== "");
                if (first?.rawValue) {
                  handleCode(first.rawValue);
                  return;
                }
              }
            } catch {
              // ignore transient frame error
            }
            rafId = window.requestAnimationFrame(() => void tick());
          };
          rafId = window.requestAnimationFrame(() => void tick());
          return;
        } catch (err) {
          stopTracks(stream);
          stream = null;
          if (err instanceof DOMException && err.name === "NotAllowedError") {
            if (!cancelled) setError("Izin kamera ditolak. Aktifkan izin kamera pada browser Anda.");
            return;
          }
        }
      }

      try {
        setEngine("zxing");
        const { BrowserMultiFormatReader } = await import("@zxing/browser");
        if (cancelled) return;
        const reader = new BrowserMultiFormatReader();
        zxingControls = await reader.decodeFromConstraints(
          { video: { facingMode: { ideal: "environment" } } },
          videoEl,
          (result) => {
            if (result) {
              handleCode(result.getText());
            }
          },
        );
      } catch (err) {
        if (!cancelled) {
          if (err instanceof DOMException && err.name === "NotAllowedError") {
            setError("Izin kamera ditolak. Aktifkan izin kamera pada browser Anda.");
          } else {
            setError("Gagal mengakses kamera. Pastikan menggunakan HTTPS dan perangkat memiliki kamera.");
          }
        }
      }
    };

    void start();

    return () => {
      cancelled = true;
      if (rafId) window.cancelAnimationFrame(rafId);
      zxingControls?.stop();
      stopTracks(stream);
      videoEl.srcObject = null;
    };
  }, [active, stopTracks]);

  return { videoRef, error, engine };
}
