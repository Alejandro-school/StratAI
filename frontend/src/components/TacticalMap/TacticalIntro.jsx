import React, { useEffect, useId, useRef } from 'react';
import { createPortal } from 'react-dom';
import { AnimatePresence, motion, useReducedMotion } from 'framer-motion';
import {
    ArrowRight,
    Crosshair,
    Radar,
    ShieldCheck,
    X,
} from 'lucide-react';
import { formatMapName } from './tacticalPresentation';
import '../../styles/TacticalMap/tacticalIntelligence.css';

const INTRO_CONCEPTS = [
    {
        id: 'briefing',
        number: '01',
        title: 'Briefing',
        description: 'Tres decisiones: una fortaleza, un hábito y el riesgo prioritario.',
        Icon: Radar,
    },
    {
        id: 'combat',
        number: '02',
        title: 'Combate',
        description: 'Compara volumen, eficiencia, impacto y riesgo con una única escala.',
        Icon: Crosshair,
    },
    {
        id: 'confidence',
        number: '03',
        title: 'Confianza',
        description: 'Cuánta evidencia sostiene cada lectura antes de sacar conclusiones.',
        Icon: ShieldCheck,
    },
];

const FOCUSABLE_SELECTOR = [
    'button:not([disabled])',
    '[href]',
    'input:not([disabled])',
    '[tabindex]:not([tabindex="-1"])',
].join(',');

/**
 * First-visit tactical briefing.
 * Persistence intentionally belongs to the parent via onDismiss/onExplore.
 */
