import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import useReplaySyncStore from "../../../stores/useReplaySyncStore";
import {
  activeCombatShotsAtTick,
  clamp,
  closestFrameIndex,
  directorRate,
  interpolateFrame,
  normalizeReplayEvent,
  normalizeRound,
} from "../domain/replayModel";
import {
  replayMetadataQueryOptions,
  replayRoundQueryOptions,
} from "../queries/replayQueries";

const PLAYBACK_RENDER_INTERVAL_MS = 1000 / 40;

function urlState() {
  if (typeof window === "undefined") return {};
  const params = new URLSearchParams(window.location.search);
  const round = Number(params.get("round"));
  const tick = Number(params.get("tick"));
  return {
    round: Number.isFinite(round) && round > 0 ? round : null,
    tick: Number.isFinite(tick) ? tick : null,
    focusPlayerId: params.get("focusPlayer") || null,
  };
}

export function useReplayController({
  matchId,
  preloadedData,
  initialRound,
  externalControl,
  onAvailabilityChange,
}) {
  const queryClient = useQueryClient();
  const initialUrl = useRef(urlState()).current;
  const [metadata, setMetadata] = useState(null);
  const [roundsSummary, setRoundsSummary] = useState([]);
  const [roundIndex, setRoundIndex] = useState(initialRound);
  const [roundData, setRoundData] = useState(null);
  const [tick, setTickState] = useState(initialUrl.tick || 0);
  const [isPlaying, setIsPlaying] = useState(false);
  const [playbackRate, setPlaybackRate] = useState(1);
  const [directorMode, setDirectorMode] = useState(false);
  const [focusPlayerId, setFocusPlayerId] = useState(initialUrl.focusPlayerId);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingRound, setIsLoadingRound] = useState(false);
  const [error, setError] = useState("");
  const cache = useRef(new Map());
  const pendingTick = useRef(initialUrl.tick);
  const tickRef = useRef(tick);
  const frameRequest = useRef(null);
  const lastRenderCommit = useRef(0);
  const { isPlaying: isAiPlaying, activeClip, annotations: aiAnnotations, updateCurrentTick } = useReplaySyncStore();
  const cacheScope = matchId || preloadedData?.metadata?.match_id || "preloaded";

  const actualRound = roundsSummary[roundIndex - 1]?.round || roundIndex;
  const events = useMemo(
    () => (roundData?.events || []).map(normalizeReplayEvent),
    [roundData?.events],
  );
  const frames = roundData?.frames || [];
  const startTick = Number(roundData?.start_tick ?? frames[0]?.tick ?? 0);
  const endTick = Number(roundData?.end_tick ?? frames.at(-1)?.tick ?? startTick);
  const tickRate = Number(metadata?.tick_rate || 64);
  const combatShots = roundData?.combat_shots;
  const currentFrame = useMemo(() => {
    const frame = interpolateFrame(frames, tick);
    if (!frame || !Array.isArray(combatShots)) return frame;
    return { ...frame, shots: activeCombatShotsAtTick(combatShots, tick) };
  }, [combatShots, frames, tick]);
  const progress = endTick > startTick ? clamp((tick - startTick) / (endTick - startTick), 0, 1) : 0;
  const effectiveRate = directorRate(events, tick, tickRate, directorMode, playbackRate);

  const setTick = useCallback((next) => {
    const numeric = Number(typeof next === "function" ? next(tickRef.current) : next);
    const clamped = clamp(numeric, startTick, endTick);
    tickRef.current = clamped;
    setTickState(clamped);
  }, [startTick, endTick]);

  useEffect(() => {
    tickRef.current = tick;
  }, [tick]);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setIsLoading(true);
      setError("");
      try {
        if (preloadedData) {
          const summary = (preloadedData.rounds || []).map((round, index) => ({
            round: round.round || index + 1,
            winner: round.winner || "",
            start_tick: round.start_tick,
            end_tick: round.end_tick,
            events: (round.events || []).map(normalizeReplayEvent),
          }));
          preloadedData.rounds?.forEach((round) => (
            cache.current.set(`${cacheScope}:${round.round}`, normalizeRound(round))
          ));
          if (!cancelled) {
            setMetadata(preloadedData.metadata);
            setRoundsSummary(summary);
          }
        } else if (matchId) {
          const result = await queryClient.fetchQuery(replayMetadataQueryOptions(matchId));
          if (!cancelled) {
            setMetadata(result.metadata);
            setRoundsSummary((result.rounds_summary || []).map((round) => ({
              ...round,
              events: (round.events || []).map(normalizeReplayEvent),
            })));
          }
        }
      } catch (reason) {
        if (!cancelled && reason.name !== "AbortError") setError(reason.message);
      } finally {
        if (!cancelled) setIsLoading(false);
      }
    }
    load();
    return () => { cancelled = true; };
  }, [cacheScope, matchId, preloadedData, queryClient]);

  useEffect(() => {
    if (!roundsSummary.length) return;
    const requestedActual = initialUrl.round || initialRound;
    const found = roundsSummary.findIndex((round) => round.round === requestedActual);
    setRoundIndex(found >= 0 ? found + 1 : clamp(initialRound, 1, roundsSummary.length));
  }, [roundsSummary, initialRound, initialUrl.round]);

  useEffect(() => {
    if (!metadata || !actualRound) return undefined;
    let cancelled = false;
    async function loadRound() {
      setIsLoadingRound(true);
      setError("");
      try {
        const cacheKey = `${cacheScope}:${actualRound}`;
        let data = cache.current.get(cacheKey);
        if (!data && preloadedData) {
          data = normalizeRound(preloadedData.rounds?.find((round) => round.round === actualRound));
        }
        if (!data && matchId) {
          data = await queryClient.fetchQuery(replayRoundQueryOptions(matchId, actualRound));
        }
        if (!cancelled && data) {
          cache.current.set(cacheKey, data);
          setRoundData(data);
          const first = data.start_tick ?? data.frames?.[0]?.tick ?? 0;
          const last = data.end_tick ?? data.frames?.at(-1)?.tick ?? first;
          const target = pendingTick.current == null ? first : clamp(pendingTick.current, first, last);
          pendingTick.current = null;
          tickRef.current = target;
          setTickState(target);
          setIsPlaying(false);
        }
      } catch (reason) {
        if (!cancelled && reason.name !== "AbortError") setError(reason.message);
      } finally {
        if (!cancelled) setIsLoadingRound(false);
      }
    }
    loadRound();
    return () => { cancelled = true; };
  }, [actualRound, cacheScope, matchId, metadata, preloadedData, queryClient]);

  useEffect(() => {
    onAvailabilityChange?.(Boolean(metadata));
  }, [metadata, onAvailabilityChange]);

  useEffect(() => {
    if (!isPlaying && !isAiPlaying) return undefined;
    let previous = performance.now();
    const animate = (now) => {
      // Preserve real elapsed time during a temporarily expensive canvas frame.
      // The wider cap still prevents a huge jump after returning to a hidden tab.
      const delta = Math.min(now - previous, 250);
      previous = now;
      const speed = directorRate(events, tickRef.current, tickRate, directorMode, playbackRate);
      const next = tickRef.current + (delta / 1000) * tickRate * speed;
      if (next >= endTick) {
        setTick(endTick);
        setIsPlaying(false);
        return;
      }
      tickRef.current = next;
      if (now - lastRenderCommit.current >= PLAYBACK_RENDER_INTERVAL_MS) {
        lastRenderCommit.current = now;
        setTickState(next);
      }
      frameRequest.current = requestAnimationFrame(animate);
    };
    frameRequest.current = requestAnimationFrame(animate);
    return () => {
      if (frameRequest.current) cancelAnimationFrame(frameRequest.current);
    };
  }, [directorMode, endTick, events, isAiPlaying, isPlaying, playbackRate, setTick, tickRate]);

  useEffect(() => {
    if (isAiPlaying || activeClip) updateCurrentTick(tick);
  }, [activeClip, isAiPlaying, tick, updateCurrentTick]);

  useEffect(() => {
    if (!activeClip || !roundsSummary.length) return;
    const index = roundsSummary.findIndex((round) => round.round === activeClip.round);
    if (index >= 0 && index + 1 !== roundIndex) {
      pendingTick.current = activeClip.startTick;
      setRoundIndex(index + 1);
    } else if (activeClip.startTick != null) {
      setTick(activeClip.startTick);
    }
    if (isAiPlaying) setIsPlaying(true);
    if (activeClip.focusPlayerId) setFocusPlayerId(String(activeClip.focusPlayerId));
  }, [activeClip, isAiPlaying, roundIndex, roundsSummary, setTick]);

  useEffect(() => {
    if (!isAiPlaying && activeClip) setIsPlaying(false);
  }, [activeClip, isAiPlaying]);

  useEffect(() => {
    if (!externalControl) return;
    if (Number.isFinite(externalControl.round)) {
      const index = roundsSummary.findIndex((round) => round.round === externalControl.round);
      const targetIndex = index >= 0 ? index + 1 : clamp(externalControl.round, 1, roundsSummary.length);
      if (targetIndex !== roundIndex) {
        pendingTick.current = externalControl.tick ?? null;
        setRoundIndex(targetIndex);
      }
    }
    if (externalControl.tick != null && Number.isFinite(Number(externalControl.tick))) setTick(Number(externalControl.tick));
    else if (typeof externalControl.time === "number") setTick(startTick + clamp(externalControl.time, 0, 1) * (endTick - startTick));
    else if (typeof externalControl.timestamp === "string" && frames.length) {
      const parts = externalControl.timestamp.split(":").map(Number);
      const seconds = parts.length === 2 ? parts[0] * 60 + parts[1] : parts[0];
      if (Number.isFinite(seconds)) {
        const closest = frames.reduce((best, frame) => (
          Math.abs((frame.time_remaining || 0) - seconds) < Math.abs((best.time_remaining || 0) - seconds) ? frame : best
        ), frames[0]);
        setTick(closest.tick);
      }
    }
    if (Number.isFinite(externalControl.seekDeltaSeconds)) {
      setTick((current) => current + externalControl.seekDeltaSeconds * tickRate);
    }
    if (typeof externalControl.play === "boolean") setIsPlaying(externalControl.play);
    if (Number.isFinite(externalControl.playbackRate)) setPlaybackRate(clamp(externalControl.playbackRate, 0.25, 4));
    if (externalControl.focusPlayerId != null) setFocusPlayerId(String(externalControl.focusPlayerId));
  }, [endTick, externalControl, frames, roundIndex, roundsSummary, setTick, startTick, tickRate]);

  useEffect(() => {
    if (isPlaying || isAiPlaying) return undefined;
    const timer = window.setTimeout(() => {
      const url = new URL(window.location.href);
      url.searchParams.set("round", String(actualRound));
      url.searchParams.set("tick", String(Math.round(tick)));
      if (focusPlayerId) url.searchParams.set("focusPlayer", String(focusPlayerId));
      else url.searchParams.delete("focusPlayer");
      window.history.replaceState(window.history.state, "", url);
    }, 250);
    return () => window.clearTimeout(timer);
  }, [actualRound, focusPlayerId, isAiPlaying, isPlaying, tick]);

  const seekBySeconds = useCallback((seconds) => setTick((current) => current + seconds * tickRate), [setTick, tickRate]);
  const changeRound = useCallback((nextIndex) => {
    const clamped = clamp(nextIndex, 1, roundsSummary.length);
    if (clamped !== roundIndex) {
      pendingTick.current = null;
      setRoundIndex(clamped);
    }
  }, [roundIndex, roundsSummary.length]);
  const seekProgress = useCallback((value) => setTick(startTick + clamp(value, 0, 1) * (endTick - startTick)), [endTick, setTick, startTick]);
  const seekEvent = useCallback((event) => setTick(event.tick), [setTick]);
  const closestIndex = closestFrameIndex(frames, tick);

  return {
    metadata,
    roundsSummary,
    roundIndex,
    actualRound,
    roundData,
    events,
    frames,
    currentFrame,
    closestFrameIndex: closestIndex,
    tick,
    startTick,
    endTick,
    tickRate,
    progress,
    isPlaying,
    playbackRate,
    effectiveRate,
    directorMode,
    focusPlayerId,
    isLoading,
    isLoadingRound,
    error,
    aiAnnotations,
    activeClip,
    setIsPlaying,
    setPlaybackRate,
    setDirectorMode,
    setFocusPlayerId,
    setTick,
    seekBySeconds,
    seekProgress,
    seekEvent,
    changeRound,
  };
}
