import React, { useCallback, useEffect, useRef, useState } from 'react';
import { AlertTriangle, Bot, CheckCircle, Eye, MessageSquare, Pause, Play, RotateCcw } from 'lucide-react';
import { CHAT_DEMO_SCRIPT, FIRST_INSIGHT_MOMENT, CRITICAL_MOMENT, INSIGHT_DELAY_FIRST, INSIGHT_DELAY_CRITICAL, RESUME_DELAY } from './replayDemoData';
import { LANDING_COPY } from './landingContent';

const script = CHAT_DEMO_SCRIPT.es;
const isAnalysis = phase => phase === 'first' || phase === 'critical';

function DemoMessage({ message, immediate, onProgress }) {
  const [length, setLength] = useState(immediate || message.isUser ? message.text.length : 0);
  useEffect(() => {
    if (immediate || message.isUser) return undefined;
    let shown = 0;
    const timer = window.setInterval(() => {
      shown = Math.min(message.text.length, shown + 2);
      setLength(shown); onProgress();
      if (shown === message.text.length) window.clearInterval(timer);
    }, 25);
    return () => window.clearInterval(timer);
  }, [message, immediate, onProgress]);
  const Icon = message.isUser ? MessageSquare : ['error','warning','consequence'].includes(message.type) ? AlertTriangle : CheckCircle;
  const displayed = immediate || message.isUser ? message.text : message.text.slice(0, length);
  return <article className="demo-message" data-tone={message.type} aria-label={message.text}>
    <span className="demo-message-author"><Icon size={15} />{message.isUser ? 'Tú' : 'Coach IA'}</span>
    <p aria-hidden="true">{displayed}{displayed.length < message.text.length && <span className="demo-writing" />}</p>
  </article>;
}

