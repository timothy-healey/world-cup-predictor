import type { Match } from "../types/api";
import { buildBracket } from "../lib/bracket";
import { flagFor } from "../data/flags";
import { formatKickoff, formatScore } from "../lib/format";
import { latestPrediction } from "../lib/trackRecord";
import { Check, X } from "./icons";

interface Props {
  matches: Match[];
}

function actualWinner(m: Match): "home" | "away" | "draw" | null {
  if (m.home_score === null || m.away_score === null) return null;
  if (m.home_score > m.away_score) return "home";
  if (m.away_score > m.home_score) return "away";
  return "draw";
}

function teamLabel(code: string): string {
  if (!code) return "TBD";
  return `${flagFor(code)} ${code}`;
}

function teamClass(code: string, side: "home" | "away", outcome: ReturnType<typeof actualWinner>): string {
  const base = "font-display text-[14px] uppercase tracking-[0.02em]";
  if (!code) return `${base} font-medium italic text-ink-4 font-body text-xs tracking-normal normal-case`;
  if (outcome === null) return `${base} font-medium text-ink-2`;
  if (outcome === "draw") return `${base} font-extrabold text-ink`;
  if (outcome === side) return `${base} font-extrabold text-ink`;
  return `${base} font-medium text-ink-4 line-through`;
}

function predictionVerdict(m: Match) {
  const pred = latestPrediction(m);
  if (!pred) return <span className="italic text-ink-3">No prediction yet</span>;
  const actual = actualWinner(m);
  if (actual === null) {
    return (
      <span className="italic text-ink-3">
        Predicted {pred.predicted_winner} {pred.predicted_score} · pending
      </span>
    );
  }
  const actualCode =
    actual === "home" ? m.home_team_code : actual === "away" ? m.away_team_code : "draw";
  const winnerCorrect = pred.predicted_winner === actualCode;
  const exact = pred.predicted_score === `${m.home_score}-${m.away_score}`;
  return (
    <span className="flex items-center gap-1.5 text-ink-2">
      {winnerCorrect ? (
        <Check size={11} className="text-correct" />
      ) : (
        <X size={11} className="text-wrong" />
      )}
      Predicted {pred.predicted_winner} {pred.predicted_score}
      {winnerCorrect && exact ? " · exact" : winnerCorrect ? " · winner correct" : " · wrong"}
    </span>
  );
}

function MatchCell({ m, accent }: { m: Match; accent?: boolean }) {
  const outcome = actualWinner(m);
  const homeScore = m.home_score;
  const awayScore = m.away_score;
  const finished = formatScore(homeScore, awayScore);

  return (
    <div
      className={`rounded-md border bg-surface p-2.5 text-xs ${
        accent ? "bg-secondary-soft border-secondary" : ""
      }`}
    >
      <div className="flex items-center justify-between py-0.5">
        <span className={teamClass(m.home_team_code, "home", outcome)}>
          {teamLabel(m.home_team_code)}
        </span>
        <span className="min-w-[18px] text-right font-display text-base font-extrabold text-ink">
          {finished ? m.home_score : <span className="text-[10px] font-body text-ink-3">{formatKickoff(m.kickoff_utc).split("·")[1]?.trim() ?? "TBD"}</span>}
        </span>
      </div>
      <div className="flex items-center justify-between py-0.5">
        <span className={teamClass(m.away_team_code, "away", outcome)}>
          {teamLabel(m.away_team_code)}
        </span>
        <span className="min-w-[18px] text-right font-display text-base font-extrabold text-ink">
          {finished ? m.away_score : ""}
        </span>
      </div>
      <div className="mt-1.5 border-t border-dashed pt-1.5 text-[10px]">
        {predictionVerdict(m)}
      </div>
    </div>
  );
}

export function KnockoutBracket({ matches }: Props) {
  const bracket = buildBracket(matches);

  return (
    <div>
      <header className="mb-4 text-xs font-semibold uppercase tracking-label text-primary">
        Knockout bracket
      </header>
      <div
        className="grid gap-3.5"
        style={{ gridTemplateColumns: "1.05fr 0.95fr 0.95fr 0.95fr 0.95fr" }}
      >
        {bracket.columns.map((col) => (
          <div key={col.stage} className="font-display text-[13px] font-extrabold uppercase tracking-[0.04em] text-center text-ink">
            {col.label}
          </div>
        ))}
      </div>
      <div
        className="mt-2 grid gap-3.5"
        style={{ gridTemplateColumns: "1.05fr 0.95fr 0.95fr 0.95fr 0.95fr" }}
      >
        {bracket.columns.map((col) => (
          <div
            key={col.stage}
            className="flex min-h-[780px] flex-col justify-around"
          >
            {col.matches.length === 0
              ? Array.from({ length: col.stage === "final" ? 1 : col.stage === "sf" ? 2 : col.stage === "qf" ? 4 : col.stage === "round-of-16" ? 8 : 16 }).map((_, i) => (
                  <div
                    key={`ph-${col.stage}-${i}`}
                    className="rounded-md border bg-surface p-2.5 text-xs text-ink-4"
                  >
                    <div className="py-0.5 italic">TBD</div>
                    <div className="py-0.5 italic">TBD</div>
                  </div>
                ))
              : col.matches.map((m) => (
                  <MatchCell key={m.id} m={m} accent={col.stage === "final"} />
                ))}
          </div>
        ))}
      </div>
      {bracket.thirdPlace && (
        <div className="mt-6">
          <div className="mb-2 text-xs font-semibold uppercase tracking-label text-ink-3">
            Third-place playoff
          </div>
          <div className="max-w-md">
            <MatchCell m={bracket.thirdPlace} />
          </div>
        </div>
      )}
    </div>
  );
}
