import React from 'react';
import { motion } from 'framer-motion';
import { CheckCircle2, Zap } from 'lucide-react';
import { difficultyConfig, categoryConfig } from '../../mocks/progressData';
import '../../styles/Progress/missionCard.css';

const cardVariants = {
  hidden: { opacity: 0, y: 12 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.35, ease: 'easeOut' } },
};

const MissionCard = ({ mission, index = 0 }) => {
  const diff = difficultyConfig[mission.difficulty];
  const cat = categoryConfig[mission.category];
  const CatIcon = cat.icon;
  const percent = Math.min(100, Math.round((mission.progress / mission.total) * 100));

  return (
    <motion.article
      className={`mc ${diff.className} ${mission.completed ? 'mc--done' : ''}`}
      variants={cardVariants}
      transition={{ delay: index * 0.04 }}
    >
      {/* Top row: category + difficulty + points */}
      <div className="mc__top">
        <div className="mc__cat">
          <CatIcon size={13} />
          <span>{cat.label}</span>
        </div>
        <div className={`mc__diff ${diff.className}`}>
          <span>{'★'.repeat(diff.stars)}{'☆'.repeat(3 - diff.stars)}</span>
          <small>{diff.label}</small>
        </div>
        <div className="mc__pts">
          {mission.completed ? <CheckCircle2 size={13} /> : <Zap size={13} />}
          <strong>+{mission.points}</strong>
        </div>
      </div>

      {/* Title + truncated summary */}
      <h3 className="mc__title">{mission.title}</h3>
      <p className="mc__summary">{mission.summary}</p>

      {/* Progress bar */}
      <div className="mc__bar-row">
        <div className="mc__track">
          <motion.div
            className="mc__fill"
            initial={{ width: 0 }}
            animate={{ width: `${percent}%` }}
            transition={{ duration: 0.8, ease: [0.22, 1, 0.36, 1], delay: 0.15 + index * 0.06 }}
          />
        </div>
        <span className="mc__pct">{mission.progress}/{mission.total}</span>
      </div>
    </motion.article>
  );
};

export default MissionCard;
