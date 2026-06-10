import type { Match, Team } from "../types/api";

export interface StandingRow {
  team_code: string;
  matches_played: number;
  wins: number;
  draws: number;
  losses: number;
  goals_for: number;
  goals_against: number;
  goal_diff: number;
  points: number;
  advancing: boolean;
}

export type GroupStandings = Record<string, StandingRow[]>;

function emptyRow(code: string): StandingRow {
  return {
    team_code: code,
    matches_played: 0,
    wins: 0,
    draws: 0,
    losses: 0,
    goals_for: 0,
    goals_against: 0,
    goal_diff: 0,
    points: 0,
    advancing: false,
  };
}

export function computeGroupStandings(teams: Team[], matches: Match[]): GroupStandings {
  const rowsByCode: Record<string, StandingRow> = {};
  const groupMembers: Record<string, string[]> = {};
  for (const t of teams) {
    if (!t.group_id) continue;
    rowsByCode[t.code] = emptyRow(t.code);
    (groupMembers[t.group_id] ??= []).push(t.code);
  }

  for (const m of matches) {
    if (m.stage !== "group") continue;
    if (m.home_score === null || m.away_score === null) continue;
    const h = rowsByCode[m.home_team_code];
    const a = rowsByCode[m.away_team_code];
    if (!h || !a) continue;
    h.matches_played += 1;
    a.matches_played += 1;
    h.goals_for += m.home_score;
    h.goals_against += m.away_score;
    a.goals_for += m.away_score;
    a.goals_against += m.home_score;
    if (m.home_score > m.away_score) {
      h.wins += 1;
      a.losses += 1;
      h.points += 3;
    } else if (m.home_score < m.away_score) {
      a.wins += 1;
      h.losses += 1;
      a.points += 3;
    } else {
      h.draws += 1;
      a.draws += 1;
      h.points += 1;
      a.points += 1;
    }
  }

  for (const row of Object.values(rowsByCode)) {
    row.goal_diff = row.goals_for - row.goals_against;
  }

  const out: GroupStandings = {};
  for (const [group, codes] of Object.entries(groupMembers)) {
    const rows = codes.map((c) => rowsByCode[c]);
    rows.sort((x, y) => {
      if (y.points !== x.points) return y.points - x.points;
      if (y.goal_diff !== x.goal_diff) return y.goal_diff - x.goal_diff;
      return y.goals_for - x.goals_for;
    });
    rows.forEach((row, idx) => {
      row.advancing = idx < 2;
    });
    out[group] = rows;
  }
  return out;
}
