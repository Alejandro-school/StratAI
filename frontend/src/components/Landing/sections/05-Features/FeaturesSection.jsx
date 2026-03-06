/**
 * FeaturesSection — Showcase platform capabilities
 */
import React from 'react';
import { motion } from 'framer-motion';
import { LayoutDashboard, Target, Award } from 'lucide-react';
import { useLang } from '../../i18n/useLang';
import '../../../../styles/Landing/sections/features.css';

const stagger = {
  hidden: {},
  show: { transition: { staggerChildren: 0.12 } },
};

const fadeUp = {
  hidden: { opacity: 0, y: 30 },
  show: { opacity: 1, y: 0, transition: { duration: 0.6, ease: [0.22, 1, 0.36, 1] } },
};

const FEATURES = [
  { key: 'dashboard', icon: LayoutDashboard, accent: 'var(--color-primary-400)' },
  { key: 'monthlyChallenges', icon: Target, accent: 'var(--color-warning)' },
  { key: 'rewards', icon: Award, accent: 'var(--color-success)' },
];

const FeaturesSection = () => {
  const { t } = useLang();

  return (
    <section id="features" className="features-section">
      <div className="features-section__inner">
        {/* Header */}
        <motion.div
          className="features-section__header"
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true, margin: '-80px' }}
          transition={{ duration: 0.6 }}
        >
          <span className="section-label">{t('features.label')}</span>
          <h2 className="section-title">{t('features.title')}</h2>
        </motion.div>

        {/* Feature cards */}
        <motion.div
          className="features-grid"
          variants={stagger}
          initial="hidden"
          whileInView="show"
          viewport={{ once: true, margin: '-60px' }}
        >
          {FEATURES.map(({ key, icon: Icon, accent }) => (
            <motion.div
              key={key}
              className="feature-card"
              style={{ '--feature-accent': accent }}
              variants={fadeUp}
              whileHover={{ y: -4, transition: { duration: 0.25 } }}
            >
              <div className="feature-card__icon">
                <Icon size={28} strokeWidth={1.6} />
              </div>
              <h3 className="feature-card__title">{t(`features.${key}.title`)}</h3>
              <p className="feature-card__desc">{t(`features.${key}.desc`)}</p>
            </motion.div>
          ))}
        </motion.div>
      </div>
    </section>
  );
};

export default FeaturesSection;
