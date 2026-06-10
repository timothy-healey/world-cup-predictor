import { describe, expect, it } from "vitest";
import { computeGroupStandings } from "../src/lib/groups";
import type { Match, Team } from "../src/types/api";

const team = (code: string, group: string): Team => ({
  code,
  name: code,
  group_id: group,
  flag_url: "",
  fifa_ranking: 0,
  manager_name: "",
  pre_tournament_form: "",
  fixture_src_id: "",
});

const match = (id: string, h: string, a: string, hs: number | null, as_: number | null): Match => ({
  id,
  home_team_code: h,
  away_team_code: a,
  kickoff_utc: "2026-06-12T00:00:00Z",
  stage: "group",
  venue: "",
  home_score: hs,
  away_score: as_,
  predictions: [],
});

const teams = [team("MEX", "A"), team("CAN", "A"), team("ECU", "A"), team("AUS", "A")];

describe("computeGroupStandings", () => {
  it("returns groups keyed by group_id with one row per team", () => {
    const standings = computeGroupStandings(teams, [
      match("m1", "MEX", "CAN", 2, 1),
      match("m2", "ECU", "AUS", 0, 1),
    ]);
    expect(standings).toHaveProperty("A");
    expect(standings["A"]).toHaveLength(4);
  });

  it("orders by Pts then GD then GF", () => {
    const standings = computeGroupStandings(teams, [
      match("m1", "MEX", "CAN", 2, 1),
      match("m2", "ECU", "AUS", 0, 1),
      match("m3", "MEX", "ECU", 3, 0),
      match("m4", "CAN", "AUS", 1, 0),
    ]);
    expect(standings["A"].map((s) => s.team_code)).toEqual(["MEX", "CAN", "AUS", "ECU"]);
  });

  it("ignores group-stage matches with no result yet", () => {
    const standings = computeGroupStandings(teams, [
      match("m1", "MEX", "CAN", 2, 1),
      match("m2", "ECU", "AUS", null, null),
    ]);
    expect(standings["A"].find((s) => s.team_code === "ECU")?.matches_played).toBe(0);
  });

  it("ignores non-group-stage matches", () => {
    const knockout = { ...match("m1", "MEX", "NED", 2, 1), stage: "round-of-32" as const };
    const standings = computeGroupStandings(teams, [knockout]);
    expect(standings["A"].find((s) => s.team_code === "MEX")?.matches_played).toBe(0);
  });

  it("marks top two as advancing", () => {
    const standings = computeGroupStandings(teams, [
      match("m1", "MEX", "CAN", 2, 1),
      match("m2", "ECU", "AUS", 0, 1),
      match("m3", "MEX", "ECU", 3, 0),
      match("m4", "CAN", "AUS", 1, 0),
    ]);
    const advancing = standings["A"].filter((s) => s.advancing);
    expect(advancing.map((s) => s.team_code)).toEqual(["MEX", "CAN"]);
  });
});
