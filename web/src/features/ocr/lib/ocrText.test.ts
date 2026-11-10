import { describe, expect, it } from "vitest";
import { extractOcrCandidates } from "./ocrText";

describe("extractOcrCandidates", () => {
  it("cleans noisy packaging OCR output and keeps unique drug name lines", () => {
    const raw = `
      !! 12 @@
      PARACETAMOL 500 mg
      Kaplet
      PARACETAMOL 500 mg
      PT FARMA INDONESIA
    `;
    const res = extractOcrCandidates(raw);
    expect(res.primaryQuery).toBe("PARACETAMOL 500 mg");
    expect(res.candidates).toEqual([
      "PARACETAMOL 500 mg",
      "Kaplet",
      "PT FARMA INDONESIA",
    ]);
  });
});
