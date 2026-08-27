import React, { lazy, Suspense, useEffect, useState } from 'react';
import { LangProvider } from './i18n/useLang';
import Navbar from './core/Navbar';
import BackgroundEffects from './core/effects/BackgroundEffects';
import HeroSection from './sections/hero/HeroSection';
import ServicesSection from './sections/services/ServicesSection';
import HowItWorksSection from './sections/how-it-works/HowItWorksSection';
import ChatDemoSection from './sections/chat-demo/ChatDemoSection';
import PricingSection from './sections/pricing/PricingSection';
import FAQSection from './sections/faq/FAQSection';
import CTASection from './sections/call-to-action/CTASection';
import '../../styles/Landing/landing.css';
import '../../styles/Landing/sections/layout.css';

const AgentBackground = lazy(() => import('./core/ParticleBackground'));
const AGENT_BACKGROUND_FALLBACK_DELAY_MS = 750;
const AGENT_BACKGROUND_IDLE_TIMEOUT_MS = 2000;

const canRenderAgents = () => (
  !window.matchMedia('(max-width: 768px)').matches
  && !window.matchMedia('(prefers-reduced-motion: reduce)').matches
);

const scheduleAgentBackground = (renderAgents) => {
  if (typeof window.requestIdleCallback === 'function') {
    const idleCallbackId = window.requestIdleCallback(
      renderAgents,
      { timeout: AGENT_BACKGROUND_IDLE_TIMEOUT_MS },
    );
    return () => window.cancelIdleCallback(idleCallbackId);
  }

  const timeoutId = window.setTimeout(renderAgents, AGENT_BACKGROUND_FALLBACK_DELAY_MS);
  return () => window.clearTimeout(timeoutId);
};

const LandingPageContent = () => {
  const [shouldRenderAgents, setShouldRenderAgents] = useState(false);

  useEffect(() => {
    const mobileQuery = window.matchMedia('(max-width: 768px)');
    const reducedMotionQuery = window.matchMedia('(prefers-reduced-motion: reduce)');
    let cancelScheduledRender = () => {};

    const updateAgentVisibility = () => {
      cancelScheduledRender();
      if (!canRenderAgents()) {
        setShouldRenderAgents(false);
        return;
      }

      cancelScheduledRender = scheduleAgentBackground(
        () => setShouldRenderAgents(true),
      );
    };

    mobileQuery.addEventListener('change', updateAgentVisibility);
    reducedMotionQuery.addEventListener('change', updateAgentVisibility);
    updateAgentVisibility();

    return () => {
      cancelScheduledRender();
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
