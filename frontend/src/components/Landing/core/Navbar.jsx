import React, { useState, useEffect, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Menu, X, ArrowUpRight } from 'lucide-react';
import { useLang } from '../i18n/useLang';
import { API_URL } from '../../../utils/api';
import '../../../styles/Landing/core/navbar.css';

const NAV_LINKS = [
  { key: 'services', href: '#services' },
  { key: 'howItWorks', href: '#how-it-works' },
  { key: 'demo', href: '#ai-demo' },
  { key: 'pricing', href: '#pricing' },
];

const Navbar = () => {
  const { t, lang, toggleLang } = useLang();
  const [scrolled, setScrolled] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 40);
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => window.removeEventListener('scroll', onScroll);
  }, []);

  const handleNavClick = useCallback((e, href) => {
    e.preventDefault();
    setMobileOpen(false);
    const el = document.querySelector(href);
    if (el) el.scrollIntoView({ behavior: 'smooth' });
  }, []);

  const handleSteamLogin = useCallback(() => {
    window.location.href = `${API_URL}/auth/steam/login`;
  }, []);

  return (
    <nav className={`navbar ${scrolled ? 'navbar--scrolled' : ''}`}>
      <div className="navbar__inner">
        {/* Logo */}
        <a href="#hero" className="navbar__logo" onClick={(e) => handleNavClick(e, '#hero')}>
          <span className="navbar__logo-text">Strat</span>
          <span className="navbar__logo-accent">AI</span>
        </a>

        {/* Desktop Links — centered */}
        <div className="navbar__links">
          {NAV_LINKS.map(({ key, href }) => (
            <a
              key={key}
              href={href}
              className="navbar__link"
              onClick={(e) => handleNavClick(e, href)}
            >
              {t(`nav.${key}`)}
            </a>
          ))}
        </div>

        {/* Actions */}
        <div className="navbar__actions">
          <button
            className="navbar__lang-toggle"
            onClick={toggleLang}
            aria-label="Toggle language"
          >
            {lang === 'es' ? 'EN' : 'ES'}
          </button>
          <button className="navbar__cta" onClick={handleSteamLogin}>
            <span>{t('nav.connectSteam')}</span>
            <ArrowUpRight size={14} strokeWidth={2.5} />
          </button>
          <button
            className="navbar__hamburger"
            onClick={() => setMobileOpen(!mobileOpen)}
            aria-label="Toggle menu"
          >
            {mobileOpen ? <X size={22} /> : <Menu size={22} />}
          </button>
        </div>
      </div>

      {/* Mobile Menu */}
      <AnimatePresence>
        {mobileOpen && (
          <motion.div
            className="navbar__mobile-menu"
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.25 }}
          >
            {NAV_LINKS.map(({ key, href }) => (
              <a
                key={key}
                href={href}
                className="navbar__mobile-link"
                onClick={(e) => handleNavClick(e, href)}
              >
                {t(`nav.${key}`)}
              </a>
            ))}
            <button className="navbar__mobile-cta" onClick={handleSteamLogin}>
              {t('nav.connectSteam')}
            </button>
          </motion.div>
        )}
      </AnimatePresence>
    </nav>
  );
};

export default Navbar;
