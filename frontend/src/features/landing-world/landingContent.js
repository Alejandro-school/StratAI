import copy from './copy.es.json';

// Preserve the landing's editorial content and original media in this feature.
export const LANDING_COPY = copy;
export const PRODUCT_CONTENT = {
  coach: { ...copy.services.coaching, image: '/images/Landing/CoachIA-.png', alt: 'Captura del Coach IA de StratAI', prospective: true },
  utility: { ...copy.services.interactiveMap, image: '/images/Landing/InteractiveMap.png', alt: 'Mapa interactivo de StratAI con posiciones, duelos y utilidad' },
  stats: { ...copy.services.stats, image: '/images/Landing/Replay2D.png', alt: 'Vista de análisis y replay de StratAI' },
  challenges: { ...copy.services.challenges, image: '/images/Landing/Challenges.png', alt: 'Captura de retos y ranking de StratAI', prospective: true },
  replay: { title: 'Replay táctico', desc: copy.aiDemo.subtitle, image: '/images/Landing/ImagenReplay.png', alt: 'Captura del replay táctico de StratAI', features: [copy.hero.badge1, copy.hero.badge2] },
};

export const INFORMATION_PANELS = [
  { id: 'guide', label: 'Cómo funciona', title: copy.howItWorks.title, color: '#b1f1d8', status: 'Propuesta del producto', place: 'STRATAI' },
  { id: 'plans', label: 'Planes y preguntas', title: copy.pricing.title, color: '#b1f1d8', status: 'Planes propuestos', place: 'STRATAI' },
];

export function planPrice(tier, yearly = false) {
  const price = Number(copy.pricing[tier].price);
  return yearly && price > 0 ? (price * 10).toFixed(0) : copy.pricing[tier].price;
}
