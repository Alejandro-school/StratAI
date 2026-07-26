import { ChevronLeft, ChevronRight } from "lucide-react";
import { formatClock } from "../domain/replayModel";

export function ReplayHeader({
  roundIndex,
  actualRound,
  rounds,
  frame,
  ctScore,
  tScore,
  onRoundChange,
}) {
  const phase = frame?.bomb?.state === "planted" || frame?.bomb?.state === "defusing"
    ? "POST-PLANT"
    : (frame?.time_remaining || 0) > 115 ? "FREEZE" : "LIVE";
  return (
    <header className="r2-header">
      <div className="r2-round-switcher">
        <button type="button" aria-label="Ronda anterior" onClick={() => onRoundChange(roundIndex - 1)} disabled={roundIndex <= 1}>
          <ChevronLeft size={16} />
        </button>
        <span className="r2-kicker">Ronda</span>
        <strong>{actualRound}</strong>
        <span className="r2-round-total">/ {rounds.length}</span>
        <button type="button" aria-label="Ronda siguiente" onClick={() => onRoundChange(roundIndex + 1)} disabled={roundIndex >= rounds.length}>
          <ChevronRight size={16} />
        </button>
      </div>
      <div className="r2-score" aria-label={`Marcador CT ${ctScore}, T ${tScore}`}>
        <span className="ct">CT</span>
        <strong>{ctScore}</strong>
        <i>:</i>
        <strong>{tScore}</strong>
        <span className="t">T</span>
      </div>
      <div className="r2-round-state">
        <span className={`r2-live-pill ${phase === "LIVE" ? "active" : ""}`}>{phase}</span>
        <time className={(frame?.time_remaining || 0) <= 10 ? "danger" : ""}>
          {formatClock(frame?.time_remaining)}
        </time>
      </div>
    </header>
  );
}
