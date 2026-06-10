import { useState } from "react";
import { TopBar, type TabId } from "./components/TopBar";
import { Dashboard } from "./pages/Dashboard";
import { Upcoming } from "./pages/Upcoming";
import { Past } from "./pages/Past";
import { useData } from "./data/useData";
import { usePredict } from "./data/usePredict";

export function App() {
  const [tab, setTab] = useState<TabId>("dashboard");
  const { data, error, loading, refresh } = useData();
  const { predict, busy, serverUp } = usePredict(refresh);

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
  const reason =
    serverUp === false
      ? "wcp serve is not running. Start it with `wcp serve` to enable on-demand predictions."
      : busy
        ? "A prediction is already running."
        : undefined;

  return (
    <>
      <TopBar active={tab} onChange={setTab} matches={data.matches} />
      {reason && (
        <div
          role="status"
          className="border-b border-pending/30 bg-pending-soft px-7 py-2.5 text-xs text-pending"
        >
          <span className="font-semibold uppercase tracking-label-mid">
            On-demand predictions paused
          </span>
          <span className="ml-2 normal-case tracking-normal">{reason}</span>
        </div>
      )}
      {tab === "dashboard" && (
        <Dashboard
          data={data}
          onPredict={predict}
          predictDisabled={predictDisabled}
        />
      )}
      {tab === "upcoming" && (
        <Upcoming
          data={data}
          onPredict={predict}
          predictDisabled={predictDisabled}
        />
      )}
      {tab === "past" && <Past data={data} />}
    </>
  );
}
