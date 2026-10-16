import { render, screen } from '@testing-library/react';
import { it, expect } from 'vitest';
import { ReplayHeader } from './ReplayHeader';

it('shows the dated native phase and an unavailable clock without showing zero', () => {
  render(<ReplayHeader roundIndex={1} rounds={[{}]} ctScore={0} tScore={0} onRoundChange={() => {}}
    frame={{ players:[], time_remaining:null, round_context:{ phase:'ended', winner:'T' } }} />);
  expect(screen.getByText('FINALIZADA · T')).toBeInTheDocument();
  expect(screen.getByText('—')).toBeInTheDocument();
  expect(screen.queryByText('0:00')).not.toBeInTheDocument();
});
