// EconomyChallenge - Tactical buy decision test
import React, { useState, useEffect, useMemo } from 'react';
import { useLanding } from '../../LandingContext';
import ChallengeWrapper from '../ChallengeWrapper';
import Button, { ArrowRightIcon } from '../../shared/Button';
import '../../../../styles/Landing/economy-challenge.css';

/**
 * EconomyChallenge
 * 
 * Tests the player's understanding of CS2 economy management.
 * One random question is selected from the pool.
 */

// Question pool
const QUESTIONS = [
  {
    id: 1,
    map: 'Inferno',
    side: 'CT',
    round: 9,
    score: { ct: 4, t: 5 },
    context: `Tu equipo lleva <strong>4 rondas ganadas</strong> y el enemigo <strong>5</strong>. 
              El enemigo ha ganado la ronda anterior y tiene <span class="highlight-danger">Full Buy</span>. 
              Todos tenéis alrededor de <span class="highlight-money">$3,000</span> y un bonus de derrota de <span class="highlight-money">$1,900</span>.`,
    money: '$3,000',
    lossBonus: '$1,900',
    question: '¿Cuál es la opción más eficiente?',
    options: [
      { letter: 'A', text: 'Comprar todos y forzar la ronda' },
      { letter: 'B', text: 'Hacer una eco completa' },
      { letter: 'C', text: 'Hacer una semi-forzada hasta $2,000' },
      { letter: 'D', text: 'Tú compras y el resto hace eco' },
    ],
    correctAnswer: 'B',
    explanation: `En la ronda 10 tendréis <strong>$5,400</strong> ($3,000 + $2,400 de bonus). Esto garantiza una compra completa con utilidades y kits de desactivación, que son vitales en Inferno para frenar ejecuciones en B o controlar medio.`,
  },
  {
    id: 2,
    map: 'Mirage',
    side: 'CT',
    round: 4,
    score: { ct: 1, t: 2 },
    context: `Marcador <strong>1-2</strong>. El enemigo viene de ganar y tiene <span class="highlight-danger">Full Buy</span>. 
              Tienes <span class="highlight-money">$3,500</span> y vuestro bonus de derrota es de <span class="highlight-money">$2,400</span>.`,
    money: '$3,500',
    lossBonus: '$2,400',
    question: '¿Qué estrategia de compra maximiza tu impacto?',
    options: [
      { letter: 'A', text: 'Comprar M4 y chaleco (sin granadas)' },
      { letter: 'B', text: 'Comprar SMG o FAMAS, utilidad completa (humos/molotov) y Kit' },
      { letter: 'C', text: 'Eco total para comprar AWP en la siguiente' },
      { letter: 'D', text: 'Comprar solo chaleco y HE grenades' },
    ],
    correctAnswer: 'B',
    explanation: `En Mirage, un CT sin humos no puede frenar un rush. Es más eficiente usar un arma inferior pero tener el <strong>control táctico del mapa</strong>. Si pierdes, el bonus de $2,900 te permite volver a comprar rápidamente.`,
  },
  {
    id: 3,
    map: 'Anubis',
    side: 'CT',
    round: 2,
    score: { ct: 1, t: 0 },
    context: `Ganasteis la ronda de pistolas y <span class="highlight-warning">el enemigo no plantó</span>. 
              Tienes <span class="highlight-money">$3,200</span>. El enemigo está eco (solo pistolas y sin armadura).`,
    money: '$3,200',
    lossBonus: 'N/A',
    question: '¿Cuál es la compra óptima para maximizar tu economía?',
    options: [
      { letter: 'A', text: 'Comprar M4A1-S para asegurar los duelos a distancia' },
      { letter: 'B', text: 'Comprar MP9 para farmear dinero por baja' },
      { letter: 'C', text: 'No comprar nada y jugar con la USP de la ronda 1' },
      { letter: 'D', text: 'Comprar solo armadura y esperar al M4 en la siguiente' },
    ],
    correctAnswer: 'B',
    explanation: `La MP9 da <strong>$600 por baja</strong> (el doble que un rifle). Como ellos están pobres y no tienen armadura, los subfusiles son letales y dispararán vuestra economía para el resto de la mitad.`,
  },
  {
    id: 4,
    map: 'Inferno',
    side: 'CT',
    round: 15,
    score: { ct: 7, t: 7 },
    context: `Situación <span class="highlight-danger">1 vs 4</span> en el sitio de B. Tienes una <strong>AWP</strong> y <span class="highlight-money">$0</span> en el banco. 
              Si mueres, cobras <span class="highlight-money">$2,900</span> de bonus de derrota.`,
    money: '$0',
    lossBonus: '$2,900',
    question: '¿Qué decisión aporta más valor a tu equipo?',
    options: [
      { letter: 'A', text: 'Intentar el retake heroico' },
      { letter: 'B', text: 'Salvar la AWP y buscar exit kills' },
      { letter: 'C', text: 'Tirar el AWP y morir para cobrar el bonus de ronda' },
      { letter: 'D', text: 'Esconderte y esperar que expire el tiempo' },
    ],
    correctAnswer: 'B',
    explanation: `Una AWP vale <strong>$4,750</strong>. Salvarla es mucho más rentable que morir solo por el bonus de $2,900. Además, mantienes la presión psicológica y ahorras casi $5,000 de presupuesto al equipo.`,
  },
];

