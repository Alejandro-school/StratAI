import React from 'react';
import { fireEvent, render } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { ZoomableMapContainer } from './MapZoomControls';

describe('ZoomableMapContainer', () => {
  it('releases pointer capture when dragging ends', () => {
    const { container } = render(
      <ZoomableMapContainer zoomLevel={2}>
        <div>Mapa</div>
      </ZoomableMapContainer>
    );
    const map = container.firstElementChild;
    map.setPointerCapture = vi.fn();
    map.hasPointerCapture = vi.fn(() => true);
    map.releasePointerCapture = vi.fn();

    fireEvent.pointerDown(map, { button: 0, pointerId: 7, clientX: 10, clientY: 10 });
    fireEvent.pointerUp(map, { pointerId: 7 });

    expect(map.setPointerCapture).toHaveBeenCalledWith(7);
    expect(map.releasePointerCapture).toHaveBeenCalledWith(7);
  });

  it('releases an active pointer capture when unmounted', () => {
    const { container, unmount } = render(
      <ZoomableMapContainer zoomLevel={2}>
        <div>Mapa</div>
      </ZoomableMapContainer>
    );
    const map = container.firstElementChild;
    map.setPointerCapture = vi.fn();
    map.hasPointerCapture = vi.fn(() => true);
    map.releasePointerCapture = vi.fn();

    fireEvent.pointerDown(map, { button: 0, pointerId: 11, clientX: 10, clientY: 10 });
    unmount();

    expect(map.releasePointerCapture).toHaveBeenCalledWith(11);
  });
});
