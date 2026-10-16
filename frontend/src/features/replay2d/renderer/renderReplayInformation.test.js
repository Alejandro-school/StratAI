import { expect, it, vi } from 'vitest';
import { drawPlayers } from './renderReplayPlayers';
import { createViewport } from './replayViewport';

it('draws a hollow dated memory without a living marker, direction or click target', () => {
  const context = Object.fromEntries(['save','restore','setLineDash','beginPath','arc','stroke','fill','fillText']
    .map(method => [method, vi.fn()]));
  context.measureText = text => ({ width: text.length * 5 });
  const frame = { perspective: 'T', players: [], death_markers: [], last_seen_markers: [
    { subject_id: '6', team: 'CT', name: 'CT1', position: { x: 20, y: -30 }, age_ticks: 512 }] };
  const targets = drawPlayers(context, frame, { pos_x: 0, pos_y: 0, scale: 1 }, createViewport(1024, 1024),
    { tickRate: 64, layers: { names: true, deaths: true } });
  expect(context.setLineDash).toHaveBeenCalledWith([3, 3]);
  expect(context.fillText).toHaveBeenCalledWith('CT1 · hace 8 s', 20, 10);
  expect(context.fill).not.toHaveBeenCalled(); expect(targets).toEqual([]);
});

it('keeps nearby memory labels readable without moving either historical position', () => {
  const context = Object.fromEntries(['save','restore','setLineDash','beginPath','arc','stroke','fill','fillText']
    .map(method => [method, vi.fn()]));
  context.measureText = text => ({ width: text.length * 5 });
  const markers = ['CT2', 'CT4'].map((name, index) => ({ subject_id: String(index + 6), team: 'CT', name,
    position: { x: 20 + index, y: -30 }, age_ticks: 1423 }));
  drawPlayers(context, { perspective: 'T', players: [], last_seen_markers: markers },
    { pos_x: 0, pos_y: 0, scale: 1 }, createViewport(1024, 1024), { tickRate: 64, layers: { names: true } });
  const labels = context.fillText.mock.calls;
  expect(labels.map(call => call[0])).toEqual(['CT2 · hace 22.2 s', 'CT4 · hace 22.2 s']);
  expect(Math.abs(labels[0][2] - labels[1][2])).toBeGreaterThanOrEqual(14);
  expect(context.arc.mock.calls.map(call => call.slice(0, 2))).toEqual([[20, 30], [21, 30]]);
  expect(context.fill).not.toHaveBeenCalled();
});
