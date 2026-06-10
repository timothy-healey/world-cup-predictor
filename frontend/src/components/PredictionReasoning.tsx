import { parseReasoning } from "../lib/reasoning";

interface Props {
  reasoning: string;
}

export function PredictionReasoning({ reasoning }: Props) {
  const lines = parseReasoning(reasoning);
  if (lines.length === 0) return null;

  return (
    <div>
      <div className="mb-2.5 text-xs font-semibold uppercase tracking-label text-ink-3">
        Reasoning
      </div>
      <ul className="list-disc max-w-[62ch] pl-5 text-sm leading-relaxed text-ink">
        {lines.map((line, idx) => (
          <li key={idx} className="mb-1.5">
            {line}
          </li>
        ))}
      </ul>
    </div>
  );
}
