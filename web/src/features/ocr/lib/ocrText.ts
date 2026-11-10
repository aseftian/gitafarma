export interface OcrExtraction {
  primaryQuery: string;
  candidates: string[];
}

export function extractOcrCandidates(rawText: string): OcrExtraction {
  const lines = rawText
    .split(/\r?\n/)
    .map((line) =>
      line
        .replace(/[^a-zA-Z0-9\s./-]/g, " ")
        .replace(/\s+/g, " ")
        .trim(),
    )
    .filter((line) => line.length >= 3 && /[a-zA-Z]{3,}/.test(line));

  const seen = new Set<string>();
  const unique: string[] = [];
  for (const line of lines) {
    const key = line.toLowerCase();
    if (seen.has(key)) continue;
    seen.add(key);
    unique.push(line);
  }

  const candidates = unique.slice(0, 5);
  const primaryQuery = candidates[0] ?? rawText.replace(/\s+/g, " ").trim();
  return { primaryQuery, candidates };
}
