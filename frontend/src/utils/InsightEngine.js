// frontend/src/utils/InsightEngine.js
// Generates varied, actionable insights from callout statistics

/**
 * Insight types with priority and styling info
 */
export const INSIGHT_TYPES = {
  WARNING: { id: 'warning', priority: 1, icon: 'AlertTriangle', color: '#ef4444' },
  TACTICAL: { id: 'tactical', priority: 2, icon: 'Crosshair', color: '#8b5cf6' },
  WEAPON: { id: 'weapon', priority: 3, icon: 'Swords', color: '#f59e0b' },
  STRENGTH: { id: 'strength', priority: 4, icon: 'TrendingUp', color: '#22c55e' },
  INFO: { id: 'info', priority: 5, icon: 'Info', color: '#3b82f6' }
};

/**
 * Generate a single, prioritized insight for a callout
 * @param {Object} stats - Stats for one callout
 * @param {string} name - Name of the callout
 * @returns {Array} Array with 0-2 insight objects (max)
 */
export const generateCalloutInsights = (stats, name) => {
  if (!stats) return [];
  
  const {
    win_rate,
    kd,
    sample_size,
    weapon_stats = [],
    context_stats = {},
    ct_t_split = {},
    flash_death_pct = 0,
    avg_distance
  } = stats;
  
  // Not enough data
  if (sample_size < 3) {
    return [{
      type: INSIGHT_TYPES.INFO,
      text: `Necesitas más duelos en ${name} para obtener insights útiles.`
    }];
  }
  
  // Collect all possible insights, then pick the most relevant
  const candidates = [];
  
  // === WEAPON-BASED INSIGHTS ===
  if (weapon_stats.length >= 2) {
    const bestWeapon = weapon_stats.find(w => w.kd >= 1.5 && w.kills >= 2);
    const worstWeapon = weapon_stats.find(w => w.kd < 0.8 && w.deaths >= 2);
    
    if (bestWeapon && worstWeapon && bestWeapon.weapon !== worstWeapon.weapon) {
      candidates.push({
        type: INSIGHT_TYPES.WEAPON,
        priority: 2,
        text: `En ${name} tienes gran dominio con ${bestWeapon.weapon}. Prioriza esta arma cuando juegues aquí.`
      });
    } else if (bestWeapon) {
      candidates.push({
        type: INSIGHT_TYPES.STRENGTH,
        priority: 4,
        text: `El ${bestWeapon.weapon} es tu arma más efectiva en ${name} con ${bestWeapon.kills} kills.`
      });
    }
    
    // Rifle vs SMG insight
    const rifles = weapon_stats.filter(w => 
      ['AK-47', 'M4A4', 'M4A1-S', 'AWP', 'SG 553', 'AUG'].includes(w.weapon)
    );
    const smgs = weapon_stats.filter(w => 
      ['MAC-10', 'MP9', 'UMP-45', 'MP7', 'P90', 'PP-Bizon'].includes(w.weapon)
    );
    
    if (rifles.length > 0 && smgs.length > 0) {
      const rifleKD = rifles.reduce((sum, w) => sum + w.kills, 0) / Math.max(1, rifles.reduce((sum, w) => sum + w.deaths, 0));
      const smgKD = smgs.reduce((sum, w) => sum + w.kills, 0) / Math.max(1, smgs.reduce((sum, w) => sum + w.deaths, 0));
      
      if (smgKD > rifleKD * 1.5) {
        candidates.push({
          type: INSIGHT_TYPES.TACTICAL,
          priority: 3,
          text: `Los duelos en ${name} son de corta distancia. Las SMGs te funcionan mejor que los rifles aquí.`
        });
      } else if (rifleKD > smgKD * 1.5) {
        candidates.push({
          type: INSIGHT_TYPES.TACTICAL,
          priority: 3,
          text: `${name} favorece duelos de distancia. Mantén posiciones donde puedas usar rifles con ventaja.`
        });
      }
    }
  }
  
  // === CT vs T INSIGHTS ===
  const ctTotal = (ct_t_split.ct_kills || 0) + (ct_t_split.ct_deaths || 0);
  const tTotal = (ct_t_split.t_kills || 0) + (ct_t_split.t_deaths || 0);
  
  if (ctTotal >= 3 && tTotal >= 3) {
    const ctWR = ct_t_split.ct_kills / Math.max(1, ctTotal) * 100;
    const tWR = ct_t_split.t_kills / Math.max(1, tTotal) * 100;
    
    if (ctWR > tWR + 25) {
      candidates.push({
        type: INSIGHT_TYPES.TACTICAL,
        priority: 2,
        text: `Juegas mejor como CT en ${name}. Como defensor, esta es una de tus mejores posiciones.`
      });
    } else if (tWR > ctWR + 25) {
      candidates.push({
        type: INSIGHT_TYPES.TACTICAL,
        priority: 2,
        text: `Eres más efectivo atacando ${name}. Prioriza entrar por aquí en tus ejecuciones.`
      });
    }
  }
  
  // === CONTEXT-BASED INSIGHTS ===
  const openingAttempts = context_stats.opening_attempts || 0;
  const openingKills = context_stats.opening_kills || 0;
  
  if (openingAttempts >= 3) {
    const openingWR = (openingKills / openingAttempts) * 100;
    
    if (openingWR >= 60) {
      candidates.push({
        type: INSIGHT_TYPES.STRENGTH,
        priority: 3,
        text: `Tienes un ${Math.round(openingWR)}% de efectividad en primeros duelos. Tu agresividad en ${name} funciona.`
      });
    } else if (openingWR <= 30 && openingAttempts >= 4) {
      candidates.push({
        type: INSIGHT_TYPES.WARNING,
        priority: 1,
        text: `Los primeros duelos en ${name} no te favorecen. Deja que un compañero entre primero.`
      });
    }
  }
  
  // Trade insights
  const tradeKills = context_stats.trade_kills || 0;
  const tradeDeaths = context_stats.trade_deaths || 0;
  
  if (tradeKills >= 2 && tradeKills > tradeDeaths) {
    candidates.push({
      type: INSIGHT_TYPES.STRENGTH,
      priority: 4,
      text: `Buen trabajo cubriendo a tu equipo en ${name}. Has conseguido ${tradeKills} trades.`
    });
  } else if (tradeDeaths >= 3 && tradeDeaths > tradeKills * 2) {
    candidates.push({
      type: INSIGHT_TYPES.WARNING,
      priority: 1,
      text: `Te están tradeando mucho en ${name}. Asegúrate de tener cobertura antes de peekear.`
    });
  }
  
  // Smoke kills
  if ((context_stats.smoke_kills || 0) >= 2) {
    candidates.push({
      type: INSIGHT_TYPES.STRENGTH,
      priority: 5,
      text: `Has conseguido ${context_stats.smoke_kills} kills a través de humo en ${name}. Buen timing.`
    });
  }
  
  // === FLASH VULNERABILITY ===
  if (flash_death_pct > 30) {
    candidates.push({
      type: INSIGHT_TYPES.WARNING,
      priority: 1,
      text: `Mueres flasheado el ${Math.round(flash_death_pct)}% de las veces en ${name}. Cambia de posición tras escuchar granadas.`
    });
  }
  
  // === DISTANCE-BASED INSIGHT ===
  if (avg_distance) {
    if (avg_distance > 1000) {
      candidates.push({
        type: INSIGHT_TYPES.TACTICAL,
        priority: 4,
        text: `Los duelos en ${name} son de larga distancia. El AWP o rifles con scope te darán ventaja.`
      });
    } else if (avg_distance < 400) {
      candidates.push({
        type: INSIGHT_TYPES.TACTICAL,
        priority: 4,
        text: `Los enfrentamientos en ${name} son de corta distancia. SMGs o escopetas son opciones viables.`
      });
    }
  }
  
  // === OVERALL PERFORMANCE ===
  if (win_rate >= 60 && kd >= 1.5 && sample_size >= 5) {
    candidates.push({
      type: INSIGHT_TYPES.STRENGTH,
      priority: 5,
      text: `${name} es una de tus zonas más fuertes con ${win_rate}% de victorias.`
    });
  } else if (win_rate <= 35 && sample_size >= 5) {
    candidates.push({
      type: INSIGHT_TYPES.WARNING,
      priority: 1,
      text: `Solo ganas el ${win_rate}% de duelos en ${name}. Considera cambiar tu forma de jugar esta zona.`
    });
  }
  
  // Sort by priority and return top 1-2
  candidates.sort((a, b) => a.priority - b.priority);
  
  return candidates.slice(0, 1).map(c => ({
    type: c.type,
    text: c.text
  }));
};

