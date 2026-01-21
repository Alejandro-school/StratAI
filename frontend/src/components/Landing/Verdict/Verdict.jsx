// Verdict - Final AI feedback and CTA
import React, { useMemo } from 'react';
import { useLanding } from '../LandingContext';
import '../../../styles/Landing/verdict.css';

/**
 * Verdict
 * 
 * Final screen showing:
 * - Summary statistics from all challenges
 * - AI analysis with personalized feedback
 * - Steam login CTA
 */

const Verdict = () => {
  const { results, skipped, nickname } = useLanding();

  // Calculate overall stats
  const stats = useMemo(() => {
    // Aim stats
    const aimResult = results.aim || {};
    const accuracy = aimResult.accuracy || 0;
    const reactionTime = aimResult.avgReactionTime || 0;
    
    // Count correct answers
    let correctAnswers = 0;
    let totalQuestions = 0;
    
    if (results.economy) {
      totalQuestions++;
      if (results.economy.isCorrect) correctAnswers++;
    }
    if (results.grenade) {
      totalQuestions++;
      if (results.grenade.isCorrect) correctAnswers++;
    }
    if (results.gamesense) {
      totalQuestions++;
      if (results.gamesense.isCorrect) correctAnswers++;
    }
    
    const knowledgeScore = totalQuestions > 0 
      ? Math.round((correctAnswers / totalQuestions) * 100) 
      : 0;
    
    // Overall score (weighted average)
    const overallScore = Math.round(
      (accuracy * 0.3) + 
      (knowledgeScore * 0.5) + 
      (reactionTime < 300 ? 100 : reactionTime < 400 ? 80 : reactionTime < 500 ? 60 : 40) * 0.2
    );
    
    return {
      accuracy,
      reactionTime,
      knowledgeScore,
      correctAnswers,
      totalQuestions,
      overallScore,
    };
  }, [results]);

  // Get value class based on performance
  const getValueClass = (value, thresholds) => {
    if (value >= thresholds.good) return 'verdict__stat-value--good';
    if (value >= thresholds.average) return 'verdict__stat-value--average';
    return 'verdict__stat-value--poor';
  };

  // Handle Steam login
  const handleSteamLogin = () => {
    // Redirect to backend Steam OAuth
    window.location.href = `${process.env.REACT_APP_API_URL || 'http://localhost:5000'}/auth/steam`;
  };

  return (
    <div className="verdict">
      <div className="verdict__content">
        {/* Header */}
        <div className="verdict__header">
          <h1 className="verdict__title">
            Tu <span className="verdict__title-highlight">Veredicto</span>
          </h1>
          <p className="verdict__subtitle">
            La IA de StratAI ha analizado tu rendimiento.
          </p>
        </div>

        {/* Stats Grid */}
        {/* Stats Grid */}
        {!skipped && (
          <>
            <div className="verdict__stats">
              <div className="verdict__stat">
                <CrosshairIcon className="verdict__stat-icon" />
                <div className={`verdict__stat-value ${getValueClass(stats.accuracy, { good: 80, average: 60 })}`}>
                  {stats.accuracy}%
                </div>
                <div className="verdict__stat-label">Precisión</div>
              </div>
              
              <div className="verdict__stat">
                <ClockIcon className="verdict__stat-icon" />
                <div className={`verdict__stat-value ${getValueClass(500 - stats.reactionTime, { good: 200, average: 100 })}`}>
                  {stats.reactionTime}ms
                </div>
                <div className="verdict__stat-label">Reacción</div>
              </div>
              
              <div className="verdict__stat">
                <BrainIcon className="verdict__stat-icon" />
                <div className={`verdict__stat-value ${getValueClass(stats.knowledgeScore, { good: 66, average: 33 })}`}>
                  {stats.correctAnswers}/{stats.totalQuestions}
                </div>
                <div className="verdict__stat-label">Conocimiento</div>
              </div>
              
              <div className="verdict__stat">
                <TrophyIcon className="verdict__stat-icon" />
                <div className={`verdict__stat-value ${getValueClass(stats.overallScore, { good: 70, average: 50 })}`}>
                  {stats.overallScore}
                </div>
                <div className="verdict__stat-label">Puntuación</div>
              </div>
            </div>

            {/* Leaderboard Section */}
            <div className="verdict__leaderboard-section">
              <h3 className="verdict__leaderboard-title">Global Rankings</h3>
              <div className="verdict__leaderboard">
                 {[
                   { rank: 1, name: "ZywOo", score: 98, tier: 'S+' },
                   { rank: 2, name: "m0NESY", score: 95, tier: 'S' },
                   { rank: 3, name: "ropz", score: 92, tier: 'S' },
                   // Insert user if they scored well enough, otherwise just show them below
                   ...(stats.overallScore > 92 ? [] : [{ 
                      rank: 999, 
                      name: nickname || 'Agent', 
                      score: stats.overallScore, 
                      tier: stats.overallScore >= 80 ? 'A' : stats.overallScore >= 60 ? 'B' : 'C',
                      isUser: true 
                   }])
                 ].sort((a,b) => b.score - a.score).map((player, idx) => (
                   <div key={idx} className={`verdict__rank-row ${player.isUser ? 'current-user' : ''}`}>
                      <div className="rank-num">#{player.rank === 999 ? 'YOU' : player.rank}</div>
                      <div className="rank-name">{player.name}</div>
                      <div className="rank-score">
                        <span className={`tier-badge tier-${player.tier.replace('+','-plus')}`}>{player.tier}</span>
                        {player.score}
                      </div>
                   </div>
                 ))}
              </div>
            </div>
          </>
        )}

        {/* AI Analysis */}
        <div className="verdict__analysis">
          <div className="verdict__analysis-header">
            <div className="verdict__ai-avatar">S</div>
            <div className="verdict__ai-info">
              <div className="verdict__ai-name">StratAI</div>
              <div className="verdict__ai-role">Coach de Inteligencia Artificial</div>
            </div>
          </div>
          
          <div className="verdict__analysis-content">
            {skipped ? (
              <p>
                Has decidido no completar las pruebas de admisión. 
                <strong> Sin datos, no puedo evaluar tu nivel real.</strong> Pero no te preocupes — 
                una vez conectes tu cuenta de Steam, analizaré tus partidas reales y 
                te mostraré exactamente dónde puedes mejorar. (Nickname: {nickname || 'Unknown'})
              </p>
            ) : stats.overallScore >= 70 ? (
              <>
                <p>
                  <strong>Resultados sólidos.</strong> Tu precisión y tiempo de reacción están 
                  por encima de la media, y demuestras buen conocimiento táctico del juego.
                </p>
                <p>
                  Sin embargo, <span className="highlight">estos tests solo arañan la superficie</span>. 
                  En tus partidas reales, hay patrones de error que no puedes ver: 
                  posicionamientos subóptimos, timings predecibles, utilidad desperdiciada.
                </p>
                <p>
                  <strong>Conecta tu cuenta de Steam</strong> y te mostraré exactamente qué corregir 
                  para llegar al siguiente nivel.
                </p>
              </>
            ) : stats.overallScore >= 50 ? (
              <>
                <p>
                  <strong>Hay potencial, pero también hay trabajo por hacer.</strong> Tu rendimiento 
                  muestra áreas claras de mejora, especialmente en {stats.accuracy < 70 ? 'precisión mecánica' : 'conocimiento táctico'}.
                </p>
                <p>
                  La buena noticia: <span className="highlight">estos errores son corregibles</span>. 
                  Con el análisis de tus partidas reales, puedo identificar patrones específicos 
                  que están frenando tu progreso.
                </p>
                <p>
                  <strong>¿Estás listo para ver la verdad sobre tu gameplay?</strong>
                </p>
              </>
            ) : (
              <>
                <p>
                  <strong>Tienes mucho margen de mejora.</strong> Y eso es algo positivo — significa 
                  que pequeños cambios pueden tener un impacto enorme en tu rendimiento.
                </p>
                <p>
                  Lo que acabas de hacer es solo una prueba básica. 
                  <span className="highlight">El verdadero coaching comienza cuando analizo tus partidas</span>: 
                  cada duelo, cada decisión, cada error.
                </p>
                <p>
                  <strong>¿Quieres saber exactamente qué te está frenando?</strong>
                </p>
              </>
            )}
          </div>
        </div>

        {/* CTA */}
        <div className="verdict__cta">
          <p className="verdict__cta-message">
            <strong>Conecta tu cuenta de Steam</strong> para que StratAI analice tus partidas reales 
            y te ofrezca coaching personalizado.
          </p>
          
          <button className="verdict__steam-btn" onClick={handleSteamLogin}>
            <SteamIcon className="verdict__steam-icon" />
            Continuar con Steam
          </button>
          
          <div className="verdict__skip">
            <button className="verdict__skip-btn">
              Explorar sin conectar
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

// ═══════════════════════════════════════════════════════════════════════════
// Icons
// ═══════════════════════════════════════════════════════════════════════════
const CrosshairIcon = ({ className }) => (
  <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="12" cy="12" r="10" />
    <line x1="22" y1="12" x2="18" y2="12" />
    <line x1="6" y1="12" x2="2" y2="12" />
    <line x1="12" y1="6" x2="12" y2="2" />
    <line x1="12" y1="22" x2="12" y2="18" />
  </svg>
);

const ClockIcon = ({ className }) => (
  <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="12" cy="12" r="10" />
    <polyline points="12 6 12 12 16 14" />
  </svg>
);

const BrainIcon = ({ className }) => (
  <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M9.5 2A2.5 2.5 0 0 1 12 4.5v15a2.5 2.5 0 0 1-4.96.44 2.5 2.5 0 0 1-2.96-3.08 3 3 0 0 1-.34-5.58 2.5 2.5 0 0 1 1.32-4.24 2.5 2.5 0 0 1 4.44-2.54" />
    <path d="M14.5 2A2.5 2.5 0 0 0 12 4.5v15a2.5 2.5 0 0 0 4.96.44 2.5 2.5 0 0 0 2.96-3.08 3 3 0 0 0 .34-5.58 2.5 2.5 0 0 0-1.32-4.24 2.5 2.5 0 0 0-4.44-2.54" />
  </svg>
);

const TrophyIcon = ({ className }) => (
  <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M6 9H4.5a2.5 2.5 0 0 1 0-5H6" />
    <path d="M18 9h1.5a2.5 2.5 0 0 0 0-5H18" />
    <path d="M4 22h16" />
    <path d="M10 14.66V17c0 .55-.47.98-.97 1.21C7.85 18.75 7 20.24 7 22" />
    <path d="M14 14.66V17c0 .55.47.98.97 1.21C16.15 18.75 17 20.24 17 22" />
    <path d="M18 2H6v7a6 6 0 0 0 12 0V2Z" />
  </svg>
);

const SteamIcon = ({ className }) => (
  <svg className={className} viewBox="0 0 24 24" fill="currentColor">
    <path d="M12 2C6.48 2 2 6.48 2 12c0 4.84 3.44 8.87 8 9.8V15H8v-3h2V9.5C10 7.57 11.57 6 13.5 6H16v3h-2c-.55 0-1 .45-1 1v2h3l-.5 3H13v6.95c5.05-.5 9-4.76 9-9.95 0-5.52-4.48-10-10-10z" />
  </svg>
);

export default Verdict;
