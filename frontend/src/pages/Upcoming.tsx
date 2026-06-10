import { useMemo, useState } from "react";
import type { ExportPayload, Match } from "../types/api";
import { PredictionCard } from "../components/PredictionCard";
import { stageLabel } from "../lib/stage";

interface Props {
  data: ExportPayload;
  onPredict: (matchID: string) => void;
  predictDisabled: boolean;
}

type SortDir = "soonest" | "latest";

const SEVEN_DAYS_MS = 7 * 24 * 60 * 60 * 1000;

export function Upcoming({ data, onPredict, predictDisabled }: Props) {
  const [team, setTeam] = useState<string>("all");
  const [sort, setSort] = useState<SortDir>("soonest");
  const [showAll, setShowAll] = useState(false);

  const teamGroup: Record<string, string> = {};
  for (const t of data.teams) teamGroup[t.code] = t.group_id;

  const filtered: Match[] = useMemo(() => {
    const now = new Date();
    const cutoff = new Date(now.getTime() + SEVEN_DAYS_MS);
    let rows = data.matches.filter((m) => new Date(m.kickoff_utc) > now);
    if (!showAll) rows = rows.filter((m) => new Date(m.kickoff_utc) <= cutoff);
    if (team !== "all") {
      rows = rows.filter(
        (m) => m.home_team_code === team || m.away_team_code === team,
      );
    }
    rows.sort((a, b) =>
      sort === "soonest"
        ? a.kickoff_utc.localeCompare(b.kickoff_utc)
        : b.kickoff_utc.localeCompare(a.kickoff_utc),
    );
    return rows;
  }, [data.matches, team, sort, showAll]);

  return (
    <div className="bg-bg px-7 py-7">
      <header className="mb-5 flex items-center justify-between">
        <h2 className="text-xs font-semibold uppercase tracking-label text-primary">
          Upcoming matches
        </h2>
        <div className="flex items-center gap-3">
          <select
            value={team}
            onChange={(e) => setTeam(e.target.value)}
            aria-label="Filter by team"
            className="rounded border bg-surface px-3 py-1.5 text-sm text-ink focus:outline-none focus-visible:shadow-focus"
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
            value={sort}
            onChange={(e) => setSort(e.target.value as SortDir)}
            aria-label="Sort order"
            className="rounded border bg-surface px-3 py-1.5 text-sm text-ink focus:outline-none focus-visible:shadow-focus"
          >
            <option value="soonest">Soonest first</option>
            <option value="latest">Latest first</option>
          </select>
          <label className="flex items-center gap-2 text-sm text-ink-2">
            <input
              type="checkbox"
              checked={showAll}
              onChange={(e) => setShowAll(e.target.checked)}
              className="focus:outline-none focus-visible:shadow-focus"
            />
            Show all
          </label>
        </div>
      </header>

      {filtered.length === 0 ? (
        <div className="rounded-lg border bg-surface p-6 text-center text-sm text-ink-3">
          No matches in this window.
        </div>
      ) : (
        filtered.map((m) => (
          <PredictionCard
            key={m.id}
            match={m}
            groupLabel={
              teamGroup[m.home_team_code]
                ? `Group ${teamGroup[m.home_team_code]} · ${stageLabel(m.stage)}`
                : stageLabel(m.stage)
            }
            onPredict={onPredict}
            predictDisabled={predictDisabled}
          />
        ))
      )}
    </div>
  );
}