/**
 * Generate global insights from all callout data
 */
export const generateGlobalInsights = (allCalloutStats) => {
  const insights = [];
  const callouts = Object.entries(allCalloutStats);
  
  if (callouts.length === 0) return insights;
  
  const significantCallouts = callouts.filter(([, s]) => (s.kills + s.deaths) >= 3);
  
  if (significantCallouts.length === 0) return insights;
  
  // Best and worst zones
  const sortedByWR = [...significantCallouts].sort((a, b) => b[1].win_rate - a[1].win_rate);
  
  if (sortedByWR.length >= 3) {
    const best = sortedByWR[0];
    const worst = sortedByWR[sortedByWR.length - 1];
    
    if (best[1].win_rate - worst[1].win_rate > 20) {
      insights.push({
        type: INSIGHT_TYPES.TACTICAL,
        text: `Tu mejor zona: ${best[0]} (${best[1].win_rate}% WR). Peor: ${worst[0]} (${worst[1].win_rate}% WR).`,
        global: true
      });
    }
  }
  
  // Overall weapon preference
  const allWeapons = {};
  for (const [, stats] of significantCallouts) {
    for (const w of (stats.weapon_stats || [])) {
      if (!allWeapons[w.weapon]) {
        allWeapons[w.weapon] = { kills: 0, deaths: 0 };
      }
      allWeapons[w.weapon].kills += w.kills;
      allWeapons[w.weapon].deaths += w.deaths;
    }
  }
  
  const weaponList = Object.entries(allWeapons)
    .map(([weapon, stats]) => ({
      weapon,
      kd: stats.deaths > 0 ? stats.kills / stats.deaths : stats.kills,
      total: stats.kills + stats.deaths
    }))
    .filter(w => w.total >= 5)
    .sort((a, b) => b.kd - a.kd);
  
  if (weaponList.length >= 2) {
    const best = weaponList[0];
    const worst = weaponList[weaponList.length - 1];
    
    if (best.kd > 1.3 && worst.kd < 0.8) {
      insights.push({
        type: INSIGHT_TYPES.WEAPON,
        text: `Mejor arma global: ${best.weapon} (${best.kd.toFixed(1)} K/D). Evita ${worst.weapon} (${worst.kd.toFixed(1)} K/D).`,
        global: true
      });
    }
  }
  
  return insights;
};

/**
 * Get all insights sorted by priority
 */
export const getAllInsights = (allCalloutStats) => {
  const allInsights = [];
  
  for (const [calloutName, stats] of Object.entries(allCalloutStats)) {
    const calloutInsights = generateCalloutInsights(stats, calloutName);
    allInsights.push(...calloutInsights);
  }
  
  const globalInsights = generateGlobalInsights(allCalloutStats);
  allInsights.push(...globalInsights);
  
  return allInsights.sort((a, b) => a.type.priority - b.type.priority);
};

/**
 * Get top N most important insights
 */
export const getTopInsights = (allCalloutStats, limit = 5) => {
  return getAllInsights(allCalloutStats).slice(0, limit);
};

export default {
  INSIGHT_TYPES,
  generateCalloutInsights,
  generateGlobalInsights,
  getAllInsights,
  getTopInsights
};
