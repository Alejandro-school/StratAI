import { useCallback, useEffect, useMemo, useState } from 'react';
import { mockAiCoachResponse } from '../mocks/aiCoachResponses';

export const ANALYSIS_PHASES = [
  {
    id: 'ingest',
    title: 'Cargando demo',
    detail: 'Leyendo metadatos, mapa y rondas disponibles.'
  },
  {
    id: 'rounds',
    title: 'Rondas clave',
    detail: 'Buscando swings de economía, entradas y cierres.'
  },
  {
    id: 'duels',
    title: 'Duelos y trades',
    detail: 'Comparando exposición, refrags y timing.'
  },
  {
    id: 'utility',
    title: 'Utilidad',
    detail: 'Revisando flashes, humos y entrada a site.'
  },
  {
    id: 'plan',
    title: 'Plan de mejora',
    detail: 'Priorizando problemas por impacto.'
  }
];

const PHASE_MS = 900;

const getMatchMap = (match) => (match?.map_name || match?.map || 'de_dust2').replace('de_', '');

const getMatchScore = (match) => `${match?.team_score ?? '?'}-${match?.opponent_score ?? '?'}`;

const getMatchResult = (match) => {
  const isWin = match?.result === 'W' || match?.result === 'victory';
  return isWin ? 'victoria' : 'derrota';
};

const buildEvidenceList = (match) => {
  const mapName = getMatchMap(match).toUpperCase();
  const score = getMatchScore(match);
  const result = getMatchResult(match);

  return [
    {
      id: 'mid-exposure',
      type: 'error',
      severity: 'critical',
      round: 6,
      title: 'Peek aislado sin trade',
      summary: `En ${mapName}, una salida a medio dejó al jugador expuesto antes de que llegara la utilidad del equipo.`,
      impact: 'Convierte una ronda estable en inferioridad temprana y obliga rotaciones reactivas.',
      recommendation: 'Esperar humo o flash de apoyo, o pedir doble contacto antes de asomar.',
      interaction: { ...mockAiCoachResponse.interaction, round: 6 }
    },
    {
      id: 'force-buy-window',
      type: 'economy',
      severity: 'high',
      round: 13,
      title: 'Compra forzada con poco valor',
      summary: `Tras el ${score}, el equipo entra en una ventana económica frágil y fuerza sin utilidad suficiente.`,
      impact: 'Reduce opciones en las dos rondas siguientes y entrega control de ritmo al rival.',
      recommendation: 'Agrupar decisión de compra: eco completa o force con plan de utilidad definido.'
    },
    {
      id: 'late-utility',
      type: 'utility',
      severity: 'high',
      round: 15,
      title: 'Flash tardía en entrada',
      summary: 'La utilidad llega después del primer contacto y no protege al entry.',
      impact: 'El rival recibe el duelo limpio y puede reposicionarse antes del segundo jugador.',
      recommendation: 'Sincronizar flash 0.8s antes del swing y bloquear línea larga con humo.'
    },
    {
      id: 'trade-distance',
      type: 'duel',
      severity: 'medium',
      round: 18,
      title: 'Distancia de trade excesiva',
      summary: `En una ronda de ${result}, el segundo jugador queda demasiado lejos para castigar la kill inicial.`,
      impact: 'El duelo perdido no genera refrag y rompe la estructura del ataque.',
      recommendation: 'Mantener distancia de trade corta en entradas o dividir roles de bait/refrag.'
    },
    {
      id: 'rotation-timing',
      type: 'timing',
      severity: 'medium',
      round: 21,
      title: 'Rotación sin información',
      summary: 'La rotación abandona zona fuerte antes de confirmar presencia rival.',
      impact: 'Abre un timing de entrada y deja al anchor sin apoyo real.',
      recommendation: 'Esperar contacto, sonido o utilidad rival antes de rotar por lectura.'
    }
  ];
};

const getVisibleEvidence = (evidenceList, activePhaseIndex, isComplete) => {
  if (isComplete) return evidenceList;
  const visibleCount = Math.max(0, activePhaseIndex);
  return evidenceList.slice(0, visibleCount);
};

const getPhaseIndex = (elapsedMs) => (
  Math.min(ANALYSIS_PHASES.length - 1, Math.floor(elapsedMs / PHASE_MS))
);

export const useMatchAnalysisSession = () => {
  const [status, setStatus] = useState('idle');
  const [analyzedMatch, setAnalyzedMatch] = useState(null);
  const [activePhaseId, setActivePhaseId] = useState(ANALYSIS_PHASES[0].id);
  const [progress, setProgress] = useState(0);
  const [evidenceList, setEvidenceList] = useState([]);
  const [selectedEvidenceId, setSelectedEvidenceId] = useState(null);

  const fullEvidenceList = useMemo(
    () => (analyzedMatch ? buildEvidenceList(analyzedMatch) : []),
    [analyzedMatch]
  );

  useEffect(() => {
    if (status !== 'analyzing') return undefined;

    const startedAt = Date.now();
    const totalMs = ANALYSIS_PHASES.length * PHASE_MS;

    const interval = window.setInterval(() => {
      const elapsed = Date.now() - startedAt;
      const nextProgress = Math.min(100, Math.round((elapsed / totalMs) * 100));
      const nextPhaseIndex = getPhaseIndex(elapsed);
      const isComplete = nextProgress >= 100;
      const nextEvidence = getVisibleEvidence(fullEvidenceList, nextPhaseIndex, isComplete);

      setProgress(nextProgress);
      setActivePhaseId(ANALYSIS_PHASES[nextPhaseIndex].id);
      setEvidenceList(nextEvidence);

      if (nextEvidence[0]) {
        setSelectedEvidenceId((current) => current || nextEvidence[0].id);
      }

      if (isComplete) {
        setStatus('complete');
        setEvidenceList(fullEvidenceList);
        setSelectedEvidenceId((current) => current || fullEvidenceList[0]?.id || null);
        window.clearInterval(interval);
      }
    }, 120);

    return () => window.clearInterval(interval);
  }, [fullEvidenceList, status]);

  const selectedEvidence = useMemo(
    () => evidenceList.find((item) => item.id === selectedEvidenceId) || evidenceList[0] || null,
    [evidenceList, selectedEvidenceId]
  );

  const startAnalysis = useCallback((match) => {
    if (!match) return;
    setAnalyzedMatch(match);
    setStatus('analyzing');
    setActivePhaseId(ANALYSIS_PHASES[0].id);
    setProgress(2);
    setEvidenceList([]);
    setSelectedEvidenceId(null);
  }, []);

  const selectEvidence = useCallback((evidenceId) => {
    setSelectedEvidenceId(evidenceId);
  }, []);

  const resetAnalysis = useCallback(() => {
    setStatus('idle');
    setProgress(0);
    setEvidenceList([]);
    setSelectedEvidenceId(null);
  }, []);

  return {
    phases: ANALYSIS_PHASES,
    status,
    analyzedMatch,
    activePhaseId,
    progress,
    evidenceList,
    selectedEvidence,
    startAnalysis,
    selectEvidence,
    resetAnalysis
  };
};

export default useMatchAnalysisSession;
