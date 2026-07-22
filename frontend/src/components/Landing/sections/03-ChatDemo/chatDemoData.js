export const FIRST_INSIGHT_MOMENT = 1.3;
export const CRITICAL_MOMENT = 6;

export const INSIGHT_DELAY_FIRST = 6000;
export const INSIGHT_DELAY_CRITICAL = 8000;
export const RESUME_DELAY = 3000;

export const CHAT_DEMO_SCRIPT = {
  es: {
    phaseLabels: {
      firstInsights: 'Análisis inicial...',
      criticalInsights: 'Error detectado',
      videoEnded: 'Ronda perdida',
      roundLost: 'RONDA PERDIDA',
      paused: 'Pausado: clic para continuar',
      analyzing: 'Analizando...',
      playing: 'Reproduciendo...',
      roundFinished: 'Ronda finalizada',
      coach: 'Coach IA',
      statusAnalyzing: 'Analizando',
      statusError: 'Error detectado',
      statusSuccess: 'Análisis listo',
      emptyStart: 'Reproduce el video para ver\nel análisis en vivo',
      emptyAnalyzing: 'Analizando situación táctica...',
      questionsPrompt: 'Hazme una pregunta:',
      you: 'TÚ',
      teammate: 'COMPAÑERO',
    },
    firstInsights: [
      {
        id: 'corta-peek',
        player: 'corta',
        type: 'positive',
        text: 'Tu peek por corta para buscar información fue una buena decisión. Tienes cobertura desde oscuro bajo, lo que limita los ángulos de exposición. Aun así, habría sido más seguro con una popflash previa para minimizar el riesgo del duelo.',
      },
      {
        id: 'b-advance',
        player: 'b',
        type: 'warning',
        text: 'Atención: tu compañero en túnel está avanzando sin necesidad. Ya hay un compañero en punto de B vigilando túnel, así que se duplica la información y Mid queda sin cobertura. Una rotación a Mid sería más efectiva tácticamente.',
      },
    ],
    criticalInsights: [
      {
        id: 'corta-error',
        player: 'corta',
        type: 'error',
        text: 'Error grave: en un 4v4 defendiendo, no necesitas tomar la iniciativa. Tu compañero de oscuro bajo ha muerto, ya no tienes cobertura y quedas expuesto a varios ángulos. Es una situación de 3v1 con muy poca probabilidad de sobrevivir o generar impacto.',
      },
      {
        id: 'b-error',
        player: 'b-critical',
        type: 'warning',
        text: 'Tu compañero en túnel ha avanzado solo de forma agresiva. En lugar de mantener la posición defensiva con su compañero en B o rotar hacia Mid, crea un riesgo innecesario que rompe la estructura del equipo.',
      },
    ],
    finalSummary: {
      id: 'final-summary',
      type: 'consequence',
      text: 'Resultado: una situación equilibrada de 4v4 se convierte en una desventaja crítica de 4v2. Los errores de posicionamiento y toma de decisiones provocan la pérdida de la ronda. Veamos qué se podría haber hecho diferente.',
    },
    userQuestions: [
      {
        id: 'what-should',
        question: '¿Qué debería haber hecho?',
        response: 'Deberías haber retrocedido, buscando apoyo de tu compañero en A Site. Si los rivales deciden pushear por corta, podéis frenarlos con utilidad y ganar tiempo para la rotación de los jugadores de B.',
      },
      {
        id: 'what-focus',
        question: '¿En qué debo fijarme la próxima vez?',
        response: 'Fíjate en el minimapa. En estas situaciones da información clave para elegir tu próxima acción. Si hubieras visto que tu compañero de oscuro había muerto, sabrías que ese peek te dejaba expuesto desde dos posiciones.',
      },
      {
        id: 'team-coord',
        question: '¿Cómo mejorar la coordinación del equipo?',
        response: 'La comunicación tiene que ser fluida. Usar micro y tener un rol claro, como un IGL que coordine al equipo, ayuda mucho a evitar estos errores. Incluso con desconocidos, usar el micro suele mejorar la estructura de la ronda.',
      },
    ],
  },
  en: {
    phaseLabels: {
      firstInsights: 'Initial analysis...',
      criticalInsights: 'Mistake detected',
      videoEnded: 'Round lost',
      roundLost: 'ROUND LOST',
      paused: 'Paused: click to continue',
      analyzing: 'Analyzing...',
      playing: 'Playing...',
      roundFinished: 'Round finished',
      coach: 'AI Coach',
      statusAnalyzing: 'Analyzing',
      statusError: 'Mistake detected',
      statusSuccess: 'Analysis ready',
      emptyStart: 'Play the video to see\nthe live analysis',
      emptyAnalyzing: 'Analyzing tactical situation...',
      questionsPrompt: 'Ask me a question:',
      you: 'YOU',
      teammate: 'TEAMMATE',
    },
    firstInsights: [
      {
        id: 'corta-peek',
        player: 'corta',
        type: 'positive',
        text: 'Your short peek to gather information was a good decision. You still had cover from lower dark, which limited your exposed angles. Even so, a pop flash first would have reduced the duel risk.',
      },
      {
        id: 'b-advance',
        player: 'b',
        type: 'warning',
        text: 'Warning: your teammate in tunnel is pushing without a clear need. Another teammate is already watching tunnel from B site, so the information is duplicated and Mid is left uncovered. A rotation to Mid would be tactically stronger.',
      },
    ],
    criticalInsights: [
      {
        id: 'corta-error',
        player: 'corta',
        type: 'error',
        text: 'Serious mistake: in a defensive 4v4, you do not need to take the initiative. Your lower dark teammate has died, you no longer have cover, and you are exposed to multiple angles. This is a 3v1 situation with very low odds of surviving or creating impact.',
      },
      {
        id: 'b-error',
        player: 'b-critical',
        type: 'warning',
        text: 'Your teammate in tunnel pushed alone and too aggressively. Instead of holding a defensive position with the B player or rotating toward Mid, this creates unnecessary risk and breaks the team structure.',
      },
    ],
    finalSummary: {
      id: 'final-summary',
      type: 'consequence',
      text: 'Result: an even 4v4 becomes a critical 4v2 disadvantage. Positioning and decision-making mistakes cause the round loss. Let us review what could have been done differently.',
    },
    userQuestions: [
      {
        id: 'what-should',
        question: 'What should I have done?',
        response: 'You should have fallen back and looked for support from your A site teammate. If the opponents decide to push short, you can slow them with utility and buy time for the B players to rotate.',
      },
      {
        id: 'what-focus',
        question: 'What should I watch next time?',
        response: 'Watch the minimap. In these situations it gives you the key information for your next action. If you had noticed that your lower dark teammate died, you would know that peek exposed you from two positions.',
      },
      {
        id: 'team-coord',
        question: 'How can we improve team coordination?',
        response: 'Communication has to be fluid. Using voice and having a clear role, such as an IGL coordinating the team, helps prevent these mistakes. Even with random teammates, using voice usually improves the round structure.',
      },
    ],
  },
};
