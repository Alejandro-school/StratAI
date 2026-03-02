// EconomyChallenge - Tactical Economy Assessment
// No start screen — jumps directly into the evaluation
import React, { useState, useMemo } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { ShieldCheck, AlertCircle, TrendingUp, ClipboardList } from 'lucide-react';
import { useLanding } from '../../LandingContext';
import { getObservationNote } from '../observationNotes';
import '../../../../styles/Landing/challenges/economyPremium.css';

/**
 * Economy Question Bank
 */
const QUESTIONS = [
  {
    id: 1,
    map: 'Inferno',
    side: 'CT',
    round: 9,
    score: { ct: 4, t: 5 },
    context: `Tu equipo lleva <strong>4 rondas ganadas</strong> y el enemigo <strong>5</strong>. 
              El enemigo ha ganado la ronda anterior y tiene <span class="ch-highlight--danger">Full Buy</span>. 
              Todos tenéis alrededor de <span class="ch-highlight--money">$3,000</span> y un bonus de derrota de <span class="ch-highlight--money">$1,900</span>.`,
    money: { current: 3000, nextMin: 5400 },
    question: '¿Cuál es la decisión económica correcta?',
    options: [
      { letter: 'A', text: 'Eco completa: guardar todo para ronda 10', correct: true, explanation: `En la ronda 10 tendréis <strong>$5,400</strong> ($3,000 + $2,400 de bonus). Esto garantiza rifles, armadura, utilidad y kits de desactivación — vitales en Inferno para frenar ejecuciones en B y controlar medio.` },
      { letter: 'B', text: 'Semi-buy: FAMAS/SMG + chaleco (sin utilidad)', correct: false, explanation: 'Un FAMAS/SMG sin granadas contra un Full Buy es un gasto de $3,050 que probablemente perderás. Peor: <strong>arruinas la compra completa de ronda 10</strong> sin tener utilidad para defender ningún sitio.' },
      { letter: 'C', text: 'Un jugador compra AWP, el resto eco', correct: false, explanation: 'Dar todo el presupuesto a un solo jugador no funciona en CT. Sin granadas colectivas, el enemigo ejecuta libremente. Además, si el AWPer muere primero, la inversión se pierde.' },
      { letter: 'D', text: 'Forzar todos: pistola mejorada + kevlar', correct: false, explanation: 'Comprar pistola + kevlar ($1,650) os deja en tierra de nadie: sin ventaja para ganar la ronda y sin dinero suficiente para una compra completa después si perdéis.' },
    ],
  },
  {
    id: 2,
    map: 'Mirage',
    side: 'CT',
    round: 4,
    score: { ct: 1, t: 2 },
    context: `Marcador <strong>1-2</strong>. El enemigo viene de ganar y tiene <span class="ch-highlight--danger">Full Buy</span>. 
              Tienes <span class="ch-highlight--money">$3,500</span> y vuestro bonus de derrota es de <span class="ch-highlight--money">$2,400</span>.`,
    money: { current: 3500, nextMin: 5900 },
    question: '¿Cómo priorizas tu compra este round?',
    options: [
      { letter: 'A', text: 'M4A4 + kevlar (pero sin granadas ni kit)', correct: false, explanation: 'Un M4 solo en Mirage es medio M4. Sin humos, no puedes cortar un rush a Ramp ni controlar Palace. El rifle sin utilidad <strong>no defiende sitios</strong>, defiende ángulos — y en CT necesitas ambos.' },
      { letter: 'B', text: 'FAMAS/SMG + kevlar + humo + molotov + kit', correct: true, explanation: `En Mirage, la utilidad es más importante que el arma. Un FAMAS/SMG con humo + molotov <strong>frena ejecuciones completas</strong>. Si pierdes, el bonus de $2,900 te permite full buy en ronda 5. Arma inferior, control superior.` },
      { letter: 'C', text: 'Eco total y guardar para AWP en ronda 5', correct: false, explanation: 'Hacer eco total concede la ronda gratuitamente. El enemigo sube a 3-1, gana confianza, y un AWP solitaria en ronda 5 no arregla que tu equipo lleva 3 ecos seguidos.' },
      { letter: 'D', text: 'Comprar solo armadura + granadas (sin arma)', correct: false, explanation: 'Granadas sin arma es una receta para hacer daño sin poder capitalizar. Retrasas un rush 5 segundos con el molly, pero ¿y luego? Con una P250 no ganas el duelo post-molly.' },
    ],
  },
  {
    id: 3,
    map: 'Anubis',
    side: 'ct',
    round: 2,
    score: { ct: 1, t: 0 },
    context: `Ganasteis la ronda de pistolas y <span class="ch-highlight--warning">el enemigo no plantó</span>. 
              Tienes <span class="ch-highlight--money">$3,200</span>. El enemigo está en eco (solo pistolas, sin armadura).`,
    money: { current: 3200, nextMin: 3200 },
    question: '¿Cuál es la compra óptima contra un eco enemigo?',
    options: [
      { letter: 'A', text: 'MP9 + kevlar + utilidad para farmear bajas', correct: true, explanation: `La MP9 da <strong>$600 por baja</strong> (el doble que un rifle). Contra un eco sin armadura, las SMG son igual de letales pero generan el doble de dinero. Esto dispara la economía de tu equipo para las rondas 3-5.` },
      { letter: 'B', text: 'M4A1-S + kevlar para asegurar al máximo', correct: false, explanation: 'El M4 cuesta $2,900 y da $300 por baja. Contra pistolas sin kevlar, es matar moscas a cañonazos: <strong>gastas más, generas menos</strong>. El rifle es para cuando el enemigo también tiene rifles.' },
      { letter: 'C', text: 'Solo kevlar, guardar el resto del dinero', correct: false, explanation: 'El kevlar te protege, pero la USP-S tiene cadencia lenta y poco daño a cuerpo. Un Glock te puede matar en un spray de cerca. Necesitas un arma que domine las anti-ecos, no solo sobrevivir.' },
      { letter: 'D', text: 'Eco completa para tener $6,400 en ronda 3', correct: false, explanation: 'Ya ganaste la pistola — este es el momento de presionar ventaja, no de guardar. El enemigo está débil AHORA. Hacer eco te regala una oportunidad de que te empaten con una compra forzada.' },
    ],
  },
  {
    id: 4,
    map: 'Inferno',
    side: 'ct',
    round: 15,
    score: { ct: 7, t: 7 },
    context: `Situación <span class="ch-highlight--danger">1 vs 3</span> en el sitio B. Tienes una <strong>AWP</strong> y <span class="ch-highlight--money">$0</span> en el banco. 
              Si mueres, cobras <span class="ch-highlight--money">$2,900</span> de bonus de derrota.`,
    money: { current: 0, nextMin: 2900 },
    question: '¿Qué decisión maximiza el valor económico para tu equipo?',
    options: [
      { letter: 'A', text: 'Salvar la AWP y buscar exit kills', correct: true, explanation: `La AWP vale <strong>$4,750</strong>. Salvarla ahorra casi $5,000 al equipo — mucho más que el bonus de $2,900 por morir. Además, entras a la siguiente ronda con un arma de $4,750 mientras el otro equipo planifica para rondas de post-planta.` },
      { letter: 'B', text: 'Intentar el retake', correct: false, explanation: 'Un 1v3 con AWP en sitio B de Inferno es un suicidio táctico. El AWP necesita distancia y un solo ángulo; los 3 enemigos pueden hacer crossfire desde New Box, Dark y Planta. Pierdes el arma y la ronda.' },
      { letter: 'C', text: 'Intentar cambiar de arma y jugar el retake', correct: false, explanation: 'Es una opción muy arriesgada y con una baja probabilidad de éxito, no es la mejor opción, no obstante, hay algúna posibilidad de que salga bien (No recomendada)' },
      { letter: 'D', text: 'Esperar a que quede poco tiempo e intentar un ninja defuse', correct: false, explanation: 'Intentar un ninja defuse en un 1 vs 3 es extremadamente arriesgado y poco probable que tenga éxito. Además, si te descubren, pierdes la AWP y la ronda.' },
    ],
  },
];

