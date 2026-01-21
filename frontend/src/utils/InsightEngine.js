// frontend/src/utils/InsightEngine.js
// Enhanced Insight Generator - Varied, actionable insights from callout statistics

/**
 * Insight types with priority and styling info
 * Lower priority = more important (shown first)
 */
export const INSIGHT_TYPES = {
  CRITICAL: { id: 'critical', priority: 0, icon: 'AlertOctagon', color: '#dc2626' },
  WARNING: { id: 'warning', priority: 1, icon: 'AlertTriangle', color: '#ef4444' },
  TACTICAL: { id: 'tactical', priority: 2, icon: 'Crosshair', color: '#8b5cf6' },
  WEAPON: { id: 'weapon', priority: 3, icon: 'Swords', color: '#f59e0b' },
  TIMING: { id: 'timing', priority: 4, icon: 'Clock', color: '#06b6d4' },
  STRENGTH: { id: 'strength', priority: 5, icon: 'TrendingUp', color: '#22c55e' },
  UTILITY: { id: 'utility', priority: 6, icon: 'Zap', color: '#a855f7' },
  POSITIONING: { id: 'positioning', priority: 7, icon: 'Move', color: '#3b82f6' },
  INFO: { id: 'info', priority: 8, icon: 'Info', color: '#64748b' }
};

/**
 * Generate varied insights for a callout
 * @param {Object} stats - Stats for one callout
 * @param {string} name - Name of the callout
 * @returns {Array} Array with insights
 */
