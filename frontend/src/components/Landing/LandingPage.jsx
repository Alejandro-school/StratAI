import React from 'react';
import Hero from './Hero';
import About from './About';
import Features from './Features';
import HowItWorks from './HowItWorks';
import Testimonials from './Testimonials';
import Comparison from './Comparison';
import CTA from './CTA';
import Footer from './Footer';
import '../../styles/Landing/landing.css';

const LandingPage = () => (
  <>
    <Hero />
    <About />
    <Features />
    <HowItWorks />
    <Testimonials />
    <Comparison />
    <CTA />
    <Footer />
  </>
);

export default LandingPage;