const EconomyChallenge = () => {
  const { nextChallenge, saveResult } = useLanding();
  
  // Select random question on mount
  const question = useMemo(() => {
    const randomIndex = Math.floor(Math.random() * QUESTIONS.length);
    return QUESTIONS[randomIndex];
  }, []);
  
  // State
  const [selectedAnswer, setSelectedAnswer] = useState(null);
  const [showFeedback, setShowFeedback] = useState(false);
  const [isCorrect, setIsCorrect] = useState(false);

  // Handle option selection
  const handleOptionClick = (letter) => {
    if (showFeedback) return; // Already answered
    
    setSelectedAnswer(letter);
    setIsCorrect(letter === question.correctAnswer);
    
    // Show feedback after a brief delay
    setTimeout(() => {
      setShowFeedback(true);
    }, 300);
  };

  // Handle continue
  const handleContinue = () => {
    saveResult('economy', {
      questionId: question.id,
      selectedAnswer,
      isCorrect,
      map: question.map,
    });
    
    nextChallenge();
  };

  // Get option class
  const getOptionClass = (letter) => {
    const classes = ['economy-challenge__option'];
    
    if (selectedAnswer === letter) {
      classes.push('economy-challenge__option--selected');
    }
    
    if (showFeedback) {
      if (letter === question.correctAnswer) {
        classes.push('economy-challenge__option--correct');
      } else if (selectedAnswer === letter) {
        classes.push('economy-challenge__option--incorrect');
      }
    }
    
    return classes.join(' ');
  };

  return (
    <ChallengeWrapper
      type="INTELECTO"
      title="Prueba de Economía"
      subtitle="La gestión del dinero separa a los buenos jugadores de los grandes."
      showTimer={false}
    >
      <div className="economy-challenge__scenario">
        {/* Header with map and score */}
        <div className="economy-challenge__scenario-header">
          <div className="economy-challenge__map-info">
            <span className={`economy-challenge__map-badge economy-challenge__map-badge--${question.side.toLowerCase()}`}>
              {question.side} Side
            </span>
            <span className="economy-challenge__map-badge">
              {question.map}
            </span>
          </div>
          
          <div className="economy-challenge__round-info">
            <div className="economy-challenge__score">
              <span className="economy-challenge__score-ct">{question.score.ct}</span>
              <span className="economy-challenge__score-divider">-</span>
              <span className="economy-challenge__score-t">{question.score.t}</span>
            </div>
            <span className="economy-challenge__round-number">Ronda {question.round}</span>
          </div>
        </div>
        
        {/* Content */}
        <div className="economy-challenge__scenario-content">
          {/* Context */}
          <p 
            className="economy-challenge__context"
            dangerouslySetInnerHTML={{ __html: question.context }}
          />
          
          {/* Economy Stats */}
          <div className="economy-challenge__stats">
            <div className="economy-challenge__stat">
              <span className="economy-challenge__stat-value">{question.money}</span>
              <span className="economy-challenge__stat-label">Tu Dinero</span>
            </div>
            <div className="economy-challenge__stat">
              <span className="economy-challenge__stat-value">{question.lossBonus}</span>
              <span className="economy-challenge__stat-label">Bonus Derrota</span>
            </div>
          </div>
          
          {/* Question */}
          <h3 className="economy-challenge__question">{question.question}</h3>
          
          {/* Options */}
          <div className="economy-challenge__options">
            {question.options.map((option) => (
              <button
                key={option.letter}
                className={getOptionClass(option.letter)}
                onClick={() => handleOptionClick(option.letter)}
                disabled={showFeedback}
              >
                <span className="economy-challenge__option-letter">{option.letter}</span>
                <span className="economy-challenge__option-text">{option.text}</span>
              </button>
            ))}
          </div>
          
          {/* Feedback */}
          {showFeedback && (
            <div className={`economy-challenge__feedback economy-challenge__feedback--${isCorrect ? 'correct' : 'incorrect'}`}>
              <div className="economy-challenge__feedback-title">
                {isCorrect ? (
                  <>
                    <CheckIcon /> ¡Correcto!
                  </>
                ) : (
                  <>
                    <XIcon /> Incorrecto
                  </>
                )}
              </div>
              <p 
                className="economy-challenge__feedback-text"
                dangerouslySetInnerHTML={{ __html: question.explanation }}
              />
            </div>
          )}
          
          {/* Continue Button */}
          {showFeedback && (
            <div className="economy-challenge__continue">
              <Button
                variant="primary"
                icon={<ArrowRightIcon />}
                onClick={handleContinue}
              >
                Continuar
              </Button>
            </div>
          )}
        </div>
      </div>
    </ChallengeWrapper>
  );
};

// Icons
const CheckIcon = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
    <polyline points="20 6 9 17 4 12" />
  </svg>
);

const XIcon = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
    <line x1="18" y1="6" x2="6" y2="18" />
    <line x1="6" y1="6" x2="18" y2="18" />
  </svg>
);

export default EconomyChallenge;
