import apiClient from "../../../lib/apiClient";
import { normalizeRound } from "../domain/replayModel";

export const replayApi = {
  metadata: (matchId, signal) => apiClient.get(
    `/match/${encodeURIComponent(matchId)}/replay/metadata`,
    { signal },
  ),
  round: async (matchId, round, signal) => normalizeRound(
    await apiClient.get(`/match/${encodeURIComponent(matchId)}/replay/round/${round}`, { signal }),
  ),
  annotations: (matchId, signal) => apiClient.get(
    `/match/${encodeURIComponent(matchId)}/replay/annotations`,
    { signal },
  ),
  createAnnotation: (matchId, annotation) => apiClient.post(
    `/match/${encodeURIComponent(matchId)}/replay/annotations`,
    annotation,
  ),
  updateAnnotation: (matchId, annotationId, patch) => apiClient.patch(
    `/match/${encodeURIComponent(matchId)}/replay/annotations/${encodeURIComponent(annotationId)}`,
    patch,
  ),
  deleteAnnotation: (matchId, annotationId) => apiClient.delete(
    `/match/${encodeURIComponent(matchId)}/replay/annotations/${encodeURIComponent(annotationId)}`,
  ),
};
