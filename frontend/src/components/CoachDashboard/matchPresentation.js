export const getMatchId = (match) => match?.match_id || match?.matchID || match?.id;

export const getMatchMap = (match) => match?.map_name || match?.map || 'de_dust2';

export const getMapImage = (mapName) => `/images/maps/${mapName || 'de_dust2'}.png`;

export const formatMatchDate = (dateValue) => {
  if (!dateValue) return 'Sin fecha';

  const parsedDate = new Date(dateValue);
  if (Number.isNaN(parsedDate.getTime())) return 'Sin fecha';

  return parsedDate.toLocaleDateString('es-ES', {
    day: '2-digit',
    month: 'short',
    year: 'numeric'
  });
};

export const getMatchScore = (match) => `${match?.team_score ?? '?'}:${match?.opponent_score ?? '?'}`;

export const isMatchWin = (match) => match?.result === 'W' || match?.result === 'victory';

export const getMatchSummary = (match) => {
  if (!match) return null;

  const rawMapName = getMatchMap(match);
  return {
    id: getMatchId(match),
    rawMapName,
    mapName: rawMapName.replace('de_', ''),
    mapImage: getMapImage(rawMapName),
    score: getMatchScore(match),
    teamScore: match?.team_score ?? '?',
    opponentScore: match?.opponent_score ?? '?',
    isWin: isMatchWin(match),
    dateLabel: formatMatchDate(match.match_date),
    totalRounds: Number(match.total_rounds) > 0 ? match.total_rounds : '—',
    teamLabel: match.user_team || '—'
  };
};

export const getMatchSearchText = (match) => [
  getMatchMap(match),
  getMatchScore(match),
  formatMatchDate(match.match_date),
  getMatchId(match)
].join(' ').toLowerCase();
