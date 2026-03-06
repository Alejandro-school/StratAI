import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { Check, Sparkles } from 'lucide-react';
import { useLang } from '../../i18n/useLang';
import '../../../../styles/Landing/sections/pricing.css';

const TIERS = ['free', 'pro', 'team'];

const PricingSection = () => {
  const { t } = useLang();
  const [yearly, setYearly] = useState(false);

  return (
    <section id="pricing" className="pricing-section">
      <div className="pricing-section__inner">
        {/* Header */}
        <div className="pricing-section__header">
          <span className="section-label">{t('pricing.label')}</span>
          <h2 className="pricing-section__title">{t('pricing.title')}</h2>
          <p className="pricing-section__subtitle">{t('pricing.subtitle')}</p>

          {/* Toggle */}
          <div className="pricing-toggle">
            <span className={!yearly ? 'pricing-toggle__label--active' : ''}>
              {t('pricing.monthly')}
            </span>
            <button
              className={`pricing-toggle__switch ${yearly ? 'pricing-toggle__switch--on' : ''}`}
              onClick={() => setYearly((v) => !v)}
              aria-label="Toggle billing period"
            >
              <span className="pricing-toggle__knob" />
            </button>
            <span className={yearly ? 'pricing-toggle__label--active' : ''}>
              {t('pricing.yearly')}
            </span>
          </div>
        </div>

        {/* Cards */}
        <div className="pricing-cards">
          {TIERS.map((tier, i) => {
            const data = t(`pricing.${tier}`, { returnObjects: true });
            const isPro = tier === 'pro';
            const rawPrice = parseFloat(data.price);
            const displayPrice = yearly && rawPrice > 0
              ? (rawPrice * 10).toFixed(0)
              : data.price;

            return (
              <motion.div
                key={tier}
                className={`pricing-card ${isPro ? 'pricing-card--featured' : ''}`}
                initial={{ opacity: 0, y: 30 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true, margin: '-60px' }}
                transition={{ duration: 0.5, delay: i * 0.1 }}
              >
                {isPro && (
                  <div className="pricing-card__badge">
                    <Sparkles size={12} />
                    {t('pricing.popular')}
                  </div>
                )}

                <h3 className="pricing-card__name">{data.name}</h3>
                <p className="pricing-card__desc">{data.desc}</p>

                <div className="pricing-card__price">
                  <span className="pricing-card__currency">€</span>
                  <span className="pricing-card__amount">{displayPrice}</span>
                  <span className="pricing-card__period">
                    {rawPrice === 0 ? data.period : yearly ? '/año' : data.period}
                  </span>
                </div>

                <ul className="pricing-card__features">
                  {data.features.map((feat, j) => (
                    <li key={j}>
                      <Check size={14} className="pricing-card__check" />
                      {feat}
                    </li>
                  ))}
                </ul>

                <button className={`pricing-card__cta ${isPro ? 'pricing-card__cta--primary' : ''}`}>
                  {data.cta}
                </button>
              </motion.div>
            );
          })}
        </div>
      </div>
    </section>
  );
};

export default PricingSection;
