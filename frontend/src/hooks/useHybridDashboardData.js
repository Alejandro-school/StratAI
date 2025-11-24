// frontend/src/hooks/useHybridDashboardData.js
// Hook híbrido: Carga stats instantáneas primero, luego enriquece con análisis profundo
import { useState, useEffect, useCallback } from 'react';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8000';

export const useHybridDashboardData = (user) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [dashboardData, setDashboardData] = useState(null);
  const [downloadStatus, setDownloadStatus] = useState(null);
  const [dataSource, setDataSource] = useState(null); // 'quick' | 'deep' | 'hybrid'

  // Función para obtener stats rápidas de Steam API
  const fetchQuickStats = useCallback(async () => {
    try {
      console.log('[HybridDashboard] Fetching quick stats from Steam API...');
      
      const response = await fetch(`${API_URL}/steam/match-history`, {
        credentials: 'include',
      });

      if (!response.ok) {
        const errorText = await response.text();
        console.error('[HybridDashboard] Quick stats error:', response.status, errorText);
        throw new Error(`Steam API error: ${response.status}`);
      }

      const data = await response.json();
      console.log('[HybridDashboard] Quick stats received:', data.matches?.length, 'matches');
      
      return data.matches || [];
    } catch (err) {
      console.error('[HybridDashboard] Error fetching quick stats:', err);
      return [];
    }
  }, []);

  // Función para obtener análisis profundo
  const fetchDeepAnalysis = useCallback(async () => {
    try {
      console.log('[HybridDashboard] Fetching deep analysis...');
      
      const response = await fetch(`${API_URL}/steam/get-processed-demos`, {
        credentials: 'include',
      });

      if (!response.ok) {
        if (response.status === 404) {
          console.log('[HybridDashboard] No demos procesadas aún');
          return [];
        }
        throw new Error('Error al obtener análisis profundo');
      }

      const data = await response.json();
      console.log('[HybridDashboard] Deep analysis received:', data.demos?.length, 'demos');
      
      return data.demos || [];
    } catch (err) {
      console.error('[HybridDashboard] Error fetching deep analysis:', err);
      return [];
    }
  }, []);

  // Función para obtener estado de descargas
  const fetchDownloadStatus = useCallback(async () => {
    try {
      const response = await fetch(`${API_URL}/steam/download-status`, {
        credentials: 'include',
      });

      if (!response.ok) {
        return null;
      }

      const status = await response.json();
      console.log('[HybridDashboard] Download status:', status);
      return status;
    } catch (err) {
      console.error('[HybridDashboard] Error fetching download status:', err);
      return null;
    }
  }, []);

  // Función principal de carga
  const loadDashboardData = useCallback(async () => {
    const steamId = user?.steam_id || user?.steamid || user?.steamID;
    
    if (!steamId) {
      console.log('[HybridDashboard] No steam ID found');
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      setError(null);

      // 1. Cargar stats rápidas PRIMERO (experiencia instantánea)
      const quickMatches = await fetchQuickStats();
      
      if (quickMatches.length > 0) {
        console.log('[HybridDashboard] Mostrando stats rápidas...');
        const quickData = processQuickStats(quickMatches, steamId);
        setDashboardData(quickData);
        setDataSource('quick');
        setLoading(false); // Ya podemos mostrar algo al usuario
      }

      // 2. Verificar estado de descargas
      const status = await fetchDownloadStatus();
      setDownloadStatus(status);

      // 3. Intentar obtener análisis profundo (si existe)
      const deepDemos = await fetchDeepAnalysis();
      
      if (deepDemos.length > 0) {
        console.log('[HybridDashboard] Enriqueciendo con análisis profundo...');
        // Importar la función processDemos del hook original
        const { useDashboardData } = await import('./useDashboardData');
        // Por ahora, mantener procesamiento simple
        const deepData = processDeepAnalysis(deepDemos, steamId);
        setDashboardData(deepData);
        setDataSource('deep');
        setLoading(false);
      } else if (quickMatches.length === 0) {
        // Si no hay ni stats rápidas ni análisis profundo
        console.log('[HybridDashboard] No hay datos disponibles aún');
        // Mostrar datos vacíos pero con el estado de descarga
        setDashboardData(getEmptyDashboardData());
        setDataSource(null);
        setLoading(false);
      }

    } catch (err) {
      console.error('[HybridDashboard] Error loading dashboard:', err);
      setError(err.message);
      setDashboardData(getEmptyDashboardData());
    } finally {
      setLoading(false);
    }
  }, [user, fetchQuickStats, fetchDeepAnalysis, fetchDownloadStatus]);

  // Efecto inicial
  useEffect(() => {
    loadDashboardData();
  }, [loadDashboardData]);

  // Polling para actualizar cuando las demos estén listas
  useEffect(() => {
    if (!downloadStatus?.is_downloading) {
      return; // No hay descargas en progreso
    }

    console.log('[HybridDashboard] Iniciando polling...');
    const interval = setInterval(async () => {
      const newStatus = await fetchDownloadStatus();
      setDownloadStatus(newStatus);

      // Si se completaron nuevas demos, recargar análisis profundo
      if (newStatus && !newStatus.is_downloading && newStatus.completed > 0) {
        console.log('[HybridDashboard] Nuevas demos completadas, recargando...');
        const deepDemos = await fetchDeepAnalysis();
        if (deepDemos.length > 0) {
          const deepData = processDeepAnalysis(deepDemos, user?.steam_id);
          setDashboardData(deepData);
          setDataSource('deep');
        }
      }
    }, 10000); // Cada 10 segundos

    return () => clearInterval(interval);
  }, [downloadStatus, fetchDownloadStatus, fetchDeepAnalysis, user]);

  return { 
    dashboardData, 
    loading, 
    error, 
    downloadStatus,
    dataSource,
    refresh: loadDashboardData 
  };
};

