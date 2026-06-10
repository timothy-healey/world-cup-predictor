import { useEffect, useMemo, useState } from "react";
import type { ExportPayload } from "../types/api";
import { TrackRecord } from "../components/TrackRecord";
import { MatchCard } from "../components/MatchCard";
import { GroupStandings } from "../components/GroupStandings";
import { KnockoutBracket } from "../components/KnockoutBracket";
import { isGroupStageComplete } from "../lib/stage";
import { buildTeamNameLookup } from "../lib/teams";
import { nextExpandedId } from "../lib/expand";

const UPCOMING_GRID = "1.6fr 1fr 1fr";
const COMPACT_REMAINDER_GRID = "1fr 1fr";

interface Props {
  data: ExportPayload;
  onPredict: (matchID: string) => void;
  predictDisabled: boolean;
}

export function Dashboard({ data, onPredict, predictDisabled }: Props) {
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const upcoming = useMemo(() => {
    const now = new Date().toISOString();
    return data.matches
      .filter((m) => m.kickoff_utc > now)
      .sort((a, b) => (a.kickoff_utc < b.kickoff_utc ? -1 : 1))
      .slice(0, 3);
  }, [data.matches]);

  const teamName = useMemo(() => buildTeamNameLookup(data.teams), [data.teams]);

  const teamGroup: Record<string, string> = {};
  for (const t of data.teams) teamGroup[t.code] = t.group_id;

  useEffect(() => {
    if (expandedId && !upcoming.some((m) => m.id === expandedId)) {
      setExpandedId(null);
    }
  }, [expandedId, upcoming]);

  useEffect(() => {
    if (expandedId === null) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setExpandedId(null);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [expandedId]);

  const groupStageDone = isGroupStageComplete(data.matches);
  const expandedMatch = expandedId
    ? upcoming.find((m) => m.id === expandedId) ?? null
    : null;
  const compactRemainder = upcoming.filter((m) => m.id !== expandedId);

  return (
    <div className="bg-bg px-7 py-7">
      <TrackRecord matches={data.matches} />

      <section className="mb-8">
        <header className="mb-3.5 flex items-baseline justify-between">
          <h2 className="text-xs font-semibold uppercase tracking-label text-primary">
            Upcoming matches
          </h2>
          <div className="text-sm text-ink-3">soonest first</div>
        </header>

        {upcoming.length === 0 ? (
          <div className="rounded-lg border bg-surface p-6 text-center text-sm text-ink-3">
            No upcoming matches. Re-run <code className="font-mono">wcp bootstrap</code> if the tournament is in progress.
          </div>
        ) : expandedMatch ? (
          <div className="flex flex-col gap-3.5">
            <MatchCard
              key={expandedMatch.id}
              match={expandedMatch}
              expanded
              teamName={teamName}
              onToggle={() => setExpandedId(null)}
              groupLabel={
                teamGroup[expandedMatch.home_team_code]
                  ? `Group ${teamGroup[expandedMatch.home_team_code]}`
                  : undefined
              }
              onPredict={onPredict}
              predictDisabled={predictDisabled}
            />
            <div
              className="grid gap-3.5"
              style={{ gridTemplateColumns: COMPACT_REMAINDER_GRID }}
            >
              {compactRemainder.map((m) => (
                <MatchCard
                  key={m.id}
                  match={m}
                  variant="compact"
                  teamName={teamName}
                  groupLabel={
                    teamGroup[m.home_team_code]
                      ? `Group ${teamGroup[m.home_team_code]}`
                      : undefined
                  }
                  onPredict={onPredict}
                  predictDisabled={predictDisabled}
                  onToggle={() => setExpandedId(nextExpandedId(expandedId, m.id))}
                />
              ))}
            </div>
          </div>
        ) : (
          <div
            className="grid gap-3.5"
            style={{ gridTemplateColumns: UPCOMING_GRID }}
          >
            {upcoming.map((m, idx) => (
              <MatchCard
                key={m.id}
                match={m}
                variant={idx === 0 ? "next" : "compact"}
                teamName={teamName}
                groupLabel={
                  teamGroup[m.home_team_code]
                    ? `Group ${teamGroup[m.home_team_code]}`
                    : undefined
                }
                onPredict={onPredict}
                predictDisabled={predictDisabled}
                onToggle={() => setExpandedId(nextExpandedId(expandedId, m.id))}
              />
            ))}
          </div>
        )}
      </section>

      <section>
        {groupStageDone ? (
          <KnockoutBracket matches={data.matches} />
        ) : (
          <GroupStandings teams={data.teams} matches={data.matches} />
        )}
      </section>
    </div>
  );
}
