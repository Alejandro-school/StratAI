import React from 'react';
import { CalendarClock, ChevronRight, Gift, ShieldCheck, Trophy } from 'lucide-react';

const OperationHeader = ({ countdown, onOpenRecap, season, user }) => (
  <header className="op-header">
    <div className="op-header__identity">
      <span className="op-kicker"><ShieldCheck size={14} aria-hidden="true" /> Operación semanal</span>
      <div className="op-header__title-row">
        <h1>Tu briefing de mejora</h1>
        <span className="op-season-code">{season.label} · {season.league}</span>
      </div>
      <p>Corrige el fallo que más rondas te cuesta y convierte cada partida en avance visible.</p>
    </div>

    <div className="op-header__status" aria-label="Estado de la operación semanal">
      <div className="op-status-cell">
        <Trophy size={15} aria-hidden="true" />
        <span>Posición</span>
        <strong>{user.rank ? `#${user.rank}` : 'Provisional'}</strong>
      </div>
      <div className="op-status-cell">
        <CalendarClock size={15} aria-hidden="true" />
        <span>Reinicio UTC</span>
        <strong>{countdown}</strong>
      </div>
      <div className="op-reward-vault">
        <div className="op-reward-vault__label"><Gift size={14} aria-hidden="true" /> Bóveda Pro</div>
        <div className="op-reward-vault__rewards">
          {season.rewards.map((reward) => <span key={reward.rank}>{reward.rank}º <b>{reward.label}</b></span>)}
        </div>
      </div>
    </div>

    <button type="button" className="op-recap-button" onClick={onOpenRecap}>
      Resumen anterior <ChevronRight size={15} aria-hidden="true" />
    </button>
  </header>
);

export default OperationHeader;