// Procesar stats rápidas de Steam API
const processQuickStats = (matches, userSteamId) => {
  console.log('[ProcessQuickStats] Processing', matches.length, 'matches');

  if (matches.length === 0) {
    return getEmptyDashboardData();
  }

  // Ordenar por fecha más reciente
  const sortedMatches = matches.sort((a, b) => b.match_time - a.match_time);

  // Calcular estadísticas principales
  const totalKills = sortedMatches.reduce((sum, m) => sum + (m.kills || 0), 0);
  const totalDeaths = sortedMatches.reduce((sum, m) => sum + (m.deaths || 0), 0);
  const avgKD = totalDeaths > 0 ? totalKills / totalDeaths : totalKills;
  
  const wins = sortedMatches.filter(m => m.result === 1).length;
  const winRate = (wins / sortedMatches.length) * 100;

  // MVPs como proxy de rendimiento general
  const avgMVPs = sortedMatches.reduce((sum, m) => sum + (m.mvps || 0), 0) / sortedMatches.length;

  // Calcular tendencias (últimas 5 vs anteriores 5)
  const recent = sortedMatches.slice(0, 5);
  const older = sortedMatches.slice(5, 10);

  const recentKD = recent.length > 0 ? 
    recent.reduce((sum, m) => sum + (m.kd_ratio || (m.kills / Math.max(m.deaths, 1))), 0) / recent.length : avgKD;
  const olderKD = older.length > 0 ?
    older.reduce((sum, m) => sum + (m.kd_ratio || (m.kills / Math.max(m.deaths, 1))), 0) / older.length : avgKD;

  const recentWinRate = recent.length > 0 ?
    (recent.filter(m => m.result === 1).length / recent.length) * 100 : winRate;
  const olderWinRate = older.length > 0 ?
    (older.filter(m => m.result === 1).length / older.length) * 100 : winRate;

  const mainStats = {
    headshot: { 
      value: 0, // No disponible en stats rápidas
      change: 0, 
      trend: "up",
      unavailable: true 
    },
    adr: { 
      value: 0, // No disponible en stats rápidas
      change: 0, 
      trend: "up",
      unavailable: true 
    },
    kd: {
      value: parseFloat(avgKD.toFixed(2)),
      change: parseFloat((recentKD - olderKD).toFixed(2)),
      trend: recentKD >= olderKD ? "up" : "down"
    },
    winRate: {
      value: Math.round(winRate),
      change: parseFloat((recentWinRate - olderWinRate).toFixed(1)),
      trend: recentWinRate >= olderWinRate ? "up" : "down"
    },
    totalKills
  };

  // Performance data (últimas 7 partidas)
  const performanceData = sortedMatches.slice(0, 7).reverse().map((match, index) => ({
    match: index + 1,
    kd: match.kd_ratio || (match.kills / Math.max(match.deaths, 1)),
    adr: 0, // No disponible
    hs: 0   // No disponible
  }));

  // Recent matches (últimas 4)
  const recentMatches = sortedMatches.slice(0, 4).map((match, index) => ({
    id: index + 1,
    map: match.map || 'unknown',
    result: match.result === 1 ? 'Victoria' : 'Derrota',
    score: `${match.rounds_won || 0}-${match.rounds_lost || 0}`,
    kd: match.kd_ratio || (match.kills / Math.max(match.deaths, 1)),
    adr: 0, // No disponible en stats rápidas
    hsp: 0, // No disponible
    kills: match.kills || 0,
    deaths: match.deaths || 0,
    assists: match.assists || 0,
    mvp: (match.mvps || 0) > 0,
    date: formatTimestamp(match.match_time),
    has_deep_analysis: match.has_deep_analysis || false
  }));

  // Map stats
  const mapData = {};
  sortedMatches.forEach(match => {
    const map = match.map || 'unknown';
    if (!mapData[map]) {
      mapData[map] = { wins: 0, losses: 0 };
    }
    if (match.result === 1) {
      mapData[map].wins++;
    } else {
      mapData[map].losses++;
    }
  });

  const mapStats = Object.entries(mapData)
    .map(([map, data]) => ({
      map,
      wins: data.wins,
      losses: data.losses,
      winRate: Math.round((data.wins / (data.wins + data.losses)) * 100)
    }))
    .sort((a, b) => (b.wins + b.losses) - (a.wins + a.losses))
    .slice(0, 4);

  return {
    mainStats,
    performanceData,
    recentMatches,
    weaponStats: [], // No disponible en stats rápidas
    mapStats,
    advancedStats: {
      multiKills: { double: 0, triple: 0, quad: 0, ace: 0 },
      clutchesWon: 0,
      entryFrags: 0,
      utilityDamage: 0,
      flashAssists: 0
    },
    totalMatches: sortedMatches.length,
    isQuickData: true // Flag para mostrar mensaje al usuario
  };
};