const TacticalIntro = ({
    isOpen = false,
    mapName,
    matchesAnalyzed = 0,
    onDismiss,
    onExplore,
}) => {
    const dialogRef = useRef(null);
    const primaryActionRef = useRef(null);
    const previousFocusRef = useRef(null);
    const titleId = useId();
    const descriptionId = useId();
    const prefersReducedMotion = useReducedMotion();
    const displayMapName = formatMapName(mapName);
    const demoLabel = matchesAnalyzed === 1 ? 'demo analizada' : 'demos analizadas';

    useEffect(() => {
        if (!isOpen || typeof document === 'undefined') {
            return undefined;
        }

        previousFocusRef.current = document.activeElement;
        const appRoot = document.getElementById('root');
        const previousOverflow = document.body.style.overflow;
        appRoot?.setAttribute('inert', '');
        appRoot?.setAttribute('aria-hidden', 'true');
        document.body.style.overflow = 'hidden';
        const focusFrame = window.requestAnimationFrame(() => primaryActionRef.current?.focus());

        return () => {
            window.cancelAnimationFrame(focusFrame);
            appRoot?.removeAttribute('inert');
            appRoot?.removeAttribute('aria-hidden');
            document.body.style.overflow = previousOverflow;
            previousFocusRef.current?.focus?.();
        };
    }, [isOpen]);

    const handleKeyDown = (event) => {
        if (event.key === 'Escape') {
            event.preventDefault();
            onDismiss?.('escape');
            return;
        }

        if (event.key !== 'Tab' || !dialogRef.current) {
            return;
        }

        const focusableElements = [...dialogRef.current.querySelectorAll(FOCUSABLE_SELECTOR)];
        const firstElement = focusableElements[0];
        const lastElement = focusableElements[focusableElements.length - 1];

        if (event.shiftKey && document.activeElement === firstElement) {
            event.preventDefault();
            lastElement?.focus();
        } else if (!event.shiftKey && document.activeElement === lastElement) {
            event.preventDefault();
            firstElement?.focus();
        }
    };

    const overlayMotion = prefersReducedMotion
        ? { initial: { opacity: 0 }, animate: { opacity: 1 }, exit: { opacity: 0 } }
        : { initial: { opacity: 0 }, animate: { opacity: 1 }, exit: { opacity: 0 } };
    const dialogMotion = prefersReducedMotion
        ? { initial: { opacity: 0 }, animate: { opacity: 1 }, exit: { opacity: 0 } }
        : {
            initial: { opacity: 0, y: 20, scale: 0.985 },
            animate: { opacity: 1, y: 0, scale: 1 },
            exit: { opacity: 0, y: 12, scale: 0.99 },
        };

    if (typeof document === 'undefined') return null;

    return createPortal(
        <AnimatePresence>
            {isOpen ? (
                <motion.div
                    className="ti-intro"
                    {...overlayMotion}
                    transition={{ duration: prefersReducedMotion ? 0.01 : 0.22 }}
                >
                    <motion.section
                        ref={dialogRef}
                        className="ti-intro__dialog"
                        role="dialog"
                        aria-modal="true"
                        aria-labelledby={titleId}
                        aria-describedby={descriptionId}
                        onKeyDown={handleKeyDown}
                        {...dialogMotion}
                        transition={{
                            duration: prefersReducedMotion ? 0.01 : 0.38,
                            ease: [0.22, 1, 0.36, 1],
                        }}
                    >
                        <div className="ti-intro__rail" aria-hidden="true">
                            <span>STRATAI / TACTICAL INTELLIGENCE</span>
                            <span>BRIEFING 01</span>
                        </div>

                        <button
                            className="ti-icon-button ti-intro__close"
                            type="button"
                            onClick={() => onDismiss?.('close')}
                            aria-label="Cerrar introducción al mapa táctico"
                        >
                            <X size={19} aria-hidden="true" />
                        </button>

                        <div className="ti-intro__hero">
                            <div className="ti-intro__scope" aria-hidden="true">
                                <Radar size={34} strokeWidth={1.35} />
                                <span />
                            </div>
                            <div>
                                <p className="ti-eyebrow">LECTURA TÁCTICA · {displayMapName}</p>
                                <h2 id={titleId}>Tu juego deja una huella.</h2>
                                <p id={descriptionId} className="ti-intro__lede">
                                    Lee qué ocurre, cuánta evidencia lo respalda y qué decisión
                                    concreta deberías probar en la siguiente partida.
                                </p>
                            </div>
                        </div>

                        <div className="ti-intro__concepts">
                            {INTRO_CONCEPTS.map(({ id, number, title, description, Icon }, index) => (
                                <motion.article
                                    key={id}
                                    className={`ti-concept ti-concept--${id}`}
                                    initial={prefersReducedMotion ? false : { opacity: 0, y: 10 }}
                                    animate={{ opacity: 1, y: 0 }}
                                    transition={{
                                        duration: prefersReducedMotion ? 0.01 : 0.3,
                                        delay: prefersReducedMotion ? 0 : 0.1 + index * 0.06,
                                    }}
                                >
                                    <span className="ti-concept__number">{number}</span>
                                    <div className="ti-concept__icon" aria-hidden="true">
                                        <Icon size={20} strokeWidth={1.7} />
                                    </div>
                                    <h3>{title}</h3>
                                    <p>{description}</p>
                                </motion.article>
                            ))}
                        </div>

                        <div className="ti-intro__sample">
                            <div>
                                <span className="ti-intro__sample-value">{matchesAnalyzed}</span>
                                <span>{demoLabel}</span>
                            </div>
                            <p>
                                Cada partida afina la media. Las muestras pequeñas se señalan;
                                nunca se presentan como una certeza.
                            </p>
                        </div>

                        <div className="ti-intro__actions">
                            <button
                                className="ti-button ti-button--quiet"
                                type="button"
                                onClick={() => onDismiss?.('skip')}
                            >
                                Ahora no
                            </button>
                            <motion.button
                                ref={primaryActionRef}
                                className="ti-button ti-button--primary"
                                type="button"
                                onClick={() => onExplore?.()}
                                whileHover={prefersReducedMotion ? undefined : { y: -2 }}
                                whileTap={prefersReducedMotion ? undefined : { scale: 0.98 }}
                            >
                                Abrir briefing
                                <ArrowRight size={18} aria-hidden="true" />
                            </motion.button>
                        </div>
                    </motion.section>
                </motion.div>
            ) : null}
        </AnimatePresence>,
        document.body,
    );
};

export default TacticalIntro;
