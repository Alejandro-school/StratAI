import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { createCoachMessage } from '../utils/coachMessageTypes';
import { mockAiCoachResponse } from '../mocks/aiCoachResponses';

/* ── Helper: next unique ID ── */
const nextId = () => Date.now() + Math.random();

/* ══════════════════════════════════════
   Proactive greeting — sent when matches finish loading.
   The AI takes initiative, analyses patterns, and suggests action.
══════════════════════════════════════ */

const buildPatternGreeting = (username, matches) => {
  const msgs = [];
  const name = username || 'Gamer';

  const mapStats = {};
  let totalWins = 0;
  let totalLosses = 0;

  matches.forEach((m) => {
    const isWin = m.result === 'W' || m.result === 'victory';
    const mapName = (m.map_name || m.map || 'desconocido').replace('de_', '');
    if (!mapStats[mapName]) mapStats[mapName] = { wins: 0, losses: 0 };
    if (isWin) { totalWins++; mapStats[mapName].wins++; }
    else { totalLosses++; mapStats[mapName].losses++; }
  });

  let weakestMap = null;
  let worstRatio = Infinity;
  Object.entries(mapStats).forEach(([map, stats]) => {
    const total = stats.wins + stats.losses;
    if (total >= 2) {
      const ratio = stats.wins / total;
      if (ratio < worstRatio) {
        worstRatio = ratio;
        weakestMap = { name: map, ...stats, total };
      }
    }
  });

  const recentMatches = matches.slice(0, 5);
  const recentLosses = recentMatches.filter(
    (m) => m.result !== 'W' && m.result !== 'victory'
  ).length;

  const winRate = matches.length > 0
    ? Math.round((totalWins / matches.length) * 100)
    : 0;

  msgs.push({
    id: nextId(),
    sender: 'ai',
    context: 'ANÁLISIS GLOBAL',
    text: `Hola ${name}. He analizado tus ${matches.length} partidas. Tienes un win rate del ${winRate}% (${totalWins}V - ${totalLosses}D). Vamos a ver qué podemos mejorar.`
  });

  const insights = [];

  if (weakestMap && worstRatio < 0.5) {
    const mapWinRate = Math.round((weakestMap.wins / weakestMap.total) * 100);
    insights.push(
      `Tu mapa más débil es ${weakestMap.name.toUpperCase()} con un ${mapWinRate}% de win rate (${weakestMap.wins}V-${weakestMap.losses}D en ${weakestMap.total} partidas). Deberíamos analizar qué está pasando ahí.`
    );
  }

  if (recentLosses >= 3) {
    insights.push(
      `He detectado una racha negativa reciente: ${recentLosses} derrotas en las últimas 5 partidas. Es importante identificar patrones comunes en esas partidas.`
    );
  } else if (recentLosses === 0 && recentMatches.length >= 3) {
    insights.push(
      `Buena racha: ¡${recentMatches.length} victorias consecutivas! Vamos a analizar qué estás haciendo bien para consolidar esos hábitos.`
    );
  }

  if (insights.length === 0) {
    insights.push(
      'No he encontrado patrones preocupantes obvios, pero siempre hay margen de mejora. Te recomiendo seleccionar una partida que recuerdes como complicada para analizarla en detalle.'
    );
  }

  msgs.push({
    id: nextId(),
    sender: 'ai',
    context: 'PATRONES DETECTADOS',
    text: insights.join(' ')
  });

  const ctaText = weakestMap && worstRatio < 0.5
    ? `Te recomiendo empezar analizando una de tus partidas en ${weakestMap.name.toUpperCase()} — selecciónala en el panel y pulsa "Analizar partida seleccionada". Si prefieres otra, tú eliges.`
    : 'Selecciona una partida en el panel y pulsa "Analizar partida seleccionada" para que te haga un breakdown completo con focos de mejora concretos.';

  msgs.push({
    id: nextId(),
    sender: 'ai',
    context: 'SIGUIENTE PASO',
    text: ctaText
  });

  return msgs;
};

/* ══════════════════════════════════════
   Auto-analysis — when user clicks "Analizar partida seleccionada"
══════════════════════════════════════ */

