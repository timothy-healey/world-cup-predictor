import { useState } from "react";
import { TopBar, type TabId } from "./components/TopBar";
import { Dashboard } from "./pages/Dashboard";
import { Upcoming } from "./pages/Upcoming";
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
      {tab === "dashboard" && (
        <Dashboard
          data={data}
          onPredict={predict}
          predictDisabled={predictDisabled}
          predictDisabledReason={reason}
        />
      )}
      {tab === "upcoming" && (
        <Upcoming
          data={data}
          onPredict={predict}
          predictDisabled={predictDisabled}
          predictDisabledReason={reason}
        />
      )}
      {tab === "past" && <div className="p-7 text-sm text-ink-3">Past results tab coming next…</div>}
    </>
  );
}
