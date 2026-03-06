/**
 * CTASection — Final conversion block + footer
 */
import React, { useCallback } from 'react';
import { motion } from 'framer-motion';
import { ArrowRight } from 'lucide-react';
import { useLang } from '../../i18n/useLang';
import { API_URL } from '../../../../utils/api';
import '../../../../styles/Landing/sections/ctaFooter.css';

const CTASection = ({ onOpenChallenge }) => {
  const { t } = useLang();

  const handleSteamLogin = useCallback(() => {
    window.location.href = `${API_URL}/auth/steam/login`;
  }, []);

  return (
    <>
      {/* CTA Block */}
      <section className="cta-section">
        <div className="cta-section__inner">
          <motion.div
            className="section-header"
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true, margin: '-80px' }}
            transition={{ duration: 0.6 }}
          >
            <h2 className="section-title">{t('cta.title')}</h2>
            <p className="section-subtitle">{t('cta.subtitle')}</p>
            <div className="cta-section__buttons">
              <button className="cta-section__btn cta-section__btn--primary" onClick={handleSteamLogin}>
                <span>{t('cta.ctaPrimary')}</span>
                <ArrowRight size={18} />
              </button>
              <button className="cta-section__btn cta-section__btn--secondary" onClick={onOpenChallenge}>
                <span>{t('cta.ctaSecondary')}</span>
              </button>
            </div>
          </motion.div>
        </div>
      </section>

      {/* Footer */}
      <footer className="landing-footer">
        <div className="landing-footer__inner">
          <div className="landing-footer__grid">
            <div className="landing-footer__brand">
              <span className="landing-footer__logo">
                <span>Strat</span><span className="text-gradient">AI</span>
              </span>
              <p className="landing-footer__tagline">
                {t('hero.subtitle')}
              </p>
            </div>
            <div className="landing-footer__col">
              <h4>{t('footer.product')}</h4>
              <a href="#services">{t('footer.services')}</a>
              <a href="#how-it-works">{t('footer.howItWorks')}</a>
              <a href="#pricing">{t('footer.pricing')}</a>
            </div>
            <div className="landing-footer__col">
              <h4>{t('footer.resources')}</h4>
              <a href="#ai-demo">Demo IA</a>
            </div>
            <div className="landing-footer__col">
              <h4>{t('footer.legal')}</h4>
              <span className="text-muted" style={{ cursor: 'not-allowed', fontSize: '14px' }}>{t('footer.privacy')}</span>
              <span className="text-muted" style={{ cursor: 'not-allowed', fontSize: '14px' }}>{t('footer.terms')}</span>
            </div>
          </div>
          <div className="landing-footer__bottom">
            <span>© {new Date().getFullYear()} StratAI. {t('footer.rights')}</span>
          </div>
        </div>
      </footer>
    </>
  );
};

export default CTASection;
