export const SIGNAL_TYPES = Object.freeze({
    strength: {
        label: 'Fortaleza',
        shortLabel: 'Fortaleza',
    },
    habit: {
        label: 'Hábito dominante',
        shortLabel: 'Hábito',
    },
    risk: {
        label: 'Riesgo',
        shortLabel: 'Atención',
    },
});

const MAP_GROUP_OFFSETS = [
    { x: -3.2, y: -2.4 },
    { x: 3.2, y: 2.4 },
    { x: 0, y: -4.2 },
];

export const formatCalloutName = (calloutName) => {
    const value = String(calloutName ?? '').trim();
    if (!value) return 'Zona sin nombre';

    return value
        .replace(/[_-]+/g, ' ')
        .replace(/([A-Z]{2,})([A-Z][a-z])/g, '$1 $2')
        .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
        .replace(/\bTopof\b/gi, 'Top of')
        .replace(/\b([CT])\s*spawn\b/gi, (_, side) => `${side.toUpperCase()} Spawn`)
        .replace(/\s+/g, ' ')
        .trim();
};

export const groupSignalsByPosition = (signals = []) => {
    const grouped = new Map();

    signals.forEach((signal) => {
        const position = signal.position ?? { x: signal.x, y: signal.y };
        if (!Number.isFinite(position?.x) || !Number.isFinite(position?.y)) return;

        const key = `${Number(position.x).toFixed(1)}:${Number(position.y).toFixed(1)}`;
        const group = grouped.get(key) ?? {
            id: `signal-group:${key}`,
            position: { x: position.x, y: position.y },
            signals: [],
        };
        group.signals.push(signal);
        grouped.set(key, group);
    });

    return [...grouped.values()].map((group) => ({
        ...group,
        signals: group.signals.map((signal, index) => {
            const offset = group.signals.length > 1
                ? MAP_GROUP_OFFSETS[index] ?? { x: index * 2.8, y: index * -2.8 }
                : { x: 0, y: 0 };
            return {
                ...signal,
                displayPosition: {
                    x: clampMapCoordinate(group.position.x + offset.x),
                    y: clampMapCoordinate(group.position.y + offset.y),
                },
            };
        }),
    }));
};

export const clampMapCoordinate = (coordinate) => {
    const numericCoordinate = Number(coordinate);

    if (!Number.isFinite(numericCoordinate)) {
        return 50;
    }

    return Math.min(96, Math.max(4, numericCoordinate));
};

export const confidenceToPercent = (confidence) => {
    const numericConfidence = Number(confidence);

    if (!Number.isFinite(numericConfidence)) {
        return null;
    }

    const normalizedConfidence = numericConfidence <= 1
        ? numericConfidence * 100
        : numericConfidence;

    return Math.round(Math.min(100, Math.max(0, normalizedConfidence)));
};

export const formatConfidence = (confidence) => {
    const percentage = confidenceToPercent(confidence);

    if (percentage !== null) {
        return `${percentage}%`;
    }

    return typeof confidence === 'string' && confidence.trim()
        ? confidence
        : 'En cálculo';
};

export const formatMapName = (mapName) => {
    if (!mapName) {
        return 'MAPA ACTUAL';
    }

    return String(mapName)
        .replace(/^de_/i, '')
        .replace(/_/g, ' ')
        .toUpperCase();
};
