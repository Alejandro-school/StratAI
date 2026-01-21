// GrenadeChallenge - Lineup identification test with 3D map cards
import React, { useState, useRef, useCallback } from 'react';
import { useLanding } from '../../LandingContext';
import ChallengeWrapper from '../ChallengeWrapper';
import Button, { ArrowRightIcon } from '../../shared/Button';
import '../../../../styles/Landing/grenade-challenge.css';

/**
 * GrenadeChallenge
 * 
 * Two-phase challenge:
 * 1. Select a map (3D hover cards)
 * 2. Watch 3 lineup videos and identify the correct one
 */

// Map data - using local images from public/images/maps/
const MAPS = [
  { 
    id: 'mirage', 
    name: 'Mirage', 
    image: '/images/maps/de_mirage.png',
    type: 'Active Duty'
  },
  { 
    id: 'inferno', 
    name: 'Inferno', 
    image: '/images/maps/de_inferno.png',
    type: 'Active Duty'
  },
  { 
    id: 'dust2', 
    name: 'Dust II', 
    image: '/images/maps/de_dust2.png',
    type: 'Active Duty'
  },
  { 
    id: 'anubis', 
    name: 'Anubis', 
    image: '/images/maps/de_anubis.png',
    type: 'Active Duty'
  },
  { 
    id: 'ancient', 
    name: 'Ancient', 
    image: '/images/maps/de_ancient.png',
    type: 'Active Duty'
  },
  { 
    id: 'nuke', 
    name: 'Nuke', 
    image: '/images/maps/de_nuke.png',
    type: 'Active Duty'
  },
];

// Lineup scenarios for each map
const LINEUP_SCENARIOS = {
  mirage: {
    title: 'Smoke Window desde T Spawn',
    description: 'Estás en <strong>T Spawn</strong> y necesitas lanzar un smoke a <strong>Window</strong> (ventana de sniper) para ejecutar hacia A. ¿Cuál de estos lineups es el correcto?',
    correctOption: 2, // 0-indexed
    options: [
      { label: 'A', description: 'Lineup desde la esquina izquierda' },
      { label: 'B', description: 'Lineup desde el carro' },
      { label: 'C', description: 'Lineup desde la caja de madera' },
    ],
    feedback: 'El lineup correcto es desde la caja de madera. Este smoke aterriza perfectamente en Window, bloqueando la visión del AWPer CT.'
  },
  inferno: {
    title: 'Molotov Coffins desde Banana',
    description: 'Estás entrando por <strong>Banana</strong> hacia B. Necesitas lanzar un molotov a <strong>Coffins</strong> (ataúdes) para despejar esa posición. ¿Cuál lineup es correcto?',
    correctOption: 0,
    options: [
      { label: 'A', description: 'Lineup desde la pared del fondo' },
      { label: 'B', description: 'Lineup corriendo' },
      { label: 'C', description: 'Lineup desde car' },
    ],
    feedback: 'El molotov desde la pared del fondo es el más consistente y cubre toda el área de coffins sin exponerte.'
  },
  dust2: {
    title: 'Smoke Xbox desde T Spawn',
    description: 'Quieres fumar <strong>Xbox</strong> (la caja en mid) para cruzar a short. ¿Cuál de estos lineups desde <strong>T Spawn</strong> es el correcto?',
    correctOption: 1,
    options: [
      { label: 'A', description: 'Lineup desde la puerta' },
      { label: 'B', description: 'Lineup desde el barril' },
      { label: 'C', description: 'Lineup saltando' },
    ],
    feedback: 'El lineup desde el barril es el estándar. Es consistente y te permite lanzar rápidamente al inicio de la ronda.'
  },
  anubis: {
    title: 'Flash Canal desde A Main',
    description: 'Estás ejecutando hacia <strong>A</strong> y necesitas flashear a los defensores en <strong>Canal</strong>. ¿Cuál flash es la más efectiva?',
    correctOption: 2,
    options: [
      { label: 'A', description: 'Pop flash por encima del muro' },
      { label: 'B', description: 'Flash rebotada en el suelo' },
      { label: 'C', description: 'Flash de bank hacia atrás' },
    ],
    feedback: 'La flash de bank (rebotada hacia atrás) es muy difícil de esquivar para los defensores y te da tiempo de entrar.'
  },
  ancient: {
    title: 'Smoke Donut desde T Spawn',
    description: 'Necesitas fumar <strong>Donut</strong> (la posición elevada en medio) para poder cruzar hacia A. ¿Cuál lineup es correcto?',
    correctOption: 0,
    options: [
      { label: 'A', description: 'Lineup alineado con la antena' },
      { label: 'B', description: 'Lineup desde la roca' },
      { label: 'C', description: 'Lineup corriendo hacia mid' },
    ],
    feedback: 'El lineup alineado con la antena del edificio es el más preciso y bloquea completamente la visión de Donut.'
  },
  nuke: {
    title: 'Smoke Heaven desde T Roof',
    description: 'Estás en el <strong>techo de T</strong> y quieres fumar <strong>Heaven</strong> (el balcón superior de A). ¿Cuál es el lineup correcto?',
    correctOption: 1,
    options: [
      { label: 'A', description: 'Lineup desde la ventilación' },
      { label: 'B', description: 'Lineup desde la esquina del silo' },
      { label: 'C', description: 'Lineup de carrera' },
    ],
    feedback: 'El lineup desde la esquina del silo es el más usado por profesionales. Bloquea Heaven perfectamente para ejecutar A.'
  },
};

