import { useMemo } from "react";
import type { ExportPayload } from "../types/api";
import { TrackRecord } from "../components/TrackRecord";
import { MatchCard } from "../components/MatchCard";
import { GroupStandings } from "../components/GroupStandings";
import { KnockoutBracket } from "../components/KnockoutBracket";
import { isGroupStageComplete } from "../lib/stage";

interface Props {
  data: ExportPayload;
  onPredict: (matchID: string) => void;
  predictDisabled: boolean;
  predictDisabledReason?: string;
}

export function Dashboard({ data, onPredict, predictDisabled, predictDisabledReason }: Props) {
  const upcoming = useMemo(() => {
    const now = new Date().toISOString();
    return data.matches
      .filter((m) => m.kickoff_utc > now)
      .sort((a, b) => (a.kickoff_utc < b.kickoff_utc ? -1 : 1))
      .slice(0, 3);
  }, [data.matches]);

  const groupStageDone = isGroupStageComplete(data.matches);
  const teamGroup: Record<string, string> = {};
  for (const t of data.teams) teamGroup[t.code] = t.group_id;

  return (
    <div className="bg-bg px-7 py-7">
      <TrackRecord matches={data.matches} />

      <section className="mb-8">
        <header className="mb-3.5 flex items-baseline justify-between">
          <div className="text-xs font-semibold uppercase tracking-label text-primary">
            Upcoming matches
          </div>
          <div className="text-sm text-ink-3">soonest first</div>
        </header>
        <div
          className="grid gap-3.5"
          style={{ gridTemplateColumns: "1.6fr 1fr 1fr" }}
        >
          {upcoming.map((m, idx) => (
            <MatchCard
              key={m.id}
              match={m}
              variant={idx === 0 ? "next" : "compact"}
              groupLabel={teamGroup[m.home_team_code] ? `Group ${teamGroup[m.home_team_code]}` : undefined}
              onPredict={onPredict}
              predictDisabled={predictDisabled}
              predictDisabledReason={predictDisabledReason}
            />
          ))}
          {upcoming.length === 0 && (
            <div className="col-span-3 rounded-lg border bg-surface p-6 text-center text-sm text-ink-3">
              No upcoming matches. Re-run <code className="font-mono">wcp bootstrap</code> if the tournament is in progress.
            </div>
          )}
        </div>
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
