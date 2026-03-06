/**
 * LandingPage — Scroll-based landing with Lenis smooth scroll
 *
 * Sections: Hero → Services → HowItWorks → ChatDemo → Pricing → FAQ → CTA
 * Challenges accessible via fullscreen modal overlay
 */
import React, { Suspense, lazy, useState, useEffect, useRef, useCallback } from 'react';

// i18n & Context
import { LangProvider } from './i18n/useLang';

// Core
import Navbar from './core/Navbar';
import AgentBackground from './core/ParticleBackground';
import BackgroundEffects from './core/effects/BackgroundEffects';

// Sections
import HeroSection from './sections/00-Hero/HeroSection';
import ServicesSection from './sections/02-Services/ServicesSection';
import HowItWorksSection from './sections/03-HowItWorks/HowItWorksSection';
import ChatDemoSection from './sections/03-ChatDemo/ChatDemoSection';
import PricingSection from './sections/04-Pricing/PricingSection';
import FAQSection from './sections/05-FAQ/FAQSection';
import CTASection from './sections/07-CTA/CTASection';

// Styles
import '../../styles/Landing/landing.css';
import '../../styles/Landing/sections/layout.css';

// Lazy-loaded modal (only when user clicks "test your level")
const ChallengeModal = lazy(() => import('./modal/ChallengeModal'));

/**
 * LandingPageContent — scroll container with Lenis
 */
const LandingPageContent = () => {
  const [modalOpen, setModalOpen] = useState(false);
  const [scrollY, setScrollY] = useState(0);
  const lenisRef = useRef(null);

  // Initialize Lenis smooth scroll
  useEffect(() => {
    let raf;
    let lenis;

    const initLenis = async () => {
      try {
        const { default: Lenis } = await import('@studio-freight/lenis');
        lenis = new Lenis({
          duration: 1.2,
          easing: (t) => Math.min(1, 1.001 - Math.pow(2, -10 * t)),
          smoothWheel: true,
        });
        lenisRef.current = lenis;

        lenis.on('scroll', ({ scroll }) => setScrollY(scroll));

        const loop = (time) => {
          lenis.raf(time);
          raf = requestAnimationFrame(loop);
        };
        raf = requestAnimationFrame(loop);
      } catch {
        // Lenis unavailable — native scroll works fine
      }
    };

    initLenis();
    return () => {
      if (raf) cancelAnimationFrame(raf);
      lenis?.destroy();
    };
  }, []);

  // Pause/resume Lenis when modal opens/closes
  useEffect(() => {
    const lenis = lenisRef.current;
    if (!lenis) return;
    if (modalOpen) {
      lenis.stop();
      document.body.style.overflow = 'hidden';
    } else {
      lenis.start();
      document.body.style.overflow = '';
    }
  }, [modalOpen]);

  const openModal = useCallback(() => setModalOpen(true), []);
  const closeModal = useCallback(() => setModalOpen(false), []);

  return (
    <div className="landing-page landing-page--scroll">
      {/* Fixed backgrounds */}
      <BackgroundEffects />
      <AgentBackground scrollY={scrollY} />

      {/* Navigation */}
      <Navbar />

      {/* Scrollable content */}
      <main className="landing-main">
        <HeroSection onOpenChallenge={openModal} />
        <ServicesSection />
        <HowItWorksSection />
        <div id="ai-demo" style={{ paddingTop: '80px', paddingBottom: '80px' }}>
          <ChatDemoSection onOpenChallenge={openModal} isScrollPage={true} />
        </div>
        <PricingSection />
        <FAQSection />
        <CTASection onOpenChallenge={openModal} />
      </main>

      {/* Challenge Modal (lazy) */}
      {modalOpen && (
        <Suspense fallback={null}>
          <ChallengeModal onClose={closeModal} />
        </Suspense>
      )}
    </div>
  );
};

/**
 * LandingPage — Entry with providers
 */
export const LandingPage = () => (
  <LangProvider>
    <LandingPageContent />
  </LangProvider>
);

export default LandingPage;
