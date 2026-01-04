// frontend/src/components/Stats/MatchTimelineView.jsx
// Timeline view with cards - simplified without date grouping when dates are missing
import React, { useMemo } from 'react';
import MatchCard from './MatchCard';
import { Calendar, Gamepad2 } from 'lucide-react';

const MatchTimelineView = ({ games, getPlayerStats, onViewDetails }) => {
  // Check if we have valid dates
  const hasValidDates = useMemo(() => {
    return games.some(game => {
      const date = game.match_date ? new Date(game.match_date) : null;
      return date && !isNaN(date.getTime());
    });
  }, [games]);

  // Group games by date only if we have valid dates
  const groupedGames = useMemo(() => {
    if (!hasValidDates) {
      // No valid dates - return all games in one group
      return [{
        label: `${games.length} partidas analizadas`,
        date: new Date(),
        games: games,
        isAllMatches: true
      }];
    }

    const groups = {};
    
    games.forEach(game => {
      const dateStr = game.match_date;
      const date = dateStr ? new Date(dateStr) : null;
      const isValidDate = date && !isNaN(date.getTime());
      
      const today = new Date();
      const yesterday = new Date(today);
      yesterday.setDate(yesterday.getDate() - 1);
      
      let dateKey;
      let dateLabel;
      
      if (!isValidDate) {
        dateKey = 'unknown';
        dateLabel = 'Otras partidas';
      } else if (date.toDateString() === today.toDateString()) {
        dateKey = 'today';
        dateLabel = 'Hoy';
      } else if (date.toDateString() === yesterday.toDateString()) {
        dateKey = 'yesterday';
        dateLabel = 'Ayer';
      } else {
        dateKey = date.toDateString();
        dateLabel = date.toLocaleDateString('es-ES', { 
          weekday: 'long', 
          day: 'numeric', 
          month: 'long' 
        });
        dateLabel = dateLabel.charAt(0).toUpperCase() + dateLabel.slice(1);
      }
      
      if (!groups[dateKey]) {
        groups[dateKey] = {
          label: dateLabel,
          date: isValidDate ? date : new Date(0),
          games: []
        };
      }
      groups[dateKey].games.push(game);
    });
    
    return Object.values(groups).sort((a, b) => b.date - a.date);
  }, [games, hasValidDates]);

  if (games.length === 0) {
    return (
      <div className="timeline-empty">
        <Calendar size={48} className="empty-icon" />
        <h3>No hay partidas</h3>
        <p>Las partidas aparecerán aquí una vez procesadas</p>
      </div>
    );
  }

  return (
    <div className="match-timeline">
      {groupedGames.map((group, groupIdx) => (
        <div key={group.label} className="timeline-group">
          {/* Date/count separator */}
          <div className="timeline-date-separator">
            <div className="date-line" />
            <div className="date-badge">
              {group.isAllMatches ? (
                <Gamepad2 size={14} />
              ) : (
                <Calendar size={14} />
              )}
              <span>{group.label}</span>
              {!group.isAllMatches && (
                <span className="games-count">{group.games.length} partidas</span>
              )}
            </div>
            <div className="date-line" />
          </div>
          
          {/* Cards grid */}
          <div className="timeline-cards-grid">
            {group.games.map((game, gameIdx) => (
              <MatchCard
                key={game.match_id}
                match={game}
                playerStats={getPlayerStats(game)}
                onViewDetails={onViewDetails}
                style={{
                  animationDelay: `${(groupIdx * 0.1) + (gameIdx * 0.05)}s`
                }}
              />
            ))}
          </div>
        </div>
      ))}
    </div>
  );
};

export default MatchTimelineView;
