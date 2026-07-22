import React from 'react';
import { motion } from 'framer-motion';
import { Award, Crown, Lock, Trophy, Unlock } from 'lucide-react';
import { getTierForPoints, TIER_THRESHOLDS } from '../../mocks/progressData';
import '../../styles/Progress/rewards.css';

const TIER_ICONS = {
  gold:   Crown,
  silver: Award,
  bronze: Trophy,
  steel:  Award,
};

const cardVariants = {
  hidden: { opacity: 0, scale: 0.95 },
  visible: { opacity: 1, scale: 1, transition: { duration: 0.35, ease: 'easeOut' } },
};

const RewardsTab = ({ rewards, currentPoints }) => {
  const currentTier = getTierForPoints(currentPoints);

  return (
    <div className="rw">
      {/* Tier roadmap */}
      <div className="rw__roadmap">
        <div className="rw__roadmap-line" />
        {TIER_THRESHOLDS.map((tier, i) => {
          const isActive = currentTier.id === tier.id;
          const isUnlocked = currentPoints >= tier.min;
          return (
            <motion.div
              key={tier.id}
              className={`rw__tier ${isActive ? 'rw__tier--active' : ''} ${isUnlocked ? 'rw__tier--unlocked' : ''}`}
              style={{ '--t-color': tier.color, '--t-glow': tier.glow }}
              variants={cardVariants}
              initial="hidden"
              animate="visible"
              transition={{ delay: i * 0.1 }}
            >
              <div className="rw__tier-marker">
                <span className="rw__tier-diamond">◆</span>
              </div>
              <div className="rw__tier-body">
                <div className="rw__tier-head">
                  <span className="rw__tier-name">{tier.label}</span>
                  {isUnlocked ? <Unlock size={12} /> : <Lock size={12} />}
                </div>
                <span className="rw__tier-range">
                  {tier.max === Infinity
                    ? `${tier.min.toLocaleString('es-ES')}+ pts`
                    : `${tier.min.toLocaleString('es-ES')} – ${tier.max.toLocaleString('es-ES')} pts`}
                </span>
              </div>
            </motion.div>
          );
        })}
      </div>

      {/* Reward cards */}
      <div className="rw__heading">
        <Trophy size={15} />
        <span>Premios mensuales — Top 5</span>
      </div>

      <div className="rw__cards">
        {rewards.map((r, i) => {
          const RIcon = TIER_ICONS[r.accent] || Award;
          return (
            <motion.div
              key={r.rank}
              className={`rw__card rw__card--${r.accent}`}
              variants={cardVariants}
              initial="hidden"
              animate="visible"
              transition={{ delay: 0.3 + i * 0.08 }}
            >
              <div className="rw__card-rank">#{r.rank}</div>
              <div className="rw__card-icon">
                <RIcon size={16} />
              </div>
              <div className="rw__card-copy">
                <strong>{r.title}</strong>
                <small>{r.detail}</small>
              </div>
            </motion.div>
          );
        })}
      </div>
    </div>
  );
};

export default RewardsTab;
