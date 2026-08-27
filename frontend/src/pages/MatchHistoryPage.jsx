// Redesigned Match History with Timeline View and Quick Stats Panel
import React, { Suspense, lazy, useState, useCallback, useMemo, useEffect, useRef } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import NavigationFrame from '../components/Layout/NavigationFrame';
import { useAuth } from '../auth/useAuth';
import useMatchProgress from '../hooks/useMatchProgress';
import matchesApi from '../features/matches/api/matchesApi';
import { matchKeys, matchesQueryOptions } from '../features/matches/queries/matchQueries';

import MatchFilters from '../features/matches/components/MatchFilters';

// Styles
import '../styles/Stats/matchHistory.css';

const QuickStatsPanel = lazy(() => import('../features/matches/components/QuickStatsPanel'));
const MatchTimelineView = lazy(() => import('../features/matches/components/MatchTimelineView'));
const MatchTableView = lazy(() => import('../features/matches/components/MatchTableView'));
const EMPTY_MATCHES = Object.freeze([]);

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

const MatchHistoryPage = () => {
  const [currentView, setCurrentView] = useState('cards'); // 'cards' or 'table'
  const [filters, setFilters] = useState({
    map: null,
    dateRange: null,
    result: null,
    search: ''
  });
  
  const { user } = useAuth();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const hasFetchedRef = useRef(false);
  const [refreshCooldown, setRefreshCooldown] = useState(0);
  const { isProcessing, latestEvent, completedCount, resetCompleted } = useMatchProgress(user?.steam_id);
  const matchesQuery = useQuery(matchesQueryOptions(user?.steam_id));
  const games = matchesQuery.data || EMPTY_MATCHES;
  const loading = Boolean(user?.steam_id) && matchesQuery.isPending;
  const error = matchesQuery.error?.status === 401
    ? 'Sesión expirada. Por favor, inicia sesión de nuevo.'
    : null;

  // Auto-fetch new matches on mount (with 5-min staleness check)
  useEffect(() => {
    if (!user?.steam_id || hasFetchedRef.current) return;
    hasFetchedRef.current = true;

    const lastFetch = localStorage.getItem(`lastMatchFetch:${user.steam_id}`);
    const staleMs = 5 * 60 * 1000;
    if (lastFetch && Date.now() - parseInt(lastFetch, 10) < staleMs) return;

    localStorage.setItem(`lastMatchFetch:${user.steam_id}`, Date.now().toString());
    matchesApi.discover().catch(() => {});
  }, [user?.steam_id]);

  // Refresh match list when SSE reports a completed match
  useEffect(() => {
    if (completedCount === 0) return;
    const timer = setTimeout(() => {
      queryClient.invalidateQueries({ queryKey: matchKeys.list(user?.steam_id) })
        .finally(resetCompleted);
    }, 1500); // small delay to let Redis propagate
    return () => clearTimeout(timer);
  }, [completedCount, queryClient, resetCompleted, user?.steam_id]);

  // Manual refresh handler
  const handleRefreshMatches = useCallback(async () => {
    if (refreshCooldown > 0) return;
    try {
      await matchesApi.discover();
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
          <Suspense fallback={<div className="skeleton-sidebar" aria-hidden="true" />}>
            <QuickStatsPanel
              games={filteredGames}
              getPlayerStats={getPlayerStats}
            />
          </Suspense>

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
            <Suspense fallback={<div className="skeleton-grid" aria-hidden="true" />}>
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
            </Suspense>
          </div>
        </div>
      </div>
    </NavigationFrame>
  );
};

export default MatchHistoryPage;

