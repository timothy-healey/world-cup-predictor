import type { Match } from "../types/api";
import { trackRecord } from "../lib/trackRecord";

interface Props {
  matches: Match[];
}

function pctParts(n: number | null): { value: string; unit: string } {
  if (n === null) return { value: "—", unit: "" };
  return { value: String(Math.round(n * 100)), unit: "%" };
}

export function TrackRecord({ matches }: Props) {
  const r = trackRecord(matches);
  const winner = pctParts(r.winnerAccuracy);
  const exact = pctParts(r.exactAccuracy);
  const avg = r.averageConfidence === null ? "—" : r.averageConfidence.toFixed(1);
  const avgLabel =
    r.averageConfidence === null
      ? "No predictions yet"
      : r.averageConfidence >= 2.5
        ? "Mostly high"
        : r.averageConfidence >= 1.5
          ? "Mostly medium"
          : "Mostly low";

  return (
    <section
      className="mb-8 grid items-end gap-0 rounded-lg border bg-surface px-7 py-6"
      style={{ gridTemplateColumns: "2.4fr 1fr 1fr 1fr" }}
    >
      <div>
        <div className="mb-1.5 text-xs font-semibold uppercase tracking-label text-ink-3">
          Winner accuracy
        </div>
        <div className="flex items-baseline gap-3 font-display text-[88px] font-black leading-[0.9] text-ink">
          <span className="text-primary">
            {winner.value}
            {winner.unit && <span className="text-[48px]">{winner.unit}</span>}
          </span>
          <span className="text-[32px] font-extrabold text-ink-3">
            {r.winnerCorrect} of {r.completed}
          </span>
        </div>
        <p className="mt-1.5 max-w-[32ch] text-xs leading-relaxed text-ink-2">
          Predictions with confirmed results so far. Sample still small. Group-stage matchday 3 will roughly double it.
        </p>
      </div>
      <SupStat
        label="Exact score"
        value={exact.value}
        unit={exact.unit}
        sub={`${r.exactCorrect} of ${r.completed}`}
      />
      <SupStat
        label="Avg confidence"
        value={avg}
        unit={r.averageConfidence === null ? "" : " / 3"}
        sub={avgLabel}
      />
      <SupStat
        label="Predictions"
        value={String(r.total)}
        unit=""
        sub={`${r.completed} with results`}
      />
    </section>
  );
}

function SupStat({
  label,
  value,
  unit,
  sub,
}: {
  label: string;
  value: string;
  unit: string;
  sub: string;
}) {
  return (
    <div className="border-l pl-6">
      <div className="mb-1.5 text-[10px] font-semibold uppercase tracking-label text-ink-3">
        {label}
      </div>
      <div className="font-display text-[28px] font-extrabold leading-none text-ink">
        {value}
        <span className="text-base text-ink-3">{unit}</span>
      </div>
      <div className="mt-1 text-xs text-ink-3">{sub}</div>
    </div>
  );
}
