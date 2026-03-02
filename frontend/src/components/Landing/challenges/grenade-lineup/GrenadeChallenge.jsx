// GrenadeChallenge - Map Selection + Video-Based Lineup Assessment
// Clean assessment UI: select map → pick lineup → watch result
import React, { useState, useRef } from 'react';
import { useLanding } from '../../LandingContext';
import { motion, AnimatePresence } from 'framer-motion';
import { Check, X, MapPin, Play, ArrowLeft, TrendingUp, ClipboardList } from 'lucide-react';
import { getObservationNote } from '../observationNotes';
import '../../../../styles/Landing/challenges/grenadePremium.css';

// Lineup scenarios — all competitive CS2 maps with video paths
const SCENARIOS = [
  {
    id: 'dust2',
    mapName: 'Dust 2',
    image: '/images/maps/de_dust2.png',
    title: 'Smoke CT desde Medio',
    description: 'Tu equipo tiene <strong>control de Medio</strong> y quiere pushear hacia <strong>B por CT</strong>. Necesitas un smoke que tape la visión desde CT Spawn para que tu equipo pueda avanzar sin ser cazado. <strong>¿Qué lineup es el correcto?</strong>',
    correctFeedback: 'Este smoke es fundamental para ejecutar splits hacia B. Al tapar CT, eliminas el ángulo más peligroso y permites que tu equipo avance por medio consiguiendo mucho control de mapa.',
    incorrectFeedback: 'El lineup que elegiste no tapa correctamente CT Spawn — deja ángulos abiertos que permiten al enemigo cazar a tu equipo en el avance. La opción correcta corta completamente la visión.',
    correctOption: 0,
    videos: [
      '/videos/Granadas/Dust2_bueno.mp4',
      '/videos/Granadas/Dust2_malo1.mp4',
      '/videos/Granadas/Dust2_malo2.mp4'
    ]
  },
  {
    id: 'inferno',
    mapName: 'Inferno',
    image: '/images/maps/de_inferno.png',
    title: 'Molotov Triples desde Banana',
    description: 'Estás en <strong>Banana</strong> preparando la ejecución hacia B. El AWPer enemigo suele jugar en <strong>Triples</strong> y necesitas sacarlo de posición antes de entrar. <strong>¿Cuál molotov usas?</strong>',
    correctFeedback: 'El molotov a Triples es esencial para cualquier ejecución de B. Obliga al AWPer a reposicionarse, dándote una ventana de 5-7 segundos para tomar espacio sin riesgo de ser eliminado instantáneamente.',
    incorrectFeedback: 'Ese molotov no impacta Triples correctamente — el AWPer mantiene su posición y sigue controlando Banana. Sin sacarlo de su ángulo, la ejecución de B es un suicidio.',
    correctOption: 0,
    videos: [
      '/videos/Granadas/Inferno_bueno.mp4',
      '/videos/Granadas/Inferno_malo1.mp4',
      '/videos/Granadas/Inferno_malo2.mp4'
    ]
  },
  {
    id: 'anubis',
    mapName: 'Anubis',
    image: '/images/maps/de_anubis.png',
    title: 'Smoke Heaven desde Aguas',
    description: 'Tu equipo va a ejecutar <strong>A</strong>. Estás posicionado en <strong>Aguas</strong> y necesitas smokear <strong>Heaven</strong> para que el CT no pueda disparar a tus compañeros que entran por Main. <strong>¿Qué lineup eliges?</strong>',
    correctFeedback: 'Heaven es la posición más dominante de A en Anubis. Sin este smoke, el CT tiene un ángulo perfecto sobre toda la entrada. Aprender este lineup te garantiza ejecuciones limpias.',
    incorrectFeedback: 'Tu lineup no bloquea Heaven — el CT mantiene visión sobre Main y puede eliminar a tus compañeros antes de que tomen posición. Heaven sin smokear significa que la ejecución depende de ganar un aim duel desfavorable.',
    correctOption: 0,
    videos: [
      '/videos/Granadas/Anubis_bueno.mp4',
      '/videos/Granadas/Anubis_malo1.mp4',
      '/videos/Granadas/Anubis_malo2.mp4'
    ]
  },
  {
    id: 'mirage',
    mapName: 'Mirage',
    image: '/images/maps/de_mirage.png',
    title: 'Smoke CT desde T Spawn',
    description: 'Ronda de ejecución hacia A. Tu equipo necesita que <strong>CT Spawn</strong> esté completamente tapado antes de que los entrys crucen por <strong>Palacio</strong> y <strong>Escaleras</strong>. Estás en <strong>T Spawn</strong>. <strong>¿Cuál es el smoke correcto?</strong>',
    correctFeedback: 'El smoke de CT desde T es uno de los más importantes de Mirage. Permite que tus entrys crucen hacia el site sin ser vistos desde CT, facilitando el plant seguro y el control de los ángulos post-plant.',
    incorrectFeedback: 'Ese smoke no cubre CT Spawn completamente. Los CTs mantienen visión parcial sobre la entrada al site — tus entrys quedan expuestos al cruzar. El lineup correcto tapa todo el paso de CT.',
    correctOption: 0,
    videos: [
      '/videos/Granadas/Mirage_bueno.mp4',
      '/videos/Granadas/Mirage_malo1.mp4',
      '/videos/Granadas/Mirage_malo2.mp4'
    ]
  },
  {
    id: 'nuke',
    mapName: 'Nuke',
    image: '/images/maps/de_nuke.png',
    title: 'Smoke Main desde Outside',
    description: 'Tu equipo va a pushear por <strong>Outside</strong>. Siempre hay un CT jugando en <strong>Main</strong> mirando hacia fuera que puede cazarte. Necesitas un smoke que neutralice esa posición y corte las rotaciones de A hacia Outside. <strong>¿Qué lineup usas?</strong>',
    correctFeedback: 'Este smoke es clave para tomar Outside de forma segura. Neutralizas al CT de Main y además cortas las rotaciones desde A, dando a tu equipo el tiempo necesario para posicionarse antes de ejecutar.',
    incorrectFeedback: 'Tu smoke no neutraliza Main — el CT sigue teniendo visión y puede eliminarte al salir. Las rotaciones desde A tampoco quedan cortadas, lo que significa que el equipo enemigo puede reforzar rápido.',
    correctOption: 0,
    videos: [
      '/videos/Granadas/Nuke_bueno.mp4',
      '/videos/Granadas/Nuke_malo1.mp4',
      '/videos/Granadas/Nuke_malo2.mp4'
    ]
  },
  {
    id: 'ancient',
    mapName: 'Ancient',
    image: '/images/maps/de_ancient.png',
    title: 'Smoke Red desde Mid',
    description: 'Tu equipo quiere tomar <strong>control de Mid</strong>. Hay un AWPer o CT jugando en <strong>Red</strong> que domina el medio. Necesitas un smoke que anule esa posición para que tu equipo pueda avanzar seguro. <strong>¿Cuál lineup es el correcto?</strong>',
    correctFeedback: 'Red es la posición más peligrosa cuando intentas tomar Mid en Ancient. Con este smoke, neutralizas al AWPer y permites que tu equipo tome control del centro del mapa sin riesgo de ser eliminado.',
    incorrectFeedback: 'Ese lineup no cubre Red correctamente. El AWPer o CT mantiene su ángulo sobre Mid y puede eliminar a todo el equipo cuando intente cruzar. El smoke correcto anula completamente esa posición.',
    correctOption: 0,
    videos: [
      '/videos/Granadas/Ancient_bueno.mp4#t=0.05',
      '/videos/Granadas/Ancient_malo1.mp4',
      '/videos/Granadas/Ancient_malo_2.mp4#t=1'
    ]
  },
  {
    id: 'overpass',
    mapName: 'Overpass',
    image: '/images/maps/de_overpass.png',
    title: 'Smoke Monster desde CT Spawn',
    description: 'Eres <strong>CT</strong> y sospechas que los T van de <strong>rush a B</strong>. Tienes un smoke desde <strong>CT Spawn</strong> que puede caer en <strong>Monster</strong> antes de que lleguen los T. <strong>¿Cuál es el lineup correcto?</strong>',
    correctFeedback: 'Este smoke preventivo es devastador contra rushes de B. Al lanzarlo desde CT Spawn, el smoke aterriza justo cuando los Ts llegan a Monster, frenándolos en seco y dando tiempo a tu equipo para posicionarse.',
    incorrectFeedback: 'Ese smoke no llega a tiempo o cae fuera de Monster. Los Ts pasan sin obstáculo y tu equipo no tiene tiempo para posicionarse. El lineup correcto sincroniza perfecto con el timing del rush.',
    correctOption: 0,
    videos: [
      '/videos/Granadas/Overpass_bueno.mp4',
      '/videos/Granadas/Overpass_malo1.mp4',
      '/videos/Granadas/Overpass_malo2.mp4'
    ]
  }
];

