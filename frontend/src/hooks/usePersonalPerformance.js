// frontend/src/hooks/usePersonalPerformance.js
import { useState, useEffect } from 'react';

const API_URL = process.env.REACT_APP_API_URL || (window.location.port === '3000' ? 'http://localhost:8000' : '');

export const usePersonalPerformance = (user) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [data, setData] = useState(null);

  useEffect(() => {
    const steamId = user?.steam_id || user?.steamid || user?.steamID;
    
    if (!steamId) {
      setLoading(false);
      return;
    }

    const fetchData = async () => {
      try {
        setLoading(true);
        
        // Obtener demos procesadas
        const response = await fetch(`${API_URL}/get-processed-demos?steam_id=${steamId}`);
        
        if (!response.ok) {
          throw new Error('No se pudieron obtener las demos');
        }

        const result = await response.json();
        const demos = result.demos || [];

        if (demos.length === 0) {
          setData(null);
          setLoading(false);
          return;
        }

        // Procesar datos para desempeño personal detallado
        const processedData = processPersonalData(demos, steamId);
        setData(processedData);
        setError(null);
      } catch (err) {
        console.error('Error fetching personal performance:', err);
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [user]);

  return { data, loading, error };
};

const processPersonalData = (demos, userSteamId) => {
  // Filtrar y obtener datos del jugador de cada demo
  const userMatches = demos
    .filter(demo => demo.players && demo.players.length > 0)
    .map(demo => {
      const player = demo.players.find(p => 
        String(p.steamID || p.steam_id || p.steamid).trim() === String(userSteamId).trim()
      );
      
      if (!player) return null;

      return {
        matchId: demo.match_id,
        map: demo.map_name,
        date: demo.date,
        kills: player.kills || 0,
        deaths: player.deaths || 0,
        assists: player.assists || 0,
        kd: player.kd_ratio || 0,
        adr: player.adr || 0,
        hs: player.hs_percentage || 0,
        mvps: player.mvp || 0,
        clutches: player.clutch_wins || 0,
        entryKills: player.entry_kills || 0,
        tradeKills: player.trade_kills || 0,
        utilityDamage: player.utility_damage || 0,
        flashAssists: player.flash_assists || 0,
        result: demo.team_score > demo.opponent_score ? 'Victoria' : 'Derrota'
      };
    })
    .filter(Boolean)
    .sort((a, b) => new Date(b.date) - new Date(a.date));

  if (userMatches.length === 0) return null;

  // Calcular promedios y tendencias
  const totalMatches = userMatches.length;
  const recentMatches = userMatches.slice(0, 10);
  const olderMatches = userMatches.slice(10, 20);

  // KPIs con comparación reciente vs anterior
  const summary = {
    kd: calculateMetricWithTrend(recentMatches, olderMatches, 'kd'),
    adr: calculateMetricWithTrend(recentMatches, olderMatches, 'adr'),
    hs: calculateMetricWithTrend(recentMatches, olderMatches, 'hs'),
    impact: calculateImpact(recentMatches, olderMatches),
    winRate: calculateWinRate(recentMatches, olderMatches),
    clutches: {
      value: recentMatches.reduce((sum, m) => sum + m.clutches, 0),
      change: recentMatches.reduce((sum, m) => sum + m.clutches, 0) - 
              (olderMatches.length > 0 ? olderMatches.reduce((sum, m) => sum + m.clutches, 0) : 0),
      trend: recentMatches.reduce((sum, m) => sum + m.clutches, 0) >= 
             (olderMatches.length > 0 ? olderMatches.reduce((sum, m) => sum + m.clutches, 0) : 0) ? 'up' : 'down'
    }
  };

  // Tendencias por semana (últimas 4 semanas)
  const trends = calculateWeeklyTrends(userMatches);

  // Top armas con estadísticas detalladas
  const topWeapons = calculateWeaponStats(demos, userSteamId);

  return {
    summary,
    trends,
    topWeapons,
    totalMatches,
    recentPerformance: recentMatches.slice(0, 5)
  };
};

const calculateMetricWithTrend = (recent, older, metric) => {
  const recentAvg = recent.reduce((sum, m) => sum + m[metric], 0) / recent.length;
  const olderAvg = older.length > 0 ? older.reduce((sum, m) => sum + m[metric], 0) / older.length : recentAvg;
  
  return {
    value: recentAvg,
    change: recentAvg - olderAvg,
    trend: recentAvg >= olderAvg ? 'up' : 'down'
  };
};

const calculateImpact = (recent, older) => {
  // Impact = (K + A) / R * 2.0 - D / R
  const calcImpact = (matches) => {
    const totalRounds = matches.length * 24; // Aproximado
    const totalKills = matches.reduce((sum, m) => sum + m.kills, 0);
    const totalAssists = matches.reduce((sum, m) => sum + m.assists, 0);
    const totalDeaths = matches.reduce((sum, m) => sum + m.deaths, 0);
    return ((totalKills + totalAssists) / totalRounds * 2.0 - totalDeaths / totalRounds);
  };

  const recentImpact = calcImpact(recent);
  const olderImpact = older.length > 0 ? calcImpact(older) : recentImpact;

  return {
    value: recentImpact,
    change: recentImpact - olderImpact,
    trend: recentImpact >= olderImpact ? 'up' : 'down'
  };
};

const calculateWinRate = (recent, older) => {
  const recentWins = recent.filter(m => m.result === 'Victoria').length;
  const recentRate = (recentWins / recent.length) * 100;
  
  const olderWins = older.length > 0 ? older.filter(m => m.result === 'Victoria').length : recentWins;
  const olderRate = older.length > 0 ? (olderWins / older.length) * 100 : recentRate;

  return {
    value: recentRate,
    change: recentRate - olderRate,
    trend: recentRate >= olderRate ? 'up' : 'down'
  };
};

const calculateWeeklyTrends = (matches) => {
  // Dividir en 4 grupos (semanas aproximadas)
  const weeksData = [];
  const matchesPerWeek = Math.ceil(matches.length / 4);

  for (let i = 0; i < 4; i++) {
    const weekMatches = matches.slice(i * matchesPerWeek, (i + 1) * matchesPerWeek);
    if (weekMatches.length === 0) continue;

    weeksData.push({
      period: `Sem ${4 - i}`,
      kd: weekMatches.reduce((sum, m) => sum + m.kd, 0) / weekMatches.length,
      adr: Math.round(weekMatches.reduce((sum, m) => sum + m.adr, 0) / weekMatches.length),
      hs: Math.round(weekMatches.reduce((sum, m) => sum + m.hs, 0) / weekMatches.length)
    });
  }

  return weeksData.reverse();
};

const calculateWeaponStats = (demos, userSteamId) => {
  const weaponData = {};

  demos.forEach(demo => {
    const userPlayer = demo.players?.find(p => 
      String(p.steamID || p.steam_id).trim() === String(userSteamId).trim()
    );
    
    if (!userPlayer) return;

    const userName = userPlayer.name;
    let events = demo.event_logs || demo.events || [];

    if (events && typeof events === 'object' && events.events_by_round) {
      const allEvents = [];
      events.events_by_round.forEach(round => {
        if (round.events) allEvents.push(...round.events);
      });
      events = allEvents;
    }

    if (Array.isArray(events)) {
      events.forEach(event => {
        if (event.event_type === 'Kill' || event.event_type === 'kill') {
          const details = event.details || '';
          const killerMatch = details.match(/Killer=([^,]+)/);
          const weaponMatch = details.match(/Weapon=([^,]+)/);
          const headshotMatch = details.match(/Headshot=(true|false)/);

          if (killerMatch && weaponMatch && killerMatch[1].trim() === userName) {
            const weapon = weaponMatch[1].trim();
            const isHeadshot = headshotMatch && headshotMatch[1] === 'true';

            if (!weaponData[weapon]) {
              weaponData[weapon] = { kills: 0, headshots: 0, damage: 0 };
            }

            weaponData[weapon].kills++;
            if (isHeadshot) weaponData[weapon].headshots++;
            weaponData[weapon].damage += 100; // Aproximado
          }
        }
      });
    }
  });

  return Object.entries(weaponData)
    .map(([weapon, stats]) => ({
      weapon: cleanWeaponName(weapon),
      accuracy: 42.5, // Placeholder
      hs: stats.kills > 0 ? ((stats.headshots / stats.kills) * 100).toFixed(1) : 0,
      avgDamage: stats.kills > 0 ? (stats.damage / stats.kills).toFixed(1) : 0,
      kills: stats.kills
    }))
    .sort((a, b) => b.kills - a.kills)
    .slice(0, 5);
};

const cleanWeaponName = (weapon) => {
  const weaponNames = {
    'AK-47': 'AK-47',
    'M4A1': 'M4A4',
    'M4A4': 'M4A4',
    'M4A1-S': 'M4A1-S',
    'AWP': 'AWP',
    'Desert Eagle': 'Deagle',
    'Glock-18': 'Glock',
    'USP-S': 'USP-S',
  };
  return weaponNames[weapon] || weapon;
};

