import { afterEach, beforeEach, describe, expect, it, vi, type MockInstance } from "vitest";
import { formatTimestamp, trace, traceFetch } from "../src/lib/trace";

describe("formatTimestamp", () => {
  it("zero-pads hours, minutes, seconds, and milliseconds", () => {
    const d = new Date(2026, 5, 11, 3, 4, 5, 7);
    expect(formatTimestamp(d)).toBe("03:04:05.007");
  });
  it("renders three-digit milliseconds verbatim", () => {
    const d = new Date(2026, 5, 11, 22, 33, 44, 999);
    expect(formatTimestamp(d)).toBe("22:33:44.999");
  });
});

describe("trace", () => {
  let logSpy: MockInstance<Parameters<typeof console.log>, void>;
  let warnSpy: MockInstance<Parameters<typeof console.warn>, void>;
  let errorSpy: MockInstance<Parameters<typeof console.error>, void>;

  beforeEach(() => {
    logSpy = vi.spyOn(console, "log").mockImplementation(() => {});
    warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    logSpy.mockRestore();
    warnSpy.mockRestore();
    errorSpy.mockRestore();
  });

  it("tags log output with a namespaced %c prefix", () => {
    const t = trace("predict");
    t.log("hello", { match: "abc" });
    expect(logSpy).toHaveBeenCalledOnce();
    const call = logSpy.mock.calls[0];
    expect(call[0]).toMatch(/^%c\[wcp:predict]%c \d{2}:\d{2}:\d{2}\.\d{3} hello$/);
    // first style + reset style + the user-provided extra arg
    expect(call.slice(3)).toEqual([{ match: "abc" }]);
  });

  it("routes warn and error through the right console channel", () => {
    const t = trace("data");
    t.warn("slow");
    t.error("boom");
    expect(warnSpy).toHaveBeenCalledOnce();
    expect(errorSpy).toHaveBeenCalledOnce();
  });

  it("time() emits start + done lines and returns the resolved value", async () => {
    const t = trace("api");
    const got = await t.time("widget", async () => 42);
    expect(got).toBe(42);
    expect(logSpy).toHaveBeenCalledTimes(2);
    expect(logSpy.mock.calls[0][0]).toMatch(/widget → start/);
    expect(logSpy.mock.calls[1][0]).toMatch(/widget ✓ done in \d+ms/);
  });

  it("time() logs an error line and rethrows when the body throws", async () => {
    const t = trace("api");
    await expect(
      t.time("widget", async () => {
        throw new Error("nope");
      }),
    ).rejects.toThrow("nope");
    expect(logSpy.mock.calls[0][0]).toMatch(/widget → start/);
    expect(errorSpy).toHaveBeenCalledOnce();
    expect(errorSpy.mock.calls[0][0]).toMatch(/widget ✗ failed after \d+ms/);
  });
});

describe("traceFetch", () => {
  let logSpy: MockInstance<Parameters<typeof console.log>, void>;
  let warnSpy: MockInstance<Parameters<typeof console.warn>, void>;
  let errorSpy: MockInstance<Parameters<typeof console.error>, void>;
  let fetchMock: MockInstance;

  beforeEach(() => {
    logSpy = vi.spyOn(console, "log").mockImplementation(() => {});
    warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    logSpy.mockRestore();
    warnSpy.mockRestore();
    errorSpy.mockRestore();
    fetchMock?.mockRestore();
  });

  it("logs a 2xx response on console.log", async () => {
    fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(new Response("ok", { status: 200, statusText: "OK" }));
    const res = await traceFetch("/api/health");
    expect(res.ok).toBe(true);
    expect(fetchMock).toHaveBeenCalledWith("/api/health", {});
    expect(logSpy.mock.calls.length).toBe(2);
    expect(logSpy.mock.calls[0][0]).toMatch(/→ GET \/api\/health$/);
    expect(logSpy.mock.calls[1][0]).toMatch(/✓ GET \/api\/health → 200 OK \(\d+ms\)/);
  });

  it("logs a non-2xx response on console.warn", async () => {
    fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(new Response("bad", { status: 503, statusText: "Service Unavailable" }));
    const res = await traceFetch("/api/predict?match=42", { method: "POST", ns: "predict" });
    expect(res.ok).toBe(false);
    expect(warnSpy).toHaveBeenCalledOnce();
    expect(warnSpy.mock.calls[0][0]).toMatch(
      /✗ POST \/api\/predict\?match=42 → 503 Service Unavailable \(\d+ms\)/,
    );
  });

  it("logs and rethrows network failures", async () => {
    fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockRejectedValue(new TypeError("Failed to fetch"));
    await expect(
      traceFetch("/api/predict?match=42", { method: "POST" }),
    ).rejects.toThrow("Failed to fetch");
    expect(errorSpy).toHaveBeenCalledOnce();
    expect(errorSpy.mock.calls[0][0]).toMatch(/✗ POST \/api\/predict\?match=42 → network error/);
  });
});
