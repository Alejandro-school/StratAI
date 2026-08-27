import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  REROLL_REPLACEMENT,
  getNextRankGap,
  getPodiumGap,
  getProgressPrototypeData,
  syncUserScore,
} from './progressPrototypeData';
import { formatResetCountdown, replaceMission } from './progressModel';

const VALID_SCENARIOS = new Set(['near-podium', 'new', 'outside', 'podium']);

const useProgressPrototype = (userName) => {
  const [searchParams, setSearchParams] = useSearchParams();
  const requestedScenario = searchParams.get('scenario') || 'near-podium';
  const scenario = import.meta.env.DEV && VALID_SCENARIOS.has(requestedScenario) ? requestedScenario : 'near-podium';
  const prototypeData = useMemo(() => getProgressPrototypeData(scenario, userName), [scenario, userName]);
  const [data, setData] = useState(prototypeData);
  const [now, setNow] = useState(() => new Date());
  const [pendingReroll, setPendingReroll] = useState(null);
  const [announcement, setAnnouncement] = useState('');

  useEffect(() => setData(prototypeData), [prototypeData]);

  useEffect(() => {
    const interval = window.setInterval(() => setNow(new Date()), 60_000);
    return () => window.clearInterval(interval);
  }, []);

  const updateQuery = useCallback((key, value) => {
    const next = new URLSearchParams(searchParams);
    if (value) next.set(key, value);
    else next.delete(key);
    setSearchParams(next, { replace: true });
  }, [searchParams, setSearchParams]);

  const confirmReroll = useCallback(() => {
    if (!pendingReroll) return;
    setData((current) => {
      const missions = replaceMission(current.missions, pendingReroll.id, REROLL_REPLACEMENT);
      const next = syncUserScore(current, missions);
      return { ...next, user: { ...next.user, hasReroll: false } };
    });
    setPendingReroll(null);
    setAnnouncement('Misión sustituida. Has utilizado tu cambio semanal.');
  }, [pendingReroll]);

  const toggleOptIn = useCallback(() => {
    setData((current) => {
      const isOptedIn = !current.user.isOptedIn;
      const currentPlayer = current.leaderboard.find((entry) => entry.isCurrentUser);
      return {
        ...current,
        user: {
          ...current.user,
          isOptedIn,
          rank: isOptedIn ? current.user.rank ?? currentPlayer?.rank ?? null : current.user.rank,
        },
      };
    });
  }, []);

  const toggleAnonymous = useCallback(() => {
    setData((current) => ({
      ...current,
      user: { ...current.user, isAnonymous: !current.user.isAnonymous },
    }));
  }, []);

  const missionId = searchParams.get('mission');
  return {
    announcement,
    countdown: formatResetCountdown(data.season.resetAt, now),
    data,
    isLeagueOpen: searchParams.get('panel') === 'league',
    isRecapOpen: searchParams.get('recap') === '1',
    nextRankGap: getNextRankGap(data),
    pendingReroll,
    podiumGap: getPodiumGap(data),
    selectedMission: data.missions.find((mission) => mission.id === missionId),
    closeLeague: () => updateQuery('panel', null),
    closeMission: () => updateQuery('mission', null),
    closeRecap: () => updateQuery('recap', null),
    confirmReroll,
    openLeague: () => updateQuery('panel', 'league'),
    openMission: (id) => updateQuery('mission', id),
    openRecap: () => updateQuery('recap', '1'),
    requestReroll: setPendingReroll,
    cancelReroll: () => setPendingReroll(null),
    toggleAnonymous,
    toggleOptIn,
  };
};

export default useProgressPrototype;
