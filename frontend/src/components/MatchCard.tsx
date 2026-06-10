import type { Match } from "../types/api";
import { latestPrediction } from "../lib/trackRecord";
import { Badge } from "./Badge";
import { Button } from "./Button";
import { Zap, Refresh } from "./icons";
import { flagFor } from "../data/flags";
import { formatKickoff, formatCountdown } from "../lib/format";
import { confidenceBadge } from "../lib/confidence";

interface Props {
  match: Match;
  variant?: "compact" | "next";
  groupLabel?: string; // e.g. "Group B"
  onPredict?: (matchID: string) => void;
  predictDisabled?: boolean;
}

export function MatchCard({
  match,
  variant = "compact",
  groupLabel,
  onPredict,
  predictDisabled,
}: Props) {
  const ko = new Date(match.kickoff_utc);
  const now = new Date();
  const within10 = ko.getTime() - now.getTime() < 10 * 60 * 1000 && ko.getTime() > now.getTime();
  const pred = latestPrediction(match);

  const teamSize =
    variant === "next"
      ? "text-display-lg"
      : "text-xl";

  return (
    <div
      className={`flex flex-col gap-2 rounded-lg border bg-surface ${
        variant === "next" ? "border-ink p-5" : "p-4"
      }`}
    >
      <div className="text-xs font-semibold uppercase tracking-label-mid text-ink-3">
        <span
          className={`mr-1.5 inline-block h-1.5 w-1.5 align-middle rounded-pill bg-pending ${
            within10 ? "animate-pulse !bg-primary" : ""
          }`}
        />
        {formatKickoff(match.kickoff_utc)} · {formatCountdown(ko, now)}
      </div>
      <div
        className={`font-display font-extrabold uppercase tracking-display text-ink leading-none ${teamSize}`}
      >
        {flagFor(match.home_team_code)} {match.home_team_code}{" "}
        <span className="text-ink-4 font-bold text-[0.7em] mx-1">vs</span>{" "}
        {flagFor(match.away_team_code)} {match.away_team_code}
      </div>
      {(groupLabel || match.venue) && (
        <div className="text-xs uppercase tracking-label-mid font-medium text-ink-3">
          {[groupLabel, match.venue].filter(Boolean).join(" · ")}
        </div>
      )}
      <div className="mt-auto flex items-center justify-between pt-2">
        {pred ? (
          <Badge tone={confidenceBadge(pred.confidence).tone}>Predicted</Badge>
        ) : (
          <Badge tone="pending">T-30 scheduled</Badge>
        )}
        {onPredict && (
          <Button
            variant={pred ? "ghost" : "primary"}
            disabled={predictDisabled}
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
        )}
      </div>
    </div>
  );
}
