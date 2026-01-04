// frontend/src/components/Stats/QuickStatsPanel.jsx
// Sidebar panel with global stats and mini trend chart
import React, { useMemo } from 'react';
import { 
  LineChart, Line, ResponsiveContainer, Tooltip
} from 'recharts';
import { TrendingUp, TrendingDown, Target, Crosshair, Flame, Percent } from 'lucide-react';

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

  // Custom tooltip for chart
  const CustomTooltip = ({ active, payload }) => {
    if (active && payload && payload.length) {
      return (
        <div className="trend-tooltip">
          <span className="tooltip-kd">K/D: {payload[0].value.toFixed(2)}</span>
        </div>
      );
    }
    return null;
  };

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

      {/* Win Rate - Highlighted */}
      <div className="stat-highlight">
        <div className="highlight-icon">
          <Target size={20} />
        </div>
        <div className="highlight-content">
          <span className="highlight-value">{stats.winRate.toFixed(0)}%</span>
          <span className="highlight-label">Win Rate</span>
        </div>
        {stats.recentStreak >= 3 && (
          <div className={`streak-badge ${stats.streakType}`}>
            <Flame size={12} />
            {stats.recentStreak} {stats.streakType === 'win' ? 'W' : 'L'}
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
            <ResponsiveContainer width="100%" height={60}>
              <LineChart data={stats.trendData}>
                <Tooltip content={<CustomTooltip />} />
                <Line 
                  type="monotone" 
                  dataKey="kd" 
                  stroke="#8b5cf6" 
                  strokeWidth={2}
                  dot={{ fill: '#8b5cf6', strokeWidth: 0, r: 3 }}
                  activeDot={{ fill: '#a78bfa', r: 5 }}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      {/* Win/Loss visualization */}
      <div className="recent-results">
        <span className="results-label">Últimas partidas</span>
        <div className="results-dots">
          {games.slice(0, 10).map((game, idx) => (
            <div 
              key={idx} 
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
