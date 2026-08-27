import React, { memo, useId } from 'react';
import { motion, useReducedMotion } from 'framer-motion';
import {
    Database,
    Radar,
    ScanLine,
    ShieldCheck,
    TriangleAlert,
} from 'lucide-react';
import {
    confidenceToPercent,
    formatConfidence,
    formatMapName,
    SIGNAL_TYPES,
} from './tacticalPresentation';
import '../../styles/TacticalMap/tacticalIntelligence.css';

const INSIGHT_ICONS = {
    strength: ShieldCheck,
    habit: Radar,
    risk: TriangleAlert,
};

const InsightCard = memo(({
    insight,
    index,
    isSelected,
    onSelect,
    prefersReducedMotion,
}) => {
    const type = SIGNAL_TYPES[insight.type] ? insight.type : 'habit';
    const typeConfig = SIGNAL_TYPES[type];
    const Icon = INSIGHT_ICONS[type];
    const title = insight.title || insight.name || `Lectura ${index + 1}`;
    const detail = insight.detail || insight.description || insight.summary;
    const confidenceLabel = formatConfidence(insight.confidence);
    const displayValue = insight.value ?? '—';
    const cardContent = (
        <>
            <div className="ti-insight-card__top">
                <span className="ti-insight-card__index">0{index + 1}</span>
                <span className="ti-insight-card__type">
                    <Icon size={15} strokeWidth={1.8} aria-hidden="true" />
                    {typeConfig.shortLabel}
                </span>
                <strong className="ti-insight-card__value">{displayValue}</strong>
            </div>
            <h3>{title}</h3>
            {detail ? <p>{detail}</p> : null}
            {insight.recommendation ? (
                <p className="ti-insight-card__recommendation">
                    <strong>Próximo ajuste</strong>
                    {insight.recommendation}
                </p>
            ) : null}
            <div className="ti-insight-card__footer">
                <span>
                    <ShieldCheck size={14} aria-hidden="true" />
                    Confianza {confidenceLabel}
                </span>
                <span>{insight.status === 'provisional' ? 'Lectura provisional' : 'Evidencia contrastada'}</span>
            </div>
        </>
    );
    const motionProps = {
        initial: prefersReducedMotion ? false : { opacity: 0, y: 10 },
        animate: { opacity: 1, y: 0 },
        transition: {
            duration: prefersReducedMotion ? 0.01 : 0.3,
            delay: prefersReducedMotion ? 0 : 0.05 + index * 0.06,
        },
    };

    if (onSelect) {
        return (
            <motion.button
                className={`ti-insight-card ${isSelected ? 'is-selected' : ''}`}
                type="button"
                data-type={type}
                aria-pressed={isSelected}
                aria-label={`${typeConfig.label}: ${title}. ${displayValue}. Confianza ${confidenceLabel}.`}
                onClick={() => onSelect(insight)}
                whileHover={prefersReducedMotion ? undefined : { x: 3 }}
                whileTap={prefersReducedMotion ? undefined : { scale: 0.99 }}
                {...motionProps}
            >
                {cardContent}
            </motion.button>
        );
    }

    return (
        <motion.article
            className="ti-insight-card"
            data-type={type}
            {...motionProps}
        >
            {cardContent}
        </motion.article>
    );
});

InsightCard.displayName = 'InsightCard';

/**
 * Compact tactical briefing with sample size, global confidence and three insights.
 */
const TacticalInsightsPanel = ({
    mapName,
    activeSide = 'ct',
    matchesAnalyzed = 0,
    confidence,
    insights = [],
    selectedInsightId = null,
    onInsightSelect,
}) => {
    const titleId = useId();
    const prefersReducedMotion = useReducedMotion();
    const confidencePercentage = confidenceToPercent(confidence);
    const confidenceLabel = formatConfidence(confidence);
    const visibleInsights = insights.slice(0, 3);
    const demoLabel = matchesAnalyzed === 1 ? 'demo' : 'demos';

    return (
        <aside className="ti-insights" aria-labelledby={titleId}>
            <header className="ti-insights__header">
                <div>
                    <p className="ti-eyebrow">INTEL / {formatMapName(mapName)} / {activeSide.toUpperCase()}</p>
                    <h2 id={titleId}>Lectura prioritaria</h2>
                </div>
                <ScanLine size={24} strokeWidth={1.5} aria-hidden="true" />
            </header>

            <div className="ti-insights__evidence">
                <div className="ti-evidence-stat">
                    <Database size={17} aria-hidden="true" />
                    <span>
                        Muestra del mapa
                        <strong>{matchesAnalyzed} {demoLabel}</strong>
                    </span>
                </div>
                <div className="ti-evidence-stat">
                    <ShieldCheck size={17} aria-hidden="true" />
                    <span>
                        Confianza global
                        <strong>{confidenceLabel}</strong>
                    </span>
                </div>
                <div
                    className="ti-confidence-track"
                    role={confidencePercentage !== null ? 'progressbar' : undefined}
                    aria-label="Confianza global de la lectura"
                    aria-valuemin={confidencePercentage !== null ? 0 : undefined}
                    aria-valuemax={confidencePercentage !== null ? 100 : undefined}
                    aria-valuenow={confidencePercentage ?? undefined}
                    aria-valuetext={confidenceLabel}
                >
                    <motion.span
                        initial={prefersReducedMotion ? false : { scaleX: 0 }}
                        animate={{ scaleX: confidencePercentage === null ? 0 : confidencePercentage / 100 }}
                        transition={{
                            duration: prefersReducedMotion ? 0.01 : 0.65,
                            ease: [0.22, 1, 0.36, 1],
                        }}
                    />
                </div>
                <p>
                    Cada demo ajusta la media y eleva la confianza; el volumen siempre
                    acompaña al resultado.
                </p>
            </div>

            <div className="ti-insights__list">
                {visibleInsights.length > 0 ? (
                    visibleInsights.map((insight, index) => (
                        <InsightCard
                            key={insight.id || `${insight.type}-${insight.title || index}`}
                            insight={insight}
                            index={index}
                            isSelected={selectedInsightId === insight.id}
                            onSelect={onInsightSelect}
                            prefersReducedMotion={prefersReducedMotion}
                        />
                    ))
                ) : (
                    <div className="ti-insights__empty" role="status">
                        <ScanLine size={22} aria-hidden="true" />
                        <strong>Aún no hay señales priorizadas</strong>
                        <span>La lectura aparecerá cuando el mapa tenga datos válidos.</span>
                    </div>
                )}
            </div>
        </aside>
    );
};

export default TacticalInsightsPanel;
