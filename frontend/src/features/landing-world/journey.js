import { CatmullRomCurve3, Vector3 } from 'three';
import { OVERVIEW } from './sceneLayout';
import { PRODUCT_CONTENT } from './landingContent';

const flight = new CatmullRomCurve3([
  new Vector3(75, 118, 90), new Vector3(50, 90, 77),
  new Vector3(17, 50, 54), new Vector3(-4, 28, 38), new Vector3(...OVERVIEW.camera),
]);
const attention = new CatmullRomCurve3([
  new Vector3(-17, 0, 0), new Vector3(-7, 0, 6),
  new Vector3(1, 1, 14), new Vector3(4, 1.5, 23), new Vector3(...OVERVIEW.target),
]);
const clamp = value => Math.max(0, Math.min(1, Number.isFinite(value) ? value : 0));

export function writeJourneyPose(progress, position, target, reducedMotion = false) {
  // The exploration phase begins at 82% scroll; its camera has already arrived.
  let t = clamp(clamp(progress) / .82);
  if (reducedMotion) t = Math.round(t * 4) / 4;
  flight.getPoint(t, position);
  attention.getPoint(t, target);
}

export function sampleJourney(progress, reducedMotion = false) {
  const position = new Vector3();
  const target = new Vector3();
  writeJourneyPose(progress, position, target, reducedMotion);
  return { position: position.toArray(), target: target.toArray() };
}

// Transform documented by the source model metadata; never used for analysis.
export const sourceToWorld = ([x, y, z]) => [x * .02 + 13.4, z * .02 + 3.2, -y * .02 - 16.94];

export const FEATURES = [
  { id: 'replay', label: 'Replay táctico', short: 'Replay', place: 'SITE A',
    position: [3, 2.2, 26], camera: OVERVIEW.camera, target: OVERVIEW.target, color: '#a8efd5',
    title: 'Vuelve al instante que importa.',
    description: 'Pausa la ronda, recorre el tiempo y sigue las posiciones desde otra perspectiva.',
    status: 'Demostración visual', action: 'Reproducir jugada' },
  { id: 'utility', label: 'Mapa interactivo', short: 'Mapa', place: 'SITE A',
    position: [12, 2.4, 22], camera: OVERVIEW.camera, target: [8, 2, 25], color: '#ffbe7b',
    title: 'El espacio también se juega.',
    description: 'Explora trayectorias y el contexto de la utilidad sobre el mapa. La distancia y la altura forman parte de la jugada.',
    status: 'Demostración visual', action: 'Ver trayectoria' },
  { id: 'stats', label: 'Estadísticas', short: 'Stats', place: 'SITE A',
    position: [-3, 2, 25], camera: OVERVIEW.camera, target: [1, 0, 25], color: '#c1c5ff',
    title: 'El marcador es el principio.',
    description: 'Consulta el rendimiento de la partida y vuelve a las rondas que hay detrás de cada cifra.',
    status: 'Datos ilustrativos', action: 'Volver a la ronda' },
  { id: 'coach', label: 'Coach IA', short: 'Coach', place: 'ESCALERAS',
    position: [4.15, 4.5, 14.56], camera: OVERVIEW.camera, target: [4.15, 2.4, 19], color: '#94d8ff',
    title: 'Del contexto al siguiente paso.',
    description: 'La visión de StratAI: observaciones respaldadas, prioridades y práctica conectadas con tu partida.',
    status: 'En desarrollo', action: 'Explorar el replay' },
  { id: 'challenges', label: 'Retos y ranking', short: 'Retos', place: 'CT',
    position: [-4.5, 2.2, 30], camera: OVERVIEW.camera, target: [-1, 0, 27], color: '#f3ca89',
    title: PRODUCT_CONTENT.challenges.title, description: PRODUCT_CONTENT.challenges.desc,
    status: 'Propuesta del producto', action: 'Ver cómo funciona' },
];

export const getFeature = id => FEATURES.find(feature => feature.id === id) || null;
