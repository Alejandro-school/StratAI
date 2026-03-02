import { useState, useEffect } from 'react';
import { API_URL } from '../utils/api';

const normalizeLegacyDashboardPayload = (legacy) => {
  const stats = legacy?.stats || {};
  const aimStats = legacy?.aim_stats || {};

  return {
    steam_id: legacy?.steam_id,
    overview: {
      total_matches: stats.total_matches || 0,
      wins: stats.wins || 0,
      losses: stats.losses || 0,
      win_rate: stats.win_rate || 0,
      kills: stats.total_kills || 0,
      deaths: stats.total_deaths || 0,
      assists: 0,
      total_damage: 0,
      kd_ratio: stats.avg_kd || 0,
      adr: stats.avg_adr || 0,
      hs_pct: stats.avg_hs || 0,
      kast: 0,
      hltv_rating: 0,
      impact_rating: 0,
    },
    sides: { ct_rating: 0, t_rating: 0, ct_adr: 0, t_adr: 0 },
    aim: {
      accuracy_overall: aimStats.accuracy_overall || 0,
      time_to_damage_avg_ms: aimStats.time_to_damage_avg_ms || 0,
      crosshair_placement_avg_error: aimStats.crosshair_placement_avg_error || 0,
      crosshair_placement_peek: 0,
      crosshair_placement_hold: 0,
      shots_fired: 0,
      shots_hit: 0,
      body_part_hits: aimStats.body_part_hits || {},
    },
    combat: {
      opening_duels_attempted: 0,
      opening_duels_won: 0,
      opening_duels_lost: 0,
      opening_success_rate: 0,
      trade_kills: 0,
      traded_deaths: 0,
      flash_assists: 0,
      clutches: { '1v1': 0, '1v2': 0, '1v3': 0, '1v4': 0, '1v5': 0 },
      multikills: { '2k': 0, '3k': 0, '4k': 0, ace: 0 },
    },
    utility: {
      grenades_thrown_total: 0,
      flashes_thrown: 0,
      enemies_flashed_total: 0,
      flash_duration_total: 0,
      enemies_flashed_per_flash: 0,
      blind_time_per_flash: 0,
      he_thrown: 0,
      he_damage_per_nade: 0,
      molotovs_thrown: 0,
      molotov_damage_per_nade: 0,
      smokes_thrown: 0,
      utility_damage: 0,
      grenade_damage: { he: 0, molotov: 0, flash: 0, smoke: 0 },
    },
    weapons: legacy?.weapon_stats || [],
    maps: (legacy?.map_stats || []).map((item) => ({
      map: item.map,
      wins: item.wins,
      losses: item.losses,
      win_rate: item.win_rate,
      avg_kd: 0,
      avg_adr: 0,
      avg_rating: 0,
      matches: (item.wins || 0) + (item.losses || 0),
    })),
    match_history: legacy?.recent_matches || [],
    economy: { rounds_survived: 0, total_rounds: 0, survival_rate: 0 },
  };
};

export const usePerformanceData = (user) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [performance, setPerformance] = useState(null);
  const [reloadKey, setReloadKey] = useState(0);

  const retry = () => setReloadKey((current) => current + 1);

  useEffect(() => {
    // If no user/steamid, don't fetch
    const steamId = user?.steamid || user?.steam_id || user?.steamID;
    if (!steamId) {
      setLoading(false);
      setPerformance(null);
      return;
    }

    const abortController = new AbortController();

    const fetchPerformanceData = async () => {
      setLoading(true);
      try {
        const res = await fetch(`${API_URL}/steam/performance-overview`, {
          credentials: 'include',
          signal: abortController.signal,
        });

        if (!res.ok) {
          const legacyRes = await fetch(`${API_URL}/steam/get-dashboard-stats`, {
            credentials: 'include',
            signal: abortController.signal,
          });
          if (!legacyRes.ok) throw new Error('Failed to fetch performance stats');
          const legacyData = await legacyRes.json();
          setPerformance(normalizeLegacyDashboardPayload(legacyData));
        } else {
          const data = await res.json();
          setPerformance(data);
        }

        setError(null);
      } catch (err) {
        if (err.name === 'AbortError') return;
        console.error("Error fetching performance data:", err);
        setError(err.message);
      } finally {
        if (!abortController.signal.aborted) {
          setLoading(false);
        }
      }
    };

    fetchPerformanceData();

    return () => abortController.abort();
  }, [user, reloadKey]);

  return {
    loading,
    error,
    performance,
    retry,
  };
};
