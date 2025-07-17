import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import LandingPage from './components/Landing/LandingPage';

// Stub media import used in Hero
jest.mock('./media/Background.mp4', () => 'video');

// Provide a basic IntersectionObserver mock
beforeAll(() => {
  class IO {
    constructor(cb) { this.cb = cb; }
    observe() { this.cb([{ isIntersecting: false }]); }
    unobserve() {}
    disconnect() {}
  }
  global.IntersectionObserver = IO;
});

test('renders landing page heading', () => {
  render(
    <MemoryRouter>
      <LandingPage />
    </MemoryRouter>
  );
  const heading = screen.getByText(/Domina tus partidas/i);
  expect(heading).toBeInTheDocument();
});
