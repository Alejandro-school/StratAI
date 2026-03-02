/**
 * Cumulative AI Observation Notes
 * 
 * After each challenge result, the AI "records" an observation that
 * references ALL prior results. This creates the feeling of a profile
 * being built progressively across challenges.
 * 
 * Flow: Economy (1st) → Grenade (2nd) → GameSense (3rd)
 */

const ECONOMY_NOTES = {
  correct: {
    label: 'Gestión Económica',
    note: 'Lectura económica clara. Tu gestión de recursos no es improvisada — eso reduce el rango de errores que la IA busca en tus demos reales.',
  },
  incorrect: {
    label: 'Gestión Económica',
    note: 'Patrón de compra impulsivo detectado. En tus demos, esto suele traducirse en rondas de eco innecesarias y compras a destiempo.',
  },
};

const GRENADE_NOTES = {
  // eco correct + grenade correct
  true_true: {
    label: 'Economía + Utilidad',
    note: 'Dos lecturas consistentes. Perfil emergente: jugador metódico que entiende el macro-game. La siguiente evaluación mide tu capacidad de lectura en tiempo real.',
  },
  // eco correct + grenade incorrect
  true_false: {
    label: 'Economía + Utilidad',
    note: 'Buenas decisiones de compra, pero no las complementas con utilidad efectiva. En tus demos, probablemente dependes de duelos directos que podrías evitar.',
  },
  // eco incorrect + grenade correct
  false_true: {
    label: 'Economía + Utilidad',
    note: 'Conocimiento de lineups sin base económica sólida. Contraste interesante — la IA cruzaría esto con tu timing de compra en partidas reales.',
  },
  // eco incorrect + grenade incorrect
  false_false: {
    label: 'Economía + Utilidad',
    note: 'Dos áreas fundamentales con margen. No es un diagnóstico completo — la IA necesita tus demos reales para separar conocimiento de ejecución bajo presión.',
  },
  // eco skipped
  skipped_true: {
    label: 'Utilidad',
    note: 'Buen conocimiento de lineups. La IA registra este dato para cruzarlo con tus patrones de uso de utilidad en demos reales.',
  },
  skipped_false: {
    label: 'Utilidad',
    note: 'Lineup incorrecto. La IA comparará esto con tu uso de utilidad real — a veces el conocimiento teórico no refleja la ejecución bajo presión.',
  },
};

const GAMESENSE_NOTES = {
  // 3/3 correct
  3: {
    label: 'Perfil Preliminar',
    note: 'Tres fundamentos sólidos. Con tus demos, la IA analiza lo que estas pruebas no pueden: timings, tendencias posicionales, hábitos bajo presión y patrones que ni tú ves.',
  },
  // 2/3 correct
  2: {
    label: 'Perfil Preliminar',
    note: 'Base sólida con un punto ciego claro. Esto es exactamente lo que la IA prioriza con demos reales — las áreas donde sabes qué hacer pero no lo ejecutas.',
  },
  // 1/3 correct
  1: {
    label: 'Perfil Preliminar',
    note: 'Un fundamento fuerte, dos áreas de desarrollo. Esto le da a la IA un roadmap claro de qué analizar primero en tus partidas.',
  },
  // 0/3 correct
  0: {
    label: 'Perfil Preliminar',
    note: 'Las tres áreas muestran margen de mejora. Esto no es negativo — le da a la IA un panorama completo para construir tu plan personalizado.',
  },
};

/**
 * Get the cumulative observation note for a challenge result.
 * 
 * @param {'economy'|'grenade'|'gamesense'} challengeId
 * @param {boolean} isCorrect - Whether the current challenge was answered correctly
 * @param {Object} completedChallenges - Full state from LandingContext
 * @returns {{ label: string, note: string }}
 */
export function getObservationNote(challengeId, isCorrect, completedChallenges) {
  switch (challengeId) {
    case 'economy':
      return isCorrect ? ECONOMY_NOTES.correct : ECONOMY_NOTES.incorrect;

    case 'grenade': {
      const ecoCompleted = completedChallenges.economy?.completed;
      const ecoSuccess = completedChallenges.economy?.success;
      
      if (!ecoCompleted) {
        // Economy was skipped
        const key = `skipped_${isCorrect}`;
        return GRENADE_NOTES[key];
      }
      const key = `${ecoSuccess}_${isCorrect}`;
      return GRENADE_NOTES[key];
    }

    case 'gamesense': {
      // Count total successes including current result
      let correctCount = 0;
      if (completedChallenges.economy?.success) correctCount++;
      if (completedChallenges.grenade?.success) correctCount++;
      if (isCorrect) correctCount++;
      
      return GAMESENSE_NOTES[correctCount];
    }

    default:
      return { label: 'Análisis', note: '' };
  }
}
