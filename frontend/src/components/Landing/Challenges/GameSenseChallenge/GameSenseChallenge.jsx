// GameSenseChallenge - Interactive Video Decision Making (Side Layout)
import React, { useState, useRef, useEffect } from 'react';
import { useLanding } from '../../LandingContext';
import ChallengeWrapper from '../ChallengeWrapper';
import Button, { ArrowRightIcon } from '../../shared/Button';
import '../../../../styles/Landing/gamesense-challenge.css';

const GameSenseChallenge = () => {
  const { nextChallenge, saveResult } = useLanding();
  const videoRef = useRef(null);
  
  // State
  const [isPlaying, setIsPlaying] = useState(false);
  const [hasStarted, setHasStarted] = useState(false);
  const [showOptions, setShowOptions] = useState(false);
  const [videoEnded, setVideoEnded] = useState(false);
  const [selectedOption, setSelectedOption] = useState(null);
  const [showFeedback, setShowFeedback] = useState(false);
  
  // Constants
  const DECISION_POINT = 7;
  const CORRECT_OPTION = 3; 
  const VIDEO_SRC = "/videos/GameSense.mp4";

  const OPTIONS = [
    {
      id: 0,
      title: "Holdear ángulo pasivo",
      description: "Aguantar la posición y esperar a que los enemigos pickeen.",
      letter: "A"
    },
    {
      id: 1,
      title: "Duelo seco en Corta",
      description: "Intentar matar al jugador de corta pickeando sin utilidad.",
      letter: "B"
    },
    {
      id: 2,
      title: "Esconderse en el humo",
      description: "Usar el humo y meterse dentro para intentar sobrevivir.",
      letter: "C"
    },
    {
      id: 3,
      title: "Aislar 1vs1 con humo",
      description: "Usar el humo one-way para aislar un duelo y cortar visión.",
      letter: "D"
    }
  ];

  // Handle video time update
  const handleTimeUpdate = () => {
    if (!videoRef.current) return;
    
    const currentTime = videoRef.current.currentTime;
    
    // Pause at decision point if options haven't been shown yet and user hasn't selected
    if (currentTime >= DECISION_POINT && !showOptions && selectedOption === null) {
      videoRef.current.pause();
      setIsPlaying(false);
      setShowOptions(true);
    }

    // Check for end - Show feedback based on selection
    if (currentTime >= videoRef.current.duration - 0.5) {
      if (!videoEnded) {
        setVideoEnded(true);
        setIsPlaying(false);
        // If they selected something, feedback is shown via conditional rendering of videoEnded block
      }
    }
  };

  // Start challenge
  const handleStart = () => {
    if (videoRef.current) {
      videoRef.current.play();
      setIsPlaying(true);
      setHasStarted(true);
    }
  };

  // Handle option selection
  const handleOptionSelect = (index) => {
    setSelectedOption(index);
    setShowOptions(false); // Hide options immediately
    
    // Resume video immediately to see the outcome
    if (videoRef.current) {
      videoRef.current.play();
      setIsPlaying(true);
    }
  };

  const handleContinue = () => {
    saveResult('gamesense', {
      selectedOption,
      isCorrect: selectedOption === CORRECT_OPTION,
    });
    nextChallenge();
  };

  const handleRetry = () => {
    if (videoRef.current) {
      videoRef.current.currentTime = 0;
      videoRef.current.play();
      setIsPlaying(true);
      setShowOptions(false);
      setShowFeedback(false);
      setSelectedOption(null);
    }
  };

  const getOptionClass = (index) => {
    const classes = ['gamesense__side-option'];
    
    if (selectedOption === index) {
      classes.push('gamesense__side-option--selected');
    }
    
    if (showFeedback) {
      if (index === CORRECT_OPTION) {
        classes.push('gamesense__side-option--correct');
      } else if (selectedOption === index) {
        classes.push('gamesense__side-option--incorrect');
      } else {
        classes.push('gamesense__side-option--dimmed');
      }
    }
    
    return classes.join(' ');
  };

  return (
    <ChallengeWrapper
      type="GAMESENSE"
      title="Toma de Decisiones"
      subtitle="Analiza la situación y elige la jugada con mayor probabilidad de éxito."
      showTimer={false}
    >
      <div className="gamesense-3d-scene">
        
        {/* Context HUD - Only visible during decision or start */}
        <div className={`gamesense-hud ${showOptions || !hasStarted ? 'visible' : ''}`}>
           <div className="gamesense-hud__glass">
             <span className="gamesense-hud__label">SITUACIÓN TÁCTICA</span>
             <h3>1 vs 2 Retake - A Site</h3>
             <p>Eres Terrorista. Tienes <strong>1 Humo</strong>. Info: Enemigos en <strong>Larga</strong> y <strong>Corta</strong>.</p>
           </div>
        </div>

        <div className="gamesense-layout-3d">
          {/* Left Panel - 3D Rotated */}
          <div className={`gamesense-panel-3d gamesense-panel-3d--left ${showOptions ? 'visible' : ''}`}>
            {OPTIONS.slice(0, 2).map((option, i) => (
              <div 
                key={option.id}
                className={getOptionClass(i)}
                onClick={() => handleOptionSelect(i)}
              >
                <div className="gamesense-card-shine"></div>
                <div className="gamesense__side-header">
                   <span className="gamesense__side-letter">{option.letter}</span>
                   <h4>{option.title}</h4>
                </div>
                <p className="gamesense__side-desc">{option.description}</p>
              </div>
            ))}
          </div>

          {/* Center Video Frame */}
          <div className="gamesense-video-frame">
            <video
              ref={videoRef}
              src={VIDEO_SRC}
              className="gamesense-video-player"
              onTimeUpdate={handleTimeUpdate}
              playsInline
            />
            
            {!hasStarted && (
              <div className="gamesense-overlay gamesense-overlay--start" onClick={handleStart}>
                <div className="gamesense-play-glow">
                  <div className="gamesense-play-btn">
                    <PlayIcon />
                  </div>
                </div>
                <p>INICIAR SIMULACIÓN</p>
              </div>
            )}

            {/* Feedback Overlay - Only shown at the very end of video */}
            {videoEnded && (
              <div className="gamesense-overlay gamesense-overlay--success">
                {selectedOption === CORRECT_OPTION ? (
                  <div className="gamesense-feedback-card correct-3d">
                    <div className="feedback-icon-wrapper"><CheckIcon /></div>
                    <h4>LECTURA PERFECTA</h4>
                    <p>Aislar el 1v1 transforma una situación imposible en dos duelos ganables.</p>
                    <Button variant="primary" onClick={handleContinue} icon={<ArrowRightIcon />}>
                      CONTINUAR
                    </Button>
                  </div>
                ) : (
                  <div className="gamesense-feedback-card incorrect-3d">
                    <div className="feedback-icon-wrapper"><XIcon /></div>
                    <h4>DECISIÓN INCORRECTA</h4>
                    <p>
                      La opción escogida te dejaba expuesto. <br/>
                      <strong>La correcta era la D (Aislar 1vs1).</strong><br/>
                      Usar el humo permite anular un ángulo y pelear con ventaja.
                    </p>
                    <div className="gamesense-actions">
                      <button onClick={handleRetry}>REINTENTAR</button>
                      <button onClick={handleContinue} className="ghost">CONTINUAR</button>
                    </div>
                  </div>
                )}
              </div>
            )}
            
            {/* Screen Reflections/Scanlines overlay */}
            <div className="gamesense-screen-effects"></div>
          </div>

          {/* Right Panel - 3D Rotated */}
          <div className={`gamesense-panel-3d gamesense-panel-3d--right ${showOptions ? 'visible' : ''}`}>
             {OPTIONS.slice(2, 4).map((option, i) => {
              const realIndex = i + 2; 
              return (
                <div 
                  key={option.id}
                  className={getOptionClass(realIndex)}
                  onClick={() => handleOptionSelect(realIndex)}
                >
                  <div className="gamesense-card-shine"></div>
                  <div className="gamesense__side-header">
                     <span className="gamesense__side-letter">{option.letter}</span>
                     <h4>{option.title}</h4>
                  </div>
                  <p className="gamesense__side-desc">{option.description}</p>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </ChallengeWrapper>
  );
};

// Icons need to be redefined since I removed them in the overwrite? No, I'll keep them.
const PlayIcon = () => (
    <svg viewBox="0 0 24 24" fill="currentColor" width="24" height="24">
      <path d="M8 5v14l11-7z" />
    </svg>
  );
  
  const CheckIcon = () => (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" width="24" height="24">
      <polyline points="20 6 9 17 4 12" />
    </svg>
  );
  
  const XIcon = () => (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" width="24" height="24">
      <line x1="18" y1="6" x2="6" y2="18" />
      <line x1="6" y1="6" x2="18" y2="18" />
    </svg>
  );

export default GameSenseChallenge;
