import { useState, useEffect } from 'react';

const API_URL = process.env.REACT_APP_API_URL || (window.location.port === '3000' ? 'http://localhost:8000' : '');

export const usePerformanceData = (user) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  
  const [overview, setOverview] = useState(null);
  const [grenades, setGrenades] = useState(null);
  const [maps, setMaps] = useState(null);
  const [movement, setMovement] = useState(null);

  useEffect(() => {
    // If no user/steamid, don't fetch
    const steamId = user?.steamid || user?.steam_id || user?.steamID;
    if (!steamId) return;

    const fetchAllData = async () => {
      setLoading(true);
      try {
        // 1. Fetch Overview (Dashboard Stats - includes One-stop shop for General, Aim, Weapons, Maps)
        const overviewRes = await fetch(`${API_URL}/steam/get-dashboard-stats`, { credentials: 'include' });
        if (!overviewRes.ok) throw new Error('Failed to fetch overview stats');
        const overviewData = await overviewRes.json();
        setOverview(overviewData);

        // 2. Fetch Grenades (Specialized endpoint)
        let topMap = "de_dust2";
        if (overviewData.map_stats && overviewData.map_stats.length > 0) {
            topMap = overviewData.map_stats[0].map || "de_dust2";
        }

        const grenadeRes = await fetch(`${API_URL}/steam/get-aggregate-grenades?map_name=${topMap}`, { credentials: 'include' });
        const grenadeData = await grenadeRes.json();
        setGrenades(grenadeData);

        // 3. Movement / Heatmap (Also per map)
        const movementRes = await fetch(`${API_URL}/steam/get-movement-stats?map_name=${topMap}`, { credentials: 'include' });
        const movementData = await movementRes.json();
        setMovement(movementData || {});

        setError(null);
      } catch (err) {
        console.error("Error fetching performance data:", err);
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };

    fetchAllData();
  }, [user]);

  return {
    loading,
    error,
    overview,   // Includes: stats, aim_stats, weapon_stats, map_stats, recent_matches
    grenades,   // Aggregate grenade usage
    movement,   // Heatmap & Flow
  };
};
