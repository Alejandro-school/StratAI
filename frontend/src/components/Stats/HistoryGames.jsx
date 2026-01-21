// frontend/src/components/Stats/HistoryGames.jsx
// Redesigned Match History with Timeline View and Quick Stats Panel
import React, { useState, useCallback, useMemo, useEffect } from 'react';
import axios from 'axios';
import { useNavigate } from 'react-router-dom';
import NavigationFrame from '../Layout/NavigationFrame';
import { useUser } from '../../context/UserContext';

// New components
import QuickStatsPanel from './QuickStatsPanel';
import MatchFilters from './MatchFilters';
import MatchTimelineView from './MatchTimelineView';
import MatchTableView from './MatchTableView';

// Styles
import '../../styles/Stats/matchHistory.css';

const API_URL = process.env.REACT_APP_API_URL || (window.location.port === '3000' ? 'http://localhost:8000' : '');

const HistoryGames = () => {
  const [games, setGames] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [currentView, setCurrentView] = useState('cards'); // 'cards' or 'table'
  const [filters, setFilters] = useState({
    map: null,
    dateRange: null,
    result: null,
    search: ''
  });
  
  const { user } = useUser();
  const navigate = useNavigate();

  // Fetch ALL matches from the dedicated endpoint
  useEffect(() => {
    const fetchAllMatches = async () => {
      if (!user?.steam_id) {
        setLoading(false);
        return;
      }

      try {
        setLoading(true);
        console.log('📡 Cargando TODAS las partidas...');
        
        const response = await axios.get(`${API_URL}/steam/get-processed-demos`, {
          withCredentials: true
        });

        const matches = response.data.matches || [];
        console.log(`✅ Encontradas ${matches.length} partidas`);
        setGames(matches);
        setError(null);
      } catch (err) {
        console.error('Error al obtener partidas:', err);
        if (err.response?.status === 401) {
          setError('Sesión expirada. Por favor, inicia sesión de nuevo.');
        } else {
          setGames([]);
        }
      } finally {
        setLoading(false);
      }
    };

    fetchAllMatches();
  }, [user?.steam_id]);

  // Apply filters
  const filteredGames = useMemo(() => {
    let result = [...games];

    if (filters.map) {
      result = result.filter(g => g.map_name === filters.map);
    }
    if (filters.result) {
      result = result.filter(g => g.result === filters.result);
    }
    if (filters.search) {
      const searchLower = filters.search.toLowerCase();
      result = result.filter(g => 
        g.map_name?.toLowerCase().includes(searchLower)
      );
    }
    if (filters.dateRange) {
      const now = new Date();
      const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
      
      let filterDate;
      switch (filters.dateRange) {
        case 'Hoy':
          filterDate = today;
          break;
        case 'Esta semana':
          filterDate = new Date(today);
          filterDate.setDate(filterDate.getDate() - 7);
          break;
        case 'Este mes':
          filterDate = new Date(today);
          filterDate.setMonth(filterDate.getMonth() - 1);
          break;
        case 'Últimos 3 meses':
          filterDate = new Date(today);
          filterDate.setMonth(filterDate.getMonth() - 3);
          break;
        default:
          filterDate = null;
      }
      
      if (filterDate) {
        result = result.filter(g => new Date(g.match_date) >= filterDate);
      }
    }

    return result;
  }, [games, filters]);

  // Get player stats from a game
  const getPlayerStats = useCallback((game) => {
    return game.players?.[0] || {};
  }, []);

  // Apply filters callback
  const handleFilterChange = useCallback((newFilters) => {
    setFilters(newFilters);
  }, []);

  // View match details
  const handleViewDetails = useCallback((matchID) => {
    navigate(`/match/${user?.steam_id}/${matchID}`);
  }, [navigate, user?.steam_id]);

  // Get available maps from games
  const availableMaps = useMemo(() => 
    [...new Set(games.map(g => g.map_name).filter(Boolean))],
    [games]
  );

  // Loading skeleton
  if (loading) {
    return (
      <NavigationFrame>
        <div className="match-history-container">
          <div className="match-history-content">
            <div className="loading-skeleton">
              <div className="skeleton-sidebar" />
              <div className="skeleton-content">
                <div className="skeleton-header" />
                <div className="skeleton-filters" />
                <div className="skeleton-grid">
                  <div className="skeleton-card" />
                  <div className="skeleton-card" />
                  <div className="skeleton-card" />
                </div>
              </div>
            </div>
          </div>
        </div>
      </NavigationFrame>
    );
  }

  // Error state
  if (error) {
    return (
      <NavigationFrame>
        <div className="match-history-container">
          <div className="match-history-content">
            <div className="matches-main-content">
              <div className="timeline-empty">
                <h3>Error</h3>
                <p>{error}</p>
              </div>
            </div>
          </div>
        </div>
      </NavigationFrame>
    );
  }

  return (
    <NavigationFrame>
      <div className="match-history-container">
        <div className="match-history-content">
          {/* Quick Stats Sidebar */}
          <QuickStatsPanel 
            games={filteredGames}
            getPlayerStats={getPlayerStats}
          />

          {/* Main Content */}
          <div className="matches-main-content">
            {/* Header */}
            <div className="matches-header">
              <h1>Historial de Partidas</h1>
              <p className="subtitle">
                {games.length} partidas analizadas
              </p>
            </div>

            {/* Filters */}
            <MatchFilters
              onFilterChange={handleFilterChange}
              onViewChange={setCurrentView}
              currentView={currentView}
              availableMaps={availableMaps}
              filters={filters}
              setFilters={setFilters}
            />

            {/* Content based on view */}
            {currentView === 'cards' ? (
              <MatchTimelineView
                games={filteredGames}
                getPlayerStats={getPlayerStats}
                onViewDetails={handleViewDetails}
              />
            ) : (
              <MatchTableView
                games={filteredGames}
                getPlayerStats={getPlayerStats}
                onViewDetails={handleViewDetails}
              />
            )}
          </div>
        </div>
      </div>
    </NavigationFrame>
  );
};

export default HistoryGames;
