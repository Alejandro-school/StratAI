import React from 'react';
import { fireEvent, render, screen, cleanup } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ProductPanel } from './ProductPanel';
import { FEATURES } from './journey';
import { INFORMATION_PANELS, LANDING_COPY, PRODUCT_CONTENT } from './landingContent';

beforeEach(() => {
  vi.spyOn(HTMLMediaElement.prototype, 'play').mockResolvedValue();
  vi.spyOn(HTMLMediaElement.prototype, 'pause').mockImplementation(() => {});
});
afterEach(() => { cleanup(); vi.restoreAllMocks(); });
const props = { playback: { current: { playing: false, progress: 0 } }, onSelect() {}, onClose() {} };

describe('landing content in the world', () => {
  it.each(FEATURES.map(feature => [feature.id, feature]))('retains the source copy and original capture for %s', (id, feature) => {
    render(<ProductPanel panel={feature} {...props} />);
    expect(screen.getByRole('heading', { name: PRODUCT_CONTENT[id].title })).toBeTruthy();
    if (id === 'replay') fireEvent.click(screen.getByText('Captura original y funciones del replay'));
    expect(screen.getByRole('button', { name: `Ampliar captura: ${PRODUCT_CONTENT[id].title}` }).querySelector('img').getAttribute('src')).toBe(PRODUCT_CONTENT[id].image);
    for (const capability of PRODUCT_CONTENT[id].features) expect(screen.getByText(capability)).toBeTruthy();
  });

  it('keeps all plan features, billing periods and FAQ answers reachable', () => {
    render(<ProductPanel panel={INFORMATION_PANELS[1]} {...props} />);
    for (const tier of ['free', 'pro', 'team']) {
      expect(screen.getByRole('heading', { name: LANDING_COPY.pricing[tier].name })).toBeTruthy();
      for (const feature of LANDING_COPY.pricing[tier].features) expect(screen.getByText(feature)).toBeTruthy();
    }
    fireEvent.click(screen.getByRole('button', { name: 'Anual' }));
    expect(screen.getByText('100', { exact: false }).textContent).toContain('100');
    fireEvent.click(screen.getByRole('button', { name: 'Preguntas frecuentes', exact: true }));
    for (const item of LANDING_COPY.faq.items) {
      expect(screen.getByText(item.q)).toBeTruthy();
      expect(screen.getByText(item.a)).toBeTruthy();
    }
    fireEvent.click(screen.getByRole('button', { name: 'Cómo funciona', exact: true }));
    for (const step of [1, 2, 3]) expect(screen.getByRole('heading', { name: LANDING_COPY.howItWorks[`step${step}Title`] })).toBeTruthy();
  });
});
