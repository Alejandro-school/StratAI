// frontend/src/pages/Progreso.jsx
// Progress Hub - Misiones, badges y evolución del jugador
import React from 'react';
import NavigationFrame from '../components/Layout/NavigationFrame';
import { 
  TrendingUp, Award, Target, Flame,
  Calendar, CheckCircle, Lock, Star,
  Trophy, Zap, Crown, Shield
} from 'lucide-react';
import '../styles/pages/progreso.css';

const Progreso = () => {
  // Mock data for placeholders
  const weeklyGoals = [
    { id: 1, title: 'Gana 5 partidas', progress: 3, total: 5, xp: 100 },
    { id: 2, title: 'Consigue 10 headshots', progress: 10, total: 10, xp: 50, completed: true },
    { id: 3, title: 'K/D > 1.0 en 3 partidas', progress: 1, total: 3, xp: 75 },
  ];

  const badges = [
    { id: 1, name: 'Clutch King', icon: Crown, unlocked: true, description: 'Gana 10 clutches 1v2' },
    { id: 2, name: 'Headhunter', icon: Target, unlocked: true, description: 'HS% > 50 en 5 partidas' },
    { id: 3, name: 'On Fire', icon: Flame, unlocked: false, description: 'Racha de 5 victorias' },
    { id: 4, name: 'Economy Master', icon: Shield, unlocked: false, description: '0 force buys en 10 partidas' },
    { id: 5, name: 'MVP', icon: Trophy, unlocked: false, description: 'Sé MVP 20 veces' },
    { id: 6, name: 'Consistent', icon: Star, unlocked: false, description: 'K/D > 1.2 durante 1 semana' },
  ];

  const streaks = [
    { label: 'Racha Actual', value: 3, type: 'win', icon: Flame },
    { label: 'Mejor Racha', value: 7, type: 'record', icon: Trophy },
    { label: 'Días Activo', value: 12, type: 'days', icon: Calendar },
  ];

  return (
    <NavigationFrame>
      <div className="progreso-container">
        {/* Header */}
        <header className="progreso-header">
          <div className="header-content">
            <h1>
              <TrendingUp className="header-icon" />
              Progreso
            </h1>
            <p className="header-subtitle">
              Misiones semanales, logros y evolución de tu rendimiento
            </p>
          </div>
        </header>

        {/* Main Content */}
        <main className="progreso-content">
          {/* Streaks Row */}
          <section className="streaks-section">
            {streaks.map((streak, i) => {
              const Icon = streak.icon;
              return (
                <div key={i} className={`streak-card ${streak.type}`}>
                  <Icon className="streak-icon" />
                  <span className="streak-value">{streak.value}</span>
                  <span className="streak-label">{streak.label}</span>
                </div>
              );
            })}
          </section>

          <div className="progreso-grid">
            {/* Weekly Goals */}
            <section className="goals-section">
              <div className="section-header">
                <h2><Target size={20} /> Misiones Semanales</h2>
                <span className="reset-timer">
                  <Calendar size={14} />
                  Reinicio en 3d 14h
                </span>
              </div>

              <div className="goals-list">
                {weeklyGoals.map(goal => (
                  <div 
                    key={goal.id} 
                    className={`goal-card ${goal.completed ? 'completed' : ''}`}
                  >
                    <div className="goal-check">
                      {goal.completed ? (
                        <CheckCircle className="check-icon" />
                      ) : (
                        <div className="check-empty" />
                      )}
                    </div>
                    <div className="goal-info">
                      <span className="goal-title">{goal.title}</span>
                      <div className="goal-progress-bar">
                        <div 
                          className="goal-progress-fill"
                          style={{ width: `${(goal.progress / goal.total) * 100}%` }}
                        />
                      </div>
                      <span className="goal-count">{goal.progress}/{goal.total}</span>
                    </div>
                    <div className="goal-xp">
                      <Zap size={12} />
                      <span>+{goal.xp} XP</span>
                    </div>
                  </div>
                ))}
              </div>
            </section>

            {/* Badges Grid */}
            <section className="badges-section">
              <div className="section-header">
                <h2><Award size={20} /> Logros</h2>
                <span className="badges-count">
                  {badges.filter(b => b.unlocked).length}/{badges.length} desbloqueados
                </span>
              </div>

              <div className="badges-grid">
                {badges.map(badge => {
                  const Icon = badge.icon;
                  return (
                    <div 
                      key={badge.id}
                      className={`badge-card ${badge.unlocked ? 'unlocked' : 'locked'}`}
                    >
                      <div className="badge-icon-wrapper">
                        {badge.unlocked ? (
                          <Icon className="badge-icon" />
                        ) : (
                          <Lock className="badge-icon" />
                        )}
                      </div>
                      <span className="badge-name">{badge.name}</span>
                      <span className="badge-desc">{badge.description}</span>
                    </div>
                  );
                })}
              </div>
            </section>
          </div>

          {/* Rank Evolution Placeholder */}
          <section className="evolution-section">
            <div className="section-header">
              <h2><TrendingUp size={20} /> Evolución de Ranking</h2>
            </div>
            <div className="evolution-placeholder">
              <TrendingUp size={48} />
              <p>Gráfico de evolución en desarrollo</p>
              <span>Visualiza tu progreso de rating a lo largo del tiempo</span>
            </div>
          </section>
        </main>
      </div>
    </NavigationFrame>
  );
};

export default Progreso;
