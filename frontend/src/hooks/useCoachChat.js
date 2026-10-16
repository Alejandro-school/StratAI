import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { createCoachMessage } from '../utils/coachMessageTypes';
import { mockAiCoachResponse } from '../mocks/aiCoachResponses';
import { readCoachHistory, saveCoachHistory } from '../auth/browserPrivacy';

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
    text: `Hola ${name}. He analizado tus ${matches.length} partidas. Tienes un porcentaje de victorias del ${winRate}% (${totalWins}V - ${totalLosses}D). Vamos a ver qué podemos mejorar.`
  });

  const insights = [];

  if (weakestMap && worstRatio < 0.5) {
    const mapWinRate = Math.round((weakestMap.wins / weakestMap.total) * 100);
    insights.push(
      `Tu mapa más débil es ${weakestMap.name.toUpperCase()} con un ${mapWinRate}% de victorias (${weakestMap.wins}V-${weakestMap.losses}D en ${weakestMap.total} partidas). Deberíamos analizar qué está pasando ahí.`
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
    analysis += '• Revisa las rondas de eco — posiblemente puedes optimizar las decisiones de compra forzada.\n';
    analysis += '• Buen trabajo en las rondas decisivas, pero hay margen para cerrar mapas con más autoridad.';
  } else if (isWin && !wasClose) {
    analysis += 'Observaciones:\n\n';
    analysis += '• Victoria dominante. Tus aperturas fueron sólidas y las rotaciones rápidas.\n';
    analysis += '• Aún así, identifiqué rondas perdidas innecesariamente por asomarte de más en solitario.\n';
    analysis += '• Consolida estos hábitos positivos y trabaja en reducir las muertes innecesarias.';
  } else if (!isWin && wasClose) {
    analysis += 'Observaciones:\n\n';
    analysis += '• Derrota ajustada. Estuviste cerca pero las rondas de pistola y los clutch definieron el partido.\n';
    analysis += '• He detectado un patrón: en las rondas 13-16 tu equipo tendió a forzar duelos desfavorables.\n';
    analysis += '• Recomiendo trabajar la disciplina en rondas de presión — mantener posición y no asomarse de más.';
  } else {
    analysis += 'Observaciones:\n\n';
    analysis += '• Derrota clara. El equipo contrario controló la economía y las aperturas a mitad de ronda.\n';
    analysis += '• He detectado falta de utilidad coordinada en las entradas a sitio.\n';
    analysis += '• Recomiendo enfocarse en: 1) Tiempo de flashes de entrada, 2) Humos de corte, 3) Disciplina económica.';
  }

  return analysis;
};

const buildFollowupResponse = (text) => {
  const lowerText = text.toLowerCase();

  if (lowerText.includes('aim') || lowerText.includes('punter')) {
    return {
      id: nextId(),
      sender: 'ai',
      context: 'PLAN DE PUNTERÍA',
      text: 'Te propongo una rutina simple de 20 minutos: 8 min de seguimiento, 8 min de precisión inicial y 4 min de cambio entre objetivos. Si quieres, te la adapto a tu horario.'
    };
  }

  if (lowerText.includes('utility') || lowerText.includes('granada') || lowerText.includes('utilidad')) {
    return {
      id: nextId(),
      sender: 'ai',
      context: 'UTILIDAD FUNDAMENTAL',
      text: 'Vamos a trabajar 2 protocolos base: cegadora de entrada + humo de corte. El objetivo es repetirlos hasta que sean automáticos.'
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
      text: 'La clave está en sincronizar las compras con tu equipo. Evita compras forzadas individuales y aprende a leer las bonificaciones por derrota del rival para optimizar tus ecos.'
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

// History and pending replies belong to one authenticated identity.

const useCoachChat = (user) => {
  const owner = user?.steam_id || null;
  const [chatInput, setChatInput] = useState('');
  const [isAiTyping, setIsAiTyping] = useState(false);
  const [conversation, setConversation] = useState(() => ({ owner, messages: readCoachHistory(owner) }));
  const messages = conversation.owner === owner ? conversation.messages : [];
  const pendingReplies = useRef(new Set());
  const hasGreetedRef = useRef(false);
  const setMessages = useCallback(update => {
    setConversation(current => current.owner !== owner ? current : {
      owner, messages: (typeof update === 'function' ? update(current.messages) : update).slice(-100),
    });
  }, [owner]);

  useEffect(() => {
    setConversation(current => current.owner === owner ? current : { owner, messages: readCoachHistory(owner) });
    setChatInput('');
    setIsAiTyping(false);
    hasGreetedRef.current = false;
    const pending = pendingReplies.current;
    return () => { pending.forEach(cancel => cancel()); pending.clear(); };
  }, [owner]);

  useEffect(() => {
    if (conversation.owner === owner) saveCoachHistory(owner, conversation.messages);
  }, [conversation, owner]);

  const quickActions = useMemo(
    () => ['Quiero mejorar mi puntería', 'Hazme un plan de utilidad', 'Quiero ser más consistente'],
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
  }, [setMessages]);

  /* ── Queue multiple AI messages sequentially with typewriter ── */
  const queueAiMessages = useCallback((msgDataArray) => {
    if (!owner || !msgDataArray.length) return;

    setIsAiTyping(true);
    let idx = 0;
    const cleanups = [];
    const cancel = () => cleanups.forEach(fn => fn?.());
    pendingReplies.current.add(cancel);

    const typeNext = () => {
      if (idx >= msgDataArray.length) {
        setIsAiTyping(false);
        pendingReplies.current.delete(cancel);
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
    return cancel;
  }, [addAiMessageWithTyping, owner]);

  /* ── Proactive greeting: called by Dashboard when matches load ── */
  const sendProactiveGreeting = useCallback((matches) => {
    if (!owner || hasGreetedRef.current) return;
    hasGreetedRef.current = true;

    // Do not send greeting if we already have restored messages (user navigated back to the tab)
    if (messages.length > 0) return;

    const greetingMessages = buildPatternGreeting(user?.username, matches);
    queueAiMessages(greetingMessages);
  }, [owner, user?.username, queueAiMessages, messages.length]);

  /* ── Auto-analyze match: called by Dashboard when "Analizar" is clicked ── */
  const analyzeMatch = useCallback((match) => {
    if (!owner || !match) return;

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
  }, [owner, queueAiMessages, setMessages]);

  /* ── Standard user message ── */
  const submitMessage = useCallback((value) => {
    const cleanValue = value.trim();
    if (!owner || !cleanValue) return;

    const userMessage = createCoachMessage({
      id: nextId(),
      sender: 'user',
      text: cleanValue
    });

    setMessages((prev) => [...prev, userMessage]);
    setChatInput('');

    const responseData = buildFollowupResponse(cleanValue);
    queueAiMessages([responseData]);
  }, [owner, queueAiMessages, setMessages]);

  return {
    messages,
    chatInput: conversation.owner === owner ? chatInput : '',
    setChatInput,
    isAiTyping: Boolean(owner && conversation.owner === owner && isAiTyping),
    quickActions,
    submitMessage,
    sendProactiveGreeting,
    analyzeMatch
  };
};

export default useCoachChat;
