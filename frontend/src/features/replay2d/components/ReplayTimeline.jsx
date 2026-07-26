import { Bomb, Crosshair, Flame, Sparkles } from "lucide-react";
import { useMemo, useRef, useState } from "react";
import { clamp, formatClock, interpolateFrame } from "../domain/replayModel";

const LANES = [
  { id: "combat", label: "Combate" },
  { id: "utility", label: "Utilidad" },
  { id: "objective", label: "Objetivo" },
];

function MarkerIcon({ event }) {
  if (event.lane === "combat") return <Crosshair size={11} />;
  if (event.lane === "objective") return <Bomb size={11} />;
  if (event.subtype === "inferno") return <Flame size={11} />;
  return <Sparkles size={11} />;
}

function markerTitle(event) {
  if (event.type === "kill") {
    return `${event.killer_name || "Jugador"} → ${event.victim_name || "Jugador"} · ${event.weapon || "arma"}`;
  }
  return `${event.label}${event.site ? ` · Site ${event.site}` : ""}`;
}

export function ReplayTimeline({
  events,
  frames,
  startTick,
  endTick,
  tick,
  progress,
  activeClip,
  onSeekProgress,
  onSeekEvent,
}) {
  const trackRef = useRef(null);
  const [dragging, setDragging] = useState(false);
  const [hover, setHover] = useState(null);
  const span = Math.max(1, endTick - startTick);
  const grouped = useMemo(() => Object.fromEntries(
    LANES.map((lane) => [lane.id, events.filter((event) => event.lane === lane.id)]),
  ), [events]);
  const objectiveSegment = useMemo(() => {
    const plant = events.find((event) => event.type === "bomb_plant");
    const end = events.find((event) => event.type === "bomb_defuse" || event.type === "bomb_explode");
    return plant ? { from: (plant.tick - startTick) / span, to: ((end?.tick || endTick) - startTick) / span } : null;
  }, [endTick, events, span, startTick]);

  const update = (clientX) => {
    const rect = trackRef.current?.getBoundingClientRect();
    if (!rect) return;
    onSeekProgress(clamp((clientX - rect.left) / rect.width, 0, 1));
  };

  const onPointerDown = (event) => {
    setDragging(true);
    event.currentTarget.setPointerCapture(event.pointerId);
    update(event.clientX);
  };

  const onPointerMove = (event) => {
    const rect = trackRef.current?.getBoundingClientRect();
    if (!rect) return;
    const value = clamp((event.clientX - rect.left) / rect.width, 0, 1);
    const previewTick = startTick + value * span;
    setHover({ x: value, tick: previewTick, frame: interpolateFrame(frames, previewTick) });
    if (dragging) update(event.clientX);
  };

  return (
    <div className="r2-timeline">
      <div
        className="r2-timeline-tracks"
        ref={trackRef}
        role="slider"
        tabIndex={0}
        aria-label="Timeline de la ronda"
        aria-valuemin={startTick}
        aria-valuemax={endTick}
        aria-valuenow={Math.round(tick)}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={(event) => {
          setDragging(false);
          event.currentTarget.releasePointerCapture(event.pointerId);
        }}
        onPointerLeave={() => {
          if (!dragging) setHover(null);
        }}
        onKeyDown={(event) => {
          if (event.key === "Home") onSeekProgress(0);
          else if (event.key === "End") onSeekProgress(1);
          else if (event.key === "ArrowLeft") onSeekProgress(progress - (event.shiftKey ? 0.01 : 0.04));
          else if (event.key === "ArrowRight") onSeekProgress(progress + (event.shiftKey ? 0.01 : 0.04));
          else return;
          event.preventDefault();
        }}
      >
        {activeClip && (
          <span
            className="r2-ai-range"
            style={{
              left: `${clamp((activeClip.startTick - startTick) / span, 0, 1) * 100}%`,
              width: `${clamp((activeClip.endTick - activeClip.startTick) / span, 0, 1) * 100}%`,
            }}
            title="Hallazgo de IA activo"
          />
        )}
        {LANES.map((lane) => (
          <div className={`r2-timeline-lane ${lane.id}`} key={lane.id}>
            <span className="r2-lane-label">{lane.label}</span>
            {lane.id === "objective" && objectiveSegment && (
              <span
                className="r2-objective-range"
                style={{ left: `${objectiveSegment.from * 100}%`, width: `${(objectiveSegment.to - objectiveSegment.from) * 100}%` }}
              />
            )}
            {grouped[lane.id].map((event) => (
              <button
                type="button"
                className={`r2-event-marker ${event.subtype || event.type}`}
                key={event.id}
                style={{ left: `${clamp((event.tick - startTick) / span, 0, 1) * 100}%` }}
                onPointerDown={(pointerEvent) => pointerEvent.stopPropagation()}
                onClick={() => onSeekEvent(event)}
                title={markerTitle(event)}
                aria-label={`Ir a ${markerTitle(event)}`}
              >
                <MarkerIcon event={event} />
              </button>
            ))}
          </div>
        ))}
        <span className="r2-playhead" style={{ left: `${progress * 100}%` }} />
        {hover && (
          <span className="r2-hover-preview" style={{ left: `${hover.x * 100}%` }}>
            <strong>{formatClock(hover.frame?.time_remaining)}</strong>
            <small>Tick {Math.round(hover.tick)}</small>
          </span>
        )}
      </div>
    </div>
  );
}
