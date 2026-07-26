import { useCallback, useEffect, useState } from "react";
import { replayApi } from "../api/replayApi";

export function useReplayAnnotations(matchId) {
  const [annotations, setAnnotations] = useState([]);
  const [error, setError] = useState("");
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (!matchId) {
      setAnnotations([]);
      return undefined;
    }
    const controller = new AbortController();
    replayApi.annotations(matchId, controller.signal)
      .then(setAnnotations)
      .catch((reason) => {
        if (reason.name !== "AbortError") setError(reason.message);
      });
    return () => controller.abort();
  }, [matchId]);

  const create = useCallback(async (annotation) => {
    if (!matchId) return null;
    setIsSaving(true);
    setError("");
    try {
      const saved = await replayApi.createAnnotation(matchId, annotation);
      setAnnotations((current) => [...current, saved]);
      return saved;
    } catch (reason) {
      setError(reason.message);
      return null;
    } finally {
      setIsSaving(false);
    }
  }, [matchId]);

  const update = useCallback(async (id, patch) => {
    setIsSaving(true);
    setError("");
    try {
      const saved = await replayApi.updateAnnotation(matchId, id, patch);
      setAnnotations((current) => current.map((item) => item.id === id ? saved : item));
      return saved;
    } catch (reason) {
      setError(reason.message);
      return null;
    } finally {
      setIsSaving(false);
    }
  }, [matchId]);

  const remove = useCallback(async (id) => {
    setIsSaving(true);
    setError("");
    try {
      await replayApi.deleteAnnotation(matchId, id);
      setAnnotations((current) => current.filter((item) => item.id !== id));
      return true;
    } catch (reason) {
      setError(reason.message);
      return false;
    } finally {
      setIsSaving(false);
    }
  }, [matchId]);

  return { annotations, create, update, remove, error, isSaving };
}
