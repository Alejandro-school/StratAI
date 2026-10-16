import { AlertTriangle, MonitorPlay } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { DEFAULT_LAYERS, getMapConfig, MAP_LEVELS } from "../domain/replayConfig";
import { deriveZThreshold, resolveLevel } from "../domain/replayModel";
import { createTeamReplayProjector } from "../domain/teamReplayProjection";
import { createReplaySoundReviewProjector } from '../domain/replaySoundInformation';
import { useReplayAnnotations } from "../hooks/useReplayAnnotations";
import { useReplayController } from "../hooks/useReplayController";
import { ReplayAnnotationList } from "./ReplayAnnotationList";
import { ReplayCanvas } from "./ReplayCanvas";
import { ReplayHeader } from "./ReplayHeader";
import { ReplayKillFeed } from "./ReplayKillFeed";
import { ReplayRosterPanel } from "./ReplayRosterPanel";
import { ReplayTimeline } from "./ReplayTimeline";
import { ReplayToolbar } from "./ReplayToolbar";
import { ReplayInformationPanel } from "./ReplayInformationPanel";
import { ReplayUtilityInformationPanel } from './ReplayUtilityInformationPanel';
import "../replay2d.css";
import { labelReplayEvents, labelReplayFrame } from "../domain/replayPlayerLabels";

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
  initialPerspective = "observer",
  confirmedVision = null,
  onPerspectiveContext,
  mapImageSources,
  contactPolicy = "calculated",
  onPositionChange,
  playerLabels = null,
}) {
  const rootRef = useRef(null);
  const [layers, setLayers] = useState(DEFAULT_LAYERS);
  const [levelMode, setLevelMode] = useState("auto");
  const [annotationTool, setAnnotationTool] = useState(null);
  const [annotationColor, setAnnotationColor] = useState("#63d7ff");
  const [noteText, setNoteText] = useState("");
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [viewResetToken, setViewResetToken] = useState(0);
  const [perspective, setPerspective] = useState(initialPerspective);
  const teamMode = perspective === "T" || perspective === "CT";
  const controller = useReplayController({
    matchId,
    preloadedData,
    initialRound,
    externalControl,
    onAvailabilityChange,
    perspective: teamMode ? perspective : null,
  });
  const projector = useMemo(() => createTeamReplayProjector(controller.roundData, perspective, confirmedVision, controller.metadata, contactPolicy, { information: true, utilityInformation: true }),
    [controller.roundData, perspective, confirmedVision, controller.metadata, contactPolicy]);
  const context = useMemo(() => projector?.(controller.tick), [projector, controller.tick]);
  const soundReviewProjector = useMemo(() => createReplaySoundReviewProjector(controller.roundData,
    perspective,controller.metadata?.map_name),[controller.roundData,perspective,controller.metadata?.map_name]);
  const soundReviewMarkers = useMemo(() => soundReviewProjector(controller.tick,controller.tickRate),
    [soundReviewProjector,controller.tick,controller.tickRate]);
  useEffect(() => { onPositionChange?.({ round: controller.actualRound, tick: controller.tick, side: perspective }); },
    [controller.actualRound, controller.tick, perspective, onPositionChange]);
  const projectedView = teamMode ? { ...controller, currentFrame: context?.frame || null,
    frames: context?.frames || [], events: context?.events || [], aiAnnotations: [], activeClip: null } : controller;
  const displayFrame = useMemo(() => labelReplayFrame(projectedView.currentFrame, playerLabels), [projectedView.currentFrame, playerLabels]);
  const displayEvents = useMemo(() => labelReplayEvents(projectedView.events, playerLabels), [projectedView.events, playerLabels]);
  const view = { ...projectedView, currentFrame: displayFrame, events: displayEvents };
  const focusPlayerId = view.currentFrame?.players?.some((p) => String(p.steam_id) === String(controller.focusPlayerId))
    ? controller.focusPlayerId : null;
  useEffect(() => { if (teamMode && context) onPerspectiveContext?.(context); }, [teamMode, context, onPerspectiveContext]);
  const persisted = useReplayAnnotations(matchId);
  const config = getMapConfig(controller.metadata);
  const mapName = controller.metadata?.map_name;
  const hasLevels = Boolean(MAP_LEVELS[mapName]);
  const zThreshold = useMemo(
    () => hasLevels ? (teamMode ? (mapName === "de_nuke" ? -500 : 0) : deriveZThreshold(controller.frames, mapName === "de_nuke" ? -500 : undefined)) : 0,
    [controller.frames, hasLevels, mapName, teamMode],
  );
  const automaticLevel = resolveLevel(view.currentFrame?.players, zThreshold, focusPlayerId);
  const activeLevel = levelMode === "auto" ? automaticLevel : levelMode;
  const currentAnnotations = useMemo(() => persisted.annotations.filter((annotation) => (
    annotation.round === controller.actualRound
    && annotation.start_tick <= controller.tick
    && annotation.end_tick >= controller.tick
  )), [controller.actualRound, controller.tick, persisted.annotations]);
  const allAnnotations = teamMode ? [] : [...currentAnnotations, ...(controller.aiAnnotations || [])];
  const ctScore = controller.roundsSummary.slice(0, Math.max(0, controller.roundIndex - 1)).filter((round) => round.winner === "CT").length;
  const tScore = controller.roundsSummary.slice(0, Math.max(0, controller.roundIndex - 1)).filter((round) => round.winner === "T").length;

  useEffect(() => {
    const onFullscreenChange = () => setIsFullscreen(document.fullscreenElement === rootRef.current);
    document.addEventListener("fullscreenchange", onFullscreenChange);
    return () => document.removeEventListener("fullscreenchange", onFullscreenChange);
  }, []);

  useEffect(() => {
    const onKeyDown = (event) => {
      if (event.target.closest("input, select, textarea, button, summary, [contenteditable='true']")) return;
      if (event.code === "Space") {
        event.preventDefault();
        controller.togglePlayback();
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
  }, [controller.seekBySeconds, controller.togglePlayback]);

  if (controller.isLoading) {
    return <div className="replay-container r2-state" role="status"><span className="r2-loader" /><strong>Cargando replay táctica…</strong></div>;
  }
  if (!controller.metadata) return <EmptyState error={controller.error} />;

  return (
    <section className={`replay-container r2-shell ${isFullscreen ? "fullscreen" : ""}`} ref={rootRef} aria-label="Replay 2D táctica">
      <ReplayHeader
        roundIndex={controller.roundIndex}
        rounds={controller.roundsSummary}
        frame={view.currentFrame}
        ctScore={ctScore}
        tScore={tScore}
        onRoundChange={controller.changeRound}
      />
      <div className="r2-perspective-bar">
        <label>Perspectiva
          <select aria-label="Perspectiva del replay" value={perspective} onChange={(event) => {
            setPerspective(event.target.value); setViewResetToken((token) => token + 1);
            controller.setFocusPlayerId(null); controller.setDirectorMode(false);
          }}>
            <option value="observer">Observador · ambos bandos</option>
            <option value="T">Analizar T</option>
            <option value="CT">Analizar CT</option>
          </select>
        </label>
        {teamMode && <span role="status">
          {context?.vision_status === "unavailable" ? "Contactos no disponibles en esta fuente."
            : context?.vision_status === "confirmed_contact_observed" ? "Contacto visual confirmado."
              : "Rivales por contacto calculado de Go · estimación."}
          {" "}Cruces: muertes comunicadas en la simulación.
          {context?.unknown_death_positions > 0 && ` ${context.unknown_death_positions} con ubicación desconocida.`}
        </span>}
        {teamMode && <button type="button" onClick={() => {
          const url = URL.createObjectURL(new Blob([JSON.stringify(context, null, 2)], { type: "application/json" }));
          const link = document.createElement("a"); link.href = url;
          link.download = `StratAI-${perspective}-ronda-${controller.actualRound}-tick-${Math.floor(controller.tick)}.json`;
          link.click(); setTimeout(() => URL.revokeObjectURL(url), 1000);
        }}>Datos de esta vista</button>}
        {teamMode && <ReplayInformationPanel information={context?.information} tick={controller.tick}
          tickRate={controller.tickRate} mapName={mapName} labels={playerLabels || Object.fromEntries(
            (view.currentFrame?.players || []).map(p => [String(p.steam_id), p.name]))} />}
        {teamMode && <ReplayUtilityInformationPanel information={context?.utility_information} tickRate={controller.tickRate} labels={playerLabels} mapName={mapName} />}
        {teamMode && layers.sound && controller.roundData?.sound_information && <span>
          Sonido · punto de la demo al emitir · últimos 1,5 s. Alcance estimado.
        </span>}
      </div>
      <div className="r2-main">
        <main className="r2-map-stage">
          {!teamMode && scenarioContext && (
            <div className="r2-scenario">
              <strong>{scenarioContext.title || "Situación táctica"}</strong>
              <span>{scenarioContext.description || "Analiza la secuencia."}</span>
            </div>
          )}
          <ReplayCanvas
            mapImageSources={mapImageSources}
            mapName={mapName}
            config={config}
            frame={view.currentFrame}
            soundReviewMarkers={soundReviewMarkers}
            frames={view.frames}
            events={view.events}
            tick={controller.tick}
            tickRate={controller.tickRate}
            layers={layers}
            annotations={allAnnotations}
            focusPlayerId={focusPlayerId}
            onFocusPlayer={controller.setFocusPlayerId}
            levelMode={levelMode}
            activeLevel={activeLevel}
            zThreshold={zThreshold}
            annotationTool={teamMode ? null : annotationTool}
            annotationColor={annotationColor}
            noteText={noteText}
            round={controller.actualRound}
            endTick={controller.endTick}
            onCreateAnnotation={persisted.create}
            viewResetToken={viewResetToken}
          />
          <ReplayKillFeed events={view.events} tick={controller.tick} tickRate={controller.tickRate} />
          <ReplayAnnotationList
            annotations={teamMode ? [] : currentAnnotations}
            isSaving={persisted.isSaving}
            error={persisted.error}
            onUpdate={persisted.update}
            onDelete={persisted.remove}
          />
          {controller.isLoadingRound && <div className="r2-round-loading" role="status"><span className="r2-loader" />Cargando ronda…</div>}
        </main>
        <ReplayRosterPanel
          players={view.currentFrame?.players}
          ctScore={ctScore}
          tScore={tScore}
          focusPlayerId={focusPlayerId}
          onSelect={(id) => controller.setFocusPlayerId((current) => current === id ? null : id)}
        />
      </div>
      <footer className="r2-dock">
        <ReplayTimeline
          events={view.events}
          frames={view.frames}
          startTick={controller.startTick}
          endTick={controller.endTick}
          tick={controller.tick}
          progress={controller.progress}
          activeClip={view.activeClip}
          onSeekProgress={controller.seekProgress}
          onSeekEvent={controller.seekEvent}
        />
        <ReplayToolbar
          restricted={teamMode}
          isPlaying={controller.isPlaying}
          playbackRate={controller.playbackRate}
          effectiveRate={controller.effectiveRate}
          directorMode={teamMode ? false : controller.directorMode}
          tick={controller.tick}
          frame={view.currentFrame}
          layers={layers}
          levelMode={levelMode}
          hasLevels={hasLevels}
          annotationTool={teamMode ? null : annotationTool}
          annotationColor={annotationColor}
          noteText={noteText}
          isFullscreen={isFullscreen}
          onTogglePlay={controller.togglePlayback}
          onSeekBySeconds={controller.seekBySeconds}
          onPlaybackRate={controller.setPlaybackRate}
          onDirectorMode={teamMode ? () => {} : controller.setDirectorMode}
          onLayer={(key) => setLayers((current) => ({ ...current, [key]: !current[key] }))}
          onLevel={setLevelMode}
          onAnnotationTool={teamMode ? () => {} : setAnnotationTool}
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
