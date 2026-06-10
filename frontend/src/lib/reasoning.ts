// Parses an LLM-emitted reasoning string into bullet lines.
// Splits on newlines, strips a leading "- " or "* " if present, trims, drops empties.
// Used by PredictionCard and PastMatchCard to render reasoning as a <ul>.
export function parseReasoning(text: string): string[] {
  return text
    .split(/\r?\n/)
    .map((line) => line.replace(/^[-*]\s*/, "").trim())
    .filter((line) => line.length > 0);
}
