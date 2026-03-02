// GameSenseChallenge - Tactical Decision Assessment
// Video pauses at critical moment → user picks strategy → video resumes
import React, { useState, useRef } from 'react';
import { useLanding } from '../../LandingContext';
import { motion, AnimatePresence } from 'framer-motion';
import { Play, Brain, ClipboardList, TrendingUp, ShieldCheck, AlertCircle } from 'lucide-react';
import { getObservationNote } from '../observationNotes';
import '../../../../styles/Landing/challenges/gamesenseRedesign.css';

const DECISION_POINT = 7;
const CORRECT_OPTION = 3;
const VIDEO_SRC = "/videos/GameSense.mp4";

const OPTIONS = [
  { 
    id: 0, letter: "A", 
    title: "Peekear agresivo", 
    description: "Salir corriendo y buscar aim duels directos contra los dos."
  },
  { 
    id: 1, letter: "B", 
    title: "Mantener posición", 
    description: "Esperar pasivamente en un ángulo y dejar que vengan a ti."
  },
  { 
    id: 2, letter: "C", 
    title: "Humo defensivo", 
    description: "Tirar el humo encima tuyo para esconderte y ganar tiempo."
  },
  { 
    id: 3, letter: "D", 
    title: "Aislar con humo", 
    description: "Bloquear la visión de un enemigo para forzar un 1vs1 justo."
  }
];

const FEEDBACK = {
  correct: {
    title: "Excelente lectura táctica",
    explanation: "Elegiste aislar con el humo, convirtiendo un 1vs2 desfavorable en dos duelos 1vs1 donde tienes ventaja de información. Esto es exactamente lo que separa a un jugador reactivo de uno que crea sus propias oportunidades.",
    tip: "Los one-way smokes no son solo un trick — son una herramienta de supervivencia en situaciones de clutch."
  },
  incorrect: {
    0: {
      title: "Demasiado agresivo",
      explanation: "Salir a buscar los duelos expone tu posición y te obliga a ganar DOS aim duels consecutivos. El segundo enemigo ya sabe dónde estás cuando matas al primero — o peor, ambos te ven a la vez.",
      better: "En inferioridad numérica nunca tomes duelos secos. Tu utilidad existe para cambiar las condiciones del enfrentamiento a tu favor."
    },
    1: {
      title: "Pasivo no es lo mismo que inteligente",
      explanation: "Esperar pasivamente parece seguro, pero le das toda la iniciativa al enemigo. Ellos son 2: pueden coordinarse, uno te flashea y el otro te peekea, o simplemente hacen crossfire a tu ángulo.",
      better: "La diferencia clave: ser pasivo es ESPERAR a que pase algo. Ser táctico es CREAR la situación que te conviene. Tenías un humo — úsalo."
    },
    2: {
      title: "Buena idea, mala ejecución",
      explanation: "Usar el humo para esconderte te compra 15 segundos, pero no cambia nada. Cuando se disipe, sigues en 1vs2 y ellos saben exactamente dónde estás. Solo retrasas el problema.",
      better: "Un humo defensivo gana tiempo. Un humo ofensivo gana ventaja. La diferencia es si el humo te protege a ti o si divide al enemigo."
    }
  }
};