const EconomyChallenge = () => {
  const { completeChallenge, saveResult, skipChallenge, completedChallenges } = useLanding();
  
  const [randomQuestionIndex] = useState(() => Math.floor(Math.random() * QUESTIONS.length));
  const [pendingOption, setPendingOption] = useState(null);
  const [selectedOption, setSelectedOption] = useState(null);
  const [showFeedback, setShowFeedback] = useState(false);
  const [isCorrect, setIsCorrect] = useState(false);
  const [isAnalyzing, setIsAnalyzing] = useState(false);

  const currentQuestion = useMemo(() => QUESTIONS[randomQuestionIndex], [randomQuestionIndex]);

  const handleOptionClick = (index) => {
    if (showFeedback || selectedOption !== null || isAnalyzing) return;
    setPendingOption(index);
  };

  const handleConfirmSelection = () => {
    if (pendingOption === null) return;
    
    const isAnswerCorrect = currentQuestion.options[pendingOption].correct;
    setSelectedOption(pendingOption);
    setIsAnalyzing(true);
    
    // Brief "AI analyzing" state before revealing result
    setTimeout(() => {
      setIsCorrect(isAnswerCorrect);
      setShowFeedback(true);
      setIsAnalyzing(false);
      
      const finalData = {
        questionId: currentQuestion.id,
        selectedAnswer: currentQuestion.options[pendingOption].letter,
        isCorrect: isAnswerCorrect,
        map: currentQuestion.map,
      };
      
      saveResult('economy', finalData);
      completeChallenge('economy', finalData, isAnswerCorrect);
    }, 1200);
  };

  const handleCancelSelection = () => {
    setPendingOption(null);
  };

  return (
    <div className="economy-challenge">
      <motion.div 
        className="economy-challenge__content"
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4 }}
      >
        {/* Main Question UI — hidden during feedback/analyzing */}
        {!showFeedback && !isAnalyzing && (
          <>
            {/* Scenario Data Bar */}
            <div className="economy-challenge__scenario-bar">
              <div className="economy-challenge__data-group">
                <span className="economy-challenge__data-label">Mapa</span>
                <span className="economy-challenge__data-value">{currentQuestion.map}</span>
              </div>
              <div className="economy-challenge__data-group">
                <span className="economy-challenge__data-label">Ronda</span>
                <span className="economy-challenge__data-value">{currentQuestion.round}/30</span>
              </div>
              <div className="economy-challenge__score-display">
                <span className="economy-challenge__score economy-challenge__score--ct">
                  {currentQuestion.score.ct}
                </span>
                <span className="economy-challenge__score-sep">:</span>
                <span className="economy-challenge__score economy-challenge__score--t">
                  {currentQuestion.score.t}
                </span>
              </div>
              <div className="economy-challenge__data-group economy-challenge__data-group--right">
                <span className="economy-challenge__data-label">Lado</span>
                <span className={`economy-challenge__data-value economy-challenge__data-value--${currentQuestion.side}`}>
                  {currentQuestion.side === 'ct' ? 'CT' : 'T'}
                </span>
              </div>
            </div>

            {/* Money Info */}
            <div className="economy-challenge__money-row">
              <div className="economy-challenge__money-item">
                <span className="economy-challenge__money-label">Banco actual</span>
                <span className="economy-challenge__money-value economy-challenge__money-value--current">
                  ${currentQuestion.money.current.toLocaleString()}
                </span>
              </div>
              <div className="economy-challenge__money-item">
                <span className="economy-challenge__money-label">Mínimo siguiente ronda</span>
                <span className="economy-challenge__money-value economy-challenge__money-value--next">
                  ${currentQuestion.money.nextMin.toLocaleString()}
                </span>
              </div>
            </div>

            {/* Context Brief */}
            <div className="economy-challenge__context">
              <p dangerouslySetInnerHTML={{ __html: currentQuestion.context }} />
            </div>

            {/* Question */}
            <div className="economy-challenge__question">
              {currentQuestion.question}
            </div>

            {/* Options */}
            <div className="economy-challenge__options">
              {currentQuestion.options.map((option, index) => {
                const isPending = pendingOption === index;
                return (
                  <motion.button
                    key={index}
                    className={`ch-option ${isPending ? 'ch-option--selected' : ''}`}
                    onClick={() => handleOptionClick(index)}
                    whileTap={{ scale: 0.99 }}
                  >
                    <span className="ch-option__letter">{option.letter}</span>
                    <span className="ch-option__text">{option.text}</span>
                  </motion.button>
                );
              })}
            </div>

            {/* Confirm / Cancel */}
            <AnimatePresence>
              {pendingOption !== null && (
                <motion.div
                  className="ch-confirm"
                  initial={{ opacity: 0, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: 8 }}
                >
                  <button className="ch-confirm__btn" onClick={handleConfirmSelection}>
                    <ShieldCheck size={16} />
                    Confirmar opción {String.fromCharCode(65 + pendingOption)}
                  </button>
                  <button className="ch-confirm__cancel" onClick={handleCancelSelection}>
                    Cancelar
                  </button>
                </motion.div>
              )}
            </AnimatePresence>

            {/* Skip */}
            <div style={{ textAlign: 'center', marginTop: '0.75rem' }}>
              <button className="ch-skip" onClick={() => skipChallenge('economy')}>
                Saltar evaluación
              </button>
            </div>
          </>
        )}

        {/* AI Analyzing State */}
        {isAnalyzing && (
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

        {/* Result */}
        <AnimatePresence>
          {showFeedback && (
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
                {isCorrect ? 'Decisión correcta' : 'Error táctico'}
              </h3>
              
              <div className="ch-result__explanation">
                <div className="ch-result__explanation-label">
                  <TrendingUp size={14} />
                  Análisis táctico
                </div>
                <p 
                  className="ch-result__explanation-text" 
                  dangerouslySetInnerHTML={{ __html: currentQuestion.options[selectedOption].explanation }} 
                />
              </div>

              {isCorrect && (
                <p className="ch-result__unlock">Módulo de Economía evaluado.</p>
              )}

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
                    {getObservationNote('economy', isCorrect, completedChallenges).label}
                  </span>
                </div>
                <p className="ch-observation__text">
                  {getObservationNote('economy', isCorrect, completedChallenges).note}
                </p>
              </motion.div>
            </motion.div>
          )}
        </AnimatePresence>
      </motion.div>
    </div>
  );
};

export default EconomyChallenge;