export const generateCalloutInsights = (stats, name) => {
  if (!stats) return [];
  
  const {
    win_rate = 0,
    kd = 0,
    sample_size = 0,
    kills = 0,
    deaths = 0,
    weapon_stats = [],
    context_stats = {},
    ct_t_split = {},
    flash_death_pct = 0,
    avg_distance,
    avg_time_to_damage
  } = stats;
  
  // Not enough data
  if (sample_size < 3) {
    return [{
      type: INSIGHT_TYPES.INFO,
      text: `Más duelos en ${name} revelarán patrones útiles.`,
      category: 'data'
    }];
  }
  
  const candidates = [];
  
  // === CRITICAL ISSUES (Priority 0) ===
  if (win_rate <= 25 && sample_size >= 8) {
    candidates.push({
      type: INSIGHT_TYPES.CRITICAL,
      priority: 0,
      text: `⚠️ ${name} es un punto crítico: solo ${win_rate}% de victorias. Cambia tu enfoque o evita esta zona.`,
      category: 'critical'
    });
  }
  
  // === FLASH & UTILITY VULNERABILITIES ===
  if (flash_death_pct >= 25) {
    candidates.push({
      type: INSIGHT_TYPES.WARNING,
      priority: 1,
      text: `Te ciegan antes de morir ${Math.round(flash_death_pct)}% de veces en ${name}. Usa una posición anti-flash.`,
      category: 'utility'
    });
  }
  
  // === TRADE ANALYSIS ===
  const tradeKills = context_stats.trade_kills || 0;
  const tradeDeaths = context_stats.trade_deaths || 0;
  
  if (tradeDeaths >= 3 && tradeDeaths > tradeKills * 1.5) {
    candidates.push({
      type: INSIGHT_TYPES.WARNING,
      priority: 1,
      text: `Te tradean al morir en ${name}. Juega más cerca de un compañero o usa utility antes.`,
      category: 'positioning'
    });
  }
  
  // === ENTRY/OPENING PERFORMANCE ===
  const openingAttempts = context_stats.opening_attempts || 0;
  const openingKills = context_stats.opening_kills || 0;
  
  if (openingAttempts >= 4) {
    const openingWR = (openingKills / openingAttempts) * 100;
    
    if (openingWR >= 65) {
      candidates.push({
        type: INSIGHT_TYPES.STRENGTH,
        priority: 3,
        text: `🎯 Ganas ${Math.round(openingWR)}% de primeros duelos en ${name}. Eres un buen entry aquí.`,
        category: 'entry'
      });
    } else if (openingWR <= 30) {
      candidates.push({
        type: INSIGHT_TYPES.TACTICAL,
        priority: 2,
        text: `Tus aperturas en ${name} fallan (${Math.round(openingWR)}%). Deja que un compañero entre o usa flash.`,
        category: 'entry'
      });
    }
  }
  
  // === CT VS T SIDE PERFORMANCE ===
  const ctTotal = (ct_t_split.ct_kills || 0) + (ct_t_split.ct_deaths || 0);
  const tTotal = (ct_t_split.t_kills || 0) + (ct_t_split.t_deaths || 0);
  
  if (ctTotal >= 4 && tTotal >= 4) {
    const ctWR = (ct_t_split.ct_kills || 0) / ctTotal * 100;
    const tWR = (ct_t_split.t_kills || 0) / tTotal * 100;
    
    if (ctWR > tWR + 30) {
      candidates.push({
        type: INSIGHT_TYPES.TACTICAL,
        priority: 2,
        text: `Dominas ${name} como CT (${Math.round(ctWR)}%). Ancla esta posición en defensas.`,
        category: 'side'
      });
    } else if (tWR > ctWR + 30) {
      candidates.push({
        type: INSIGHT_TYPES.TACTICAL,
        priority: 2,
        text: `Mejor rendimiento atacando ${name} (${Math.round(tWR)}%). Lidera las entradas por aquí.`,
        category: 'side'
      });
    }
  }
  
  // === WEAPON PERFORMANCE INSIGHTS ===
  if (weapon_stats.length >= 2) {
    // Find best performing weapon
    const weaponsWithData = weapon_stats.filter(w => (w.kills + w.deaths) >= 2);
    const sorted = [...weaponsWithData].sort((a, b) => {
      const aKD = a.deaths > 0 ? a.kills / a.deaths : a.kills;
      const bKD = b.deaths > 0 ? b.kills / b.deaths : b.kills;
      return bKD - aKD;
    });
    
    if (sorted.length >= 2) {
      const best = sorted[0];
      const worst = sorted[sorted.length - 1];
      const bestKD = best.deaths > 0 ? (best.kills / best.deaths).toFixed(1) : best.kills;
      const worstKD = worst.deaths > 0 ? (worst.kills / worst.deaths).toFixed(1) : worst.kills;
      
      if (parseFloat(bestKD) >= 1.5 && parseFloat(worstKD) <= 0.7) {
        candidates.push({
          type: INSIGHT_TYPES.WEAPON,
          priority: 3,
          text: `El ${best.weapon} (${bestKD} K/D) te funciona aquí. Evita ${worst.weapon} (${worstKD} K/D) en ${name}.`,
          category: 'weapon'
        });
      } else if (parseFloat(bestKD) >= 1.5) {
        candidates.push({
          type: INSIGHT_TYPES.WEAPON,
          priority: 4,
          text: `Usa ${best.weapon} en ${name}: ${best.kills} kills con ${bestKD} K/D.`,
          category: 'weapon'
        });
      }
    }
    
    // Pistol vs Rifle analysis
    const pistols = weapon_stats.filter(w => 
      ['USP-S', 'Glock-18', 'P250', 'Five-SeveN', 'Tec-9', 'Desert Eagle', 'CZ75-Auto'].includes(w.weapon)
    );
    const rifles = weapon_stats.filter(w =>
      ['AK-47', 'M4A4', 'M4A1', 'AWP', 'AUG', 'SG 553'].includes(w.weapon)
    );
    
    if (pistols.length > 0 && rifles.length > 0) {
      const pistolKills = pistols.reduce((sum, w) => sum + w.kills, 0);
      const rifleKills = rifles.reduce((sum, w) => sum + w.kills, 0);
      
      if (pistolKills > rifleKills * 1.3 && pistolKills >= 3) {
        candidates.push({
          type: INSIGHT_TYPES.TACTICAL,
          priority: 4,
          text: `Muchos kills con pistolas en ${name}. Los duelos son cercanos, SMGs podrían funcionar.`,
          category: 'weapon'
        });
      }
    }
  }
  
  // === DISTANCE-BASED INSIGHTS ===
  if (avg_distance) {
    if (avg_distance > 1200) {
      candidates.push({
        type: INSIGHT_TYPES.POSITIONING,
        priority: 5,
        text: `Duelos a larga distancia en ${name} (${Math.round(avg_distance)}u). AWP o rifle con scope recomendado.`,
        category: 'distance'
      });
    } else if (avg_distance < 350) {
      candidates.push({
        type: INSIGHT_TYPES.POSITIONING,
        priority: 5,
        text: `Combate cercano en ${name} (${Math.round(avg_distance)}u). SMG o escopeta viable.`,
        category: 'distance'
      });
    }
  }
  
  // === TIMING INSIGHTS ===
  if (avg_time_to_damage && avg_time_to_damage > 0) {
    if (avg_time_to_damage > 900) {
      candidates.push({
        type: INSIGHT_TYPES.TIMING,
        priority: 5,
        text: `Tardas ${Math.round(avg_time_to_damage)}ms en hacer daño en ${name}. Pre-aim común posiciones.`,
        category: 'timing'
      });
    } else if (avg_time_to_damage < 400) {
      candidates.push({
        type: INSIGHT_TYPES.STRENGTH,
        priority: 6,
        text: `Reflejos rápidos en ${name}: ${Math.round(avg_time_to_damage)}ms para primer daño.`,
        category: 'timing'
      });
    }
  }
  
  // === SMOKE/WALLBANG PERFORMANCE ===
  const smokeKills = context_stats.smoke_kills || 0;
  const wallbangKills = context_stats.wallbang_kills || 0;
  
  if (smokeKills >= 2) {
    candidates.push({
      type: INSIGHT_TYPES.UTILITY,
      priority: 6,
      text: `${smokeKills} kills a través de humo en ${name}. Tu timing de spam es bueno aquí.`,
      category: 'utility'
    });
  }
  
  if (wallbangKills >= 2) {
    candidates.push({
      type: INSIGHT_TYPES.STRENGTH,
      priority: 6,
      text: `${wallbangKills} wallbangs en ${name}. Conoces bien los spams de esta zona.`,
      category: 'positioning'
    });
  }
  
  // === OVERALL STRENGTH/WEAKNESS ===
  if (win_rate >= 65 && kd >= 1.8 && sample_size >= 6) {
    candidates.push({
      type: INSIGHT_TYPES.STRENGTH,
      priority: 7,
      text: `🏆 ${name} es tu zona estrella: ${win_rate}% victorias, ${kd} K/D.`,
      category: 'strength'
    });
  } else if (win_rate >= 55 && kd >= 1.3 && sample_size >= 5) {
    candidates.push({
      type: INSIGHT_TYPES.STRENGTH,
      priority: 8,
      text: `Buen rendimiento en ${name}. Mantén tu estilo actual aquí.`,
      category: 'strength'
    });
  }
  
  // Trade positive
  if (tradeKills >= 3 && tradeKills > tradeDeaths) {
    candidates.push({
      type: INSIGHT_TYPES.STRENGTH,
      priority: 7,
      text: `${tradeKills} trades en ${name}. Cubres bien a tu equipo aquí.`,
      category: 'teamplay'
    });
  }
  
  // Sort by priority and return top 2
  candidates.sort((a, b) => a.priority - b.priority);
  
  return candidates.slice(0, 2).map(c => ({
    type: c.type,
    text: c.text,
    category: c.category
  }));
};

