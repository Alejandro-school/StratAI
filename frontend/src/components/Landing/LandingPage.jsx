import React, { lazy, Suspense, useEffect, useState } from 'react';
import { LangProvider } from './i18n/useLang';
import Navbar from './core/Navbar';
import BackgroundEffects from './core/effects/BackgroundEffects';
import HeroSection from './sections/00-Hero/HeroSection';
import ServicesSection from './sections/02-Services/ServicesSection';
import HowItWorksSection from './sections/03-HowItWorks/HowItWorksSection';
import ChatDemoSection from './sections/03-ChatDemo/ChatDemoSection';
import PricingSection from './sections/04-Pricing/PricingSection';
import FAQSection from './sections/05-FAQ/FAQSection';
import CTASection from './sections/07-CTA/CTASection';
import '../../styles/Landing/landing.css';
import '../../styles/Landing/sections/layout.css';

const AgentBackground = lazy(() => import('./core/ParticleBackground'));

const canRenderAgents = () => (
  !window.matchMedia('(max-width: 768px)').matches
  && !window.matchMedia('(prefers-reduced-motion: reduce)').matches
);

const LandingPageContent = () => {
  const [shouldRenderAgents, setShouldRenderAgents] = useState(canRenderAgents);

  useEffect(() => {
    const mobileQuery = window.matchMedia('(max-width: 768px)');
    const reducedMotionQuery = window.matchMedia('(prefers-reduced-motion: reduce)');
    const updateAgentVisibility = () => setShouldRenderAgents(canRenderAgents());

    mobileQuery.addEventListener('change', updateAgentVisibility);
    reducedMotionQuery.addEventListener('change', updateAgentVisibility);

    return () => {
      mobileQuery.removeEventListener('change', updateAgentVisibility);
      reducedMotionQuery.removeEventListener('change', updateAgentVisibility);
    };
  }, []);

  useEffect(() => {
    let raf;
    let lenis;
    let isCancelled = false;

    const initLenis = async () => {
      try {
        const { default: Lenis } = await import('lenis');
        const instance = new Lenis({
          duration: 1.2,
          easing: (t) => Math.min(1, 1.001 - Math.pow(2, -10 * t)),
          smoothWheel: true,
        });

        if (isCancelled) {
          instance.destroy();
          return;
        }

        lenis = instance;

        const loop = (time) => {
          lenis.raf(time);
          raf = requestAnimationFrame(loop);
        };
        raf = requestAnimationFrame(loop);
      } catch {
        // Native scroll remains available if Lenis cannot load.
      }
    };

    initLenis();
    return () => {
      isCancelled = true;
      if (raf) cancelAnimationFrame(raf);
      lenis?.destroy();
    };
  }, []);

  return (
    <div className="landing-page landing-page--scroll">
      <BackgroundEffects />
      {shouldRenderAgents && (
        <Suspense fallback={null}>
          <AgentBackground />
        </Suspense>
      )}
      <Navbar />

      <main className="landing-main">
        <HeroSection />
        <div id="ai-demo" className="landing-demo-band">
          <ChatDemoSection isScrollPage={true} />
        </div>
        <ServicesSection />
        <HowItWorksSection />
        <PricingSection />
        <FAQSection />
        <CTASection />
      </main>
    </div>
  );
};

export const LandingPage = () => (
  <LangProvider>
    <LandingPageContent />
  </LangProvider>
);

export default LandingPage;
