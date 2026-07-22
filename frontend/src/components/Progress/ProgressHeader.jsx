import React from 'react';
import { Calendar, RefreshCw, TrendingUp } from 'lucide-react';
import RankBadge from './RankBadge';
import { getTierForPoints } from '../../mocks/progressData';
import '../../styles/Progress/progressHeader.css';

const ProgressHeader = ({
  currentPlayer,
  monthReset,
  refreshCountdown,
  lastRefreshAt,
  isRefreshing,
  totalMissions,
  completedMissions,
}) => {
  const mins = String(Math.floor(refreshCountdown / 60)).padStart(2, '0');
  const secs = String(refreshCountdown % 60).padStart(2, '0');
  const tier = getTierForPoints(currentPlayer.points);

  return (
    <header className="ph">
      {/* Left: Rank identity */}
      <div className="ph__left">
        <RankBadge points={currentPlayer.points} rank={currentPlayer.rank} />
        <div className="ph__user">
          <span className="ph__name">{currentPlayer.name}</span>
          <div className="ph__stats">
            <span className="ph__stat" style={{ '--stat-color': tier.color }}>
              <TrendingUp size={11} />
              {currentPlayer.trend} este mes
            </span>
          </div>
        </div>
      </div>

      {/* Right: Timers compact */}
      <div className="ph__right">
        <div className="ph__pill">
          <Calendar size={12} />
          <span>{monthReset.days}d {monthReset.hours}h {monthReset.minutes}m</span>
        </div>
        <div className={`ph__pill ph__pill--sync ${isRefreshing ? 'ph__pill--pulse' : ''}`}>
          <RefreshCw size={12} />
          <span>{mins}:{secs}</span>
        </div>
      </div>
    </header>
  );
};

export default ProgressHeader;
