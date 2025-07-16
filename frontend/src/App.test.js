import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import LandingPage from './components/Landing/LandingPage';

// Mock GSAP and ScrollTrigger to avoid importing ESM modules in Jest
jest.mock('gsap', () => ({
  __esModule: true,
  default: { registerPlugin: jest.fn(), to: jest.fn() }
}));
jest.mock('gsap/ScrollTrigger', () => ({
  __esModule: true,
  default: {}
}));
jest.mock('./media/Background.mp4', () => 'video');

test('renders landing page heading', () => {
  render(
    <MemoryRouter>
      <LandingPage />

import App from './App';

test('renders landing page heading', () => {
  render(
    <MemoryRouter>
      <App />

    </MemoryRouter>
  );
  const heading = screen.getByText(/Domina tus partidas/i);
  expect(heading).toBeInTheDocument();
});
