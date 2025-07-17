// Archivo: LandingPage.jsx
import React, { useEffect, useRef } from 'react';
import '../../styles/Landing/landing.css';

import Hero from './Hero';
import About from './About';
import Features from './Features';
import HowItWorks from './HowItWorks';
import Testimonials from './Testimonials';
import Comparison from './Comparison';
import CTA from './CTA';
import Footer from './Footer';
import SectionIndex from './SectionIndex';

const LandingPage = () => {
  const indexRef = useRef(0);

  useEffect(() => {
    const sections = document.querySelectorAll('main > section');
    if (sections.length === 0) return;

    const interval = setInterval(() => {
      indexRef.current = (indexRef.current + 1) % sections.length;
      sections[indexRef.current].scrollIntoView({ behavior: 'smooth' });
    }, 10000); // cada 10 segundos

    return () => clearInterval(interval);
  }, []);

  return (
    <>
      <SectionIndex />
      <main>
        <Hero />
        <About />
        <Features />
        <HowItWorks />
        <Testimonials />
        <Comparison />
        <CTA />
        <Footer />
      </main>
    </>
  );
};

export default LandingPage;