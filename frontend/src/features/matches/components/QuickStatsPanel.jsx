// frontend/src/features/matches/components/QuickStatsPanel.jsx
// Sidebar panel with global stats and mini trend chart
import React, { useMemo } from 'react';
import { TrendingUp, TrendingDown, Target, Crosshair, Flame, Percent } from 'lucide-react';

const CHART_WIDTH = 240;
const CHART_HEIGHT = 60;
const CHART_PADDING = 5;

const buildTrendPoints = (data) => {
  if (!data.length) return [];
  const values = data.map(({ kd }) => kd);
  const minimum = Math.min(...values);
  const maximum = Math.max(...values);
  const range = maximum - minimum || 1;
  const width = CHART_WIDTH - CHART_PADDING * 2;
  const height = CHART_HEIGHT - CHART_PADDING * 2;

  return data.map(({ kd }, index) => ({
    kd,
    x: CHART_PADDING + (data.length === 1 ? width / 2 : index / (data.length - 1) * width),
    y: CHART_PADDING + (maximum - kd) / range * height,
  }));
};

const TrendChart = ({ data }) => {
  const points = useMemo(() => buildTrendPoints(data), [data]);
  const polyline = points.map(({ x, y }) => `${x},${y}`).join(' ');

  return (
    <svg
      viewBox={`0 0 ${CHART_WIDTH} ${CHART_HEIGHT}`}
      preserveAspectRatio="none"
      width="100%"
      height={CHART_HEIGHT}
      role="img"
      aria-label="Tendencia de K/D"
    >
      <polyline
        points={polyline}
        fill="none"
        stroke="var(--color-secondary-500)"
        strokeWidth="2"
        vectorEffect="non-scaling-stroke"
      />
      {points.map(({ kd, x, y }, index) => (
        <circle key={`${x}-${index}`} cx={x} cy={y} r="3" fill="var(--color-secondary-500)">
          <title>K/D: {kd.toFixed(2)}</title>
        </circle>
      ))}
    </svg>
  );
};

const QuickStatsPanel = ({ games, getPlayerStats }) => {
  // Calculate aggregate stats
  const stats = useMemo(() => {
    if (!games || games.length === 0) {
      return {
        totalGames: 0,
        wins: 0,
        winRate: 0,
        avgKD: 0,
        avgADR: 0,
        avgHS: 0,
        trendData: [],
        kdTrend: 'neutral',
        recentStreak: 0,
        streakType: null
      };
    }

    let totalKD = 0;
    let totalADR = 0;
    let totalHS = 0;
    let validStats = 0;
    const wins = games.filter(g => g.result === 'victory').length;

    // Build trend data from last 10 games (reversed for chart)
    const recentGames = games.slice(0, 10).reverse();
    const trendData = recentGames.map((game, idx) => {
      const ps = getPlayerStats(game);
      return {
        index: idx + 1,
        kd: ps?.kd_ratio || 0,
        result: game.result === 'victory' ? 1 : 0
      };
    });

    // Calculate averages
    games.forEach(game => {
      const ps = getPlayerStats(game);
      if (ps) {
        totalKD += ps.kd_ratio || 0;
        totalADR += ps.adr || 0;
        totalHS += ps.hs_percentage || 0;
        validStats++;
      }
    });

    const avgKD = validStats > 0 ? totalKD / validStats : 0;
    const avgADR = validStats > 0 ? totalADR / validStats : 0;
    const avgHS = validStats > 0 ? totalHS / validStats : 0;

    // Calculate K/D trend (comparing first half vs second half of recent games)
    let kdTrend = 'neutral';
    if (trendData.length >= 4) {
      const mid = Math.floor(trendData.length / 2);
      const firstHalf = trendData.slice(0, mid).reduce((sum, d) => sum + d.kd, 0) / mid;
      const secondHalf = trendData.slice(mid).reduce((sum, d) => sum + d.kd, 0) / (trendData.length - mid);
      if (secondHalf > firstHalf * 1.1) kdTrend = 'up';
      else if (secondHalf < firstHalf * 0.9) kdTrend = 'down';
    }

    // Calculate current streak
    let streak = 0;
    let streakType = null;
    for (let i = 0; i < games.length; i++) {
      const currentResult = games[i].result === 'victory';
      if (i === 0) {
        streakType = currentResult ? 'win' : 'loss';
        streak = 1;
      } else if ((currentResult && streakType === 'win') || (!currentResult && streakType === 'loss')) {
        streak++;
      } else {
        break;
      }
    }

    return {
      totalGames: games.length,
      wins,
      winRate: games.length > 0 ? (wins / games.length) * 100 : 0,
      avgKD,
      avgADR,
      avgHS,
      trendData,
      kdTrend,
      recentStreak: streak,
      streakType
    };
  }, [games, getPlayerStats]);

  const getTrendIcon = () => {
    if (stats.kdTrend === 'up') return <TrendingUp size={14} className="trend-up" />;
    if (stats.kdTrend === 'down') return <TrendingDown size={14} className="trend-down" />;
    return null;
  };

  return (
    <div className="quick-stats-panel">
      <div className="panel-header">
        <h3>Resumen</h3>
        <span className="games-count">{stats.totalGames} partidas</span>
      </div>

      <div className="stat-highlight">
        <div className="highlight-icon">
          <Target size={20} />
        </div>
        <div className="highlight-content">
          <span className="highlight-value">{stats.winRate.toFixed(0)}%</span>
          <span className="highlight-label">% de victorias</span>
        </div>
        {stats.recentStreak >= 3 && (
          <div className={`streak-badge ${stats.streakType}`}>
            <Flame size={12} />
            {stats.recentStreak} {stats.streakType === 'win' ? 'V' : 'D'}
          </div>
        )}
      </div>

      {/* Stats Grid */}
      <div className="stats-grid">
        <div className="stat-box">
          <div className="stat-header">
            <Crosshair size={14} />
            <span>K/D</span>
            {getTrendIcon()}
          </div>
          <span className="stat-value">{stats.avgKD.toFixed(2)}</span>
        </div>

        <div className="stat-box">
          <div className="stat-header">
            <Flame size={14} />
            <span>ADR</span>
          </div>
          <span className="stat-value">{Math.round(stats.avgADR)}</span>
        </div>

        <div className="stat-box">
          <div className="stat-header">
            <Percent size={14} />
            <span>HS%</span>
          </div>
          <span className="stat-value">{Math.round(stats.avgHS)}%</span>
        </div>
      </div>

      {/* Trend Chart */}
      {stats.trendData.length >= 3 && (
        <div className="trend-chart-container">
          <span className="chart-label">Tendencia K/D (últimas {stats.trendData.length} partidas)</span>
          <div className="trend-chart">
            <TrendChart data={stats.trendData} />
          </div>
        </div>
      )}

      {/* Win/Loss visualization */}
      <div className="recent-results">
        <span className="results-label">Últimas partidas</span>
        <div className="results-dots">
          {games.slice(0, 10).map((game, idx) => (
            <div 
              key={game.match_id || `${game.map_name}-${idx}`}
              className={`result-dot ${game.result === 'victory' ? 'win' : 'loss'}`}
              title={`${game.map_name?.replace('de_', '')} - ${game.result === 'victory' ? 'Victoria' : 'Derrota'}`}
            />
          ))}
        </div>
      </div>
    </div>
  );
};

export default QuickStatsPanel;
