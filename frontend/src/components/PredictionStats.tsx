import type { Prediction } from "../types/api";

interface Props {
  prediction: Prediction;
  teamName: (code: string) => string;
}

export function PredictionStats({ prediction, teamName }: Props) {
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
          <div className="mb-1.5 text-2xs font-semibold uppercase tracking-label text-ink-3">
            Win probability
          </div>
          <div className="inline-block border-b-[3px] border-secondary pb-0.5 font-display text-3xl font-extrabold leading-none text-ink">
            {Math.round(prediction.win_probability * 100)}%
          </div>
        </div>
      </div>
    </div>
  );
}
