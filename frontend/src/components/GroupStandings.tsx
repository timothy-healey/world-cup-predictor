import type { Match, Team } from "../types/api";
import { computeGroupStandings } from "../lib/groups";
import { flagFor } from "../data/flags";

interface Props {
  teams: Team[];
  matches: Match[];
}

function matchdayLabel(rows: { matches_played: number }[]): string {
  const maxMP = rows.reduce((m, r) => Math.max(m, r.matches_played), 0);
  if (maxMP === 0) return "Not started";
  return `MD${maxMP}`;
}

export function GroupStandings({ teams, matches }: Props) {
  const standings = computeGroupStandings(teams, matches);
  const groups = Object.keys(standings).sort();

  return (
    <div>
      <header className="mb-3.5 flex items-baseline justify-between">
        <h2 className="text-xs font-semibold uppercase tracking-label text-primary">
          Group standings
        </h2>
        <div className="text-sm text-ink-3">
          Knockout bracket replaces this when group stage ends
        </div>
      </header>
      <div className="grid grid-cols-4 gap-3">
        {groups.map((g) => {
          const rows = standings[g];
          return (
            <div key={g} className="rounded-md border bg-surface p-3">
              <div className="mb-2 flex items-baseline justify-between font-display text-sm font-extrabold uppercase tracking-label-tight text-ink">
                <span>Group {g}</span>
                <span className="font-body text-2xs font-medium text-ink-3 normal-case tracking-normal">
                  {matchdayLabel(rows)}
                </span>
              </div>
              <table className="w-full border-collapse text-xs">
                <thead>
                  <tr>
                    <th className="py-1 text-left text-3xs font-semibold uppercase tracking-label-mid text-ink-3">
                      Team
                    </th>
                    <th className="py-1 text-right text-3xs font-semibold uppercase tracking-label-mid text-ink-3">
                      MP
                    </th>
                    <th className="py-1 text-right text-3xs font-semibold uppercase tracking-label-mid text-ink-3">
                      GD
                    </th>
                    <th className="py-1 text-right text-3xs font-semibold uppercase tracking-label-mid text-ink-3">
                      Pts
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((r) => (
                    <tr key={r.team_code}>
                      <td className="py-1">
                        <span
                          className={`font-display text-2sm font-extrabold uppercase tracking-label-tight ${
                            r.advancing ? "text-ink" : "text-ink-2"
                          }`}
                        >
                          {r.advancing ? (
                            <span className="mr-1.5 inline-block h-1 w-1 rounded-pill bg-correct align-middle" />
                          ) : (
                            <span className="mr-1.5 inline-block w-1" />
                          )}
                          {flagFor(r.team_code)} {r.team_code}
                        </span>
                      </td>
                      <td className="py-1 text-right text-ink-2">{r.matches_played}</td>
                      <td className="py-1 text-right text-ink-2">
                        {r.goal_diff > 0 ? `+${r.goal_diff}` : r.goal_diff}
                      </td>
                      <td className="py-1 text-right font-semibold text-ink">{r.points}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          );
        })}
      </div>
      <p className="mt-5 border-t border-dashed pt-5 text-center text-xs text-ink-3">
        Green dot marks the top 2 in each group, currently auto-advancing to round of 32.
      </p>
    </div>
  );
}
