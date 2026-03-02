/**
 * Datos simulados (Mocks) para la respuesta de la IA del Coach.
 * Esto define el "Contrato de Datos" (Data Contract) entre el Backend (IA) y el Frontend.
 */

export const mockAiCoachResponse = {
  id: "err_001",
  message: "He detectado una posición vulnerable en la ronda 6. Te abriste demasiado en Medio sin esperar el humo de tu compañero. Fíjate en el replay cómo quedaste expuesto al AWP enemigo.",
  interaction: {
    action: "PLAY_CLIP",
    // Usamos valores normalizados (0.0 a 1.0) para currentTime en vez de ticks absolutos
    // para asegurar que siempre haya un frame válido.
    startTick: 0.35,         // 35% de la ronda
    criticalTick: 0.45,      // 45% de la ronda (momento del error)
    endTick: 0.55,           // 55% de la ronda (fin del clip)
    focusPlayerId: 3,        // ID del jugador principal a enfocar en el canvas
    annotations: [
      { 
        type: "DANGER_ZONE", 
        x: -450, 
        y: 800, 
        radius: 300, 
        color: "rgba(255, 0, 0, 0.4)",
        description: "Línea de visión del AWP"
      },
      { 
        type: "SUGGESTED_PATH", 
        points: [
          { x: -200, y: 500 }, 
          { x: -300, y: 650 }, 
          { x: -500, y: 700 }
        ],
        color: "rgba(0, 255, 0, 0.8)",
        description: "Ruta de rotación segura"
      }
    ]
  }
};

/**
 * Función que simula la petición al backend de la IA.
 * Introduce un retardo artificial para probar los estados de carga en la UI.
 */
export const fetchMockAiResponse = () => {
  return new Promise((resolve) => {
    setTimeout(() => {
      resolve(mockAiCoachResponse);
    }, 1500); // Simula 1.5 segundos de procesamiento de la IA
  });
};
