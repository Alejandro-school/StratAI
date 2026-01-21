/**
 * AimTrainingHUD - Glassmorphic HUD overlay for aim training
 * 
 * Features:
 * - Real-time stats with GSAP counter animations
 * - Countdown display
 * - Progress indicator
 * - Results screen
 * - Rajdhani font styling
 */
import React, { useRef, useEffect, useCallback } from 'react';
import gsap from 'gsap';
import useAimTrainingStore, { TEST_STATES, AIM_CONFIG } from '../../../../hooks/useAimTrainingStore';
import { useLanding } from '../../LandingContext';
import Button, { ArrowRightIcon } from '../../shared/Button';
import './AimTrainingHUD.css';

// Animated Counter Component
const AnimatedCounter = ({ value, suffix = '', prefix = '' }) => {
  const counterRef = useRef(null);
  const prevValue = useRef(value);
  
  useEffect(() => {
    if (!counterRef.current) return;
    
    const obj = { value: prevValue.current };
    
    gsap.to(obj, {
      value: value,
      duration: 0.4,
      ease: 'power2.out',
      onUpdate: () => {
        if (counterRef.current) {
          counterRef.current.textContent = `${prefix}${Math.round(obj.value)}${suffix}`;
        }
      },
    });
    
    prevValue.current = value;
  }, [value, suffix, prefix]);
  
  return <span ref={counterRef}>{prefix}{value}{suffix}</span>;
};

// Stat Card Component
const StatCard = ({ label, value, suffix = '', icon, highlight = false }) => {
  return (
    <div className={`aim-hud__stat ${highlight ? 'aim-hud__stat--highlight' : ''}`}>
      {icon && <div className="aim-hud__stat-icon">{icon}</div>}
      <div className="aim-hud__stat-content">
        <span className="aim-hud__stat-value">
          <AnimatedCounter value={value} suffix={suffix} />
        </span>
        <span className="aim-hud__stat-label">{label}</span>
      </div>
    </div>
  );
};

// Progress Bar Component
const ProgressBar = ({ current, total }) => {
  const progress = (current / total) * 100;
  
  return (
    <div className="aim-hud__progress">
      <div className="aim-hud__progress-bar">
        <div 
          className="aim-hud__progress-fill" 
          style={{ width: `${progress}%` }}
        />
      </div>
      <span className="aim-hud__progress-text">{current}/{total}</span>
    </div>
  );
};

// Start Screen
const StartScreen = ({ onStart }) => {
  const containerRef = useRef(null);
  
  useEffect(() => {
    if (!containerRef.current) return;
    
    // Animate container scale-up (fly-in effect) matching warp duration
    // "expo.in" creates the effect of approaching a distant object (slow growth then fast fill)
    gsap.fromTo(containerRef.current,
      { opacity: 0, scale: 0.05 }, // Start as tiny distant dot
      { opacity: 1, scale: 1, duration: 3.2, ease: 'expo.in' }
    );
    
    // Animate children content
    gsap.fromTo(containerRef.current.children, 
      { opacity: 0, y: 30 },
      { 
        opacity: 1, 
        y: 0, 
        duration: 0.8, 
        delay: 0.3,
        stagger: 0.1,
        ease: 'power3.out'
      }
    );
  }, []);
  
  const handleStartClick = (e) => {
    console.log('🚀 Start button clicked!', { onStart, event: e });
    e.stopPropagation();
    if (onStart) {
      onStart();
    }
  };
  
  return (
    <div className="aim-hud__overlay aim-hud__start" ref={containerRef}>
      <h2 className="aim-hud__title">ENTRENAMIENTO DE AIM</h2>
      <p className="aim-hud__subtitle">
        Elimina los hologramas enemigos lo más rápido posible.
        <br />
        Tu precisión, tiempo de reacción y patrón de movimiento serán analizados.
      </p>
      <div className="aim-hud__instructions">
        <div className="aim-hud__instruction">
          <span className="aim-hud__instruction-icon">🎯</span>
          <span>Haz clic en los objetivos holográficos</span>
        </div>
        <div className="aim-hud__instruction">
          <span className="aim-hud__instruction-icon">⚡</span>
          <span>Reacciona rápido - cada objetivo desaparece</span>
        </div>
        <div className="aim-hud__instruction">
          <span className="aim-hud__instruction-icon">🧠</span>
          <span>Tu mouse pathing será evaluado por IA</span>
        </div>
      </div>
      <Button variant="primary" onClick={handleStartClick}>
        INICIAR PRUEBA
      </Button>
    </div>
  );
};

// Countdown Screen
const CountdownScreen = ({ count }) => {
  const numberRef = useRef(null);
  
  useEffect(() => {
    if (!numberRef.current) return;
    
    gsap.fromTo(numberRef.current,
      { scale: 0.5, opacity: 0 },
      { 
        scale: 1, 
        opacity: 1, 
        duration: 0.3,
        ease: 'back.out(2)'
      }
    );
    
    gsap.to(numberRef.current, {
      scale: 1.2,
      opacity: 0,
      duration: 0.7,
      delay: 0.3,
      ease: 'power2.in'
    });
  }, [count]);
  
  return (
    <div className="aim-hud__overlay aim-hud__countdown">
      <div className="aim-hud__countdown-text">PREPÁRATE</div>
      <div className="aim-hud__countdown-number" ref={numberRef}>
        {count}
      </div>
    </div>
  );
};

