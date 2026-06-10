import { useCallback, useEffect, useRef, useState } from "react";

interface PredictState {
  busy: boolean;
  serverUp: boolean | null; // null = unknown until first probe
  lastError: string | null;
}

export function usePredict(onSuccess: () => void) {
  const [state, setState] = useState<PredictState>({
    busy: false,
    serverUp: null,
    lastError: null,
  });
  const cancelled = useRef(false);

  const probe = useCallback(async () => {
    try {
      const res = await fetch("/api/health", { cache: "no-store" });
      if (!cancelled.current) setState((s) => ({ ...s, serverUp: res.ok }));
    } catch {
      if (!cancelled.current) setState((s) => ({ ...s, serverUp: false }));
    }
  }, []);

  useEffect(() => {
    cancelled.current = false;
    probe();
    const id = setInterval(probe, 30_000);
    return () => {
      cancelled.current = true;
      clearInterval(id);
    };
  }, [probe]);

  const predict = useCallback(
    async (matchID: string) => {
      setState((s) => ({ ...s, busy: true, lastError: null }));
      try {
        const res = await fetch(`/api/predict?match=${encodeURIComponent(matchID)}`, {
          method: "POST",
        });
        if (!res.ok) {
          const body = await res.text();
          throw new Error(`HTTP ${res.status}: ${body || res.statusText}`);
        }
        setState({ busy: false, serverUp: true, lastError: null });
        onSuccess();
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        setState({
          busy: false,
          serverUp: msg.includes("Failed to fetch") ? false : true,
          lastError: msg,
        });
      }
    },
    [onSuccess],
  );

  return { ...state, predict };
}
