import React, { useMemo } from 'react';
import { motion } from 'framer-motion';
import { Clock3, TrendingUp } from 'lucide-react';
import MissionCard from './MissionCard';
import '../../styles/Progress/missionsTab.css';

const containerVariants = {
  hidden: {},
  visible: { transition: { staggerChildren: 0.06 } },
};

const MissionsTab = ({ missions }) => {
  const enriched = useMemo(
    () => missions.map((m) => ({ ...m, percent: Math.min(100, Math.round((m.progress / m.total) * 100)) })),
    [missions],
  );

  const completedCount = enriched.filter((m) => m.completed).length;
  const rotationPct = Math.round((completedCount / enriched.length) * 100);
  const totalPts = enriched.reduce((sum, m) => sum + m.points, 0);

  return (
    <div className="mt">
      {/* Scorecard strip */}
      <div className="mt__scorecard">
        <div className="mt__stat">
          <span>Puntos en juego</span>
          <strong>{totalPts}</strong>
        </div>
        <div className="mt__stat">
          <span>Completadas</span>
          <strong>{completedCount}/{enriched.length}</strong>
        </div>
        <div className="mt__stat mt__stat--wide">
          <span>Rotación</span>
          <div className="mt__rot-bar">
            <div className="mt__rot-fill" style={{ width: `${rotationPct}%` }} />
          </div>
          <strong>{rotationPct}%</strong>
        </div>
      </div>

      {/* Mission grid */}
      <motion.div
        className="mt__grid"
        variants={containerVariants}
        initial="hidden"
        animate="visible"
      >
        {enriched.map((m, i) => (
          <MissionCard key={m.id} mission={m} index={i} />
        ))}
      </motion.div>

      {/* Rotation footer */}
      <div className="mt__footer">
        <div className="mt__footer-icon">
          <Clock3 size={15} />
        </div>
        <p>
          Completa las {enriched.length} misiones para desbloquear una nueva rotación.
          <span> La IA prioriza tareas accionables basadas en tus debilidades reales.</span>
        </p>
        <div className="mt__footer-badge">
          <TrendingUp size={13} />
          <span>{completedCount}/{enriched.length}</span>
        </div>
      </div>
    </div>
  );
};

export default MissionsTab;
