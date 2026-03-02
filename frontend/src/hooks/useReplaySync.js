import { useCallback, useState } from 'react';

const parseRoundFromText = (text) => {
  const match = text.match(/ronda\s*(\d{1,2})/i);
  return match ? Number(match[1]) : null;
};

const useReplaySync = () => {
  const [externalControl, setExternalControl] = useState(null);

  const issueControl = useCallback((control) => {
    setExternalControl({
      id: `${Date.now()}-${Math.random().toString(16).slice(2)}`,
      ...control
    });
  }, []);

  const jumpToScenario = useCallback((scenario) => {
    if (!scenario) return;
    issueControl({
      round: scenario.round,
      tick: scenario.tick ?? null,
      timestamp: scenario.timestamp,
      play: false
    });
  }, [issueControl]);

  const executeReviewCommand = useCallback((rawText, selectedScenario) => {
    const text = rawText.toLowerCase();

    if (text.includes('10s antes')) {
      issueControl({ seekDeltaSeconds: -10, play: false });
      return true;
    }

    if (text.includes('10s después') || text.includes('10s despues')) {
      issueControl({ seekDeltaSeconds: 10, play: false });
      return true;
    }

    if (text.includes('pausa') || text.includes('detén') || text.includes('deten')) {
      issueControl({ play: false });
      return true;
    }

    if (text.includes('reproduce') || text.includes('play')) {
      issueControl({ play: true });
      return true;
    }

    if (text.includes('inicio ronda')) {
      issueControl({ time: 0, play: false });
      return true;
    }

    if (text.includes('final ronda')) {
      issueControl({ time: 0.98, play: false });
      return true;
    }

    if (text.includes('siguiente error') && selectedScenario?.round) {
      issueControl({ round: Math.min(selectedScenario.round + 1, 30), time: 0.35, play: false });
      return true;
    }

    const explicitRound = parseRoundFromText(text);
    if (explicitRound) {
      issueControl({ round: explicitRound, time: 0.2, play: false });
      return true;
    }

    return false;
  }, [issueControl]);

  return {
    externalControl,
    jumpToScenario,
    executeReviewCommand
  };
};

export default useReplaySync;
