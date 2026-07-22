import React from 'react';
import { motion } from 'framer-motion';
import { ArrowUpRight, Crown, Medal, Trophy, Gift, Zap } from 'lucide-react';
import RankBadge from './RankBadge';
import { getTierForPoints } from '../../mocks/progressData';
import '../../styles/Progress/leaderboard.css';

const PODIUM_META = {
  1: { icon: Crown, label: '1st', order: 2, height: 'tall' },
  2: { icon: Medal, label: '2nd', order: 1, height: 'mid' },
  3: { icon: Trophy, label: '3rd', order: 3, height: 'short' },
};

const getInitials = (name) =>
  name
    .split(/[\s_]+/)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase())
    .join('');

const RankingTab = ({ leaderboard, currentPlayer, rewards = [] }) => {
  const rewardMap = Object.fromEntries(rewards.map((r) => [r.rank, r]));
  const podium = leaderboard.filter((e) => e.rank <= 3);
  const rest = leaderboard.filter((e) => e.rank > 3);
  const maxPts = podium[0]?.points || 1;

  return (
    <div className="lb">
      {/* ── Podium — classic 2-1-3 layout ── */}
      <div className="lb__podium">
        {podium.map((entry, i) => {
          const meta = PODIUM_META[entry.rank];
          const PIcon = meta.icon;
          const tier = getTierForPoints(entry.points);
          const reward = rewardMap[entry.rank];
          const pctOfMax = Math.round((entry.points / maxPts) * 100);

          return (
            <motion.div
              key={entry.rank}
              className={`lb__pod lb__pod--${meta.height}`}
              style={{ '--pod-color': tier.color, '--pod-glow': tier.glow, order: meta.order }}
              initial={{ opacity: 0, y: 24, scale: 0.95 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              transition={{ delay: i * 0.1, duration: 0.45, ease: [0.16, 1, 0.3, 1] }}
            >
              {/* Glow ring behind avatar for #1 */}
              {entry.rank === 1 && <div className="lb__pod-glow-ring" />}

              {/* Avatar circle */}
              <div className="lb__pod-avatar" style={{ '--av-color': tier.color }}>
                <span>{getInitials(entry.name)}</span>
                <div className="lb__pod-icon-badge">
                  <PIcon size={12} />
                </div>
              </div>

              {/* Rank label */}
              <span className="lb__pod-rank">{meta.label}</span>

              {/* Name */}
              <strong className="lb__pod-name">{entry.name}</strong>

              {/* Points — big number */}
              <div className="lb__pod-pts">{entry.points.toLocaleString('es-ES')}</div>

              {/* Mini bar showing relative points */}
              <div className="lb__pod-bar">
                <motion.div
                  className="lb__pod-bar-fill"
                  initial={{ width: 0 }}
                  animate={{ width: `${pctOfMax}%` }}
                  transition={{ delay: 0.3 + i * 0.1, duration: 0.6, ease: 'easeOut' }}
                />
              </div>

              {/* Trend */}
              <div className="lb__pod-trend">
                <ArrowUpRight size={12} />
                <span>{entry.trend}</span>
              </div>

              {/* Reward badge */}
              {reward && (
                <div className={`lb__pod-reward lb__pod-reward--${reward.accent}`}>
                  <Gift size={11} />
                  <span>{reward.title}</span>
                </div>
              )}
            </motion.div>
          );
        })}
      </div>

      {/* ── Divider ── */}
      <div className="lb__divider">
        <Zap size={12} />
        <span>Clasificación general</span>
        <Zap size={12} />
      </div>

      {/* ── List table ── */}
      <div className="lb__list">
        <div className="lb__list-head">
          <span>Pos</span>
          <span>Jugador</span>
          <span>Puntos</span>
          <span>Tendencia</span>
          <span>Premio</span>
        </div>
        {rest.map((entry, i) => {
          const reward = rewardMap[entry.rank];
          const tier = getTierForPoints(entry.points);
          return (
            <motion.div
              key={entry.rank}
              className={`lb__row ${entry.isCurrentUser ? 'lb__row--you' : ''}`}
              initial={{ opacity: 0, x: -10 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: 0.35 + i * 0.04 }}
            >
              <span className="lb__row-rank" style={{ color: tier.color }}>#{entry.rank}</span>
              <span className="lb__row-name">
                <span className="lb__row-avatar" style={{ '--av-color': tier.color }}>
                  {getInitials(entry.name)}
                </span>
                {entry.name}
                {entry.isCurrentUser && <span className="lb__you-tag">Tú</span>}
              </span>
              <strong className="lb__row-pts">{entry.points.toLocaleString('es-ES')}</strong>
              <span className="lb__row-trend">
                <ArrowUpRight size={11} />
                {entry.trend}
              </span>
              <span className={`lb__row-reward ${reward ? `lb__row-reward--${reward.accent}` : ''}`}>
                {reward ? reward.title : '—'}
              </span>
            </motion.div>
          );
        })}
      </div>

      {/* ── Your zone (if user not visible in list) ── */}
      {currentPlayer && !rest.some((e) => e.isCurrentUser) && (
        <div className="lb__your-zone">
          <span className="lb__your-zone-label">Tu posición</span>
          <div className="lb__your-zone-card">
            <RankBadge points={currentPlayer.points} rank={currentPlayer.rank} size="compact" />
            <strong>{currentPlayer.name}</strong>
            <span className="lb__your-zone-trend">
              <ArrowUpRight size={12} />
              {currentPlayer.trend}
            </span>
          </div>
        </div>
      )}
    </div>
  );
};

export default RankingTab;
