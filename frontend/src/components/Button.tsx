import type { ButtonHTMLAttributes, ReactNode } from "react";

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "primary" | "ghost";
  children: ReactNode;
}

export function Button({ variant = "primary", className = "", children, ...rest }: Props) {
  const base =
    "inline-flex items-center gap-1.5 rounded px-4 py-2 text-sm font-semibold transition-colors focus:outline-none focus-visible:shadow-focus disabled:cursor-not-allowed disabled:opacity-50";
  const styles =
    variant === "primary"
      ? "bg-ink text-secondary hover:bg-ink-2"
      : "border border-strong bg-surface text-ink hover:bg-surface-sunk";
  return (
    <button {...rest} className={`${base} ${styles} ${className}`.trim()}>
      {children}
    </button>
  );
}
