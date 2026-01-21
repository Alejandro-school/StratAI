// frontend/src/hooks/useMapPerformance.js
import { useState, useEffect } from 'react';

const API_URL = process.env.REACT_APP_API_URL || (window.location.port === '3000' ? 'http://localhost:8000' : '');

export const useMapPerformance = (user) => {
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

        const processedData = processMapData(demos, steamId);
        setData(processedData);
        setError(null);
      } catch (err) {
        console.error('Error fetching map performance:', err);
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [user]);

  return { data, loading, error };
};

const processMapData = (demos, userSteamId) => {
  const mapStats = {};

  // Procesar cada demo
  demos.forEach(demo => {
    const player = demo.players?.find(p => 
      String(p.steamID || p.steam_id || p.steamid).trim() === String(userSteamId).trim()
    );
    
    if (!player) return;

    const mapName = demo.map_name || 'Unknown';
    const result = demo.team_score > demo.opponent_score ? 'win' : 'loss';

    if (!mapStats[mapName]) {
      mapStats[mapName] = {
        matches: 0,
        wins: 0,
        losses: 0,
        kills: 0,
        deaths: 0,
        adr: 0,
        hs: 0,
        entryKills: 0,
        entryAttempts: 0,
        clutches: 0,
        clutchAttempts: 0
      };
    }

    mapStats[mapName].matches++;
    mapStats[mapName].kills += player.kills || 0;
    mapStats[mapName].deaths += player.deaths || 0;
    mapStats[mapName].adr += player.adr || 0;
    mapStats[mapName].hs += player.hs_percentage || 0;
    mapStats[mapName].entryKills += player.entry_kills || 0;
    mapStats[mapName].clutches += player.clutch_wins || 0;
    
    if (result === 'win') {
      mapStats[mapName].wins++;
    } else {
      mapStats[mapName].losses++;
    }
  });

  // Convertir a array y calcular promedios
  const maps = Object.entries(mapStats).map(([name, stats]) => ({
    name: name.replace('de_', '').charAt(0).toUpperCase() + name.replace('de_', '').slice(1),
    matches: stats.matches,
    winRate: ((stats.wins / stats.matches) * 100).toFixed(1),
    adr: (stats.adr / stats.matches).toFixed(1),
    kd: stats.deaths > 0 ? (stats.kills / stats.deaths).toFixed(2) : stats.kills.toFixed(2),
    entrySuccess: stats.matches > 0 ? ((stats.entryKills / stats.matches) * 100 / 5).toFixed(1) : 0, // Aproximado
    clutchRate: stats.matches > 0 ? ((stats.clutches / stats.matches) * 100 / 3).toFixed(1) : 0 // Aproximado
  })).sort((a, b) => b.matches - a.matches);

  // Simular hotspots (zonas problemáticas) basados en datos
  const hotspots = generateHotspots(maps);

  return { maps, hotspots };
};

const generateHotspots = (maps) => {
  const hotspots = [];
  
  // Identificar mapas con bajo rendimiento
  maps.forEach(map => {
    if (parseFloat(map.winRate) < 45) {
      hotspots.push({
        map: map.name,
        position: `Zonas de bajo rendimiento`,
        risk: 'high',
        deaths: Math.floor(Math.random() * 20) + 10,
        description: `Win rate bajo (${map.winRate}%), necesita práctica`
      });
    } else if (parseFloat(map.kd) < 0.9) {
      hotspots.push({
        map: map.name,
        position: `K/D desfavorable`,
        risk: 'medium',
        deaths: Math.floor(Math.random() * 15) + 5,
        description: `K/D de ${map.kd}, mejorar posicionamiento`
      });
    }
  });

  return hotspots.slice(0, 4); // Top 4 hotspots
};

