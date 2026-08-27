/**
 * ServicesSection — Horizontal scroll gallery driven by vertical scroll.
 * Each card uses the service image as full background with content overlay.
 */
import React, { useRef } from 'react';
import { motion, useScroll, useTransform } from 'framer-motion';
import { Brain, Map, BarChart3, Trophy } from 'lucide-react';
import { useLang } from '../../i18n/useLang';
import '../../../../styles/Landing/sections/services.css';

const SERVICES = [
  {
    key: 'coaching',
    icon: Brain,
    accent: '#818cf8',
    num: '01',
    image: '/images/Landing/CoachIA-.png',
  },
  {
    key: 'interactiveMap',
    icon: Map,
    accent: '#60a5fa',
    num: '02',
    image: '/images/Landing/InteractiveMap.png',
  },
  {
    key: 'stats',
    icon: BarChart3,
    accent: '#10b981',
    num: '03',
    image: '/images/Landing/Replay2D.png',
  },
  {
    key: 'challenges',
    icon: Trophy,
    accent: '#f59e0b',
    num: '04',
    image: '/images/Landing/Challenges.png',
  },
];

const ServicesSection = () => {
  const { t } = useLang();
  const containerRef = useRef(null);
  const { scrollYProgress } = useScroll({
    target: containerRef,
    offset: ['start start', 'end end'],
  });

  // Continuous, slow crossfade animations with no dead zones at the start
  const opacities = [
    useTransform(scrollYProgress, [0, 0.3], [1, 0]),
    useTransform(scrollYProgress, [0, 0.3, 0.35, 0.65], [0, 1, 1, 0]),
    useTransform(scrollYProgress, [0.35, 0.65, 0.7, 1], [0, 1, 1, 0]),
    useTransform(scrollYProgress, [0.7, 1], [0, 1]),
  ];

  const scales = [
    useTransform(scrollYProgress, [0, 0.3], [1, 0.95]),
    useTransform(scrollYProgress, [0, 0.3, 0.35, 0.65], [1.05, 1, 1, 0.95]),
    useTransform(scrollYProgress, [0.35, 0.65, 0.7, 1], [1.05, 1, 1, 0.95]),
    useTransform(scrollYProgress, [0.7, 1], [1.05, 1]),
  ];

  const yPositions = [
    useTransform(scrollYProgress, [0, 0.3], [0, -40]),
    useTransform(scrollYProgress, [0, 0.3, 0.35, 0.65], [40, 0, 0, -40]),
    useTransform(scrollYProgress, [0.35, 0.65, 0.7, 1], [40, 0, 0, -40]),
    useTransform(scrollYProgress, [0.7, 1], [40, 0]),
  ];

  return (
    <section id="services" className="services-section">
      {/* Horizontal scroll area */}
      <div
        ref={containerRef}
        className="services-scroll-container"
        style={{ height: '250vh' }}
      >
        <div className="services-sticky-wrapper">
          {/* Intro header now inside sticky wrapper to stay visible and remove large gap */}
          <div className="services-intro">
            <span className="section-label">{t('services.label')}</span>
            <h2 className="services-intro__title">{t('services.title')}</h2>
            <p className="services-intro__subtitle">{t('services.subtitle')}</p>
          </div>

          <div className="services-gallery-window">
          <div className="services-gallery">
            {SERVICES.map((service, index) => {
              const Icon = service.icon;
              const features = t(`services.${service.key}.features`);
              const opacity = opacities[index];
              const scale = scales[index];
              const y = yPositions[index];
              
              return (
                <motion.div
                  key={service.key}
                  className="services-gallery__item"
                  style={{
                    opacity,
                    scale,
                    y,
                    '--service-accent': service.accent,
                    '--service-image': `url(${service.image})`,
                  }}
                >
                  {/* Number — top left */}
                  <span className="services-gallery__num">{service.num}</span>

                  {/* Content — bottom */}
                  <div className="services-gallery__content">
                    <div className="services-gallery__icon">
                      <Icon size={22} strokeWidth={1.8} />
                    </div>
                    <h3 className="services-gallery__title">
                      {t(`services.${service.key}.title`)}
                    </h3>
                    <p className="services-gallery__desc">
                      {t(`services.${service.key}.desc`)}
                    </p>
                    {Array.isArray(features) && (
                      <div className="services-gallery__tags">
                        {features.map((feat, i) => (
                          <span key={i} className="services-gallery__tag">
                            {feat}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                </motion.div>
              );
            })}
          </div>
          </div>
        </div>
      </div>
    </section>
  );
};

export default ServicesSection;
