import { describe, expect, it } from "vitest";
import { nextExpandedId } from "../src/lib/expand";

describe("nextExpandedId", () => {
  it("opens the clicked id when nothing is expanded", () => {
    expect(nextExpandedId(null, "match-1")).toBe("match-1");
  });

  it("collapses when the clicked id is already expanded", () => {
    expect(nextExpandedId("match-1", "match-1")).toBeNull();
  });

  it("switches to the clicked id when a different one is expanded", () => {
    expect(nextExpandedId("match-1", "match-2")).toBe("match-2");
  });
});
