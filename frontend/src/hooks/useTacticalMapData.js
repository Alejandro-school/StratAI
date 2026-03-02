// frontend/src/hooks/useTacticalMapData.js
import { useState, useEffect } from 'react';
import { API_URL } from '../utils/api';

export const useTacticalMapData = (user) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [dashboardData, setDashboardData] = useState(null);

  useEffect(() => {
    const steamId = user?.steam_id || user?.steamid || user?.steamID;
    
    if (!steamId) {
      setLoading(false);
      return;
    }

    const fetchDashboardData = async () => {
      try {
        setLoading(true);

        const hasRefreshed = sessionStorage.getItem('dashboard_refreshed');
        const forceRefresh = !hasRefreshed ? 'true' : 'false';

        const response = await fetch(`${API_URL}/steam/get-dashboard-stats?force_refresh=${forceRefresh}`, {
          credentials: 'include',
        });
        
        if (!hasRefreshed) {
          sessionStorage.setItem('dashboard_refreshed', 'true');
        }

        if (!response.ok) {
          throw new Error('No se pudieron obtener las estadísticas');
        }

        const data = await response.json();
        
        if (data.stats.total_matches === 0) {
          setDashboardData(getEmptyDashboardData());
          setLoading(false);
          return;
        }

        const processedData = transformBackendData(data);
        setDashboardData(processedData);
        setError(null);
      } catch (err) {
        console.error('Error fetching dashboard data:', err);
        setError(err.message);
        setDashboardData(getEmptyDashboardData());
      } finally {
        setLoading(false);
      }
    };

    fetchDashboardData();
  }, [user]);

  return { dashboardData, loading, error };
};

const transformBackendData = (backendData) => {
  const { stats, recent_matches, weapon_stats, map_stats } = backendData;
  
  const recentMatches = recent_matches.slice(0, 5);
  const olderMatches = recent_matches.slice(5, 10);
  
  const recentHS = recentMatches.length > 0 
    ? recentMatches.reduce((sum, m) => sum + m.hs_percentage, 0) / recentMatches.length 
    : stats.avg_hs;
  const olderHS = olderMatches.length > 0 
    ? olderMatches.reduce((sum, m) => sum + m.hs_percentage, 0) / olderMatches.length 
    : stats.avg_hs;
  
  const recentADR = recentMatches.length > 0 
    ? recentMatches.reduce((sum, m) => sum + m.adr, 0) / recentMatches.length 
    : stats.avg_adr;
  const olderADR = olderMatches.length > 0 
    ? olderMatches.reduce((sum, m) => sum + m.adr, 0) / olderMatches.length 
    : stats.avg_adr;
  
  const recentKD = recentMatches.length > 0 
    ? recentMatches.reduce((sum, m) => sum + m.kd_ratio, 0) / recentMatches.length 
    : stats.avg_kd;
  const olderKD = olderMatches.length > 0 
    ? olderMatches.reduce((sum, m) => sum + m.kd_ratio, 0) / olderMatches.length 
    : stats.avg_kd;
  
  const recentWins = recentMatches.filter(m => m.result === 'W').length;
  const recentWR = recentMatches.length > 0 ? (recentWins / recentMatches.length) * 100 : stats.win_rate;
  const olderWins = olderMatches.filter(m => m.result === 'W').length;
  const olderWR = olderMatches.length > 0 ? (olderWins / olderMatches.length) * 100 : stats.win_rate;
  
  return {
    totalMatches: stats.total_matches,
    mainStats: {
      headshot: {
        value: Math.round(stats.avg_hs),
        change: parseFloat((recentHS - olderHS).toFixed(1)),
        trend: recentHS >= olderHS ? "up" : "down"
      },
      adr: {
        value: Math.round(stats.avg_adr),
        change: parseFloat((recentADR - olderADR).toFixed(1)),
        trend: recentADR >= olderADR ? "up" : "down"
      },
      kd: {
        value: parseFloat(stats.avg_kd.toFixed(2)),
        change: parseFloat((recentKD - olderKD).toFixed(2)),
        trend: recentKD >= olderKD ? "up" : "down"
      },
      winRate: {
        value: Math.round(stats.win_rate),
        change: parseFloat((recentWR - olderWR).toFixed(1)),
        trend: recentWR >= olderWR ? "up" : "down"
      },
      totalKills: stats.total_kills
    },
    performanceData: recent_matches.slice(0, 7).reverse().map((match, index) => ({
      match: index + 1,
      kd: match.kd_ratio || 0,
      adr: Math.round(match.adr || 0),
      hs: Math.round(match.hs_percentage || 0)
    })),
    recentMatches: recent_matches.slice(0, 4).map((match, index) => ({
      id: index + 1,
      map: match.map,
      result: match.result === 'W' ? 'Victoria' : 'Derrota',
      score: `${match.team_score}-${match.opponent_score}`,
      kd: match.kd_ratio || 0,
      adr: Math.round(match.adr || 0),
      hsp: Math.round(match.hs_percentage || 0),
      kills: match.kills || 0,
      deaths: match.deaths || 0,
      assists: match.assists || 0,
      date: formatRelativeDate(match.date)
    })),
    weaponStats: weapon_stats.map(w => ({
      weapon: cleanWeaponName(w.weapon),
      kills: w.kills,
      accuracy: 0
    })),
    mapStats: map_stats.slice(0, 4).map(m => ({
      map: m.map,
      wins: m.wins,
      losses: m.losses,
      winRate: Math.round(m.win_rate)
    })),
    advancedStats: calculateAdvancedStatsFromMatches(recent_matches)
  };
};

