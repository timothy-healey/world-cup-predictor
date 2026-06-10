import type { ReactNode } from "react";

export type BadgeTone =
  | "correct"
  | "wrong"
  | "secondary"
  | "pending"
  | "neutral";

const TONE: Record<BadgeTone, string> = {
  correct: "bg-correct-soft text-correct",
  wrong: "bg-wrong-soft text-wrong",
  secondary: "bg-secondary-soft text-[#7A5910]",
  pending: "bg-pending-soft text-pending",
  neutral: "bg-surface-sunk text-ink-2",
};

interface Props {
  tone: BadgeTone;
  pulse?: boolean;
  children: ReactNode;
}

export function Badge({ tone, pulse, children }: Props) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-pill px-2.5 py-1 text-xs font-semibold tracking-[0.02em] ${TONE[tone]}`}
    >
      <span
        className={`h-1.5 w-1.5 rounded-pill bg-current ${pulse ? "animate-pulse" : ""}`}
      />
      {children}
    </span>
  );
}
