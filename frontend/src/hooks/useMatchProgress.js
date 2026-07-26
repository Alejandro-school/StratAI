// frontend/src/hooks/useMatchProgress.js
// SSE hook for real-time match processing progress
import { useState, useEffect, useCallback, useRef } from 'react';
import { API_URL } from '../utils/api';

const ACTIVE_STAGES = ['discovery', 'queued', 'resolving', 'downloading', 'analyzing', 'retry_wait'];

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
  const [snapshot, setSnapshot] = useState(null);
  const [connectionState, setConnectionState] = useState('idle');
  const eventSourceRef = useRef(null);
  const completedJobsRef = useRef(new Set());

  const connect = useCallback(() => {
    if (!steamId) return;

    // Close existing connection
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    const es = new EventSource(`${API_URL}/steam/download-progress`, { withCredentials: true });
    eventSourceRef.current = es;
    setConnectionState('connecting');

    es.onopen = () => setConnectionState('connected');

    const handlePipelineEvent = (e) => {
      try {
        const data = JSON.parse(e.data);
        setLatestEvent(data);
        setEvents(prev => [...prev.slice(-49), data]); // keep last 50

        const completionKey = data.job_id || data.match_id;
        if (
          data.stage === 'completed'
          && completionKey
          && !completedJobsRef.current.has(completionKey)
        ) {
          completedJobsRef.current.add(completionKey);
          setCompletedCount(c => c + 1);
        }

        setIsProcessing(ACTIVE_STAGES.includes(data.stage));
      } catch {
        // ignore parse errors (heartbeats, etc.)
      }
    };
    es.addEventListener('pipeline', handlePipelineEvent);
    es.addEventListener('snapshot', (event) => {
      try {
        const data = JSON.parse(event.data);
        setSnapshot(data);
        const jobs = data.jobs || [];
        const activeJob = jobs.find((job) => ACTIVE_STAGES.includes(job.stage));
        setLatestEvent(activeJob || jobs[0] || null);
        setIsProcessing(Boolean(activeJob));
      } catch {
        setSnapshot(null);
      }
    });

    es.onerror = () => {
      setConnectionState('reconnecting');
      if (es.readyState === EventSource.CLOSED) {
        setIsProcessing(false);
        setConnectionState('closed');
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

  const resetCompleted = useCallback(() => {
    completedJobsRef.current.clear();
    setCompletedCount(0);
  }, []);

  return {
    events,
    isProcessing,
    latestEvent,
    completedCount,
    snapshot,
    connectionState,
    resetCompleted,
  };
}