const cleanWeaponName = (weapon) => {
  if (!weapon) return 'Desconocida';
  return weapon.replace('weapon_', '').replace(/_/g, ' ').toUpperCase();
};

const calculateAdvancedStatsFromMatches = (matches) => {
  if (!matches || matches.length === 0) {
    return {
      multiKills: { double: 0, triple: 0, quad: 0, ace: 0 },
      clutchesWon: 0,
      entryFrags: 0,
      utilityDamage: 0,
      flashAssists: 0,
      avgHltvRating: 0,
      avgKAST: 0,
      totalOpeningDuels: 0,
      openingSuccessRate: 0,
      totalTradeKills: 0,
      avgUtilityEfficiency: 0
    };
  }

  const totals = {
    hltvRating: 0,
    kast: 0,
    openingDuelsWon: 0,
    openingDuelsLost: 0,
    tradeKills: 0,
    flashAssists: 0,
    clutches1v1: 0,
    clutches1v2: 0,
    enemiesFlashedPerFlash: 0,
    heDamagePerNade: 0,
    molotovDamagePerNade: 0
  };

  let matchCount = 0;

  matches.forEach(match => {
    if (!match) return;
    
    matchCount++;
    
    totals.hltvRating += match.hltv_rating || 0;
    totals.kast += match.kast || 0;

    totals.openingDuelsWon += match.opening_duels_won || 0;
    totals.openingDuelsLost += match.opening_duels_lost || 0;

    totals.tradeKills += match.trade_kills || 0;
    totals.flashAssists += match.flash_assists || 0;

    totals.clutches1v1 += match.clutches_1v1_won || 0;
    totals.clutches1v2 += match.clutches_1v2_won || 0;

    totals.enemiesFlashedPerFlash += match.enemies_flashed_per_flash || 0;
    totals.heDamagePerNade += match.he_damage_per_nade || 0;
    totals.molotovDamagePerNade += match.molotov_damage_per_nade || 0;
  });

  const avgHltvRating = matchCount > 0 ? totals.hltvRating / matchCount : 0;
  const avgKAST = matchCount > 0 ? totals.kast / matchCount : 0;
  const totalOpeningDuels = totals.openingDuelsWon + totals.openingDuelsLost;
  const openingSuccessRate = totalOpeningDuels > 0 
    ? (totals.openingDuelsWon / totalOpeningDuels) * 100 
    : 0;
  
  const avgUtilityEff = matchCount > 0 
    ? (totals.enemiesFlashedPerFlash + totals.heDamagePerNade + totals.molotovDamagePerNade) / 3 / matchCount
    : 0;

  return {
    multiKills: { double: 0, triple: 0, quad: 0, ace: 0 },
    clutchesWon: totals.clutches1v1 + totals.clutches1v2,
    entryFrags: totals.openingDuelsWon,
    utilityDamage: Math.round(totals.heDamagePerNade + totals.molotovDamagePerNade),
    flashAssists: totals.flashAssists,
    
    avgHltvRating: parseFloat(avgHltvRating.toFixed(2)),
    avgKAST: parseFloat(avgKAST.toFixed(1)),
    totalOpeningDuels,
    openingSuccessRate: parseFloat(openingSuccessRate.toFixed(1)),
    totalTradeKills: totals.tradeKills,
    avgUtilityEfficiency: parseFloat(avgUtilityEff.toFixed(2))
  };
};

const formatRelativeDate = (dateString) => {
  if (!dateString) return 'Hace tiempo';
  
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now - date;
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  const diffDays = Math.floor(diffHours / 24);

  if (diffHours < 1) return 'Hace menos de 1 hora';
  if (diffHours < 24) return `Hace ${diffHours} hora${diffHours > 1 ? 's' : ''}`;
  if (diffDays === 1) return 'Ayer';
  if (diffDays < 7) return `Hace ${diffDays} días`;
  if (diffDays < 30) return `Hace ${Math.floor(diffDays / 7)} semana${Math.floor(diffDays / 7) > 1 ? 's' : ''}`;
  return `Hace ${Math.floor(diffDays / 30)} mes${Math.floor(diffDays / 30) > 1 ? 'es' : ''}`;
};

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
  totalMatches: 0
});

