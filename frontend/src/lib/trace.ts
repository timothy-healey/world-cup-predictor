// Namespaced console tracing for the dashboard.
//
// Everything goes through `console.{log,warn,error}` with a `%c[wcp:<ns>]`
// prefix so the browser console can filter by namespace. Use `traceFetch` in
// place of `fetch` to capture method/URL/status/duration/errors automatically.
//
// Namespaces in use: `boot`, `data`, `predict`, `health`, `api`.

const COLORS: Record<string, string> = {
  boot: "#f59e0b",
  data: "#10b981",
  predict: "#a855f7",
  health: "#84cc16",
  api: "#0ea5e9",
};

const FALLBACK_COLOR = "#64748b";

function colorFor(ns: string): string {
  return COLORS[ns] ?? FALLBACK_COLOR;
}

export function formatTimestamp(d: Date): string {
  const hh = String(d.getHours()).padStart(2, "0");
  const mm = String(d.getMinutes()).padStart(2, "0");
  const ss = String(d.getSeconds()).padStart(2, "0");
  const ms = String(d.getMilliseconds()).padStart(3, "0");
  return `${hh}:${mm}:${ss}.${ms}`;
}

export interface Tracer {
  log: (msg: string, ...args: unknown[]) => void;
  warn: (msg: string, ...args: unknown[]) => void;
  error: (msg: string, ...args: unknown[]) => void;
  time: <T>(label: string, fn: () => Promise<T>) => Promise<T>;
}

type ConsoleKind = "log" | "warn" | "error";

export function trace(ns: string): Tracer {
  const tag = `%c[wcp:${ns}]%c`;
  const tagStyle = `color: ${colorFor(ns)}; font-weight: 600;`;
  const restStyle = "color: inherit; font-weight: normal;";

  const emit =
    (kind: ConsoleKind) =>
    (msg: string, ...args: unknown[]) => {
      // eslint-disable-next-line no-console
      console[kind](`${tag} ${formatTimestamp(new Date())} ${msg}`, tagStyle, restStyle, ...args);
    };

  const log = emit("log");
  const warn = emit("warn");
  const error = emit("error");

  const time = async <T,>(label: string, fn: () => Promise<T>): Promise<T> => {
    const start = performance.now();
    log(`${label} → start`);
    try {
      const result = await fn();
      const ms = Math.round(performance.now() - start);
      log(`${label} ✓ done in ${ms}ms`);
      return result;
    } catch (e) {
      const ms = Math.round(performance.now() - start);
      error(`${label} ✗ failed after ${ms}ms`, e);
      throw e;
    }
  };

  return { log, warn, error, time };
}

export interface TraceFetchOptions extends RequestInit {
  /** Tracing namespace; defaults to `api`. */
  ns?: string;
  /** Optional label appended to the URL in trace output. */
  label?: string;
}

export async function traceFetch(
  input: string,
  init: TraceFetchOptions = {},
): Promise<Response> {
  const { ns = "api", label, ...rest } = init;
  const t = trace(ns);
  const method = (rest.method ?? "GET").toUpperCase();
  const tail = label ? ` (${label})` : "";
  const start = performance.now();

  t.log(`→ ${method} ${input}${tail}`);

  let res: Response;
  try {
    res = await fetch(input, rest);
  } catch (e) {
    const ms = Math.round(performance.now() - start);
    t.error(`✗ ${method} ${input}${tail} → network error after ${ms}ms`, e);
    throw e;
  }

  const ms = Math.round(performance.now() - start);
  const line = `${method} ${input}${tail} → ${res.status} ${res.statusText} (${ms}ms)`;
  if (res.ok) {
    t.log(`✓ ${line}`);
  } else {
    t.warn(`✗ ${line}`);
  }
  return res;
}
