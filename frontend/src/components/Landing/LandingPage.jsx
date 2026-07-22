import React, { useEffect, useRef, useState } from 'react';
import { LangProvider } from './i18n/useLang';
import Navbar from './core/Navbar';
import AgentBackground from './core/ParticleBackground';
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

const LandingPageContent = () => {
  const [scrollY, setScrollY] = useState(0);
  const lenisRef = useRef(null);

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
        // Native scroll remains available if Lenis cannot load.
      }
    };

    initLenis();
    return () => {
      if (raf) cancelAnimationFrame(raf);
      lenis?.destroy();
    };
  }, []);

  return (
    <div className="landing-page landing-page--scroll">
      <BackgroundEffects />
      <AgentBackground scrollY={scrollY} />
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
