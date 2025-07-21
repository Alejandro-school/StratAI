import React from 'react';

import Hero from './Hero';
import ProblemSolution from './ProblemSolution';
import StatsCounter from './StatsCounter';
import About from './About';
import Features from './Features';
import VideoDemo from './VideoDemo';
import UseCases from './UseCases';
import Integrations from './Integrations';
import Pricing from './Pricing';
import Testimonials from './Testimonials';
import FAQ from './FAQ';
import Team from './Team';
import SecurityPrivacy from './SecurityPrivacy';
import Comparison from './Comparison';
import CTA from './CTA';
import Footer from './Footer';

const sections = [
  { id: 'hero',        content: <Hero /> },
  { id: 'problem',     content: <ProblemSolution /> },
  { id: 'stats',       content: <StatsCounter /> },
  { id: 'about',       content: <About /> },
  { id: 'features',    content: <Features /> },
  { id: 'video',       content: <VideoDemo /> },
  { id: 'usecases',    content: <UseCases /> },
  { id: 'integrations',content: <Integrations /> },
  { id: 'pricing',     content: <Pricing /> },
  { id: 'testimonials',content: <Testimonials /> },
  { id: 'comparison',  content: <Comparison /> },
  { id: 'faq',         content: <FAQ /> },
  { id: 'team',        content: <Team /> },
  { id: 'security',    content: <SecurityPrivacy /> },
  { id: 'cta',         content: <CTA /> },
  { id: 'footer',      content: <Footer /> },
];

const LandingPage = () => (
  <>
    {sections.map(({ id, content }) => (
      <React.Fragment key={id}>{content}</React.Fragment>
    ))}
  </>
);

export default LandingPage;
