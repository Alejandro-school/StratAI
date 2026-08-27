import React, { useMemo, useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import {
  Banknote,
  Crosshair,
  Gauge,
  LayoutDashboard,
  ShieldAlert,
  Swords,
} from 'lucide-react';
import { buildPerformanceDetail } from '../../data/performanceDataAdapters';
import ArsenalSection from './ArsenalSection';
import DuelsSection from './DuelsSection';
import EconomySection from './EconomySection';
import MechanicsSection from './MechanicsSection';
import OverviewSection from './OverviewSection';
import UtilitySection from './UtilitySection';

const SECTIONS = [
  { id: 'overview', label: 'Resumen competitivo', icon: LayoutDashboard },
  { id: 'utility', label: 'Utilidad táctica', icon: ShieldAlert },
  { id: 'weapons', label: 'Arsenal', icon: Crosshair },
  { id: 'duels', label: 'Enfrentamientos', icon: Swords },
  { id: 'mechanics', label: 'Mecánicas', icon: Gauge },
  { id: 'economy', label: 'Economía', icon: Banknote },
];

const StatisticsWorkspace = ({
  context,
  payload,
  activeSide,
  onSideChange,
}) => {
  const [activeSection, setActiveSection] = useState('overview');
  const detail = useMemo(() => buildPerformanceDetail(payload), [payload]);

  const sectionContent = {
    overview: (
      <OverviewSection
        context={context}
        activeSide={activeSide}
        onSideChange={onSideChange}
        data={detail.overview}
      />
    ),
    utility: <UtilitySection data={detail.utility} />,
    weapons: <ArsenalSection data={detail.weapons} />,
    duels: <DuelsSection data={detail.duels} />,
    mechanics: <MechanicsSection data={detail.mechanics} />,
    economy: <EconomySection data={detail.economy} />,
  };

  return (
    <section className="pf3-statistics-workspace" aria-label="Panel estadístico">
      <header className="pf3-statistics-head">
        <div>
          <span className="pf3-kicker">Centro de análisis</span>
          <h2>{context.name}</h2>
          <p>Métricas segmentadas directamente desde las demos procesadas por Go.</p>
        </div>
        <span className="pf3-simulated-badge">
          {payload.sources?.summary_matches || context.matches} partidas · datos reales
        </span>
      </header>

      <div className="pf3-stat-tabs" role="tablist" aria-label="Categorías estadísticas">
        {SECTIONS.map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            type="button"
            role="tab"
            aria-selected={activeSection === id}
            className={activeSection === id ? 'is-active' : ''}
            onClick={() => setActiveSection(id)}
          >
            <Icon size={17} aria-hidden="true" />
            <span>{label}</span>
          </button>
        ))}
      </div>

      <AnimatePresence mode="wait">
        <motion.div
          key={`${context.id}-${activeSection}`}
          className="pf3-stat-section"
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: -6 }}
          transition={{ duration: 0.22, ease: 'easeOut' }}
          role="tabpanel"
        >
          {sectionContent[activeSection]}
        </motion.div>
      </AnimatePresence>
    </section>
  );
};

export default StatisticsWorkspace;