const buildMatchAnalysis = (match) => {
  const mapName = (match.map_name || match.map || 'mapa').replace('de_', '');
  const score = `${match.team_score ?? '?'}-${match.opponent_score ?? '?'}`;
  const isWin = match.result === 'W' || match.result === 'victory';
  const totalRounds = Number(match.total_rounds) || 0;

  const closeness = Math.abs((match.team_score || 0) - (match.opponent_score || 0));
  const wasClose = closeness <= 3;

  let analysis = `Partida en ${mapName.toUpperCase()} — resultado: ${score} (${isWin ? 'Victoria' : 'Derrota'}).`;

  if (totalRounds > 0) {
    analysis += ` ${totalRounds} rondas totales.`;
  }

  analysis += '\n\n';

  if (isWin && wasClose) {
    analysis += 'Observaciones:\n\n';
    analysis += '• Victoria ajustada. Hubo rondas clave donde la economía pudo haber sido mejor gestionada.\n';
    analysis += '• Revisa las rondas de eco — posiblemente puedes optimizar las decisiones de force buy.\n';
    analysis += '• Buen trabajo en las rondas decisivas, pero hay margen para cerrar mapas con más autoridad.';
  } else if (isWin && !wasClose) {
    analysis += 'Observaciones:\n\n';
    analysis += '• Victoria dominante. Tus aperturas fueron sólidas y las rotaciones rápidas.\n';
    analysis += '• Aún así, identifiqué rondas perdidas innecesariamente por sobrepeek individual.\n';
    analysis += '• Consolida estos hábitos positivos y trabaja en reducir las muertes innecesarias.';
  } else if (!isWin && wasClose) {
    analysis += 'Observaciones:\n\n';
    analysis += '• Derrota ajustada. Estuviste cerca pero las rondas de pistola y los clutch definieron el partido.\n';
    analysis += '• He detectado un patrón: en las rondas 13-16 tu equipo tendió a forzar duelos desfavorables.\n';
    analysis += '• Recomiendo trabajar la disciplina en rondas de presión — mantener posición y no sobrepeekear.';
  } else {
    analysis += 'Observaciones:\n\n';
    analysis += '• Derrota clara. El equipo contrario controló la economía y las aperturas mid-round.\n';
    analysis += '• He detectado falta de utilidad coordinada en las entradas a sitio.\n';
    analysis += '• Recomiendo enfocarse en: 1) Timing de flashes de entrada, 2) Smokes de corte, 3) Disciplina económica.';
  }

  return analysis;
};

const buildFollowupResponse = (text) => {
  const lowerText = text.toLowerCase();

  if (lowerText.includes('aim') || lowerText.includes('punter')) {
    return {
      id: nextId(),
      sender: 'ai',
      context: 'PLAN DE AIM',
      text: 'Te propongo una rutina simple de 20 minutos: 8 min tracking, 8 min precisión inicial y 4 min transfer entre objetivos. Si quieres, te la adapto a tu horario.'
    };
  }

  if (lowerText.includes('utility') || lowerText.includes('granada') || lowerText.includes('utilidad')) {
    return {
      id: nextId(),
      sender: 'ai',
      context: 'UTILITY FUNDAMENTAL',
      text: 'Vamos a trabajar 2 protocolos base: flash de entrada + smoke de corte. El objetivo es repetirlos hasta que sean automáticos.'
    };
  }

  if (lowerText.includes('mental') || lowerText.includes('consistencia')) {
    return {
      id: nextId(),
      sender: 'ai',
      context: 'CONSISTENCIA',
      text: 'Te recomiendo un bloque fijo: objetivo de la sesión, 3 puntos de foco y cierre rápido post-partida. Eso reduce picos de rendimiento.'
    };
  }

  if (lowerText.includes('economía') || lowerText.includes('eco') || lowerText.includes('buy')) {
    return {
      id: nextId(),
      sender: 'ai',
      context: 'ECONOMÍA',
      text: 'La clave está en sincronizar las compras con tu equipo. Evita force buys individuales y aprende a leer los loss bonus del rival para optimizar tus ecos.'
    };
  }

  if (lowerText.includes('error') || lowerText.includes('replay') || lowerText.includes('fall') || lowerText.includes('mal')) {
    return {
      id: nextId(),
      sender: 'ai',
      context: 'ANÁLISIS VÍDEO 2D',
      text: mockAiCoachResponse.message,
      interaction: mockAiCoachResponse.interaction
    };
  }

  return {
    id: nextId(),
    sender: 'ai',
    context: 'COACHING',
    text: 'Perfecto. Si me dices objetivo y tiempo disponible, te preparo un plan de trabajo concreto para hoy.'
  };
};

/* ══════════════════════════════════════
   HOOK
══════════════════════════════════════ */

const SESSION_CHAT_KEY = 'stratai_coach_chat_history';

