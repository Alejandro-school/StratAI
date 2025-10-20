// frontend/src/hooks/useDashboardData.js
import { useState, useEffect } from 'react';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8000';

export const useDashboardData = (user) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [dashboardData, setDashboardData] = useState(null);

  useEffect(() => {
    // Verificar que el usuario tenga steam_id
    const steamId = user?.steam_id || user?.steamid || user?.steamID;
    
    if (!steamId) {
      console.log('Dashboard: No steam ID found in user object:', user);
      setLoading(false);
      return;
    }

    const fetchDashboardData = async () => {
      try {
        setLoading(true);
        
        console.log('Dashboard: Fetching demos for steam_id:', steamId);
        
        // Obtener demos procesadas
        const response = await fetch(`${API_URL}/steam/get-processed-demos`, {
          credentials: 'include',
        });

        if (!response.ok) {
          throw new Error('No se pudieron obtener las demos');
        }

        const data = await response.json();
        console.log('Dashboard: Received data:', data);
        
        const demos = data.demos || [];
        console.log('Dashboard: Number of demos:', demos.length);

        if (demos.length === 0) {
          console.log('Dashboard: No demos found');
          setDashboardData(getEmptyDashboardData());
          setLoading(false);
          return;
        }

        // Procesar datos
        const processedData = processDemos(demos, steamId);
        console.log('Dashboard: Processed data:', processedData);
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

// Procesar demos para obtener estadísticas del dashboard
const processDemos = (demos, userSteamId) => {
  console.log('Processing demos. User Steam ID:', userSteamId);
  console.log('Total demos to process:', demos.length);
  
  // Filtrar demos válidas y ordenar por fecha
  const sortedDemos = demos
    .filter(demo => demo.players && demo.players.length > 0)
    .sort((a, b) => new Date(b.date) - new Date(a.date));

  console.log('Sorted demos:', sortedDemos.length);

  // Encontrar el jugador en cada demo
  const userMatches = sortedDemos.map((demo, index) => {
    // Buscar el jugador con diferentes variantes del campo
    const player = demo.players.find(p => 
      p.steamID === userSteamId || 
      p.steam_id === userSteamId ||
      p.steamid === userSteamId
    );
    
    if (!player) {
      console.log(`Demo ${index}: Player not found. Available steamIDs:`, 
        demo.players.map(p => p.steamID || p.steam_id || p.steamid));
      return null;
    }

    console.log(`Demo ${index}: Player found in ${demo.map_name}`);

    return {
      matchId: demo.match_id,
      map: demo.map_name,
      date: demo.date,
      duration: demo.duration,
      teamScore: demo.team_score,
      opponentScore: demo.opponent_score,
      result: demo.team_score > demo.opponent_score ? 'W' : 'L',
      ...player
    };
  }).filter(Boolean);

  console.log('User matches found:', userMatches.length);

  if (userMatches.length === 0) {
    return getEmptyDashboardData();
  }

  // Calcular estadísticas principales
  const mainStats = calculateMainStats(userMatches);
  
  // Últimas 7 partidas para el gráfico
  const performanceData = userMatches.slice(0, 7).reverse().map((match, index) => ({
    match: index + 1,
    kd: match.kd_ratio || 0,
    adr: Math.round(match.adr || 0),
    hs: Math.round(match.hs_percentage || 0)
  }));

  // Últimas 4 partidas para la tabla
  const recentMatches = userMatches.slice(0, 4).map((match, index) => ({
    id: index + 1,
    map: match.map,
    result: match.result,
    score: `${match.teamScore}-${match.opponentScore}`,
    kd: match.kd_ratio || 0, // Mantener como número para formatear en el componente
    adr: Math.round(match.adr || 0),
    hsp: Math.round(match.hs_percentage || 0), // Cambiar "hs" a "hsp" para consistencia
    kills: match.kills || 0,
    deaths: match.deaths || 0,
    assists: match.assists || 0, // Añadir assists que faltaba
    mvp: false, // TODO: Detectar MVP si está disponible en los datos
    date: formatRelativeDate(match.date)
  }));

  // Estadísticas de armas reales basadas en los datos
  const weaponStats = calculateWeaponStats(sortedDemos, userSteamId);

  // Estadísticas por mapa
  const mapStats = calculateMapStats(userMatches);

  // Estadísticas avanzadas (clutches, multi-kills, etc.)
  const advancedStats = calculateAdvancedStats(userMatches);

  return {
    mainStats,
    performanceData,
    recentMatches,
    weaponStats,
    mapStats,
    advancedStats,
    totalMatches: userMatches.length
  };
};

// Calcular estadísticas principales
const calculateMainStats = (matches) => {
  if (matches.length === 0) {
    return {
      headshot: { value: 0, change: 0, trend: "up" },
      adr: { value: 0, change: 0, trend: "up" },
      kd: { value: 0, change: 0, trend: "up" },
      winRate: { value: 0, change: 0, trend: "up" },
      totalKills: 0
    };
  }

  // Promedios generales
  const totalKills = matches.reduce((sum, m) => sum + (m.kills || 0), 0);
  const avgHS = matches.reduce((sum, m) => sum + (m.hs_percentage || 0), 0) / matches.length;
  const avgADR = matches.reduce((sum, m) => sum + (m.adr || 0), 0) / matches.length;
  const avgKD = matches.reduce((sum, m) => sum + (m.kd_ratio || 0), 0) / matches.length;
  const wins = matches.filter(m => m.result === 'W').length;
  const winRate = (wins / matches.length) * 100;

  // Calcular tendencias comparando últimas 5 vs anteriores
  const recent = matches.slice(0, Math.min(5, matches.length));
  const older = matches.slice(5, Math.min(10, matches.length));

  const recentHS = recent.length > 0 ? recent.reduce((sum, m) => sum + (m.hs_percentage || 0), 0) / recent.length : avgHS;
  const olderHS = older.length > 0 ? older.reduce((sum, m) => sum + (m.hs_percentage || 0), 0) / older.length : avgHS;
  
  const recentADR = recent.length > 0 ? recent.reduce((sum, m) => sum + (m.adr || 0), 0) / recent.length : avgADR;
  const olderADR = older.length > 0 ? older.reduce((sum, m) => sum + (m.adr || 0), 0) / older.length : avgADR;
  
  const recentKD = recent.length > 0 ? recent.reduce((sum, m) => sum + (m.kd_ratio || 0), 0) / recent.length : avgKD;
  const olderKD = older.length > 0 ? older.reduce((sum, m) => sum + (m.kd_ratio || 0), 0) / older.length : avgKD;
  
  const recentWins = recent.filter(m => m.result === 'W').length;
  const recentWinRate = recent.length > 0 ? (recentWins / recent.length) * 100 : winRate;
  const olderWins = older.filter(m => m.result === 'W').length;
  const olderWinRate = older.length > 0 ? (olderWins / older.length) * 100 : winRate;

  return {
    headshot: {
      value: Math.round(avgHS),
      change: parseFloat((recentHS - olderHS).toFixed(1)),
      trend: recentHS >= olderHS ? "up" : "down"
    },
    adr: {
      value: Math.round(avgADR),
      change: parseFloat((recentADR - olderADR).toFixed(1)),
      trend: recentADR >= olderADR ? "up" : "down"
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
};

// Calcular estadísticas por mapa
const calculateMapStats = (matches) => {
  const mapData = {};

  matches.forEach(match => {
    if (!mapData[match.map]) {
      mapData[match.map] = { wins: 0, losses: 0 };
    }
    if (match.result === 'W') {
      mapData[match.map].wins++;
    } else {
      mapData[match.map].losses++;
    }
  });

  return Object.entries(mapData)
    .map(([map, data]) => ({
      map,
      wins: data.wins,
      losses: data.losses,
      winRate: Math.round((data.wins / (data.wins + data.losses)) * 100)
    }))
    .sort((a, b) => (b.wins + b.losses) - (a.wins + a.losses))
    .slice(0, 4); // Top 4 mapas más jugados
};

// Calcular estadísticas de armas basadas en datos reales
const calculateWeaponStats = (demos, userSteamId) => {
  const weaponKills = {};
  
  console.log('calculateWeaponStats: Processing', demos.length, 'demos for user', userSteamId);

  demos.forEach((demo, demoIdx) => {
    // DEBUG: Ver estructura completa de la demo
    console.log(`Demo ${demoIdx} estructura:`, Object.keys(demo));
    
    // Buscar el nombre del usuario en esta demo
    const userPlayer = demo.players?.find(p => 
      String(p.steamID || p.steam_id || '').trim() === String(userSteamId).trim()
    );
    
    if (!userPlayer) {
      console.log(`Demo ${demoIdx}: User not found`);
      return;
    }
    
    const userName = userPlayer.name;
    console.log(`Demo ${demoIdx}: User name is "${userName}"`);

    // DEBUG: Ver qué campos de eventos existen
    console.log(`Demo ${demoIdx} tiene:`, {
      event_logs: !!demo.event_logs,
      events: !!demo.events,
      EventLogs: !!demo.EventLogs
    });

    // Los event_logs pueden estar en diferentes ubicaciones
    let events = demo.event_logs || demo.EventLogs || demo.events || [];
    
    console.log(`Demo ${demoIdx}: event_logs type:`, typeof events, 'length:', Array.isArray(events) ? events.length : 'not array');
    
    // Si event_logs es un objeto con rounds, extraer todos los eventos
    if (events && typeof events === 'object' && events.events_by_round) {
      const allEvents = [];
      events.events_by_round.forEach(round => {
        if (round.events) {
          allEvents.push(...round.events);
        }
      });
      events = allEvents;
      console.log(`Demo ${demoIdx}: Extracted ${events.length} events from events_by_round`);
    }
    
    // Si events es un array directo
    if (Array.isArray(events)) {
      console.log(`Demo ${demoIdx}: Es un array directo con ${events.length} eventos`);
      
      // DEBUG: Mostrar algunos eventos de ejemplo
      if (events.length > 0) {
        console.log(`Demo ${demoIdx}: Primeros 3 eventos:`, events.slice(0, 3));
      }
    } else {
      console.log(`Demo ${demoIdx}: NO es un array, es:`, typeof events, events);
    }

    console.log(`Demo ${demoIdx}: Processing ${Array.isArray(events) ? events.length : 0} events`);

    let killCount = 0;
    events.forEach((event, eventIdx) => {
      if (event.event_type === 'Kill' || event.event_type === 'kill') {
        // DEBUG: Mostrar evento de kill
        console.log(`Demo ${demoIdx}, Event ${eventIdx}: Kill event:`, event);
        
        // Parsear el campo details que viene como string
        // Formato: "Killer=nombre, Victim=nombre, Weapon=arma, ..."
        const details = event.details || '';
        
        // Extraer el killer y el arma usando regex
        const killerMatch = details.match(/Killer=([^,]+)/);
        const weaponMatch = details.match(/Weapon=([^,]+)/);
        
        if (killerMatch && weaponMatch) {
          const killer = killerMatch[1].trim();
          const weapon = weaponMatch[1].trim();
          
          console.log(`Demo ${demoIdx}, Event ${eventIdx}: Killer="${killer}", Weapon="${weapon}", User="${userName}"`);
          
          // Verificar si el killer es el usuario
          if (killer === userName) {
            killCount++;
            weaponKills[weapon] = (weaponKills[weapon] || 0) + 1;
            console.log(`Demo ${demoIdx}, Event ${eventIdx}: ✅ MATCH! Kill count: ${killCount}, Weapon kills: ${weaponKills[weapon]}`);
          }
        } else {
          console.log(`Demo ${demoIdx}, Event ${eventIdx}: ❌ No se pudo parsear killer/weapon de: "${details}"`);
        }
      }
    });
    
    console.log(`Demo ${demoIdx}: Found ${killCount} kills for user`);
  });

  console.log('Total weapons found:', weaponKills);

  // Convertir a array y ordenar por kills
  const weaponArray = Object.entries(weaponKills)
    .map(([weapon, kills]) => ({
      weapon: cleanWeaponName(weapon),
      kills: kills,
      accuracy: 0 // No tenemos datos de precision por arma aún
    }))
    .sort((a, b) => b.kills - a.kills)
    .slice(0, 4); // Top 4 armas

  // Si no hay datos, devolver array vacío
  if (weaponArray.length === 0) {
    console.warn('No weapon data found');
    return [];
  }

  console.log('Top weapons:', weaponArray);
  return weaponArray;
};

// Limpiar nombres de armas (quitar prefijos weapon_, etc.)
const cleanWeaponName = (weapon) => {
  if (!weapon) return 'Desconocida';
  
  // Mapeo de nombres completos a nombres cortos
  const weaponNames = {
    // Nombres completos de los eventos
    'AK-47': 'AK-47',
    'M4A1': 'M4A4',
    'M4A4': 'M4A4',
    'M4A1-S': 'M4A1-S',
    'AWP': 'AWP',
    'Desert Eagle': 'Deagle',
    'Deagle': 'Deagle',
    'Glock-18': 'Glock',
    'USP-S': 'USP-S',
    'P2000': 'P2000',
    'FAMAS': 'FAMAS',
    'Galil AR': 'Galil',
    'AUG': 'AUG',
    'SG 553': 'SG 553',
    'MP9': 'MP9',
    'MAC-10': 'MAC-10',
    'P90': 'P90',
    'UMP-45': 'UMP-45',
    'Nova': 'Nova',
    'MAG-7': 'MAG-7',
    'Sawed-Off': 'Sawed-Off',
    'XM1014': 'XM1014',
    'Knife': 'Cuchillo',
    'Dual Berettas': 'Dual Berettas',
    'Five-SeveN': 'Five-Seven',
    'CZ75-Auto': 'CZ75',
    'Tec-9': 'Tec-9',
    'P250': 'P250',
    'R8 Revolver': 'R8',
    'HE Grenade': 'Granada HE',
    // Nombres técnicos de backup
    'weapon_ak47': 'AK-47',
    'weapon_m4a1': 'M4A4',
    'weapon_m4a1_silencer': 'M4A1-S',
    'weapon_awp': 'AWP',
    'weapon_deagle': 'Deagle',
  };

  // Intentar mapeo directo primero
  if (weaponNames[weapon]) {
    return weaponNames[weapon];
  }

  // Si no, limpiar el nombre
  return weapon.replace('weapon_', '').replace(/_/g, ' ').toUpperCase();
};

// Calcular estadísticas avanzadas (multi-kills, clutches, etc.)
const calculateAdvancedStats = (matches) => {
  if (matches.length === 0) {
    return {
      multiKills: { double: 0, triple: 0, quad: 0, ace: 0 },
      clutchesWon: 0,
      entryFrags: 0,
      utilityDamage: 0,
      flashAssists: 0
    };
  }

  const stats = {
    multiKills: { double: 0, triple: 0, quad: 0, ace: 0 },
    clutchesWon: 0,
    entryFrags: 0,
    utilityDamage: 0,
    flashAssists: 0
  };

  matches.forEach(match => {
    stats.multiKills.double += match.double_kills || 0;
    stats.multiKills.triple += match.triple_kills || 0;
    stats.multiKills.quad += match.quad_kills || 0;
    stats.multiKills.ace += match.ace || 0;
    stats.clutchesWon += match.clutch_wins || 0;
    stats.entryFrags += match.entry_kills || 0;
    stats.utilityDamage += match.utility_damage || 0;
    stats.flashAssists += match.flash_assists || 0;
  });

  return stats;
};

// Formatear fecha relativa
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

// Datos vacíos cuando no hay demos
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