const GrenadeChallenge = () => {
  const { completeChallenge, skipChallenge, completedChallenges } = useLanding();
  
  const [selectedMap, setSelectedMap] = useState(null);
  const [previewOption, setPreviewOption] = useState(null);
  const [confirmedOption, setConfirmedOption] = useState(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [showResult, setShowResult] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const videoRef = useRef(null);

  const scenario = selectedMap !== null ? SCENARIOS[selectedMap] : null;

  const handleBackToMaps = () => {
    setSelectedMap(null);
    setPreviewOption(null);
    setConfirmedOption(null);
    setIsPlaying(false);
    setShowResult(false);
    setIsSuccess(false);
  };

  const handleMapSelect = (index) => {
    if (selectedMap !== null) return;
    setSelectedMap(index);
  };

  const handleOptionClick = (index) => {
    if (confirmedOption !== null) return;
    setPreviewOption(index);
  };

  const handleCancelPreview = () => {
    setPreviewOption(null);
  };

  const handleConfirmSelection = () => {
    if (previewOption === null || !scenario) return;
    setConfirmedOption(previewOption);
    setPreviewOption(null);
    setIsPlaying(true);
  };

  const handleVideoEnd = () => {
    setIsPlaying(false);
    setIsAnalyzing(true);
    
    const success = confirmedOption === scenario.correctOption;
    
    // Brief "AI analyzing" state before revealing result
    setTimeout(() => {
      setIsSuccess(success);
      setShowResult(true);
      setIsAnalyzing(false);
      
      completeChallenge('grenade', {
        scenarioId: scenario.id,
        selectedOption: confirmedOption,
        isCorrect: success,
      }, success);
    }, 1200);
  };

  // ── MAP SELECTION ──────────────────────────────────────
  if (selectedMap === null) {
    return (
      <div className="grenade-challenge">
        <motion.div 
          className="grenade-challenge__map-selection"
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.4 }}
        >
          <h3 className="grenade-challenge__section-title">Selecciona un mapa</h3>

          <div className="grenade-challenge__maps-grid">
            {SCENARIOS.map((sc, index) => (
              <motion.div
                key={sc.id}
                className="grenade-challenge__map-card"
                onClick={() => handleMapSelect(index)}
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: index * 0.04 }}
              >
                <div className="grenade-challenge__map-image">
                  <img src={sc.image} alt={sc.mapName} />
                  <div className="grenade-challenge__map-image-overlay" />
                </div>
                <div className="grenade-challenge__map-info">
                  <h4>{sc.mapName}</h4>
                  <p>{sc.title}</p>
                </div>
              </motion.div>
            ))}
          </div>

          <div style={{ textAlign: 'center', marginTop: '1.5rem' }}>
            <button className="ch-skip" onClick={() => skipChallenge('grenade')}>
              Saltar evaluación
            </button>
          </div>
        </motion.div>
      </div>
    );
  }

  // ── QUESTION SCREEN ────────────────────────────────────
  return (
    <div className="grenade-challenge">
      <motion.div 
        className="grenade-challenge__lineup"
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4 }}
      >
        {/* Back button */}
        {!showResult && confirmedOption === null && (
          <button className="grenade-challenge__back" onClick={handleBackToMaps}>
            <ArrowLeft size={16} />
            Cambiar mapa
          </button>
        )}

        {/* Header */}
        <div className="grenade-challenge__header">
          <div className="grenade-challenge__map-tag">
            <MapPin size={14} />
            <span>{scenario.mapName}</span>
          </div>
          <h3 className="grenade-challenge__title">{scenario.title}</h3>
        </div>

        <p 
          className="grenade-challenge__description" 
          dangerouslySetInnerHTML={{ __html: scenario.description }} 
        />

        {/* Video Options */}
        {!showResult && (
          <div className="grenade-challenge__options">
            {scenario.videos.map((videoSrc, index) => (
              <motion.div
                key={index}
                className={`grenade-challenge__option ${
                  confirmedOption === index ? 'grenade-challenge__option--selected' : ''
                } ${confirmedOption !== null ? 'grenade-challenge__option--locked' : ''}`}
                onClick={() => handleOptionClick(index)}
                whileTap={confirmedOption === null ? { scale: 0.98 } : {}}
              >
                <span className="grenade-challenge__option-letter">
                  {String.fromCharCode(65 + index)}
                </span>
                <div className="grenade-challenge__thumbnail">
                  <video src={videoSrc} preload="metadata" muted />
                  <div className="grenade-challenge__thumbnail-overlay">
                    <Play size={28} />
                  </div>
                </div>
              </motion.div>
            ))}
          </div>
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
        {showResult && (
          <motion.div 
            className={`ch-result ${isSuccess ? 'ch-result--success' : 'ch-result--error'}`}
            initial={{ opacity: 0, scale: 0.96 }}
            animate={{ opacity: 1, scale: 1 }}
          >
            <div className="ch-result__icon">
              {isSuccess ? <Check size={24} /> : <X size={24} />}
            </div>
            <h3 className="ch-result__title">
              {isSuccess ? 'Lineup correcto' : 'Lineup incorrecto'}
            </h3>
            <p className="ch-result__subtitle">
              {isSuccess 
                ? 'Has identificado correctamente el lineup.' 
                : `La respuesta correcta era la opción ${String.fromCharCode(65 + scenario.correctOption)}.`}
            </p>
            <div className="ch-result__explanation">
              <div className="ch-result__explanation-label">
                <TrendingUp size={14} />
                Contexto táctico
              </div>
              <p className="ch-result__explanation-text">
                {isSuccess ? scenario.correctFeedback : scenario.incorrectFeedback}
              </p>
            </div>
            {isSuccess && (
              <p className="ch-result__unlock">Módulo de Utilidades evaluado.</p>
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
                  {getObservationNote('grenade', isSuccess, completedChallenges).label}
                </span>
              </div>
              <p className="ch-observation__text">
                {getObservationNote('grenade', isSuccess, completedChallenges).note}
              </p>
            </motion.div>
          </motion.div>
        )}
      </motion.div>

      {/* ── Preview Modal ────────────────────────────────── */}
      <AnimatePresence>
        {previewOption !== null && (
          <motion.div
            className="grenade-challenge__modal-backdrop"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={handleCancelPreview}
          >
            <motion.div
              className="grenade-challenge__modal"
              initial={{ scale: 0.92, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.92, opacity: 0 }}
              onClick={(e) => e.stopPropagation()}
            >
              <div className="grenade-challenge__modal-header">
                <h4>Opción {String.fromCharCode(65 + previewOption)}</h4>
                <p>¿Confirmas esta selección?</p>
              </div>
              
              <div className="grenade-challenge__modal-video">
                <video
                  src={scenario.videos[previewOption]}
                  preload="metadata"
                  muted
                />
                {/* Crosshair overlay */}
                <div 
                  className="grenade-challenge__crosshair-overlay" 
                  data-map={scenario.id} 
                  data-option={previewOption}
                >
                  <div className="grenade-challenge__crosshair">
                    <span className="grenade-challenge__crosshair-h"></span>
                    <span className="grenade-challenge__crosshair-v"></span>
                    <span className="grenade-challenge__crosshair-dot"></span>
                  </div>
                </div>
              </div>

              <div className="grenade-challenge__modal-actions">
                <button 
                  className="grenade-challenge__modal-btn grenade-challenge__modal-btn--cancel"
                  onClick={handleCancelPreview}
                >
                  <X size={18} />
                  Cancelar
                </button>
                <button 
                  className="grenade-challenge__modal-btn grenade-challenge__modal-btn--confirm"
                  onClick={handleConfirmSelection}
                >
                  <Check size={18} />
                  Confirmar
                </button>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* ── Playing Video Modal ───────────────────────────── */}
      <AnimatePresence>
        {isPlaying && confirmedOption !== null && (
          <motion.div
            className="grenade-challenge__modal-backdrop"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          >
            <motion.div
              className="grenade-challenge__modal grenade-challenge__modal--playing"
              initial={{ scale: 0.92, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.92, opacity: 0 }}
            >
              <div className="grenade-challenge__modal-header">
                <h4>Reproduciendo Lineup {String.fromCharCode(65 + confirmedOption)}</h4>
              </div>
              <div className="grenade-challenge__modal-video grenade-challenge__modal-video--playing">
                <video
                  ref={videoRef}
                  src={scenario.videos[confirmedOption]}
                  autoPlay
                  onEnded={handleVideoEnd}
                />
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

export default GrenadeChallenge;
