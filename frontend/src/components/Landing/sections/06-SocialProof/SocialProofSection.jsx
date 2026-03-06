/**
 * SocialProofSection — Early access beta CTA + tech stack
 */
import React from 'react';
import { motion } from 'framer-motion';
import { Rocket } from 'lucide-react';
import { useLang } from '../../i18n/useLang';
import { API_URL } from '../../../../utils/api';
import '../../../../styles/Landing/sections/socialProof.css';

const TECH_LOGOS = [
  { name: 'Steam API', label: 'Steam' },
  { name: 'Go', label: 'Go' },
  { name: 'React', label: 'React' },
  { name: 'Three.js', label: 'Three.js' },
  { name: 'FastAPI', label: 'FastAPI' },
  { name: 'Redis', label: 'Redis' },
];

const SocialProofSection = () => {
  const { t } = useLang();

  const handleJoin = () => {
    window.location.href = `${API_URL}/auth/steam/login`;
  };

  return (
    <section id="social-proof" className="sp-section">
      <div className="sp-section__inner">
        {/* Early Access Block */}
        <motion.div
          className="section-header"
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true, margin: '-80px' }}
          transition={{ duration: 0.6 }}
        >
          <span className="section-label">{t('socialProof.label')}</span>
          <h2 className="section-title">{t('socialProof.title')}</h2>
          <p className="section-subtitle">{t('socialProof.subtitle')}</p>

          <div className="sp-beta__spots">
            <div className="sp-beta__spot-count">
              <span className="sp-beta__spot-number">50</span>
              <span className="sp-beta__spot-label">{t('socialProof.spotsLabel')}</span>
            </div>
          </div>

          <motion.button
            className="sp-beta__cta"
            onClick={handleJoin}
            whileHover={{ scale: 1.03 }}
            whileTap={{ scale: 0.98 }}
          >
            <Rocket size={18} />
            <span>{t('socialProof.ctaJoin')}</span>
          </motion.button>
        </motion.div>

        {/* Tech Stack */}
        <motion.div
          className="sp-tech"
          initial={{ opacity: 0 }}
          whileInView={{ opacity: 1 }}
          viewport={{ once: true, margin: '-60px' }}
          transition={{ delay: 0.3, duration: 0.6 }}
        >
          <span className="sp-tech__label">{t('socialProof.poweredBy')}</span>
          <div className="sp-tech__logos">
            {TECH_LOGOS.map(({ name, label }) => (
              <div key={name} className="sp-tech__logo" title={name}>
                <span>{label}</span>
              </div>
            ))}
          </div>
        </motion.div>
      </div>
    </section>
  );
};

export default SocialProofSection;
