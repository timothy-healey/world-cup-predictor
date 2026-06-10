import type { Match, Prediction } from "../types/api";
import { Badge } from "./Badge";
import { Check, X } from "./icons";
import { flagFor } from "../data/flags";
import { formatKickoff, formatScore, formatTimestamp } from "../lib/format";
import { confidenceBadge } from "../lib/confidence";
import { actualWinnerCode } from "../lib/outcome";
import { parseReasoning } from "../lib/reasoning";
import { stageLabel } from "../lib/stage";

interface Props {
  match: Match;
  teamName: (code: string) => string;
}

interface ActualOutcome {
  winner: string;
  score: string;
}

function verdict(p: Prediction, actual: ActualOutcome) {
  const winnerOk = p.predicted_winner === actual.winner;
  const scoreOk = p.predicted_score === actual.score;
  if (winnerOk && scoreOk) return { tone: "correct" as const, label: "Exact" };
  if (winnerOk) return { tone: "correct" as const, label: "Winner correct" };
  return { tone: "wrong" as const, label: "Wrong" };
}

export function PastMatchCard({ match, teamName }: Props) {
  const winner = actualWinnerCode(match);
  if (winner === null) return null;
  const score = formatScore(match.home_score, match.away_score);
  if (score === null) return null;
  const actual: ActualOutcome = { winner, score };
  const tintCorrect = match.predictions.some((p) => p.predicted_winner === winner);
  const surface = match.predictions.length === 0
    ? "bg-surface"
    : tintCorrect
      ? "bg-correct-soft/30"
      : "bg-wrong-soft/30";
  const sorted = [...match.predictions].sort((a, b) =>
    a.created_at < b.created_at ? 1 : -1,
  );
  const latest = sorted[0];

  return (
    <article className={`mb-3.5 rounded-lg border ${surface} px-6 py-5`}>
      <div className="grid grid-cols-[1fr_2fr] gap-6">
        <div>
          <div className="text-xs font-semibold uppercase tracking-label text-ink-3">
            {formatKickoff(match.kickoff_utc)} · {stageLabel(match.stage)}
          </div>
          <div className="mt-1.5 font-display text-2xl font-extrabold uppercase leading-none tracking-display text-ink">
            {flagFor(match.home_team_code)} {teamName(match.home_team_code)}
            <span className="mx-2 text-[0.65em] font-bold text-ink-4">vs</span>
            {flagFor(match.away_team_code)} {teamName(match.away_team_code)}
          </div>
          <div className="mt-3 font-display text-3xl font-black leading-none text-ink">
            {formatScore(match.home_score, match.away_score)}
          </div>
          {match.venue && <div className="mt-2 text-sm text-ink-2">{match.venue}</div>}
        </div>
        <div>
          <div className="mb-2 text-xs font-semibold uppercase tracking-label text-ink-3">
            Predictions ({sorted.length})
          </div>
          {sorted.length === 0 ? (
            <div className="text-sm italic text-ink-3">No prediction was made.</div>
          ) : (
            <ul className="space-y-1.5">
              {sorted.map((p) => {
                const v = verdict(p, actual);
                return (
                  <li
                    key={p.id}
                    className="flex items-center justify-between rounded-md border bg-surface px-3 py-1.5"
                  >
                    <div className="flex items-center gap-3 text-sm">
                      <span className="font-mono text-xs text-ink-3">
                        {formatTimestamp(p.created_at)}
                      </span>
                      <span className="font-display text-base font-extrabold uppercase text-ink">
                        {teamName(p.predicted_winner)} {p.predicted_score}
                      </span>
                      <Badge tone={confidenceBadge(p.confidence).tone}>
                        {confidenceBadge(p.confidence).label}
                      </Badge>
                      <span className="text-xs text-ink-3">
                        {p.trigger === "scheduled" ? "scheduled" : "on demand"}
                      </span>
                    </div>
                    {v.tone === "correct" ? (
                      <span className="flex items-center gap-1.5 text-sm font-semibold text-correct">
                        <Check size={14} /> {v.label}
                      </span>
                    ) : (
                      <span className="flex items-center gap-1.5 text-sm font-semibold text-wrong">
                        <X size={14} /> {v.label}
                      </span>
                    )}
                  </li>
                );
              })}
            </ul>
          )}
          {latest && (
            <div className="mt-4">
              <div className="mb-1.5 text-xs font-semibold uppercase tracking-label text-ink-3">
                Reasoning (latest)
              </div>
              <ul className="max-w-[62ch] list-disc pl-5 text-sm leading-relaxed">
                {parseReasoning(latest.reasoning).map((line, idx) => (
                  <li key={idx} className="mb-1">
                    {line}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </div>
    </article>
  );
}