// ═══════════════════════════════════════════════════════════════════════════
// 3D Card Component with Mouse Tracking
// ═══════════════════════════════════════════════════════════════════════════
const MapCard3D = ({ map, onClick }) => {
  const cardRef = useRef(null);
  const [transform, setTransform] = useState('');

  const handleMouseMove = useCallback((e) => {
    if (!cardRef.current) return;
    
    const rect = cardRef.current.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    
    const centerX = rect.width / 2;
    const centerY = rect.height / 2;
    
    const rotateX = (y - centerY) / 10;
    const rotateY = (centerX - x) / 10;
    
    setTransform(`perspective(1000px) rotateX(${rotateX}deg) rotateY(${rotateY}deg) scale3d(1.05, 1.05, 1.05)`);
  }, []);

  const handleMouseLeave = useCallback(() => {
    setTransform('');
  }, []);

  return (
    <div
      ref={cardRef}
      className="grenade-challenge__card"
      style={{ transform }}
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
      onClick={() => onClick(map.id)}
    >
      <img 
        src={map.image} 
        alt={map.name}
        className="grenade-challenge__card-image"
      />
      <div className="grenade-challenge__card-overlay" />
      <div className="grenade-challenge__card-content">
        <h3 className="grenade-challenge__card-name">{map.name}</h3>
        <span className="grenade-challenge__card-type">{map.type}</span>
      </div>
      <div className="grenade-challenge__card-shine" />
    </div>
  );
};

