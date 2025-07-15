// Archivo: LandingPage.jsx
import React from 'react';
import '../../styles/Landing/landing.css';

import Hero from './Hero';
import About from './About';
import Features from './Features';
import HowItWorks from './HowItWorks';
import Testimonials from './Testimonials';
import Comparison from './Comparison';
import CTA from './CTA';
import Footer from './Footer';

const LandingPage = () => {
  return (
    <>
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