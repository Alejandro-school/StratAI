import React, { useMemo, useState } from 'react';
import {
  AlertTriangle,
  Crosshair,
  Flame,
  LayoutDashboard,
  Map as MapIcon,
  Swords,
  Target,
  Users,
} from 'lucide-react';
import NavigationFrame from '../components/Layout/NavigationFrame';
import { useUser } from '../context/UserContext';
import { usePerformanceData } from '../hooks/usePerformanceData';
import { computeTrend } from '../utils/performanceFormatters';
import {
  AimTab,
  CombatTab,
  CompareTab,
  MapsTab,
  OverviewTab,
  UtilityTab,
  WeaponsTab,
} from '../components/Performance';
import { PerformanceHero, TrainingAreaNav } from '../components/Performance/PerformanceBriefing';
import { buildPerformanceViewModel } from '../components/Performance/performanceViewModel';
import '../styles/pages/performance.css';

const TABS = [
  { id: 'overview', label: 'Resumen', icon: LayoutDashboard },
  { id: 'combat', label: 'Combate', icon: Swords },
  { id: 'aim', label: 'Puntería', icon: Crosshair },
  { id: 'weapons', label: 'Arsenal', icon: Target },
  { id: 'maps', label: 'Mapas', icon: MapIcon },
  { id: 'utility', label: 'Utilidad', icon: Flame },
  { id: 'compare', label: 'Comparar', icon: Users },
];

const PerformanceSkeleton = () => (
  <div className="p-page p-skeleton-view" aria-hidden="true">
    <header className="p-header">
      <div className="p-skel p-skel--title" />
      <div className="p-skel p-skel--sub" />
      <div className="p-skel-tabs p-mt-20">
        {Array.from({ length: 7 }).map((_, i) => (
          <div key={i} className="p-skel p-skel--tab" />
        ))}
      </div>
    </header>
    <div className="p-skel-metric-row">
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="p-skel p-skel--metric" />
      ))}
    </div>
    <div className="p-grid p-grid-wide">
      <div className="p-card p-skel p-skel--card-lg" />
      <div className="p-card p-skel p-skel--card-lg" />
    </div>
  </div>
);

const Performance = () => {
  const { user } = useUser();
  const { performance, loading, error, retry } = usePerformanceData(user);
  const [activeTab, setActiveTab] = useState('overview');

  const sections = useMemo(() => ({
    overview: performance?.overview || {},
    sides:    performance?.sides    || {},
    aim:      performance?.aim      || {},
    combat:   performance?.combat   || {},
    utility:  performance?.utility  || {},
    weapons:  performance?.weapons  || [],
    maps:     performance?.maps     || [],
    history:  performance?.match_history || [],
    economy:  performance?.economy  || {},
  }), [performance]);

  const trends = useMemo(() => {
    const h = sections.history;
    const o = sections.overview;
    return {
      hltv_rating: computeTrend(h, 'hltv_rating', o.hltv_rating),
      kd_ratio:    computeTrend(h, 'kd_ratio',    o.kd_ratio),
      adr:         computeTrend(h, 'adr',         o.adr),
      hs_pct:      computeTrend(h, 'hs_percentage', o.hs_pct),
    };
  }, [sections.history, sections.overview]);

  const viewModel = useMemo(() => buildPerformanceViewModel({
    overview: sections.overview,
    sides: sections.sides,
    aim: sections.aim,
    combat: sections.combat,
    utility: sections.utility,
    weapons: sections.weapons,
    maps: sections.maps,
    history: sections.history,
    trends,
  }), [sections, trends]);

  if (loading) {
    return (
      <NavigationFrame>
        <PerformanceSkeleton />
      </NavigationFrame>
    );
  }

  if (error || !performance) {
    return (
      <NavigationFrame>
        <div className="p-error">
          <AlertTriangle size={44} color="var(--p-red-text)" />
          <h2>Error cargando datos</h2>
          <p>{error || 'No se encontraron datos de rendimiento.'}</p>
          <button type="button" className="p-retry-btn" onClick={retry}>
            Reintentar
          </button>
        </div>
      </NavigationFrame>
    );
  }

  return (
    <NavigationFrame>
      <div className="p-page">
        <PerformanceHero
          viewModel={viewModel}
          overview={sections.overview}
          maps={sections.maps}
          onSelectArea={setActiveTab}
        />

        <TrainingAreaNav
          tabs={TABS}
          activeTab={activeTab}
          onChange={setActiveTab}
          areas={viewModel.areas}
        />

        <main
          key={activeTab}
          id={`p-panel-${activeTab}`}
          role="tabpanel"
          aria-labelledby={`p-tab-${activeTab}`}
          className="p-panel"
        >
          {activeTab === 'overview' && (
            <OverviewTab
              overview={sections.overview}
              sides={sections.sides}
              matchHistory={sections.history}
              maps={sections.maps}
              aim={sections.aim}
              trends={trends}
              viewModel={viewModel}
              onSelectArea={setActiveTab}
            />
          )}

          {activeTab === 'combat' && (
            <CombatTab
              combat={sections.combat}
              overview={sections.overview}
              economy={sections.economy}
            />
          )}

          {activeTab === 'aim' && (
            <AimTab
              aim={sections.aim}
              combat={sections.combat}
              overview={sections.overview}
            />
          )}

          {activeTab === 'weapons' && (
            <WeaponsTab
              weapons={sections.weapons}
              overview={sections.overview}
            />
          )}

          {activeTab === 'maps' && <MapsTab maps={sections.maps} />}

          {activeTab === 'utility' && (
            <UtilityTab
              utility={sections.utility}
              combat={sections.combat}
              economy={sections.economy}
            />
          )}

          {activeTab === 'compare' && (
            <CompareTab
              overview={sections.overview}
              aim={sections.aim}
              combat={sections.combat}
            />
          )}
        </main>
      </div>
    </NavigationFrame>
  );
};

export default Performance;
