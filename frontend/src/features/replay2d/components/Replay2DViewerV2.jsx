import { AlertTriangle, MonitorPlay } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { DEFAULT_LAYERS, getMapConfig, MAP_LEVELS } from "../domain/replayConfig";
import { deriveZThreshold, resolveLevel } from "../domain/replayModel";
import { useReplayAnnotations } from "../hooks/useReplayAnnotations";
import { useReplayController } from "../hooks/useReplayController";
import { ReplayAnnotationList } from "./ReplayAnnotationList";
import { ReplayCanvas } from "./ReplayCanvas";
import { ReplayHeader } from "./ReplayHeader";
import { ReplayKillFeed } from "./ReplayKillFeed";
import { ReplayRosterPanel } from "./ReplayRosterPanel";
import { ReplayTimeline } from "./ReplayTimeline";
import { ReplayToolbar } from "./ReplayToolbar";
import "../replay2d.css";

function EmptyState({ error }) {
  return (
    <div className="replay-container r2-state" role="status">
      <div className="r2-state-icon">{error ? <AlertTriangle size={24} /> : <MonitorPlay size={24} />}</div>
      <strong>{error ? "No se pudo cargar la replay" : "Replay 2D no disponible"}</strong>
      <p>{error || "Esta demo aún no contiene fotogramas reproducibles."}</p>
    </div>
  );
}

