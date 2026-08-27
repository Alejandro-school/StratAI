import React, { Suspense, lazy, useCallback, useDeferredValue, useEffect, useMemo, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import AnalysisProgressRail from '../components/CoachAnalysis/AnalysisProgressRail';
import CoachConversationPanel from '../components/CoachAnalysis/CoachConversationPanel';
import TacticalVault from '../components/CoachDashboard/TacticalVault';
import {
  getMapImage,
  getMatchId,
  getMatchMap,
  getMatchSearchText,
  isMatchWin
} from '../components/CoachDashboard/matchPresentation';
import NavigationFrame from '../components/Layout/NavigationFrame';
import { useAuth } from '../auth/useAuth';
import useCoachChat from '../hooks/useCoachChat';
import useMatchAnalysisSession from '../hooks/useMatchAnalysisSession';
import useReplaySyncStore from '../stores/useReplaySyncStore';
import { matchesQueryOptions } from '../features/matches/queries/matchQueries';
import '../styles/pages/coachDashboard.css';
import '../styles/pages/tacticalVault.css';

const AnalysisReplayStage = lazy(() => import('../components/CoachAnalysis/AnalysisReplayStage'));
const EMPTY_MATCHES = Object.freeze([]);

const matchesDateFilter = (match, dateFilter) => {
  if (dateFilter === 'all') return true;

  const matchDate = new Date(match.match_date || 0);
  if (Number.isNaN(matchDate.getTime())) return false;

  const now = new Date();
  if (dateFilter === 'year') return matchDate.getFullYear() === now.getFullYear();

  const days = Number(dateFilter);
  const earliestDate = now.getTime() - days * 24 * 60 * 60 * 1000;
  return matchDate.getTime() >= earliestDate;
};

const AnalysisMode = ({
  match,
  matchId,
  session,
  chat,
  replay,
  chatEndRef,
  onSubmitMessage,
  onPlayEvidence,
  onTogglePlan,
  savedEvidenceIds,
  onReset
}) => {
  const mapImage = getMapImage(getMatchMap(match));
  const [mobileView, setMobileView] = useState('coach');

  return (
    <div className={`analysis-room-layout mobile-view-${mobileView}`}>
      <AnalysisProgressRail
        phases={session.phases}
        activePhaseId={session.activePhaseId}
        progress={session.progress}
        status={session.status}
        evidenceCount={session.evidenceList.length}
        onReset={onReset}
      />

      <div className="analysis-mobile-tabs" role="tablist" aria-label="Vista de análisis">
        <button
          type="button"
          role="tab"
          aria-selected={mobileView === 'coach'}
          onClick={() => setMobileView('coach')}
        >
          Coach
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={mobileView === 'replay'}
          onClick={() => setMobileView('replay')}
        >
          Replay 2D
        </button>
      </div>

      <main className="analysis-room-main">
        <Suspense fallback={<div className="analysis-replay-stage" role="status">Cargando replay…</div>}>
          <AnalysisReplayStage
            match={match}
            matchId={matchId}
            mapImage={mapImage}
            selectedEvidence={session.selectedEvidence}
            status={session.status}
            isPlaying={replay.isPlaying}
            activeClip={replay.activeClip}
            onPlayEvidence={onPlayEvidence}
            onTogglePlan={onTogglePlan}
            isSavedToPlan={savedEvidenceIds.has(session.selectedEvidence?.id)}
          />
        </Suspense>
      </main>

      <CoachConversationPanel
        messages={chat.messages}
        chatInput={chat.chatInput}
        setChatInput={chat.setChatInput}
        isAiTyping={chat.isAiTyping}
        quickActions={chat.quickActions}
        chatEndRef={chatEndRef}
        isPlaying={replay.isPlaying}
        activeClip={replay.activeClip}
        evidenceList={session.evidenceList}
        selectedEvidence={session.selectedEvidence}
        analysisStatus={session.status}
        onSelectEvidence={session.selectEvidence}
        onPlayEvidence={onPlayEvidence}
        onTogglePlan={onTogglePlan}
        isSavedToPlan={savedEvidenceIds.has(session.selectedEvidence?.id)}
        onSubmitMessage={onSubmitMessage}
        onPlayInteraction={replay.playAiClip}
      />
    </div>
  );
};

const CoachDashboard = () => {
  const { user } = useAuth();
  const replay = useReplaySyncStore();
  const chatEndRef = useRef(null);
  const greetingSentRef = useRef(false);
  const {
    data: allMatches = EMPTY_MATCHES,
    error: matchesQueryError,
    isPending: areMatchesPending,
    refetch: refetchMatches,
  } = useQuery(matchesQueryOptions(user?.steam_id));
  const loadingMatches = Boolean(user?.steam_id) && areMatchesPending;
  const matchesError = matchesQueryError?.message || null;
  const [savedEvidenceIds, setSavedEvidenceIds] = useState(() => new Set());
  const [selectedMatchId, setSelectedMatchId] = useState(null);
  const [matchFilterQuery, setMatchFilterQuery] = useState('');
  const [mapFilter, setMapFilter] = useState('all');
  const [resultFilter, setResultFilter] = useState('all');
  const [dateFilter, setDateFilter] = useState('all');
  const [sortBy, setSortBy] = useState('date_desc');
  const deferredFilterQuery = useDeferredValue(matchFilterQuery);
  const analysisSession = useMatchAnalysisSession();

  const {
    messages,
    chatInput,
    setChatInput,
    isAiTyping,
    quickActions,
    submitMessage,
    sendProactiveGreeting,
    analyzeMatch
  } = useCoachChat(user);

  useEffect(() => {
    if (greetingSentRef.current || !allMatches.length) return;
    greetingSentRef.current = true;
    setSelectedMatchId(getMatchId(allMatches[0]) || null);
    sendProactiveGreeting(allMatches);
  }, [allMatches, sendProactiveGreeting]);

  const filteredMatches = useMemo(() => {
    const query = deferredFilterQuery.trim().toLowerCase();

    return allMatches.filter((match) => {
      if (query && !getMatchSearchText(match).includes(query)) return false;
      if (mapFilter !== 'all' && getMatchMap(match) !== mapFilter) return false;
      if (resultFilter === 'win' && !isMatchWin(match)) return false;
      if (resultFilter === 'loss' && isMatchWin(match)) return false;
      return matchesDateFilter(match, dateFilter);
    });
  }, [allMatches, dateFilter, deferredFilterQuery, mapFilter, resultFilter]);

  const availableMaps = useMemo(
    () => [...new Set(allMatches.map(getMatchMap))].sort((a, b) => a.localeCompare(b)),
    [allMatches]
  );

  const sortedMatches = useMemo(() => {
    const matches = [...filteredMatches];
    const getDateValue = (match) => new Date(match.match_date || 0).getTime() || 0;
    const getMapValue = (match) => getMatchMap(match).toLowerCase();

    if (sortBy === 'date_asc') return matches.sort((a, b) => getDateValue(a) - getDateValue(b));
    if (sortBy === 'map_asc') return matches.sort((a, b) => getMapValue(a).localeCompare(getMapValue(b)));
    if (sortBy === 'map_desc') return matches.sort((a, b) => getMapValue(b).localeCompare(getMapValue(a)));
    return matches.sort((a, b) => getDateValue(b) - getDateValue(a));
  }, [filteredMatches, sortBy]);

  useEffect(() => {
    if (!sortedMatches.length) {
      setSelectedMatchId(null);
      return;
    }

    const selectedStillVisible = sortedMatches.some((match) => getMatchId(match) === selectedMatchId);
    if (!selectedStillVisible) {
      setSelectedMatchId(getMatchId(sortedMatches[0]));
    }
  }, [selectedMatchId, sortedMatches]);

  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'auto' });
  }, [messages, isAiTyping, analysisSession.selectedEvidence]);

  const selectedMatch = useMemo(
    () => sortedMatches.find((match) => getMatchId(match) === selectedMatchId) || null,
    [selectedMatchId, sortedMatches]
  );

  const dashboardMode = analysisSession.status === 'idle' ? 'command' : 'analysis';

  const submitCoachMessage = (text) => {
    if (!text?.trim()) return;
    submitMessage(text);
  };

  const startSelectedMatchAnalysis = useCallback(() => {
    if (!selectedMatch) return;
    setSavedEvidenceIds(new Set());
    analysisSession.startAnalysis(selectedMatch);
    analyzeMatch(selectedMatch);
  }, [analysisSession, analyzeMatch, selectedMatch]);

  const retryMatches = useCallback(() => refetchMatches(), [refetchMatches]);

  const resetMatchFilters = useCallback(() => {
    setMatchFilterQuery('');
    setMapFilter('all');
    setResultFilter('all');
    setDateFilter('all');
    setSortBy('date_desc');
  }, []);

  const hasActiveFilters = Boolean(
    matchFilterQuery || mapFilter !== 'all' || resultFilter !== 'all' || dateFilter !== 'all' || sortBy !== 'date_desc'
  );

  const playEvidence = (evidence) => {
    analysisSession.selectEvidence(evidence.id);
    if (evidence.interaction) {
      replay.playAiClip(evidence.interaction);
    }
  };

  const resetAnalysis = () => {
    replay.stopAiClip();
    analysisSession.resetAnalysis();
  };

  const toggleEvidenceInPlan = (evidenceId) => {
    if (!evidenceId) return;
    setSavedEvidenceIds((current) => {
      const next = new Set(current);
      if (next.has(evidenceId)) next.delete(evidenceId);
      else next.add(evidenceId);
      return next;
    });
  };

  return (
    <NavigationFrame>
      <div className={`analysis-dashboard mode-${dashboardMode}`}>
        {dashboardMode === 'command' ? (
          <TacticalVault
            matches={sortedMatches}
            allMatchesCount={allMatches.length}
            selectedMatch={selectedMatch}
            selectedMatchId={selectedMatchId}
            loading={loadingMatches}
            error={matchesError}
            query={matchFilterQuery}
            mapFilter={mapFilter}
            resultFilter={resultFilter}
            dateFilter={dateFilter}
            sortBy={sortBy}
            availableMaps={availableMaps}
            hasActiveFilters={hasActiveFilters}
            analysisStatus={analysisSession.status}
            onQueryChange={setMatchFilterQuery}
            onMapFilterChange={setMapFilter}
            onResultFilterChange={setResultFilter}
            onDateFilterChange={setDateFilter}
            onSortChange={setSortBy}
            onResetFilters={resetMatchFilters}
            onSelectMatch={setSelectedMatchId}
            onAnalyze={startSelectedMatchAnalysis}
            onRetry={retryMatches}
          />
        ) : (
          <AnalysisMode
            match={analysisSession.analyzedMatch || selectedMatch}
            matchId={getMatchId(analysisSession.analyzedMatch || selectedMatch)}
            session={analysisSession}
            chat={{
              messages,
              chatInput,
              setChatInput,
              isAiTyping,
              quickActions
            }}
            replay={replay}
            chatEndRef={chatEndRef}
            onSubmitMessage={submitCoachMessage}
            onPlayEvidence={playEvidence}
            onTogglePlan={toggleEvidenceInPlan}
            savedEvidenceIds={savedEvidenceIds}
            onReset={resetAnalysis}
          />
        )}
      </div>
    </NavigationFrame>
  );
};

export default CoachDashboard;
