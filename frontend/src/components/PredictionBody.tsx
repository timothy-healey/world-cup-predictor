import type { Match } from "../types/api";
import { latestPrediction } from "../lib/trackRecord";
import { flagFor } from "../data/flags";
import { formatKickoff, formatCountdown } from "../lib/format";
import { confidenceBadge } from "../lib/confidence";
import { Badge } from "./Badge";
import { Button } from "./Button";
import { Refresh, Zap } from "./icons";
import { PredictionStats } from "./PredictionStats";
import { PredictionReasoning } from "./PredictionReasoning";

interface Props {
  match: Match;
  teamName: (code: string) => string;
  variant: "dashboard" | "upcoming";
  groupLabel?: string;
  onPredict?: (matchID: string) => void;
  predictDisabled?: boolean;
  onCollapse?: () => void;
}

export function PredictionBody({
  match,
  teamName,
  variant,
  groupLabel,
  onPredict,
  predictDisabled,
  onCollapse,
}: Props) {
  const pred = latestPrediction(match);
  const ko = new Date(match.kickoff_utc);
  const homeName = teamName(match.home_team_code);
  const awayName = teamName(match.away_team_code);

  return (
    <div className="wcp-reveal">
      <header className="mb-4 flex flex-wrap items-baseline justify-between gap-2">
        <div className="text-xs font-semibold uppercase tracking-label-mid text-ink-3">
          {formatKickoff(match.kickoff_utc)} · {formatCountdown(ko)}
        </div>
        <div className="flex items-center gap-3">
          {groupLabel && (
            <span className="text-xs font-semibold uppercase tracking-label text-primary">
              {groupLabel}
            </span>
          )}
          {pred && (
            <Badge tone={confidenceBadge(pred.confidence).tone}>
              {confidenceBadge(pred.confidence).label} confidence
            </Badge>
          )}
        </div>
      </header>

      <div className="font-display text-display-lg font-extrabold uppercase leading-none tracking-display text-ink">
        {flagFor(match.home_team_code)} {homeName}
        <span className="mx-3 text-[0.55em] font-bold text-ink-4">vs</span>
        {flagFor(match.away_team_code)} {awayName}
      </div>
      {match.venue && (
        <div className="mt-2 text-sm text-ink-2">{match.venue}</div>
      )}

      {pred ? (
        <div className="mt-6 grid grid-cols-1 gap-6 md:grid-cols-2 md:gap-8">
          <PredictionStats prediction={pred} teamName={teamName} />
          <PredictionReasoning reasoning={pred.reasoning} />
        </div>
      ) : (
        <div className="mt-6 rounded-md border border-dashed bg-surface-sunk px-5 py-4 text-sm text-ink-2">
          No prediction yet. The scheduled launchd agent will fire at T-30, or
          you can predict now manually.
        </div>
      )}

      {(onPredict || (variant === "dashboard" && onCollapse)) && (
        <div className="mt-6 flex items-center justify-between border-t pt-4">
          <div className="flex gap-2.5">
            {onPredict && (
              <Button
                variant={pred ? "ghost" : "primary"}
                disabled={predictDisabled}
                onClick={(e) => {
                  e.stopPropagation();
                  onPredict(match.id);
                }}
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
            )}
          </div>
          {variant === "dashboard" && onCollapse && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                onCollapse();
              }}
              className="text-2xs font-semibold uppercase tracking-label text-ink-3 hover:text-ink focus:outline-none focus-visible:shadow-focus"
            >
              ▴ Collapse
            </button>
          )}
        </div>
      )}
    </div>
  );
}
