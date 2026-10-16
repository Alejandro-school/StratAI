import { ChevronLeft, ChevronRight } from "lucide-react";
import { formatClock } from "../domain/replayModel";

export function ReplayHeader({
  roundIndex,
  rounds,
  frame,
  ctScore,
  tScore,
  onRoundChange,
}) {
  const bomb = frame?.bomb;
  const carrier = frame?.players?.find((p) => String(p.steam_id) === String(bomb?.carrier_id));
  const objectiveLabel = { carried: "transportado", dropped: "caído", planting: "plantando",
    planted: "plantado", defusing: "desactivando", defused: "desactivado", exploded: "detonado", resolved: "resuelto" }[bomb?.state] || "desconocido";
  const roundContext = frame?.round_context;
  const phase = roundContext ? ({ ended: `FINALIZADA${roundContext.winner ? ` · ${roundContext.winner}` : ''}`,
    preplant: 'LIVE', planted: 'POST-PLANT', freezetime: 'FREEZE' }[roundContext.phase] || 'FASE DESCONOCIDA')
    : bomb?.objective_phase === "resolved" ? "FINALIZADA"
    : bomb?.state === "planted" || bomb?.state === "defusing"
    ? "POST-PLANT"
    : (frame?.time_remaining || 0) > 115 ? "FREEZE" : "LIVE";
  return (
    <header className="r2-header">
      <div className="r2-round-switcher">
        <button type="button" aria-label="Ronda anterior" onClick={() => onRoundChange(roundIndex - 1)} disabled={roundIndex <= 1}>
          <ChevronLeft size={16} />
        </button>
        <span className="r2-kicker">Ronda</span>
        <strong>{roundIndex}</strong>
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
        <span className="r2-objective-status">C4 {objectiveLabel}{carrier ? ` · ${carrier.name}` : ""}
          <small>{bomb?.objective_phase === "resolved" ? (bomb.was_planted_this_round ? "Hubo plantado" : "Objetivo cerrado")
            : bomb?.position_status === "observed" ? "Ubicación observada"
              : bomb?.last_observed_position ? "Última muestra indicada en el mapa" : "Ubicación desconocida"}</small>
        </span>
        <span className={`r2-live-pill ${phase === "LIVE" ? "active" : ""}`}>{phase}</span>
        <time aria-label={roundContext && frame?.time_remaining == null ? 'Reloj no disponible' : 'Tiempo restante'}
          className={Number.isFinite(frame?.time_remaining) && frame.time_remaining <= 10 ? "danger" : ""}>
          {roundContext && frame?.time_remaining == null ? '—' : formatClock(frame?.time_remaining)}
        </time>
      </div>
    </header>
  );
}
