import { useCallback, useEffect, useRef, useState } from "react";
import type { Worker } from "tesseract.js";
import { extractOcrCandidates } from "../lib/ocrText";

export function useOcr() {
  const workerRef = useRef<Worker | null>(null);
  const [processing, setProcessing] = useState(false);
  const [progress, setProgress] = useState(0);
  const [rawText, setRawText] = useState("");
  const [editedText, setEditedText] = useState("");
  const [candidates, setCandidates] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    return () => {
      if (workerRef.current) {
        void workerRef.current.terminate();
        workerRef.current = null;
      }
    };
  }, []);

  const recognizeFile = useCallback(async (file: File) => {
    setProcessing(true);
    setProgress(0);
    setError(null);

    try {
      if (!workerRef.current) {
        const { createWorker } = await import("tesseract.js");
        workerRef.current = await createWorker("eng", undefined, {
          logger: (msg) => {
            if (msg.status === "recognizing text" && typeof msg.progress === "number") {
              setProgress(Math.round(msg.progress * 100));
            }
          },
        });
      }

      const result = await workerRef.current.recognize(file);
      const text = result.data.text ?? "";
      setRawText(text);
      const extracted = extractOcrCandidates(text);
      setCandidates(extracted.candidates);
      setEditedText(extracted.primaryQuery);
    } catch (err: unknown) {
      setError(
        err instanceof Error
          ? err.message
          : "Gagal membaca teks dari gambar. Coba foto lebih dekat dan terang.",
      );
    } finally {
      setProcessing(false);
    }
  }, []);

  const reset = useCallback(() => {
    setRawText("");
    setEditedText("");
    setCandidates([]);
    setProgress(0);
    setError(null);
  }, []);

  return {
    processing,
    progress,
    rawText,
    editedText,
    setEditedText,
    candidates,
    error,
    recognizeFile,
    reset,
  };
}
