import { describe, expect, it } from "vitest";
import { formatDuration, okCount, pillTone } from "../src/lib/traceFormat";
import type { TraceEntry } from "../src/types/api";

function entry(kind: TraceEntry["kind"], ok: boolean): TraceEntry {
  return {
    kind,
    started_at: "2026-06-25T17:30:00.000Z",
    duration_ms: 0,
    ok,
    error: ok ? "" : "x",
    snippet: "",
  };
}

describe("formatDuration", () => {
  it("renders sub-second values in ms", () => {
    expect(formatDuration(0)).toBe("0ms");
    expect(formatDuration(380)).toBe("380ms");
    expect(formatDuration(999)).toBe("999ms");
  });
  it("renders values >= 1000ms in seconds with one decimal", () => {
    expect(formatDuration(1000)).toBe("1.0s");
    expect(formatDuration(3614)).toBe("3.6s");
    expect(formatDuration(91240)).toBe("91.2s");
  });
});

describe("okCount", () => {
  it("counts ok entries", () => {
    expect(okCount([entry("odds", true), entry("news", false), entry("lineup", true), entry("context", true), entry("predict", true)])).toBe(4);
  });
  it("returns 0 for empty array", () => {
    expect(okCount([])).toBe(0);
  });
});

describe("pillTone", () => {
  it("returns 'ok' when all entries are healthy", () => {
    expect(pillTone(5, 5)).toBe("ok");
  });
  it("returns 'degraded' when any entry failed", () => {
    expect(pillTone(4, 5)).toBe("degraded");
    expect(pillTone(0, 5)).toBe("degraded");
  });
  it("returns 'degraded' for an empty or short trace as a defensive default", () => {
    expect(pillTone(0, 0)).toBe("degraded");
  });
});