// Procesar análisis profundo (mismo que useDashboardData.js)
const processDeepAnalysis = (demos, userSteamId) => {
  // Reutilizar lógica existente de useDashboardData.js
  // Por simplicidad, importar o duplicar la función processDemos
  console.log('[ProcessDeepAnalysis] Processing', demos.length, 'demos');
  
  // TODO: Importar processDemos de useDashboardData.js
  // Por ahora, retornar estructura básica
  return {
    ...processQuickStats([], userSteamId),
    isQuickData: false
  };
};

// Formatear timestamp de Unix a fecha legible
const formatTimestamp = (timestamp) => {
  if (!timestamp) return 'Hace tiempo';
  
  const date = new Date(timestamp * 1000);
  const now = new Date();
  const diffMs = now - date;
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  const diffDays = Math.floor(diffHours / 24);

  if (diffHours < 1) return 'Hace menos de 1 hora';
  if (diffHours < 24) return `Hace ${diffHours} hora${diffHours > 1 ? 's' : ''}`;
  if (diffDays === 1) return 'Ayer';
  if (diffDays < 7) return `Hace ${diffDays} días`;
  if (diffDays < 30) return `Hace ${Math.floor(diffDays / 7)} semana${Math.floor(diffDays / 7) > 1 ? 's' : ''}`;
  return date.toLocaleDateString('es-ES', { day: 'numeric', month: 'short' });
};

// Datos vacíos
const getEmptyDashboardData = () => ({
  mainStats: {
    headshot: { value: 0, change: 0, trend: "up" },
    adr: { value: 0, change: 0, trend: "up" },
    kd: { value: 0, change: 0, trend: "up" },
    winRate: { value: 0, change: 0, trend: "up" },
    totalKills: 0
  },
  performanceData: [],
  recentMatches: [],
  weaponStats: [],
  mapStats: [],
  advancedStats: {
    multiKills: { double: 0, triple: 0, quad: 0, ace: 0 },
    clutchesWon: 0,
    entryFrags: 0,
    utilityDamage: 0,
    flashAssists: 0
  },
  totalMatches: 0,
  isQuickData: false
});
