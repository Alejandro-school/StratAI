import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { ChevronDown } from 'lucide-react';
import { useLang } from '../../i18n/useLang';
import '../../../../styles/Landing/sections/faq.css';

const FAQItem = ({ question, answer, isOpen, onToggle }) => (
  <div className={`faq-item ${isOpen ? 'faq-item--open' : ''}`}>
    <button className="faq-item__trigger" onClick={onToggle} aria-expanded={isOpen}>
      <span className="faq-item__question">{question}</span>
      <ChevronDown size={18} className={`faq-item__icon ${isOpen ? 'faq-item__icon--open' : ''}`} />
    </button>
    <AnimatePresence initial={false}>
      {isOpen && (
        <motion.div
          className="faq-item__body"
          initial={{ height: 0, opacity: 0 }}
          animate={{ height: 'auto', opacity: 1 }}
          exit={{ height: 0, opacity: 0 }}
          transition={{ duration: 0.3, ease: [0.22, 1, 0.36, 1] }}
        >
          <p className="faq-item__answer">{answer}</p>
        </motion.div>
      )}
    </AnimatePresence>
  </div>
);

const FAQSection = () => {
  const { t } = useLang();
  const [openIdx, setOpenIdx] = useState(0);
  const items = t('faq.items', { returnObjects: true }) || [];

  return (
    <section id="faq" className="faq-section">
      <div className="faq-section__inner">
        <div className="faq-section__header">
          <span className="section-label">{t('faq.label')}</span>
          <h2 className="faq-section__title">{t('faq.title')}</h2>
        </div>

        <div className="faq-list">
          {items.map((item, i) => (
            <FAQItem
              key={i}
              question={item.q}
              answer={item.a}
              isOpen={openIdx === i}
              onToggle={() => setOpenIdx(openIdx === i ? -1 : i)}
            />
          ))}
        </div>
      </div>
    </section>
  );
};

export default FAQSection;