// ═══════════════════════════════════════════════════════════════════════════
// Main Component
// ═══════════════════════════════════════════════════════════════════════════
const GrenadeChallenge = () => {
  const { nextChallenge, saveResult } = useLanding();
  
  // State
  const [selectedMap, setSelectedMap] = useState(null);
  const [selectedOption, setSelectedOption] = useState(null);
  const [showFeedback, setShowFeedback] = useState(false);
  const [isCorrect, setIsCorrect] = useState(false);

  // Get current scenario
  const scenario = selectedMap ? LINEUP_SCENARIOS[selectedMap] : null;

  // Handle map selection
  const handleMapSelect = (mapId) => {
    setSelectedMap(mapId);
    setSelectedOption(null);
    setShowFeedback(false);
  };

  // Handle option selection
  const handleOptionSelect = (index) => {
    if (showFeedback) return;
    
    setSelectedOption(index);
    const correct = index === scenario.correctOption;
    setIsCorrect(correct);
    
    setTimeout(() => {
      setShowFeedback(true);
    }, 300);
  };

  // Handle back to map selection
  const handleBack = () => {
    setSelectedMap(null);
    setSelectedOption(null);
    setShowFeedback(false);
  };

  // Handle continue
  const handleContinue = () => {
    saveResult('grenade', {
      map: selectedMap,
      selectedOption,
      isCorrect,
    });
    
    nextChallenge();
  };

  // Get option class
  const getOptionClass = (index) => {
    const classes = ['grenade-challenge__option'];
    
    if (selectedOption === index) {
      classes.push('grenade-challenge__option--selected');
    }
    
    if (showFeedback) {
      if (index === scenario.correctOption) {
        classes.push('grenade-challenge__option--correct');
      } else if (selectedOption === index) {
        classes.push('grenade-challenge__option--incorrect');
      }
    }
    
    return classes.join(' ');
  };

  return (
    <ChallengeWrapper
      type="TÁCTICA"
      title={selectedMap ? null : 'Prueba de Granadas'}
      subtitle={selectedMap ? null : 'Selecciona un mapa para demostrar tu conocimiento de lineups.'}
      showTimer={false}
    >
      {/* Map Selection Grid */}
      {!selectedMap && (
        <div className="grenade-challenge__maps">
          {MAPS.map((map) => (
            <MapCard3D 
              key={map.id} 
              map={map} 
              onClick={handleMapSelect}
            />
          ))}
        </div>
      )}

      {/* Lineup Selection View */}
      {selectedMap && scenario && (
        <div className="grenade-challenge__lineup">
          {/* Header */}
          <div className="grenade-challenge__lineup-header">
            <div className="grenade-challenge__lineup-title">
              <span className="grenade-challenge__lineup-map">
                {MAPS.find(m => m.id === selectedMap)?.name}
              </span>
              <h3 className="grenade-challenge__lineup-question">{scenario.title}</h3>
            </div>
            <button className="grenade-challenge__back-btn" onClick={handleBack}>
              <BackIcon /> Cambiar mapa
            </button>
          </div>

          {/* Scenario Description */}
          <div className="grenade-challenge__scenario">
            <p 
              className="grenade-challenge__scenario-text"
              dangerouslySetInnerHTML={{ __html: scenario.description }}
            />
          </div>

          {/* Video Options */}
          <div className="grenade-challenge__options">
            {scenario.options.map((option, index) => (
              <div
                key={option.label}
                className={getOptionClass(index)}
                onClick={() => handleOptionSelect(index)}
              >
                <span className="grenade-challenge__option-label">{option.label}</span>
                
                {/* Video Placeholder */}
                <div className="grenade-challenge__video-placeholder">
                  <VideoIcon className="grenade-challenge__video-icon" />
                  <span className="grenade-challenge__video-label">{option.description}</span>
                </div>

                {/* Result Icon */}
                {showFeedback && (index === scenario.correctOption || selectedOption === index) && (
                  <div className={`grenade-challenge__result-icon grenade-challenge__result-icon--${index === scenario.correctOption ? 'correct' : 'incorrect'}`}>
                    {index === scenario.correctOption ? <CheckIcon /> : <XIcon />}
                  </div>
                )}
              </div>
            ))}
          </div>

          {/* Feedback */}
          {showFeedback && (
            <div className={`grenade-challenge__feedback grenade-challenge__feedback--${isCorrect ? 'correct' : 'incorrect'}`}>
              <h4 className="grenade-challenge__feedback-title">
                {isCorrect ? '¡Correcto!' : 'Incorrecto'}
              </h4>
              <p className="grenade-challenge__feedback-text">{scenario.feedback}</p>
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
      )}
    </ChallengeWrapper>
  );
};

// ═══════════════════════════════════════════════════════════════════════════
// Icons
// ═══════════════════════════════════════════════════════════════════════════
const VideoIcon = ({ className }) => (
  <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <polygon points="5 3 19 12 5 21 5 3" />
  </svg>
);

const BackIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <polyline points="15 18 9 12 15 6" />
  </svg>
);

const CheckIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
    <polyline points="20 6 9 17 4 12" />
  </svg>
);

const XIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
    <line x1="18" y1="6" x2="6" y2="18" />
    <line x1="6" y1="6" x2="18" y2="18" />
  </svg>
);

export default GrenadeChallenge;
