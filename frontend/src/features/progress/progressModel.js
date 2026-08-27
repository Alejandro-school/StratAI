export const MAX_WEEKLY_POINTS = 1000;

const clamp = (value, min = 0, max = 1) => Math.min(max, Math.max(min, value));

export const getMissionProgress = (mission) => {
  const distance = mission.target - mission.baseline;
  if (distance === 0) return mission.completed ? 1 : 0;

  const progress = (mission.current - mission.baseline) / distance;
  return clamp(progress);
};

export const getMissionPoints = (mission) => {
  const progress = getMissionProgress(mission);
  const completionBonus = mission.completed || progress >= 1 ? 0.2 : 0;
  return Math.round(mission.maxPoints * ((progress * 0.8) + completionBonus));
};

export const getWeeklyScore = (missions) => (
  missions.reduce((total, mission) => total + getMissionPoints(mission), 0)
);

export const canRerollMission = (mission, hasReroll) => (
  hasReroll && !mission.completed && getMissionProgress(mission) < 0.2
);

export const formatMetricValue = (value, unit) => {
  if (unit === '%') return `${new Intl.NumberFormat('es-ES', { maximumFractionDigits: 1 }).format(value)}%`;
  if (unit === 'ms') return `${Math.round(value)} ms`;
  if (unit === 'ratio') return new Intl.NumberFormat('es-ES', { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(value);
  return new Intl.NumberFormat('es-ES', { maximumFractionDigits: 1 }).format(value);
};

export const getNextWeeklyReset = (now = new Date()) => {
  const reset = new Date(now);
  const daysUntilMonday = (8 - reset.getUTCDay()) % 7 || 7;
  reset.setUTCDate(reset.getUTCDate() + daysUntilMonday);
  reset.setUTCHours(0, 0, 0, 0);
  return reset;
};

export const formatResetCountdown = (resetAt, now = new Date()) => {
  const remaining = Math.max(resetAt.getTime() - now.getTime(), 0);
  const days = Math.floor(remaining / 86_400_000);
  const hours = Math.floor((remaining % 86_400_000) / 3_600_000);
  const minutes = Math.floor((remaining % 3_600_000) / 60_000);
  return `${days}d ${hours}h ${minutes}m`;
};

export const replaceMission = (missions, missionId, replacement) => (
  missions.map((mission) => {
    if (mission.id !== missionId) return mission;
    return { ...replacement, role: mission.role, maxPoints: mission.maxPoints };
  })
);
