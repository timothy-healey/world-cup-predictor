import type { Prediction } from "../types/api";
import { okCount, pillTone } from "../lib/traceFormat";

interface Props {
  prediction: Prediction;
  teamName: (code: string) => string;
  onTraceClick?: () => void;
}

export function PredictionStats({ prediction, teamName, onTraceClick }: Props) {
  const traceAvailable = prediction.trace !== null && onTraceClick !== undefined;
  const tone =
    prediction.trace !== null
      ? pillTone(okCount(prediction.trace), prediction.trace.length)
      : "degraded";

  return (
    <div className="rounded-md bg-surface-sunk p-5 sm:p-6">
      <div className="mb-2 text-2xs font-semibold uppercase tracking-label text-ink-3">
        Predicted winner
      </div>
      <div className="font-display text-display-lg font-extrabold uppercase leading-none tracking-display text-primary">
        {teamName(prediction.predicted_winner)}
      </div>

      <div className="mt-5 grid grid-cols-2 gap-5">
        <div>
          <div className="mb-1.5 text-2xs font-semibold uppercase tracking-label text-ink-3">
            Score
          </div>
          <div className="font-display text-3xl font-extrabold leading-none text-ink">
            {prediction.predicted_score}
          </div>
        </div>
        <div>
          <div className="mb-1.5 flex items-center gap-2 text-2xs font-semibold uppercase tracking-label text-ink-3">
            Win probability
            {traceAvailable && (
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  onTraceClick!();
                }}
                aria-label="View input trace"
                title="View input trace"
                className={
                  "inline-flex h-5 w-5 items-center justify-center rounded-full border text-[10px] font-bold leading-none transition-colors " +
                  (tone === "ok"
                    ? "border-black/15 bg-black/[0.04] text-ink-3 hover:bg-black/10"
                    : "border-primary/25 bg-primary/[0.08] text-primary hover:bg-primary/15")
                }
              >
                i
              </button>
            )}
          </div>
          <div className="inline-block border-b-[3px] border-secondary pb-0.5 font-display text-3xl font-extrabold leading-none text-ink">
            {Math.round(prediction.win_probability * 100)}%
          </div>
        </div>
      </div>
    </div>
  );
}
