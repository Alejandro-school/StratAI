/**
 * VerdictSection — Professional AI evaluation report
 * Detailed per-area breakdown with strengths/weaknesses,
 * specific demo-analysis hooks, and compelling CTA.
 */
import React, { useMemo } from 'react';
import { motion } from 'framer-motion';
import { useLanding } from '../../LandingContext';
import { API_URL } from '../../../../utils/api';
import {
  ArrowRight,
  Brain,
  DollarSign,
  Bomb,
  Crosshair,
  Check,
  X,
  Minus,
  Target,
  Gauge,
  ShieldCheck,
  Activity,
} from 'lucide-react';
import '../../../../styles/Landing/sections/verdictRedesign.css';

/* ── Per-challenge deep analysis ──────────────────────────────────────── */
const CHALLENGE_ANALYSIS = {
  economy: {
    label: 'Gestión Económica',
    icon: DollarSign,
    pass: {
      verdict: 'Sólida',
      summary: 'Comprendes cuándo ahorrar, cuándo forzar y cómo maximizar el valor de cada ronda. Esto es una ventaja que la mayoría de jugadores no tiene.',
      demoHook: 'Con tus demos, la IA verificará si esta decisión es consistente — si compras bien en la ronda 4 pero fuerzas mal en la 14, hay un patrón de fatiga que detectamos.',
    },
    fail: {
      verdict: 'En desarrollo',
      summary: 'Las decisiones de compra no están optimizadas. Esto impacta directamente en cuántas rondas tienes el equipamiento correcto para ganar.',
      demoHook: 'La IA analiza cada ronda de tus demos: cuándo compras, cuánto gastas, y si tu equipo pierde rondas por falta de coordinación económica.',
    },
    skip: {
      verdict: 'No evaluada',
      summary: 'Sin datos suficientes para evaluar tu gestión económica en esta sesión.',
      demoHook: 'La IA extrae automáticamente cada decisión de compra de tus partidas — no necesitas hacer la prueba.',
    },
  },
  grenade: {
    label: 'Uso de Utilidad',
    icon: Bomb,
    pass: {
      verdict: 'Sólida',
      summary: 'Identificas correctamente los lineups clave. Sabes qué granadas importan y por qué — eso te da ventaja antes de que empiece el duelo.',
      demoHook: 'La IA mapea cada granada que tiras en tus partidas: dónde cae, si ciega a alguien, si retrasa una ejecución... Verás exactamente cuánto impacto tiene tu utilidad real y recibiras recomendaciones personalizadas para mejorarla.',
    },
    fail: {
      verdict: 'En desarrollo',
      summary: 'El conocimiento de lineups necesita trabajo. Sin utilidad efectiva, cada enfrentamiento depende solo del aim — y eso no escala.',
      demoHook: 'La IA detecta patrones: granadas que no impactan, smokes tardíos, flashes que ciegan a tu equipo. Te muestra exactamente qué mejorar y en qué mapas.',
    },
    skip: {
      verdict: 'No evaluada',
      summary: 'Sin datos suficientes para evaluar tu conocimiento de utilidad.',
      demoHook: 'Con tus demos, la IA registra cada granada y su efectividad — sin necesidad de pruebas teóricas.',
    },
  },
  gamesense: {
    label: 'Inteligencia Táctica',
    icon: Crosshair,
    pass: {
      verdict: 'Sólida',
      summary: 'Sabes leer situaciones y usar recursos para crear ventajas. Esto es lo que separa a jugadores que "saben" de jugadores que "ganan".',
      demoHook: 'La IA analiza tus clutches, retakes y post-plants reales. Verificará si esta decisión se repite o si bajo presión real tomas caminos diferentes.',
    },
    fail: {
      verdict: 'En desarrollo',
      summary: 'Las decisiones bajo presión no son óptimas. Es el área con más impacto inmediato — mejorar aquí cambia resultados de rondas directamente.',
      demoHook: 'La IA extrae cada situación de clutch de tus demos: qué decidiste, cuánto tardaste, si tenías información disponible que no usaste.',
    },
    skip: {
      verdict: 'No evaluada',
      summary: 'Sin datos suficientes para evaluar tu lectura táctica.',
      demoHook: 'Tus demos contienen decenas de decisiones tácticas por partida — la IA las analiza todas automáticamente.',
    },
  },
};

