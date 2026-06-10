import { describe, expect, it } from "vitest";
import { buildTeamNameLookup } from "../src/lib/teams";
import type { Team } from "../src/types/api";

function team(code: string, name: string): Team {
  return {
    code,
    name,
    group_id: "A",
    flag_url: "",
    fifa_ranking: 0,
    manager_name: "",
    pre_tournament_form: "",
    fixture_src_id: "",
  };
}

describe("buildTeamNameLookup", () => {
  const lookup = buildTeamNameLookup([
    team("BRA", "Brazil"),
    team("ESP", "Spain"),
  ]);

  it("returns the full name when the code matches", () => {
    expect(lookup("BRA")).toBe("Brazil");
    expect(lookup("ESP")).toBe("Spain");
  });

  it("returns the code unchanged when no team matches", () => {
    expect(lookup("XYZ")).toBe("XYZ");
  });

  it("returns 'Draw' for the special 'draw' value", () => {
    expect(lookup("draw")).toBe("Draw");
  });

  it("is case-sensitive on the code (matches the data convention)", () => {
    expect(lookup("bra")).toBe("bra");
  });
});
