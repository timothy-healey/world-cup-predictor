import { describe, expect, it } from "vitest";
import {
  formatKickoff,
  formatCountdown,
  formatRelativeDay,
  formatScore,
  formatTimestamp,
} from "../src/lib/format";

describe("formatKickoff", () => {
  it("renders a UTC ISO timestamp in the user's local zone", () => {
    // 2026-06-25 11:00 UTC → 21:00 AEST (UTC+10) on Thu Jun 25
    const out = formatKickoff("2026-06-25T11:00:00Z", "Australia/Sydney");
    expect(out).toBe("Thu, Jun 25 · 9:00 PM");
  });
});

describe("formatTimestamp", () => {
  it("renders short month/day + local time", () => {
    // 2026-06-25 11:00 UTC → 21:00 in Sydney on Jun 25
    const out = formatTimestamp("2026-06-25T11:00:00Z", "Australia/Sydney");
    expect(out).toBe("Jun 25, 9:00 PM");
  });
});

describe("formatCountdown", () => {
  it("renders days, hours, and minutes when more than a day away", () => {
    const now = new Date("2026-06-22T10:00:00Z");
    const ko = new Date("2026-06-25T14:30:00Z");
    expect(formatCountdown(ko, now)).toBe("in 3d 4h 30m");
  });
  it("renders hours and minutes when more than an hour away", () => {
    const now = new Date("2026-06-24T22:12:00Z");
    const ko = new Date("2026-06-25T05:00:00Z");
    expect(formatCountdown(ko, now)).toBe("in 6h 48m");
  });
  it("renders minutes only when under an hour", () => {
    const now = new Date("2026-06-25T04:35:00Z");
    const ko = new Date("2026-06-25T05:00:00Z");
    expect(formatCountdown(ko, now)).toBe("in 25m");
  });
  it("renders 'kicking off' when within one minute", () => {
    const now = new Date("2026-06-25T04:59:30Z");
    const ko = new Date("2026-06-25T05:00:00Z");
    expect(formatCountdown(ko, now)).toBe("kicking off");
  });
  it("renders 'started' for past kickoffs", () => {
    const now = new Date("2026-06-25T06:00:00Z");
    const ko = new Date("2026-06-25T05:00:00Z");
    expect(formatCountdown(ko, now)).toBe("started");
  });
  it("renders exactly 1 day cleanly", () => {
    const now = new Date("2026-06-24T05:00:00Z");
    const ko = new Date("2026-06-25T05:00:00Z");
    expect(formatCountdown(ko, now)).toBe("in 1d 0h 0m");
  });
});

describe("formatRelativeDay", () => {
  it("returns 'today' for same day", () => {
    // Construct in local time directly so the test is timezone-agnostic.
    const now = new Date(2026, 5, 25, 10, 0);
    const d = new Date(2026, 5, 25, 22, 0);
    expect(formatRelativeDay(d, now)).toBe("today");
  });
  it("returns 'tomorrow' for next day", () => {
    const now = new Date(2026, 5, 25, 10, 0);
    const d = new Date(2026, 5, 26, 10, 0);
    expect(formatRelativeDay(d, now)).toBe("tomorrow");
  });
});

describe("formatScore", () => {
  it("renders 'a-b' from numeric inputs", () => {
    expect(formatScore(2, 1)).toBe("2-1");
  });
  it("returns null when either side is null", () => {
    expect(formatScore(null, 1)).toBeNull();
    expect(formatScore(2, null)).toBeNull();
  });
});
