import { useCallback, useEffect, useRef, useState } from "react";
import { trace, traceFetch } from "../lib/trace";

const PROBE_INTERVAL_MS = 30_000;
const ELAPSED_TICK_MS = 1_000;

const t = trace("predict");
const th = trace("health");

interface PredictState {
  busy: boolean;
  serverUp: boolean | null; // null = unknown until first probe
  lastError: string | null;
  activeMatchId: string | null;
  startedAt: number | null;
}

export interface UsePredictResult extends PredictState {
  /** Milliseconds since the current predict call started; 0 when idle. */
  elapsedMs: number;
  predict: (matchID: string) => Promise<void>;
}

export function usePredict(onSuccess: () => void): UsePredictResult {
  const [state, setState] = useState<PredictState>({
    busy: false,
    serverUp: null,
    lastError: null,
    activeMatchId: null,
    startedAt: null,
  });
  const [elapsedMs, setElapsedMs] = useState(0);
  const cancelled = useRef(false);

  const probe = useCallback(async () => {
    try {
      const res = await traceFetch("/api/health", {
        cache: "no-store",
        ns: "health",
        label: "probe",
      });
      if (!cancelled.current) {
        setState((s) => {
          if (s.serverUp !== res.ok) {
            th.log(`server ${res.ok ? "reachable" : "unreachable"} (HTTP ${res.status})`);
          }
          return { ...s, serverUp: res.ok };
        });
      }
    } catch {
      if (!cancelled.current) {
        setState((s) => {
          if (s.serverUp !== false) th.warn("server unreachable (network error)");
          return { ...s, serverUp: false };
        });
      }
    }
  }, []);

  useEffect(() => {
    cancelled.current = false;
    th.log(`health probe every ${PROBE_INTERVAL_MS}ms`);
    probe();
    const id = setInterval(probe, PROBE_INTERVAL_MS);
    return () => {
      cancelled.current = true;
      clearInterval(id);
    };
  }, [probe]);

  // Tick the elapsed counter once a second while a predict is in flight.
  useEffect(() => {
    if (!state.busy || state.startedAt == null) {
      setElapsedMs(0);
      return;
    }
    const startedAt = state.startedAt;
    setElapsedMs(Date.now() - startedAt);
    const id = setInterval(() => setElapsedMs(Date.now() - startedAt), ELAPSED_TICK_MS);
    return () => clearInterval(id);
  }, [state.busy, state.startedAt]);

  const predict = useCallback(
    async (matchID: string) => {
      const startedAt = Date.now();
      t.log(`predict requested for match=${matchID}`);
      setState((s) => ({
        ...s,
        busy: true,
        lastError: null,
        activeMatchId: matchID,
        startedAt,
      }));
      try {
        const res = await traceFetch(`/api/predict?match=${encodeURIComponent(matchID)}`, {
          method: "POST",
          ns: "predict",
          label: `match=${matchID}`,
        });
        if (!res.ok) {
          const body = await res.text();
          throw new Error(`HTTP ${res.status}: ${body || res.statusText}`);
        }
        const elapsed = Date.now() - startedAt;
        t.log(`predict ✓ match=${matchID} completed in ${elapsed}ms; refreshing data`);
        setState({
          busy: false,
          serverUp: true,
          lastError: null,
          activeMatchId: null,
          startedAt: null,
        });
        onSuccess();
      } catch (e) {
        const elapsed = Date.now() - startedAt;
        const msg = e instanceof Error ? e.message : String(e);
        const networkDown = msg.includes("Failed to fetch");
        t.error(
          `predict ✗ match=${matchID} failed after ${elapsed}ms${networkDown ? " (server unreachable)" : ""}: ${msg}`,
        );
        setState({
          busy: false,
          serverUp: networkDown ? false : true,
          lastError: msg,
          activeMatchId: null,
          startedAt: null,
        });
      }
    },
    [onSuccess],
  );

  return { ...state, elapsedMs, predict };
}
