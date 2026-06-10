import { useCallback, useEffect, useRef, useState } from "react";
import type { ExportPayload } from "../types/api";
import { trace, traceFetch } from "../lib/trace";

const POLL_MS = 60_000;
const SOURCE = "/predictions.json";

const t = trace("data");

interface DataState {
  data: ExportPayload | null;
  error: string | null;
  loading: boolean;
}

// Strip non-"full" predictions before exposing the payload. The backend
// `predictions.variant` column reserves "no-odds"/"no-news"/etc. for a future
// ablation experiment harness. Dashboard accuracy stats, badges, and lists must
// not mix those rows with production predictions. Filtering once here is the
// single chokepoint — every downstream consumer is automatically safe.
function filterProductionVariants(payload: ExportPayload): ExportPayload {
  return {
    ...payload,
    matches: (payload.matches ?? []).map((m) => ({
      ...m,
      // Backend should emit `[]` (see store/export.go), but be defensive against
      // an old JSON on disk where matches with no predictions serialized as null.
      predictions: (m.predictions ?? []).filter((p) => p.variant === "full"),
    })),
  };
}

function summarize(payload: ExportPayload): Record<string, number> {
  let predictions = 0;
  for (const m of payload.matches ?? []) predictions += (m.predictions ?? []).length;
  return {
    teams: (payload.teams ?? []).length,
    matches: (payload.matches ?? []).length,
    predictions,
  };
}

export function useData() {
  const [state, setState] = useState<DataState>({ data: null, error: null, loading: true });
  const cancelled = useRef(false);
  const tickRef = useRef(0);

  const fetchOnce = useCallback(async () => {
    const tick = ++tickRef.current;
    const url = `${SOURCE}?t=${Date.now()}`;
    t.log(`tick #${tick} fetching predictions.json`);
    try {
      const res = await traceFetch(url, { cache: "no-store", ns: "data", label: `tick #${tick}` });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const raw: ExportPayload = await res.json();
      const filtered = filterProductionVariants(raw);
      t.log(`tick #${tick} payload loaded`, summarize(filtered));
      if (!cancelled.current) {
        setState({ data: filtered, error: null, loading: false });
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      t.error(`tick #${tick} failed: ${msg}`);
      if (!cancelled.current) {
        setState((prev) => ({
          data: prev.data,
          error: msg,
          loading: false,
        }));
      }
    }
  }, []);

  useEffect(() => {
    cancelled.current = false;
    t.log(`mounted; polling every ${POLL_MS}ms`);
    fetchOnce();
    const id = setInterval(fetchOnce, POLL_MS);
    return () => {
      cancelled.current = true;
      clearInterval(id);
      t.log("unmounted; polling stopped");
    };
  }, [fetchOnce]);

  return { ...state, refresh: fetchOnce };
}
