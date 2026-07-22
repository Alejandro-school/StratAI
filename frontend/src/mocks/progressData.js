import {
  Flame,
  Shield,
  Sparkles,
  Star,
  Target,
} from 'lucide-react';

/* ─── Tier thresholds ─── */
export const TIER_THRESHOLDS = [
  { id: 'bronze',  label: 'Bronce',   min: 0,    max: 999,  color: '#CD7F32', glow: 'rgba(205,127,50,0.35)' },
  { id: 'silver',  label: 'Plata',    min: 1000, max: 1499, color: '#C0C0C0', glow: 'rgba(192,192,192,0.35)' },
  { id: 'gold',    label: 'Oro',      min: 1500, max: 1999, color: '#FFD700', glow: 'rgba(255,215,0,0.35)' },
  { id: 'diamond', label: 'Diamante', min: 2000, max: Infinity, color: '#B9F2FF', glow: 'rgba(185,242,255,0.4)' },
];

export const getTierForPoints = (points) =>
  TIER_THRESHOLDS.find((t) => points >= t.min && points <= t.max) || TIER_THRESHOLDS[0];

/* ─── Difficulty config ─── */
export const difficultyConfig = {
  easy:   { label: 'Fácil',   stars: 1, className: 'easy' },
  medium: { label: 'Media',   stars: 2, className: 'medium' },
  hard:   { label: 'Difícil', stars: 3, className: 'hard' },
};

/* ─── Category config ─── */
export const categoryConfig = {
  aim:         { label: 'Aim',          icon: Target },
  utility:     { label: 'Utilidad',     icon: Sparkles },
  economy:     { label: 'Economía',     icon: Shield },
  mechanics:   { label: 'Mecánicas',    icon: Flame },
  consistency: { label: 'Consistencia', icon: Star },
};

/* ─── Mission seed ─── */
export const missionSeed = [
  {
    id: 1,
    category: 'aim',
    difficulty: 'medium',
    title: 'Sube tu HS% por encima del 47% en 4 partidas',
    summary: 'La IA ha detectado buenas aperturas pero poco cierre limpio en duelos medios.',
    progress: 2,
    total: 4,
    points: 150,
  },
  {
    id: 2,
    category: 'utility',
    difficulty: 'easy',
    title: 'Fuerza 12 asistencias de flash antes del reinicio mensual',
    summary: 'Tus rondas de apoyo aportan visión, pero aún convierten poco en ventaja real.',
    progress: 8,
    total: 12,
    points: 90,
  },
  {
    id: 3,
    category: 'economy',
    difficulty: 'medium',
    title: 'Evita 3 compras forzadas negativas después de perder pistola',
    summary: 'Se pierde impacto por compras reactivas que rompen el ciclo económico del equipo.',
    progress: 1,
    total: 3,
    points: 160,
  },
  {
    id: 4,
    category: 'mechanics',
    difficulty: 'hard',
    title: 'Lleva el counter-strafe eficaz al 65% en tus próximos duelos',
    summary: 'Tu primer disparo mejora, pero aún entras con velocidad residual en demasiados picos.',
    progress: 41,
    total: 65,
    points: 280,
  },
  {
    id: 5,
    category: 'consistency',
    difficulty: 'hard',
    title: 'Mantén un K/D superior a 1.15 durante 5 partidas seguidas',
    summary: 'Cuando ganas mapa controlas bien el ritmo; el problema aparece al encadenar sesiones largas.',
    progress: 5,
    total: 5,
    points: 240,
    completed: true,
  },
  {
    id: 6,
    category: 'aim',
    difficulty: 'hard',
    title: 'Consigue 3 aces o cuádruples en tus próximas 10 partidas',
    summary: 'Tu capacidad de clutch existe pero no conviertes suficientes rondas de ventaja numérica.',
    progress: 1,
    total: 3,
    points: 300,
  },
];

/* ─── Leaderboard seed ─── */
export const leaderboardSeed = [
  { rank: 1,  name: 'Nexus',              points: 2340, trend: '+180', rewardTier: 'gold' },
  { rank: 2,  name: 'Raze',               points: 2195, trend: '+120', rewardTier: 'silver' },
  { rank: 3,  name: 'Vektor',             points: 2050, trend: '+95',  rewardTier: 'bronze' },
  { rank: 4,  name: 'Horizon',            points: 1910, trend: '+75' },
  { rank: 5,  name: 'Pulse',              points: 1820, trend: '+60' },
  { rank: 10, name: 'Ghostline',          points: 1395, trend: '+35' },
  { rank: 11, name: 'Rift',               points: 1320, trend: '+28' },
  { rank: 12, name: '__CURRENT_USER__',   points: 1250, trend: '+42', isCurrentUser: true },
  { rank: 13, name: 'Anchor',             points: 1215, trend: '+18' },
];

/* ─── Reward seed ─── */
export const rewardSeed = [
  { rank: 1, title: '3 meses de IA Pro',            accent: 'gold' },
  { rank: 2, title: '1 mes de IA Pro',              accent: 'silver' },
  { rank: 3, title: '2 semanas de IA Pro',           accent: 'bronze' },
  { rank: 4, title: '1 semana de IA Pro',            accent: 'steel' },
  { rank: 5, title: '3 días de IA Pro',              accent: 'steel' },
];
