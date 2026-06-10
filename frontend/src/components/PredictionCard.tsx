import type { Match } from "../types/api";
import { PredictionBody } from "./PredictionBody";

interface Props {
  match: Match;
  teamName: (code: string) => string;
  groupLabel?: string;
  onPredict?: (matchID: string) => void;
  predictDisabled?: boolean;
  activeMatchId?: string | null;
  elapsedMs?: number;
}

export function PredictionCard({
  match,
  teamName,
  groupLabel,
  onPredict,
  predictDisabled,
  activeMatchId,
  elapsedMs,
}: Props) {
  return (
    <article className="mb-3.5 rounded-lg border bg-surface px-6 py-5 sm:px-8 sm:py-6">
      <PredictionBody
        match={match}
        teamName={teamName}
        variant="upcoming"
        groupLabel={groupLabel}
        onPredict={onPredict}
        predictDisabled={predictDisabled}
        activeMatchId={activeMatchId}
        elapsedMs={elapsedMs}
      />
    </article>
  );
}
