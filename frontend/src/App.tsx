import { useEffect, useMemo, useState } from "react";
import { TopBar, type TabId } from "./components/TopBar";
import { Dashboard } from "./pages/Dashboard";
import { Upcoming } from "./pages/Upcoming";
import { Past } from "./pages/Past";
import { ThinkingIndicator } from "./components/ThinkingIndicator";
import { useData } from "./data/useData";
import { usePredict } from "./data/usePredict";
import { trace } from "./lib/trace";

const tBoot = trace("boot");

export function App() {
  const [tab, setTab] = useState<TabId>("dashboard");
  const { data, error, loading, refresh } = useData();
  const { predict, busy, serverUp, lastError, activeMatchId, elapsedMs } = usePredict(refresh);

  useEffect(() => {
    tBoot.log("dashboard mounted");
  }, []);

  useEffect(() => {
    if (lastError) tBoot.warn(`last predict error surfaced to UI: ${lastError}`);
  }, [lastError]);

  const activeMatch = useMemo(() => {
    if (!activeMatchId || !data) return null;
    return data.matches.find((m) => m.id === activeMatchId) ?? null;
  }, [activeMatchId, data]);

  if (loading && !data) {
    return <div className="p-8 text-sm text-ink-3">Loading predictions…</div>;
  }
  if (!data) {
    return (
      <div className="p-8 text-sm text-wrong">
        Could not load predictions.json. {error ?? "Is wcp serve running?"}
      </div>
    );
  }

  const predictDisabled = serverUp === false || busy;

  let banner: { tone: "pending" | "wrong" | "busy"; label: string; body: React.ReactNode } | null = null;
  if (serverUp === false) {
    banner = {
      tone: "pending",
      label: "On-demand predictions paused",
      body: "wcp serve is not running. Start it with `wcp serve` to enable on-demand predictions.",
    };
  } else if (busy) {
    const matchLabel = activeMatch
      ? `${activeMatch.home_team_code} vs ${activeMatch.away_team_code}`
      : `match ${activeMatchId}`;
    banner = {
      tone: "busy",
      label: "Prediction in progress",
      body: (
        <span className="inline-flex items-center gap-2">
          <ThinkingIndicator elapsedMs={elapsedMs} />
          <span className="text-ink-3">·</span>
          <span>{matchLabel} — fetchers + Claude typically take 3–5 minutes.</span>
        </span>
      ),
    };
  } else if (lastError) {
    banner = {
      tone: "wrong",
      label: "Last prediction failed",
      body: lastError,
    };
  }

  return (
    <>
      <TopBar active={tab} onChange={setTab} matches={data.matches} />
      {banner && <BannerStrip {...banner} />}
      {tab === "dashboard" && (
        <Dashboard
          data={data}
          onPredict={predict}
          predictDisabled={predictDisabled}
          activeMatchId={activeMatchId}
          elapsedMs={elapsedMs}
        />
      )}
      {tab === "upcoming" && (
        <Upcoming
          data={data}
          onPredict={predict}
          predictDisabled={predictDisabled}
          activeMatchId={activeMatchId}
          elapsedMs={elapsedMs}
        />
      )}
      {tab === "past" && <Past data={data} />}
    </>
  );
}

interface BannerProps {
  tone: "pending" | "wrong" | "busy";
  label: string;
  body: React.ReactNode;
}

function BannerStrip({ tone, label, body }: BannerProps) {
  const styles =
    tone === "wrong"
      ? "border-wrong/30 bg-wrong-soft text-wrong"
      : tone === "busy"
        ? "border-primary/30 bg-primary-soft text-primary"
        : "border-pending/30 bg-pending-soft text-pending";

  return (
    <div role="status" className={`border-b px-7 py-2.5 text-xs ${styles}`}>
      <span className="font-semibold uppercase tracking-label-mid">{label}</span>
      <span className="ml-2 normal-case tracking-normal">{body}</span>
    </div>
  );
}
