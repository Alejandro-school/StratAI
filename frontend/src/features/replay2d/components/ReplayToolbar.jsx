import {
  ArrowRight,
  Circle,
  FastForward,
  Layers3,
  Maximize2,
  MousePointer2,
  Pause,
  Pencil,
  Play,
  RotateCcw,
  StickyNote,
} from "lucide-react";
import { useState } from "react";
import { ANNOTATION_COLORS } from "../domain/replayConfig";
import { formatClock } from "../domain/replayModel";

const LAYER_LABELS = {
  names: "Nombres",
  fov: "FOV",
  shots: "Disparos",
  trajectories: "Trayectorias",
  utility: "Utilidad",
  deaths: "Muertes",
  annotations: "Anotaciones",
};

export function ReplayToolbar({
  isPlaying,
  playbackRate,
  effectiveRate,
  directorMode,
  tick,
  frame,
  layers,
  levelMode,
  hasLevels,
  annotationTool,
  annotationColor,
  noteText,
  isFullscreen,
  onTogglePlay,
  onSeekBySeconds,
  onPlaybackRate,
  onDirectorMode,
  onLayer,
  onLevel,
  onAnnotationTool,
  onAnnotationColor,
  onNoteText,
  onResetView,
  onFullscreen,
}) {
  const [layersOpen, setLayersOpen] = useState(false);
  return (
    <div className="r2-toolbar">
      <div className="r2-transport">
        <button type="button" className="primary" onClick={onTogglePlay} aria-label={isPlaying ? "Pausar" : "Reproducir"}>
          {isPlaying ? <Pause size={17} /> : <Play size={17} />}
        </button>
        <button type="button" onClick={() => onSeekBySeconds(-5)} aria-label="Retroceder 5 segundos">−5</button>
        <button type="button" onClick={() => onSeekBySeconds(5)} aria-label="Avanzar 5 segundos">+5</button>
        <select value={playbackRate} onChange={(event) => onPlaybackRate(Number(event.target.value))} aria-label="Velocidad">
          {[0.25, 0.5, 0.75, 1, 1.5, 2, 4].map((rate) => <option value={rate} key={rate}>{rate}×</option>)}
        </select>
        <button type="button" className={directorMode ? "active" : ""} onClick={() => onDirectorMode(!directorMode)} aria-pressed={directorMode} title="Acelera pausas y reduce la velocidad cerca de eventos">
          <FastForward size={15} /><span>Director</span>
        </button>
        {directorMode && <span className="r2-effective-rate">{effectiveRate.toFixed(2)}×</span>}
      </div>
      <div className="r2-clock-readout">
        <strong>{formatClock(frame?.time_remaining)}</strong>
        <span>Tick {Math.round(tick)}</span>
      </div>
      <div className="r2-analysis-tools">
        <div className="r2-tool-group" role="group" aria-label="Herramientas de anotación">
          {[
            [null, MousePointer2, "Seleccionar"],
            ["arrow", ArrowRight, "Flecha"],
            ["circle", Circle, "Área"],
            ["freehand", Pencil, "Trazo libre"],
            ["note", StickyNote, "Nota"],
          ].map(([tool, Icon, label]) => (
            <button type="button" key={label} className={annotationTool === tool ? "active" : ""} onClick={() => onAnnotationTool(tool)} aria-label={label} aria-pressed={annotationTool === tool}>
              <Icon size={15} />
            </button>
          ))}
        </div>
        {annotationTool && (
          <div className="r2-colors" aria-label="Color de anotación">
            {ANNOTATION_COLORS.map((color) => (
              <button type="button" key={color} style={{ "--swatch": color }} className={annotationColor === color ? "active" : ""} onClick={() => onAnnotationColor(color)} aria-label={`Color ${color}`} />
            ))}
          </div>
        )}
        {annotationTool === "note" && (
          <input value={noteText} onChange={(event) => onNoteText(event.target.value)} placeholder="Texto de la nota" aria-label="Texto de la nota" maxLength={160} />
        )}
        {hasLevels && (
          <select value={levelMode} onChange={(event) => onLevel(event.target.value)} aria-label="Nivel del mapa">
            <option value="auto">Nivel auto</option>
            <option value="upper">Superior</option>
            <option value="lower">Inferior</option>
          </select>
        )}
        <div className="r2-layer-menu">
          <button type="button" className={layersOpen ? "active" : ""} onClick={() => setLayersOpen((open) => !open)} aria-expanded={layersOpen}>
            <Layers3 size={15} /><span>Capas</span>
          </button>
          {layersOpen && (
            <div className="r2-layer-popover">
              {Object.entries(LAYER_LABELS).map(([key, label]) => (
                <label key={key}><input type="checkbox" checked={layers[key]} onChange={() => onLayer(key)} />{label}</label>
              ))}
            </div>
          )}
        </div>
        <button type="button" onClick={onResetView} aria-label="Restablecer vista"><RotateCcw size={15} /></button>
        <button type="button" onClick={onFullscreen} aria-label={isFullscreen ? "Salir de pantalla completa" : "Pantalla completa"}><Maximize2 size={15} /></button>
      </div>
    </div>
  );
}
