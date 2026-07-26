// frontend/src/pages/HistoryGames.jsx
// Redesigned Match History with Timeline View and Quick Stats Panel
import React, { useState, useCallback, useMemo, useEffect, useRef } from 'react';
import axios from 'axios';
import { useNavigate } from 'react-router-dom';
import NavigationFrame from '../components/Layout/NavigationFrame';
import { useAuth } from '../auth/useAuth';
import { API_URL } from '../utils/api';
import useMatchProgress from '../hooks/useMatchProgress';

// New components
import QuickStatsPanel from '../components/Stats/QuickStatsPanel';
import MatchFilters from '../components/Stats/MatchFilters';
import MatchTimelineView from '../components/Stats/MatchTimelineView';
import MatchTableView from '../components/Stats/MatchTableView';

// Styles
import '../styles/Stats/matchHistory.css';

const PIPELINE_ERROR_MESSAGES = {
  demo_url_unavailable: 'Steam ya no ofrece esta demo. Las partidas nuevas se procesarán automáticamente.',
  cdn_url_expired: 'El enlace de descarga ha caducado; se solicitará uno nuevo automáticamente.',
  steam_credentials_invalid: 'Steam ha rechazado los códigos del historial. Vuelve a configurarlos.',
};

function getPipelineMessage(event) {
  if (!event) return '';
  if (event.stage === 'discovery') return 'Buscando partidas nuevas...';
  if (event.stage === 'queued') return 'Partida en cola...';
  if (event.stage === 'resolving') return 'Preparando la descarga...';
  if (event.stage === 'downloading') return 'Descargando demo...';
  if (event.stage === 'analyzing') return 'Analizando demo...';
  if (event.stage === 'retry_wait') return 'Steam no ha respondido; reintentando automáticamente...';
  if (event.stage === 'failed') {
    return PIPELINE_ERROR_MESSAGES[event.error_code]
      || `No se pudo procesar la partida (${event.error_code || 'error desconocido'}).`;
  }
  return '';
}

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
  
  const { user } = useAuth();
  const navigate = useNavigate();
  const hasFetchedRef = useRef(false);
  const [refreshCooldown, setRefreshCooldown] = useState(0);
  const { isProcessing, latestEvent, completedCount, resetCompleted } = useMatchProgress(user?.steam_id);

  // Auto-fetch new matches on mount (with 5-min staleness check)
  useEffect(() => {
    if (!user?.steam_id || hasFetchedRef.current) return;
    hasFetchedRef.current = true;

    const lastFetch = localStorage.getItem(`lastMatchFetch:${user.steam_id}`);
    const staleMs = 5 * 60 * 1000;
    if (lastFetch && Date.now() - parseInt(lastFetch, 10) < staleMs) return;

    localStorage.setItem(`lastMatchFetch:${user.steam_id}`, Date.now().toString());
    fetch(`${API_URL}/steam/discovery`, {
      method: 'POST',
      credentials: 'include',
    }).catch(() => {});
  }, [user?.steam_id]);

  // Refresh match list when SSE reports a completed match
  useEffect(() => {
    if (completedCount === 0) return;
    const timer = setTimeout(async () => {
      try {
        const response = await axios.get(`${API_URL}/steam/get-processed-demos`, {
          withCredentials: true,
        });
        setGames(response.data.matches || []);
      } catch {}
      resetCompleted();
    }, 1500); // small delay to let Redis propagate
    return () => clearTimeout(timer);
  }, [completedCount, resetCompleted]);

  // Manual refresh handler
  const handleRefreshMatches = useCallback(async () => {
    if (refreshCooldown > 0) return;
    try {
      const res = await fetch(`${API_URL}/steam/discovery`, {
        method: 'POST',
        credentials: 'include',
      });
      if (!res.ok) throw new Error(`Discovery failed: ${res.status}`);
      setRefreshCooldown(60);
      localStorage.setItem(`lastMatchFetch:${user?.steam_id}`, Date.now().toString());
    } catch {
      setRefreshCooldown(10);
    }
  }, [refreshCooldown, user?.steam_id]);

  // Cooldown timer
  useEffect(() => {
    if (refreshCooldown <= 0) return;
    const timer = setInterval(() => {
      setRefreshCooldown(c => (c <= 1 ? 0 : c - 1));
    }, 1000);
    return () => clearInterval(timer);
  }, [refreshCooldown]);

  // Fetch ALL matches from the dedicated endpoint
  useEffect(() => {
    const fetchAllMatches = async () => {
      if (!user?.steam_id) {
        setLoading(false);
        return;
      }

      try {
        setLoading(true);
        
        const response = await axios.get(`${API_URL}/steam/get-processed-demos`, {
          withCredentials: true
        });

        const matches = response.data.matches || [];
        setGames(matches);
        setError(null);
      } catch (err) {
        console.error('Error al obtener partidas:', err);
        if (err.response?.status === 401) {
          setError('SesiÃ³n expirada. Por favor, inicia sesiÃ³n de nuevo.');
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
        case 'Ãšltimos 3 meses':
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
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <div>
                  <h1>Historial de Partidas</h1>
                  <p className="subtitle">
                    {games.length} partidas analizadas
                  </p>
                </div>
                <button
                  onClick={handleRefreshMatches}
                  disabled={refreshCooldown > 0}
                  className="refresh-matches-btn"
                  title={refreshCooldown > 0 ? `Disponible en ${refreshCooldown}s` : 'Buscar nuevas partidas'}
                >
                  {isProcessing ? '⏳ Procesando...' : refreshCooldown > 0 ? `${refreshCooldown}s` : '🔄 Buscar partidas'}
                </button>
              </div>
              {latestEvent && (isProcessing || latestEvent.stage === 'failed') && (
                <div className="match-progress-bar">
                  <span className="progress-stage">
                    {getPipelineMessage(latestEvent)}
                  </span>
                </div>
              )}
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

