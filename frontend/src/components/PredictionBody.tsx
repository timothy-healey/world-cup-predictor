import { useState } from "react";
import type { Match } from "../types/api";
import { latestPrediction } from "../lib/trackRecord";
import { flagFor } from "../data/flags";
import { formatKickoff, formatCountdown } from "../lib/format";
import { confidenceBadge } from "../lib/confidence";
import { okCount, pillTone } from "../lib/traceFormat";
import { Badge } from "./Badge";
import { Button } from "./Button";
import { Refresh, Zap } from "./icons";
import { PredictionStats } from "./PredictionStats";
import { PredictionReasoning } from "./PredictionReasoning";
import { PredictionTrace } from "./PredictionTrace";
import { ThinkingIndicator } from "./ThinkingIndicator";

interface Props {
  match: Match;
  teamName: (code: string) => string;
  variant: "dashboard" | "upcoming";
  groupLabel?: string;
  onPredict?: (matchID: string) => void;
  predictDisabled?: boolean;
  activeMatchId?: string | null;
  elapsedMs?: number;
  onCollapse?: () => void;
}

export function PredictionBody({
  match,
  teamName,
  variant,
  groupLabel,
  onPredict,
  predictDisabled,
  activeMatchId,
  elapsedMs,
  onCollapse,
}: Props) {
  const isPredictingThis = activeMatchId === match.id;
  const pred = latestPrediction(match);
  const ko = new Date(match.kickoff_utc);
  const homeName = teamName(match.home_team_code);
  const awayName = teamName(match.away_team_code);
  const [traceOpen, setTraceOpen] = useState(false);

  const traceAvailable = pred?.trace != null;
  const okN = traceAvailable ? okCount(pred!.trace!) : 0;
  const totalN = traceAvailable ? pred!.trace!.length : 0;
  const tone = traceAvailable ? pillTone(okN, totalN) : "degraded";

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
          {traceAvailable && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                setTraceOpen((o) => !o);
              }}
              className={
                "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-[3px] text-[10px] font-bold uppercase tracking-label transition-colors " +
                (tone === "ok"
                  ? "border-black/12 bg-black/[0.04] text-ink-3 hover:bg-black/10"
                  : "border-primary/22 bg-primary/[0.08] text-primary hover:bg-primary/15")
              }
              aria-expanded={traceOpen}
              aria-label={`Input trace ${okN} of ${totalN} ok`}
            >
              <span
                className={
                  "inline-block h-[5px] w-[5px] rounded-full " +
                  (tone === "ok" ? "bg-emerald-600" : "bg-primary")
                }
                aria-hidden
              />
              {okN}/{totalN} inputs
            </button>
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
        <>
          <div className="mt-6 grid grid-cols-1 gap-6 md:grid-cols-2 md:gap-8">
            <PredictionStats
              prediction={pred}
              teamName={teamName}
              onTraceClick={
                traceAvailable ? () => setTraceOpen((o) => !o) : undefined
              }
            />
            <PredictionReasoning reasoning={pred.reasoning} />
          </div>
          {traceAvailable && (
            <PredictionTrace
              trace={pred.trace!}
              open={traceOpen}
              onToggle={() => setTraceOpen((o) => !o)}
            />
          )}
        </>
      ) : (
        <div className="mt-6 rounded-md border border-dashed bg-surface-sunk px-5 py-4 text-sm text-ink-2">
          No prediction yet. The scheduled launchd agent will fire at T-30, or
          you can predict now manually.
        </div>
      )}

      {(onPredict || (variant === "dashboard" && onCollapse)) && (
        <div className="mt-6 flex items-center justify-between border-t pt-4">
          <div className="flex gap-2.5">
            {onPredict &&
              (isPredictingThis ? (
                <span className="px-4 py-2 text-sm font-semibold text-ink-2">
                  <ThinkingIndicator elapsedMs={elapsedMs} />
                </span>
              ) : (
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
              ))}
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
