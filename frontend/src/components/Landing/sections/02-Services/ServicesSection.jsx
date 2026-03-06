/**
 * ServicesSection — Horizontal scroll gallery driven by vertical scroll.
 * Each card uses the service image as full background with content overlay.
 */
import React, { useEffect, useRef, useState } from 'react';
import { motion, useScroll, useTransform } from 'framer-motion';
import { Brain, Map, BarChart3, Trophy } from 'lucide-react';
import { useLang } from '../../i18n/useLang';
import '../../../../styles/Landing/sections/services.css';

const SERVICES = [
  {
    key: 'coaching',
    icon: Brain,
    accent: '#a78bfa',
    num: '01',
    image: '/images/Landing/CoachIA-.png',
  },
  {
    key: 'interactiveMap',
    icon: Map,
    accent: '#22d3ee',
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
  const stickyRef = useRef(null);
  const galleryRef = useRef(null);
  const [trackDistance, setTrackDistance] = useState(0);
  const { scrollYProgress } = useScroll({
    target: containerRef,
    offset: ['start start', 'end end'],
  });

  useEffect(() => {
    const measure = () => {
      if (!galleryRef.current || !stickyRef.current) return;
      const galleryWidth = galleryRef.current.scrollWidth;
      const viewportWidth = stickyRef.current.clientWidth;
      setTrackDistance(Math.max(galleryWidth - viewportWidth, 0));
    };

    measure();
    window.addEventListener('resize', measure);
    return () => window.removeEventListener('resize', measure);
  }, []);

  const x = useTransform(scrollYProgress, [0, 1], [0, -trackDistance]);

  return (
    <section id="services" className="services-section">
      {/* Intro header */}
      <div className="services-intro">
        <span className="section-label">{t('services.label')}</span>
        <h2 className="services-intro__title">{t('services.title')}</h2>
        <p className="services-intro__subtitle">{t('services.subtitle')}</p>
      </div>

      {/* Horizontal scroll area */}
      <div
        ref={containerRef}
        className="services-scroll-container"
        style={{ '--scroll-distance': `${trackDistance}px` }}
      >
        <div ref={stickyRef} className="services-sticky-wrapper">
          <motion.div ref={galleryRef} className="services-gallery" style={{ x }}>
            {SERVICES.map((service) => {
              const Icon = service.icon;
              const features = t(`services.${service.key}.features`);
              return (
                <div
                  key={service.key}
                  className="services-gallery__item"
                  style={{
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
                </div>
              );
            })}
          </motion.div>
        </div>
      </div>
    </section>
  );
};

export default ServicesSection;
