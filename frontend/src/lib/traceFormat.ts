import type { TraceEntry } from "../types/api";

export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

export function okCount(entries: TraceEntry[]): number {
  let n = 0;
  for (const e of entries) if (e.ok) n++;
  return n;
}

export type PillTone = "ok" | "degraded";

// pillTone returns 'ok' only when every entry succeeded AND the trace is the
// expected length (5). Anything shorter is treated as degraded so the UI
// doesn't display a green "0/0" pill on a malformed trace.
export function pillTone(okN: number, total: number): PillTone {
  if (total === 0) return "degraded";
  return okN === total ? "ok" : "degraded";
}
