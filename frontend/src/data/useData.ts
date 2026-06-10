import { useCallback, useEffect, useRef, useState } from "react";
import type { ExportPayload } from "../types/api";

const POLL_MS = 60_000;
const SOURCE = "/predictions.json";

interface DataState {
  data: ExportPayload | null;
  error: string | null;
  loading: boolean;
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
        setState({ data: json, error: null, loading: false });
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
