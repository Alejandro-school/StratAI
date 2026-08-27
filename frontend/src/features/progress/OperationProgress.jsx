import React from 'react';
import { RefreshCcw } from 'lucide-react';
import LeagueDialog from './components/LeagueDialog';
import LeaguePanel from './components/LeaguePanel';
import MatchPulse from './components/MatchPulse';
import MissionEvidenceDialog from './components/MissionEvidenceDialog';
import OperationHeader from './components/OperationHeader';
import PrimaryMission from './components/PrimaryMission';
import ProgressDialog from './components/ProgressDialog';
import SupportMissions from './components/SupportMissions';
import WeeklyRecapDialog from './components/WeeklyRecapDialog';
import useProgressPrototype from './useProgressPrototype';
import './styles/operationShell.css';
import './styles/missionBrief.css';
import './styles/league.css';
import './styles/dialogs.css';
import './styles/responsive.css';

const OperationProgress = ({ userName }) => {
  const progress = useProgressPrototype(userName);
  const primaryMission = progress.data.missions.find((mission) => mission.role === 'primary');
  const supportMissions = progress.data.missions.filter((mission) => mission.role === 'support');

  return (
    <div className="op-page">
      <div className="op-page__atmosphere" aria-hidden="true" />
      <OperationHeader
        countdown={progress.countdown}
        onOpenRecap={progress.openRecap}
        season={progress.data.season}
        user={progress.data.user}
      />

      <div className="op-command-grid">
        <div className="op-command-grid__missions">
          <PrimaryMission mission={primaryMission} onOpenEvidence={progress.openMission} />
          <SupportMissions
            hasReroll={progress.data.user.hasReroll}
            missions={supportMissions}
            onOpenEvidence={progress.openMission}
            onRequestReroll={progress.requestReroll}
          />
        </div>
        <LeaguePanel
          data={progress.data}
          nextRankGap={progress.nextRankGap}
          onOpenLeague={progress.openLeague}
          onOptIn={progress.toggleOptIn}
          podiumGap={progress.podiumGap}
        />
      </div>

      <MatchPulse
        matches={progress.data.recentMatches}
        scoringMatches={progress.data.user.scoringMatches}
        maxMatches={progress.data.season.maxScoringMatches}
      />

      <div className="op-announcement" role="status" aria-live="polite">{progress.announcement}</div>

      {progress.selectedMission ? <MissionEvidenceDialog mission={progress.selectedMission} onClose={progress.closeMission} /> : null}
      {progress.isLeagueOpen ? (
        <LeagueDialog
          data={progress.data}
          onClose={progress.closeLeague}
          onToggleAnonymous={progress.toggleAnonymous}
          onToggleOptIn={progress.toggleOptIn}
        />
      ) : null}
      {progress.isRecapOpen ? <WeeklyRecapDialog recap={progress.data.recap} onClose={progress.closeRecap} /> : null}
      {progress.pendingReroll ? (
        <ProgressDialog ariaLabel="Confirmar cambio de misión" eyebrow="1 cambio semanal" title="¿Sustituir esta misión?" onClose={progress.cancelReroll}>
          <div className="op-reroll-confirm">
            <RefreshCcw size={24} aria-hidden="true" />
            <p>Cambiaremos <b>{progress.pendingReroll.title}</b> por el siguiente fallo verificable. Mantendrá {progress.pendingReroll.maxPoints} puntos máximos.</p>
          </div>
          <div className="op-dialog-actions">
            <button type="button" className="op-secondary-action" onClick={progress.cancelReroll}>Conservar misión</button>
            <button type="button" className="op-dialog-action" onClick={progress.confirmReroll}>Usar mi cambio</button>
          </div>
        </ProgressDialog>
      ) : null}
    </div>
  );
};

export default OperationProgress;
