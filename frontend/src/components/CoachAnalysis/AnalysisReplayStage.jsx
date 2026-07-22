import React, { useEffect, useState } from 'react';
import { AlertTriangle, Check, ClipboardPlus, Clock3, MonitorPlay, PlayCircle } from 'lucide-react';
import Replay2DViewer from '../Stats/Replay2DViewer';

const AnalysisReplayStage = ({
  match,
  matchId,
  mapImage,
  selectedEvidence,
  status,
  isPlaying,
  activeClip,
  onPlayEvidence,
  onTogglePlan,
  isSavedToPlan
}) => {
  const [isReplayAvailable, setIsReplayAvailable] = useState(false);
  const mapName = (match?.map_name || match?.map || 'de_dust2').replace('de_', '').toUpperCase();
  const score = `${match?.team_score ?? '?'}-${match?.opponent_score ?? '?'}`;
  const hasPlayableEvidence = isReplayAvailable && Boolean(selectedEvidence?.interaction);
  const isCurrentClip = isPlaying && activeClip?.startTick === selectedEvidence?.interaction?.startTick;
  useEffect(() => {
    setIsReplayAvailable(false);
  }, [matchId]);

  return (
    <section className="analysis-replay-stage">
      <div
        className="analysis-stage-top"
        style={{ backgroundImage: `linear-gradient(90deg, rgba(5,10,18,0.9), rgba(5,10,18,0.48)), url(${mapImage})` }}
      >
        <div>
          <span className="analysis-kicker">Replay táctico 2D</span>
          <h2>{mapName} · {score}</h2>
        </div>
        <div className="analysis-stage-actions">
          <div className={`analysis-live-badge ${status}`}>
            <span />
            {status === 'complete' ? 'Revisión lista' : 'Analizando demo'}
          </div>
        </div>
      </div>

      <div className="analysis-replay-frame">
        {matchId ? (
          <Replay2DViewer
            matchId={matchId}
            initialRound={1}
            fitMode="contain"
            onAvailabilityChange={setIsReplayAvailable}
          />
        ) : (
          <div className="analysis-replay-empty">
            <MonitorPlay size={34} />
            <span>Selecciona una partida para abrir el replay táctico.</span>
          </div>
        )}
      </div>

      <div className="analysis-evidence-focus">
        {selectedEvidence ? (
          <>
            <div className="focus-icon">
              <AlertTriangle size={18} />
            </div>
            <div className="focus-copy">
              <span>Foco actual · Ronda {selectedEvidence.round}</span>
              <strong>{selectedEvidence.title}</strong>
              <p>{selectedEvidence.summary}</p>
              <p className="focus-impact"><b>Impacto:</b> {selectedEvidence.impact}</p>
              <p className="focus-recommendation"><b>Próximo paso:</b> {selectedEvidence.recommendation}</p>
            </div>
            <div className="focus-actions">
              <button
                type="button"
                className="focus-play-btn"
                onClick={() => onPlayEvidence(selectedEvidence)}
                disabled={!hasPlayableEvidence}
              >
                {hasPlayableEvidence ? <PlayCircle size={15} aria-hidden="true" /> : <Clock3 size={15} aria-hidden="true" />}
                {isCurrentClip ? 'Reproduciendo' : hasPlayableEvidence ? 'Ver momento' : 'Replay pendiente'}
              </button>
              <button
                type="button"
                className={`focus-plan-btn ${isSavedToPlan ? 'saved' : ''}`}
                onClick={() => onTogglePlan(selectedEvidence.id)}
                aria-pressed={isSavedToPlan}
              >
                {isSavedToPlan ? <Check size={15} aria-hidden="true" /> : <ClipboardPlus size={15} aria-hidden="true" />}
                {isSavedToPlan ? 'Añadido al plan' : 'Añadir al plan'}
              </button>
            </div>
          </>
        ) : (
          <div className="focus-copy empty">
            <strong>Esperando evidencias</strong>
            <p>La IA irá fijando momentos concretos conforme avance la revisión.</p>
          </div>
        )}
      </div>
    </section>
  );
};

export default AnalysisReplayStage;
