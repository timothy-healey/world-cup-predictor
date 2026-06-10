import { useCallback, useEffect, useRef, useState } from "react";
import type { ExportPayload } from "../types/api";

const POLL_MS = 60_000;
const SOURCE = "/predictions.json";

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

export function useData() {
  const [state, setState] = useState<DataState>({ data: null, error: null, loading: true });
  const cancelled = useRef(false);

  const fetchOnce = useCallback(async () => {
    try {
      const res = await fetch(`${SOURCE}?t=${Date.now()}`, { cache: "no-store" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json: ExportPayload = await res.json();
      if (!cancelled.current) {
        setState({ data: filterProductionVariants(json), error: null, loading: false });
      }
    } catch (e) {
      if (!cancelled.current) {
        setState((prev) => ({
          data: prev.data,
          error: e instanceof Error ? e.message : String(e),
          loading: false,
        }));
      }
    }
  }, []);

  useEffect(() => {
    cancelled.current = false;
    fetchOnce();
    const id = setInterval(fetchOnce, POLL_MS);
    return () => {
      cancelled.current = true;
      clearInterval(id);
    };
  }, [fetchOnce]);

  return { ...state, refresh: fetchOnce };
}