/* ── Profile classification ───────────────────────────────────────────── */
const getProfile = (successCount, anySkipped) => {
  if (anySkipped && successCount === 0) return { tag: 'Evaluación incompleta', cls: 'skipped' };
  if (successCount === 3) return { tag: 'Fundamentos sólidos', cls: 'elite' };
  if (successCount === 2) return { tag: 'Base avanzada', cls: 'advanced' };
  if (successCount === 1) return { tag: 'En desarrollo', cls: 'intermediate' };
  return { tag: 'Punto de partida', cls: 'beginner' };
};

/* ── Component ────────────────────────────────────────────────────────── */
const VerdictSection = () => {
  const { nickname, completedChallenges, skipped } = useLanding();

  const challengeKeys = ['economy', 'grenade', 'gamesense'];
  const successCount = challengeKeys.filter(k => completedChallenges[k]?.success).length;
  const failCount = challengeKeys.filter(k => completedChallenges[k]?.completed && !completedChallenges[k]?.success && !completedChallenges[k]?.skipped).length;
  const anySkipped = skipped || challengeKeys.some(k => completedChallenges[k]?.skipped);
  const evaluatedCount = challengeKeys.filter(k => completedChallenges[k]?.completed && !completedChallenges[k]?.skipped).length;
  const skippedCount = challengeKeys.filter(k => completedChallenges[k]?.skipped).length;
  const reportScore = Math.round((successCount / 3) * 100);
  const reliability = anySkipped ? 'Parcial' : successCount === 3 ? 'Alta' : successCount >= 1 ? 'Media' : 'Baja';

  const profile = getProfile(successCount, anySkipped);

  // Build analysis data for each challenge
  const analysisData = useMemo(() => {
    return challengeKeys.map(key => {
      const data = completedChallenges[key];
      const config = CHALLENGE_ANALYSIS[key];
      const wasSkipped = !!data?.skipped;
      const passed = !!data?.success;
      
      const tier = wasSkipped ? 'skip' : passed ? 'pass' : 'fail';
      return {
        key,
        ...config,
        tier,
        passed,
        wasSkipped,
        ...config[tier],
      };
    });
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [completedChallenges]);

  const handleCTA = () => { window.location.href = `${API_URL}/auth/steam/login`; };

  const stagger = { hidden: {}, visible: { transition: { staggerChildren: 0.08 } } };
  const fadeUp = { hidden: { opacity: 0, y: 12 }, visible: { opacity: 1, y: 0, transition: { duration: 0.4 } } };

  return (
    <section className="verdict">
      <div className="verdict__ambient" aria-hidden="true" />
      <div className="verdict__content">

        {/* ── Header ────────────────────────────────────────── */}
        <motion.div
          className="verdict__header"
          initial={{ opacity: 0, y: -12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5 }}
        >
          <div className="verdict__badge">
            <Brain size={14} />
            <span>Informe de Evaluación</span>
          </div>
          <h1 className="verdict__title">
            {nickname ? `${nickname}, ` : ''}Diagnóstico completado.
          </h1>
          <p className="verdict__subtitle">
            Resumen ejecutivo de tu perfil táctico basado en las pruebas rápidas. El informe real
            se genera al analizar tus demos.
          </p>
          <div className="verdict__profile-row">
            <span className={`verdict__profile-tag verdict__profile-tag--${profile.cls}`}>
              {profile.tag}
            </span>
            <span className="verdict__score">{successCount}/3 fundamentos</span>
          </div>
          {anySkipped && (
            <div className="verdict__notice">
              <ShieldCheck size={12} />
              Evaluación incompleta. Conecta Steam para completar el análisis.
            </div>
          )}
        </motion.div>

        <hr className="verdict__separator" />

        <div className="verdict__grid">
          {/* ── Summary Panel ─────────────────────────────────── */}
          <motion.div
            className="verdict__summary"
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.15, duration: 0.5 }}
          >
            <div className="verdict__report">
              <div className="verdict__report-header">
                <Gauge size={16} />
                <span>Resumen operativo</span>
              </div>
              <div className="verdict__score-block">
                <div className="verdict__score-main">
                  <span className="verdict__score-number">{reportScore}</span>
                  <span className="verdict__score-max">/100</span>
                </div>
                <div className="verdict__score-caption">Consistencia teórica</div>
                <div className="verdict__score-track" aria-hidden="true">
                  <span className="verdict__score-fill" style={{ width: `${reportScore}%` }} />
                </div>
              </div>
              <div className="verdict__report-meta">
                <div className="verdict__meta-row">
                  <span className="verdict__meta-label">Confiabilidad</span>
                  <span className="verdict__meta-value">{reliability}</span>
                </div>
                <div className="verdict__meta-row">
                  <span className="verdict__meta-label">Cobertura</span>
                  <span className="verdict__meta-value">{evaluatedCount}/3 fundamentos</span>
                </div>
                <div className="verdict__meta-row">
                  <span className="verdict__meta-label">Estado</span>
                  <span className="verdict__meta-value">{anySkipped ? 'Pendiente de datos' : 'Completo'}</span>
                </div>
              </div>
              <div className="verdict__kpis">
                <div className="verdict__kpi">
                  <span className="verdict__kpi-value">{successCount}</span>
                  <span className="verdict__kpi-label">Fortalezas claras</span>
                </div>
                <div className="verdict__kpi">
                  <span className="verdict__kpi-value">{failCount}</span>
                  <span className="verdict__kpi-label">Focos críticos</span>
                </div>
                <div className="verdict__kpi">
                  <span className="verdict__kpi-value">{skippedCount}</span>
                  <span className="verdict__kpi-label">Sin evidencia</span>
                </div>
              </div>
              <div className="verdict__mini-row">
                <span className="verdict__mini">
                  <ShieldCheck size={12} /> {evaluatedCount}/3 evaluados
                </span>
                <span className="verdict__mini">
                  <Activity size={12} /> {failCount} en desarrollo
                </span>
              </div>
            </div>
          </motion.div>

          {/* ── Detailed Breakdown ────────────────────────────── */}
          <div className="verdict__details">
            <motion.div
              className="verdict__breakdown"
              variants={stagger}
              initial="hidden"
              animate="visible"
            >
              {analysisData.map((area) => {
                const Icon = area.icon;
                const statusClass = area.wasSkipped ? 'skipped' : area.passed ? 'passed' : 'failed';
                
                return (
                  <motion.div
                    key={area.key}
                    className={`verdict__area verdict__area--${statusClass}`}
                    variants={fadeUp}
                  >
                    <div className="verdict__area-header">
                      <div className="verdict__area-icon">
                        <Icon size={16} />
                      </div>
                      <div className="verdict__area-title-group">
                        <span className="verdict__area-label">{area.label}</span>
                        <span className={`verdict__area-verdict verdict__area-verdict--${statusClass}`}>
                          {area.verdict}
                        </span>
                      </div>
                      <div className={`verdict__area-status verdict__area-status--${statusClass}`}>
                        {area.wasSkipped ? <Minus size={14} /> : area.passed ? <Check size={14} /> : <X size={14} />}
                      </div>
                    </div>

                    <p className="verdict__area-summary">{area.summary}</p>

                    <div className="verdict__area-demo-hook">
                      <Target size={13} />
                      <p>{area.demoHook}</p>
                    </div>
                  </motion.div>
                );
              })}
            </motion.div>

          </div>
        </div>

        {/* ── CTA ────────────────────────────────────────────── */}
        <motion.div
          className="verdict__cta"
          initial={{ opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.7, duration: 0.5 }}
        >
          <div className="verdict__cta-copy">
            <span className="verdict__cta-label">Siguiente paso</span>
            <p className="verdict__cta-text">
              Conecta Steam para generar un informe tactico real con patrones, timings y
              recomendaciones personalizadas.
            </p>
          </div>
          <button className="verdict__cta-btn" onClick={handleCTA}>
            <span>Conectar Steam y obtener análisis real</span>
            <ArrowRight size={18} />
          </button>
          <p className="verdict__cta-sub">
            La IA descarga y analiza tus últimas partidas automáticamente. Sin configuración.
          </p>
        </motion.div>

      </div>
    </section>
  );
};

export default VerdictSection;
