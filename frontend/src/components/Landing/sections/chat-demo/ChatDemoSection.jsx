/**
 * ChatDemoSection - Interactive replay + AI chat demo.
 */
import React, { useMemo, useState, useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import {
  MessageSquare,
  Bot,
  Sparkles,
  Target,
  RotateCcw,
  AlertTriangle,
  Eye
} from 'lucide-react';
import { useLang } from '../../i18n/useLang';
import '../../../../styles/Landing/sections/chatDemo.css';

import {
  FIRST_INSIGHT_MOMENT,
  CRITICAL_MOMENT,
  INSIGHT_DELAY_FIRST,
  INSIGHT_DELAY_CRITICAL,
  RESUME_DELAY,
  CHAT_DEMO_SCRIPT
} from './chatDemoData';

import ChatMessage from './ChatMessage';

const ChatDemoSection = ({ isScrollPage = false }) => {
  const { lang, t } = useLang();
  const script = useMemo(() => CHAT_DEMO_SCRIPT[lang] || CHAT_DEMO_SCRIPT.es, [lang]);
  const labels = script.phaseLabels;

  const videoRef = useRef(null);
  const chatContainerRef = useRef(null);
  const addedInsightIds = useRef(new Set());

  const [hasStarted, setHasStarted] = useState(false);
  const [isPlaying, setIsPlaying] = useState(false);
  const [highlightedPlayer, setHighlightedPlayer] = useState(null);
  const [analysisPhase, setAnalysisPhase] = useState('initial');
  const [currentInsightIndex, setCurrentInsightIndex] = useState(0);
  const [displayedInsights, setDisplayedInsights] = useState([]);
  const [criticalIndex, setCriticalIndex] = useState(0);
  const [answeredQuestions, setAnsweredQuestions] = useState([]);

  useEffect(() => {
    if (videoRef.current) {
      videoRef.current.pause();
      videoRef.current.currentTime = 0;
    }

    setHasStarted(false);
    setIsPlaying(false);
    setHighlightedPlayer(null);
    setAnalysisPhase('initial');
    setCurrentInsightIndex(0);
    setDisplayedInsights([]);
    setCriticalIndex(0);
    setAnsweredQuestions([]);
    addedInsightIds.current.clear();
  }, [lang]);

  const scrollToBottom = React.useCallback(() => {
    if (chatContainerRef.current) {
      chatContainerRef.current.scrollTop = chatContainerRef.current.scrollHeight;
    }
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [displayedInsights, scrollToBottom]);

  useEffect(() => {
    const el = chatContainerRef.current;
    if (!el) return;

    const handleWheel = (e) => {
      const { scrollTop, scrollHeight, clientHeight } = el;
      const atTop = scrollTop <= 0 && e.deltaY < 0;
      const atBottom = scrollTop + clientHeight >= scrollHeight - 1 && e.deltaY > 0;

      if (!atTop && !atBottom) {
        e.preventDefault();
        e.stopPropagation();
        el.scrollTop += e.deltaY;
      }
    };

    el.addEventListener('wheel', handleWheel, { passive: false });
    return () => el.removeEventListener('wheel', handleWheel);
  }, []);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    const handleTimeUpdate = () => {
      if (video.currentTime >= FIRST_INSIGHT_MOMENT && analysisPhase === 'playing' && currentInsightIndex === 0) {
        video.pause();
        setIsPlaying(false);
        setAnalysisPhase('first-insights');
        setHighlightedPlayer('corta');
      }

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
      setDisplayedInsights((prev) => [...prev, script.finalSummary]);
    };

    video.addEventListener('timeupdate', handleTimeUpdate);
    video.addEventListener('ended', handleVideoEnded);

    return () => {
      video.removeEventListener('timeupdate', handleTimeUpdate);
      video.removeEventListener('ended', handleVideoEnded);
    };
  }, [analysisPhase, currentInsightIndex, script.finalSummary]);

  useEffect(() => {
    if (analysisPhase !== 'first-insights') return;

    if (currentInsightIndex >= script.firstInsights.length) {
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

    const insight = script.firstInsights[currentInsightIndex];
    setHighlightedPlayer(insight.player);

    const delay = currentInsightIndex === 0 ? 500 : 1500;
    const showTimer = setTimeout(() => {
      if (!addedInsightIds.current.has(insight.id)) {
        addedInsightIds.current.add(insight.id);
        setDisplayedInsights((prev) => [...prev, insight]);
      }
    }, delay);

    const advanceTimer = setTimeout(() => {
      setCurrentInsightIndex((prev) => prev + 1);
    }, delay + INSIGHT_DELAY_FIRST);

    return () => {
      clearTimeout(showTimer);
      clearTimeout(advanceTimer);
    };
  }, [analysisPhase, currentInsightIndex, script.firstInsights]);

  useEffect(() => {
    if (analysisPhase !== 'critical-insights') return;

    if (criticalIndex >= script.criticalInsights.length) {
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

    const insight = script.criticalInsights[criticalIndex];
    if (insight.player) {
      setHighlightedPlayer(insight.player);
    }

    const delay = criticalIndex === 0 ? 500 : 1500;
    const showTimer = setTimeout(() => {
      if (!addedInsightIds.current.has(insight.id)) {
        addedInsightIds.current.add(insight.id);
        setDisplayedInsights((prev) => [...prev, insight]);
      }
    }, delay);

    const advanceTimer = setTimeout(() => {
      setCriticalIndex((prev) => prev + 1);
    }, delay + INSIGHT_DELAY_CRITICAL);

    return () => {
      clearTimeout(showTimer);
      clearTimeout(advanceTimer);
    };
  }, [analysisPhase, criticalIndex, script.criticalInsights]);

  const handleVideoClick = () => {
    if (!videoRef.current) return;
    if (analysisPhase === 'first-insights' || analysisPhase === 'critical-insights') return;

    if (!hasStarted) {
      setHasStarted(true);
      setAnalysisPhase('playing');
      videoRef.current.play().catch((e) => console.error(e));
      setIsPlaying(true);
      return;
    }

    if (analysisPhase === 'playing' || analysisPhase === 'playing-final') {
      if (isPlaying) {
        videoRef.current.pause();
        setIsPlaying(false);
      } else {
        videoRef.current.play().catch((e) => console.error(e));
        setIsPlaying(true);
      }
    }
  };

  useEffect(() => {
    if (!isScrollPage || hasStarted) return;

    const currentContainer = chatContainerRef.current;
    if (!currentContainer) return;

    const observer = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting && !hasStarted && videoRef.current) {
          setHasStarted(true);
          setAnalysisPhase('playing');
          setIsPlaying(true);
          videoRef.current.play().catch((e) => {
            console.log('Autoplay prevented:', e);
            setIsPlaying(false);
            setAnalysisPhase('initial');
          });
        }
      });
    }, { threshold: 0.5 });

    observer.observe(currentContainer);

    return () => {
      observer.unobserve(currentContainer);
    };
  }, [hasStarted, isScrollPage]);

  const handleRestart = () => {
    if (!videoRef.current) return;

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
    addedInsightIds.current.clear();
  };

  const handleQuestionClick = (question) => {
    setAnsweredQuestions((prev) => [...prev, question.id]);
    setDisplayedInsights((prev) => [
      ...prev,
      {
        id: `user-${question.id}-${Date.now()}`,
        type: 'user-question',
        text: question.question,
        isUser: true
      }
    ]);

    setTimeout(() => {
      setDisplayedInsights((prev) => [
        ...prev,
        {
          id: `answer-${question.id}-${Date.now()}`,
          type: 'solution',
          text: question.response
        }
      ]);
    }, 1000);
  };

  const remainingQuestions = script.userQuestions.filter((q) => !answeredQuestions.includes(q.id));

  const getPhaseLabel = () => {
    switch (analysisPhase) {
      case 'first-insights':
        return labels.firstInsights;
      case 'critical-insights':
        return labels.criticalInsights;
      case 'video-ended':
        return labels.videoEnded;
      default:
        return null;
    }
  };

  const renderMultiline = (text) =>
    text.split('\n').map((line, index, lines) => (
      <React.Fragment key={`${line}-${index}`}>
        {line}
        {index < lines.length - 1 && <br />}
      </React.Fragment>
    ));

  return (
    <section className={`chat-demo-section ${isScrollPage ? 'chat-demo-section--scroll' : ''}`}>
      <div className="chat-demo-section__container">
        <motion.div
          className="chat-demo-section__header"
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6 }}
        >
          <div className="header-badge">
            <Sparkles size={14} />
            <span>{t('aiDemo.badge')}</span>
          </div>
          <h2>{t('aiDemo.title')}</h2>
          <p>{t('aiDemo.subtitle')}</p>
        </motion.div>

        <motion.div
          className="replay-chat-grid"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3, duration: 0.5 }}
        >
          <div className="replay-panel">
            <div className="replay-header">
              <div className="replay-badge">
                <Target size={14} />
                <span>{t('aiDemo.replayLabel')}</span>
              </div>
              <div className="replay-context">
                <span className="context-map">{t('aiDemo.map')}</span>
                <span className="context-situation">{t('aiDemo.situation')}</span>
                <span className="context-player">{t('aiDemo.playerContext')}</span>
              </div>
            </div>

            <div
              className={`replay-video-container ${!hasStarted ? 'replay-video-container--clickable' : ''} ${isPlaying ? '' : 'replay-video-container--paused'}`}
              onClick={handleVideoClick}
            >
              {!hasStarted && (
                <div className="replay-click-prompt">
                  <span>{t('aiDemo.clickToPlay')}</span>
                </div>
              )}

              <video
                ref={videoRef}
                className="replay-video"
                src="/videos/ChatIA.mp4"
                muted
                playsInline
              />

              <AnimatePresence>
                {highlightedPlayer === 'corta' && (
                  <motion.div
                    className="player-highlight player-highlight--corta"
                    initial={{ opacity: 0, scale: 0.8 }}
                    animate={{ opacity: 1, scale: 1 }}
                    exit={{ opacity: 0 }}
                  >
                    <div className="highlight-ring" />
                    <div className="highlight-label">
                      <Eye size={12} />
                      <span>{labels.you}</span>
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
                    <div className="highlight-ring" />
                    <div className="highlight-label">
                      <Eye size={12} />
                      <span>{labels.teammate}</span>
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
                    <div className="highlight-ring" />
                    <div className="highlight-label">
                      <Eye size={12} />
                      <span>{labels.teammate}</span>
                    </div>
                  </motion.div>
                )}
              </AnimatePresence>

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

              {analysisPhase === 'video-ended' && (
                <motion.div
                  className="replay-analysis-overlay replay-analysis-overlay--ended"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                >
                  <div className="analysis-badge analysis-badge--ended">
                    <AlertTriangle size={16} />
                    <span>{labels.roundLost}</span>
                  </div>
                </motion.div>
              )}

              {hasStarted && !isPlaying && (analysisPhase === 'playing' || analysisPhase === 'playing-final') && (
                <div className="replay-pause-indicator">
                  <span>{labels.paused}</span>
                </div>
              )}
            </div>

            <div className="replay-controls">
              <button className="control-btn" onClick={handleRestart} disabled={!hasStarted}>
                <RotateCcw size={16} />
                <span>{t('aiDemo.restart')}</span>
              </button>
              {(analysisPhase === 'first-insights' || analysisPhase === 'critical-insights') && (
                <span className="control-hint analyzing">{labels.analyzing}</span>
              )}
              {(analysisPhase === 'playing' || analysisPhase === 'playing-final') && isPlaying && (
                <span className="control-hint">{labels.playing}</span>
              )}
              {analysisPhase === 'video-ended' && (
                <span className="control-hint error">{labels.roundFinished}</span>
              )}
            </div>
          </div>

          <div className="chat-panel">
            <div className="chat-header">
              <Bot size={18} />
              <span>{labels.coach}</span>
              {analysisPhase === 'first-insights' && (
                <span className="chat-status chat-status--analyzing">{labels.statusAnalyzing}</span>
              )}
              {analysisPhase === 'critical-insights' && (
                <span className="chat-status chat-status--error">{labels.statusError}</span>
              )}
              {analysisPhase === 'video-ended' && (
                <span className="chat-status chat-status--success">{labels.statusSuccess}</span>
              )}
            </div>

            <div className="chat-messages" ref={chatContainerRef}>
              {!hasStarted && (
                <div className="chat-empty-state">
                  <Target size={28} />
                  <p>{renderMultiline(labels.emptyStart)}</p>
                </div>
              )}

              {hasStarted && (analysisPhase === 'playing' || analysisPhase === 'playing-final') && displayedInsights.length === 0 && (
                <div className="chat-empty-state">
                  <Eye size={28} />
                  <p>{labels.emptyAnalyzing}</p>
                </div>
              )}

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

            {analysisPhase === 'video-ended' && remainingQuestions.length > 0 && (
              <motion.div
                className="chat-questions-section"
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
              >
                <p className="questions-prompt">{labels.questionsPrompt}</p>
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
      </div>
    </section>
  );
};

export default ChatDemoSection;
