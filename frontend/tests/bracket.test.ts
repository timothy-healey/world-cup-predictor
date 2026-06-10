import { describe, expect, it } from "vitest";
import { buildBracket, type BracketColumn } from "../src/lib/bracket";
import type { Match } from "../src/types/api";

const m = (id: string, stage: Match["stage"], h: string, a: string, hs: number | null = null, as_: number | null = null): Match => ({
  id,
  home_team_code: h,
  away_team_code: a,
  kickoff_utc: "2026-07-01T20:00:00Z",
  stage,
  venue: "",
  home_score: hs,
  away_score: as_,
  predictions: [],
});

describe("buildBracket", () => {
  it("groups matches by stage in R32→Final order", () => {
    const b = buildBracket([
      m("k1", "round-of-32", "MEX", "DEN", 2, 0),
      m("k2", "round-of-16", "MEX", "NED"),
      m("k3", "qf", "MEX", "FRA"),
      m("k4", "sf", "MEX", "ESP"),
      m("k5", "final", "MEX", "ARG"),
    ]);
    expect(b.columns.map((c: BracketColumn) => c.stage)).toEqual([
      "round-of-32", "round-of-16", "qf", "sf", "final",
    ]);
    expect(b.columns[0].matches).toHaveLength(1);
    expect(b.columns[4].matches[0].id).toBe("k5");
  });

  it("isolates the third-place playoff", () => {
    const b = buildBracket([
      m("k1", "sf", "MEX", "ESP"),
      m("k2", "third-place", "ITA", "FRA"),
    ]);
    expect(b.thirdPlace?.id).toBe("k2");
    expect(b.columns.find((c) => c.stage === "sf")?.matches).toHaveLength(1);
  });

  it("ignores group-stage matches", () => {
    const b = buildBracket([m("g1", "group", "MEX", "CAN")]);
    expect(b.columns.every((c) => c.matches.length === 0)).toBe(true);
    expect(b.thirdPlace).toBeNull();
  });

  it("sorts matches within a column by kickoff time", () => {
    const a = m("a", "round-of-32", "X", "Y");
    a.kickoff_utc = "2026-07-01T20:00:00Z";
    const b = m("b", "round-of-32", "P", "Q");
    b.kickoff_utc = "2026-07-01T16:00:00Z";
    const out = buildBracket([a, b]);
    expect(out.columns[0].matches.map((x) => x.id)).toEqual(["b", "a"]);
  });
});