export default function Replay2DViewerV2({
  matchId,
  replayData: preloadedData,
  initialRound = 1,
  externalControl = null,
  scenarioContext = null,
  fitMode: _fitMode = "focus",
  compactTeams: _compactTeams = false,
  onAvailabilityChange,
}) {
  const rootRef = useRef(null);
  const [layers, setLayers] = useState(DEFAULT_LAYERS);
  const [levelMode, setLevelMode] = useState("auto");
  const [annotationTool, setAnnotationTool] = useState(null);
  const [annotationColor, setAnnotationColor] = useState("#63d7ff");
  const [noteText, setNoteText] = useState("");
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [viewResetToken, setViewResetToken] = useState(0);
  const controller = useReplayController({
    matchId,
    preloadedData,
    initialRound,
    externalControl,
    onAvailabilityChange,
  });
  const persisted = useReplayAnnotations(matchId);
  const config = getMapConfig(controller.metadata);
  const mapName = controller.metadata?.map_name;
  const hasLevels = Boolean(MAP_LEVELS[mapName]);
  const zThreshold = useMemo(
    () => hasLevels ? deriveZThreshold(controller.frames, mapName === "de_nuke" ? -500 : undefined) : 0,
    [controller.frames, hasLevels, mapName],
  );
  const automaticLevel = resolveLevel(controller.currentFrame?.players, zThreshold, controller.focusPlayerId);
  const activeLevel = levelMode === "auto" ? automaticLevel : levelMode;
  const currentAnnotations = useMemo(() => persisted.annotations.filter((annotation) => (
    annotation.round === controller.actualRound
    && annotation.start_tick <= controller.tick
    && annotation.end_tick >= controller.tick
  )), [controller.actualRound, controller.tick, persisted.annotations]);
  const allAnnotations = [...currentAnnotations, ...(controller.aiAnnotations || [])];
  const ctScore = controller.roundsSummary.slice(0, Math.max(0, controller.roundIndex - 1)).filter((round) => round.winner === "CT").length;
  const tScore = controller.roundsSummary.slice(0, Math.max(0, controller.roundIndex - 1)).filter((round) => round.winner === "T").length;

  useEffect(() => {
    const onFullscreenChange = () => setIsFullscreen(document.fullscreenElement === rootRef.current);
    document.addEventListener("fullscreenchange", onFullscreenChange);
    return () => document.removeEventListener("fullscreenchange", onFullscreenChange);
  }, []);

  useEffect(() => {
    const onKeyDown = (event) => {
      if (event.target.closest("input, select, textarea, button, [contenteditable='true']")) return;
      if (event.code === "Space") {
        event.preventDefault();
        controller.setIsPlaying((playing) => !playing);
      } else if (event.key === "ArrowLeft") {
        event.preventDefault();
        controller.seekBySeconds(event.shiftKey ? -1 : -5);
      } else if (event.key === "ArrowRight") {
        event.preventDefault();
        controller.seekBySeconds(event.shiftKey ? 1 : 5);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [controller.seekBySeconds, controller.setIsPlaying]);

  if (controller.isLoading) {
    return <div className="replay-container r2-state" role="status"><span className="r2-loader" /><strong>Cargando replay táctica…</strong></div>;
  }
  if (!controller.metadata) return <EmptyState error={controller.error} />;

  return (
    <section className={`replay-container r2-shell ${isFullscreen ? "fullscreen" : ""}`} ref={rootRef} aria-label="Replay 2D táctica">
      <ReplayHeader
        roundIndex={controller.roundIndex}
        actualRound={controller.actualRound}
        rounds={controller.roundsSummary}
        frame={controller.currentFrame}
        ctScore={ctScore}
        tScore={tScore}
        onRoundChange={controller.changeRound}
      />
      <div className="r2-main">
        <main className="r2-map-stage">
          {scenarioContext && (
            <div className="r2-scenario">
              <strong>{scenarioContext.title || "Situación táctica"}</strong>
              <span>{scenarioContext.description || "Analiza la secuencia."}</span>
            </div>
          )}
          <ReplayCanvas
            mapName={mapName}
            config={config}
            frame={controller.currentFrame}
            frames={controller.frames}
            events={controller.events}
            tick={controller.tick}
            tickRate={controller.tickRate}
            layers={layers}
            annotations={allAnnotations}
            focusPlayerId={controller.focusPlayerId}
            onFocusPlayer={controller.setFocusPlayerId}
            levelMode={levelMode}
            activeLevel={activeLevel}
            zThreshold={zThreshold}
            annotationTool={annotationTool}
            annotationColor={annotationColor}
            noteText={noteText}
            round={controller.actualRound}
            endTick={controller.endTick}
            onCreateAnnotation={persisted.create}
            viewResetToken={viewResetToken}
          />
          <ReplayKillFeed events={controller.events} tick={controller.tick} tickRate={controller.tickRate} />
          <ReplayAnnotationList
            annotations={currentAnnotations}
            isSaving={persisted.isSaving}
            error={persisted.error}
            onUpdate={persisted.update}
            onDelete={persisted.remove}
          />
          {controller.isLoadingRound && <div className="r2-round-loading" role="status"><span className="r2-loader" />Cargando ronda…</div>}
        </main>
        <ReplayRosterPanel
          players={controller.currentFrame?.players}
          ctScore={ctScore}
          tScore={tScore}
          focusPlayerId={controller.focusPlayerId}
          onSelect={(id) => controller.setFocusPlayerId((current) => current === id ? null : id)}
        />
      </div>
      <footer className="r2-dock">
        <ReplayTimeline
          events={controller.events}
          frames={controller.frames}
          startTick={controller.startTick}
          endTick={controller.endTick}
          tick={controller.tick}
          progress={controller.progress}
          activeClip={controller.activeClip}
          onSeekProgress={controller.seekProgress}
          onSeekEvent={controller.seekEvent}
        />
        <ReplayToolbar
          isPlaying={controller.isPlaying}
          playbackRate={controller.playbackRate}
          effectiveRate={controller.effectiveRate}
          directorMode={controller.directorMode}
          tick={controller.tick}
          frame={controller.currentFrame}
          layers={layers}
          levelMode={levelMode}
          hasLevels={hasLevels}
          annotationTool={annotationTool}
          annotationColor={annotationColor}
          noteText={noteText}
          isFullscreen={isFullscreen}
          onTogglePlay={() => controller.setIsPlaying((playing) => !playing)}
          onSeekBySeconds={controller.seekBySeconds}
          onPlaybackRate={controller.setPlaybackRate}
          onDirectorMode={controller.setDirectorMode}
          onLayer={(key) => setLayers((current) => ({ ...current, [key]: !current[key] }))}
          onLevel={setLevelMode}
          onAnnotationTool={setAnnotationTool}
          onAnnotationColor={setAnnotationColor}
          onNoteText={setNoteText}
          onResetView={() => setViewResetToken((token) => token + 1)}
          onFullscreen={async () => {
            if (document.fullscreenElement) await document.exitFullscreen();
            else await rootRef.current?.requestFullscreen();
          }}
        />
      </footer>
    </section>
  );
}
