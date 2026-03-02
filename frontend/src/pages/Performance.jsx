import React, { useMemo, useState } from 'react';
import { AlertTriangle, BarChart2, Flame, LayoutDashboard, Map as MapIcon, Shield } from 'lucide-react';
import NavigationFrame from '../components/Layout/NavigationFrame';
import { useUser } from '../context/UserContext';
import { usePerformanceData } from '../hooks/usePerformanceData';
import { computeTrend } from '../utils/performanceFormatters';
import {
  MapsTab,
  OverviewTab,
  RendimientoTab,
  UtilityTab,
} from '../components/Performance';
import '../styles/pages/performance.css';

const TABS = [
  { id: 'overview',    label: 'Resumen',     icon: LayoutDashboard },
  { id: 'rendimiento', label: 'Rendimiento', icon: BarChart2 },
  { id: 'maps',        label: 'Mapas',       icon: MapIcon },
  { id: 'utility',     label: 'Utilidad',    icon: Flame },
];

const PerformanceSkeleton = () => (
  <div className="p-page p-skeleton-view" aria-hidden="true">
    <header className="p-header">
      <div className="p-skel p-skel--title" />
      <div className="p-skel p-skel--sub" />
      <div className="p-skel-tabs p-mt-20">
        {Array.from({ length: 4 }).map((_, i) => (
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
        <header className="p-header">
          <h1 className="p-header-title">
            <Shield className="p-header-icon" size={22} />
            Centro de rendimiento
          </h1>
          <p className="p-header-sub">Análisis integral de tu desempeño como jugador</p>

          <nav className="p-tabs" role="tablist" aria-label="Secciones de rendimiento">
            {TABS.map((tab) => (
              <button
                key={tab.id}
                id={`p-tab-${tab.id}`}
                role="tab"
                aria-selected={activeTab === tab.id}
                aria-controls={`p-panel-${tab.id}`}
                className={`p-tab ${activeTab === tab.id ? 'active' : ''}`}
                onClick={() => setActiveTab(tab.id)}
              >
                <tab.icon size={15} />
                <span>{tab.label}</span>
              </button>
            ))}
          </nav>
        </header>

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
              trends={trends}
            />
          )}

          {activeTab === 'rendimiento' && (
            <RendimientoTab
              aim={sections.aim}
              combat={sections.combat}
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
        </main>
      </div>
    </NavigationFrame>
  );
};

export default Performance;