const useCoachChat = (user) => {
  const [chatInput, setChatInput] = useState('');
  const [isAiTyping, setIsAiTyping] = useState(false);
  const [messages, setMessages] = useState(() => {
    // Lazy initialization from session storage
    if (typeof window !== 'undefined') {
      try {
        const stored = sessionStorage.getItem(SESSION_CHAT_KEY);
        if (stored) {
          return JSON.parse(stored);
        }
      } catch (err) {
        console.error('Error reading chat history from session storage:', err);
      }
    }
    return [];
  });

  // Sync messages to session storage on change
  useEffect(() => {
    if (typeof window !== 'undefined') {
      try {
        sessionStorage.setItem(SESSION_CHAT_KEY, JSON.stringify(messages));
      } catch (err) {
        console.error('Error saving chat history to session storage:', err);
      }
    }
  }, [messages]);

  // Use ref to prevent React StrictMode double-mount from duplicating greeting
  const hasGreetedRef = useRef(false);

  const quickActions = useMemo(
    () => ['Quiero mejorar mi aim', 'Hazme un plan de utility', 'Quiero ser más consistente'],
    []
  );

  /* ── Add AI message with typewriter: render full text invisibly to hold space ── */
  const addAiMessageWithTyping = useCallback((msgData, onComplete) => {
    const fullText = msgData.text;
    const msgId = msgData.id;

    // 1. Add message with full text to hold layout space, but 0 visible chars
    const shell = createCoachMessage({
      ...msgData,
      visibleChars: 0,
      isTyping: true
    });
    setMessages((prev) => [...prev, shell]);

    // 2. Increment visibleChars instead of slicing text
    let charIdx = 0;
    const speed = 18; // ms per character

    const interval = setInterval(() => {
      charIdx++;
      if (charIdx <= fullText.length) {
        setMessages((prev) =>
          prev.map((m) =>
            m.id === msgId
              ? { ...m, visibleChars: charIdx, isTyping: true }
              : m
          )
        );
      } else {
        // Done typing
        setMessages((prev) =>
          prev.map((m) =>
            m.id === msgId ? { ...m, isTyping: false } : m
          )
        );
        clearInterval(interval);
        onComplete?.();
      }
    }, speed);

    return () => clearInterval(interval);
  }, []);

  /* ── Queue multiple AI messages sequentially with typewriter ── */
  const queueAiMessages = useCallback((msgDataArray) => {
    if (!msgDataArray.length) return;

    setIsAiTyping(true);
    let idx = 0;
    const cleanups = [];

    const typeNext = () => {
      if (idx >= msgDataArray.length) {
        setIsAiTyping(false);
        return;
      }

      const msgData = msgDataArray[idx];
      idx++;

      // Small delay between messages
      const delayMs = idx === 1 ? 400 : 800;
      const timer = setTimeout(() => {
        const cleanup = addAiMessageWithTyping(
          createCoachMessage(msgData),
          typeNext
        );
        cleanups.push(cleanup);
      }, delayMs);
      cleanups.push(() => clearTimeout(timer));
    };

    typeNext();

    // Return cleanup
    return () => cleanups.forEach((fn) => fn?.());
  }, [addAiMessageWithTyping]);

  /* ── Proactive greeting: called by Dashboard when matches load ── */
  const sendProactiveGreeting = useCallback((matches) => {
    if (hasGreetedRef.current) return;
    hasGreetedRef.current = true;

    // Do not send greeting if we already have restored messages (user navigated back to the tab)
    if (messages.length > 0) return;

    const greetingMessages = buildPatternGreeting(user?.username, matches);
    queueAiMessages(greetingMessages);
  }, [user?.username, queueAiMessages, messages.length]);

  /* ── Auto-analyze match: called by Dashboard when "Analizar" is clicked ── */
  const analyzeMatch = useCallback((match) => {
    if (!match) return;

    const mapName = (match.map_name || match.map || 'mapa').replace('de_', '');
    const score = `${match.team_score ?? '?'}-${match.opponent_score ?? '?'}`;

    // Add user message
    const triggerMsg = createCoachMessage({
      id: nextId(),
      sender: 'user',
      text: `Analiza mi partida en ${mapName.toUpperCase()} (${score}).`
    });
    setMessages((prev) => [...prev, triggerMsg]);

    // Generate analysis with typewriter
    const analysisText = buildMatchAnalysis(match);
    const analysisData = {
      id: nextId(),
      sender: 'ai',
      context: 'ANÁLISIS TÁCTICO',
      text: analysisText
    };

    queueAiMessages([analysisData]);
  }, [queueAiMessages]);

  /* ── Standard user message ── */
  const submitMessage = useCallback((value) => {
    const cleanValue = value.trim();
    if (!cleanValue) return;

    const userMessage = createCoachMessage({
      id: nextId(),
      sender: 'user',
      text: cleanValue
    });

    setMessages((prev) => [...prev, userMessage]);
    setChatInput('');

    const responseData = buildFollowupResponse(cleanValue);
    queueAiMessages([responseData]);
  }, [queueAiMessages]);

  return {
    messages,
    chatInput,
    setChatInput,
    isAiTyping,
    quickActions,
    submitMessage,
    sendProactiveGreeting,
    analyzeMatch
  };
};

export default useCoachChat;