// Results Screen
const ResultsScreen = ({ onContinue }) => {
  const containerRef = useRef(null);
  const { kills, getAccuracy, getAverageRTT, getResults } = useAimTrainingStore();
  const { saveResult } = useLanding();
  
  const accuracy = getAccuracy();
  const avgTTK = getAverageRTT();
  
  useEffect(() => {
    if (!containerRef.current) return;
    
    gsap.fromTo(containerRef.current.children,
      { opacity: 0, y: 40 },
      {
        opacity: 1,
        y: 0,
        duration: 0.8,
        stagger: 0.15,
        ease: 'power3.out'
      }
    );
  }, []);
  
  const handleContinue = useCallback(() => {
    // Save results to context
    saveResult('aim', getResults());
    if (onContinue) onContinue();
  }, [saveResult, getResults, onContinue]);
  
  // Performance rating
  const getRating = () => {
    if (accuracy >= 90 && avgTTK < 400) return { text: 'ÉLITE', class: 'elite' };
    if (accuracy >= 75 && avgTTK < 500) return { text: 'EXPERTO', class: 'expert' };
    if (accuracy >= 60 && avgTTK < 600) return { text: 'AVANZADO', class: 'advanced' };
    if (accuracy >= 40) return { text: 'INTERMEDIO', class: 'intermediate' };
    return { text: 'NOVATO', class: 'novice' };
  };
  
  const rating = getRating();
  
  return (
    <div className="aim-hud__overlay aim-hud__results" ref={containerRef}>
      <h2 className="aim-hud__title">PRUEBA COMPLETADA</h2>
      
      <div className={`aim-hud__rating aim-hud__rating--${rating.class}`}>
        {rating.text}
      </div>
      
      <div className="aim-hud__results-grid">
        <div className="aim-hud__result-card">
          <span className="aim-hud__result-value">{kills}/{AIM_CONFIG.TOTAL_TARGETS}</span>
          <span className="aim-hud__result-label">Eliminaciones</span>
        </div>
        <div className="aim-hud__result-card">
          <span className="aim-hud__result-value">{accuracy}%</span>
          <span className="aim-hud__result-label">Precisión</span>
        </div>
        <div className="aim-hud__result-card">
          <span className="aim-hud__result-value">{avgTTK}ms</span>
          <span className="aim-hud__result-label">TTK Promedio</span>
        </div>
      </div>
      
      <Button 
        variant="primary" 
        icon={<ArrowRightIcon />} 
        onClick={handleContinue}
      >
        Continuar
      </Button>
    </div>
  );
};

// Main HUD Component
const AimTrainingHUD = () => {
  const { 
    testState, 
    countdown,
    kills, 
    targetsSpawned,
    startCountdown,
    decrementCountdown,
    getAccuracy,
    getAverageRTT,
  } = useAimTrainingStore();
  
  const { nextChallenge } = useLanding();
  
  // Countdown timer effect
  useEffect(() => {
    if (testState !== TEST_STATES.COUNTDOWN) return;
    
    const timer = setInterval(() => {
      decrementCountdown();
    }, 1000);
    
    return () => clearInterval(timer);
  }, [testState, decrementCountdown]);
  
  return (
    <div className="aim-hud">
      {/* Start Screen */}
      {testState === TEST_STATES.START && (
        <StartScreen onStart={startCountdown} />
      )}
      
      {/* Countdown */}
      {testState === TEST_STATES.COUNTDOWN && (
        <CountdownScreen count={countdown} />
      )}
      
      {/* Playing State HUD */}
      {testState === TEST_STATES.PLAYING && (
        <div className="aim-hud__playing">
          {/* Top Stats Bar */}
          <div className="aim-hud__top-bar">
            <StatCard 
              label="Eliminaciones" 
              value={kills} 
              icon={<TargetIcon />}
              highlight
            />
            <StatCard 
              label="Precisión" 
              value={getAccuracy()} 
              suffix="%"
              icon={<AccuracyIcon />}
            />
            <StatCard 
              label="TTK Avg" 
              value={getAverageRTT()} 
              suffix="ms"
              icon={<TimerIcon />}
            />
          </div>
          
          {/* Progress */}
          <div className="aim-hud__bottom-bar">
            <ProgressBar current={targetsSpawned} total={AIM_CONFIG.TOTAL_TARGETS} />
          </div>
        </div>
      )}
      
      {/* Results Screen */}
      {testState === TEST_STATES.RESULTS && (
        <ResultsScreen onContinue={nextChallenge} />
      )}
    </div>
  );
};

// Icons
const TargetIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="12" cy="12" r="10" />
    <circle cx="12" cy="12" r="6" />
    <circle cx="12" cy="12" r="2" />
  </svg>
);

const AccuracyIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
    <polyline points="22 4 12 14.01 9 11.01" />
  </svg>
);

const TimerIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="12" cy="12" r="10" />
    <polyline points="12 6 12 12 16 14" />
  </svg>
);

export default AimTrainingHUD;
