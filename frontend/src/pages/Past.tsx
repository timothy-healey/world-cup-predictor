import { useMemo, useState } from "react";
import type { ExportPayload, Match } from "../types/api";
import { PastMatchCard } from "../components/PastMatchCard";

interface Props {
  data: ExportPayload;
}

type Verdict = "all" | "correct" | "wrong";

function actualWinner(m: Match): string | null {
  if (m.home_score === null || m.away_score === null) return null;
  if (m.home_score > m.away_score) return m.home_team_code;
  if (m.away_score > m.home_score) return m.away_team_code;
  return "draw";
}

function anyCorrect(m: Match): boolean | null {
  const actual = actualWinner(m);
  if (actual === null || m.predictions.length === 0) return null;
  return m.predictions.some((p) => p.predicted_winner === actual);
}

export function Past({ data }: Props) {
  const [team, setTeam] = useState<string>("all");
  const [verdict, setVerdict] = useState<Verdict>("all");

  const filtered: Match[] = useMemo(() => {
    let rows = data.matches.filter(
      (m) => m.home_score !== null && m.away_score !== null,
    );
    if (team !== "all") {
      rows = rows.filter(
        (m) => m.home_team_code === team || m.away_team_code === team,
      );
    }
    if (verdict !== "all") {
      rows = rows.filter((m) => {
        const ok = anyCorrect(m);
        if (ok === null) return false;
        return verdict === "correct" ? ok : !ok;
      });
    }
    rows.sort((a, b) => (a.kickoff_utc < b.kickoff_utc ? 1 : -1));
    return rows;
  }, [data.matches, team, verdict]);

  return (
    <div className="bg-bg px-7 py-7">
      <header className="mb-5 flex items-center justify-between">
        <div className="text-xs font-semibold uppercase tracking-label text-primary">
          Past results
        </div>
        <div className="flex items-center gap-3">
          <select
            value={team}
            onChange={(e) => setTeam(e.target.value)}
            className="rounded border bg-surface px-3 py-1.5 text-sm text-ink"
          >
            <option value="all">All teams</option>
            {[...data.teams]
              .sort((a, b) => a.name.localeCompare(b.name))
              .map((t) => (
                <option key={t.code} value={t.code}>
                  {t.name}
                </option>
              ))}
          </select>
          <select
            value={verdict}
            onChange={(e) => setVerdict(e.target.value as Verdict)}
            className="rounded border bg-surface px-3 py-1.5 text-sm text-ink"
          >
            <option value="all">All predictions</option>
            <option value="correct">Correct only</option>
            <option value="wrong">Wrong only</option>
          </select>
        </div>
      </header>
      {filtered.length === 0 ? (
        <div className="rounded-lg border bg-surface p-6 text-center text-sm text-ink-3">
          No past results match these filters.
        </div>
      ) : (
        filtered.map((m) => <PastMatchCard key={m.id} match={m} />)
      )}
    </div>
  );
}
