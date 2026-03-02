/**
 * Data for the ChatDemoSection
 */

// Pause moments
export const FIRST_INSIGHT_MOMENT = 1.3;  // First pause for insights
export const CRITICAL_MOMENT = 6;         // Critical error pause

// Time between insights (ms)
export const INSIGHT_DELAY_FIRST = 6000;    // 5 seconds between first insights
export const INSIGHT_DELAY_CRITICAL = 8000; // 8 seconds between critical insights
export const RESUME_DELAY = 3000;            // 3 seconds before auto-resume

// Scripted AI insights at first pause (1.3s)
export const FIRST_INSIGHTS = [
  {
    id: 'corta-peek',
    player: 'corta',
    type: 'positive',
    text: 'Tu peek por Corta para buscar información fue una buena decisión. Tienes cobertura desde Oscuro Bajo, lo que limita los ángulos de exposición. Sin embargo, hubiera sido más seguro con una popflash previa para minimizar el riesgo del duelo.'
  },
  {
    id: 'b-advance',
    player: 'b',
    type: 'warning',
    text: 'Atención: Tu compañero en Túnel está avanzando sin necesidad, ya hay un compañero en punto de B vigilando túnel. Esto genera duplicidad de información y deja Mid sin cobertura. Una rotación a Mid sería más efectiva tácticamente.'
  }
];

// Scripted AI insights at critical moment (6s)
export const CRITICAL_INSIGHTS = [
  {
    id: 'corta-error',
    player: 'corta',
    type: 'error',
    text: 'Error grave: En un 4v4 defendiendo, no hay necesidad de tomar la iniciativa. Tu compañero de Oscuro Bajo ha muerto, ya no tienes cobertura y quedas expuesto a varios ángulos. Es una situación de 3v1 en la que tienes muy poca probabilidad de sobrevivir y de generar impacto.'
  },
  {
    id: 'b-error',
    player: 'b-critical',
    type: 'warning',
    text: 'Tu compañero en Túnel ha avanzado solo de forma agresiva. En lugar de mantener la posición defensiva con su compañero en B o rotar hacia Mid, está creando una situación de riesgo innecesario que compromete la estructura del equipo y provoca desequilibrio.'
  }
];

// Final summary after video ends
export const FINAL_SUMMARY = {
  id: 'final-summary',
  type: 'consequence',
  text: 'Resultado: Lo que era una situación equilibrada (4v4) se ha convertido en una desventaja crítica (4v2). Los errores de posicionamiento y la toma de decisiones han provocado la pérdida de la ronda. Analicemos qué se podría haber hecho diferente.'
};

// User questions after video ends
export const USER_QUESTIONS = [
  {
    id: 'what-should',
    question: '¿Qué debería haber hecho?',
    response: 'Deberías haber retrocedido hacia atrás, buscando apoyo de tu compañero en A Site. Si los enemigos deciden pushear por Corta, podéis frenarlos con utilidad y ganar tiempo para una rotación de los jugadores de B.'
  },
  {
    id: 'what-focus',
    question: '¿En qué debo fijarme la próxima vez?',
    response: 'Fíjate en el minimapa, en estas situaciones otorga mucha información que te ayudará a elegir tu próxima acción. Si hubieras visto que tu compañero de Oscuro ha muerto en el minimapa, no hubieras tomado la decisión de peekear porque sabes que quedas expuesto a que te disparen desde 2 sitios diferentes.'
  },
  {
    id: 'team-coord',
    question: '¿Cómo mejorar la coordinación del equipo?',
    response: 'Es importante que la comunicación sea fluida. El uso del micro y un rol marcado, un "IGL" que coordine al equipo, es clave para evitar este tipo de errores. En caso de jugar con desconocidos, te animo a usar el micro igualmente. Cuando llevéis 2-3 rondas ya se verá qué jugador tiene la mejor toma de decisiones y podéis apoyaros en él.'
  }
];
