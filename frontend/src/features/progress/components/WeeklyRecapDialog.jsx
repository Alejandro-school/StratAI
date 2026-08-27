import React from 'react';
import { Award, ChevronRight, ShieldCheck, Sparkles, Trophy } from 'lucide-react';
import ProgressDialog from './ProgressDialog';

const WeeklyRecapDialog = ({ onClose, recap }) => (
  <ProgressDialog ariaLabel="Resumen de la semana anterior" eyebrow={recap.season} title={recap.headline} onClose={onClose}>
    <div className="op-recap-hero">
      <div className="op-recap-rank"><Trophy size={22} aria-hidden="true" /><span>Puesto final</span><strong>#{recap.finalRank}</strong></div>
      <div className="op-recap-score"><Sparkles size={18} aria-hidden="true" /><span>Puntuación</span><strong>{recap.points}</strong></div>
      <div className="op-recap-score"><ShieldCheck size={18} aria-hidden="true" /><span>Misiones</span><strong>{recap.missionsCompleted}/3</strong></div>
    </div>
    <div className="op-recap-improvement"><span>La mejora que permanece</span><p>{recap.improvement}</p></div>
    <div className="op-recap-reward"><Award size={24} aria-hidden="true" /><div><span>Logro desbloqueado</span><strong>{recap.reward}</strong></div></div>
    <button type="button" className="op-dialog-action" onClick={onClose}>Volver a la operación actual <ChevronRight size={15} aria-hidden="true" /></button>
  </ProgressDialog>
);

export default WeeklyRecapDialog;
