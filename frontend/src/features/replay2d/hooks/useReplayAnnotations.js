import { useCallback } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { replayApi } from "../api/replayApi";
import { replayAnnotationsQueryOptions, replayKeys } from "../queries/replayQueries";

const EMPTY_ANNOTATIONS = Object.freeze([]);

export function useReplayAnnotations(matchId) {
  const queryClient = useQueryClient();
  const annotationsQuery = useQuery(replayAnnotationsQueryOptions(matchId));
  const updateCache = useCallback((updater) => {
    queryClient.setQueryData(replayKeys.annotations(matchId), (current = []) => updater(current));
  }, [matchId, queryClient]);

  const {
    error: createError,
    isPending: isCreating,
    mutateAsync: createAnnotation,
  } = useMutation({
    mutationFn: (annotation) => replayApi.createAnnotation(matchId, annotation),
    onSuccess: (saved) => updateCache((current) => [...current, saved]),
  });
  const {
    error: updateError,
    isPending: isUpdating,
    mutateAsync: updateAnnotation,
  } = useMutation({
    mutationFn: ({ id, patch }) => replayApi.updateAnnotation(matchId, id, patch),
    onSuccess: (saved) => updateCache((current) => (
      current.map((item) => item.id === saved.id ? saved : item)
    )),
  });
  const {
    error: deleteError,
    isPending: isDeleting,
    mutateAsync: deleteAnnotation,
  } = useMutation({
    mutationFn: (id) => replayApi.deleteAnnotation(matchId, id).then(() => id),
    onSuccess: (id) => updateCache((current) => current.filter((item) => item.id !== id)),
  });

  const create = useCallback(async (annotation) => {
    if (!matchId) return null;
    try {
      return await createAnnotation(annotation);
    } catch {
      return null;
    }
  }, [createAnnotation, matchId]);

  const update = useCallback(async (id, patch) => {
    try {
      return await updateAnnotation({ id, patch });
    } catch {
      return null;
    }
  }, [updateAnnotation]);

  const remove = useCallback(async (id) => {
    try {
      await deleteAnnotation(id);
      return true;
    } catch {
      return false;
    }
  }, [deleteAnnotation]);

  const mutationError = createError || updateError || deleteError;
  return {
    annotations: annotationsQuery.data || EMPTY_ANNOTATIONS,
    create,
    update,
    remove,
    error: annotationsQuery.error?.message || mutationError?.message || "",
    isSaving: isCreating || isUpdating || isDeleting,
  };
}
