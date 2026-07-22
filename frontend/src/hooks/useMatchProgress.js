// frontend/src/hooks/useMatchProgress.js
// SSE hook for real-time match processing progress
import { useState, useEffect, useCallback, useRef } from 'react';
import { API_URL } from '../utils/api';

/**
 * Connects to the Node.js SSE endpoint to receive real-time
 * match processing events (gc_resolving → downloading → processing → completed | error).
 *
 * @param {string|null} steamId - User's Steam ID. Null disables the connection.
 * @returns {{ events, isProcessing, latestEvent, completedCount }}
 */
export default function useMatchProgress(steamId) {
  const [events, setEvents] = useState([]);
  const [isProcessing, setIsProcessing] = useState(false);
  const [latestEvent, setLatestEvent] = useState(null);
  const [completedCount, setCompletedCount] = useState(0);
  const eventSourceRef = useRef(null);

  const connect = useCallback(() => {
    if (!steamId) return;

    // Close existing connection
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    const es = new EventSource(`${API_URL}/steam/download-progress`, { withCredentials: true });
    eventSourceRef.current = es;

    es.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);
        setLatestEvent(data);
        setEvents(prev => [...prev.slice(-49), data]); // keep last 50

        if (data.stage === 'completed') {
          setCompletedCount(c => c + 1);
        }

        const activeStages = ['gc_resolving', 'downloading', 'processing'];
        setIsProcessing(activeStages.includes(data.stage));
      } catch {
        // ignore parse errors (heartbeats, etc.)
      }
    };

    es.onerror = () => {
      // EventSource auto-reconnects; just clear processing state on close
      if (es.readyState === EventSource.CLOSED) {
        setIsProcessing(false);
      }
    };
  }, [steamId]);

  useEffect(() => {
    connect();
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    };
  }, [connect]);

  const resetCompleted = useCallback(() => setCompletedCount(0), []);

  return { events, isProcessing, latestEvent, completedCount, resetCompleted };
}
