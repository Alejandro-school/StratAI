/**
 * ChatDemoSection - Interactive Replay + AI Chat Demo
 * 
 * Shows a real 2D replay with LIVE analysis:
 * - Pause at 1.3s: AI insights about Corta peek and B player positioning
 * - Pause at 6s: Critical errors analysis with player highlights  
 * - Video continues to end, then shows summary and questions
 */
import React, { useState, useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  MessageSquare, Bot, Sparkles, 
  ChevronRight, Zap, Target, RotateCcw,
  AlertTriangle, Eye
} from 'lucide-react';
import { useLanding } from '../../LandingContext';
import { API_URL } from '../../../../utils/api';
import '../../../../styles/Landing/sections/chatDemo.css';

// Imported Data & Components
import {
  FIRST_INSIGHT_MOMENT,
  CRITICAL_MOMENT,
  INSIGHT_DELAY_FIRST,
  INSIGHT_DELAY_CRITICAL,
  RESUME_DELAY,
  FIRST_INSIGHTS,
  CRITICAL_INSIGHTS,
  FINAL_SUMMARY,
  USER_QUESTIONS
} from './chatDemoData';

import ChatMessage from './ChatMessage';

const ChatDemoSection = () => {
  const { startChallenges } = useLanding();
  
  // Video state
  const videoRef = useRef(null);
  const [hasStarted, setHasStarted] = useState(false);
  const [isPlaying, setIsPlaying] = useState(false);
  const [highlightedPlayer, setHighlightedPlayer] = useState(null);
  
  // Analysis phases
  const [analysisPhase, setAnalysisPhase] = useState('initial'); 
  // 'initial' | 'first-insights' | 'playing' | 'critical-insights' | 'playing-final' | 'video-ended'
  
  const [currentInsightIndex, setCurrentInsightIndex] = useState(0);
  const [displayedInsights, setDisplayedInsights] = useState([]);
  const [criticalIndex, setCriticalIndex] = useState(0);
  
  // Question/answer state
  const [answeredQuestions, setAnsweredQuestions] = useState([]);
  
  const [hasInteracted, setHasInteracted] = useState(false);
  const chatContainerRef = useRef(null);

  // Scroll chat to bottom helper
  const scrollToBottom = React.useCallback(() => {
    if (chatContainerRef.current) {
      chatContainerRef.current.scrollTop = chatContainerRef.current.scrollHeight;
    }
  }, []);

  // Scroll chat to bottom when new messages (initial render of message)
  useEffect(() => {
    scrollToBottom();
  }, [displayedInsights, scrollToBottom]);

  // Monitor video time for pause points
  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    const handleTimeUpdate = () => {
      // First pause at 1.3s
      if (video.currentTime >= FIRST_INSIGHT_MOMENT && analysisPhase === 'playing' && currentInsightIndex === 0) {
        video.pause();
        setIsPlaying(false);
        setAnalysisPhase('first-insights');
        setHighlightedPlayer('corta');
      }
      
      // Critical pause at 6s
      if (video.currentTime >= CRITICAL_MOMENT && analysisPhase === 'playing') {
        video.pause();
        setIsPlaying(false);
        setAnalysisPhase('critical-insights');
        setCriticalIndex(0);
        setHighlightedPlayer('corta');
      }
    };

    const handleVideoEnded = () => {
      setIsPlaying(false);
      setAnalysisPhase('video-ended');
      setHighlightedPlayer(null);
      // Add final summary
      setDisplayedInsights(prev => [...prev, FINAL_SUMMARY]);
      setHasInteracted(true);
    };

    video.addEventListener('timeupdate', handleTimeUpdate);
    video.addEventListener('ended', handleVideoEnded);
    return () => {
      video.removeEventListener('timeupdate', handleTimeUpdate);
      video.removeEventListener('ended', handleVideoEnded);
    };
  }, [analysisPhase, currentInsightIndex]);

  // Display insights one by one during first-insights phase
  useEffect(() => {
    if (analysisPhase !== 'first-insights') return;
    
    if (currentInsightIndex >= FIRST_INSIGHTS.length) {
      // All insights shown, wait and resume video
      const resumeTimer = setTimeout(() => {
        setAnalysisPhase('playing');
        setHighlightedPlayer(null);
        if (videoRef.current) {
          videoRef.current.play();
          setIsPlaying(true);
        }
      }, RESUME_DELAY);
      return () => clearTimeout(resumeTimer);
    }

    const insight = FIRST_INSIGHTS[currentInsightIndex];
    setHighlightedPlayer(insight.player);
    
    const insightTimer = setTimeout(() => {
      setDisplayedInsights(prev => [...prev, insight]);
      
      const nextTimer = setTimeout(() => {
        setCurrentInsightIndex(prev => prev + 1);
      }, INSIGHT_DELAY_FIRST);
      
      return () => clearTimeout(nextTimer);
    }, currentInsightIndex === 0 ? 500 : 1500);

    return () => clearTimeout(insightTimer);
  }, [analysisPhase, currentInsightIndex]);

  // Display insights one by one during critical-insights phase
  useEffect(() => {
    if (analysisPhase !== 'critical-insights') return;
    
    if (criticalIndex >= CRITICAL_INSIGHTS.length) {
      // All critical insights shown, resume video to end
      const resumeTimer = setTimeout(() => {
        setAnalysisPhase('playing-final');
        setHighlightedPlayer(null);
        if (videoRef.current) {
          videoRef.current.play();
          setIsPlaying(true);
        }
      }, RESUME_DELAY);
      return () => clearTimeout(resumeTimer);
    }

    const insight = CRITICAL_INSIGHTS[criticalIndex];
    if (insight.player) {
      setHighlightedPlayer(insight.player);
    }
    
    const insightTimer = setTimeout(() => {
      setDisplayedInsights(prev => [...prev, insight]);
      
      const nextTimer = setTimeout(() => {
        setCriticalIndex(prev => prev + 1);
      }, INSIGHT_DELAY_CRITICAL);
      
      return () => clearTimeout(nextTimer);
    }, criticalIndex === 0 ? 500 : 1500);

    return () => clearTimeout(insightTimer);
  }, [analysisPhase, criticalIndex]);

  // Click to start video
  const handleVideoClick = () => {
    if (!videoRef.current) return;
    
    if (analysisPhase === 'first-insights' || analysisPhase === 'critical-insights') {
      return;
    }
    
    if (!hasStarted) {
      setHasStarted(true);
      setAnalysisPhase('playing');
      videoRef.current.play();
      setIsPlaying(true);
    } else if (analysisPhase === 'playing' || analysisPhase === 'playing-final') {
      if (isPlaying) {
        videoRef.current.pause();
        setIsPlaying(false);
      } else {
        videoRef.current.play();
        setIsPlaying(true);
      }
    }
  };

  // Restart video
  const handleRestart = () => {
    if (videoRef.current) {
      videoRef.current.currentTime = 0;
      videoRef.current.play();
      setHasStarted(true);
      setIsPlaying(true);
      setAnalysisPhase('playing');
      setHighlightedPlayer(null);
      setCurrentInsightIndex(0);
      setCriticalIndex(0);
      setDisplayedInsights([]);
      setAnsweredQuestions([]);
      setHasInteracted(false);
    }
  };

  // Handle question click
  const handleQuestionClick = (question) => {
    // Mark this question as answered
    setAnsweredQuestions(prev => [...prev, question.id]);
    
    // Add user question to chat
    setDisplayedInsights(prev => [...prev, { 
      id: `user-${question.id}-${Date.now()}`, 
      type: 'user-question',
      text: question.question,
      isUser: true
    }]);
    
    // Show answer after delay
    setTimeout(() => {
      setDisplayedInsights(prev => [...prev, { 
        id: `answer-${question.id}-${Date.now()}`, 
        type: 'solution',
        text: question.response
      }]);
      setHasInteracted(true);
    }, 1000);
  };

  // Get remaining unanswered questions
  const remainingQuestions = USER_QUESTIONS.filter(q => !answeredQuestions.includes(q.id));

  // CTA handlers
  const handleSteamLogin = () => {
    window.location.href = `${API_URL}/auth/steam/login`;
  };

  const handleStartChallenges = () => {
    startChallenges();
  };

  // Get current phase label
  const getPhaseLabel = () => {
    switch (analysisPhase) {
      case 'first-insights': return 'Análisis inicial...';
      case 'critical-insights': return 'Error detectado';
      case 'video-ended': return 'Ronda perdida';
      default: return null;
    }
  };

  return (
    <section className="chat-demo-section">
      <div className="chat-demo-section__container">
        
        {/* Header */}
        <motion.div 
          className="chat-demo-section__header"
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6 }}
        >
          <div className="header-badge">
            <Sparkles size={14} />
            <span>ANÁLISIS EN VIVO</span>
          </div>
          <h2>Mira cómo analiza la IA</h2>
          <p>Observa esta jugada real mientras la IA detecta errores tácticos</p>
        </motion.div>

        {/* Main Content: Replay + Chat Grid */}
        <motion.div 
          className="replay-chat-grid"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3, duration: 0.5 }}
        >
          {/* Left: Video Replay */}
          <div className="replay-panel">
            <div className="replay-header">
              <div className="replay-badge">
                <Target size={14} />
                <span>REPLAY TÁCTICO</span>
              </div>
              <div className="replay-context">
                <span className="context-map">MIRAGE</span>
                <span className="context-situation">4v4 • Ronda 12 • CT Side</span>
                <span className="context-player">Tú eres el jugador de Corta</span>
              </div>
            </div>

            <div 
              className={`replay-video-container ${!hasStarted ? 'replay-video-container--clickable' : ''} ${isPlaying ? '' : 'replay-video-container--paused'}`}
              onClick={handleVideoClick}
            >
              {/* Click to play prompt */}
              {!hasStarted && (
                <div className="replay-click-prompt">
                  <span>Haz clic para reproducir</span>
                </div>
              )}

              {/* Video element */}
              <video 
                ref={videoRef}
                className="replay-video"
                src="/videos/ChatIA.mp4"
                muted
                playsInline
              />

              {/* Player highlight overlays */}
              <AnimatePresence>
                {highlightedPlayer === 'corta' && (
                  <motion.div 
                    className="player-highlight player-highlight--corta"
                    initial={{ opacity: 0, scale: 0.8 }}
                    animate={{ opacity: 1, scale: 1 }}
                    exit={{ opacity: 0 }}
                  >
                    <div className="highlight-ring"></div>
                    <div className="highlight-label">
                      <Eye size={12} />
                      <span>TÚ</span>
                    </div>
                  </motion.div>
                )}
                {highlightedPlayer === 'b' && (
                  <motion.div 
                    className="player-highlight player-highlight--b"
                    initial={{ opacity: 0, scale: 0.8 }}
                    animate={{ opacity: 1, scale: 1 }}
                    exit={{ opacity: 0 }}
                  >
                    <div className="highlight-ring"></div>
                    <div className="highlight-label">
                      <Eye size={12} />
                      <span>COMPAÑERO</span>
                    </div>
                  </motion.div>
                )}
                {highlightedPlayer === 'b-critical' && (
                  <motion.div 
                    className="player-highlight player-highlight--b-critical"
                    initial={{ opacity: 0, scale: 0.8 }}
                    animate={{ opacity: 1, scale: 1 }}
                    exit={{ opacity: 0 }}
                  >
                    <div className="highlight-ring"></div>
                    <div className="highlight-label">
                      <Eye size={12} />
                      <span>COMPAÑERO</span>
                    </div>
                  </motion.div>
                )}
              </AnimatePresence>

              {/* Analysis overlay */}
              {(analysisPhase === 'first-insights' || analysisPhase === 'critical-insights') && (
                <motion.div 
                  className={`replay-analysis-overlay ${analysisPhase === 'critical-insights' ? 'replay-analysis-overlay--critical' : ''}`}
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                >
                  <div className={`analysis-badge ${analysisPhase === 'critical-insights' ? 'analysis-badge--error' : ''}`}>
                    {analysisPhase === 'critical-insights' ? <AlertTriangle size={16} /> : <Eye size={16} />}
                    <span>{getPhaseLabel()}</span>
                  </div>
                </motion.div>
              )}

              {/* Video ended overlay */}
              {analysisPhase === 'video-ended' && (
                <motion.div 
                  className="replay-analysis-overlay replay-analysis-overlay--ended"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                >
                  <div className="analysis-badge analysis-badge--ended">
                    <AlertTriangle size={16} />
                    <span>RONDA PERDIDA</span>
                  </div>
                </motion.div>
              )}

              {/* Pause indicator */}
              {hasStarted && !isPlaying && (analysisPhase === 'playing' || analysisPhase === 'playing-final') && (
                <div className="replay-pause-indicator">
                  <span>Pausado — clic para continuar</span>
                </div>
              )}
            </div>

            {/* Video controls */}
            <div className="replay-controls">
              <button 
                className="control-btn"
                onClick={handleRestart}
                disabled={!hasStarted}
              >
                <RotateCcw size={16} />
                <span>Reiniciar</span>
              </button>
              {(analysisPhase === 'first-insights' || analysisPhase === 'critical-insights') && (
                <span className="control-hint analyzing">Analizando...</span>
              )}
              {(analysisPhase === 'playing' || analysisPhase === 'playing-final') && isPlaying && (
                <span className="control-hint">Reproduciendo...</span>
              )}
              {analysisPhase === 'video-ended' && (
                <span className="control-hint error">Ronda finalizada</span>
              )}
            </div>
          </div>

          {/* Right: Chat Interface */}
          <div className="chat-panel">
            <div className="chat-header">
              <Bot size={18} />
              <span>Coach IA</span>
              {analysisPhase === 'first-insights' && (
                <span className="chat-status chat-status--analyzing">Analizando</span>
              )}
              {analysisPhase === 'critical-insights' && (
                <span className="chat-status chat-status--error">Error detectado</span>
              )}
              {analysisPhase === 'video-ended' && (
                <span className="chat-status chat-status--success">Análisis listo</span>
              )}
            </div>

            {/* Chat messages with scroll */}
            <div className="chat-messages" ref={chatContainerRef}>
              {/* Initial empty state */}
              {!hasStarted && (
                <div className="chat-empty-state">
                  <Target size={28} />
                  <p>Reproduce el video para ver<br/>el análisis en vivo</p>
                </div>
              )}

              {/* Playing state before first insights */}
              {hasStarted && (analysisPhase === 'playing' || analysisPhase === 'playing-final') && displayedInsights.length === 0 && (
                <div className="chat-empty-state">
                  <Eye size={28} />
                  <p>Analizando situación táctica...</p>
                </div>
              )}

              {/* Display all insights */}
              <AnimatePresence>
                {displayedInsights.map((insight) => (
                  <ChatMessage 
                    key={insight.id}
                    message={insight.text}
                    isUser={insight.isUser || false}
                    messageType={insight.type}
                    onProgress={scrollToBottom}
                  />
                ))}
              </AnimatePresence>
              
            </div>

            {/* Questions section - show after video ends */}
            {analysisPhase === 'video-ended' && remainingQuestions.length > 0 && (
              <motion.div 
                className="chat-questions-section"
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
              >
                <p className="questions-prompt">Hazme una pregunta:</p>
                <div className="questions-list">
                  {remainingQuestions.map((q) => (
                    <motion.button
                      key={q.id}
                      className="question-btn"
                      onClick={() => handleQuestionClick(q)}
                      whileHover={{ scale: 1.02 }}
                      whileTap={{ scale: 0.98 }}
                    >
                      <MessageSquare size={14} />
                      <span>{q.question}</span>
                    </motion.button>
                  ))}
                </div>
              </motion.div>
            )}
          </div>
        </motion.div>

        {/* Dual CTA Section */}
        <motion.div 
          className="chat-demo-section__cta-grid"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: hasInteracted ? 1 : 0.5, y: 0 }}
          transition={{ delay: 0.6, duration: 0.5 }}
        >
          <div className="cta-card cta-card--primary">
            <div className="cta-card__icon">
              <Zap size={24} />
            </div>
            <h3>¿Ya te convencí?</h3>
            <p>
              Conecta tu cuenta de Steam y desbloquea tu máximo nivel. 
              Analizaré tus partidas reales y te daré feedback personalizado.
            </p>
            <button className="cta-btn cta-btn--primary" onClick={handleSteamLogin}>
              <span>CONECTAR STEAM</span>
              <ChevronRight size={18} />
            </button>
          </div>

          <div className="cta-card cta-card--secondary">
            <div className="cta-card__icon">
              <Target size={24} />
            </div>
            <h3>¿Quieres ver más?</h3>
            <p>
              Realiza 3 pruebas tácticas rápidas para recibir un diagnóstico 
              de tu perfil y entender exactamente cómo puedo ayudarte.
            </p>
            <button className="cta-btn cta-btn--secondary" onClick={handleStartChallenges}>
              <span>HACER PRUEBAS</span>
              <ChevronRight size={18} />
            </button>
          </div>
        </motion.div>

      </div>
    </section>
  );
};

export default ChatDemoSection;