export function LandingDemo({ reducedMotion = false }) {
  const video = useRef(), chat = useRef(), phaseRef = useRef('playing');
  const visited = useRef(new Set()), answers = useRef(new Set()), answerTimers = useRef(new Set());
  const mounted = useRef(false), playRequest = useRef(0);
  const [phase, setPhase] = useState('playing');
  const [playing, setPlaying] = useState(false), [time, setTime] = useState(0);
  const [index, setIndex] = useState(0), [messages, setMessages] = useState([]), [answered, setAnswered] = useState([]);
  const [analysisRun, setAnalysisRun] = useState(0);
  const changePhase = useCallback(value => { phaseRef.current = value; setPhase(value); }, []);
  const append = useCallback(message => setMessages(current => current.some(item => item.id === message.id) ? current : [...current, message]), []);
  const scrollToLatest = useCallback(() => { if (chat.current) chat.current.scrollTop = chat.current.scrollHeight; }, []);
  const play = useCallback(() => {
    const request = ++playRequest.current;
    try {
      Promise.resolve(video.current?.play()).then(() => {
        if (mounted.current && request === playRequest.current && !isAnalysis(phaseRef.current)) setPlaying(true);
      }).catch(() => { if (mounted.current && request === playRequest.current) setPlaying(false); });
    } catch { if (mounted.current) setPlaying(false); }
  }, []);
  useEffect(() => {
    mounted.current = true;
    const player = video.current, timers = answerTimers.current;
    play();
    return () => {
      mounted.current = false; playRequest.current++;
      player?.pause(); for (const timer of timers) window.clearTimeout(timer); timers.clear();
    };
  }, [play]);
  useEffect(scrollToLatest, [messages, scrollToLatest]);
  useEffect(() => {
    if (!isAnalysis(phase)) return undefined;
    const insights = phase === 'first' ? script.firstInsights : script.criticalInsights;
    if (index >= insights.length) {
      const timer = window.setTimeout(() => { changePhase('playing'); play(); }, RESUME_DELAY);
      return () => window.clearTimeout(timer);
    }
    const delay = index === 0 ? 500 : 1500;
    const show = window.setTimeout(() => append(insights[index]), delay);
    const next = window.setTimeout(() => setIndex(current => current + 1), delay + (phase === 'first' ? INSIGHT_DELAY_FIRST : INSIGHT_DELAY_CRITICAL));
    return () => { window.clearTimeout(show); window.clearTimeout(next); };
  }, [phase, index, analysisRun, append, changePhase, play]);
  const beginAnalysis = kind => {
    const moment = kind === 'first' ? FIRST_INSIGHT_MOMENT : CRITICAL_MOMENT;
    visited.current.add(kind); setIndex(0); setAnalysisRun(current => current + 1); changePhase(kind);
    video.current?.pause();
    // Freeze the calibrated frame: timeupdate may arrive after its checkpoint.
    if (video.current) video.current.currentTime = moment;
    setTime(moment); setPlaying(false);
  };
  const updateTime = event => {
    const current = event.currentTarget.currentTime; setTime(current);
    if (phaseRef.current !== 'playing') return;
    if (!visited.current.has('first') && current >= FIRST_INSIGHT_MOMENT) beginAnalysis('first');
    else if (!visited.current.has('critical') && current >= CRITICAL_MOMENT) beginAnalysis('critical');
  };
  const clearAnswers = () => {
    for (const timer of answerTimers.current) window.clearTimeout(timer);
    answerTimers.current.clear(); answers.current.clear(); setAnswered([]);
  };
  const restart = () => {
    clearAnswers(); visited.current.clear(); setMessages([]); setIndex(0); setTime(0);
    if (phase === 'error') video.current?.load();
    if (video.current) video.current.currentTime = 0;
    changePhase('playing'); play();
  };
  const chapter = kind => {
    clearAnswers(); setMessages(kind === 'critical' ? [...script.firstInsights] : []);
    visited.current = new Set(kind === 'critical' ? ['first','critical'] : ['first']);
    beginAnalysis(kind);
  };
  const toggle = () => {
    if (phase === 'ended' || phase === 'error') restart();
    else if (isAnalysis(phase)) { changePhase('playing'); play(); }
    else if (playing) video.current?.pause();
    else play();
  };
  const ask = question => {
    if (answers.current.has(question.id)) return;
    answers.current.add(question.id); setAnswered(current => [...current, question.id]);
    append({ id:`question-${question.id}`, text:question.question, type:'question', isUser:true });
    const timer = window.setTimeout(() => {
      answerTimers.current.delete(timer); append({ id:`answer-${question.id}`, text:question.response, type:'solution' });
    }, 1000);
    answerTimers.current.add(timer);
  };
  const insights = phase === 'first' ? script.firstInsights : script.criticalInsights;
  const highlighted = isAnalysis(phase) ? insights[Math.min(index, insights.length-1)]?.player : null;
  const status = phase === 'first' ? script.phaseLabels.firstInsights : phase === 'critical' ? script.phaseLabels.criticalInsights : phase === 'ended' ? script.phaseLabels.roundFinished : phase === 'error' ? 'Vídeo no disponible' : playing ? script.phaseLabels.playing : 'Pulsa reproducir para continuar';
  const action = phase === 'error' ? 'Reintentar vídeo' : phase === 'ended' ? 'Volver a ver la jugada' : isAnalysis(phase) ? 'Continuar jugada' : playing ? 'Pausar jugada' : 'Reproducir jugada';
  return <section className="landing-demo replay-demo" aria-label="Demo táctica original" data-replay-phase={phase}>
    <div className="demo-context"><strong>{LANDING_COPY.aiDemo.map} · {LANDING_COPY.aiDemo.situation}</strong><span>{LANDING_COPY.aiDemo.playerContext}</span></div>
    <div className="replay-workspace">
      <div className="demo-viewer">
        <div className="demo-screen">
          <video ref={video} autoPlay muted playsInline preload="auto" poster="/images/Landing/ImagenReplay.png"
            aria-label="Vídeo de la jugada de Dust2 del landing original" src="/videos/ChatIA.mp4"
            onTimeUpdate={updateTime} onPlay={() => setPlaying(true)} onPause={() => setPlaying(false)}
            onError={() => { changePhase('error'); setPlaying(false); }}
            onEnded={() => { if (phaseRef.current === 'ended') return; setPlaying(false); changePhase('ended'); append(script.finalSummary); }} />
          {highlighted && <div className={`demo-player-focus focus-${highlighted}`} aria-hidden="true"><span className="demo-highlight-ring" /><span className="demo-highlight-label"><Eye size={12} />{highlighted === 'corta' ? script.phaseLabels.you : script.phaseLabels.teammate}</span></div>}
          {(isAnalysis(phase) || phase === 'ended') && <div className="demo-analysis-badge" data-critical={phase !== 'first'}><span />{status}</div>}
          {!playing && !isAnalysis(phase) && phase !== 'ended' && <button className="demo-play-overlay" onClick={toggle} aria-label={phase === 'error' ? 'Reintentar vídeo en el mapa' : 'Reproducir vídeo'}><Play size={25} /><span>{phase === 'error' ? 'Reintentar vídeo' : 'Reproducir jugada'}</span></button>}
        </div>
        <div className="demo-transport">
          <button onClick={toggle} aria-label={action}>{playing ? <Pause size={17} /> : <Play size={17} />}<span>{action}</span></button>
          <span className="demo-clock">00:{Math.floor(time).toString().padStart(2,'0')}</span>
          <button onClick={restart} aria-label="Reiniciar demo"><RotateCcw size={16} /><span>Reiniciar</span></button>
        </div>
        <nav className="demo-chapters" aria-label="Momentos de la demo"><button onClick={() => chapter('first')}>00:01 · Contexto</button><button onClick={() => chapter('critical')}>00:06 · Decisión</button></nav>
      </div>
      <aside className="demo-coach" aria-label="Análisis de la jugada">
        <header><span><Bot size={21} />Coach IA</span><span className="demo-coach-state" role="status">{status}</span></header>
        <div className="demo-messages" ref={chat} role="log" aria-label="Comentarios del Coach IA" aria-live="polite" aria-relevant="additions">
          {messages.length === 0 && <div className="demo-waiting"><Eye size={28} /><h3>La jugada, paso a paso.</h3><p>El replay se detiene en los momentos clave mientras el Coach explica qué está pasando.</p></div>}
          {messages.map(message => <DemoMessage key={message.id} message={message} immediate={reducedMotion} onProgress={scrollToLatest} />)}
        </div>
        {isAnalysis(phase) && <p className="demo-resume-note">La jugada continuará automáticamente tras el análisis.</p>}
        {phase === 'ended' && <div className="demo-followup"><p>{script.phaseLabels.questionsPrompt}</p>{script.userQuestions.filter(question => !answered.includes(question.id)).map(question => <button key={question.id} onClick={() => ask(question)}><MessageSquare size={14} />{question.question}</button>)}</div>}
      </aside>
    </div>
    <p className="content-disclosure">Demo guionizada del landing original. Los comentarios ilustran la propuesta del Coach IA, en desarrollo.</p>
    {phase === 'error' && <p role="alert" className="content-disclosure">No se ha podido cargar el vídeo. Puedes reintentarlo con el botón de reproducción.</p>}
  </section>;
}