/**
 * Generate global insights from all callout data
 */
export const generateGlobalInsights = (allCalloutStats) => {
  const insights = [];
  const callouts = Object.entries(allCalloutStats);
  
  if (callouts.length === 0) return insights;
  
  const significantCallouts = callouts.filter(([, s]) => (s.kills + s.deaths) >= 4);
  
  if (significantCallouts.length < 2) return insights;
  
  // === BEST/WORST ZONES ===
  const sortedByWR = [...significantCallouts].sort((a, b) => b[1].win_rate - a[1].win_rate);
  
  if (sortedByWR.length >= 3) {
    const best = sortedByWR[0];
    const worst = sortedByWR[sortedByWR.length - 1];
    
    if (best[1].win_rate - worst[1].win_rate > 25) {
      insights.push({
        type: INSIGHT_TYPES.TACTICAL,
        text: `📊 Fortaleza: ${best[0]} (${best[1].win_rate}%). Debilidad: ${worst[0]} (${worst[1].win_rate}%).`,
        global: true,
        category: 'overview'
      });
    }
  }
  
  // === WEAPON GLOBAL PERFORMANCE ===
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
      kills: stats.kills,
      total: stats.kills + stats.deaths
    }))
    .filter(w => w.total >= 5)
    .sort((a, b) => b.kd - a.kd);
  
  if (weaponList.length >= 2) {
    const best = weaponList[0];
    
    if (best.kd >= 1.5) {
      insights.push({
        type: INSIGHT_TYPES.WEAPON,
        text: `🔫 Tu mejor arma: ${best.weapon} con ${best.kd.toFixed(1)} K/D (${best.kills} kills).`,
        global: true,
        category: 'weapon'
      });
    }
  }
  
  // === SIDE ANALYSIS ===
  let totalCTKills = 0, totalCTDeaths = 0, totalTKills = 0, totalTDeaths = 0;
  
  for (const [, stats] of significantCallouts) {
    const split = stats.ct_t_split || {};
    totalCTKills += split.ct_kills || 0;
    totalCTDeaths += split.ct_deaths || 0;
    totalTKills += split.t_kills || 0;
    totalTDeaths += split.t_deaths || 0;
  }
  
  if (totalCTKills + totalCTDeaths >= 10 && totalTKills + totalTDeaths >= 10) {
    const ctKD = totalCTDeaths > 0 ? totalCTKills / totalCTDeaths : totalCTKills;
    const tKD = totalTDeaths > 0 ? totalTKills / totalTDeaths : totalTKills;
    
    if (ctKD > tKD * 1.4) {
      insights.push({
        type: INSIGHT_TYPES.TACTICAL,
        text: `🛡️ Mejor defensivamente: ${ctKD.toFixed(1)} K/D como CT vs ${tKD.toFixed(1)} como T.`,
        global: true,
        category: 'side'
      });
    } else if (tKD > ctKD * 1.4) {
      insights.push({
        type: INSIGHT_TYPES.TACTICAL,
        text: `⚔️ Mejor ofensivamente: ${tKD.toFixed(1)} K/D como T vs ${ctKD.toFixed(1)} como CT.`,
        global: true,
        category: 'side'
      });
    }
  }
  
  // === OPENING DUELS GLOBAL ===
  let totalOpeningKills = 0, totalOpeningAttempts = 0;
  
  for (const [, stats] of significantCallouts) {
    const ctx = stats.context_stats || {};
    totalOpeningKills += ctx.opening_kills || 0;
    totalOpeningAttempts += ctx.opening_attempts || 0;
  }
  
  if (totalOpeningAttempts >= 8) {
    const openingWR = (totalOpeningKills / totalOpeningAttempts) * 100;
    
    if (openingWR >= 55) {
      insights.push({
        type: INSIGHT_TYPES.STRENGTH,
        text: `🎯 ${Math.round(openingWR)}% en primeros duelos. Eres un buen entry fragger.`,
        global: true,
        category: 'entry'
      });
    } else if (openingWR <= 35) {
      insights.push({
        type: INSIGHT_TYPES.WARNING,
        text: `⚠️ Solo ${Math.round(openingWR)}% en primeros duelos. Considera un rol de soporte.`,
        global: true,
        category: 'entry'
      });
    }
  }
  
  // === CONSISTENCY INSIGHT ===
  const winRates = significantCallouts.map(([, s]) => s.win_rate);
  if (winRates.length >= 4) {
    const avg = winRates.reduce((a, b) => a + b, 0) / winRates.length;
    const variance = winRates.reduce((sum, wr) => sum + Math.pow(wr - avg, 2), 0) / winRates.length;
    const stdDev = Math.sqrt(variance);
    
    if (stdDev <= 10 && avg >= 50) {
      insights.push({
        type: INSIGHT_TYPES.STRENGTH,
        text: `📈 Rendimiento consistente: ${Math.round(avg)}% WR promedio con poca variación.`,
        global: true,
        category: 'consistency'
      });
    } else if (stdDev > 20) {
      insights.push({
        type: INSIGHT_TYPES.INFO,
        text: `📊 Rendimiento variable entre zonas. Enfócate en tus puntos débiles.`,
        global: true,
        category: 'consistency'
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
export const getTopInsights = (allCalloutStats, limit = 6) => {
  const all = getAllInsights(allCalloutStats);
  
  // Deduplicate by category to get variety
  const seen = new Set();
  const unique = [];
  
  for (const insight of all) {
    const cat = insight.category || 'general';
    if (!seen.has(cat) || unique.length < 3) {
      unique.push(insight);
      seen.add(cat);
      if (unique.length >= limit) break;
    }
  }
  
  return unique;
};

export default {
  INSIGHT_TYPES,
  generateCalloutInsights,
  generateGlobalInsights,
  getAllInsights,
  getTopInsights
};
