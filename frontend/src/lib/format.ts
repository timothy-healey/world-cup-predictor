const DAY_MS = 24 * 60 * 60 * 1000;

export function formatKickoff(iso: string, timeZone?: string): string {
  const d = new Date(iso);
  const tz = timeZone ?? Intl.DateTimeFormat().resolvedOptions().timeZone;
  const day = new Intl.DateTimeFormat("en-US", {
    weekday: "short",
    month: "short",
    day: "numeric",
    timeZone: tz,
  }).format(d);
  const time = new Intl.DateTimeFormat("en-US", {
    hour: "numeric",
    minute: "2-digit",
    timeZone: tz,
  }).format(d);
  return `${day} · ${time}`;
}

export function formatCountdown(kickoff: Date, now: Date = new Date()): string {
  const ms = kickoff.getTime() - now.getTime();
  if (ms < -60_000) return "started";
  if (ms < 60_000) return "kicking off";
  const totalMin = Math.floor(ms / 60_000);
  const hours = Math.floor(totalMin / 60);
  const minutes = totalMin % 60;
  if (hours > 0) return `in ${hours}h ${minutes}m`;
  return `in ${minutes}m`;
}

export function formatRelativeDay(d: Date, now: Date = new Date()): string {
  const dayStart = (x: Date) =>
    new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
  const diffDays = Math.round((dayStart(d) - dayStart(now)) / DAY_MS);
  if (diffDays === 0) return "today";
  if (diffDays === 1) return "tomorrow";
  if (diffDays === -1) return "yesterday";
  return new Intl.DateTimeFormat("en-US", {
    weekday: "short",
    month: "short",
    day: "numeric",
  }).format(d);
}

export function formatScore(home: number | null, away: number | null): string | null {
  if (home === null || away === null) return null;
  return `${home}-${away}`;
}
