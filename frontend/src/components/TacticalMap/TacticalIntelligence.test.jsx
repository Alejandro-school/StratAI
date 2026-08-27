import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import TacticalInsightsPanel from './TacticalInsightsPanel';
import TacticalIntro from './TacticalIntro';
import TacticalSignalLayer from './TacticalSignalLayer';

const signals = [
    {
        id: 'habit-a',
        type: 'habit',
        name: 'Conector',
        value: '28% del tiempo',
        confidence: 0.82,
        x: 45,
        y: 38,
    },
    {
        id: 'strength-a',
        type: 'strength',
        name: 'Site A',
        value: '+14 impacto',
        confidence: 74,
        x: 80,
        y: 30,
    },
    {
        id: 'risk-a',
        type: 'risk',
        name: 'Palacio',
        value: '43% de éxito',
        confidence: 0.66,
        x: 62,
        y: 22,
    },
    {
        id: 'ignored',
        type: 'risk',
        name: 'No debe mostrarse',
        value: 'Sin prioridad',
        confidence: 0.4,
        x: 20,
        y: 20,
    },
];

describe('Tactical intelligence presenters', () => {
    it('exposes an accessible briefing and delegates persistence decisions', () => {
        const onDismiss = vi.fn();
        const onExplore = vi.fn();

        render(
            <TacticalIntro
                isOpen
                mapName="de_mirage"
                matchesAnalyzed={18}
                onDismiss={onDismiss}
                onExplore={onExplore}
            />,
        );

        expect(screen.getByRole('dialog', { name: 'Tu juego deja una huella.' }))
            .toBeInTheDocument();
        expect(screen.getByText('18')).toBeInTheDocument();

        fireEvent.click(screen.getByRole('button', { name: 'Abrir briefing' }));
        expect(onExplore).toHaveBeenCalledOnce();

        fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' });
        expect(onDismiss).toHaveBeenCalledWith('escape');
    });

    it('limits the radar to three selectable priority signals', () => {
        const onSelect = vi.fn();

        render(
            <div>
                <TacticalSignalLayer
                    signals={signals}
                    selectedSignalId="strength-a"
                    onSelect={onSelect}
                />
            </div>,
        );

        const signalButtons = screen.getAllByRole('button');
        expect(signalButtons).toHaveLength(3);
        expect(screen.queryByText('No debe mostrarse')).not.toBeInTheDocument();
        expect(screen.getByRole('button', { name: /Fortaleza: Site A/ })).toHaveAttribute(
            'aria-pressed',
            'true',
        );

        fireEvent.click(screen.getByRole('button', { name: /Riesgo: Palacio/ }));
        expect(onSelect).toHaveBeenCalledWith(signals[2]);
    });

    it('collapses signals sharing coordinates and expands them after selection', () => {
        const onSelect = vi.fn();
        const overlapping = [
            { ...signals[0], x: 50, y: 50, position: { x: 50, y: 50 } },
            { ...signals[1], x: 50, y: 50, position: { x: 50, y: 50 } },
        ];
        const { rerender } = render(
            <TacticalSignalLayer signals={overlapping} onSelect={onSelect} />,
        );

        fireEvent.click(screen.getByRole('button', { name: /2 señales en Conector/ }));
        expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ id: 'habit-a' }));

        rerender(
            <TacticalSignalLayer
                signals={overlapping}
                selectedSignalId="habit-a"
                onSelect={onSelect}
            />,
        );
        expect(screen.getAllByRole('button')).toHaveLength(2);
    });

    it('shows the map sample, confidence and at most three insight cards', () => {
        render(
            <TacticalInsightsPanel
                mapName="de_mirage"
                matchesAnalyzed={18}
                confidence={0.78}
                insights={signals.map((signal) => ({
                    ...signal,
                    title: signal.name,
                    detail: 'Lectura táctica priorizada para esta zona.',
                }))}
                onInsightSelect={() => {}}
            />,
        );

        expect(screen.getByText('18 demos')).toBeInTheDocument();
        expect(screen.getByRole('progressbar', { name: 'Confianza global de la lectura' }))
            .toHaveAttribute('aria-valuenow', '78');
        expect(screen.getAllByRole('button')).toHaveLength(3);
        expect(screen.queryByText('No debe mostrarse')).not.toBeInTheDocument();
    });
});
