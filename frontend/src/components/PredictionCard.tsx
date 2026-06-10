import type { Match } from "../types/api";
import { latestPrediction } from "../lib/trackRecord";
import { Button } from "./Button";
import { Badge } from "./Badge";
import { Refresh, Zap } from "./icons";
import { flagFor } from "../data/flags";
import { formatKickoff, formatCountdown } from "../lib/format";
import { confidenceBadge } from "../lib/confidence";

interface Props {
  match: Match;
  groupLabel?: string;
  onPredict?: (matchID: string) => void;
  predictDisabled?: boolean;
  predictDisabledReason?: string;
}

function parseReasoning(text: string): string[] {
  return text
    .split(/\r?\n/)
    .map((line) => line.replace(/^[-*]\s*/, "").trim())
    .filter((line) => line.length > 0);
}

function teamDisplay(code: string, name: string): string {
  return `${flagFor(code)} ${name || code}`;
}

export function PredictionCard({
  match,
  groupLabel,
  onPredict,
  predictDisabled,
  predictDisabledReason,
}: Props) {
  const pred = latestPrediction(match);
  const ko = new Date(match.kickoff_utc);
  const homeDisplay = teamDisplay(match.home_team_code, "");
  const awayDisplay = teamDisplay(match.away_team_code, "");

  return (
    <article className="mb-3.5 rounded-lg border bg-surface px-6 py-5">
      <header className="flex items-start justify-between gap-6">
        <div>
          {groupLabel && (
            <div className="text-xs font-semibold uppercase tracking-label text-primary">
              {groupLabel}
            </div>
          )}
          <div className="mt-1.5 font-display text-[36px] font-extrabold uppercase leading-none tracking-display text-ink">
            {homeDisplay}
            <span className="mx-2 text-[0.65em] font-bold text-ink-4">vs</span>
            {awayDisplay}
          </div>
          {match.venue && <div className="mt-1 text-sm text-ink-2">{match.venue}</div>}
        </div>
        <div className="text-right text-sm text-ink-2">
          {formatKickoff(match.kickoff_utc)}
          <br />
          <span className="font-semibold text-ink">{formatCountdown(ko)}</span>
        </div>
      </header>

      {pred ? (
        <>
          <div className="mt-4 flex items-end gap-6 rounded-md bg-surface-sunk px-5 py-4">
            <Stat label="Predicted winner" value={pred.predicted_winner} valueClass="text-primary" />
            <Stat label="Score" value={pred.predicted_score} />
            <Stat
              label="Win prob"
              value={`${Math.round(pred.win_probability * 100)}%`}
              valueClass="text-secondary [-webkit-text-stroke:0.5px_#1B0E12]"
            />
            <div className="flex-1" />
            <div>
              <div className="mb-1 text-[10px] font-semibold uppercase tracking-label text-ink-3">
                Confidence
              </div>
              <Badge tone={confidenceBadge(pred.confidence).tone}>
                {confidenceBadge(pred.confidence).label}
              </Badge>
            </div>
          </div>
          <div className="mt-4 text-xs font-semibold uppercase tracking-label text-ink-3">
            Reasoning
          </div>
          <ul className="mt-2 max-w-[62ch] list-disc pl-5 text-sm leading-relaxed">
            {parseReasoning(pred.reasoning).map((line, idx) => (
              <li key={idx} className="mb-1">
                {line}
              </li>
            ))}
          </ul>
        </>
      ) : (
        <div className="mt-4 rounded-md border border-dashed bg-surface-sunk px-5 py-4 text-sm text-ink-2">
          No prediction yet — the scheduled launchd agent will fire at T-30. You can also predict now manually.
        </div>
      )}

      {onPredict && (
        <div className="mt-5 flex gap-2.5">
          <Button
            variant={pred ? "ghost" : "primary"}
            disabled={predictDisabled}
            title={predictDisabled ? predictDisabledReason : undefined}
            onClick={() => onPredict(match.id)}
          >
            {pred ? (
              <>
                <Refresh /> Re-predict
              </>
            ) : (
              <>
                <Zap /> Predict now
              </>
            )}
          </Button>
        </div>
      )}
    </article>
  );
}

function Stat({ label, value, valueClass = "text-ink" }: { label: string; value: string; valueClass?: string }) {
  return (
    <div className="flex flex-col gap-1">
      <div className="text-[10px] font-semibold uppercase tracking-label text-ink-3">{label}</div>
      <div className={`font-display text-3xl font-extrabold leading-none ${valueClass}`}>
        {value}
      </div>
    </div>
  );
}