const GameSenseChallenge = () => {
  const { completeChallenge, skipChallenge, completedChallenges } = useLanding();
  const videoRef = useRef(null);
  
  const [phase, setPhase] = useState('intro'); // intro, playing, deciding, analyzing, result
  const [selectedOption, setSelectedOption] = useState(null);
  const [isCorrect, setIsCorrect] = useState(false);
  const [hasDecided, setHasDecided] = useState(false);

  const handleTimeUpdate = () => {
    if (!videoRef.current) return;
    const time = videoRef.current.currentTime;
    if (time >= DECISION_POINT && phase === 'playing' && !hasDecided) {
      videoRef.current.pause();
      setPhase('deciding');
    }
  };

  const handleVideoEnd = () => {
    if (phase === 'playing' || selectedOption !== null) {
      setPhase('analyzing');
      const correct = selectedOption === CORRECT_OPTION;
      
      // Brief "AI analyzing" state before revealing result
      setTimeout(() => {
        setIsCorrect(correct);
        setPhase('result');
        completeChallenge('gamesense', { selectedOption, isCorrect: correct }, correct);
      }, 1200);
    }
  };

  const handleStart = () => {
    setPhase('playing');
    if (videoRef.current) {
      const playPromise = videoRef.current.play();
      if (playPromise !== undefined) {
        playPromise.catch(() => setPhase('deciding'));
      }
    }
  };

  const handleVideoError = () => {
    setPhase('deciding');
  };

  const handleSelect = (index) => {
    if (phase !== 'deciding') return;
    setSelectedOption(index);
    setHasDecided(true);
    setPhase('playing');
    if (videoRef.current) videoRef.current.play();
  };

  // ── INTRO SCREEN ───────────────────────────────────────
  if (phase === 'intro') {
    return (
      <div className="gamesense-challenge">
        <motion.div 
          className="gamesense-challenge__intro"
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.4 }}
        >
          <div className="gamesense-challenge__intro-badge">
            <Brain size={14} />
            <span>Análisis de Situación</span>
          </div>
          
          <h3 className="gamesense-challenge__intro-title">Situación de Clutch</h3>
          
          <div className="gamesense-challenge__intro-stats">
            <div className="gamesense-challenge__stat">
              <span className="gamesense-challenge__stat-label">Situación</span>
              <span className="gamesense-challenge__stat-value">1 vs 2</span>
            </div>
            <div className="gamesense-challenge__stat">
              <span className="gamesense-challenge__stat-label">Objetivo</span>
              <span className="gamesense-challenge__stat-value">Retake A</span>
            </div>
            <div className="gamesense-challenge__stat">
              <span className="gamesense-challenge__stat-label">Utilidad</span>
              <span className="gamesense-challenge__stat-value">1 Humo</span>
            </div>
          </div>

          <p className="gamesense-challenge__intro-desc">
            El vídeo se pausará en el momento crítico. Analiza la situación y elige la mejor estrategia.
          </p>

          <button className="gamesense-challenge__start-btn" onClick={handleStart}>
            <Play size={18} />
            Iniciar simulación
          </button>

          <button className="ch-skip" onClick={() => skipChallenge('gamesense')}>
            Saltar evaluación
          </button>
        </motion.div>
      </div>
    );
  }

  // ── MAIN VIEW (video + decision + result) ──────────────
  return (
    <div className="gamesense-challenge">
      {/* Context bar */}
      <motion.div 
        className="gamesense-challenge__context-bar"
        initial={{ opacity: 0, y: -8 }}
        animate={{ opacity: 1, y: 0 }}
      >
        <div className={`gamesense-challenge__status-dot ${phase === 'deciding' ? 'critical' : ''}`} />
        <span className="gamesense-challenge__status-text">
          {phase === 'deciding' 
            ? 'Momento crítico — Elige tu acción' 
            : phase === 'analyzing' || phase === 'result'
              ? 'Evaluación completada'
              : '1vs2 Clutch · 2 enemigos localizados · 40s restantes'}
        </span>
      </motion.div>

      {/* Video — visible only during playing/deciding */}
      {(phase === 'playing' || phase === 'deciding') && (
        <div className="gamesense-challenge__video-container">
          <video
            ref={videoRef}
            src={VIDEO_SRC}
            className="gamesense-challenge__video"
            onTimeUpdate={handleTimeUpdate}
            onEnded={handleVideoEnd}
            onError={handleVideoError}
            playsInline
            muted
            autoPlay
          />
        </div>
      )}

      {/* AI Analyzing State — standalone, replaces video */}
      {phase === 'analyzing' && (
        <motion.div
          className="ch-ai-processing"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
        >
          <div className="ch-ai-processing__dot" />
          <div className="ch-ai-processing__dot" />
          <div className="ch-ai-processing__dot" />
          <span>Analizando respuesta...</span>
        </motion.div>
      )}

      {/* Result — standalone card, consistent with Economy/Grenade */}
      <AnimatePresence>
        {phase === 'result' && (
          <motion.div 
            className={`ch-result ${isCorrect ? 'ch-result--success' : 'ch-result--error'}`}
            initial={{ opacity: 0, scale: 0.96 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.3 }}
          >
            <div className="ch-result__icon">
              {isCorrect ? <ShieldCheck size={24} /> : <AlertCircle size={24} />}
            </div>
            
            <h3 className="ch-result__title">
              {isCorrect ? FEEDBACK.correct.title : FEEDBACK.incorrect[selectedOption]?.title}
            </h3>
            
            <div className="ch-result__explanation">
              <div className="ch-result__explanation-label">
                <TrendingUp size={14} />
                Análisis táctico
              </div>
              <p className="ch-result__explanation-text">
                {isCorrect ? FEEDBACK.correct.explanation : FEEDBACK.incorrect[selectedOption]?.explanation}
              </p>
            </div>

            {isCorrect ? (
              <div className="gamesense-challenge__result-tip">
                {FEEDBACK.correct.tip}
              </div>
            ) : (
              <div className="gamesense-challenge__result-better">
                {FEEDBACK.incorrect[selectedOption]?.better}
              </div>
            )}

            <div className="gamesense-challenge__result-selection">
              Tu selección: <strong>{OPTIONS[selectedOption]?.letter} — {OPTIONS[selectedOption]?.title}</strong>
            </div>

            {/* Cumulative AI Observation */}
            <motion.div 
              className="ch-observation"
              initial={{ opacity: 0, y: 6 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.4, duration: 0.35 }}
            >
              <div className="ch-observation__header">
                <ClipboardList size={14} />
                <span className="ch-observation__label">
                  {getObservationNote('gamesense', isCorrect, completedChallenges).label}
                </span>
              </div>
              <p className="ch-observation__text">
                {getObservationNote('gamesense', isCorrect, completedChallenges).note}
              </p>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Decision options — below video, 2x2 grid */}
      <AnimatePresence>
        {phase === 'deciding' && (
          <motion.div 
            className="gamesense-challenge__options"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 12 }}
            transition={{ duration: 0.3 }}
          >
            {OPTIONS.map((option, i) => (
              <motion.button
                key={option.id}
                className="ch-option"
                onClick={() => handleSelect(option.id)}
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: i * 0.06 }}
              >
                <span className="ch-option__letter">{option.letter}</span>
                <div className="ch-option__content">
                  <span className="ch-option__title">{option.title}</span>
                  <span className="ch-option__text">{option.description}</span>
                </div>
              </motion.button>
            ))}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

export default GameSenseChallenge;
