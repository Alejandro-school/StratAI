// frontend/src/utils/mapLevelFallbacks.js
// Fallback para asignar callouts a niveles cuando no hay datos de Z

/**
 * Manual mapping of callouts to LOWER level for multi-level maps
 * Anything NOT in this list defaults to UPPER
 * ONLY include callouts we're 100% certain are lower level
 */
export const LOWER_LEVEL_CALLOUTS = {
  de_nuke: [
    // B Site area (definitely lower)
    'b site', 'b-site', 'bsite', 'b plant',
    
    // Vents and secret (lower level)
    'vents', 'vent', 'secret',
    
    // Decon area (lower)
    'decon', 'decontamination',
    
    // Tunnels (lower)
    'tunnels', 'tunnel'
    
    // NOTE: Rafters, Lockers, Catwalk are UPPER - not in this list
  ],
  
  de_vertigo: [
    'b site', 'b-site', 'bsite',
    'ladder', 'underground', 'stairs',
    'sandbags', 'ramp room'
  ],
  
  de_train: [
    'lower b', 'lower a', 
    'showers', 'connector', 'tunnels',
    'old bomb', 'old bombsite'
  ]
};

/**
 * Determine which level a callout belongs to based on its name
 * Returns 'upper' or 'lower'
 * 
 * LOGIC: If callout name contains ANY pattern from lower list -> lower
 *        Otherwise -> upper (safer default)
 */
export const guessCalloutLevel = (calloutName, mapName) => {
  const lowerPatterns = LOWER_LEVEL_CALLOUTS[mapName];
  if (!lowerPatterns) return 'upper'; // Unknown map -> default upper
  
  const nameLower = calloutName.toLowerCase().trim();
  
  // Check if name contains any lower-level pattern
  const isLower = lowerPatterns.some(pattern => nameLower.includes(pattern));
  
  return isLower ? 'lower' : 'upper';
};

