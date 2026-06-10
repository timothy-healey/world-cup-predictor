import type { TraceEntry } from "../types/api";
import { formatDuration, okCount } from "../lib/traceFormat";

interface Props {
  trace: TraceEntry[];
  open: boolean;
  onToggle: () => void;
}

export function PredictionTrace({ trace, open, onToggle }: Props) {
  const ok = okCount(trace);
  if (!open) return null;
  return (
    <div className="mt-4 overflow-hidden rounded-md border bg-surface-sunk">
      <div className="flex items-center justify-between border-b bg-black/[0.03] px-4 py-2">
        <h5 className="m-0 text-2xs font-semibold uppercase tracking-label text-ink">
          Input trace · {ok}/{trace.length} ok
        </h5>
        <button
          type="button"
          onClick={onToggle}
          className="text-2xs font-semibold uppercase tracking-label text-ink-3 hover:text-ink focus:outline-none focus-visible:shadow-focus"
        >
          ▾ Collapse
        </button>
      </div>
      <ul className="m-0 list-none p-0">
        {trace.map((e) => (
          <li
            key={e.kind}
            className="border-b border-black/[0.06] px-4 py-3 last:border-b-0"
          >
            <div className="flex items-center justify-between">
              <span className="flex items-center gap-2 text-2xs font-semibold uppercase tracking-label text-ink">
                <span
                  className={`inline-block h-2 w-2 rounded-full ${
                    e.ok ? "bg-emerald-600" : "bg-red-600"
                  }`}
                  aria-hidden
                />
                {e.kind}
              </span>
              <span className="text-2xs tabular-nums text-ink-3">
                {e.ok ? (
                  <span className="font-semibold text-emerald-700">✓ ok</span>
                ) : (
                  <span className="font-semibold text-red-700">✗ failed</span>
                )}
                {" · "}
                {formatDuration(e.duration_ms)}
              </span>
            </div>
            {e.error && (
              <div className="mt-1 pl-4 text-xs text-red-700">{e.error}</div>
            )}
            {e.snippet && (
              <pre className="mt-1 ml-4 overflow-x-auto rounded bg-black/[0.04] px-2 py-1 text-[10.5px] leading-snug text-ink-2">
                {e.snippet}
              </pre>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}
