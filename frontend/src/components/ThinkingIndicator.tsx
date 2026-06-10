import { useEffect, useState } from "react";

const GERUNDS = [
  "Pontificating",
  "Vibing",
  "Ruminating",
  "Marinating",
  "Mulling",
  "Deliberating",
  "Scheming",
  "Concocting",
  "Divining",
  "Manifesting",
  "Discombobulating",
  "Schlepping",
  "Hypothesising",
  "Sleuthing",
  "Consulting the oracle",
  "Reading the tea leaves",
  "Reticulating splines",
  "Crunching xG",
  "Polling the back four",
];

// Oscillates through increasing weights — small dot, light teardrop asterisk,
// heavier teardrop, six-point star, eight-spoke asterisk, four balloon-spoke —
// then back down, producing a soft pulsing twinkle.
const SPARKLE_FRAMES = ["·", "✻", "✽", "✶", "✳", "✢", "✳", "✶", "✽", "✻"];

const WORD_ROTATE_MS = 2200;
const SPARKLE_TICK_MS = 220;

function formatElapsed(ms: number | null | undefined): string | null {
  if (ms == null) return null;
  const totalSeconds = Math.max(0, Math.floor(ms / 1000));
  const m = Math.floor(totalSeconds / 60);
  const s = totalSeconds % 60;
  return `${m}:${String(s).padStart(2, "0")}`;
}

interface Props {
  elapsedMs?: number | null;
}

export function ThinkingIndicator({ elapsedMs }: Props) {
  const [wordIdx, setWordIdx] = useState(() => Math.floor(Math.random() * GERUNDS.length));
  const [frame, setFrame] = useState(0);

  useEffect(() => {
    const id = setInterval(() => {
      setWordIdx(
        (i) => (i + 1 + Math.floor(Math.random() * (GERUNDS.length - 1))) % GERUNDS.length,
      );
    }, WORD_ROTATE_MS);
    return () => clearInterval(id);
  }, []);

  useEffect(() => {
    const id = setInterval(() => {
      setFrame((f) => (f + 1) % SPARKLE_FRAMES.length);
    }, SPARKLE_TICK_MS);
    return () => clearInterval(id);
  }, []);

  const word = GERUNDS[wordIdx];
  const elapsed = formatElapsed(elapsedMs);

  return (
    <span className="inline-flex items-baseline gap-1.5" role="status" aria-live="polite">
      <span
        aria-hidden="true"
        className="inline-block w-3 text-center font-mono leading-none"
      >
        {SPARKLE_FRAMES[frame]}
      </span>
      <span key={word} className="animate-word-in">
        {word}…
      </span>
      {elapsed && (
        <span className="font-mono tabular-nums text-2xs text-ink-3">{elapsed}</span>
      )}
    </span>
  );
}
