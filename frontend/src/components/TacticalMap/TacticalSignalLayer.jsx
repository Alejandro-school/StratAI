import React from 'react';
import { motion, useReducedMotion } from 'framer-motion';
import { Layers3, Radar, ShieldCheck, TriangleAlert } from 'lucide-react';
import {
    clampMapCoordinate,
    formatConfidence,
    groupSignalsByPosition,
    SIGNAL_TYPES,
} from './tacticalPresentation';
import '../../styles/TacticalMap/tacticalIntelligence.css';

const SIGNAL_ICONS = {
    strength: ShieldCheck,
    habit: Radar,
    risk: TriangleAlert,
};

const getSourceSignal = ({ displayPosition, ...signal }) => signal;

const SignalButton = ({ signal, index, selected, onSelect, prefersReducedMotion }) => {
    const type = SIGNAL_TYPES[signal.type] ? signal.type : 'habit';
    const typeConfig = SIGNAL_TYPES[type];
    const Icon = SIGNAL_ICONS[type];
    const confidenceLabel = formatConfidence(signal.confidence);
    const signalName = signal.zone || signal.name || `Señal ${index + 1}`;
    const signalValue = signal.value ?? 'Dato disponible';
    const position = signal.displayPosition ?? signal.position ?? { x: signal.x, y: signal.y };
    const x = clampMapCoordinate(position.x);
    const align = signal.align || (x > 72 ? 'left' : 'right');

    return (
        <div
            className={`ti-signal-position ${selected ? 'is-selected' : ''}`}
            style={{ left: `${x}%`, top: `${clampMapCoordinate(position.y)}%` }}
        >
            <div className="ti-signal-anchor">
                <motion.button
                    className="ti-signal"
                    type="button"
                    data-type={type}
                    data-align={align}
                    aria-pressed={selected}
                    aria-label={`${typeConfig.label}: ${signalName}. ${signalValue}. Confianza ${confidenceLabel}.`}
                    onClick={() => onSelect?.(getSourceSignal(signal))}
                    initial={prefersReducedMotion ? false : { opacity: 0, scale: 0.9 }}
                    animate={{ opacity: 1, scale: 1 }}
                    transition={{
                        duration: prefersReducedMotion ? 0.01 : 0.28,
                        delay: prefersReducedMotion ? 0 : 0.06 + index * 0.06,
                        ease: [0.22, 1, 0.36, 1],
                    }}
                    whileHover={prefersReducedMotion ? undefined : { y: -2 }}
                    whileTap={prefersReducedMotion ? undefined : { scale: 0.97 }}
                >
                    <span className="ti-signal__node" aria-hidden="true">
                        <span className="ti-signal__index">0{index + 1}</span>
                        <Icon size={18} strokeWidth={1.8} />
                    </span>
                    <span className="ti-signal__label">
                        <span className="ti-signal__meta">
                            {typeConfig.shortLabel}
                            <span>{confidenceLabel}</span>
                        </span>
                        <strong>{signalName}</strong>
                        <span className="ti-signal__value">{signalValue}</span>
                    </span>
                </motion.button>
            </div>
        </div>
    );
};

const TacticalSignalLayer = ({
    signals = [],
    selectedSignalId = null,
    onSelect,
    ariaLabel = 'Señales tácticas prioritarias',
    toneMap = true,
}) => {
    const prefersReducedMotion = useReducedMotion();
    const visibleSignals = signals.slice(0, 3);
    const groups = groupSignalsByPosition(visibleSignals);
    let visualIndex = 0;

    return (
        <div
            className={`ti-signal-layer ${toneMap ? 'ti-signal-layer--tone-map' : ''}`}
            role="group"
            aria-label={ariaLabel}
        >
            <span className="ti-sr-only">
                {visibleSignals.length} señales prioritarias sobre el mapa.
            </span>

            {groups.map((group) => {
                const selectedInGroup = group.signals.some(({ id }) => id === selectedSignalId);
                if (group.signals.length > 1 && !selectedInGroup) {
                    const firstSignal = group.signals[0];
                    const groupName = firstSignal.zone || firstSignal.name || 'esta zona';
                    return (
                        <div
                            key={group.id}
                            className="ti-signal-position ti-signal-position--group"
                            style={{
                                left: `${clampMapCoordinate(group.position.x)}%`,
                                top: `${clampMapCoordinate(group.position.y)}%`,
                            }}
                        >
                            <div className="ti-signal-anchor">
                                <button
                                    type="button"
                                    className="ti-signal-cluster"
                                    aria-label={`${group.signals.length} señales en ${groupName}. Mostrar señales agrupadas.`}
                                    onClick={() => onSelect?.(getSourceSignal(firstSignal))}
                                >
                                    <Layers3 size={19} aria-hidden="true" />
                                    <strong>{group.signals.length}</strong>
                                    <span>{groupName}</span>
                                </button>
                            </div>
                        </div>
                    );
                }

                return group.signals.map((signal) => {
                    const index = visualIndex;
                    visualIndex += 1;
                    return (
                        <SignalButton
                            key={signal.id}
                            signal={signal}
                            index={index}
                            selected={selectedSignalId === signal.id}
                            onSelect={onSelect}
                            prefersReducedMotion={prefersReducedMotion}
                        />
                    );
                });
            })}
        </div>
    );
};

export default TacticalSignalLayer;
