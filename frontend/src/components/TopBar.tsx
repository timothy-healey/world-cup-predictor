import type { Match } from "../types/api";

export type TabId = "dashboard" | "upcoming" | "past";

interface Props {
  active: TabId;
  onChange: (tab: TabId) => void;
  matches: Match[];
}

const TABS: { id: TabId; label: string }[] = [
  { id: "dashboard", label: "Dashboard" },
  { id: "upcoming", label: "Upcoming" },
  { id: "past", label: "Past results" },
];

function countNext24h(matches: Match[], now = new Date()): number {
  const cutoff = now.getTime() + 24 * 60 * 60 * 1000;
  return matches.filter((m) => {
    const ko = new Date(m.kickoff_utc).getTime();
    return ko >= now.getTime() && ko < cutoff;
  }).length;
}

export function TopBar({ active, onChange, matches }: Props) {
  const now = new Date();
  const dateLabel = new Intl.DateTimeFormat("en-US", {
    weekday: "short",
    month: "short",
    day: "numeric",
  }).format(now);
  const timeLabel = new Intl.DateTimeFormat("en-US", {
    hour: "numeric",
    minute: "2-digit",
    timeZoneName: "short",
  }).format(now);
  const next24 = countNext24h(matches, now);

  return (
    <header className="flex items-end justify-between border-b bg-bg px-7 pt-5">
      <div className="flex items-center gap-3">
        <div className="h-[34px] w-1.5 rounded-sm bg-primary" />
        <div>
          <h1 className="font-display text-2xl font-extrabold uppercase tracking-display text-ink">
            FIFA World Cup 2026 Predictor
          </h1>
          <div className="mb-4 mt-1 text-xs text-ink-3">
            {`${dateLabel} · ${timeLabel} · ${next24} ${next24 === 1 ? "match" : "matches"} in next 24h`}
          </div>
        </div>
      </div>
      <nav className="flex gap-1" aria-label="Dashboard sections">
        {TABS.map((t) => {
          const isActive = active === t.id;
          return (
            <button
              key={t.id}
              onClick={() => onChange(t.id)}
              aria-current={isActive ? "page" : undefined}
              className={`-mb-px border-b-2 px-4 py-2.5 text-sm font-medium transition-colors focus:outline-none focus-visible:shadow-focus ${
                isActive
                  ? "border-ink text-ink"
                  : "border-transparent text-ink-3 hover:text-ink"
              }`}
            >
              {t.label}
            </button>
          );
        })}
      </nav>
    </header>
  );
}
